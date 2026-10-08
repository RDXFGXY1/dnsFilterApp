package plugins

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxManifestBytes     = 256 << 10
	maxResponseBytes     = 2 << 20
	maxPluginUploadBytes = 50 << 20
)

var identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,79}$`)

type Manifest struct {
	SchemaVersion int      `json:"schema_version"`
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Description   string   `json:"description"`
	Publisher     string   `json:"publisher"`
	Icon          string   `json:"icon,omitempty"`
	UI            *UI      `json:"ui,omitempty"`
	Server        Server   `json:"server"`
	Permissions   []string `json:"permissions"`
	Panels        []Panel  `json:"panels"`
	Actions       []Action `json:"actions"`
}

type UI struct {
	Mode        string `json:"mode"`
	DefaultView string `json:"default_view,omitempty"`
	WebPath     string `json:"web_path,omitempty"`
}

type Server struct {
	BaseURL        string   `json:"base_url"`
	HealthEndpoint string   `json:"health_endpoint"`
	StartMode      string   `json:"start_mode"`
	Runtime        *Runtime `json:"runtime,omitempty"`
}

type Runtime struct {
	Windows                *Command `json:"windows,omitempty"`
	Linux                  *Command `json:"linux,omitempty"`
	Darwin                 *Command `json:"darwin,omitempty"`
	StartupTimeoutSeconds  int      `json:"startup_timeout_seconds,omitempty"`
	ShutdownTimeoutSeconds int      `json:"shutdown_timeout_seconds,omitempty"`
}

type Command struct {
	Executable       string   `json:"executable"`
	Args             []string `json:"args,omitempty"`
	WorkingDirectory string   `json:"working_directory,omitempty"`
}

type Panel struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Actions     []string `json:"actions"`
}

type Action struct {
	ID             string         `json:"id"`
	Label          string         `json:"label"`
	Description    string         `json:"description,omitempty"`
	Method         string         `json:"method"`
	Path           string         `json:"path"`
	Permission     string         `json:"permission"`
	Confirm        string         `json:"confirm,omitempty"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
	BodyType       string         `json:"body_type,omitempty"`
	ResponseType   string         `json:"response_type,omitempty"`
	Binary         *BinaryRequest `json:"binary,omitempty"`
	Fields         []Field        `json:"fields,omitempty"`
}

type BinaryRequest struct {
	FileField      string `json:"file_field"`
	FilenameHeader string `json:"filename_header"`
	MetadataHeader string `json:"metadata_header,omitempty"`
	MetadataRoot   string `json:"metadata_root,omitempty"`
	MaxBytes       int64  `json:"max_bytes,omitempty"`
}

type Field struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Default     any      `json:"default,omitempty"`
	Options     []Option `json:"options,omitempty"`
	Accept      string   `json:"accept,omitempty"`
	Transform   string   `json:"transform,omitempty"`
}

type Option struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}

type Installed struct {
	Manifest    Manifest `json:"manifest"`
	Enabled     bool     `json:"enabled"`
	InstalledAt string   `json:"installed_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type Event struct {
	ID        int64  `json:"id"`
	PluginID  string `json:"plugin_id"`
	Action    string `json:"action"`
	Success   bool   `json:"success"`
	Status    int    `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type Manager struct {
	db                *sql.DB
	client            *http.Client
	runtimeMu         sync.Mutex
	processes         map[string]*managedProcess
	lastRuntimeErrors map[string]string
}

func New(db *sql.DB) (*Manager, error) {
	if db == nil {
		return nil, errors.New("plugin database is unavailable")
	}
	manager := &Manager{db: db, processes: make(map[string]*managedProcess), lastRuntimeErrors: make(map[string]string)}
	manager.client = &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("plugin redirects are disabled")
		},
	}
	if err := manager.initialize(); err != nil {
		return nil, err
	}
	go manager.startAutomaticPlugins()
	return manager, nil
}

func (m *Manager) initialize() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS plugins (
			id TEXT PRIMARY KEY,
			manifest TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 0,
			installed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS plugin_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plugin_id TEXT NOT NULL,
			action TEXT NOT NULL,
			success INTEGER NOT NULL,
			status INTEGER NOT NULL DEFAULT 0,
			message TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_plugin_events_plugin_time ON plugin_events(plugin_id, created_at DESC);
	`)
	if err != nil {
		return fmt.Errorf("initialize plugin registry: %w", err)
	}
	return nil
}

func ParseManifest(reader io.Reader) (Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(io.LimitReader(reader, maxManifestBytes+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("invalid plugin manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return manifest, errors.New("plugin manifest must contain exactly one JSON object")
	}
	if err := ValidateManifest(manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

func ValidateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != 1 {
		return errors.New("unsupported plugin schema version")
	}
	if !identifierPattern.MatchString(manifest.ID) {
		return errors.New("plugin id must use lowercase letters, numbers, dots, underscores, or hyphens")
	}
	if strings.TrimSpace(manifest.Name) == "" || len(manifest.Name) > 80 {
		return errors.New("plugin name is required and must be at most 80 characters")
	}
	if strings.TrimSpace(manifest.Version) == "" || len(manifest.Version) > 40 {
		return errors.New("plugin version is required")
	}
	if len(manifest.Description) > 500 || len(manifest.Publisher) > 100 {
		return errors.New("plugin description or publisher is too long")
	}
	if manifest.UI != nil {
		switch manifest.UI.Mode {
		case "native":
			if manifest.UI.WebPath != "" {
				return errors.New("native plugin UI cannot declare a web path")
			}
		case "web", "both":
			if err := validateEndpoint(manifest.UI.WebPath); err != nil {
				return fmt.Errorf("invalid plugin web path: %w", err)
			}
		default:
			return errors.New("plugin UI mode must be native, web, or both")
		}
		if manifest.UI.DefaultView != "" && manifest.UI.DefaultView != "native" && manifest.UI.DefaultView != "web" {
			return errors.New("plugin default view must be native or web")
		}
		if manifest.UI.Mode == "native" && manifest.UI.DefaultView == "web" || manifest.UI.Mode == "web" && manifest.UI.DefaultView == "native" {
			return errors.New("plugin default view is not available in its UI mode")
		}
	}
	if manifest.Server.StartMode == "" {
		manifest.Server.StartMode = "external"
	}
	switch manifest.Server.StartMode {
	case "external", "manual", "on_demand", "automatic":
	default:
		return errors.New("plugin start mode must be external, manual, on_demand, or automatic")
	}
	if manifest.Server.StartMode != "external" && manifest.Server.Runtime == nil {
		return errors.New("managed plugin servers require a runtime command")
	}
	if runtime := manifest.Server.Runtime; runtime != nil {
		if runtime.StartupTimeoutSeconds < 0 || runtime.StartupTimeoutSeconds > 120 || runtime.ShutdownTimeoutSeconds < 0 || runtime.ShutdownTimeoutSeconds > 30 {
			return errors.New("plugin runtime timeouts are outside the allowed range")
		}
		for _, command := range []*Command{runtime.Windows, runtime.Linux, runtime.Darwin} {
			if command == nil {
				continue
			}
			if strings.TrimSpace(command.Executable) == "" || !filepath.IsAbs(command.Executable) {
				return errors.New("plugin runtime executable must be an absolute path")
			}
			if command.WorkingDirectory != "" && !filepath.IsAbs(command.WorkingDirectory) {
				return errors.New("plugin runtime working directory must be an absolute path")
			}
			if len(command.Args) > 40 {
				return errors.New("plugin runtime contains too many arguments")
			}
			for _, argument := range command.Args {
				if len(argument) > 2048 || strings.ContainsRune(argument, 0) {
					return errors.New("plugin runtime contains an invalid argument")
				}
			}
		}
	}
	if err := validateBaseURL(manifest.Server.BaseURL); err != nil {
		return err
	}
	if err := validateEndpoint(manifest.Server.HealthEndpoint); err != nil {
		return fmt.Errorf("invalid health endpoint: %w", err)
	}
	if len(manifest.Permissions) > 30 || len(manifest.Actions) > 40 || len(manifest.Panels) > 12 {
		return errors.New("plugin manifest contains too many capabilities")
	}

	permissions := make(map[string]bool, len(manifest.Permissions))
	for _, permission := range manifest.Permissions {
		if !identifierPattern.MatchString(permission) {
			return fmt.Errorf("invalid permission %q", permission)
		}
		permissions[permission] = true
	}
	actions := make(map[string]bool, len(manifest.Actions))
	for _, action := range manifest.Actions {
		if !identifierPattern.MatchString(action.ID) || actions[action.ID] {
			return fmt.Errorf("invalid or duplicate action id %q", action.ID)
		}
		actions[action.ID] = true
		method := strings.ToUpper(action.Method)
		if method != http.MethodGet && method != http.MethodPost {
			return fmt.Errorf("action %s uses an unsupported method", action.ID)
		}
		if err := validateEndpoint(action.Path); err != nil {
			return fmt.Errorf("action %s has an invalid endpoint: %w", action.ID, err)
		}
		if !permissions[action.Permission] {
			return fmt.Errorf("action %s requests an undeclared permission", action.ID)
		}
		if action.TimeoutSeconds < 0 || action.TimeoutSeconds > 120 {
			return fmt.Errorf("action %s timeout must be between 1 and 120 seconds", action.ID)
		}
		if action.BodyType == "" {
			action.BodyType = "json"
		}
		if action.ResponseType == "" {
			action.ResponseType = "json"
		}
		if action.BodyType != "json" && action.BodyType != "binary" {
			return fmt.Errorf("action %s has an unsupported body type", action.ID)
		}
		if action.ResponseType != "json" && action.ResponseType != "image" && action.ResponseType != "download" {
			return fmt.Errorf("action %s has an unsupported response type", action.ID)
		}
		if action.BodyType == "binary" {
			if action.Binary == nil || !identifierPattern.MatchString(action.Binary.FileField) {
				return fmt.Errorf("action %s requires binary upload settings", action.ID)
			}
			if !safeHeaderName(action.Binary.FilenameHeader) || (action.Binary.MetadataHeader != "" && !safeHeaderName(action.Binary.MetadataHeader)) {
				return fmt.Errorf("action %s contains an invalid binary header", action.ID)
			}
			if action.Binary.MaxBytes < 0 || action.Binary.MaxBytes > maxPluginUploadBytes {
				return fmt.Errorf("action %s upload limit is invalid", action.ID)
			}
		}
		if len(action.Fields) > 30 {
			return fmt.Errorf("action %s contains too many fields", action.ID)
		}
		fieldIDs := map[string]bool{}
		for _, field := range action.Fields {
			if !identifierPattern.MatchString(field.ID) || fieldIDs[field.ID] {
				return fmt.Errorf("action %s contains an invalid or duplicate field", action.ID)
			}
			fieldIDs[field.ID] = true
			switch field.Type {
			case "text", "textarea", "number", "select", "checkbox", "file", "url":
			default:
				return fmt.Errorf("action %s field %s has an unsupported type", action.ID, field.ID)
			}
			if field.Transform != "" && field.Transform != "cards_lines" {
				return fmt.Errorf("action %s field %s has an unsupported transform", action.ID, field.ID)
			}
		}
	}
	for _, panel := range manifest.Panels {
		if !identifierPattern.MatchString(panel.ID) || strings.TrimSpace(panel.Title) == "" {
			return errors.New("plugin panel id and title are required")
		}
		for _, actionID := range panel.Actions {
			if !actions[actionID] {
				return fmt.Errorf("panel %s references unknown action %s", panel.ID, actionID)
			}
		}
	}
	return nil
}

func safeHeaderName(value string) bool {
	if value == "" || len(value) > 80 {
		return false
	}
	for _, r := range value {
		if !(r == '-' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			return false
		}
	}
	return true
}

func validateBaseURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("plugin server must be a plain HTTP loopback URL")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return errors.New("plugin base URL cannot contain a path")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("plugin server must use localhost, 127.0.0.1, or ::1")
	}
	return nil
}

func validateEndpoint(path string) error {
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.Contains(path, "\\") || strings.Contains(path, "..") || strings.ContainsAny(path, "?#") {
		return errors.New("endpoint must be a simple absolute path")
	}
	return nil
}

func (m *Manager) Install(manifest Manifest) error {
	if err := ValidateManifest(manifest); err != nil {
		return err
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`INSERT INTO plugins(id, manifest, enabled) VALUES(?,?,0)
		ON CONFLICT(id) DO UPDATE SET manifest=excluded.manifest, updated_at=CURRENT_TIMESTAMP`, manifest.ID, string(data))
	if err != nil {
		return fmt.Errorf("save plugin: %w", err)
	}
	m.audit(manifest.ID, "install", true, 0, "manifest installed")
	return nil
}

func (m *Manager) List() ([]Installed, error) {
	rows, err := m.db.Query(`SELECT manifest, enabled, installed_at, updated_at FROM plugins ORDER BY installed_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Installed{}
	for rows.Next() {
		var item Installed
		var raw string
		if err := rows.Scan(&raw, &item.Enabled, &item.InstalledAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &item.Manifest); err != nil {
			return nil, fmt.Errorf("read installed plugin: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *Manager) Get(id string) (Installed, error) {
	var item Installed
	var raw string
	err := m.db.QueryRow(`SELECT manifest, enabled, installed_at, updated_at FROM plugins WHERE id=?`, id).
		Scan(&raw, &item.Enabled, &item.InstalledAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	if err := json.Unmarshal([]byte(raw), &item.Manifest); err != nil {
		return item, err
	}
	return item, nil
}

func (m *Manager) SetEnabled(id string, enabled bool) error {
	result, err := m.db.Exec(`UPDATE plugins SET enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, enabled, id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return sql.ErrNoRows
	}
	m.audit(id, map[bool]string{true: "enable", false: "disable"}[enabled], true, 0, "")
	if !enabled {
		_ = m.Stop(id)
	} else if plugin, getErr := m.Get(id); getErr == nil && plugin.Manifest.Server.StartMode == "automatic" {
		go func() { _ = m.Start(context.Background(), id) }()
	}
	return nil
}

func (m *Manager) SetStartMode(id, mode string) error {
	plugin, err := m.Get(id)
	if err != nil {
		return err
	}
	switch mode {
	case "external", "manual", "on_demand", "automatic":
	default:
		return errors.New("invalid plugin runtime mode")
	}
	if mode != "external" && plugin.Manifest.Server.Runtime == nil {
		return errors.New("this plugin has no managed runtime command")
	}
	plugin.Manifest.Server.StartMode = mode
	data, err := json.Marshal(plugin.Manifest)
	if err != nil {
		return err
	}
	if _, err = m.db.Exec(`UPDATE plugins SET manifest=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, string(data), id); err != nil {
		return err
	}
	m.audit(id, "runtime-mode", true, 0, mode)
	if mode == "external" {
		_ = m.Stop(id)
	}
	if mode == "automatic" && plugin.Enabled {
		go func() { _ = m.Start(context.Background(), id) }()
	}
	return nil
}

func (m *Manager) Remove(id string) error {
	_ = m.Stop(id)
	result, err := m.db.Exec(`DELETE FROM plugins WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return sql.ErrNoRows
	}
	m.audit(id, "uninstall", true, 0, "")
	return nil
}

func (m *Manager) Health(ctx context.Context, id string) (any, int, error) {
	plugin, err := m.Get(id)
	if err != nil {
		return nil, 0, err
	}
	if !plugin.Enabled {
		return nil, 0, errors.New("plugin is disabled")
	}
	if plugin.Manifest.Server.StartMode == "on_demand" {
		if err := m.Start(ctx, id); err != nil {
			return nil, 0, err
		}
	}
	return m.request(ctx, plugin.Manifest, http.MethodGet, plugin.Manifest.Server.HealthEndpoint, nil, 5*time.Second)
}

func (m *Manager) Invoke(ctx context.Context, pluginID, actionID string, input map[string]any) (any, int, error) {
	plugin, err := m.Get(pluginID)
	if err != nil {
		return nil, 0, err
	}
	if !plugin.Enabled {
		return nil, 0, errors.New("plugin is disabled")
	}
	if plugin.Manifest.Server.StartMode == "on_demand" {
		if err := m.Start(ctx, pluginID); err != nil {
			return nil, 0, err
		}
	}
	var selected *Action
	for i := range plugin.Manifest.Actions {
		if plugin.Manifest.Actions[i].ID == actionID {
			selected = &plugin.Manifest.Actions[i]
			break
		}
	}
	if selected == nil {
		return nil, 0, errors.New("plugin action was not found")
	}
	clean, err := validateInput(*selected, input)
	if err != nil {
		return nil, 0, err
	}
	payload := clean
	if strings.EqualFold(selected.Method, http.MethodPost) {
		payload = expandDottedInput(clean)
	}
	timeout := effectiveActionTimeout(*selected)
	data, status, requestErr := m.request(ctx, plugin.Manifest, strings.ToUpper(selected.Method), selected.Path, payload, timeout)
	m.audit(pluginID, actionID, requestErr == nil && status >= 200 && status < 300, status, errorMessage(requestErr))
	return data, status, requestErr
}

func effectiveActionTimeout(action Action) time.Duration {
	if action.TimeoutSeconds > 0 {
		return time.Duration(action.TimeoutSeconds) * time.Second
	}
	// Compatibility for manifests installed before timeout_seconds existed.
	// Hardware discovery and calibration are predictably slower than ordinary
	// local JSON actions but remain strictly bounded.
	switch {
	case strings.Contains(action.Path, "/scan"):
		return 60 * time.Second
	case strings.Contains(action.Path, "/calibration"):
		return 120 * time.Second
	case strings.Contains(action.Path, "/latex"):
		return 90 * time.Second
	default:
		return 45 * time.Second
	}
}

func validateInput(action Action, input map[string]any) (map[string]any, error) {
	if input == nil {
		input = map[string]any{}
	}
	allowed := make(map[string]Field, len(action.Fields))
	for _, field := range action.Fields {
		allowed[field.ID] = field
		if field.Required {
			value, ok := input[field.ID]
			if !ok || strings.TrimSpace(fmt.Sprint(value)) == "" {
				return nil, fmt.Errorf("%s is required", field.Label)
			}
		}
	}
	clean := make(map[string]any, len(input))
	for key, value := range input {
		field, ok := allowed[key]
		if !ok {
			return nil, fmt.Errorf("unexpected field %s", key)
		}
		if len(fmt.Sprint(value)) > 100000 {
			return nil, fmt.Errorf("field %s is too large", field.Label)
		}
		clean[key] = value
	}
	return clean, nil
}

func expandDottedInput(input map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range input {
		parts := strings.Split(key, ".")
		cursor := result
		for _, part := range parts[:len(parts)-1] {
			next, ok := cursor[part].(map[string]any)
			if !ok {
				next = make(map[string]any)
				cursor[part] = next
			}
			cursor = next
		}
		cursor[parts[len(parts)-1]] = value
	}
	return result
}

func (m *Manager) request(ctx context.Context, manifest Manifest, method, endpoint string, input map[string]any, timeout time.Duration) (any, int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	target := strings.TrimRight(manifest.Server.BaseURL, "/") + endpoint
	var body io.Reader
	if method == http.MethodGet && len(input) > 0 {
		parsed, _ := url.Parse(target)
		query := parsed.Query()
		keys := make([]string, 0, len(input))
		for key := range input {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			query.Set(key, fmt.Sprint(input[key]))
		}
		parsed.RawQuery = query.Encode()
		target = parsed.String()
	} else if method == http.MethodPost {
		encoded, err := json.Marshal(input)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, 0, err
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "DNSFilter-PluginHost/1")
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("plugin server is unreachable: %w", err)
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if len(data) > maxResponseBytes {
		return nil, resp.StatusCode, errors.New("plugin response is too large")
	}
	var decoded any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, resp.StatusCode, errors.New("plugin returned a non-JSON response")
		}
	} else {
		decoded = map[string]any{"ok": resp.StatusCode >= 200 && resp.StatusCode < 300}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decoded, resp.StatusCode, fmt.Errorf("plugin returned HTTP %d", resp.StatusCode)
	}
	return decoded, resp.StatusCode, nil
}

func (m *Manager) Events(pluginID string, limit int) ([]Event, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := m.db.Query(`SELECT id, plugin_id, action, success, status, message, created_at
		FROM plugin_events WHERE (?='' OR plugin_id=?) ORDER BY id DESC LIMIT ?`, pluginID, pluginID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Event{}
	for rows.Next() {
		var item Event
		if err := rows.Scan(&item.ID, &item.PluginID, &item.Action, &item.Success, &item.Status, &item.Message, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *Manager) audit(pluginID, action string, success bool, status int, message string) {
	if len(message) > 300 {
		message = message[:300]
	}
	_, _ = m.db.Exec(`INSERT INTO plugin_events(plugin_id, action, success, status, message) VALUES(?,?,?,?,?)`, pluginID, action, success, status, message)
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
