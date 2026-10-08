package plugins

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "plugins.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func validManifest(baseURL string) Manifest {
	return Manifest{
		SchemaVersion: 1,
		ID:            "org.nullstudio.test",
		Name:          "Test Plugin",
		Version:       "1.0.0",
		Publisher:     "NullStudio",
		Server:        Server{BaseURL: baseURL, HealthEndpoint: "/health", StartMode: "external"},
		Permissions:   []string{"test.run"},
		Panels:        []Panel{{ID: "main", Title: "Main", Actions: []string{"run"}}},
		Actions:       []Action{{ID: "run", Label: "Run", Method: "POST", Path: "/run", Permission: "test.run", Fields: []Field{{ID: "settings.level", Label: "Level", Type: "number", Required: true}}}},
	}
}

func TestValidateManifestRejectsRemoteServer(t *testing.T) {
	manifest := validManifest("http://example.com:8765")
	if err := ValidateManifest(manifest); err == nil || !strings.Contains(err.Error(), "localhost") {
		t.Fatalf("expected loopback validation error, got %v", err)
	}
}

func TestManagedModeRequiresRuntimeCommand(t *testing.T) {
	manifest := validManifest("http://127.0.0.1:8765")
	manifest.Server.StartMode = "on_demand"
	if err := ValidateManifest(manifest); err == nil || !strings.Contains(err.Error(), "runtime command") {
		t.Fatalf("expected managed runtime validation error, got %v", err)
	}
	manifest.Server.Runtime = &Runtime{Windows: &Command{Executable: `C:\tools\plugin.exe`}}
	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("valid managed runtime rejected: %v", err)
	}
}

func TestRuntimeModePersists(t *testing.T) {
	manager, err := New(testDB(t))
	if err != nil {
		t.Fatal(err)
	}
	manifest := validManifest("http://127.0.0.1:8765")
	manifest.Server.Runtime = &Runtime{Windows: &Command{Executable: `C:\tools\plugin.exe`}}
	if err := manager.Install(manifest); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetStartMode(manifest.ID, "manual"); err != nil {
		t.Fatal(err)
	}
	item, err := manager.Get(manifest.ID)
	if err != nil || item.Manifest.Server.StartMode != "manual" {
		t.Fatalf("runtime mode was not persisted: %#v %v", item, err)
	}
}

func TestInstallEnableInvokeAndAudit(t *testing.T) {
	manager, err := New(testDB(t))
	if err != nil {
		t.Fatal(err)
	}
	manager.client.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		var response any
		switch r.URL.Path {
		case "/health":
			response = map[string]any{"ready": true}
		case "/run":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			settings, _ := body["settings"].(map[string]any)
			response = map[string]any{"level": settings["level"]}
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":"not found"}`)), Header: make(http.Header)}, nil
		}
		encoded, _ := json.Marshal(response)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(encoded))), Header: http.Header{"Content-Type": []string{"application/json"}}}, nil
	})
	manifest := validManifest("http://127.0.0.1:8765")
	if err := manager.Install(manifest); err != nil {
		t.Fatal(err)
	}
	items, err := manager.List()
	if err != nil || len(items) != 1 || items[0].Enabled {
		t.Fatalf("unexpected install state: %#v %v", items, err)
	}
	if _, _, err := manager.Invoke(context.Background(), manifest.ID, "run", map[string]any{"settings.level": 4}); err == nil {
		t.Fatal("disabled plugin action was allowed")
	}
	if err := manager.SetEnabled(manifest.ID, true); err != nil {
		t.Fatal(err)
	}
	data, status, err := manager.Invoke(context.Background(), manifest.ID, "run", map[string]any{"settings.level": 4})
	if err != nil || status != http.StatusOK {
		t.Fatalf("invoke failed: status=%d err=%v", status, err)
	}
	result, _ := data.(map[string]any)
	if result["level"] != float64(4) {
		t.Fatalf("nested payload was not sent: %#v", result)
	}
	events, err := manager.Events(manifest.ID, 20)
	if err != nil || len(events) < 3 {
		t.Fatalf("expected audit events, got %#v %v", events, err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestParseManifestRejectsUnknownFields(t *testing.T) {
	_, err := ParseManifest(strings.NewReader(`{"schema_version":1,"id":"org.test","name":"Test","version":"1","server":{"base_url":"http://127.0.0.1:1","health_endpoint":"/health","start_mode":"external"},"permissions":[],"panels":[],"actions":[],"surprise":true}`))
	if err == nil {
		t.Fatal("unknown manifest field was accepted")
	}
}

func TestManifestUIValidation(t *testing.T) {
	manifest := validManifest("http://127.0.0.1:8765")
	manifest.UI = &UI{Mode: "both", DefaultView: "web", WebPath: "/"}
	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("valid dual UI was rejected: %v", err)
	}
	manifest.UI.WebPath = "https://example.com/escape"
	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("external plugin web path was accepted")
	}
	manifest.UI = &UI{Mode: "native", DefaultView: "web"}
	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("unavailable default plugin view was accepted")
	}
}

func TestActionTimeoutIsBounded(t *testing.T) {
	manager, err := New(testDB(t))
	if err != nil {
		t.Fatal(err)
	}
	manager.client.Transport = roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	manifest := validManifest("http://127.0.0.1:8765")
	manifest.Actions[0].TimeoutSeconds = 1
	if err := manager.Install(manifest); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetEnabled(manifest.ID, true); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_, _, err = manager.Invoke(context.Background(), manifest.ID, "run", map[string]any{"settings.level": 4})
	if err == nil || time.Since(started) > 3*time.Second {
		t.Fatalf("action timeout was not enforced: elapsed=%s err=%v", time.Since(started), err)
	}
}

func TestLegacyHardwareActionsReceiveSafeTimeouts(t *testing.T) {
	if got := effectiveActionTimeout(Action{Path: "/api/scan"}); got != 60*time.Second {
		t.Fatalf("legacy scan timeout = %s", got)
	}
	if got := effectiveActionTimeout(Action{Path: "/api/print/calibration"}); got != 120*time.Second {
		t.Fatalf("legacy calibration timeout = %s", got)
	}
}

func TestTinyPrinterManifestIsValid(t *testing.T) {
	file, err := os.Open(filepath.Join("..", "..", "integrations", "tinyprinter", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	manifest, err := ParseManifest(file)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ID != "org.nullstudio.tinyprinter" || len(manifest.Actions) < 5 || manifest.UI == nil || manifest.UI.Mode != "both" {
		t.Fatalf("unexpected TinyPrinter manifest: %#v", manifest)
	}
}
