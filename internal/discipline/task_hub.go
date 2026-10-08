package discipline

import (
	"fmt"
	"strings"
)

type TaskTemplate struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Notes       string `json:"notes"`
	Priority    string `json:"priority"`
	Labels      string `json:"labels"`
	EstimateMin int    `json:"estimate_minutes"`
	Energy      string `json:"energy"`
	Checklist   string `json:"checklist"`
	CreatedAt   string `json:"created_at"`
}

func (m *Manager) ListTaskTemplates() ([]TaskTemplate, error) {
	rows, err := m.db.Query(`SELECT id,name,title,notes,priority,labels,estimate_minutes,energy,checklist,created_at FROM discipline_task_templates ORDER BY name,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TaskTemplate{}
	for rows.Next() {
		var t TaskTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Title, &t.Notes, &t.Priority, &t.Labels, &t.EstimateMin, &t.Energy, &t.Checklist, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (m *Manager) CreateTaskTemplate(t TaskTemplate) (int64, error) {
	if strings.TrimSpace(t.Name) == "" || strings.TrimSpace(t.Title) == "" {
		return 0, fmt.Errorf("template name and task title are required")
	}
	if t.Priority == "" {
		t.Priority = "medium"
	}
	if t.Energy == "" {
		t.Energy = "medium"
	}
	r, err := m.db.Exec(`INSERT INTO discipline_task_templates(name,title,notes,priority,labels,estimate_minutes,energy,checklist) VALUES(?,?,?,?,?,?,?,?)`, strings.TrimSpace(t.Name), strings.TrimSpace(t.Title), t.Notes, t.Priority, t.Labels, t.EstimateMin, t.Energy, t.Checklist)
	if err != nil {
		return 0, err
	}
	id, _ := r.LastInsertId()
	return id, nil
}

func (m *Manager) DeleteTaskTemplate(id int64) error {
	_, err := m.db.Exec(`DELETE FROM discipline_task_templates WHERE id=?`, id)
	return err
}

func (m *Manager) ApplyTaskTemplate(id, projectID int64, dueDate string, myDay bool) (int64, error) {
	var t TaskTemplate
	err := m.db.QueryRow(`SELECT id,name,title,notes,priority,labels,estimate_minutes,energy,checklist,created_at FROM discipline_task_templates WHERE id=?`, id).Scan(&t.ID, &t.Name, &t.Title, &t.Notes, &t.Priority, &t.Labels, &t.EstimateMin, &t.Energy, &t.Checklist, &t.CreatedAt)
	if err != nil {
		return 0, err
	}
	taskID, err := m.CreateTask(TaskInput{ProjectID: projectID, Title: t.Title, Notes: t.Notes, Priority: t.Priority, Labels: t.Labels, EstimateMin: t.EstimateMin, Energy: t.Energy, DueDate: dueDate, MyDay: myDay})
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(t.Checklist, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			_, _ = m.AddChecklist(taskID, line)
		}
	}
	return taskID, nil
}
