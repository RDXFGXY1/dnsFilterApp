package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
)

type briefingProvider struct {
	ID                          int64
	Provider, Name, Model, Base string
	Key                         []byte
}
type briefingItem struct {
	Title    string `json:"title"`
	Status   string `json:"status,omitempty"`
	Priority string `json:"priority,omitempty"`
	Due      string `json:"due,omitempty"`
	Detail   string `json:"detail,omitempty"`
}
type briefingSnapshot struct {
	GeneratedAt string         `json:"generated_at"`
	Tasks       []briefingItem `json:"tasks"`
	Goals       []briefingItem `json:"goals"`
	Projects    []briefingItem `json:"projects"`
	Habits      []briefingItem `json:"habits"`
	Journal     []briefingItem `json:"recent_journal"`
}
type aiBriefing struct {
	Subject     string `json:"subject"`
	Body        string `json:"body"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	ActionLabel string `json:"action_label"`
	ActionView  string `json:"action_view"`
}

func (s *Server) defaultBriefingProvider() (briefingProvider, error) {
	var p briefingProvider
	var sealed []byte
	err := s.db.GetDB().QueryRow(`SELECT id,provider,name,model,base_url,api_key_encrypted FROM ai_provider_configs WHERE enabled=1 ORDER BY is_default DESC,id LIMIT 1`).Scan(&p.ID, &p.Provider, &p.Name, &p.Model, &p.Base, &sealed)
	if err != nil {
		return p, err
	}
	if s.vault == nil {
		return p, errors.New("Secure Vault unavailable")
	}
	p.Key, err = s.vault.Decrypt(sealed)
	return p, err
}

func (s *Server) disciplineBriefingSnapshot() (briefingSnapshot, error) {
	x := briefingSnapshot{GeneratedAt: time.Now().Format(time.RFC3339), Tasks: []briefingItem{}, Goals: []briefingItem{}, Projects: []briefingItem{}, Habits: []briefingItem{}, Journal: []briefingItem{}}
	db := s.db.GetDB()
	rows, e := db.Query(`SELECT title,status,priority,due_date,CASE WHEN my_day=1 THEN 'My Day' ELSE '' END FROM discipline_tasks WHERE archived=0 AND done=0 ORDER BY my_day DESC,CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END,due_date LIMIT 40`)
	if e != nil {
		return x, e
	}
	for rows.Next() {
		var i briefingItem
		if e = rows.Scan(&i.Title, &i.Status, &i.Priority, &i.Due, &i.Detail); e != nil {
			rows.Close()
			return x, e
		}
		x.Tasks = append(x.Tasks, i)
	}
	rows.Close()
	queries := []struct {
		q   string
		dst *[]briefingItem
	}{{`SELECT name,status,target_date,description FROM discipline_goals WHERE status<>'done' ORDER BY target_date LIMIT 20`, &x.Goals}, {`SELECT name,status,target_date,objective FROM discipline_projects WHERE status<>'done' ORDER BY target_date LIMIT 20`, &x.Projects}, {`SELECT h.name,h.frequency,'',CASE WHEN c.habit_id IS NULL THEN 'not checked today' ELSE 'completed today' END FROM discipline_habits h LEFT JOIN discipline_habit_checkins c ON c.habit_id=h.id AND c.day=date('now','localtime') ORDER BY h.name LIMIT 25`, &x.Habits}, {`SELECT title,day,'',substr(content,1,500) FROM discipline_journal_entries ORDER BY day DESC,created_at DESC LIMIT 10`, &x.Journal}}
	for _, q := range queries {
		rows, e = db.Query(q.q)
		if e != nil {
			continue
		}
		for rows.Next() {
			var i briefingItem
			if rows.Scan(&i.Title, &i.Status, &i.Due, &i.Detail) == nil {
				*q.dst = append(*q.dst, i)
			}
		}
		rows.Close()
	}
	return x, nil
}

func briefingPrompt(x briefingSnapshot) (string, error) {
	b, e := json.Marshal(x)
	if e != nil {
		return "", e
	}
	return `You are Discipline Intelligence, a concise and honest personal execution analyst. Analyze only the supplied local data. Identify the most useful pattern, risk, reminder, or next decision. Never claim facts absent from the data. Return exactly one JSON object with these keys: subject, body, category, severity, action_label, action_view. severity must be normal, attention, or success. action_view must be one of today, tasks, goals, habits, reminders, journal, analytics, projects, activity, command, home. body must be plain text, useful, under 1200 characters, and may use bullet lines. Do not wrap JSON in markdown.\n\nLOCAL DISCIPLINE SNAPSHOT:\n` + string(b), nil
}

func (s *Server) generateIntelligenceBriefing(ctx context.Context) (int64, string, error) {
	p, e := s.defaultBriefingProvider()
	if errors.Is(e, sql.ErrNoRows) {
		id, x := s.discipline.GenerateDailyBriefing()
		return id, "local", x
	}
	if e != nil {
		// The API key is deliberately unavailable while the vault is locked.
		// A daily briefing is productivity data, so it must still work using the
		// deterministic local analyzer instead of forcing a vault unlock.
		id, localErr := s.discipline.GenerateDailyBriefing()
		return id, "local_fallback", localErr
	}
	snap, e := s.disciplineBriefingSnapshot()
	if e != nil {
		return 0, "ai", e
	}
	prompt, e := briefingPrompt(snap)
	if e != nil {
		return 0, "ai", e
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	raw, e := callBriefingProvider(ctx, p, prompt)
	if e != nil {
		s.recordAIProviderResult(p.ID, false, friendlyAIProviderError(e))
		id, localErr := s.discipline.GenerateDailyBriefing()
		return id, "local_fallback", localErr
	}
	msg, e := parseAIBriefing(raw)
	if e != nil {
		s.recordAIProviderResult(p.ID, false, e.Error())
		id, localErr := s.discipline.GenerateDailyBriefing()
		return id, "local_fallback", localErr
	}
	id, e := s.discipline.CreateInboxMessage(discipline.InboxInput{Sender: p.Name, Subject: msg.Subject, Body: msg.Body, Category: msg.Category, Severity: msg.Severity, ActionLabel: msg.ActionLabel, ActionView: msg.ActionView})
	if e != nil {
		return 0, "ai", e
	}
	s.recordAIProviderResult(p.ID, true, "")
	return id, "ai", nil
}

// friendlyAIProviderError keeps temporary provider failures understandable and
// prevents a large upstream JSON response from leaking into UI notifications.
func friendlyAIProviderError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "http 429"):
		return "The AI provider is rate-limited. Local analysis was used instead."
	case strings.Contains(text, "http 500"), strings.Contains(text, "http 502"), strings.Contains(text, "http 503"), strings.Contains(text, "http 504"), strings.Contains(text, "unavailable"), strings.Contains(text, "high demand"):
		return "The AI provider is temporarily unavailable. Local analysis was used instead."
	case errors.Is(err, context.DeadlineExceeded), strings.Contains(text, "timeout"), strings.Contains(text, "deadline exceeded"):
		return "The AI provider timed out. Local analysis was used instead."
	default:
		return "The AI provider could not complete the request. Local analysis was used instead."
	}
}
func (s *Server) recordAIProviderResult(id int64, ok bool, detail string) {
	status := "connected"
	if !ok {
		status = "failed"
	}
	if len(detail) > 700 {
		detail = detail[:700]
	}
	_, _ = s.db.GetDB().Exec(`UPDATE ai_provider_configs SET last_status=?,last_error=?,last_tested_at=CURRENT_TIMESTAMP WHERE id=?`, status, detail, id)
}

func callBriefingProvider(ctx context.Context, p briefingProvider, prompt string) (string, error) {
	var endpoint string
	var payload any
	switch p.Provider {
	case "gemini":
		endpoint = strings.TrimRight(p.Base, "/") + "/models/" + url.PathEscape(p.Model) + ":generateContent"
		payload = map[string]any{"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]string{"text": prompt}}}}, "generationConfig": map[string]any{"temperature": 0.25, "responseMimeType": "application/json"}}
	case "openai", "xai":
		endpoint = strings.TrimRight(p.Base, "/") + "/responses"
		payload = map[string]any{"model": p.Model, "input": prompt, "store": false, "max_output_tokens": 1400}
	default:
		endpoint = strings.TrimRight(p.Base, "/") + "/chat/completions"
		payload = map[string]any{"model": p.Model, "messages": []any{map[string]string{"role": "user", "content": prompt}}, "temperature": 0.25}
	}
	b, _ := json.Marshal(payload)
	req, e := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(b))
	if e != nil {
		return "", e
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Provider == "gemini" {
		req.Header.Set("x-goog-api-key", string(p.Key))
	} else {
		req.Header.Set("Authorization", "Bearer "+string(p.Key))
	}
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	data, e := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if e != nil {
		return "", e
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%s returned HTTP %d: %s", p.Name, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if p.Provider == "gemini" {
		var x struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if json.Unmarshal(data, &x) != nil || len(x.Candidates) == 0 {
			return "", errors.New("Gemini returned no briefing")
		}
		var out strings.Builder
		for _, part := range x.Candidates[0].Content.Parts {
			out.WriteString(part.Text)
		}
		return out.String(), nil
	}
	if p.Provider == "compatible" {
		var x struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if json.Unmarshal(data, &x) != nil || len(x.Choices) == 0 {
			return "", errors.New("provider returned no briefing")
		}
		return x.Choices[0].Message.Content, nil
	}
	var x struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		OutputText string `json:"output_text"`
	}
	if json.Unmarshal(data, &x) != nil {
		return "", errors.New("provider returned an invalid response")
	}
	if x.OutputText != "" {
		return x.OutputText, nil
	}
	var out strings.Builder
	for _, o := range x.Output {
		for _, part := range o.Content {
			if part.Type == "output_text" || part.Type == "text" {
				out.WriteString(part.Text)
			}
		}
	}
	if out.Len() == 0 {
		return "", errors.New("provider returned no briefing text")
	}
	return out.String(), nil
}

func parseAIBriefing(raw string) (aiBriefing, error) {
	var x aiBriefing
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return x, errors.New("AI response did not contain a JSON briefing")
	}
	if e := json.Unmarshal([]byte(raw[start:end+1]), &x); e != nil {
		return x, fmt.Errorf("AI briefing JSON was invalid: %w", e)
	}
	x.Subject = strings.TrimSpace(x.Subject)
	x.Body = strings.TrimSpace(x.Body)
	x.Category = strings.TrimSpace(x.Category)
	x.ActionLabel = strings.TrimSpace(x.ActionLabel)
	if x.Subject == "" || x.Body == "" {
		return x, errors.New("AI briefing was missing subject or body")
	}
	if len(x.Subject) > 120 {
		x.Subject = x.Subject[:120]
	}
	if len(x.Body) > 4000 {
		x.Body = x.Body[:4000]
	}
	if x.Severity != "attention" && x.Severity != "success" {
		x.Severity = "normal"
	}
	if !validAIActionView(x.ActionView) {
		x.ActionView = ""
		x.ActionLabel = ""
	}
	if x.Category == "" {
		x.Category = "AI briefing"
	}
	return x, nil
}
func validAIActionView(x string) bool {
	switch x {
	case "today", "tasks", "goals", "habits", "reminders", "journal", "analytics", "projects", "activity", "command", "home":
		return true
	}
	return false
}
