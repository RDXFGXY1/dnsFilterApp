package api

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type dashboardTodo struct {
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parent_id"`
	Title    string `json:"title"`
	Done     bool   `json:"done"`
	Priority string `json:"priority"`
	Created  string `json:"created_at"`
}

type todoYAMLItem struct {
	Title    string         `yaml:"title" json:"title"`
	Done     bool           `yaml:"done,omitempty" json:"done"`
	Priority string         `yaml:"priority,omitempty" json:"priority"`
	Subtodos []todoYAMLItem `yaml:"subtodos,omitempty" json:"subtodos"`
}

type todoYAMLDocument struct {
	Version int            `yaml:"version" json:"version"`
	List    string         `yaml:"list" json:"list"`
	Todos   []todoYAMLItem `yaml:"todos" json:"todos"`
}

type dashboardNote struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Pinned  bool   `json:"pinned"`
	Updated string `json:"updated_at"`
}

func (s *Server) registerOrganizerRoutes(api *gin.RouterGroup) {
	_, err := s.db.GetDB().Exec(`
		CREATE TABLE IF NOT EXISTS dashboard_todos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_id INTEGER,
			title TEXT NOT NULL,
			done BOOLEAN NOT NULL DEFAULT 0,
			priority TEXT NOT NULL DEFAULT 'medium',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_dashboard_todos_parent ON dashboard_todos(parent_id);
		CREATE TABLE IF NOT EXISTS dashboard_notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL DEFAULT 'Untitled note',
			content TEXT NOT NULL DEFAULT '',
			pinned BOOLEAN NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`)
	if err != nil {
		s.log.Warnf("Dashboard organizer unavailable: %v", err)
	}
	_, _ = s.db.GetDB().Exec(`ALTER TABLE dashboard_todos ADD COLUMN priority TEXT NOT NULL DEFAULT 'medium'`)
	api.GET("/organizer/todos", s.listDashboardTodos)
	api.POST("/organizer/todos", s.createDashboardTodo)
	api.PATCH("/organizer/todos/:id", s.updateDashboardTodo)
	api.DELETE("/organizer/todos/:id", s.deleteDashboardTodo)
	api.POST("/organizer/todos/import-yaml", s.importDashboardTodosYAML)
	api.POST("/organizer/todos/validate-yaml", s.validateDashboardTodosYAML)
	api.GET("/organizer/todos/export-yaml", s.exportDashboardTodosYAML)
	api.GET("/organizer/notes", s.listDashboardNotes)
	api.POST("/organizer/notes", s.createDashboardNote)
	api.PATCH("/organizer/notes/:id", s.updateDashboardNote)
	api.DELETE("/organizer/notes/:id", s.deleteDashboardNote)
}

func (s *Server) listDashboardTodos(c *gin.Context) {
	rows, err := s.db.GetDB().Query(`SELECT id, parent_id, title, done, priority, COALESCE(CAST(created_at AS TEXT),'') FROM dashboard_todos ORDER BY parent_id IS NOT NULL, created_at, id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []dashboardTodo{}
	for rows.Next() {
		var item dashboardTodo
		if err := rows.Scan(&item.ID, &item.ParentID, &item.Title, &item.Done, &item.Priority, &item.Created); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) createDashboardTodo(c *gin.Context) {
	var body struct {
		Title    string `json:"title"`
		ParentID *int64 `json:"parent_id"`
		Priority string `json:"priority"`
	}
	if c.ShouldBindJSON(&body) != nil || strings.TrimSpace(body.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	body.Title = strings.TrimSpace(body.Title)
	body.Priority = normalizeTodoPriority(body.Priority)
	if len(body.Title) > 300 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is too long"})
		return
	}
	if body.ParentID != nil {
		var exists int
		if s.db.GetDB().QueryRow(`SELECT COUNT(*) FROM dashboard_todos WHERE id = ?`, *body.ParentID).Scan(&exists) != nil || exists == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parent todo not found"})
			return
		}
	}
	result, err := s.db.GetDB().Exec(`INSERT INTO dashboard_todos(parent_id,title,priority) VALUES(?,?,?)`, body.ParentID, body.Title, body.Priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id})
}

func organizerID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

func (s *Server) updateDashboardTodo(c *gin.Context) {
	id, ok := organizerID(c)
	if !ok {
		return
	}
	var body struct {
		Title    *string `json:"title"`
		Done     *bool   `json:"done"`
		Priority *string `json:"priority"`
	}
	if c.ShouldBindJSON(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if body.Title != nil {
		title := strings.TrimSpace(*body.Title)
		if title == "" || len(title) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid title"})
			return
		}
		if _, err := s.db.GetDB().Exec(`UPDATE dashboard_todos SET title=? WHERE id=?`, title, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if body.Done != nil {
		if _, err := s.db.GetDB().Exec(`UPDATE dashboard_todos SET done=? WHERE id=? OR parent_id=?`, *body.Done, id, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if *body.Done {
			_, _ = s.db.GetDB().Exec(`UPDATE dashboard_todos SET done=CASE WHEN NOT EXISTS(SELECT 1 FROM dashboard_todos c WHERE c.parent_id=dashboard_todos.id AND c.done=0) THEN 1 ELSE done END WHERE id=(SELECT parent_id FROM dashboard_todos WHERE id=?)`, id)
		}
	}
	if body.Priority != nil {
		priority := normalizeTodoPriority(*body.Priority)
		if _, err := s.db.GetDB().Exec(`UPDATE dashboard_todos SET priority=? WHERE id=?`, priority, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func normalizeTodoPriority(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "medium", "high", "urgent":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "medium"
	}
}

func validateTodoYAML(doc *todoYAMLDocument) error {
	if doc.Version != 1 {
		return fmt.Errorf("version must be 1")
	}
	if strings.TrimSpace(doc.List) == "" {
		return fmt.Errorf("list name is required")
	}
	if len(doc.Todos) == 0 {
		return fmt.Errorf("the file does not contain any todos")
	}
	total := 0
	for i := range doc.Todos {
		item := &doc.Todos[i]
		item.Title = strings.TrimSpace(item.Title)
		if item.Title == "" {
			return fmt.Errorf("todo %d is missing a title", i+1)
		}
		if len(item.Title) > 300 {
			return fmt.Errorf("todo %q has a title longer than 300 characters", item.Title)
		}
		if item.Priority != "" && normalizeTodoPriority(item.Priority) != strings.ToLower(strings.TrimSpace(item.Priority)) {
			return fmt.Errorf("todo %q has invalid priority %q; use low, medium, high, or urgent", item.Title, item.Priority)
		}
		item.Priority = normalizeTodoPriority(item.Priority)
		total++
		for j := range item.Subtodos {
			sub := &item.Subtodos[j]
			sub.Title = strings.TrimSpace(sub.Title)
			if sub.Title == "" {
				return fmt.Errorf("subtodo %d under %q is missing a title", j+1, item.Title)
			}
			if len(sub.Title) > 300 {
				return fmt.Errorf("subtodo %q has a title longer than 300 characters", sub.Title)
			}
			if sub.Priority != "" && normalizeTodoPriority(sub.Priority) != strings.ToLower(strings.TrimSpace(sub.Priority)) {
				return fmt.Errorf("subtodo %q has invalid priority %q", sub.Title, sub.Priority)
			}
			sub.Priority = normalizeTodoPriority(sub.Priority)
			total++
		}
	}
	if total > 500 {
		return fmt.Errorf("a single import can contain at most 500 todos and subtodos")
	}
	return nil
}

func decodeTodoYAML(content string) (todoYAMLDocument, error) {
	var doc todoYAMLDocument
	decoder := yaml.NewDecoder(bytes.NewBufferString(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&doc); err != nil {
		return doc, fmt.Errorf("YAML could not be read: %w", err)
	}
	if err := validateTodoYAML(&doc); err != nil {
		return doc, err
	}
	return doc, nil
}

func (s *Server) validateDashboardTodosYAML(c *gin.Context) {
	var body struct {
		YAML string `json:"yaml"`
	}
	if c.ShouldBindJSON(&body) != nil || strings.TrimSpace(body.YAML) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "YAML content is required"})
		return
	}
	doc, err := decodeTodoYAML(body.YAML)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	total := len(doc.Todos)
	for _, item := range doc.Todos {
		total += len(item.Subtodos)
	}
	c.JSON(http.StatusOK, gin.H{"valid": true, "document": doc, "total": total})
}

func (s *Server) importDashboardTodosYAML(c *gin.Context) {
	var body struct {
		YAML string `json:"yaml"`
		Mode string `json:"mode"`
	}
	if c.ShouldBindJSON(&body) != nil || strings.TrimSpace(body.YAML) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "YAML content is required"})
		return
	}
	if body.Mode != "add" && body.Mode != "replace" && body.Mode != "skip" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be add, replace, or skip"})
		return
	}
	doc, err := decodeTodoYAML(body.YAML)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tx, err := s.db.GetDB().Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	if body.Mode == "replace" {
		if _, err = tx.Exec(`DELETE FROM dashboard_todos`); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	imported, skipped := 0, 0
	for _, item := range doc.Todos {
		if body.Mode == "skip" {
			var exists int
			_ = tx.QueryRow(`SELECT COUNT(*) FROM dashboard_todos WHERE parent_id IS NULL AND lower(title)=lower(?)`, item.Title).Scan(&exists)
			if exists > 0 {
				skipped += 1 + len(item.Subtodos)
				continue
			}
		}
		result, execErr := tx.Exec(`INSERT INTO dashboard_todos(parent_id,title,done,priority) VALUES(NULL,?,?,?)`, item.Title, item.Done, item.Priority)
		if execErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": execErr.Error()})
			return
		}
		parentID, _ := result.LastInsertId()
		imported++
		for _, sub := range item.Subtodos {
			if _, execErr = tx.Exec(`INSERT INTO dashboard_todos(parent_id,title,done,priority) VALUES(?,?,?,?)`, parentID, sub.Title, sub.Done, sub.Priority); execErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": execErr.Error()})
				return
			}
			imported++
		}
	}
	_, _ = tx.Exec(`INSERT INTO settings(key,value,updated_at) VALUES('organizer.todo_list_name',?,CURRENT_TIMESTAMP) ON CONFLICT(key) DO UPDATE SET value=excluded.value,updated_at=CURRENT_TIMESTAMP`, strings.TrimSpace(doc.List))
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "list": doc.List, "imported": imported, "skipped": skipped})
}

func (s *Server) exportDashboardTodosYAML(c *gin.Context) {
	listName := "My Todo List"
	_ = s.db.GetDB().QueryRow(`SELECT value FROM settings WHERE key='organizer.todo_list_name'`).Scan(&listName)
	rows, err := s.db.GetDB().Query(`SELECT id,parent_id,title,done,priority FROM dashboard_todos ORDER BY parent_id IS NOT NULL,created_at,id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	type rowItem struct {
		id     int64
		parent *int64
		item   todoYAMLItem
	}
	all := []rowItem{}
	for rows.Next() {
		var r rowItem
		if err := rows.Scan(&r.id, &r.parent, &r.item.Title, &r.item.Done, &r.item.Priority); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		all = append(all, r)
	}
	doc := todoYAMLDocument{Version: 1, List: listName, Todos: []todoYAMLItem{}}
	for _, r := range all {
		if r.parent == nil {
			item := r.item
			for _, sub := range all {
				if sub.parent != nil && *sub.parent == r.id {
					item.Subtodos = append(item.Subtodos, sub.item)
				}
			}
			doc.Todos = append(doc.Todos, item)
		}
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", `attachment; filename="todo-list.yaml"`)
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", data)
}

func (s *Server) deleteDashboardTodo(c *gin.Context) {
	id, ok := organizerID(c)
	if !ok {
		return
	}
	_, err := s.db.GetDB().Exec(`WITH RECURSIVE descendants(id) AS (SELECT id FROM dashboard_todos WHERE id=? UNION ALL SELECT t.id FROM dashboard_todos t JOIN descendants d ON t.parent_id=d.id) DELETE FROM dashboard_todos WHERE id IN descendants`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) listDashboardNotes(c *gin.Context) {
	rows, err := s.db.GetDB().Query(`SELECT id,title,content,pinned,COALESCE(CAST(updated_at AS TEXT),'') FROM dashboard_notes ORDER BY pinned DESC, updated_at DESC, id DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []dashboardNote{}
	for rows.Next() {
		var item dashboardNote
		if err := rows.Scan(&item.ID, &item.Title, &item.Content, &item.Pinned, &item.Updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) createDashboardNote(c *gin.Context) {
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	_ = c.ShouldBindJSON(&body)
	title := strings.TrimSpace(body.Title)
	if title == "" {
		title = "Untitled note"
	}
	result, err := s.db.GetDB().Exec(`INSERT INTO dashboard_notes(title,content) VALUES(?,?)`, title, body.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id})
}

func (s *Server) updateDashboardNote(c *gin.Context) {
	id, ok := organizerID(c)
	if !ok {
		return
	}
	var body struct {
		Title   *string `json:"title"`
		Content *string `json:"content"`
		Pinned  *bool   `json:"pinned"`
	}
	if c.ShouldBindJSON(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if body.Title != nil {
		title := strings.TrimSpace(*body.Title)
		if title == "" {
			title = "Untitled note"
		}
		_, _ = s.db.GetDB().Exec(`UPDATE dashboard_notes SET title=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, title, id)
	}
	if body.Content != nil {
		_, _ = s.db.GetDB().Exec(`UPDATE dashboard_notes SET content=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, *body.Content, id)
	}
	if body.Pinned != nil {
		_, _ = s.db.GetDB().Exec(`UPDATE dashboard_notes SET pinned=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, *body.Pinned, id)
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) deleteDashboardNote(c *gin.Context) {
	id, ok := organizerID(c)
	if !ok {
		return
	}
	result, err := s.db.GetDB().Exec(`DELETE FROM dashboard_notes WHERE id=?`, id)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	changed, _ := result.RowsAffected()
	c.JSON(http.StatusOK, gin.H{"success": true, "deleted": changed})
}
