package discipline

import "testing"

func TestTaskHubPlanningFieldsAndTemplates(t *testing.T) {
	m := newJournalTestManager(t)
	id, err := m.CreateTask(TaskInput{Title: "Prepare release", Priority: "high", MyDay: true, ScheduledAt: "2026-10-08T09:00", WaitingFor: "design approval", Evidence: "release URL", EstimateMin: 90})
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := m.ListTasks(0, true)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("unexpected tasks: %#v %v", tasks, err)
	}
	if !tasks[0].MyDay || tasks[0].ScheduledAt == "" || tasks[0].WaitingFor != "design approval" || tasks[0].Evidence != "release URL" {
		t.Fatalf("planning fields were not persisted: %#v", tasks[0])
	}
	name := "approval received"
	if _, _, err = m.UpdateTask(id, TaskPatch{WaitingFor: &name}); err != nil {
		t.Fatal(err)
	}
	tpl, err := m.CreateTaskTemplate(TaskTemplate{Name: "Release checklist", Title: "Ship a release", Priority: "high", EstimateMin: 60, Checklist: "Build\nTest\nPublish"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := m.ApplyTaskTemplate(tpl, 0, "2026-10-09", true)
	if err != nil {
		t.Fatal(err)
	}
	check, err := m.Checklist(created)
	if err != nil || len(check) != 3 {
		t.Fatalf("template checklist missing: %#v %v", check, err)
	}
}
