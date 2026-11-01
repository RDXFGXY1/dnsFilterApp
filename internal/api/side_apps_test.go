package api

import (
	"bytes"
	"database/sql"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/RDXFGXY1/dns-filter-app/internal/sideapps"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func sideAppTestServer(t *testing.T) (*Server, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, e := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	manager, e := sideapps.New(db)
	if e != nil {
		t.Fatal(e)
	}
	s := testSecurityServer()
	s.sideApps = manager
	s.cfg.Security.SetupComplete = true
	r := gin.New()
	r.Use(s.securityMiddleware(), requestSizeMiddleware())
	s.registerSideAppRoutes(r.Group("/api", s.authMiddleware(), s.vaultMiddleware()))
	r.GET("/side-apps/run/:id", s.authMiddleware(), s.runSideApp)
	return s, r
}
func sideAppTestRequest(t *testing.T, r *gin.Engine, method, path, contentType string, body io.Reader, authenticated, csrf bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://localhost"+path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if authenticated {
		req.AddCookie(&http.Cookie{Name: "session", Value: "session-one"})
	}
	if csrf {
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf-one"})
		req.Header.Set("X-CSRF-Token", "csrf-one")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func sideAppUpload(t *testing.T, r *gin.Engine, path, filename, digest string, raw []byte) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, e := w.CreateFormFile("package", filename)
	if e != nil {
		t.Fatal(e)
	}
	f.Write(raw)
	w.WriteField("sha256", digest)
	w.Close()
	return sideAppTestRequest(t, r, "POST", path, w.FormDataContentType(), &b, true, true)
}
func TestSideAppHostAuthInstallRunStorage(t *testing.T) {
	_, r := sideAppTestServer(t)
	for _, path := range []string{"/api/side-apps", "/side-apps/run/focus-os", "/api/side-apps/focus-os/storage"} {
		w := sideAppTestRequest(t, r, "GET", path, "", nil, false, false)
		if w.Code != 401 {
			t.Fatalf("unauthenticated %s: %d", path, w.Code)
		}
	}
	w := sideAppTestRequest(t, r, "POST", "/api/side-apps/focus-os/enabled", "application/json", strings.NewReader(`{"enabled":false}`), true, false)
	if w.Code != 403 {
		t.Fatalf("missing CSRF: %d", w.Code)
	}
	// The checked-in example is packaged by the command's own tests; build a zip here from it.
	raw := sideAppExampleArchive(t)
	p, e := sideapps.Parse(raw)
	if e != nil {
		t.Fatal(e)
	}
	w = sideAppUpload(t, r, "/api/side-apps/preview", "focus-os.json", "", raw)
	if w.Code != 400 {
		t.Fatalf("wrong extension: %d", w.Code)
	}
	w = sideAppUpload(t, r, "/api/side-apps/preview", "focus-os.adnf", "", raw)
	if w.Code != 200 || !strings.Contains(w.Body.String(), p.SHA256) {
		t.Fatalf("preview: %d %s", w.Code, w.Body.String())
	}
	w = sideAppUpload(t, r, "/api/side-apps/install", "focus-os.adnf", "wrong", raw)
	if w.Code != 400 {
		t.Fatalf("digest: %d", w.Code)
	}
	w = sideAppUpload(t, r, "/api/side-apps/install", "focus-os.adnf", p.SHA256, raw)
	if w.Code != 201 {
		t.Fatalf("install: %d %s", w.Code, w.Body.String())
	}
	w = sideAppTestRequest(t, r, "GET", "/side-apps/run/focus-os", "", nil, true, false)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "ADNSF") {
		t.Fatalf("run: %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Security-Policy") != sideapps.AppCSP || w.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Fatal("missing sandbox response policies")
	}
	w = sideAppTestRequest(t, r, "PUT", "/api/side-apps/focus-os/storage", "application/json", strings.NewReader(`{"notes":"only here"}`), true, true)
	if w.Code != 200 {
		t.Fatalf("storage write: %d %s", w.Code, w.Body.String())
	}
	w = sideAppTestRequest(t, r, "GET", "/api/side-apps/focus-os/storage", "", nil, true, false)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "only here") {
		t.Fatalf("read: %s", w.Body.String())
	}
	w = sideAppTestRequest(t, r, "POST", "/api/side-apps/focus-os/enabled", "application/json", strings.NewReader(`{"enabled":false}`), true, true)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = sideAppTestRequest(t, r, "GET", "/side-apps/run/focus-os", "", nil, true, false)
	if w.Code != 403 {
		t.Fatalf("disabled app ran: %d", w.Code)
	}
	if !vaultProtectsPath("/api/side-apps/install") || !vaultProtectsPath("/api/side-apps/storage") || vaultProtectsPath("/api/side-apps/focus-os/storage") {
		t.Fatal("incorrect install/storage vault boundary")
	}
}
func TestSideAppsDashboardContainsInstallAndSandboxHost(t *testing.T) {
	tmpl, e := template.ParseGlob("../../web/templates/*")
	if e != nil {
		t.Fatal(e)
	}
	var b bytes.Buffer
	if e = tmpl.ExecuteTemplate(&b, "dashboard.html", gin.H{"title": "DNS Filter"}); e != nil {
		t.Fatal(e)
	}
	for _, fragment := range []string{`id="sideapps-nav"`, `id="sideapps-nav-add"`, `id="sideapps-package-file"`, `accept=".adnf"`, `data-view="sideapp"`} {
		if !strings.Contains(b.String(), fragment) {
			t.Errorf("missing %s", fragment)
		}
	}
	source, e := os.ReadFile("../../web/static/js/side-apps.js")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(source), "allow-same-origin") || !strings.Contains(string(source), "setAttribute('sandbox','allow-scripts')") {
		t.Fatal("unsafe frame permissions")
	}
}
