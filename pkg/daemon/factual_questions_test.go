package daemon

import (
	"testing"
)

func TestListFactualQuestions_CLI(t *testing.T) {
	// Test the CLI fallback for listing factual questions
	// This may error if bd is not in PATH or beads is not initialized
	questions, err := listFactualQuestionsCLI()
	if err != nil {
		// Error is expected if bd CLI is not available or beads not initialized
		t.Logf("CLI fallback returned error (expected if bd not in PATH): %v", err)
		// When error occurs, questions should be nil (not an empty slice)
		if questions != nil {
			t.Errorf("listFactualQuestionsCLI() on error returned non-nil questions: %v", questions)
		}
		return
	}
	// If no error, questions should be a valid slice (possibly empty)
	if questions == nil {
		t.Error("listFactualQuestionsCLI() returned nil without error, expected empty slice")
	}
}

func TestProcessFactualQuestions_Disabled(t *testing.T) {
	config := Config{
		SpawnFactualQuestions: false, // Disabled
	}
	d := NewWithConfig(config)

	spawned := d.ProcessFactualQuestions()
	if spawned != 0 {
		t.Errorf("ProcessFactualQuestions() with disabled feature = %d spawned, want 0", spawned)
	}
}

func TestProcessFactualQuestions_AtCapacity(t *testing.T) {
	config := Config{
		SpawnFactualQuestions: true,
		MaxAgents:             0, // Will be at capacity immediately
	}
	d := NewWithConfig(config)

	// Mock activeCountFunc to return high count (at capacity)
	d.activeCountFunc = func() int {
		return 100 // Always at capacity
	}

	spawned := d.ProcessFactualQuestions()
	if spawned != 0 {
		t.Errorf("ProcessFactualQuestions() at capacity = %d spawned, want 0", spawned)
	}
}

func TestProcessFactualQuestions_Enabled(t *testing.T) {
	config := Config{
		SpawnFactualQuestions: true,
		MaxAgents:             5, // Allow some spawns
		Verbose:               false,
	}
	d := NewWithConfig(config)

	// Mock functions
	var spawnCalls []string
	d.spawnFunc = func(beadsID string) error {
		spawnCalls = append(spawnCalls, beadsID)
		return nil
	}

	d.activeCountFunc = func() int {
		return 0 // Always available
	}

	// ProcessFactualQuestions should query for factual questions
	// It may spawn 0 if no questions exist, which is valid
	spawned := d.ProcessFactualQuestions()

	// This is a smoke test - we're verifying it doesn't crash
	// The actual number spawned depends on what's in the beads database
	if spawned < 0 {
		t.Errorf("ProcessFactualQuestions() = %d spawned, want >= 0", spawned)
	}

	// Verify spawn was called for each reported spawn
	if len(spawnCalls) != spawned {
		t.Errorf("ProcessFactualQuestions() reported %d spawned but spawnFunc called %d times", spawned, len(spawnCalls))
	}
}

func TestConfig_DefaultSpawnFactualQuestions(t *testing.T) {
	config := DefaultConfig()
	if config.SpawnFactualQuestions != false {
		t.Errorf("DefaultConfig().SpawnFactualQuestions = %v, want false (opt-in feature)", config.SpawnFactualQuestions)
	}
}
