package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func now() time.Time { return time.Now() }
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func oneOfKey(value string, options ...string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}
func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width < 2 {
		return value[:1]
	}
	return value[:minInt(len(value), width-1)] + "…"
}
func pad(value string, width int) string {
	value = truncate(value, width)
	return value + strings.Repeat(" ", maxInt(0, width-lipgloss.Width(value)))
}
func comma(value int) string {
	s := fmt.Sprintf("%d", value)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
func sparkline(points []TimelinePoint, width int) string {
	if width < 1 || len(points) == 0 {
		return "No traffic history yet"
	}
	if len(points) > width {
		points = points[len(points)-width:]
	}
	maxValue := 1
	for _, point := range points {
		if point.Count > maxValue {
			maxValue = point.Count
		}
	}
	levels := []rune("▁▂▃▄▅▆▇█")
	var out strings.Builder
	for _, point := range points {
		out.WriteRune(levels[point.Count*(len(levels)-1)/maxValue])
	}
	return out.String()
}
func since(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t).Round(time.Second)
	if d < time.Second {
		return "now"
	}
	return d.String() + " ago"
}
