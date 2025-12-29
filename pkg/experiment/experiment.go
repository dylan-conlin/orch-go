// Package experiment provides types and utilities for running behavioral experiments
// on agent context/attention patterns.
package experiment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Status represents the current state of an experiment.
type Status string

const (
	StatusRunning   Status = "running"
	StatusComplete  Status = "complete"
	StatusAbandoned Status = "abandoned"
)

// SpawnContextMods defines how to modify the spawn context for a condition.
type SpawnContextMods struct {
	ConstraintPosition string `yaml:"constraint_position,omitempty" json:"constraint_position,omitempty"` // top, buried
	ConstraintFraming  string `yaml:"constraint_framing,omitempty" json:"constraint_framing,omitempty"`   // known_answer, constraint, decision
}

// Condition defines a single experimental condition.
type Condition struct {
	Description      string           `yaml:"description" json:"description"`
	SpawnContextMods SpawnContextMods `yaml:"spawn_context_mods,omitempty" json:"spawn_context_mods,omitempty"`
}

// Experiment represents a behavioral experiment configuration.
type Experiment struct {
	Name             string               `yaml:"name" json:"name"`
	Hypothesis       string               `yaml:"hypothesis" json:"hypothesis"`
	Created          string               `yaml:"created" json:"created"` // YYYY-MM-DD format
	Status           Status               `yaml:"status" json:"status"`
	Conditions       map[string]Condition `yaml:"conditions" json:"conditions"`
	RunsPerCondition int                  `yaml:"runs_per_condition" json:"runs_per_condition"`
	Metrics          []string             `yaml:"metrics" json:"metrics"`
}

// RunMetrics holds the measured metrics for a single run.
type RunMetrics struct {
	ToolCallsBeforeRecognition int  `json:"tool_calls_before_recognition"`
	RecognizedExistingAnswer   bool `json:"recognized_existing_answer"`
	TimeToRecognitionSeconds   int  `json:"time_to_recognition_seconds"`
}

// Run represents a single execution of an experimental condition.
type Run struct {
	Condition string     `json:"condition"`
	RunNumber int        `json:"run"`
	Timestamp string     `json:"timestamp"`
	SessionID string     `json:"session_id"`
	BeadsID   string     `json:"beads_id"`
	Metrics   RunMetrics `json:"metrics"`
	Notes     string     `json:"notes,omitempty"`
}

// ExperimentDir returns the directory path for a named experiment.
func ExperimentDir(projectDir, name string) string {
	return filepath.Join(projectDir, ".orch", "experiments", name)
}

// ExperimentsDir returns the base experiments directory.
func ExperimentsDir(projectDir string) string {
	return filepath.Join(projectDir, ".orch", "experiments")
}

// LoadExperiment loads an experiment from its directory.
func LoadExperiment(expDir string) (*Experiment, error) {
	yamlPath := filepath.Join(expDir, "experiment.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read experiment.yaml: %w", err)
	}

	var exp Experiment
	if err := yaml.Unmarshal(data, &exp); err != nil {
		return nil, fmt.Errorf("failed to parse experiment.yaml: %w", err)
	}

	return &exp, nil
}

// Save writes the experiment configuration to experiment.yaml.
func (e *Experiment) Save(expDir string) error {
	data, err := yaml.Marshal(e)
	if err != nil {
		return fmt.Errorf("failed to marshal experiment: %w", err)
	}

	yamlPath := filepath.Join(expDir, "experiment.yaml")
	if err := os.WriteFile(yamlPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write experiment.yaml: %w", err)
	}

	return nil
}

// LoadRuns loads all runs for a given experiment.
func LoadRuns(expDir string) ([]Run, error) {
	runsDir := filepath.Join(expDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No runs yet
		}
		return nil, fmt.Errorf("failed to read runs directory: %w", err)
	}

	var runs []Run
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(runsDir, entry.Name()))
		if err != nil {
			continue // Skip files we can't read
		}

		var run Run
		if err := json.Unmarshal(data, &run); err != nil {
			continue // Skip files we can't parse
		}

		runs = append(runs, run)
	}

	return runs, nil
}

// SaveRun saves a run to the runs directory.
func SaveRun(expDir string, run *Run) error {
	runsDir := filepath.Join(expDir, "runs")
	if err := os.MkdirAll(runsDir, 0755); err != nil {
		return fmt.Errorf("failed to create runs directory: %w", err)
	}

	// Generate filename: <condition>-<run_number>.json
	filename := fmt.Sprintf("%s-%d.json", run.Condition, run.RunNumber)
	filepath := filepath.Join(runsDir, filename)

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal run: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write run file: %w", err)
	}

	return nil
}

// GetNextRunNumber returns the next run number for a condition.
func GetNextRunNumber(expDir, condition string) (int, error) {
	runs, err := LoadRuns(expDir)
	if err != nil {
		return 1, err
	}

	maxRun := 0
	for _, run := range runs {
		if run.Condition == condition && run.RunNumber > maxRun {
			maxRun = run.RunNumber
		}
	}

	return maxRun + 1, nil
}

// RunsByCondition groups runs by condition ID.
func RunsByCondition(runs []Run) map[string][]Run {
	grouped := make(map[string][]Run)
	for _, run := range runs {
		grouped[run.Condition] = append(grouped[run.Condition], run)
	}
	return grouped
}

// ListExperiments returns all experiments in the project.
func ListExperiments(projectDir string) ([]*Experiment, error) {
	expBaseDir := ExperimentsDir(projectDir)
	entries, err := os.ReadDir(expBaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No experiments directory yet
		}
		return nil, fmt.Errorf("failed to read experiments directory: %w", err)
	}

	var experiments []*Experiment
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		expDir := filepath.Join(expBaseDir, entry.Name())
		exp, err := LoadExperiment(expDir)
		if err != nil {
			continue // Skip directories that don't have valid experiment.yaml
		}

		experiments = append(experiments, exp)
	}

	// Sort by creation date (newest first)
	sort.Slice(experiments, func(i, j int) bool {
		return experiments[i].Created > experiments[j].Created
	})

	return experiments, nil
}

// CreateExperiment scaffolds a new experiment directory with template files.
func CreateExperiment(projectDir, name, hypothesis string) (string, error) {
	// Generate dated directory name
	date := time.Now().Format("2006-01-02")
	dirName := fmt.Sprintf("%s-%s", date, name)
	expDir := ExperimentDir(projectDir, dirName)

	// Check if directory already exists
	if _, err := os.Stat(expDir); err == nil {
		return "", fmt.Errorf("experiment directory already exists: %s", dirName)
	}

	// Create experiment directory structure
	if err := os.MkdirAll(expDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create experiment directory: %w", err)
	}

	runsDir := filepath.Join(expDir, "runs")
	if err := os.MkdirAll(runsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create runs directory: %w", err)
	}

	// Create default experiment.yaml
	exp := &Experiment{
		Name:       name,
		Hypothesis: hypothesis,
		Created:    date,
		Status:     StatusRunning,
		Conditions: map[string]Condition{
			"A": {
				Description: "Control condition",
				SpawnContextMods: SpawnContextMods{
					ConstraintPosition: "top",
					ConstraintFraming:  "known_answer",
				},
			},
			"B": {
				Description: "Treatment condition",
				SpawnContextMods: SpawnContextMods{
					ConstraintPosition: "buried",
					ConstraintFraming:  "constraint",
				},
			},
		},
		RunsPerCondition: 3,
		Metrics: []string{
			"tool_calls_before_recognition",
			"recognized_existing_answer",
			"time_to_recognition_seconds",
		},
	}

	if err := exp.Save(expDir); err != nil {
		return "", err
	}

	// Create analysis.md template
	analysisTemplate := fmt.Sprintf(`# Experiment Analysis: %s

## Hypothesis

%s

## Conditions

| Condition | Description | Position | Framing |
|-----------|-------------|----------|---------|
| A | Control condition | top | known_answer |
| B | Treatment condition | buried | constraint |

## Results

*Run 'orch experiment analyze %s' to populate this section.*

## Observations

*Add qualitative observations here.*

## Conclusions

*Add conclusions after sufficient data is collected.*

## Next Steps

*What follow-up experiments or actions are suggested?*
`, name, hypothesis, dirName)

	analysisPath := filepath.Join(expDir, "analysis.md")
	if err := os.WriteFile(analysisPath, []byte(analysisTemplate), 0644); err != nil {
		return "", fmt.Errorf("failed to write analysis.md: %w", err)
	}

	return expDir, nil
}

// GenerateAnalysis creates or updates the analysis.md file with current results.
func GenerateAnalysis(expDir string) (string, error) {
	exp, err := LoadExperiment(expDir)
	if err != nil {
		return "", err
	}

	runs, err := LoadRuns(expDir)
	if err != nil {
		return "", err
	}

	byCondition := RunsByCondition(runs)

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Experiment Analysis: %s\n\n", exp.Name))
	sb.WriteString(fmt.Sprintf("**Created:** %s\n", exp.Created))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n\n", exp.Status))

	sb.WriteString("## Hypothesis\n\n")
	sb.WriteString(exp.Hypothesis + "\n\n")

	sb.WriteString("## Conditions\n\n")
	sb.WriteString("| Condition | Description | Runs Completed |\n")
	sb.WriteString("|-----------|-------------|----------------|\n")

	// Get sorted condition IDs
	var conditionIDs []string
	for id := range exp.Conditions {
		conditionIDs = append(conditionIDs, id)
	}
	sort.Strings(conditionIDs)

	for _, id := range conditionIDs {
		cond := exp.Conditions[id]
		runCount := len(byCondition[id])
		sb.WriteString(fmt.Sprintf("| %s | %s | %d/%d |\n", id, cond.Description, runCount, exp.RunsPerCondition))
	}

	sb.WriteString("\n## Results Summary\n\n")

	if len(runs) == 0 {
		sb.WriteString("*No runs completed yet.*\n\n")
	} else {
		sb.WriteString("### Per-Condition Metrics\n\n")
		sb.WriteString("| Condition | Runs | Avg Tool Calls | Recognition Rate | Avg Time (s) |\n")
		sb.WriteString("|-----------|------|----------------|------------------|---------------|\n")

		for _, id := range conditionIDs {
			condRuns := byCondition[id]
			if len(condRuns) == 0 {
				sb.WriteString(fmt.Sprintf("| %s | 0 | - | - | - |\n", id))
				continue
			}

			var totalToolCalls, recognizedCount, totalTime int
			for _, r := range condRuns {
				totalToolCalls += r.Metrics.ToolCallsBeforeRecognition
				if r.Metrics.RecognizedExistingAnswer {
					recognizedCount++
				}
				totalTime += r.Metrics.TimeToRecognitionSeconds
			}

			avgToolCalls := float64(totalToolCalls) / float64(len(condRuns))
			recognitionRate := float64(recognizedCount) / float64(len(condRuns)) * 100
			avgTime := float64(totalTime) / float64(len(condRuns))

			sb.WriteString(fmt.Sprintf("| %s | %d | %.1f | %.0f%% | %.0f |\n",
				id, len(condRuns), avgToolCalls, recognitionRate, avgTime))
		}

		sb.WriteString("\n### Individual Runs\n\n")
		for _, id := range conditionIDs {
			condRuns := byCondition[id]
			if len(condRuns) == 0 {
				continue
			}

			sb.WriteString(fmt.Sprintf("#### Condition %s\n\n", id))
			for _, r := range condRuns {
				recognized := "No"
				if r.Metrics.RecognizedExistingAnswer {
					recognized = "Yes"
				}
				sb.WriteString(fmt.Sprintf("- **Run %d**: Tool calls=%d, Recognized=%s, Time=%ds",
					r.RunNumber, r.Metrics.ToolCallsBeforeRecognition, recognized, r.Metrics.TimeToRecognitionSeconds))
				if r.Notes != "" {
					sb.WriteString(fmt.Sprintf(" | Notes: %s", r.Notes))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("## Observations\n\n")
	sb.WriteString("*Add qualitative observations here.*\n\n")

	sb.WriteString("## Conclusions\n\n")
	sb.WriteString("*Add conclusions after sufficient data is collected.*\n\n")

	sb.WriteString("## Next Steps\n\n")
	sb.WriteString("*What follow-up experiments or actions are suggested?*\n")

	content := sb.String()
	analysisPath := filepath.Join(expDir, "analysis.md")
	if err := os.WriteFile(analysisPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write analysis.md: %w", err)
	}

	return analysisPath, nil
}
