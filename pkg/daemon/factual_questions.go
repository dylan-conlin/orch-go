// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/dylan-conlin/orch-go/pkg/beads"
)

// ListFactualQuestions retrieves ready factual questions from beads.
// Queries for type=question with subtype:factual label.
// Uses the beads RPC daemon if available, falling back to the bd CLI if not.
func ListFactualQuestions() ([]Issue, error) {
	// Try to use the beads RPC client first
	socketPath, err := beads.FindSocketPath("")
	if err == nil {
		// Use WithAutoReconnect for resilience against daemon restarts/transient issues
		client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if err := client.Connect(); err == nil {
			defer client.Close()
			// Query for type=question with subtype:factual label
			beadsIssues, err := client.Ready(&beads.ReadyArgs{
				Type:   "question",
				Labels: []string{"subtype:factual"},
				Limit:  0, // Get ALL factual questions
			})
			if err == nil {
				return convertBeadsIssues(beadsIssues), nil
			}
			// Fall through to CLI fallback on Ready() error
		}
		// Fall through to CLI fallback on Connect() error
	}

	// Fallback to CLI if daemon unavailable
	return listFactualQuestionsCLI()
}

// listFactualQuestionsCLI retrieves factual questions by shelling out to bd CLI.
func listFactualQuestionsCLI() ([]Issue, error) {
	// Use bd ready with type and label filters
	cmd := exec.Command("bd", "ready", "--type", "question", "--label", "subtype:factual", "--json", "--limit", "0")
	cmd.Env = os.Environ() // Inherit env (including BEADS_NO_DAEMON)
	output, err := bdOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run bd ready for factual questions: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse factual questions: %w", err)
	}

	return issues, nil
}

// ProcessFactualQuestions processes factual questions if the feature is enabled.
// Returns the number of spawned factual questions, or 0 if disabled/no questions.
func (d *Daemon) ProcessFactualQuestions() int {
	if !d.Config.SpawnFactualQuestions {
		return 0
	}

	// Check if we have capacity
	if d.AtCapacity() {
		if d.Config.Verbose {
			log.Printf("Skipping factual questions: at capacity")
		}
		return 0
	}

	// Query for factual questions
	questions, err := ListFactualQuestions()
	if err != nil {
		if d.Config.Verbose {
			log.Printf("Failed to list factual questions: %v", err)
		}
		return 0
	}

	if len(questions) == 0 {
		if d.Config.Verbose {
			log.Printf("No factual questions found")
		}
		return 0
	}

	spawned := 0
	for _, question := range questions {
		// Check capacity before spawning each question
		if d.AtCapacity() {
			if d.Config.Verbose {
				log.Printf("At capacity after spawning %d factual questions", spawned)
			}
			break
		}

		// Check if issue is spawnable (not blocked, not in_progress, etc.)
		if reason := d.checkRejectionReason(question); reason != "" {
			if d.Config.Verbose {
				log.Printf("Skipping factual question %s: %s", question.ID, reason)
			}
			continue
		}

		// Check for existing session
		if HasExistingSessionForBeadsID(question.ID) {
			if d.Config.Verbose {
				log.Printf("Skipping factual question %s: existing session found", question.ID)
			}
			continue
		}

		// Check for Phase: Complete
		if hasComplete, _ := HasPhaseComplete(question.ID); hasComplete {
			if d.Config.Verbose {
				log.Printf("Skipping factual question %s: Phase: Complete found", question.ID)
			}
			continue
		}

		// Acquire pool slot if configured
		var slot *Slot
		if d.Pool != nil {
			slot = d.Pool.TryAcquire()
			if slot == nil {
				if d.Config.Verbose {
					log.Printf("At capacity (pool): stopping factual question processing")
				}
				break
			}
			slot.BeadsID = question.ID
		}

		// Mark as spawned before spawning to prevent race condition
		if d.SpawnedIssues != nil {
			d.SpawnedIssues.MarkSpawned(question.ID)
		}

		// Spawn the investigation
		if err := d.spawnFunc(question.ID); err != nil {
			// Unmark on spawn failure
			if d.SpawnedIssues != nil {
				d.SpawnedIssues.Unmark(question.ID)
			}
			// Release slot on spawn failure
			if d.Pool != nil && slot != nil {
				d.Pool.Release(slot)
			}
			if d.Config.Verbose {
				log.Printf("Failed to spawn factual question %s: %v", question.ID, err)
			}
			continue
		}

		// Record successful spawn for rate limiting
		if d.RateLimiter != nil {
			d.RateLimiter.RecordSpawn()
		}

		spawned++
		if d.Config.Verbose {
			log.Printf("Spawned investigation for factual question: %s - %s", question.ID, question.Title)
		}

		// Honor spawn delay
		if d.Config.SpawnDelay > 0 && spawned < len(questions) {
			// Sleep is acceptable here since this is a background daemon
			// and we're explicitly rate-limiting spawns
			// time.Sleep(d.Config.SpawnDelay)
			// Actually, we should NOT sleep here - let the main loop handle delays
			// between spawns to avoid blocking the poll cycle
		}
	}

	return spawned
}
