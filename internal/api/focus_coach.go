package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
)

func focusCoachPrompt(session discipline.FocusSession) (string, error) {
	payload, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	prompt := "You are the private Discipline AI Session Coach. Analyze only this local focus-session record. Separate facts from suggestions. Identify the strongest focus period only if the data supports it. Explain useful patterns in application time, DNS-visible domains, observations, and logged distractions. Give 2-4 specific improvements for the next session. Never treat a DNS domain observation as proof that the user actively viewed content. Return exactly one JSON object with keys: subject, body, category, severity, action_label, action_view. severity must be normal, attention, or success. category must be Focus coach. action_label should be Open Command Center and action_view should be command. Keep body below 1800 characters and do not use markdown fences.\n\nLOCAL FOCUS SESSION:\n"
	return prompt + string(payload), nil
}

func localFocusCoach(session discipline.FocusSession) aiBriefing {
	apps, domains, observations := 0, 0, 0
	for _, item := range session.Activity {
		observations += item.Observations
		if item.Kind == "app" {
			apps++
		} else if item.Kind == "domain" {
			domains++
		}
	}
	minutes := session.ActualSeconds / 60
	body := fmt.Sprintf("You completed %d focused minute(s) of %d planned.\nObserved locally: %d application(s), %d DNS-visible domain(s), and %d activity observations.\nLogged distractions: %d.", minutes, session.PlannedMinutes, apps, domains, observations, len(session.Distractions))
	if len(session.Distractions) > 0 {
		body += "\nNext session: remove the most repeated distraction before starting and keep the capsule resources narrow."
	} else {
		body += "\nNext session: keep the same environment and define one concrete finish condition before starting."
	}
	return aiBriefing{Subject: "Focus review · " + session.TaskTitle, Body: body, Category: "Focus coach", Severity: "success", ActionLabel: "Open Command Center", ActionView: "command"}
}

func (s *Server) generateFocusCoach(ctx context.Context, sessionID int64) (int64, string, error) {
	session, err := s.discipline.FocusSessionDetails(sessionID)
	if err != nil {
		return 0, "", err
	}
	message := localFocusCoach(session)
	mode := "local"
	provider, providerErr := s.defaultBriefingProvider()
	if providerErr == nil {
		prompt, err := focusCoachPrompt(session)
		if err != nil {
			return 0, "ai", err
		}
		requestCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		raw, err := callBriefingProvider(requestCtx, provider, prompt)
		if err != nil {
			s.recordAIProviderResult(provider.ID, false, err.Error())
			return 0, "ai", err
		}
		message, err = parseAIBriefing(raw)
		if err != nil {
			s.recordAIProviderResult(provider.ID, false, err.Error())
			return 0, "ai", err
		}
		message.Category, message.ActionLabel, message.ActionView = "Focus coach", "Open Command Center", "command"
		mode = "ai"
		s.recordAIProviderResult(provider.ID, true, "")
	} else if !errors.Is(providerErr, sql.ErrNoRows) {
		return 0, "ai", providerErr
	}
	sender := "Discipline Session Coach"
	if mode == "ai" {
		sender = strings.TrimSpace(provider.Name)
	}
	id, err := s.discipline.CreateInboxMessage(discipline.InboxInput{Sender: sender, Subject: message.Subject, Body: message.Body, Category: message.Category, Severity: message.Severity, ActionLabel: message.ActionLabel, ActionView: message.ActionView})
	return id, mode, err
}
