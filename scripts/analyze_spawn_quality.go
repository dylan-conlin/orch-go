package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

type SessionSpawned struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Timestamp int64  `json:"timestamp"`
	Data      struct {
		BeadsID                string   `json:"beads_id"`
		Model                  string   `json:"model"`
		Skill                  string   `json:"skill"`
		Task                   string   `json:"task"`
		SpawnMode              string   `json:"spawn_mode"`
		NoTrack                bool     `json:"no_track"`
		GapContextQuality      int      `json:"gap_context_quality"`
		GapHasGaps             bool     `json:"gap_has_gaps"`
		SkipArtifactCheck      bool     `json:"skip_artifact_check"`
		Workspace              string   `json:"workspace"`
		GapMatchTotal          int      `json:"gap_match_total"`
		GapMatchConstraints    int      `json:"gap_match_constraints"`
		GapMatchDecisions      int      `json:"gap_match_decisions"`
		GapMatchInvestigations int      `json:"gap_match_investigations"`
		GapTypes               []string `json:"gap_types"`
	} `json:"data"`
}

type AgentCompleted struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Timestamp int64  `json:"timestamp"`
	Data      struct {
		BeadsID            string `json:"beads_id"`
		Outcome            string `json:"outcome"`
		Skill              string `json:"skill"`
		VerificationPassed bool   `json:"verification_passed"`
		TokensInput        int    `json:"tokens_input"`
		TokensOutput       int    `json:"tokens_output"`
		Workspace          string `json:"workspace"`
		Reason             string `json:"reason"`
		Orchestrator       bool   `json:"orchestrator"`
		Untracked          bool   `json:"untracked"`
	} `json:"data"`
}

type AgentAbandoned struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Timestamp int64  `json:"timestamp"`
	Data      struct {
		BeadsID   string `json:"beads_id"`
		Reason    string `json:"reason"`
		Workspace string `json:"workspace"`
	} `json:"data"`
}

func main() {
	eventsPath := os.ExpandEnv("$HOME/.orch/events.jsonl")
	file, err := os.Open(eventsPath)
	if err != nil {
		log.Fatalf("Failed to open events.jsonl: %v", err)
	}
	defer file.Close()

	spawns := make(map[string]SessionSpawned)
	completed := make(map[string]AgentCompleted)
	abandoned := make(map[string]AgentAbandoned)

	scanner := bufio.NewScanner(file)
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
			var spawn SessionSpawned
			if err := json.Unmarshal(scanner.Bytes(), &spawn); err == nil {
				if spawn.Data.BeadsID != "" {
					spawns[spawn.Data.BeadsID] = spawn
				}
			}
		case "agent.completed":
			var comp AgentCompleted
			if err := json.Unmarshal(scanner.Bytes(), &comp); err == nil {
				if comp.Data.BeadsID != "" {
					completed[comp.Data.BeadsID] = comp
				}
			}
		case "agent.abandoned":
			var aband AgentAbandoned
			if err := json.Unmarshal(scanner.Bytes(), &aband); err == nil {
				if aband.Data.BeadsID != "" {
					abandoned[aband.Data.BeadsID] = aband
				}
			}
		}
	}

	fmt.Printf("=== SPAWN PROMPT QUALITY ANALYSIS ===\n\n")
	fmt.Printf("Total spawns: %d\n", len(spawns))
	fmt.Printf("Total completed: %d\n", len(completed))
	fmt.Printf("Total abandoned: %d\n", len(abandoned))
	fmt.Printf("Completion rate: %.1f%%\n\n", float64(len(completed))/float64(len(spawns))*100)

	// 1. Completion rate by skill type
	skillSpawns := make(map[string]int)
	skillCompleted := make(map[string]int)
	skillAbandoned := make(map[string]int)

	for _, spawn := range spawns {
		skillSpawns[spawn.Data.Skill]++
		if _, ok := completed[spawn.Data.BeadsID]; ok {
			skillCompleted[spawn.Data.Skill]++
		}
		if _, ok := abandoned[spawn.Data.BeadsID]; ok {
			skillAbandoned[spawn.Data.Skill]++
		}
	}

	fmt.Printf("=== COMPLETION RATE BY SKILL ===\n")
	skills := make([]string, 0, len(skillSpawns))
	for skill := range skillSpawns {
		skills = append(skills, skill)
	}
	sort.Strings(skills)

	for _, skill := range skills {
		total := skillSpawns[skill]
		comp := skillCompleted[skill]
		aband := skillAbandoned[skill]
		rate := float64(comp) / float64(total) * 100
		fmt.Printf("%-25s: %3d spawned, %3d completed (%.1f%%), %3d abandoned\n",
			skill, total, comp, rate, aband)
	}
	fmt.Println()

	// 2. Model vs outcome
	modelSpawns := make(map[string]int)
	modelCompleted := make(map[string]int)

	for _, spawn := range spawns {
		modelSpawns[spawn.Data.Model]++
		if _, ok := completed[spawn.Data.BeadsID]; ok {
			modelCompleted[spawn.Data.Model]++
		}
	}

	fmt.Printf("=== COMPLETION RATE BY MODEL ===\n")
	models := make([]string, 0, len(modelSpawns))
	for model := range modelSpawns {
		models = append(models, model)
	}
	sort.Strings(models)

	for _, model := range models {
		total := modelSpawns[model]
		comp := modelCompleted[model]
		rate := float64(comp) / float64(total) * 100
		fmt.Printf("%-40s: %3d spawned, %3d completed (%.1f%%)\n",
			model, total, comp, rate)
	}
	fmt.Println()

	// 3. Triage routing vs outcome
	triageReady := 0
	triageBypassed := 0
	triageReadyCompleted := 0
	triageBypassedCompleted := 0

	for _, spawn := range spawns {
		if spawn.Data.NoTrack {
			triageBypassed++
			if _, ok := completed[spawn.Data.BeadsID]; ok {
				triageBypassedCompleted++
			}
		} else {
			triageReady++
			if _, ok := completed[spawn.Data.BeadsID]; ok {
				triageReadyCompleted++
			}
		}
	}

	fmt.Printf("=== TRIAGE ROUTING VS OUTCOME ===\n")
	fmt.Printf("Triage:ready (daemon):   %3d spawned, %3d completed (%.1f%%)\n",
		triageReady, triageReadyCompleted, float64(triageReadyCompleted)/float64(triageReady)*100)
	fmt.Printf("Triage bypassed (manual): %3d spawned, %3d completed (%.1f%%)\n\n",
		triageBypassed, triageBypassedCompleted, float64(triageBypassedCompleted)/float64(triageBypassed)*100)

	// 4. Spawn mode vs outcome
	modeSpawns := make(map[string]int)
	modeCompleted := make(map[string]int)

	for _, spawn := range spawns {
		mode := spawn.Data.SpawnMode
		if mode == "" {
			mode = "unknown"
		}
		modeSpawns[mode]++
		if _, ok := completed[spawn.Data.BeadsID]; ok {
			modeCompleted[mode]++
		}
	}

	fmt.Printf("=== SPAWN MODE VS OUTCOME ===\n")
	modes := make([]string, 0, len(modeSpawns))
	for mode := range modeSpawns {
		modes = append(modes, mode)
	}
	sort.Strings(modes)

	for _, mode := range modes {
		total := modeSpawns[mode]
		comp := modeCompleted[mode]
		rate := float64(comp) / float64(total) * 100
		fmt.Printf("%-15s: %3d spawned, %3d completed (%.1f%%)\n",
			mode, total, comp, rate)
	}
	fmt.Println()

	// 5. Prompt length vs outcome
	promptLengthBuckets := map[string]struct {
		total     int
		completed int
	}{
		"<500":      {},
		"500-1000":  {},
		"1000-2000": {},
		"2000-3000": {},
		">3000":     {},
	}

	for _, spawn := range spawns {
		length := len(spawn.Data.Task)
		var bucket string
		switch {
		case length < 500:
			bucket = "<500"
		case length < 1000:
			bucket = "500-1000"
		case length < 2000:
			bucket = "1000-2000"
		case length < 3000:
			bucket = "2000-3000"
		default:
			bucket = ">3000"
		}

		b := promptLengthBuckets[bucket]
		b.total++
		if _, ok := completed[spawn.Data.BeadsID]; ok {
			b.completed++
		}
		promptLengthBuckets[bucket] = b
	}

	fmt.Printf("=== PROMPT LENGTH VS OUTCOME ===\n")
	for _, bucket := range []string{"<500", "500-1000", "1000-2000", "2000-3000", ">3000"} {
		b := promptLengthBuckets[bucket]
		if b.total == 0 {
			continue
		}
		rate := float64(b.completed) / float64(b.total) * 100
		fmt.Printf("%-12s: %3d spawned, %3d completed (%.1f%%)\n",
			bucket, b.total, b.completed, rate)
	}
	fmt.Println()

	// 6. Context quality vs outcome
	qualityBuckets := map[string]struct {
		total     int
		completed int
	}{
		"<50":    {},
		"50-70":  {},
		"70-85":  {},
		"85-95":  {},
		"95-100": {},
	}

	for _, spawn := range spawns {
		quality := spawn.Data.GapContextQuality
		var bucket string
		switch {
		case quality < 50:
			bucket = "<50"
		case quality < 70:
			bucket = "50-70"
		case quality < 85:
			bucket = "70-85"
		case quality < 95:
			bucket = "85-95"
		default:
			bucket = "95-100"
		}

		b := qualityBuckets[bucket]
		b.total++
		if _, ok := completed[spawn.Data.BeadsID]; ok {
			b.completed++
		}
		qualityBuckets[bucket] = b
	}

	fmt.Printf("=== CONTEXT QUALITY VS OUTCOME ===\n")
	for _, bucket := range []string{"<50", "50-70", "70-85", "85-95", "95-100"} {
		b := qualityBuckets[bucket]
		if b.total == 0 {
			continue
		}
		rate := float64(b.completed) / float64(b.total) * 100
		fmt.Printf("%-10s: %3d spawned, %3d completed (%.1f%%)\n",
			bucket, b.total, b.completed, rate)
	}
	fmt.Println()

	// 7. Identify patterns in failed/abandoned spawns
	fmt.Printf("=== ABANDONED AGENT PATTERNS ===\n")
	abandonedBySkill := make(map[string][]string)
	for beadsID, aband := range abandoned {
		spawn, ok := spawns[beadsID]
		if !ok {
			continue
		}
		skill := spawn.Data.Skill
		reason := aband.Data.Reason
		if reason == "" {
			reason = "no reason given"
		}
		abandonedBySkill[skill] = append(abandonedBySkill[skill], reason)
	}

	for skill, reasons := range abandonedBySkill {
		fmt.Printf("\n%s (%d abandoned):\n", skill, len(reasons))
		reasonCounts := make(map[string]int)
		for _, reason := range reasons {
			reasonCounts[reason]++
		}
		for reason, count := range reasonCounts {
			fmt.Printf("  - %s: %d\n", reason, count)
		}
	}
	fmt.Println()

	// 8. Find spawns with no completion or abandonment (still running or stuck)
	fmt.Printf("=== AGENTS WITH NO OUTCOME (potentially stuck) ===\n")
	stuckCount := 0
	for beadsID, spawn := range spawns {
		if _, completed := completed[beadsID]; completed {
			continue
		}
		if _, abandoned := abandoned[beadsID]; abandoned {
			continue
		}
		stuckCount++
		if stuckCount <= 10 {
			fmt.Printf("%s: %s (skill: %s, model: %s)\n",
				beadsID, spawn.Data.Workspace, spawn.Data.Skill, spawn.Data.Model)
		}
	}
	if stuckCount > 10 {
		fmt.Printf("... and %d more\n", stuckCount-10)
	}
	fmt.Printf("Total stuck/in-progress: %d\n\n", stuckCount)

	// 9. Analyze prompt structure - look for key indicators
	fmt.Printf("=== PROMPT STRUCTURE ANALYSIS ===\n")

	structureMetrics := struct {
		hasScope          int
		hasKeyFiles       int
		hasPriorArtifacts int
		hasVerification   int
		hasExitCriteria   int
	}{}

	for _, spawn := range spawns {
		task := strings.ToLower(spawn.Data.Task)
		if strings.Contains(task, "scope:") || strings.Contains(task, "out of scope") || strings.Contains(task, "in scope") {
			structureMetrics.hasScope++
		}
		if strings.Contains(task, "key file") || strings.Contains(task, "relevant file") || strings.Contains(task, ".go") || strings.Contains(task, ".rb") {
			structureMetrics.hasKeyFiles++
		}
		if strings.Contains(task, "prior") || strings.Contains(task, "previous") || strings.Contains(task, "investigation") || strings.Contains(task, "decision") {
			structureMetrics.hasPriorArtifacts++
		}
		if strings.Contains(task, "test") || strings.Contains(task, "verify") || strings.Contains(task, "validation") {
			structureMetrics.hasVerification++
		}
		if strings.Contains(task, "exit criteria") || strings.Contains(task, "done when") || strings.Contains(task, "complete when") {
			structureMetrics.hasExitCriteria++
		}
	}

	total := float64(len(spawns))
	fmt.Printf("Prompts with scope boundaries: %d (%.1f%%)\n", structureMetrics.hasScope, float64(structureMetrics.hasScope)/total*100)
	fmt.Printf("Prompts with key files: %d (%.1f%%)\n", structureMetrics.hasKeyFiles, float64(structureMetrics.hasKeyFiles)/total*100)
	fmt.Printf("Prompts with prior artifacts: %d (%.1f%%)\n", structureMetrics.hasPriorArtifacts, float64(structureMetrics.hasPriorArtifacts)/total*100)
	fmt.Printf("Prompts with verification requirements: %d (%.1f%%)\n", structureMetrics.hasVerification, float64(structureMetrics.hasVerification)/total*100)
	fmt.Printf("Prompts with exit criteria: %d (%.1f%%)\n\n", structureMetrics.hasExitCriteria, float64(structureMetrics.hasExitCriteria)/total*100)
}
