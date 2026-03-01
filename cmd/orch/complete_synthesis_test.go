package main

import (
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
)

func TestFindMatchingSynthesisTopics_NoSuggestions(t *testing.T) {
	result := findMatchingSynthesisTopics(nil, []string{"daemon"})
	if len(result) != 0 {
		t.Errorf("expected 0 matches for nil suggestions, got %d", len(result))
	}

	suggestions := &daemon.ReflectSuggestions{}
	result = findMatchingSynthesisTopics(suggestions, []string{"daemon"})
	if len(result) != 0 {
		t.Errorf("expected 0 matches for empty suggestions, got %d", len(result))
	}
}

func TestFindMatchingSynthesisTopics_NoKeywords(t *testing.T) {
	suggestions := &daemon.ReflectSuggestions{
		Synthesis: []daemon.SynthesisSuggestion{
			{Topic: "daemon", Count: 5},
		},
	}
	result := findMatchingSynthesisTopics(suggestions, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 matches for nil keywords, got %d", len(result))
	}
}

func TestFindMatchingSynthesisTopics_ExactMatch(t *testing.T) {
	suggestions := &daemon.ReflectSuggestions{
		Synthesis: []daemon.SynthesisSuggestion{
			{Topic: "daemon", Count: 5, Investigations: []string{"inv-1", "inv-2", "inv-3", "inv-4", "inv-5"}},
			{Topic: "spawn", Count: 3, Investigations: []string{"inv-6", "inv-7", "inv-8"}},
		},
	}

	result := findMatchingSynthesisTopics(suggestions, []string{"daemon", "lifecycle"})
	if len(result) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result))
	}
	if result[0].Topic != "daemon" {
		t.Errorf("expected topic 'daemon', got %q", result[0].Topic)
	}
}

func TestFindMatchingSynthesisTopics_SubstringMatch(t *testing.T) {
	suggestions := &daemon.ReflectSuggestions{
		Synthesis: []daemon.SynthesisSuggestion{
			{Topic: "daemon-lifecycle", Count: 4},
			{Topic: "verification", Count: 3},
		},
	}

	result := findMatchingSynthesisTopics(suggestions, []string{"daemon"})
	if len(result) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result))
	}
	if result[0].Topic != "daemon-lifecycle" {
		t.Errorf("expected topic 'daemon-lifecycle', got %q", result[0].Topic)
	}
}

func TestFindMatchingSynthesisTopics_CaseInsensitive(t *testing.T) {
	suggestions := &daemon.ReflectSuggestions{
		Synthesis: []daemon.SynthesisSuggestion{
			{Topic: "Daemon", Count: 5},
		},
	}

	result := findMatchingSynthesisTopics(suggestions, []string{"daemon"})
	if len(result) != 1 {
		t.Errorf("expected 1 match (case insensitive), got %d", len(result))
	}
}

func TestFindMatchingSynthesisTopics_BelowThreshold(t *testing.T) {
	suggestions := &daemon.ReflectSuggestions{
		Synthesis: []daemon.SynthesisSuggestion{
			{Topic: "daemon", Count: 2}, // Below CompletionSynthesisThreshold (3)
		},
	}

	result := findMatchingSynthesisTopics(suggestions, []string{"daemon"})
	if len(result) != 0 {
		t.Errorf("expected 0 matches below threshold, got %d", len(result))
	}
}

func TestFormatSynthesisCheckpointAdvisory_Empty(t *testing.T) {
	result := formatSynthesisCheckpointAdvisory(nil)
	if result != "" {
		t.Errorf("expected empty string for nil topics, got %q", result)
	}

	result = formatSynthesisCheckpointAdvisory([]daemon.SynthesisSuggestion{})
	if result != "" {
		t.Errorf("expected empty string for empty topics, got %q", result)
	}
}

func TestFormatSynthesisCheckpointAdvisory_SingleTopic(t *testing.T) {
	topics := []daemon.SynthesisSuggestion{
		{Topic: "daemon", Count: 5, Suggestion: "Consolidate daemon investigations"},
	}

	result := formatSynthesisCheckpointAdvisory(topics)
	if !strings.Contains(result, "SYNTHESIS CHECKPOINT") {
		t.Error("expected SYNTHESIS CHECKPOINT header")
	}
	if !strings.Contains(result, "daemon") {
		t.Error("expected topic name in output")
	}
	if !strings.Contains(result, "5") {
		t.Error("expected investigation count in output")
	}
}

func TestFormatSynthesisCheckpointAdvisory_MultipleTopic(t *testing.T) {
	topics := []daemon.SynthesisSuggestion{
		{Topic: "daemon", Count: 5},
		{Topic: "spawn", Count: 3},
	}

	result := formatSynthesisCheckpointAdvisory(topics)
	if !strings.Contains(result, "daemon") {
		t.Error("expected daemon topic in output")
	}
	if !strings.Contains(result, "spawn") {
		t.Error("expected spawn topic in output")
	}
}

func TestFormatSynthesisCheckpointAdvisory_HighCount(t *testing.T) {
	topics := []daemon.SynthesisSuggestion{
		{Topic: "daemon", Count: 12}, // Above SynthesisWarningThreshold (10)
	}

	result := formatSynthesisCheckpointAdvisory(topics)
	if !strings.Contains(result, "kb chronicle") || !strings.Contains(result, "daemon") {
		t.Error("expected kb chronicle suggestion for high-count topic")
	}
}

func TestSuggestionsAreFresh(t *testing.T) {
	// Fresh suggestions
	fresh := &daemon.ReflectSuggestions{
		Timestamp: time.Now().Add(-1 * time.Hour),
	}
	if !suggestionsAreFresh(fresh) {
		t.Error("expected 1-hour-old suggestions to be fresh")
	}

	// Stale suggestions
	stale := &daemon.ReflectSuggestions{
		Timestamp: time.Now().Add(-25 * time.Hour),
	}
	if suggestionsAreFresh(stale) {
		t.Error("expected 25-hour-old suggestions to be stale")
	}

	// Nil suggestions
	if suggestionsAreFresh(nil) {
		t.Error("expected nil suggestions to be stale")
	}
}
