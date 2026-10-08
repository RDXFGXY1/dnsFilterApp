package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxPluginContentBytes = 20 << 20

func (m *Manager) actionFor(ctx context.Context, pluginID, actionID string) (Installed, Action, error) {
	plugin, err := m.Get(pluginID)
	if err != nil {
		return plugin, Action{}, err
	}
	if !plugin.Enabled {
		return plugin, Action{}, errors.New("plugin is disabled")
	}
	if plugin.Manifest.Server.StartMode == "on_demand" {
		if err := m.Start(ctx, pluginID); err != nil {
			return plugin, Action{}, err
		}
	}
	for _, action := range plugin.Manifest.Actions {
		if action.ID == actionID {
			return plugin, action, nil
		}
	}
	return plugin, Action{}, errors.New("plugin action was not found")
}

func (m *Manager) InvokeFile(ctx context.Context, pluginID, actionID, filename string, content io.Reader, size int64, input map[string]any) (any, int, error) {
	plugin, action, err := m.actionFor(ctx, pluginID, actionID)
	if err != nil {
		return nil, 0, err
	}
	if action.BodyType != "binary" || action.Binary == nil {
		return nil, 0, errors.New("plugin action does not accept a file")
	}
	limit := action.Binary.MaxBytes
	if limit <= 0 {
		limit = maxPluginUploadBytes
	}
	if size < 0 || size > limit {
		return nil, 0, fmt.Errorf("file exceeds the %d MB action limit", limit/(1<<20))
	}
	cleanInput := make(map[string]any, len(input)+1)
	for key, value := range input {
		cleanInput[key] = value
	}
	cleanInput[action.Binary.FileField] = filename
	clean, err := validateInput(action, cleanInput)
	if err != nil {
		return nil, 0, err
	}
	delete(clean, action.Binary.FileField)
	expanded := expandDottedInput(clean)
	headers := http.Header{"Content-Type": []string{"application/octet-stream"}}
	headers.Set(action.Binary.FilenameHeader, filename)
	if action.Binary.MetadataHeader != "" {
		metadata := any(expanded)
		if action.Binary.MetadataRoot != "" {
			metadata = expanded[action.Binary.MetadataRoot]
		}
		encoded, marshalErr := json.Marshal(metadata)
		if marshalErr != nil {
			return nil, 0, marshalErr
		}
		headers.Set(action.Binary.MetadataHeader, string(encoded))
	}
	data, contentType, status, requestErr := m.requestContent(ctx, plugin.Manifest, action, content, headers)
	var decoded any
	if len(data) > 0 {
		_ = json.Unmarshal(data, &decoded)
	}
	if decoded == nil {
		decoded = map[string]any{"ok": requestErr == nil}
	}
	m.audit(pluginID, actionID, requestErr == nil, status, errorMessage(requestErr))
	_ = contentType
	return decoded, status, requestErr
}

func (m *Manager) InvokeContent(ctx context.Context, pluginID, actionID string, input map[string]any) ([]byte, string, int, error) {
	plugin, action, err := m.actionFor(ctx, pluginID, actionID)
	if err != nil {
		return nil, "", 0, err
	}
	if action.ResponseType != "image" && action.ResponseType != "download" {
		return nil, "", 0, errors.New("plugin action does not return content")
	}
	clean, err := validateInput(action, input)
	if err != nil {
		return nil, "", 0, err
	}
	payload, err := json.Marshal(expandDottedInput(clean))
	if err != nil {
		return nil, "", 0, err
	}
	data, contentType, status, requestErr := m.requestContent(ctx, plugin.Manifest, action, bytes.NewReader(payload), http.Header{"Content-Type": []string{"application/json"}})
	m.audit(pluginID, actionID, requestErr == nil, status, errorMessage(requestErr))
	return data, contentType, status, requestErr
}

func (m *Manager) requestContent(ctx context.Context, manifest Manifest, action Action, body io.Reader, headers http.Header) ([]byte, string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, effectiveActionTimeout(action))
	defer cancel()
	target := strings.TrimRight(manifest.Server.BaseURL, "/") + action.Path
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(action.Method), target, body)
	if err != nil {
		return nil, "", 0, err
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Accept", "application/json, image/*, application/pdf")
	req.Header.Set("User-Agent", "DNSFilter-PluginHost/1")
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("plugin server is unreachable: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPluginContentBytes+1))
	if err != nil {
		return nil, resp.Header.Get("Content-Type"), resp.StatusCode, err
	}
	if len(data) > maxPluginContentBytes {
		return nil, resp.Header.Get("Content-Type"), resp.StatusCode, errors.New("plugin content response is too large")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return data, resp.Header.Get("Content-Type"), resp.StatusCode, fmt.Errorf("plugin returned HTTP %d", resp.StatusCode)
	}
	return data, resp.Header.Get("Content-Type"), resp.StatusCode, nil
}
