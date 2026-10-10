package discipline

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

// FocusCapsule is a reusable deep-work environment shared by Discipline and
// the browser extension. Application paths are persisted now and are applied
// by the OS network controller in a separate, recoverable enforcement step.
type FocusCapsule struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	AllowedApps     []string `json:"allowed_apps"`
	AllowedDomains  []string `json:"allowed_domains"`
	YouTubeChannels []string `json:"youtube_channels"`
	BlockStyle      string   `json:"block_style"`
	Strict          bool     `json:"strict"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type FocusCapsuleInput struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	AllowedApps     []string `json:"allowed_apps"`
	AllowedDomains  []string `json:"allowed_domains"`
	YouTubeChannels []string `json:"youtube_channels"`
	BlockStyle      string   `json:"block_style"`
	Strict          bool     `json:"strict"`
}

type FocusPageDecision struct {
	Active     bool   `json:"active"`
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason,omitempty"`
	BlockStyle string `json:"block_style,omitempty"`
	Capsule    string `json:"capsule,omitempty"`
	SessionID  int64  `json:"session_id,omitempty"`
}

func jsonStrings(values []string) string {
	clean := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[strings.ToLower(value)] {
			continue
		}
		seen[strings.ToLower(value)] = true
		clean = append(clean, value)
	}
	b, _ := json.Marshal(clean)
	return string(b)
}

func parseJSONStrings(raw string) []string {
	var values []string
	_ = json.Unmarshal([]byte(raw), &values)
	if values == nil {
		values = []string{}
	}
	return values
}

func normalizeBlockStyle(style string) string {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "roast", "meme", "strict":
		return strings.ToLower(strings.TrimSpace(style))
	default:
		return "motivation"
	}
}

func (m *Manager) ListFocusCapsules() ([]FocusCapsule, error) {
	rows, err := m.db.Query(`SELECT id,name,description,allowed_apps,allowed_domains,youtube_channels,block_style,strict,created_at,updated_at FROM discipline_focus_capsules ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []FocusCapsule{}
	for rows.Next() {
		var item FocusCapsule
		var apps, domains, channels string
		var strict int
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &apps, &domains, &channels, &item.BlockStyle, &strict, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.AllowedApps, item.AllowedDomains, item.YouTubeChannels = parseJSONStrings(apps), parseJSONStrings(domains), parseJSONStrings(channels)
		item.Strict = strict == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *Manager) FocusCapsule(id int64) (FocusCapsule, error) {
	var item FocusCapsule
	var apps, domains, channels string
	var strict int
	err := m.db.QueryRow(`SELECT id,name,description,allowed_apps,allowed_domains,youtube_channels,block_style,strict,created_at,updated_at FROM discipline_focus_capsules WHERE id=?`, id).
		Scan(&item.ID, &item.Name, &item.Description, &apps, &domains, &channels, &item.BlockStyle, &strict, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.AllowedApps, item.AllowedDomains, item.YouTubeChannels = parseJSONStrings(apps), parseJSONStrings(domains), parseJSONStrings(channels)
	item.Strict = strict == 1
	return item, nil
}

func (m *Manager) SaveFocusCapsule(id int64, in FocusCapsuleInput) (FocusCapsule, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return FocusCapsule{}, errors.New("capsule name is required")
	}
	in.BlockStyle = normalizeBlockStyle(in.BlockStyle)
	strict := 0
	if in.Strict {
		strict = 1
	}
	if id == 0 {
		result, err := m.db.Exec(`INSERT INTO discipline_focus_capsules(name,description,allowed_apps,allowed_domains,youtube_channels,block_style,strict) VALUES(?,?,?,?,?,?,?)`, in.Name, strings.TrimSpace(in.Description), jsonStrings(in.AllowedApps), jsonStrings(in.AllowedDomains), jsonStrings(in.YouTubeChannels), in.BlockStyle, strict)
		if err != nil {
			return FocusCapsule{}, err
		}
		id, _ = result.LastInsertId()
	} else {
		if _, err := m.db.Exec(`UPDATE discipline_focus_capsules SET name=?,description=?,allowed_apps=?,allowed_domains=?,youtube_channels=?,block_style=?,strict=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, in.Name, strings.TrimSpace(in.Description), jsonStrings(in.AllowedApps), jsonStrings(in.AllowedDomains), jsonStrings(in.YouTubeChannels), in.BlockStyle, strict, id); err != nil {
			return FocusCapsule{}, err
		}
	}
	return m.FocusCapsule(id)
}

func (m *Manager) DeleteFocusCapsule(id int64) error {
	var active int
	_ = m.db.QueryRow(`SELECT COUNT(*) FROM discipline_focus_sessions WHERE capsule_id=? AND status='active'`, id).Scan(&active)
	if active > 0 {
		return errors.New("end the active focus session before deleting this capsule")
	}
	_, err := m.db.Exec(`DELETE FROM discipline_focus_capsules WHERE id=?`, id)
	return err
}

func (m *Manager) ActiveFocusCapsule() (FocusCapsule, bool) {
	m.focusMu.RLock()
	sessionID := m.focusID
	m.focusMu.RUnlock()
	if sessionID == 0 {
		return FocusCapsule{}, false
	}
	var capsuleID int64
	if m.db.QueryRow(`SELECT capsule_id FROM discipline_focus_sessions WHERE id=? AND status='active'`, sessionID).Scan(&capsuleID) != nil || capsuleID == 0 {
		return FocusCapsule{}, false
	}
	item, err := m.FocusCapsule(capsuleID)
	return item, err == nil
}

func (m *Manager) ActiveFocusRemaining() time.Duration {
	m.focusMu.RLock()
	sessionID := m.focusID
	m.focusMu.RUnlock()
	if sessionID == 0 {
		return 0
	}
	var planned int
	var started string
	if m.db.QueryRow(`SELECT planned_minutes,started_at FROM discipline_focus_sessions WHERE id=? AND status='active'`, sessionID).Scan(&planned, &started) != nil {
		return 0
	}
	start, err := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSuffix(started, "Z"), time.Local)
	if err != nil {
		start, err = time.Parse(time.RFC3339, started)
	}
	if err != nil {
		return 0
	}
	remaining := time.Duration(planned)*time.Minute - time.Since(start)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func cleanDomain(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "*.")
	if parsed, err := url.Parse(value); err == nil && parsed.Hostname() != "" {
		value = parsed.Hostname()
	}
	return strings.Trim(value, ".")
}

func domainMatches(host string, allowed []string) bool {
	host = cleanDomain(host)
	for _, candidate := range allowed {
		candidate = cleanDomain(candidate)
		if candidate != "" && (host == candidate || strings.HasSuffix(host, "."+candidate)) {
			return true
		}
	}
	return false
}

func channelMatches(observed []string, allowed []string) bool {
	for _, a := range allowed {
		a = strings.ToLower(strings.Trim(strings.TrimSpace(a), "/"))
		if a == "" {
			continue
		}
		for _, value := range observed {
			value = strings.ToLower(strings.Trim(strings.TrimSpace(value), "/"))
			if value == a || strings.HasSuffix(value, "/"+a) || strings.HasSuffix(a, "/"+value) {
				return true
			}
		}
	}
	return false
}

func preferredChannel(values []string) string {
	for _, value := range values {
		value = strings.Trim(strings.TrimSpace(value), "/")
		if strings.HasPrefix(strings.ToLower(value), "channel/") || strings.HasPrefix(value, "@") {
			return value
		}
	}
	if len(values) > 0 {
		return strings.Trim(strings.TrimSpace(values[0]), "/")
	}
	return ""
}

// EvaluateFocusPage is the single source of truth used by the existing Hard
// System extension. The extension extracts browser-only YouTube channel data;
// the local service owns the active capsule and final decision.
func (m *Manager) EvaluateFocusPage(rawURL string, channels []string) FocusPageDecision {
	m.focusMu.RLock()
	sessionID := m.focusID
	m.focusMu.RUnlock()
	if sessionID == 0 {
		return FocusPageDecision{Allowed: true}
	}
	var capsuleID int64
	if err := m.db.QueryRow(`SELECT capsule_id FROM discipline_focus_sessions WHERE id=? AND status='active'`, sessionID).Scan(&capsuleID); err != nil || capsuleID == 0 {
		return FocusPageDecision{Active: true, Allowed: true, SessionID: sessionID}
	}
	capsule, err := m.FocusCapsule(capsuleID)
	if err != nil {
		return FocusPageDecision{Active: true, Allowed: true, SessionID: sessionID}
	}
	decision := FocusPageDecision{Active: true, Allowed: true, BlockStyle: capsule.BlockStyle, Capsule: capsule.Name, SessionID: sessionID}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return decision
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" || host == "127.0.0.1" || strings.HasSuffix(host, ".localhost") {
		return decision
	}
	isYouTube := host == "youtu.be" || host == "youtube.com" || strings.HasSuffix(host, ".youtube.com")
	if isYouTube {
		allowedChannels := capsule.YouTubeChannels
		if len(allowedChannels) == 0 {
			var locked string
			_ = m.db.QueryRow(`SELECT locked_youtube_channel FROM discipline_focus_sessions WHERE id=?`, sessionID).Scan(&locked)
			if locked == "" && (parsed.Path == "/watch" || host == "youtu.be") {
				locked = preferredChannel(channels)
				if locked != "" {
					_, _ = m.db.Exec(`UPDATE discipline_focus_sessions SET locked_youtube_channel=? WHERE id=? AND locked_youtube_channel=''`, locked, sessionID)
				}
			}
			if locked != "" {
				allowedChannels = []string{locked}
			}
		}
		if len(allowedChannels) > 0 && (parsed.Path == "/watch" || host == "youtu.be") && channelMatches(channels, allowedChannels) {
			return decision
		}
		decision.Allowed = false
		if len(allowedChannels) == 0 {
			decision.Reason = "Open the study video selected for this capsule so its channel can be identified."
		} else {
			decision.Reason = "This YouTube page is outside the channel locked to this focus capsule."
		}
		return decision
	}
	if domainMatches(host, capsule.AllowedDomains) {
		return decision
	}
	if capsule.Strict {
		decision.Allowed = false
		decision.Reason = "This website is outside the allowed resources for this focus capsule."
	}
	return decision
}
