package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type networkProfileState struct {
	Filtering  bool            `json:"filtering"`
	Apps       map[string]bool `json:"apps"`
	Categories map[string]bool `json:"categories"`
	Whitelist  []string        `json:"whitelist"`
}

type networkProfile struct {
	ID          int64               `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	State       networkProfileState `json:"state"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

func (s *Server) ensureNetworkProfiles() error {
	_, err := s.db.GetDB().Exec(`CREATE TABLE IF NOT EXISTS network_profiles(
		id INTEGER PRIMARY KEY AUTOINCREMENT,name TEXT NOT NULL UNIQUE,description TEXT NOT NULL DEFAULT '',
		state_json TEXT NOT NULL,created_at TIMESTAMP NOT NULL,updated_at TIMESTAMP NOT NULL)`)
	return err
}

func (s *Server) captureNetworkProfile() networkProfileState {
	state := networkProfileState{Filtering: true, Apps: map[string]bool{}, Categories: map[string]bool{}, Whitelist: s.filter.GetWhitelist()}
	if s.dnsServer != nil {
		state.Filtering, _ = s.dnsServer.FilteringStatus()
	}
	for _, app := range s.filter.GetApps() {
		id, _ := app["id"].(string)
		blocked, _ := app["blocked"].(bool)
		if id != "" {
			state.Apps[id] = blocked
		}
	}
	for _, category := range s.filter.GetAllCategories() {
		state.Categories[category.ID] = category.Enabled
	}
	sort.Strings(state.Whitelist)
	return state
}

func (s *Server) listNetworkProfiles(c *gin.Context) {
	if err := s.ensureNetworkProfiles(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows, err := s.db.GetDB().Query(`SELECT id,name,description,state_json,created_at,updated_at FROM network_profiles ORDER BY updated_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	profiles := []networkProfile{}
	for rows.Next() {
		var p networkProfile
		var raw string
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &raw, &p.CreatedAt, &p.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := json.Unmarshal([]byte(raw), &p.State); err != nil {
			continue
		}
		profiles = append(profiles, p)
	}
	c.JSON(http.StatusOK, profiles)
}

func (s *Server) createNetworkProfile(c *gin.Context) {
	if err := s.ensureNetworkProfiles(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var in struct{ Name, Description string }
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile name required"})
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) > 60 || len(in.Description) > 240 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile name or description is too long"})
		return
	}
	raw, _ := json.Marshal(s.captureNetworkProfile())
	now := time.Now().UTC()
	result, err := s.db.GetDB().Exec(`INSERT INTO network_profiles(name,description,state_json,created_at,updated_at) VALUES(?,?,?,?,?)`, in.Name, strings.TrimSpace(in.Description), string(raw), now, now)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a profile with this name already exists"})
		return
	}
	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id})
}

func (s *Server) loadNetworkProfile(id int64) (networkProfile, error) {
	var p networkProfile
	var raw string
	err := s.db.GetDB().QueryRow(`SELECT id,name,description,state_json,created_at,updated_at FROM network_profiles WHERE id=?`, id).Scan(&p.ID, &p.Name, &p.Description, &raw, &p.CreatedAt, &p.UpdatedAt)
	if err == nil {
		err = json.Unmarshal([]byte(raw), &p.State)
	}
	return p, err
}

func profileID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	return id, err == nil && id > 0
}

func profileChanges(current, target networkProfileState) gin.H {
	apps, categories := 0, 0
	for id, value := range target.Apps {
		if current.Apps[id] != value {
			apps++
		}
	}
	for id, value := range target.Categories {
		if current.Categories[id] != value {
			categories++
		}
	}
	currentWhite := map[string]bool{}
	for _, domain := range current.Whitelist {
		currentWhite[domain] = true
	}
	targetWhite := map[string]bool{}
	for _, domain := range target.Whitelist {
		targetWhite[domain] = true
	}
	added, removed := 0, 0
	for domain := range targetWhite {
		if !currentWhite[domain] {
			added++
		}
	}
	for domain := range currentWhite {
		if !targetWhite[domain] {
			removed++
		}
	}
	return gin.H{"filtering_changes": current.Filtering != target.Filtering, "app_changes": apps, "category_changes": categories, "whitelist_added": added, "whitelist_removed": removed}
}

func (s *Server) previewNetworkProfile(c *gin.Context) {
	if err := s.ensureNetworkProfiles(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	id, ok := profileID(c)
	if !ok {
		c.JSON(400, gin.H{"error": "invalid profile"})
		return
	}
	p, err := s.loadNetworkProfile(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "profile not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": p, "changes": profileChanges(s.captureNetworkProfile(), p.State)})
}

func (s *Server) applyNetworkProfile(c *gin.Context) {
	if err := s.ensureNetworkProfiles(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	id, ok := profileID(c)
	if !ok {
		c.JSON(400, gin.H{"error": "invalid profile"})
		return
	}
	p, err := s.loadNetworkProfile(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "profile not found"})
		return
	}
	for appID, blocked := range p.State.Apps {
		if err := s.filter.SetAppBlocked(appID, blocked); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	}
	for categoryID, enabled := range p.State.Categories {
		if err := s.filter.ToggleCategory(categoryID, enabled); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	}
	current := s.filter.GetWhitelist()
	wanted := map[string]bool{}
	for _, domain := range p.State.Whitelist {
		wanted[domain] = true
	}
	for _, domain := range current {
		if !wanted[domain] {
			s.filter.RemoveFromWhitelist(domain)
		}
	}
	for domain := range wanted {
		s.filter.AddToWhitelist(domain)
	}
	if s.dnsServer != nil {
		s.dnsServer.SetFiltering(p.State.Filtering)
		s.dnsServer.ClearCache()
	}
	_, _ = s.db.GetDB().Exec(`UPDATE network_profiles SET updated_at=? WHERE id=?`, time.Now().UTC(), id)
	c.JSON(http.StatusOK, gin.H{"success": true, "name": p.Name})
}

func (s *Server) deleteNetworkProfile(c *gin.Context) {
	if err := s.ensureNetworkProfiles(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	id, ok := profileID(c)
	if !ok {
		c.JSON(400, gin.H{"error": "invalid profile"})
		return
	}
	result, err := s.db.GetDB().Exec(`DELETE FROM network_profiles WHERE id=?`, id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		c.JSON(404, gin.H{"error": "profile not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
