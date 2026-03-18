package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeEvents(t *testing.T, dir string, events []Event) string {
	t.Helper()
	path := filepath.Join(dir, "events.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, e := range events {
		data, _ := json.Marshal(e)
		f.Write(append(data, '\n'))
	}
	return path
}

func TestComputeLearning_Empty(t *testing.T) {
	dir := t.TempDir()
	path := writeEvents(t, dir, nil)

	store, err := ComputeLearning(path)
	if err != nil {
		t.Fatalf("ComputeLearning() error = %v", err)
	}
	if len(store.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(store.Skills))
	}
}

func TestComputeLearning_MissingFile(t *testing.T) {
	store, err := ComputeLearning("/nonexistent/events.jsonl")
	if err != nil {
		t.Fatalf("ComputeLearning() should not error on missing file, got %v", err)
	}
	if len(store.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(store.Skills))
	}
}

func TestComputeLearning_SuccessRate(t *testing.T) {
	dir := t.TempDir()

	events := []Event{
		// 2 successful completions for feature-impl
		{Type: EventTypeAgentCompleted, Timestamp: 1000, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success", "duration_seconds": float64(600),
			"verification_passed": true,
		}},
		{Type: EventTypeAgentCompleted, Timestamp: 2000, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success", "duration_seconds": float64(900),
			"verification_passed": true,
		}},
		// 1 forced completion for feature-impl
		{Type: EventTypeAgentCompleted, Timestamp: 3000, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "forced", "duration_seconds": float64(1200),
			"verification_passed": false,
		}},
		// 1 abandonment for feature-impl
		{Type: EventTypeAgentAbandonedTelemetry, Timestamp: 4000, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "abandoned", "duration_seconds": float64(2000),
		}},
	}

	path := writeEvents(t, dir, events)
	store, err := ComputeLearning(path)
	if err != nil {
		t.Fatalf("ComputeLearning() error = %v", err)
	}

	skill, ok := store.Skills["feature-impl"]
	if !ok {
		t.Fatal("expected feature-impl in skills")
	}

	if skill.TotalCompletions != 3 {
		t.Errorf("TotalCompletions = %d, want 3", skill.TotalCompletions)
	}
	if skill.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", skill.SuccessCount)
	}
	if skill.ForcedCount != 1 {
		t.Errorf("ForcedCount = %d, want 1", skill.ForcedCount)
	}
	if skill.AbandonedCount != 1 {
		t.Errorf("AbandonedCount = %d, want 1", skill.AbandonedCount)
	}

	// Success rate: 2 / (3 + 1) = 0.5
	expectedRate := 0.5
	if skill.SuccessRate < expectedRate-0.01 || skill.SuccessRate > expectedRate+0.01 {
		t.Errorf("SuccessRate = %f, want ~%f", skill.SuccessRate, expectedRate)
	}
}

func TestComputeLearning_CompletionTimes(t *testing.T) {
	dir := t.TempDir()

	events := []Event{
		{Type: EventTypeAgentCompleted, Timestamp: 1000, Data: map[string]interface{}{
			"skill": "investigation", "outcome": "success", "duration_seconds": float64(300),
		}},
		{Type: EventTypeAgentCompleted, Timestamp: 2000, Data: map[string]interface{}{
			"skill": "investigation", "outcome": "success", "duration_seconds": float64(600),
		}},
		{Type: EventTypeAgentCompleted, Timestamp: 3000, Data: map[string]interface{}{
			"skill": "investigation", "outcome": "success", "duration_seconds": float64(900),
		}},
	}

	path := writeEvents(t, dir, events)
	store, err := ComputeLearning(path)
	if err != nil {
		t.Fatalf("ComputeLearning() error = %v", err)
	}

	skill := store.Skills["investigation"]
	// avg = (300+600+900)/3 = 600
	if skill.AvgDurationSeconds != 600 {
		t.Errorf("AvgDurationSeconds = %d, want 600", skill.AvgDurationSeconds)
	}
	// median of [300, 600, 900] = 600
	if skill.MedianDurationSeconds != 600 {
		t.Errorf("MedianDurationSeconds = %d, want 600", skill.MedianDurationSeconds)
	}
}

func TestComputeLearning_GateHitRates(t *testing.T) {
	dir := t.TempDir()

	events := []Event{
		// Gate blocked
		{Type: EventTypeSpawnGateDecision, Timestamp: 1000, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "block", "skill": "feature-impl",
		}},
		// Gate allowed (3 times)
		{Type: EventTypeSpawnGateDecision, Timestamp: 1001, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "skill": "feature-impl",
		}},
		{Type: EventTypeSpawnGateDecision, Timestamp: 1002, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "skill": "feature-impl",
		}},
		{Type: EventTypeSpawnGateDecision, Timestamp: 1003, Data: map[string]interface{}{
			"gate_name": "hotspot", "decision": "allow", "skill": "feature-impl",
		}},
		// Verification failed
		{Type: EventTypeVerificationFailed, Timestamp: 2000, Data: map[string]interface{}{
			"skill": "feature-impl", "gates_failed": []interface{}{"test_evidence"},
		}},
		// Verification bypassed
		{Type: EventTypeVerificationBypassed, Timestamp: 2001, Data: map[string]interface{}{
			"skill": "feature-impl", "gate": "test_evidence",
		}},
	}

	path := writeEvents(t, dir, events)
	store, err := ComputeLearning(path)
	if err != nil {
		t.Fatalf("ComputeLearning() error = %v", err)
	}

	skill := store.Skills["feature-impl"]

	// Gate hits: hotspot blocked 1 out of 4 evaluations = 25%
	hotspot, ok := skill.GateHitRates["hotspot"]
	if !ok {
		t.Fatal("expected hotspot in gate_hit_rates")
	}
	if hotspot.BlockCount != 1 {
		t.Errorf("hotspot BlockCount = %d, want 1", hotspot.BlockCount)
	}
	if hotspot.TotalEvaluations != 4 {
		t.Errorf("hotspot TotalEvaluations = %d, want 4", hotspot.TotalEvaluations)
	}

	// Verification failures
	if skill.VerificationFailures != 1 {
		t.Errorf("VerificationFailures = %d, want 1", skill.VerificationFailures)
	}
	if skill.VerificationBypasses != 1 {
		t.Errorf("VerificationBypasses = %d, want 1", skill.VerificationBypasses)
	}
}

func TestComputeLearning_MultipleSkills(t *testing.T) {
	dir := t.TempDir()

	events := []Event{
		{Type: EventTypeAgentCompleted, Timestamp: 1000, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success",
		}},
		{Type: EventTypeAgentCompleted, Timestamp: 2000, Data: map[string]interface{}{
			"skill": "investigation", "outcome": "success",
		}},
		{Type: EventTypeAgentCompleted, Timestamp: 3000, Data: map[string]interface{}{
			"skill": "architect", "outcome": "success",
		}},
	}

	path := writeEvents(t, dir, events)
	store, err := ComputeLearning(path)
	if err != nil {
		t.Fatalf("ComputeLearning() error = %v", err)
	}

	if len(store.Skills) != 3 {
		t.Errorf("expected 3 skills, got %d", len(store.Skills))
	}

	for _, name := range []string{"feature-impl", "investigation", "architect"} {
		if _, ok := store.Skills[name]; !ok {
			t.Errorf("expected skill %q in store", name)
		}
	}
}

func TestComputeLearning_SkipsEventsWithoutSkill(t *testing.T) {
	dir := t.TempDir()

	events := []Event{
		// Completed without skill field — should be ignored
		{Type: EventTypeAgentCompleted, Timestamp: 1000, Data: map[string]interface{}{
			"outcome": "success",
		}},
		// With skill
		{Type: EventTypeAgentCompleted, Timestamp: 2000, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success",
		}},
	}

	path := writeEvents(t, dir, events)
	store, err := ComputeLearning(path)
	if err != nil {
		t.Fatalf("ComputeLearning() error = %v", err)
	}

	if len(store.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(store.Skills))
	}
}

func TestComputeLearning_IgnoresNonRelevantEvents(t *testing.T) {
	dir := t.TempDir()

	events := []Event{
		// Daemon-internal events should be ignored
		{Type: "daemon.recovery", Timestamp: 1000, Data: map[string]interface{}{
			"message": "something",
		}},
		{Type: "daemon.phase_timeout", Timestamp: 1001, Data: map[string]interface{}{
			"message": "something",
		}},
		{Type: EventTypeSessionSpawned, Timestamp: 1002, Data: map[string]interface{}{
			"skill": "feature-impl",
		}},
		// Only this should be counted
		{Type: EventTypeAgentCompleted, Timestamp: 2000, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success",
		}},
	}

	path := writeEvents(t, dir, events)
	store, err := ComputeLearning(path)
	if err != nil {
		t.Fatalf("ComputeLearning() error = %v", err)
	}

	skill := store.Skills["feature-impl"]
	if skill.TotalCompletions != 1 {
		t.Errorf("TotalCompletions = %d, want 1", skill.TotalCompletions)
	}
	// SpawnCount should be tracked from session.spawned
	if skill.SpawnCount != 1 {
		t.Errorf("SpawnCount = %d, want 1", skill.SpawnCount)
	}
}

func TestComputeLearning_CorruptLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	// Write a mix of valid and corrupt lines
	f, _ := os.Create(path)
	valid := Event{Type: EventTypeAgentCompleted, Timestamp: 1000, Data: map[string]interface{}{
		"skill": "feature-impl", "outcome": "success",
	}}
	data, _ := json.Marshal(valid)
	f.Write(append(data, '\n'))
	f.WriteString("this is not json\n")
	f.Write(append(data, '\n'))
	f.Close()

	store, err := ComputeLearning(path)
	if err != nil {
		t.Fatalf("ComputeLearning() should skip corrupt lines, got error: %v", err)
	}

	skill := store.Skills["feature-impl"]
	if skill.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2 (should skip corrupt line)", skill.SuccessCount)
	}
}

func TestComputeLearningInWindow_FiltersByTimestamp(t *testing.T) {
	dir := t.TempDir()

	now := time.Now()
	oldTS := now.Add(-60 * 24 * time.Hour).Unix()  // 60 days ago
	newTS := now.Add(-10 * 24 * time.Hour).Unix()   // 10 days ago

	evts := []Event{
		// Old event — success
		{Type: EventTypeAgentCompleted, Timestamp: oldTS, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success",
		}},
		// Old event — success
		{Type: EventTypeAgentCompleted, Timestamp: oldTS + 100, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success",
		}},
		// Recent event — abandoned
		{Type: EventTypeAgentAbandonedTelemetry, Timestamp: newTS, Data: map[string]interface{}{
			"skill": "feature-impl",
		}},
		// Recent event — success
		{Type: EventTypeAgentCompleted, Timestamp: newTS + 100, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success",
		}},
	}

	path := writeEvents(t, dir, evts)
	boundary := now.Add(-30 * 24 * time.Hour)

	// Previous window: everything before 30 days ago
	prevStore, err := ComputeLearningInWindow(path, time.Time{}, boundary)
	if err != nil {
		t.Fatalf("ComputeLearningInWindow() error = %v", err)
	}
	prevSkill := prevStore.Skills["feature-impl"]
	if prevSkill == nil {
		t.Fatal("expected feature-impl in previous window")
	}
	if prevSkill.SuccessCount != 2 {
		t.Errorf("previous SuccessCount = %d, want 2", prevSkill.SuccessCount)
	}
	if prevSkill.AbandonedCount != 0 {
		t.Errorf("previous AbandonedCount = %d, want 0", prevSkill.AbandonedCount)
	}

	// Recent window: everything after 30 days ago
	recentStore, err := ComputeLearningInWindow(path, boundary, time.Time{})
	if err != nil {
		t.Fatalf("ComputeLearningInWindow() error = %v", err)
	}
	recentSkill := recentStore.Skills["feature-impl"]
	if recentSkill == nil {
		t.Fatal("expected feature-impl in recent window")
	}
	if recentSkill.SuccessCount != 1 {
		t.Errorf("recent SuccessCount = %d, want 1", recentSkill.SuccessCount)
	}
	if recentSkill.AbandonedCount != 1 {
		t.Errorf("recent AbandonedCount = %d, want 1", recentSkill.AbandonedCount)
	}
	// Recent success rate: 1 / (1 + 1) = 0.5
	if recentSkill.SuccessRate < 0.49 || recentSkill.SuccessRate > 0.51 {
		t.Errorf("recent SuccessRate = %f, want ~0.5", recentSkill.SuccessRate)
	}
}

func TestComputeLearningInWindow_EmptyWindow(t *testing.T) {
	dir := t.TempDir()

	now := time.Now()
	oldTS := now.Add(-60 * 24 * time.Hour).Unix()

	evts := []Event{
		{Type: EventTypeAgentCompleted, Timestamp: oldTS, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success",
		}},
	}

	path := writeEvents(t, dir, evts)

	// Query a window that contains no events
	after := now.Add(-10 * 24 * time.Hour)
	store, err := ComputeLearningInWindow(path, after, time.Time{})
	if err != nil {
		t.Fatalf("ComputeLearningInWindow() error = %v", err)
	}
	if len(store.Skills) != 0 {
		t.Errorf("expected 0 skills in empty window, got %d", len(store.Skills))
	}
}

func TestComputeLearning_ReworkTracking(t *testing.T) {
	dir := t.TempDir()

	evts := []Event{
		// 3 completions for feature-impl
		{Type: EventTypeAgentCompleted, Timestamp: 1000, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success",
		}},
		{Type: EventTypeAgentCompleted, Timestamp: 2000, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success",
		}},
		{Type: EventTypeAgentCompleted, Timestamp: 3000, Data: map[string]interface{}{
			"skill": "feature-impl", "outcome": "success",
		}},
		// 1 rework for feature-impl
		{Type: EventTypeAgentReworked, Timestamp: 4000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "orch-go-abc12",
		}},
		// 2 completions for investigation, 1 rework
		{Type: EventTypeAgentCompleted, Timestamp: 5000, Data: map[string]interface{}{
			"skill": "investigation", "outcome": "success",
		}},
		{Type: EventTypeAgentCompleted, Timestamp: 6000, Data: map[string]interface{}{
			"skill": "investigation", "outcome": "success",
		}},
		{Type: EventTypeAgentReworked, Timestamp: 7000, Data: map[string]interface{}{
			"skill": "investigation", "beads_id": "orch-go-def34",
		}},
	}

	path := writeEvents(t, dir, evts)
	store, err := ComputeLearning(path)
	if err != nil {
		t.Fatalf("ComputeLearning() error = %v", err)
	}

	// feature-impl: 1 rework / 3 completions = 0.333
	fi := store.Skills["feature-impl"]
	if fi.ReworkCount != 1 {
		t.Errorf("feature-impl ReworkCount = %d, want 1", fi.ReworkCount)
	}
	expectedRate := 1.0 / 3.0
	if fi.ReworkRate < expectedRate-0.01 || fi.ReworkRate > expectedRate+0.01 {
		t.Errorf("feature-impl ReworkRate = %f, want ~%f", fi.ReworkRate, expectedRate)
	}

	// investigation: 1 rework / 2 completions = 0.5
	inv := store.Skills["investigation"]
	if inv.ReworkCount != 1 {
		t.Errorf("investigation ReworkCount = %d, want 1", inv.ReworkCount)
	}
	expectedRate = 0.5
	if inv.ReworkRate < expectedRate-0.01 || inv.ReworkRate > expectedRate+0.01 {
		t.Errorf("investigation ReworkRate = %f, want ~%f", inv.ReworkRate, expectedRate)
	}
}

func TestComputeLearning_ReworkWithZeroCompletions(t *testing.T) {
	dir := t.TempDir()

	evts := []Event{
		// Rework without any completions (edge case)
		{Type: EventTypeAgentReworked, Timestamp: 1000, Data: map[string]interface{}{
			"skill": "feature-impl", "beads_id": "orch-go-abc12",
		}},
	}

	path := writeEvents(t, dir, evts)
	store, err := ComputeLearning(path)
	if err != nil {
		t.Fatalf("ComputeLearning() error = %v", err)
	}

	fi := store.Skills["feature-impl"]
	if fi.ReworkCount != 1 {
		t.Errorf("ReworkCount = %d, want 1", fi.ReworkCount)
	}
	// With 0 completions, rework rate should be 0 (avoid division by zero)
	if fi.ReworkRate != 0 {
		t.Errorf("ReworkRate = %f, want 0 (zero completions)", fi.ReworkRate)
	}
}
