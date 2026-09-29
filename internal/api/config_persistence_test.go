package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/gin-gonic/gin"
)

func configWithBrokenSavePath(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := "filtering:\n  block_action: nxdomain\nblocklists:\n  auto_update_interval: 24\n  sources:\n    - name: Existing\n      url: https://example.test/list.txt\n      category: test\n      enabled: true\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestConfigHandlersRollbackWhenSaveFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("block action", func(t *testing.T) {
		cfg := configWithBrokenSavePath(t)
		s := &Server{cfg: cfg}
		router := gin.New()
		router.POST("/block-action", s.setBlockAction)
		response := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/block-action", strings.NewReader(`{"action":"redirect"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, req)
		if response.Code != http.StatusInternalServerError || cfg.Filtering.BlockAction != "nxdomain" {
			t.Fatalf("failed save was not rolled back: status=%d action=%q body=%s", response.Code, cfg.Filtering.BlockAction, response.Body.String())
		}
	})

	t.Run("add source", func(t *testing.T) {
		cfg := configWithBrokenSavePath(t)
		s := &Server{cfg: cfg}
		router := gin.New()
		router.POST("/sources", s.addSource)
		response := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/sources", strings.NewReader(`{"name":"New","url":"https://example.test/new.txt"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, req)
		if response.Code != http.StatusInternalServerError || len(cfg.Blocklists.Sources) != 1 {
			t.Fatalf("failed source save was not rolled back: status=%d sources=%d body=%s", response.Code, len(cfg.Blocklists.Sources), response.Body.String())
		}
	})

	t.Run("auto update", func(t *testing.T) {
		cfg := configWithBrokenSavePath(t)
		s := &Server{cfg: cfg}
		router := gin.New()
		router.POST("/auto-update", s.setAutoUpdate)
		response := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auto-update", strings.NewReader(`{"hours":12}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, req)
		if response.Code != http.StatusInternalServerError || cfg.Blocklists.AutoUpdateInterval != 24 {
			t.Fatalf("failed interval save was not rolled back: status=%d hours=%d body=%s", response.Code, cfg.Blocklists.AutoUpdateInterval, response.Body.String())
		}
	})
}
