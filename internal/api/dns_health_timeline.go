package api

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type healthTimelineItem struct {
	Kind   string    `json:"kind"`
	Status string    `json:"status"`
	Title  string    `json:"title"`
	Detail string    `json:"detail"`
	At     time.Time `json:"at"`
}

func (s *Server) dnsHealthTimeline(c *gin.Context) {
	stored, err := s.db.GetDNSHealthEvents(100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load DNS health history"})
		return
	}
	items := make([]healthTimelineItem, 0, len(stored)+100)
	for _, e := range stored {
		items = append(items, healthTimelineItem{Kind: e.Kind, Status: e.Status, Title: e.Title, Detail: e.Detail, At: e.At})
	}
	var current any
	if s.dnsWatch != nil {
		current = s.dnsWatch.Current()
		routes, routeErr := s.dnsWatch.Events()
		if routeErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load DNS route history"})
			return
		}
		for _, e := range routes {
			status, title := "warning", "External DNS detected"
			if e.Protected {
				status, title = "ok", "Local DNS Filter active"
			}
			if e.Error != "" {
				status, title = "error", "DNS route check failed"
			}
			detail := e.Interface + " · " + e.DNS
			if e.Error != "" {
				detail = e.Error
			}
			items = append(items, healthTimelineItem{Kind: "route", Status: status, Title: title, Detail: detail, At: e.ChangedAt})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].At.After(items[j].At) })
	if len(items) > 100 {
		items = items[:100]
	}
	c.JSON(http.StatusOK, gin.H{"current_route": current, "events": items})
}
