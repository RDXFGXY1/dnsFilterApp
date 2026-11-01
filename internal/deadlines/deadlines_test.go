package deadlines

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"testing"
	"time"
)

func deadlineDB(t *testing.T) *sql.DB {
	t.Helper()
	db, e := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	for _, q := range []string{
		`CREATE TABLE discipline_tasks(id INTEGER,title TEXT,due_date TEXT,reminder_at TEXT,done INTEGER DEFAULT 0,archived INTEGER DEFAULT 0)`,
		`CREATE TABLE discipline_reminders(id INTEGER,text TEXT,remind_at TEXT,done INTEGER DEFAULT 0)`,
		`CREATE TABLE discipline_projects(id INTEGER,name TEXT,target_date TEXT,status TEXT)`,
		`CREATE TABLE discipline_goals(id INTEGER,name TEXT,target_date TEXT,status TEXT)`,
		`CREATE TABLE discipline_milestones(id INTEGER,name TEXT,target_date TEXT,status TEXT,project_id INTEGER)`,
	} {
		if _, e = db.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	return db
}
func exec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, e := db.Exec(q); e != nil {
		t.Fatal(e)
	}
}
func TestDeadlineStagesAndDeduplication(t *testing.T) {
	db := deadlineDB(t)
	now := time.Date(2026, 10, 27, 8, 50, 0, 0, time.UTC)
	exec(t, db, `INSERT INTO discipline_tasks VALUES(1,'Read notes','2026-10-27','2026-10-27 09:00',0,0)`)
	items, e := Collect(db, now)
	if e != nil || len(items) != 1 {
		t.Fatalf("expected one combined reminder/deadline, got %+v %v", items, e)
	}
	key := items[0].Key
	items, e = Collect(db, now.Add(time.Minute))
	if e != nil || len(items) != 1 || items[0].Key != key {
		t.Fatal("countdown changed delivery identity")
	}
	items, e = Collect(db, now.Add(10*time.Minute))
	if e != nil || len(items) != 1 || items[0].Key == key || items[0].Title != "Task reminder is due" {
		t.Fatalf("deadline stage missing: %+v %v", items, e)
	}
	exec(t, db, `UPDATE discipline_tasks SET due_date='2026-10-28',reminder_at='2026-10-28 09:00'`)
	items, e = Collect(db, now)
	if e != nil || len(items) != 0 {
		t.Fatal("future reschedule still alerted")
	}
}
func TestCompletedArchivedAndOldDeadlinesAreExcluded(t *testing.T) {
	db := deadlineDB(t)
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	exec(t, db, `INSERT INTO discipline_tasks VALUES(1,'Done','2026-10-27','',1,0),(2,'Archived','2026-10-27','',0,1),(3,'Old','2026-10-20','',0,0),(4,'Invalid','not a date','',0,0)`)
	exec(t, db, `INSERT INTO discipline_projects VALUES(1,'Done project','2026-10-27','done'),(2,'Paused project','2026-10-27','paused')`)
	exec(t, db, `INSERT INTO discipline_milestones VALUES(1,'Hidden milestone','2026-10-27','planned',1)`)
	items, e := Collect(db, now)
	if e != nil || len(items) != 0 {
		t.Fatalf("inactive/old work alerted: %+v %v", items, e)
	}
}
func TestReminderGoalProjectAndMilestone(t *testing.T) {
	db := deadlineDB(t)
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	exec(t, db, `INSERT INTO discipline_reminders VALUES(1,'Check exercise','2026-10-27 09:00',0)`)
	exec(t, db, `INSERT INTO discipline_projects VALUES(1,'Exam prep','2026-10-27','active')`)
	exec(t, db, `INSERT INTO discipline_goals VALUES(1,'Study goal','2026-10-27','active')`)
	exec(t, db, `INSERT INTO discipline_milestones VALUES(1,'Finish chapter','2026-10-27','planned',1)`)
	items, e := Collect(db, now)
	if e != nil || len(items) != 4 {
		t.Fatalf("missing deadline source: %+v %v", items, e)
	}
}
func TestDeadlineDateFormatsAndLocalTime(t *testing.T) {
	location := time.FixedZone("Local", 3600)
	for _, raw := range []string{"2026-10-27 09:00", "2026-10-27T09:00", "2026-10-27T08:00:00Z", "2026-10-27"} {
		due, ok := parseDate(raw, location)
		if !ok || !due.Equal(time.Date(2026, 10, 27, 9, 0, 0, 0, location)) {
			t.Errorf("wrong date: %s -> %s", raw, due)
		}
	}
}
