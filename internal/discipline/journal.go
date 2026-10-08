package discipline

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type JournalType struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	EntryCount int    `json:"entry_count"`
	CreatedAt  string `json:"created_at"`
}

type JournalEntry struct {
	ID              int64  `json:"id"`
	TypeID          int64  `json:"type_id"`
	TypeName        string `json:"type_name"`
	TypeColor       string `json:"type_color"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	Day             string `json:"day"`
	DurationMinutes int    `json:"duration_minutes"`
	Mood            int    `json:"mood"`
	Focus           int    `json:"focus"`
	Tags            string `json:"tags"`
	ProjectID       int64  `json:"project_id"`
	ProjectName     string `json:"project_name"`
	Pinned          bool   `json:"pinned"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type JournalEntryInput struct {
	TypeID          int64  `json:"type_id"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	Day             string `json:"day"`
	DurationMinutes int    `json:"duration_minutes"`
	Mood            int    `json:"mood"`
	Focus           int    `json:"focus"`
	Tags            string `json:"tags"`
	ProjectID       int64  `json:"project_id"`
	Pinned          bool   `json:"pinned"`
}

type JournalFilters struct {
	Query  string
	TypeID int64
	From   string
	To     string
	Limit  int
}

type JournalDay struct {
	Day             string `json:"day"`
	Entries         int    `json:"entries"`
	DurationMinutes int    `json:"duration_minutes"`
}

type JournalStats struct {
	TotalEntries    int          `json:"total_entries"`
	ActiveDays      int          `json:"active_days"`
	CurrentStreak   int          `json:"current_streak"`
	LongestStreak   int          `json:"longest_streak"`
	DurationMinutes int          `json:"duration_minutes"`
	AverageMood     float64      `json:"average_mood"`
	AverageFocus    float64      `json:"average_focus"`
	Last28Days      []JournalDay `json:"last_28_days"`
}

func normalizeJournalDay(day string) (string, error) {
	day = strings.TrimSpace(day)
	if day == "" {
		return time.Now().Format("2006-01-02"), nil
	}
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return "", errors.New("day must use YYYY-MM-DD")
	}
	return day, nil
}

func normalizeJournalScore(n int) int {
	if n < 0 {
		return 0
	}
	if n > 10 {
		return 10
	}
	return n
}

func (m *Manager) ListJournalTypes() ([]JournalType, error) {
	rows, err := m.db.Query(`SELECT t.id,t.name,t.color,t.created_at,COUNT(e.id) FROM discipline_journal_types t LEFT JOIN discipline_journal_entries e ON e.type_id=t.id GROUP BY t.id ORDER BY lower(t.name)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []JournalType{}
	for rows.Next() {
		var item JournalType
		if err := rows.Scan(&item.ID, &item.Name, &item.Color, &item.CreatedAt, &item.EntryCount); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (m *Manager) CreateJournalType(name, color string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("log type name is required")
	}
	if len(name) > 40 {
		return 0, errors.New("log type name is too long")
	}
	color = strings.TrimSpace(color)
	if color == "" {
		color = "#4f73d9"
	}
	if !regexp.MustCompile(`^#[0-9a-fA-F]{6}$`).MatchString(color) {
		return 0, errors.New("color must be a six-digit hex value")
	}
	result, err := m.db.Exec(`INSERT INTO discipline_journal_types(name,color) VALUES(?,?)`, name, color)
	if err != nil {
		return 0, fmt.Errorf("create log type: %w", err)
	}
	return result.LastInsertId()
}

func (m *Manager) DeleteJournalType(id int64) error {
	if id <= 0 {
		return errors.New("invalid log type")
	}
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE discipline_journal_entries SET type_id=0 WHERE type_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM discipline_journal_types WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (m *Manager) CreateJournalEntry(input JournalEntryInput) (int64, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return 0, errors.New("entry title is required")
	}
	if len(input.Title) > 160 {
		return 0, errors.New("entry title is too long")
	}
	day, err := normalizeJournalDay(input.Day)
	if err != nil {
		return 0, err
	}
	if input.DurationMinutes < 0 || input.DurationMinutes > 1440 {
		return 0, errors.New("duration must be between 0 and 1440 minutes")
	}
	input.Mood = normalizeJournalScore(input.Mood)
	input.Focus = normalizeJournalScore(input.Focus)
	result, err := m.db.Exec(`INSERT INTO discipline_journal_entries(type_id,title,content,day,duration_minutes,mood,focus,tags,project_id,pinned) VALUES(?,?,?,?,?,?,?,?,?,?)`, input.TypeID, input.Title, strings.TrimSpace(input.Content), day, input.DurationMinutes, input.Mood, input.Focus, strings.TrimSpace(input.Tags), input.ProjectID, input.Pinned)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err == nil {
		m.LogEvent("journal_entry", input.ProjectID, 0)
	}
	return id, err
}

func (m *Manager) UpdateJournalEntry(id int64, input JournalEntryInput) error {
	if id <= 0 {
		return errors.New("invalid journal entry")
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return errors.New("entry title is required")
	}
	day, err := normalizeJournalDay(input.Day)
	if err != nil {
		return err
	}
	if input.DurationMinutes < 0 || input.DurationMinutes > 1440 {
		return errors.New("duration must be between 0 and 1440 minutes")
	}
	result, err := m.db.Exec(`UPDATE discipline_journal_entries SET type_id=?,title=?,content=?,day=?,duration_minutes=?,mood=?,focus=?,tags=?,project_id=?,pinned=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, input.TypeID, input.Title, strings.TrimSpace(input.Content), day, input.DurationMinutes, normalizeJournalScore(input.Mood), normalizeJournalScore(input.Focus), strings.TrimSpace(input.Tags), input.ProjectID, input.Pinned, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (m *Manager) DeleteJournalEntry(id int64) error {
	result, err := m.db.Exec(`DELETE FROM discipline_journal_entries WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (m *Manager) ListJournalEntries(filters JournalFilters) ([]JournalEntry, error) {
	where := []string{"1=1"}
	args := []any{}
	if filters.TypeID > 0 {
		where = append(where, "e.type_id=?")
		args = append(args, filters.TypeID)
	}
	if filters.From != "" {
		where = append(where, "e.day>=?")
		args = append(args, filters.From)
	}
	if filters.To != "" {
		where = append(where, "e.day<=?")
		args = append(args, filters.To)
	}
	if q := strings.TrimSpace(filters.Query); q != "" {
		where = append(where, "(e.title LIKE ? OR e.content LIKE ? OR e.tags LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}
	if filters.Limit <= 0 || filters.Limit > 500 {
		filters.Limit = 200
	}
	args = append(args, filters.Limit)
	rows, err := m.db.Query(`SELECT e.id,e.type_id,COALESCE(t.name,'General'),COALESCE(t.color,'#8993a4'),e.title,e.content,e.day,e.duration_minutes,e.mood,e.focus,e.tags,e.project_id,COALESCE(p.name,''),e.pinned,e.created_at,e.updated_at FROM discipline_journal_entries e LEFT JOIN discipline_journal_types t ON t.id=e.type_id LEFT JOIN discipline_projects p ON p.id=e.project_id WHERE `+strings.Join(where, " AND ")+` ORDER BY e.pinned DESC,e.day DESC,e.created_at DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []JournalEntry{}
	for rows.Next() {
		var e JournalEntry
		if err := rows.Scan(&e.ID, &e.TypeID, &e.TypeName, &e.TypeColor, &e.Title, &e.Content, &e.Day, &e.DurationMinutes, &e.Mood, &e.Focus, &e.Tags, &e.ProjectID, &e.ProjectName, &e.Pinned, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func streakFromJournalDays(days []string) (current, longest int) {
	if len(days) == 0 {
		return 0, 0
	}
	set := map[string]bool{}
	for _, d := range days {
		set[d] = true
	}
	for d := range set {
		t, err := time.Parse("2006-01-02", d)
		if err != nil {
			continue
		}
		n := 0
		for set[t.Format("2006-01-02")] {
			n++
			t = t.AddDate(0, 0, -1)
		}
		if n > longest {
			longest = n
		}
	}
	t := time.Now()
	if !set[t.Format("2006-01-02")] {
		t = t.AddDate(0, 0, -1)
	}
	for set[t.Format("2006-01-02")] {
		current++
		t = t.AddDate(0, 0, -1)
	}
	return
}

func (m *Manager) JournalStats() (JournalStats, error) {
	var s JournalStats
	err := m.db.QueryRow(`SELECT COUNT(*),COUNT(DISTINCT day),COALESCE(SUM(duration_minutes),0),COALESCE(AVG(NULLIF(mood,0)),0),COALESCE(AVG(NULLIF(focus,0)),0) FROM discipline_journal_entries`).Scan(&s.TotalEntries, &s.ActiveDays, &s.DurationMinutes, &s.AverageMood, &s.AverageFocus)
	if err != nil {
		return s, err
	}
	rows, err := m.db.Query(`SELECT day,COUNT(*),COALESCE(SUM(duration_minutes),0) FROM discipline_journal_entries WHERE day>=date('now','-27 days','localtime') GROUP BY day ORDER BY day`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	byDay := map[string]JournalDay{}
	allDays := []string{}
	for rows.Next() {
		var d JournalDay
		if err := rows.Scan(&d.Day, &d.Entries, &d.DurationMinutes); err != nil {
			return s, err
		}
		byDay[d.Day] = d
	}
	all, err := m.db.Query(`SELECT DISTINCT day FROM discipline_journal_entries ORDER BY day`)
	if err != nil {
		return s, err
	}
	defer all.Close()
	for all.Next() {
		var d string
		if err := all.Scan(&d); err != nil {
			return s, err
		}
		allDays = append(allDays, d)
	}
	s.CurrentStreak, s.LongestStreak = streakFromJournalDays(allDays)
	start := time.Now().AddDate(0, 0, -27)
	for i := 0; i < 28; i++ {
		key := start.AddDate(0, 0, i).Format("2006-01-02")
		d := byDay[key]
		d.Day = key
		s.Last28Days = append(s.Last28Days, d)
	}
	return s, rows.Err()
}
