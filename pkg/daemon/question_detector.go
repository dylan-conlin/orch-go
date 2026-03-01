// Package daemon provides autonomous overnight processing capabilities.
// This file contains QUESTION phase detection: finds agents that have reported
// Phase: QUESTION and surfaces them for notification. Tracks which agents have
// already been notified to avoid duplicate notifications.
package daemon

import (
	"fmt"
	"strings"
	"time"
)

// QuestionDetectionResult contains the result of a QUESTION phase detection scan.
type QuestionDetectionResult struct {
	// NewQuestions contains agents that entered QUESTION phase since last check.
	NewQuestions []QuestionAgent

	// TotalQuestions is the total number of agents currently in QUESTION phase.
	TotalQuestions int

	// Error is set if the detection failed.
	Error error

	// Message is a human-readable summary.
	Message string
}

// QuestionAgent represents an agent that is in QUESTION phase.
type QuestionAgent struct {
	BeadsID  string
	Title    string
	Phase    string // Full phase text (e.g., "QUESTION - Should we use JWT?")
	Question string // Extracted question text (after " - " separator)
}

// QuestionDetectionSnapshot is a point-in-time snapshot for the daemon status file.
type QuestionDetectionSnapshot struct {
	QuestionCount int       `json:"question_count"`
	LastCheck     time.Time `json:"last_check"`
}

// Snapshot converts a QuestionDetectionResult to a dashboard-ready snapshot.
func (r *QuestionDetectionResult) Snapshot() QuestionDetectionSnapshot {
	return QuestionDetectionSnapshot{
		QuestionCount: r.TotalQuestions,
		LastCheck:     time.Now(),
	}
}

// ShouldRunQuestionDetection returns true if QUESTION phase detection should run.
// Piggybacks on PhaseTimeout config — runs on the same interval since it uses
// the same agent discovery infrastructure.
func (d *Daemon) ShouldRunQuestionDetection() bool {
	if !d.Config.PhaseTimeoutEnabled || d.Config.PhaseTimeoutInterval <= 0 {
		return false
	}
	if d.lastQuestionDetection.IsZero() {
		return true
	}
	return time.Since(d.lastQuestionDetection) >= d.Config.PhaseTimeoutInterval
}

// RunPeriodicQuestionDetection detects agents in QUESTION phase.
// Returns only NEW question agents (not previously notified) to avoid duplicate notifications.
// Uses the same AgentDiscoverer as phase timeout — no additional API calls.
func (d *Daemon) RunPeriodicQuestionDetection() *QuestionDetectionResult {
	if !d.ShouldRunQuestionDetection() {
		return nil
	}

	agentDiscoverer := d.Agents
	if agentDiscoverer == nil {
		agentDiscoverer = &defaultAgentDiscoverer{}
	}

	agents, err := agentDiscoverer.GetActiveAgents()
	if err != nil {
		return &QuestionDetectionResult{
			Error:   err,
			Message: fmt.Sprintf("Question detection failed to list agents: %v", err),
		}
	}

	var newQuestions []QuestionAgent
	totalQuestions := 0

	for _, agent := range agents {
		if agent.BeadsID == "" {
			continue
		}

		// Extract phase name (before " - " detail)
		phaseName := agent.Phase
		if idx := strings.Index(phaseName, " - "); idx >= 0 {
			phaseName = strings.TrimSpace(phaseName[:idx])
		}

		if !strings.EqualFold(phaseName, "QUESTION") {
			continue
		}

		totalQuestions++

		// Check if we've already notified about this agent
		if d.questionNotified == nil {
			d.questionNotified = make(map[string]time.Time)
		}
		if _, alreadyNotified := d.questionNotified[agent.BeadsID]; alreadyNotified {
			continue
		}

		// Extract question text from phase detail
		questionText := ""
		if idx := strings.Index(agent.Phase, " - "); idx >= 0 {
			questionText = strings.TrimSpace(agent.Phase[idx+3:])
		}

		newQuestions = append(newQuestions, QuestionAgent{
			BeadsID:  agent.BeadsID,
			Title:    agent.Title,
			Phase:    agent.Phase,
			Question: questionText,
		})

		// Mark as notified
		d.questionNotified[agent.BeadsID] = time.Now()
	}

	// Clean up stale entries (agents no longer in QUESTION phase)
	d.cleanQuestionNotified(agents)

	d.lastQuestionDetection = time.Now()

	msg := fmt.Sprintf("Question detection: %d total, %d new", totalQuestions, len(newQuestions))
	return &QuestionDetectionResult{
		NewQuestions:    newQuestions,
		TotalQuestions:  totalQuestions,
		Message:         msg,
	}
}

// cleanQuestionNotified removes entries for agents that are no longer in QUESTION phase.
// This allows re-notification if an agent re-enters QUESTION phase later.
func (d *Daemon) cleanQuestionNotified(agents []ActiveAgent) {
	if d.questionNotified == nil {
		return
	}

	// Build set of current QUESTION agents
	currentQuestion := make(map[string]bool)
	for _, agent := range agents {
		phaseName := agent.Phase
		if idx := strings.Index(phaseName, " - "); idx >= 0 {
			phaseName = strings.TrimSpace(phaseName[:idx])
		}
		if strings.EqualFold(phaseName, "QUESTION") {
			currentQuestion[agent.BeadsID] = true
		}
	}

	// Remove entries no longer in QUESTION
	for id := range d.questionNotified {
		if !currentQuestion[id] {
			delete(d.questionNotified, id)
		}
	}
}
