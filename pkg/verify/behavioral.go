// Package verify provides verification helpers for agent completion.
package verify

import (
	"regexp"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

// BehavioralValidationType indicates the type of behavioral validation suggested.
type BehavioralValidationType string

const (
	// BehavioralTypeUI indicates UI/visual changes that should be verified in browser.
	BehavioralTypeUI BehavioralValidationType = "ui"
	// BehavioralTypeAPI indicates API/endpoint changes that should be tested.
	BehavioralTypeAPI BehavioralValidationType = "api"
	// BehavioralTypeConcurrency indicates concurrency changes (locks, async) that need verification.
	BehavioralTypeConcurrency BehavioralValidationType = "concurrency"
	// BehavioralTypeIntegration indicates external integration changes (Redis, databases, APIs).
	BehavioralTypeIntegration BehavioralValidationType = "integration"
	// BehavioralTypeCLI indicates CLI command changes that should be tested.
	BehavioralTypeCLI BehavioralValidationType = "cli"
)

// BehavioralValidationResult contains structured information about behavioral validation.
// This is informational output, not a blocking gate.
type BehavioralValidationResult struct {
	// BehavioralValidationSuggested is true if the changes appear to be behavior-changing
	// and would benefit from manual behavioral validation.
	BehavioralValidationSuggested bool `json:"behavioral_validation_suggested"`

	// ValidationType indicates what type of behavioral change was detected.
	ValidationType BehavioralValidationType `json:"validation_type,omitempty"`

	// SuggestedURL is the URL to test (for UI changes).
	SuggestedURL string `json:"suggested_url,omitempty"`

	// SuggestedSteps are the recommended validation steps.
	SuggestedSteps []string `json:"suggested_steps,omitempty"`

	// HasBehavioralEvidence indicates if the agent already provided behavioral evidence.
	HasBehavioralEvidence bool `json:"has_behavioral_evidence"`

	// Evidence contains matched behavioral evidence patterns.
	Evidence []string `json:"evidence,omitempty"`

	// TriggerReason explains why behavioral validation was suggested.
	TriggerReason string `json:"trigger_reason,omitempty"`

	// ChangedFiles are the files that triggered the suggestion.
	ChangedFiles []string `json:"changed_files,omitempty"`
}

// behavioralEvidencePatterns match actual behavior demonstration in comments.
// These patterns indicate the agent observed the behavior working, not just tests passing.
var behavioralEvidencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)behavior\s+verified:\s*.+`),          // "Behavior verified: X → Y"
	regexp.MustCompile(`(?i)ran\s+locally:\s*.+→.+`),             // "Ran locally: X → Y"
	regexp.MustCompile(`(?i)observed\s+expected\s+behavior`),     // "Observed expected behavior"
	regexp.MustCompile(`(?i)demonstrated:\s*.+working`),          // "Demonstrated: X working"
	regexp.MustCompile(`(?i)manual\s+test:\s*.+passed`),          // "Manual test: X passed"
	regexp.MustCompile(`(?i)verified\s+in\s+browser`),            // "Verified in browser"
	regexp.MustCompile(`(?i)tested\s+endpoint:\s*.+`),            // "Tested endpoint: /api/..."
	regexp.MustCompile(`(?i)curl\s+test:\s*.+`),                  // "curl test: ..."
	regexp.MustCompile(`(?i)confirmed\s+behavior:\s*.+`),         // "Confirmed behavior: ..."
	regexp.MustCompile(`(?i)visual\s+verification:\s*.+`),        // "Visual verification: ..."
	regexp.MustCompile(`(?i)smoke\s+test:\s*.+`),                 // "Smoke test: ..."
	regexp.MustCompile(`(?i)e2e\s+(test|verification):\s*.+`),    // "E2E test: ..."
	regexp.MustCompile(`(?i)integration\s+test:\s*.+working`),    // "Integration test: X working"
	regexp.MustCompile(`(?i)lock\s+(acquired|released|works)`),   // Concurrency verification
	regexp.MustCompile(`(?i)redis\s+(connected|working|tested)`), // Redis verification
}

// File path patterns that indicate behavior-changing work.
var behaviorChangeFilePatterns = struct {
	UI          []string
	API         []string
	Concurrency []string
	Integration []string
	CLI         []string
}{
	UI: []string{
		"web/", "src/components/", "src/pages/", "src/views/",
		"frontend/", "client/", "ui/", "public/",
		".svelte", ".vue", ".jsx", ".tsx",
	},
	API: []string{
		"routes/", "api/", "handlers/", "controllers/",
		"endpoints/", "server/", "http/",
	},
	Concurrency: []string{
		"lock", "mutex", "sync", "async", "concurrent",
		"goroutine", "channel", "worker", "pool",
	},
	Integration: []string{
		"redis", "database", "db/", "postgres", "mysql", "mongo",
		"kafka", "rabbitmq", "queue/", "pubsub", "webhook",
		"external/", "client/", "sdk/",
	},
	CLI: []string{
		"cmd/", "cli/", "command", "flag",
	},
}

// Commit message patterns that indicate behavior changes.
var behaviorChangeCommitPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(feat|fix|refactor)(\(.+\))?:`), // Semantic commits that change behavior
	regexp.MustCompile(`(?i)add(ed|s)?\s+(new\s+)?(endpoint|api|route)`),
	regexp.MustCompile(`(?i)(implement|add)(ed|s)?\s+.*lock`), // Match "implement X lock" or "add lock"
	regexp.MustCompile(`(?i)connect(s|ed)?\s+to\s+(redis|database)`),
	regexp.MustCompile(`(?i)ui\s+(change|update|fix)`),
}

// HasBehavioralEvidence checks beads comments for evidence of behavioral verification.
// Returns true if any comment contains actual behavioral demonstration patterns.
func HasBehavioralEvidence(comments []Comment) (bool, []string) {
	var evidence []string

	for _, comment := range comments {
		for _, pattern := range behavioralEvidencePatterns {
			if pattern.MatchString(comment.Text) {
				matches := pattern.FindString(comment.Text)
				if matches != "" {
					evidence = append(evidence, matches)
				}
			}
		}
	}

	return len(evidence) > 0, evidence
}

// DetectBehaviorChangeType analyzes changed files and returns the type of behavioral change.
// Returns empty string if no behavior-changing patterns detected.
func DetectBehaviorChangeType(files []string) (BehavioralValidationType, []string) {
	var matchedFiles []string

	// Check UI patterns first (highest priority for visual verification)
	for _, file := range files {
		for _, pattern := range behaviorChangeFilePatterns.UI {
			if strings.Contains(strings.ToLower(file), strings.ToLower(pattern)) {
				matchedFiles = append(matchedFiles, file)
			}
		}
	}
	if len(matchedFiles) > 0 {
		return BehavioralTypeUI, matchedFiles
	}

	// Check API patterns
	for _, file := range files {
		for _, pattern := range behaviorChangeFilePatterns.API {
			if strings.Contains(strings.ToLower(file), strings.ToLower(pattern)) {
				matchedFiles = append(matchedFiles, file)
			}
		}
	}
	if len(matchedFiles) > 0 {
		return BehavioralTypeAPI, matchedFiles
	}

	// Check concurrency patterns
	for _, file := range files {
		for _, pattern := range behaviorChangeFilePatterns.Concurrency {
			if strings.Contains(strings.ToLower(file), strings.ToLower(pattern)) {
				matchedFiles = append(matchedFiles, file)
			}
		}
	}
	if len(matchedFiles) > 0 {
		return BehavioralTypeConcurrency, matchedFiles
	}

	// Check integration patterns
	for _, file := range files {
		for _, pattern := range behaviorChangeFilePatterns.Integration {
			if strings.Contains(strings.ToLower(file), strings.ToLower(pattern)) {
				matchedFiles = append(matchedFiles, file)
			}
		}
	}
	if len(matchedFiles) > 0 {
		return BehavioralTypeIntegration, matchedFiles
	}

	// Check CLI patterns
	for _, file := range files {
		for _, pattern := range behaviorChangeFilePatterns.CLI {
			if strings.Contains(strings.ToLower(file), strings.ToLower(pattern)) {
				matchedFiles = append(matchedFiles, file)
			}
		}
	}
	if len(matchedFiles) > 0 {
		return BehavioralTypeCLI, matchedFiles
	}

	return "", nil
}

// HasBehaviorChangeCommitPattern checks if any commit message indicates behavior changes.
func HasBehaviorChangeCommitPattern(commitMessages []string) bool {
	for _, msg := range commitMessages {
		for _, pattern := range behaviorChangeCommitPatterns {
			if pattern.MatchString(msg) {
				return true
			}
		}
	}
	return false
}

// GetSuggestedValidationSteps returns validation steps based on the change type.
func GetSuggestedValidationSteps(validationType BehavioralValidationType) []string {
	switch validationType {
	case BehavioralTypeUI:
		return []string{
			"Open the application in browser",
			"Navigate to the changed component/page",
			"Verify the visual changes render correctly",
			"Test interactive elements (buttons, forms, etc.)",
		}
	case BehavioralTypeAPI:
		return []string{
			"Start the server locally",
			"Test the endpoint with curl or API client",
			"Verify request/response format",
			"Test error cases",
		}
	case BehavioralTypeConcurrency:
		return []string{
			"Run the code with multiple concurrent requests",
			"Verify locks acquire and release correctly",
			"Check for race conditions",
			"Test timeout/retry behavior",
		}
	case BehavioralTypeIntegration:
		return []string{
			"Ensure external service is running (Redis, database, etc.)",
			"Test connection establishment",
			"Verify data flow between systems",
			"Test error handling and reconnection",
		}
	case BehavioralTypeCLI:
		return []string{
			"Run the CLI command with typical arguments",
			"Verify the output format",
			"Test error handling",
			"Check help text and flag parsing",
		}
	default:
		return nil
	}
}

// GetSuggestedURL returns a suggested URL for UI validation based on project config.
func GetSuggestedURL(projectDir string, validationType BehavioralValidationType) string {
	if validationType != BehavioralTypeUI {
		return ""
	}

	// Try to load project config for server ports
	cfg, err := config.Load(projectDir)
	if err != nil {
		return ""
	}

	// Check for web server port
	if port, ok := cfg.GetServerPort("web"); ok {
		return "http://localhost:" + itoa(port)
	}

	// Check for dev server port
	if port, ok := cfg.GetServerPort("dev"); ok {
		return "http://localhost:" + itoa(port)
	}

	// Check for frontend port
	if port, ok := cfg.GetServerPort("frontend"); ok {
		return "http://localhost:" + itoa(port)
	}

	return ""
}

// getChangedFilesSinceSpawnForBehavioral gets changed files for behavioral analysis.
// Uses workspace-filtered commits to only consider THIS agent's changes.
func getChangedFilesSinceSpawnForBehavioral(projectDir string, spawnTime time.Time, workspacePath string) []string {
	if spawnTime.IsZero() || projectDir == "" {
		return getRecentChangedFiles(projectDir)
	}

	// Use the existing function from test_evidence.go if available
	return getChangedFilesSinceSpawn(projectDir, spawnTime, workspacePath)
}

// getRecentChangedFiles gets files changed in recent commits (fallback).
func getRecentChangedFiles(projectDir string) []string {
	files, err := getChangedFiles(projectDir, "")
	if err != nil {
		return nil
	}

	return files
}

// getRecentCommitMessages gets commit messages from recent commits.
func getRecentCommitMessages(projectDir string, spawnTime time.Time) []string {
	var output string
	var err error
	if !spawnTime.IsZero() {
		sinceStr := spawnTime.Format(time.RFC3339)
		output, err = runGitOutput(projectDir, "log", "--since="+sinceStr, "--format=%s")
	} else {
		output, err = runGitOutput(projectDir, "log", "-5", "--format=%s")
	}

	if err != nil {
		return nil
	}

	var messages []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			messages = append(messages, line)
		}
	}
	return messages
}

// CheckBehavioralValidation checks if behavioral validation should be suggested.
// This is an informational check, not a blocking gate.
func CheckBehavioralValidation(beadsID, workspacePath, projectDir string, comments []Comment) *BehavioralValidationResult {
	return CheckBehavioralValidationWithComments(beadsID, workspacePath, projectDir, comments)
}

// CheckBehavioralValidationWithComments analyzes changes and determines if behavioral validation is suggested.
func CheckBehavioralValidationWithComments(beadsID, workspacePath, projectDir string, comments []Comment) *BehavioralValidationResult {
	result := &BehavioralValidationResult{}

	// Short-circuit if no project directory provided
	if projectDir == "" {
		return result
	}

	// Get spawn time for accurate change detection
	spawnTime := spawn.ReadSpawnTime(workspacePath)

	// Get changed files
	changedFiles := getChangedFilesSinceSpawnForBehavioral(projectDir, spawnTime, workspacePath)
	if len(changedFiles) == 0 {
		return result // No changes, no suggestion
	}

	// Detect behavior change type from files
	validationType, matchedFiles := DetectBehaviorChangeType(changedFiles)

	// Also check commit messages for behavior indicators
	commitMessages := getRecentCommitMessages(projectDir, spawnTime)
	hasBehaviorCommit := HasBehaviorChangeCommitPattern(commitMessages)

	// If no behavior-changing patterns detected, return empty result
	if validationType == "" && !hasBehaviorCommit {
		return result
	}

	// Behavior change detected - check if evidence exists
	hasEvidence, evidence := HasBehavioralEvidence(comments)

	// Build result
	result.BehavioralValidationSuggested = true
	result.ValidationType = validationType
	result.HasBehavioralEvidence = hasEvidence
	result.Evidence = evidence
	result.ChangedFiles = matchedFiles

	// Set trigger reason
	if validationType != "" {
		result.TriggerReason = "File changes detected: " + string(validationType)
	} else if hasBehaviorCommit {
		result.TriggerReason = "Commit message indicates behavior change"
	}

	// Get suggested steps and URL
	result.SuggestedSteps = GetSuggestedValidationSteps(validationType)
	result.SuggestedURL = GetSuggestedURL(projectDir, validationType)

	return result
}

// CheckBehavioralValidationForCompletion is a convenience function for use in VerifyCompletionFull.
// Returns nil if no behavioral validation is suggested.
func CheckBehavioralValidationForCompletion(beadsID, workspacePath, projectDir string, comments []Comment) *BehavioralValidationResult {
	result := CheckBehavioralValidationWithComments(beadsID, workspacePath, projectDir, comments)

	// Return nil if not suggested (to match pattern of other completion checks)
	if !result.BehavioralValidationSuggested {
		return nil
	}

	return result
}
