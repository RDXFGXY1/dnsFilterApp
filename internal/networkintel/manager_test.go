package networkintel

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

type learningPlatform struct{ items []Connection }

func (p learningPlatform) Connections() ([]Connection, error) { return p.items, nil }
func (p learningPlatform) Supported() bool                    { return true }

func TestLearningBaselineIsScopedToSession(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mgr, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	path := `C:\Apps\Spotify\Spotify.exe`
	mgr.platform = learningPlatform{items: []Connection{{Path: path, RemoteIP: "1.2.3.4", RemotePort: 443, Protocol: "TCP"}}}
	id, err := mgr.StartLearning(path, 15)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.Refresh(); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Refresh(); err != nil {
		t.Fatal(err)
	}
	session, baseline, err := mgr.LearningBaseline(id)
	if err != nil {
		t.Fatal(err)
	}
	if session.ID != id || len(baseline) != 1 {
		t.Fatalf("unexpected baseline: session=%d connections=%d", session.ID, len(baseline))
	}
	if baseline[0].RemoteIP != "1.2.3.4" || baseline[0].Observations != 2 {
		t.Fatalf("unexpected observation: %+v", baseline[0])
	}
	connections, err := mgr.AppConnections(path, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(connections) != 1 || connections[0].RemotePort != 443 || connections[0].Seen != 2 {
		t.Fatalf("unexpected application details: %+v", connections)
	}
	trusted, err := mgr.TrustCurrentDestinations(path)
	if err != nil || trusted != 1 {
		t.Fatalf("trust baseline failed: count=%d err=%v", trusted, err)
	}
	connections, err = mgr.AppConnections(path, 20)
	if err != nil || len(connections) != 1 || !connections[0].Trusted {
		t.Fatalf("trusted destination was not marked: %+v err=%v", connections, err)
	}
	mgr.platform = learningPlatform{items: []Connection{
		{Path: path, RemoteIP: "1.2.3.4", RemotePort: 443, Protocol: "TCP"},
		{Path: path, RemoteIP: "5.6.7.8", RemotePort: 8443, Protocol: "TCP"},
	}}
	if err := mgr.Refresh(); err != nil {
		t.Fatal(err)
	}
	alerts, err := mgr.NewDestinations(20)
	if err != nil || len(alerts) != 1 || alerts[0].RemoteIP != "5.6.7.8" || alerts[0].Application != "Spotify.exe" {
		t.Fatalf("unexpected new destination alerts: %+v err=%v", alerts, err)
	}
	if err := mgr.TrustDestination(path, "5.6.7.8", 8443, "tcp"); err != nil {
		t.Fatalf("trust destination failed: %v", err)
	}
	alerts, err = mgr.NewDestinations(20)
	if err != nil || len(alerts) != 0 {
		t.Fatalf("trusted destination remained in alerts: %+v err=%v", alerts, err)
	}
	if err := mgr.TrustDestination(path, "not-an-ip", 443, "TCP"); err == nil {
		t.Fatal("invalid destination IP was accepted")
	}
	if err := mgr.SetRetentionDays(30); err != nil {
		t.Fatalf("set retention failed: %v", err)
	}
	if err := mgr.SetNotificationSettings(NotificationSettings{Enabled: true, MinRisk: 35, Applications: []string{"Discord.exe", " discord.exe ", "Code.exe"}}); err != nil {
		t.Fatalf("set notification settings failed: %v", err)
	}
	notifications, err := mgr.NotificationSettings()
	if err != nil || !notifications.Enabled || notifications.MinRisk != 35 || len(notifications.Applications) != 2 {
		t.Fatalf("unexpected notification settings: %+v err=%v", notifications, err)
	}
	if err := mgr.SetNotificationSettings(NotificationSettings{Enabled: true, MinRisk: 101}); err == nil {
		t.Fatal("invalid notification risk was accepted")
	}
	if err := mgr.SetAutomaticResponseSettings(AutomaticResponseSettings{Enabled: true, MinRisk: 35, Action: "review"}); err != nil {
		t.Fatalf("set automatic response failed: %v", err)
	}
	automation, err := mgr.AutomaticResponseSettings()
	if err != nil || !automation.Enabled || automation.MinRisk != 35 || automation.Action != "review" {
		t.Fatalf("unexpected automatic response settings: %+v err=%v", automation, err)
	}
	mgr.platform = learningPlatform{items: []Connection{
		{Path: path, RemoteIP: "1.2.3.4", RemotePort: 443, Protocol: "TCP"},
		{Path: path, RemoteIP: "5.6.7.8", RemotePort: 8443, Protocol: "TCP"},
		{Path: path, RemoteIP: "9.9.9.9", RemotePort: 9443, Protocol: "TCP"},
	}}
	if err := mgr.Refresh(); err != nil {
		t.Fatal(err)
	}
	candidates, err := mgr.AutomaticCandidates(35, 20)
	if err != nil || len(candidates) != 1 || candidates[0].RemoteIP != "9.9.9.9" {
		t.Fatalf("unexpected automatic candidates: %+v err=%v", candidates, err)
	}
	if err := mgr.RecordAutomaticResponse(candidates[0], "review"); err != nil {
		t.Fatalf("record automatic response failed: %v", err)
	}
	candidates, err = mgr.AutomaticCandidates(35, 20)
	if err != nil || len(candidates) != 0 {
		t.Fatalf("automatic response was offered twice: %+v err=%v", candidates, err)
	}
	analytics, err := mgr.SecurityAnalytics()
	if err != nil || analytics.Applications != 1 || analytics.Destinations != 3 || len(analytics.TopApps) != 1 {
		t.Fatalf("unexpected security analytics: %+v err=%v", analytics, err)
	}
	status, err := mgr.HistoryStatus()
	if err != nil || status.RetentionDays != 30 || status.Connections != 3 {
		t.Fatalf("unexpected history status: %+v err=%v", status, err)
	}
	report, err := mgr.ConnectionReport()
	if err != nil || len(report) != 3 {
		t.Fatalf("unexpected connection report: %+v err=%v", report, err)
	}
	if err := mgr.ClearHistory(); err != nil {
		t.Fatalf("clear history failed: %v", err)
	}
	status, err = mgr.HistoryStatus()
	if err != nil || status.Connections != 0 || status.Applications != 0 {
		t.Fatalf("history was not cleared: %+v err=%v", status, err)
	}
	var baselines int
	if err := db.QueryRow(`SELECT COUNT(*) FROM network_destination_baseline`).Scan(&baselines); err != nil || baselines != 2 {
		t.Fatalf("trusted baselines were removed: count=%d err=%v", baselines, err)
	}
}
