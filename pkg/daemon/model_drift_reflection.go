// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"

	"github.com/dylan-conlin/orch-go/pkg/modeldrift"
)

// ModelDriftResult is an alias for modeldrift.Result.
type ModelDriftResult = modeldrift.Result

// ModelDriftIssueCreateArgs is an alias for modeldrift.IssueCreateArgs.
type ModelDriftIssueCreateArgs = modeldrift.IssueCreateArgs

// ModelDriftMetadata is an alias for modeldrift.Metadata.
type ModelDriftMetadata = modeldrift.Metadata

// ShouldRunModelDriftReflection returns true if model drift reflection should run.
func (d *Daemon) ShouldRunModelDriftReflection() bool {
	return d.Scheduler.IsDue(TaskModelDriftReflect)
}

// RunPeriodicModelDriftReflection runs model drift reflection analysis if due.
// Returns the result if reflection was run, or nil if it wasn't due.
func (d *Daemon) RunPeriodicModelDriftReflection() *ModelDriftResult {
	if !d.ShouldRunModelDriftReflection() {
		return nil
	}

	result, err := d.RunModelDriftReflection()
	if err != nil {
		if result == nil {
			return &ModelDriftResult{
				Error:   err,
				Message: fmt.Sprintf("Model drift reflection failed: %v", err),
			}
		}
		return result
	}

	if result != nil && result.Error == nil {
		d.Scheduler.MarkRun(TaskModelDriftReflect)
	}

	return result
}

// RunModelDriftReflection scans staleness events and creates model-maintenance issues.
func (d *Daemon) RunModelDriftReflection() (*ModelDriftResult, error) {
	store := d.ModelDrift
	if store == nil {
		store = modeldrift.NewDefaultStore()
	}
	querier := &modelDriftIssueAdapter{q: d.resolveIssueQuerier()}
	return modeldrift.Analyze(store, querier)
}

// modelDriftIssueAdapter adapts daemon.IssueQuerier to modeldrift.IssueQuerier.
type modelDriftIssueAdapter struct {
	q IssueQuerier
}

func (a *modelDriftIssueAdapter) ListIssuesWithLabel(label string) ([]modeldrift.Issue, error) {
	issues, err := a.q.ListIssuesWithLabel(label)
	if err != nil {
		return nil, err
	}
	result := make([]modeldrift.Issue, len(issues))
	for i, iss := range issues {
		result[i] = modeldrift.Issue{Title: iss.Title, Description: iss.Description}
	}
	return result, nil
}
