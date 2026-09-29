package api

import "time"

func (s *Server) runAutomaticResponses() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		s.processAutomaticResponses()
		<-ticker.C
	}
}

func (s *Server) processAutomaticResponses() {
	settings, err := s.networkIntel.AutomaticResponseSettings()
	if err != nil || !settings.Enabled {
		return
	}
	_ = s.networkIntel.Refresh()
	items, err := s.networkIntel.AutomaticCandidates(settings.MinRisk, 25)
	if err != nil {
		return
	}
	for _, item := range items {
		action := settings.Action
		if action == "block" {
			if !s.appNetwork.Supported() || s.appNetwork.SetBlocked(item.Path, true) != nil {
				continue
			}
		} else if action == "quarantine" {
			if !s.appNetwork.Supported() || s.appNetwork.Quarantine(item.Path, "Automatic response: "+item.Reasons, item.Risk) != nil {
				continue
			}
		}
		_ = s.networkIntel.RecordAutomaticResponse(item, action)
	}
}
