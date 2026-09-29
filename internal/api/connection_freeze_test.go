package api

import (
	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/connectionfreeze"
	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/internal/vault"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFreezeAuthenticationAndLockedRecovery(t *testing.T) {
	db, err := database.New(t.TempDir() + "/freeze.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	manager, err := connectionfreeze.New(db.GetDB())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	v, err := vault.New(t.TempDir(), time.Minute, 5, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = v.Setup("test-freeze-password-123"); err != nil {
		t.Fatal(err)
	}
	v.Lock()
	s := &Server{cfg: &config.Config{Security: config.SecurityConfig{SetupComplete: true}}, freeze: manager, vault: v, sessions: map[string]time.Time{"test-session": time.Now().Add(time.Minute)}}
	router := gin.New()
	group := router.Group("/api")
	group.Use(s.authMiddleware(), s.vaultMiddleware())
	s.registerConnectionFreezeRoutes(group)
	for _, test := range []struct {
		method, path string
		auth         bool
		want         int
	}{
		{"GET", "/api/connection-freeze", false, 401},
		{"POST", "/api/connection-freeze/start", false, 401},
		{"POST", "/api/connection-freeze/stop", false, 401},
		{"GET", "/api/connection-freeze", true, 200},
		{"POST", "/api/connection-freeze/start", true, 423},
		{"POST", "/api/connection-freeze/stop", true, 200},
	} {
		r := httptest.NewRequest(test.method, test.path, strings.NewReader(`{"minutes":15}`))
		r.Header.Set("Content-Type", "application/json")
		if test.auth {
			r.AddCookie(&http.Cookie{Name: "session", Value: "test-session"})
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != test.want {
			t.Fatalf("%s %s auth=%v: %d: %s", test.method, test.path, test.auth, w.Code, w.Body.String())
		}
	}
}
