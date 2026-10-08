package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/RDXFGXY1/dns-filter-app/internal/plugins"
	"github.com/gin-gonic/gin"
)

func (s *Server) registerPluginRoutes(api *gin.RouterGroup) {
	registry := api.Group("/plugin-registry")
	{
		registry.GET("", s.listPlugins)
		registry.POST("/install", s.installPlugin)
		registry.POST("/:id/enabled", s.setPluginEnabled)
		registry.DELETE("/:id", s.removePlugin)
	}

	runtime := api.Group("/plugins")
	{
		runtime.GET("/:id/health", s.pluginHealth)
		runtime.GET("/:id/runtime", s.pluginRuntimeStatus)
		runtime.POST("/:id/runtime/:operation", s.pluginRuntimeOperation)
		runtime.POST("/:id/runtime-mode", s.setPluginRuntimeMode)
		runtime.POST("/:id/actions/:action", s.invokePluginAction)
		runtime.POST("/:id/actions/:action/file", s.invokePluginFileAction)
		runtime.POST("/:id/actions/:action/content", s.invokePluginContentAction)
		runtime.GET("/events/recent", s.pluginEvents)
	}
}

func (s *Server) invokePluginFileAction(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "choose a file first"})
		return
	}
	defer file.Close()
	input := map[string]any{}
	if raw := c.PostForm("input"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file settings"})
			return
		}
	}
	data, status, err := s.plugins.InvokeFile(c.Request.Context(), c.Param("id"), c.Param("action"), header.Filename, file, header.Size, input)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "plugin_status": status, "data": data})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "plugin_status": status, "data": data})
}

func (s *Server) invokePluginContentAction(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	var input struct {
		Input map[string]any `json:"input"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action input"})
		return
	}
	data, contentType, status, err := s.plugins.InvokeContent(c.Request.Context(), c.Param("id"), c.Param("action"), input.Input)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "plugin_status": status})
		return
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Data(http.StatusOK, contentType, data)
}

func (s *Server) setPluginRuntimeMode(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	var input struct {
		Mode string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "runtime mode is required"})
		return
	}
	if err := s.plugins.SetStartMode(c.Param("id"), input.Mode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "mode": input.Mode})
}

func (s *Server) pluginRuntimeStatus(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	status, err := s.plugins.RuntimeStatus(c.Param("id"))
	if err != nil {
		pluginError(c, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

func (s *Server) pluginRuntimeOperation(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	var err error
	switch c.Param("operation") {
	case "start":
		err = s.plugins.Start(c.Request.Context(), c.Param("id"))
	case "stop":
		err = s.plugins.Stop(c.Param("id"))
	case "restart":
		err = s.plugins.Restart(c.Request.Context(), c.Param("id"))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown runtime operation"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	status, _ := s.plugins.RuntimeStatus(c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"success": true, "runtime": status})
}

func (s *Server) requirePlugins(c *gin.Context) bool {
	if s.plugins == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin Center is unavailable"})
		return false
	}
	return true
}

func (s *Server) listPlugins(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	items, err := s.plugins.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"plugins": items})
}

func (s *Server) installPlugin(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	manifest, err := plugins.ParseManifest(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.plugins.Install(manifest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "plugin_id": manifest.ID, "enabled": false})
}

func (s *Server) setPluginEnabled(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	var input struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enabled state is required"})
		return
	}
	if err := s.plugins.SetEnabled(c.Param("id"), input.Enabled); err != nil {
		pluginError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "enabled": input.Enabled})
}

func (s *Server) removePlugin(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	if err := s.plugins.Remove(c.Param("id")); err != nil {
		pluginError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) pluginHealth(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	data, status, err := s.plugins.Health(c.Request.Context(), c.Param("id"))
	if err != nil {
		code := http.StatusBadGateway
		if errors.Is(err, sql.ErrNoRows) {
			code = http.StatusNotFound
		}
		c.JSON(code, gin.H{"online": false, "error": err.Error(), "plugin_status": status})
		return
	}
	c.JSON(http.StatusOK, gin.H{"online": true, "plugin_status": status, "data": data})
}

func (s *Server) invokePluginAction(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	var input struct {
		Input map[string]any `json:"input"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action input"})
		return
	}
	data, status, err := s.plugins.Invoke(c.Request.Context(), c.Param("id"), c.Param("action"), input.Input)
	if err != nil {
		code := http.StatusBadGateway
		if errors.Is(err, sql.ErrNoRows) {
			code = http.StatusNotFound
		}
		if status >= 400 && status <= 599 {
			code = status
		}
		c.JSON(code, gin.H{"error": err.Error(), "plugin_status": status, "data": data})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "plugin_status": status, "data": data})
}

func (s *Server) pluginEvents(c *gin.Context) {
	if !s.requirePlugins(c) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := s.plugins.Events(c.Query("plugin_id"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": items})
}

func pluginError(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "plugin not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
