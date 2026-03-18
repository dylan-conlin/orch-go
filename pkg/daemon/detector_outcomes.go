// Package daemon provides autonomous overnight processing capabilities.
// detector_outcomes.go computes per-detector outcome rates from beads issue data.
// This closes the feedback loop: detectors create issues, this code measures
// whether those issues led to useful work (completed) or waste (abandoned).
package daemon

const (
	// MinResolvedForPenalty is the minimum number of resolved (completed + abandoned)
	// issues before a detector can be penalized. Prevents overreacting to small samples.
	MinResolvedForPenalty = 5

	// LowUsefulRateThreshold triggers budget halving.
	LowUsefulRateThreshold = 0.3

	// VeryLowUsefulRateThreshold triggers budget disabling.
	VeryLowUsefulRateThreshold = 0.1
)

// DetectorIssue represents a beads issue created by a pattern detector.
type DetectorIssue struct {
	ID       string // beads issue ID
	Detector string // detector name (from daemon:trigger:{name} label)
	Status   string // "open" or "closed"
	Outcome  string // "completed", "abandoned", or "" (still open)
}

// DetectorOutcome holds aggregated outcome metrics for a single detector.
type DetectorOutcome struct {
	Detector      string  `json:"detector"`
	IssuesCreated int     `json:"issues_created"`
	Completed     int     `json:"completed"`
	Abandoned     int     `json:"abandoned"`
	UsefulRate    float64 `json:"useful_rate"`
}

// DetectorOutcomeService provides I/O for querying detector issue outcomes.
type DetectorOutcomeService interface {
	// ListDetectorIssues returns all issues with daemon:trigger labels
	// and their current status/outcome.
	ListDetectorIssues() ([]DetectorIssue, error)
}

// ComputeDetectorOutcomes aggregates per-detector outcome rates from beads data.
// Returns a map of detector name → outcome metrics.
func ComputeDetectorOutcomes(svc DetectorOutcomeService) map[string]*DetectorOutcome {
	issues, err := svc.ListDetectorIssues()
	if err != nil || len(issues) == 0 {
		return map[string]*DetectorOutcome{}
	}

	outcomes := make(map[string]*DetectorOutcome)
	for _, issue := range issues {
		o, ok := outcomes[issue.Detector]
		if !ok {
			o = &DetectorOutcome{Detector: issue.Detector}
			outcomes[issue.Detector] = o
		}
		o.IssuesCreated++

		switch issue.Outcome {
		case "completed":
			o.Completed++
		case "abandoned":
			o.Abandoned++
		}
	}

	// Compute useful rates
	for _, o := range outcomes {
		resolved := o.Completed + o.Abandoned
		if resolved > 0 {
			o.UsefulRate = float64(o.Completed) / float64(resolved)
		}
	}

	return outcomes
}

// AdjustedBudget returns the budget for a detector after outcome-based adjustment.
// Detectors with UsefulRate < 0.3 get halved; < 0.1 get disabled.
// Detectors with insufficient samples or unknown detectors keep full budget.
func AdjustedBudget(baseBudget int, detectorName string, outcomes map[string]*DetectorOutcome) int {
	o, ok := outcomes[detectorName]
	if !ok {
		return baseBudget // unknown detector → no penalty
	}

	resolved := o.Completed + o.Abandoned
	if resolved < MinResolvedForPenalty {
		return baseBudget // insufficient samples → no penalty
	}

	if o.UsefulRate < VeryLowUsefulRateThreshold {
		return 0 // disabled
	}
	if o.UsefulRate < LowUsefulRateThreshold {
		return baseBudget / 2 // halved
	}

	return baseBudget
}
