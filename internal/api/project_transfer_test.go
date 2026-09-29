package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
	"github.com/gin-gonic/gin"
)

func TestDecodeProjectsYAMLHierarchy(t *testing.T) {
	doc, err := decodeProjectsYAML(`
version: 1
projects:
  - name: Launch
    objective: Release the MVP
    milestones:
      - name: MVP ready
    cycles:
      - name: Week 1
    tasks:
      - title: Research users
        priority: high
        milestone: MVP ready
        cycle: Week 1
        subtasks:
          - title: Run interviews
      - title: Build product
        depends_on: [research-users]
        checklist:
          - text: Test accessibility
`)
	if err != nil {
		t.Fatalf("decodeProjectsYAML returned error: %v", err)
	}
	if len(doc.Projects) != 1 || len(doc.Projects[0].Tasks) != 2 || len(doc.Projects[0].Tasks[0].Subtasks) != 1 {
		t.Fatalf("unexpected hierarchy: %#v", doc)
	}
	if doc.Projects[0].Tasks[1].Priority != "medium" {
		t.Fatalf("default priority not applied: %#v", doc.Projects[0].Tasks[1])
	}
	if doc.Projects[0].Tasks[0].Key != "research-users" || doc.Projects[0].Tasks[0].Subtasks[0].Key != "run-interviews" {
		t.Fatalf("portable task keys were not generated: %#v", doc.Projects[0].Tasks[0])
	}
}

func TestProjectYAMLImportAndExportRoundTrip(t *testing.T) {
	db, err := database.New(t.TempDir() + "/transfer.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mgr, err := discipline.New(db.GetDB())
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{db: db, discipline: mgr}
	source := `version: 1
projects:
  - name: Portable plan
    objective: Verify the complete pipeline
    milestones:
      - name: Ready
    tasks:
      - title: First task
        milestone: Ready
      - title: Second task
        depends_on: [first-task]
        checklist:
          - text: Verify result
`
	body, _ := json.Marshal(gin.H{"yaml": source, "mode": "add"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	s.importProjectsYAML(c)
	if w.Code != http.StatusOK {
		t.Fatalf("import failed: %d %s", w.Code, w.Body.String())
	}
	projects, err := mgr.ListProjects()
	if err != nil || len(projects) != 1 {
		t.Fatalf("projects after import: %#v, %v", projects, err)
	}
	tasks, err := mgr.ListTasks(projects[0].ID, false)
	if err != nil || len(tasks) != 2 {
		t.Fatalf("tasks after import: %#v, %v", tasks, err)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?project="+fmt.Sprint(projects[0].ID), nil)
	s.exportProjectsYAML(c)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "depends_on:") {
		t.Fatalf("export failed: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "task-1") || !strings.Contains(w.Body.String(), "key: first-task") {
		t.Fatalf("export did not use portable generated task keys: %s", w.Body.String())
	}
}

func TestDecodeProjectsYAMLRejectsBrokenReferences(t *testing.T) {
	_, err := decodeProjectsYAML(`version: 1
projects:
  - name: Broken
    tasks:
      - key: build
        title: Build
        depends_on: [missing]
`)
	if err == nil || !strings.Contains(err.Error(), "unknown task key") {
		t.Fatalf("expected dependency validation error, got %v", err)
	}
}
