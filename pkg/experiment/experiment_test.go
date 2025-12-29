package experiment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateExperiment(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "experiment-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create experiment
	expDir, err := CreateExperiment(tmpDir, "test-exp", "Test hypothesis")
	if err != nil {
		t.Fatalf("CreateExperiment failed: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(expDir); os.IsNotExist(err) {
		t.Errorf("experiment directory not created: %s", expDir)
	}

	// Verify experiment.yaml exists and is valid
	exp, err := LoadExperiment(expDir)
	if err != nil {
		t.Fatalf("LoadExperiment failed: %v", err)
	}

	if exp.Name != "test-exp" {
		t.Errorf("expected name 'test-exp', got '%s'", exp.Name)
	}

	if exp.Hypothesis != "Test hypothesis" {
		t.Errorf("expected hypothesis 'Test hypothesis', got '%s'", exp.Hypothesis)
	}

	if exp.Status != StatusRunning {
		t.Errorf("expected status 'running', got '%s'", exp.Status)
	}

	if len(exp.Conditions) != 2 {
		t.Errorf("expected 2 default conditions, got %d", len(exp.Conditions))
	}

	// Verify analysis.md exists
	analysisPath := filepath.Join(expDir, "analysis.md")
	if _, err := os.Stat(analysisPath); os.IsNotExist(err) {
		t.Error("analysis.md not created")
	}

	// Verify runs directory exists
	runsDir := filepath.Join(expDir, "runs")
	if _, err := os.Stat(runsDir); os.IsNotExist(err) {
		t.Error("runs directory not created")
	}
}

func TestSaveAndLoadRun(t *testing.T) {
	// Create temp directory with experiment structure
	tmpDir, err := os.MkdirTemp("", "experiment-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	expDir, err := CreateExperiment(tmpDir, "test-runs", "Test runs")
	if err != nil {
		t.Fatalf("CreateExperiment failed: %v", err)
	}

	// Create a run
	run := &Run{
		Condition: "A",
		RunNumber: 1,
		Timestamp: "2025-12-28T10:00:00Z",
		SessionID: "ses_test123",
		BeadsID:   "proj-abc",
		Metrics: RunMetrics{
			ToolCallsBeforeRecognition: 3,
			RecognizedExistingAnswer:   true,
			TimeToRecognitionSeconds:   45,
		},
		Notes: "Test run",
	}

	// Save run
	if err := SaveRun(expDir, run); err != nil {
		t.Fatalf("SaveRun failed: %v", err)
	}

	// Verify file exists
	runFile := filepath.Join(expDir, "runs", "A-1.json")
	if _, err := os.Stat(runFile); os.IsNotExist(err) {
		t.Errorf("run file not created: %s", runFile)
	}

	// Load runs
	runs, err := LoadRuns(expDir)
	if err != nil {
		t.Fatalf("LoadRuns failed: %v", err)
	}

	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	loaded := runs[0]
	if loaded.Condition != "A" {
		t.Errorf("expected condition 'A', got '%s'", loaded.Condition)
	}
	if loaded.Metrics.ToolCallsBeforeRecognition != 3 {
		t.Errorf("expected 3 tool calls, got %d", loaded.Metrics.ToolCallsBeforeRecognition)
	}
	if !loaded.Metrics.RecognizedExistingAnswer {
		t.Error("expected recognized_existing_answer to be true")
	}
}

func TestGetNextRunNumber(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "experiment-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	expDir, err := CreateExperiment(tmpDir, "test-next", "Test next run number")
	if err != nil {
		t.Fatalf("CreateExperiment failed: %v", err)
	}

	// First run should be 1
	next, err := GetNextRunNumber(expDir, "A")
	if err != nil {
		t.Fatalf("GetNextRunNumber failed: %v", err)
	}
	if next != 1 {
		t.Errorf("expected next run number 1, got %d", next)
	}

	// Add a run
	run := &Run{Condition: "A", RunNumber: 1}
	if err := SaveRun(expDir, run); err != nil {
		t.Fatalf("SaveRun failed: %v", err)
	}

	// Next should be 2
	next, err = GetNextRunNumber(expDir, "A")
	if err != nil {
		t.Fatalf("GetNextRunNumber failed: %v", err)
	}
	if next != 2 {
		t.Errorf("expected next run number 2, got %d", next)
	}

	// Condition B should still be 1
	next, err = GetNextRunNumber(expDir, "B")
	if err != nil {
		t.Fatalf("GetNextRunNumber failed: %v", err)
	}
	if next != 1 {
		t.Errorf("expected next run number 1 for B, got %d", next)
	}
}

func TestRunsByCondition(t *testing.T) {
	runs := []Run{
		{Condition: "A", RunNumber: 1},
		{Condition: "A", RunNumber: 2},
		{Condition: "B", RunNumber: 1},
	}

	grouped := RunsByCondition(runs)

	if len(grouped["A"]) != 2 {
		t.Errorf("expected 2 runs for A, got %d", len(grouped["A"]))
	}
	if len(grouped["B"]) != 1 {
		t.Errorf("expected 1 run for B, got %d", len(grouped["B"]))
	}
}

func TestListExperiments(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "experiment-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// List empty should return nil, not error
	experiments, err := ListExperiments(tmpDir)
	if err != nil {
		t.Fatalf("ListExperiments failed on empty dir: %v", err)
	}
	if experiments != nil && len(experiments) != 0 {
		t.Errorf("expected empty list, got %d experiments", len(experiments))
	}

	// Create two experiments
	_, err = CreateExperiment(tmpDir, "exp1", "Hypothesis 1")
	if err != nil {
		t.Fatalf("CreateExperiment failed: %v", err)
	}
	_, err = CreateExperiment(tmpDir, "exp2", "Hypothesis 2")
	if err != nil {
		t.Fatalf("CreateExperiment failed: %v", err)
	}

	// List should return both
	experiments, err = ListExperiments(tmpDir)
	if err != nil {
		t.Fatalf("ListExperiments failed: %v", err)
	}
	if len(experiments) != 2 {
		t.Errorf("expected 2 experiments, got %d", len(experiments))
	}
}

func TestGenerateAnalysis(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "experiment-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	expDir, err := CreateExperiment(tmpDir, "test-analysis", "Test analysis generation")
	if err != nil {
		t.Fatalf("CreateExperiment failed: %v", err)
	}

	// Add some runs
	runs := []*Run{
		{
			Condition: "A",
			RunNumber: 1,
			Metrics: RunMetrics{
				ToolCallsBeforeRecognition: 2,
				RecognizedExistingAnswer:   true,
				TimeToRecognitionSeconds:   30,
			},
		},
		{
			Condition: "A",
			RunNumber: 2,
			Metrics: RunMetrics{
				ToolCallsBeforeRecognition: 4,
				RecognizedExistingAnswer:   true,
				TimeToRecognitionSeconds:   60,
			},
		},
		{
			Condition: "B",
			RunNumber: 1,
			Metrics: RunMetrics{
				ToolCallsBeforeRecognition: 10,
				RecognizedExistingAnswer:   false,
				TimeToRecognitionSeconds:   120,
			},
		},
	}

	for _, run := range runs {
		if err := SaveRun(expDir, run); err != nil {
			t.Fatalf("SaveRun failed: %v", err)
		}
	}

	// Generate analysis
	analysisPath, err := GenerateAnalysis(expDir)
	if err != nil {
		t.Fatalf("GenerateAnalysis failed: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(analysisPath)
	if err != nil {
		t.Fatalf("failed to read analysis.md: %v", err)
	}

	// Verify key content is present
	contentStr := string(content)
	if !contains(contentStr, "test-analysis") {
		t.Error("analysis.md missing experiment name")
	}
	if !contains(contentStr, "Test analysis generation") {
		t.Error("analysis.md missing hypothesis")
	}
	if !contains(contentStr, "Per-Condition Metrics") {
		t.Error("analysis.md missing metrics section")
	}
	if !contains(contentStr, "Individual Runs") {
		t.Error("analysis.md missing individual runs section")
	}
}

func TestRunMetricsSerialization(t *testing.T) {
	metrics := RunMetrics{
		ToolCallsBeforeRecognition: 5,
		RecognizedExistingAnswer:   true,
		TimeToRecognitionSeconds:   90,
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var loaded RunMetrics
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if loaded.ToolCallsBeforeRecognition != 5 {
		t.Errorf("expected 5, got %d", loaded.ToolCallsBeforeRecognition)
	}
	if !loaded.RecognizedExistingAnswer {
		t.Error("expected true")
	}
	if loaded.TimeToRecognitionSeconds != 90 {
		t.Errorf("expected 90, got %d", loaded.TimeToRecognitionSeconds)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
