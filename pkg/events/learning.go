package events

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"
)

// LearningStore holds aggregated learning data per skill.
// This is a functional computation from events.jsonl, not a cache.
// It respects the No Local Agent State constraint.
type LearningStore struct {
	Skills map[string]*SkillLearning `json:"skills"`
}

// SkillLearning holds aggregated metrics for a single skill.
type SkillLearning struct {
	SpawnCount            int                    `json:"spawn_count"`
	TotalCompletions      int                    `json:"total_completions"`
	SuccessCount          int                    `json:"success_count"`
	ForcedCount           int                    `json:"forced_count"`
	AbandonedCount        int                    `json:"abandoned_count"`
	SuccessRate           float64                `json:"success_rate"`
	AvgDurationSeconds    int                    `json:"avg_duration_seconds"`
	MedianDurationSeconds int                    `json:"median_duration_seconds"`
	VerificationFailures  int                    `json:"verification_failures"`
	VerificationBypasses  int                    `json:"verification_bypasses"`
	GateHitRates          map[string]*GateStats  `json:"gate_hit_rates"`
}

// GateStats tracks block/bypass/allow counts for a spawn gate.
type GateStats struct {
	BlockCount       int     `json:"block_count"`
	BypassCount      int     `json:"bypass_count"`
	AllowCount       int     `json:"allow_count"`
	TotalEvaluations int     `json:"total_evaluations"`
	BlockRate        float64 `json:"block_rate"`
}

// ComputeLearning reads events.jsonl and aggregates per-skill metrics.
// Returns an empty store if the file doesn't exist (graceful on first run).
// Skips corrupt/unparseable lines.
func ComputeLearning(eventsPath string) (*LearningStore, error) {
	store := &LearningStore{Skills: make(map[string]*SkillLearning)}

	f, err := os.Open(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	defer f.Close()

	// Track durations per skill for median calculation
	durations := make(map[string][]int)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024) // 256KB line buffer
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue // skip corrupt lines
		}

		skill, _ := event.Data["skill"].(string)

		switch event.Type {
		case EventTypeSessionSpawned:
			if skill == "" {
				continue
			}
			sl := store.ensureSkill(skill)
			sl.SpawnCount++

		case EventTypeAgentCompleted:
			if skill == "" {
				continue
			}
			sl := store.ensureSkill(skill)
			sl.TotalCompletions++

			outcome, _ := event.Data["outcome"].(string)
			switch outcome {
			case "success":
				sl.SuccessCount++
			case "forced":
				sl.ForcedCount++
			}

			if dur, ok := event.Data["duration_seconds"].(float64); ok && dur > 0 {
				durations[skill] = append(durations[skill], int(dur))
			}

		case EventTypeAgentAbandonedTelemetry:
			if skill == "" {
				continue
			}
			sl := store.ensureSkill(skill)
			sl.AbandonedCount++

		case EventTypeSpawnGateDecision:
			if skill == "" {
				continue
			}
			sl := store.ensureSkill(skill)
			gateName, _ := event.Data["gate_name"].(string)
			if gateName == "" {
				continue
			}
			gate := sl.ensureGate(gateName)
			gate.TotalEvaluations++

			decision, _ := event.Data["decision"].(string)
			switch decision {
			case "block":
				gate.BlockCount++
			case "bypass":
				gate.BypassCount++
			case "allow":
				gate.AllowCount++
			}

		case EventTypeVerificationFailed:
			if skill == "" {
				continue
			}
			sl := store.ensureSkill(skill)
			sl.VerificationFailures++

		case EventTypeVerificationBypassed:
			if skill == "" {
				continue
			}
			sl := store.ensureSkill(skill)
			sl.VerificationBypasses++
		}
	}

	// Compute derived metrics
	for name, sl := range store.Skills {
		total := sl.TotalCompletions + sl.AbandonedCount
		if total > 0 {
			sl.SuccessRate = float64(sl.SuccessCount) / float64(total)
		}

		if durs := durations[name]; len(durs) > 0 {
			sum := 0
			for _, d := range durs {
				sum += d
			}
			sl.AvgDurationSeconds = sum / len(durs)

			sort.Ints(durs)
			sl.MedianDurationSeconds = durs[len(durs)/2]
		}

		for _, gate := range sl.GateHitRates {
			if gate.TotalEvaluations > 0 {
				gate.BlockRate = float64(gate.BlockCount) / float64(gate.TotalEvaluations)
			}
		}
	}

	return store, nil
}

func (s *LearningStore) ensureSkill(name string) *SkillLearning {
	if sl, ok := s.Skills[name]; ok {
		return sl
	}
	sl := &SkillLearning{
		GateHitRates: make(map[string]*GateStats),
	}
	s.Skills[name] = sl
	return sl
}

func (sl *SkillLearning) ensureGate(name string) *GateStats {
	if g, ok := sl.GateHitRates[name]; ok {
		return g
	}
	g := &GateStats{}
	sl.GateHitRates[name] = g
	return g
}
