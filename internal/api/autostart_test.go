package api

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStartupRejectsInvalidRequestBeforeSystemChanges(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{}
	r := gin.New()
	r.POST("/autostart", s.setAutostart)
	for _, body := range []string{`{}`, `{"mode":"other"}`, `{"enabled":"yes"}`, `{"mode":"boot","recover_dns":"yes"}`} {
		w := httptest.NewRecorder()
		request := httptest.NewRequest("POST", "/autostart", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, request)
		if w.Code != 400 {
			t.Fatalf("invalid request %s: %d %s", body, w.Code, w.Body.String())
		}
	}
}
