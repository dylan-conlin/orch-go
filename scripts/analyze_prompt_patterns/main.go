package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/dylan-conlin/orch-go/scripts/eventtypes"
)

func main() {
	eventsPath := os.ExpandEnv("$HOME/.orch/events.jsonl")
	file, err := os.Open(eventsPath)
	if err != nil {
		log.Fatalf("Failed to open events.jsonl: %v", err)
	}
	defer file.Close()

	spawns := make(map[string]eventtypes.SessionSpawned)
	completed := make(map[string]eventtypes.AgentCompleted)
	abandoned := make(map[string]eventtypes.AgentAbandoned)

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var rawEvent map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &rawEvent); err != nil {
			continue
		}

		eventType, ok := rawEvent["type"].(string)
		if !ok {
			continue
		}

		switch eventType {
		case "session.spawned":
			var spawn eventtypes.SessionSpawned
			if err := json.Unmarshal(scanner.Bytes(), &spawn); err == nil {
				if spawn.Data.BeadsID != "" {
					spawns[spawn.Data.BeadsID] = spawn
				}
			}
		case "agent.completed":
			var comp eventtypes.AgentCompleted
			if err := json.Unmarshal(scanner.Bytes(), &comp); err == nil {
				if comp.Data.BeadsID != "" {
					completed[comp.Data.BeadsID] = comp
				}
			}
		case "agent.abandoned":
			var aband eventtypes.AgentAbandoned
			if err := json.Unmarshal(scanner.Bytes(), &aband); err == nil {
				if aband.Data.BeadsID != "" {
					abandoned[aband.Data.BeadsID] = aband
				}
			}
		}
	}

	// Analyze successful prompts vs failed prompts
	var successfulPrompts []eventtypes.SessionSpawned
	var failedPrompts []eventtypes.SessionSpawned
	var inProgressPrompts []eventtypes.SessionSpawned

	for beadsID, spawn := range spawns {
		if comp, ok := completed[beadsID]; ok && comp.Data.VerificationPassed {
			successfulPrompts = append(successfulPrompts, spawn)
		} else if _, ok := abandoned[beadsID]; ok {
			failedPrompts = append(failedPrompts, spawn)
		} else if _, ok := completed[beadsID]; !ok {
			inProgressPrompts = append(inProgressPrompts, spawn)
		}
	}

	fmt.Printf("=== PROMPT PATTERN DEEP DIVE ===\n\n")
	fmt.Printf("Successful: %d, Failed: %d, In-progress: %d\n\n",
		len(successfulPrompts), len(failedPrompts), len(inProgressPrompts))

	// Common keywords in successful prompts
	successKeywords := make(map[string]int)
	failKeywords := make(map[string]int)

	keywords := []string{
		"test", "verify", "exit criteria", "project_dir", "scope",
		"implement", "fix", "add", "update", "refactor",
		"investigation", "bug", "feature", "prior", "context",
		"file:", "function", "method", "class", "package",
		"done when", "deliverable", "requirement", "constraint",
	}

	for _, spawn := range successfulPrompts {
		task := strings.ToLower(spawn.Data.Task)
		for _, kw := range keywords {
			if strings.Contains(task, kw) {
				successKeywords[kw]++
			}
		}
	}

	for _, spawn := range failedPrompts {
		task := strings.ToLower(spawn.Data.Task)
		for _, kw := range keywords {
			if strings.Contains(task, kw) {
				failKeywords[kw]++
			}
		}
	}

	fmt.Printf("=== KEYWORD PREVALENCE ===\n")
	fmt.Printf("%-20s %10s %10s %10s %10s\n", "Keyword", "Success", "Success%", "Fail", "Fail%")
	fmt.Printf("%-20s %10s %10s %10s %10s\n", "-------", "-------", "--------", "----", "-----")

	type kwStat struct {
		keyword    string
		successPct float64
		failPct    float64
	}
	var kwStats []kwStat

	for _, kw := range keywords {
		succCount := successKeywords[kw]
		failCount := failKeywords[kw]
		succPct := float64(succCount) / float64(len(successfulPrompts)) * 100
		failPct := float64(failCount) / float64(len(failedPrompts)) * 100

		kwStats = append(kwStats, kwStat{kw, succPct, failPct})
	}

	// Sort by success percentage
	sort.Slice(kwStats, func(i, j int) bool {
		return kwStats[i].successPct > kwStats[j].successPct
	})

	for _, stat := range kwStats {
		fmt.Printf("%-20s %10d %9.1f%% %10d %8.1f%%\n",
			stat.keyword,
			successKeywords[stat.keyword], stat.successPct,
			failKeywords[stat.keyword], stat.failPct)
	}
	fmt.Println()

	// Show sample successful prompts
	fmt.Printf("=== SAMPLE SUCCESSFUL PROMPTS (first 5) ===\n\n")
	for i, spawn := range successfulPrompts {
		if i >= 5 {
			break
		}
		comp := completed[spawn.Data.BeadsID]
		fmt.Printf("BeadsID: %s\n", spawn.Data.BeadsID)
		fmt.Printf("Skill: %s, Model: %s, Mode: %s\n", spawn.Data.Skill, spawn.Data.Model, spawn.Data.SpawnMode)
		fmt.Printf("Quality: %d, NoTrack: %v\n", spawn.Data.GapContextQuality, spawn.Data.NoTrack)
		fmt.Printf("Outcome: %s\n", comp.Data.Outcome)
		fmt.Printf("Task: %s\n", truncate(spawn.Data.Task, 300))
		fmt.Printf("Reason: %s\n", truncate(comp.Data.Reason, 200))
		fmt.Println("---")
	}

	// Show sample failed prompts
	fmt.Printf("\n=== SAMPLE FAILED/ABANDONED PROMPTS (first 5) ===\n\n")
	for i, spawn := range failedPrompts {
		if i >= 5 {
			break
		}
		aband := abandoned[spawn.Data.BeadsID]
		fmt.Printf("BeadsID: %s\n", spawn.Data.BeadsID)
		fmt.Printf("Skill: %s, Model: %s, Mode: %s\n", spawn.Data.Skill, spawn.Data.Model, spawn.Data.SpawnMode)
		fmt.Printf("Quality: %d, NoTrack: %v\n", spawn.Data.GapContextQuality, spawn.Data.NoTrack)
		fmt.Printf("Abandon reason: %s\n", aband.Data.Reason)
		fmt.Printf("Task: %s\n", truncate(spawn.Data.Task, 300))
		fmt.Println("---")
	}

	// Identify AT-RISK agents (in-progress but likely stuck)
	fmt.Printf("\n=== AT-RISK AGENTS (in-progress, likely stuck) ===\n")
	fmt.Printf("Total in-progress: %d\n\n", len(inProgressPrompts))

	// Sort by timestamp (oldest first)
	sort.Slice(inProgressPrompts, func(i, j int) bool {
		return spawns[inProgressPrompts[i].Data.BeadsID].Timestamp < spawns[inProgressPrompts[j].Data.BeadsID].Timestamp
	})

	// Show oldest 10
	for i, spawn := range inProgressPrompts {
		if i >= 10 {
			break
		}
		fmt.Printf("BeadsID: %s\n", spawn.Data.BeadsID)
		fmt.Printf("Age: %d hours\n", (1770928000-spawn.Timestamp)/3600) // Approximate current timestamp
		fmt.Printf("Skill: %s, Model: %s, Mode: %s\n", spawn.Data.Skill, spawn.Data.Model, spawn.Data.SpawnMode)
		fmt.Printf("Task: %s\n", truncate(spawn.Data.Task, 200))
		fmt.Println("---")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
