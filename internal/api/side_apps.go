package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/RDXFGXY1/dns-filter-app/internal/sideapps"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
)

func sideAppError(c *gin.Context, e error) {
	code := http.StatusBadRequest
	if errors.Is(e, sql.ErrNoRows) {
		code = 404
	}
	if errors.Is(e, sideapps.ErrForbidden) {
		code = 403
	}
	if errors.Is(e, sideapps.ErrConflict) {
		code = 409
	}
	c.JSON(code, gin.H{"error": e.Error()})
}
func (s *Server) requireSideApps(c *gin.Context) bool {
	if s.sideApps == nil {
		c.JSON(503, gin.H{"error": "Side Apps is unavailable"})
		return false
	}
	return true
}
func readSideAppUpload(c *gin.Context) ([]byte, error) {
	file, header, e := c.Request.FormFile("package")
	if e != nil {
		return nil, errors.New("choose an .adnf application package")
	}
	defer file.Close()
	defer c.Request.MultipartForm.RemoveAll()
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".adnf") {
		return nil, errors.New("application packages must use the .adnf extension")
	}
	raw, e := io.ReadAll(io.LimitReader(file, sideapps.MaxPackageBytes+1))
	if e != nil {
		return nil, e
	}
	return raw, nil
}
func (s *Server) registerSideAppRoutes(api *gin.RouterGroup) {
	group := api.Group("/side-apps")
	group.Use(func(c *gin.Context) {
		if !s.requireSideApps(c) {
			c.Abort()
			return
		}
		c.Next()
	})
	group.GET("", func(c *gin.Context) {
		apps, e := s.sideApps.List()
		if e != nil {
			c.JSON(500, gin.H{"error": "Could not load installed apps"})
			return
		}
		c.JSON(200, apps)
	})
	group.POST("/preview", func(c *gin.Context) {
		raw, e := readSideAppUpload(c)
		if e != nil {
			sideAppError(c, e)
			return
		}
		p, e := sideapps.Parse(raw)
		if e != nil {
			sideAppError(c, e)
			return
		}
		if _, e = p.Render(); e != nil {
			sideAppError(c, e)
			return
		}
		c.JSON(200, p)
	})
	group.POST("/install", func(c *gin.Context) {
		raw, e := readSideAppUpload(c)
		if e != nil {
			sideAppError(c, e)
			return
		}
		p, e := sideapps.Parse(raw)
		if e != nil {
			sideAppError(c, e)
			return
		}
		if _, e = p.Render(); e != nil {
			sideAppError(c, e)
			return
		}
		p, e = s.sideApps.Install(raw, c.PostForm("sha256"), c.PostForm("replace") == "true")
		if e != nil {
			sideAppError(c, e)
			return
		}
		c.JSON(201, gin.H{"success": true, "id": p.Manifest.ID})
	})
	group.POST("/:id/enabled", func(c *gin.Context) {
		var input struct {
			Enabled *bool `json:"enabled"`
		}
		if c.ShouldBindJSON(&input) != nil || input.Enabled == nil {
			c.JSON(400, gin.H{"error": "enabled is required"})
			return
		}
		if e := s.sideApps.SetEnabled(c.Param("id"), *input.Enabled); e != nil {
			sideAppError(c, e)
			return
		}
		c.JSON(200, gin.H{"success": true})
	})
	group.DELETE("/:id", func(c *gin.Context) {
		if e := s.sideApps.Remove(c.Param("id")); e != nil {
			sideAppError(c, e)
			return
		}
		c.JSON(200, gin.H{"success": true})
	})
	group.GET("/:id/storage", func(c *gin.Context) {
		raw, e := s.sideApps.ReadStorage(c.Param("id"))
		if e != nil {
			sideAppError(c, e)
			return
		}
		c.JSON(200, gin.H{"value": raw})
	})
	group.PUT("/:id/storage", func(c *gin.Context) {
		raw, e := io.ReadAll(io.LimitReader(c.Request.Body, sideapps.MaxStorageBytes+1))
		if e != nil || len(raw) > sideapps.MaxStorageBytes || !json.Valid(raw) {
			c.JSON(400, gin.H{"error": "storage must be valid JSON up to 1 MB"})
			return
		}
		if e = s.sideApps.WriteStorage(c.Param("id"), raw); e != nil {
			sideAppError(c, e)
			return
		}
		c.JSON(200, gin.H{"success": true})
	})
}
func (s *Server) runSideApp(c *gin.Context) {
	if !s.requireSideApps(c) {
		return
	}
	p, e := s.sideApps.Package(c.Param("id"))
	if e != nil {
		sideAppError(c, e)
		return
	}
	body, e := p.Render()
	if e != nil {
		sideAppError(c, e)
		return
	}
	c.Header("Content-Security-Policy", sideapps.AppCSP)
	c.Header("X-Frame-Options", "SAMEORIGIN")
	c.Header("Cache-Control", "no-store")
	c.Data(200, "text/html; charset=utf-8", body)
}
