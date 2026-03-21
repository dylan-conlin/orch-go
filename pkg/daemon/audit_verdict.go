// Package daemon provides autonomous overnight processing capabilities.
// audit_verdict.go handles parsing AUDIT_VERDICT.md files from audit agent
// workspaces and routing verdicts to orch reject or labeling for human review.
package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

const (
	// AuditVerdictFile is the filename audit agents produce with their verdict.
	AuditVerdictFile = "AUDIT_VERDICT.md"

	// LabelAuditDeepReview marks issues selected for random deep audit.
	LabelAuditDeepReview = "audit:deep-review"

	// LabelAuditNeedsReview marks issues where audit confidence is low,
	// requiring human judgment instead of auto-rejection.
	LabelAuditNeedsReview = "audit:needs-review"
)

// AuditVerdict represents a parsed AUDIT_VERDICT.md file.
type AuditVerdict struct {
	Verdict       string // PASS or FAIL
	OriginalIssue string // beads ID of the issue being audited
	Category      string // quality, scope, approach, stale (only meaningful for FAIL)
	Confidence    string // high, medium, low
	Reason        string // explanation
	Evidence      string // file/line references
}

// AuditVerdictResult contains the outcome of processing an audit verdict.
type AuditVerdictResult struct {
	Rejected    bool   // true if orch reject was called
	Passed      bool   // true if verdict was PASS
	NeedsReview bool   // true if low confidence, labeled for human review
	Error       error  // any error during processing
	Action      string // description of action taken
}

// Rejector calls orch reject on a beads issue.
type Rejector interface {
	Reject(beadsID, reason, category, workdir string) error
}

// AuditLabeler adds/removes labels on beads issues for audit processing.
type AuditLabeler interface {
	AddLabel(beadsID, label, workdir string) error
	RemoveLabel(beadsID, label, workdir string) error
}

// OrcRejector is the production Rejector that shells out to `orch reject`.
type OrcRejector struct{}

// Reject shells out to `orch reject <beadsID> "<reason>" --category <category>`.
func (r *OrcRejector) Reject(beadsID, reason, category, workdir string) error {
	args := []string{"reject", beadsID, reason, "--category", category}
	if workdir != "" {
		args = append(args, "--workdir", workdir)
	}
	cmd := exec.Command("orch", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("orch reject failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// BeadsAuditLabeler is the production AuditLabeler backed by bd CLI.
type BeadsAuditLabeler struct{}

func (l *BeadsAuditLabeler) AddLabel(beadsID, label, workdir string) error {
	args := []string{"label", "add", beadsID, label}
	if workdir != "" {
		_, err := runBdCommandInDir(workdir, args...)
		return err
	}
	_, err := runBdCommand(args...)
	return err
}

func (l *BeadsAuditLabeler) RemoveLabel(beadsID, label, workdir string) error {
	args := []string{"label", "remove", beadsID, label}
	if workdir != "" {
		_, err := runBdCommandInDir(workdir, args...)
		return err
	}
	_, err := runBdCommand(args...)
	return err
}

// ParseAuditVerdict parses an AUDIT_VERDICT.md file content.
// The format is simple key-value pairs, one per line:
//
//	verdict: PASS | FAIL
//	original_issue: <beads-id>
//	category: quality | scope | approach | stale
//	confidence: high | medium | low
//	reason: <explanation>
//	evidence: <file/line references>
func ParseAuditVerdict(data []byte) (*AuditVerdict, error) {
	v := &AuditVerdict{}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "verdict":
			v.Verdict = strings.ToUpper(value)
		case "original_issue":
			v.OriginalIssue = value
		case "category":
			v.Category = value
		case "confidence":
			v.Confidence = strings.ToLower(value)
		case "reason":
			v.Reason = value
		case "evidence":
			v.Evidence = value
		}
	}

	// Validate required fields
	if v.Verdict == "" {
		return nil, fmt.Errorf("AUDIT_VERDICT.md missing required field: verdict")
	}
	if v.Verdict != "PASS" && v.Verdict != "FAIL" {
		return nil, fmt.Errorf("AUDIT_VERDICT.md invalid verdict %q: must be PASS or FAIL", v.Verdict)
	}
	if v.OriginalIssue == "" {
		return nil, fmt.Errorf("AUDIT_VERDICT.md missing required field: original_issue")
	}

	return v, nil
}

// ReadAuditVerdictFromWorkspace reads and parses AUDIT_VERDICT.md from a workspace directory.
// Returns nil, nil if no AUDIT_VERDICT.md file exists (not an audit agent).
func ReadAuditVerdictFromWorkspace(workspacePath string) (*AuditVerdict, error) {
	path := filepath.Join(workspacePath, AuditVerdictFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", AuditVerdictFile, err)
	}
	return ParseAuditVerdict(data)
}

// ProcessAuditVerdict routes an audit verdict to the appropriate action:
//   - FAIL + high/medium confidence → orch reject
//   - FAIL + low confidence → label audit:needs-review (human review)
//   - PASS → remove audit:deep-review label
func (d *Daemon) ProcessAuditVerdict(verdict *AuditVerdict, auditBeadsID, workdir string) AuditVerdictResult {
	switch verdict.Verdict {
	case "FAIL":
		if verdict.Confidence == "low" {
			return d.handleLowConfidenceFail(verdict, workdir)
		}
		return d.handleHighConfidenceFail(verdict, auditBeadsID, workdir)

	case "PASS":
		return d.handlePass(verdict, workdir)

	default:
		return AuditVerdictResult{
			Error:  fmt.Errorf("unexpected verdict %q", verdict.Verdict),
			Action: "none",
		}
	}
}

func (d *Daemon) handleHighConfidenceFail(verdict *AuditVerdict, auditBeadsID, workdir string) AuditVerdictResult {
	if d.Rejector == nil {
		return AuditVerdictResult{
			Error:  fmt.Errorf("no Rejector configured, cannot reject %s", verdict.OriginalIssue),
			Action: "reject-skipped",
		}
	}

	reason := fmt.Sprintf("Audit %s: %s", auditBeadsID, verdict.Reason)
	category := verdict.Category
	if category == "" {
		category = "quality"
	}

	if err := d.Rejector.Reject(verdict.OriginalIssue, reason, category, workdir); err != nil {
		return AuditVerdictResult{
			Error:  fmt.Errorf("orch reject failed for %s: %w", verdict.OriginalIssue, err),
			Action: "reject-failed",
		}
	}

	return AuditVerdictResult{
		Rejected: true,
		Action:   fmt.Sprintf("rejected %s (category=%s)", verdict.OriginalIssue, category),
	}
}

func (d *Daemon) handleLowConfidenceFail(verdict *AuditVerdict, workdir string) AuditVerdictResult {
	if d.AuditLabeler != nil {
		if err := d.AuditLabeler.AddLabel(verdict.OriginalIssue, LabelAuditNeedsReview, workdir); err != nil {
			return AuditVerdictResult{
				NeedsReview: true,
				Error:       fmt.Errorf("failed to label %s for review: %w", verdict.OriginalIssue, err),
				Action:      "label-failed",
			}
		}
	}

	return AuditVerdictResult{
		NeedsReview: true,
		Action:      fmt.Sprintf("labeled %s as %s (low confidence)", verdict.OriginalIssue, LabelAuditNeedsReview),
	}
}

func (d *Daemon) handlePass(verdict *AuditVerdict, workdir string) AuditVerdictResult {
	if d.AuditLabeler != nil {
		if err := d.AuditLabeler.RemoveLabel(verdict.OriginalIssue, LabelAuditDeepReview, workdir); err != nil {
			// Non-fatal: the label removal is cleanup, not critical
			fmt.Fprintf(os.Stderr, "Warning: failed to remove %s from %s: %v\n",
				LabelAuditDeepReview, verdict.OriginalIssue, err)
		}
	}

	return AuditVerdictResult{
		Passed: true,
		Action: fmt.Sprintf("passed %s, removed %s", verdict.OriginalIssue, LabelAuditDeepReview),
	}
}

// processAuditVerdictIfPresent checks if a completed agent produced AUDIT_VERDICT.md
// and processes the verdict. Called from CompletionOnce after successful completion processing.
func (d *Daemon) processAuditVerdictIfPresent(agent CompletedAgent, config CompletionConfig, logger *events.Logger) {
	if agent.WorkspacePath == "" {
		return
	}

	verdict, err := ReadAuditVerdictFromWorkspace(agent.WorkspacePath)
	if err != nil {
		if config.Verbose {
			fmt.Printf("    Warning: failed to read audit verdict from %s: %v\n", agent.WorkspacePath, err)
		}
		return
	}
	if verdict == nil {
		return // not an audit agent
	}

	effectiveProjectDir := agent.ProjectDir
	if effectiveProjectDir == "" {
		effectiveProjectDir = config.ProjectDir
	}

	result := d.ProcessAuditVerdict(verdict, agent.BeadsID, effectiveProjectDir)

	// Log the audit outcome
	if result.Rejected {
		if logger != nil {
			_ = logger.Log(events.Event{
				Type:      "audit.failed",
				Timestamp: time.Now().UnixNano(),
				Data: map[string]interface{}{
					"audit_beads_id":    agent.BeadsID,
					"original_beads_id": verdict.OriginalIssue,
					"category":          verdict.Category,
					"reason":            verdict.Reason,
				},
			})
		}
		fmt.Printf("    Audit verdict: FAIL → rejected %s (category=%s)\n", verdict.OriginalIssue, verdict.Category)
	} else if result.Passed {
		if logger != nil {
			_ = logger.Log(events.Event{
				Type:      "audit.passed",
				Timestamp: time.Now().UnixNano(),
				Data: map[string]interface{}{
					"audit_beads_id":    agent.BeadsID,
					"original_beads_id": verdict.OriginalIssue,
				},
			})
		}
		if config.Verbose {
			fmt.Printf("    Audit verdict: PASS for %s\n", verdict.OriginalIssue)
		}
	} else if result.NeedsReview {
		fmt.Printf("    Audit verdict: FAIL (low confidence) → labeled %s for human review\n", verdict.OriginalIssue)
	}

	if result.Error != nil && config.Verbose {
		fmt.Printf("    Warning: audit verdict processing error: %v\n", result.Error)
	}
}
