package api

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPluginRoutesRegisterWithoutConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	server := &Server{}
	server.registerPluginRoutes(router.Group("/api"))
	if len(router.Routes()) != 12 {
		t.Fatalf("expected 12 plugin routes, got %d", len(router.Routes()))
	}
}
