package daemon

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	// LabelEffortSmall marks light-tier work eligible for auto-completion
	// without explain-back or verified gates.
	LabelEffortSmall = "effort:small"

	// LabelEffortMedium marks medium-tier work (default behavior — tag ready-review).
	LabelEffortMedium = "effort:medium"

	// LabelEffortLarge marks heavy-tier work requiring full gates.
	LabelEffortLarge = "effort:large"
)

// AutoCompleter runs the full completion pipeline for an agent.
// This is used by the daemon to auto-complete auto-tier agents
// by shelling out to `orch complete`.
type AutoCompleter interface {
	// Complete runs the full completion pipeline for the given beads ID.
	// workdir is the project directory for cross-project operations.
	// Returns an error if the completion pipeline fails (gate failure, escalation, etc.).
	Complete(beadsID, workdir string) error
}

// LightAutoCompleter extends AutoCompleter with light-tier completion support.
// Light-tier completion skips explain-back and verified gates, used for effort:small work.
type LightAutoCompleter interface {
	AutoCompleter
	// CompleteLight runs the completion pipeline with explain-back and verified gates skipped.
	CompleteLight(beadsID, workdir string) error
}

// OrcCompleter is the production AutoCompleter that shells out to `orch complete`.
type OrcCompleter struct{}

// Complete shells out to `orch complete <beadsID> --force --workdir <workdir>`.
// Uses --force to skip interactive prompts since daemon runs non-interactively.
// The review tier is read from the workspace manifest by orch complete itself.
func (c *OrcCompleter) Complete(beadsID, workdir string) error {
	args := []string{"complete", beadsID, "--force"}
	if workdir != "" {
		args = append(args, "--workdir", workdir)
	}

	cmd := exec.Command("orch", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("orch complete failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// CompleteLight shells out to `orch complete --headless` for light-tier work.
// Headless mode forces review-tier=auto (skipping checkpoint gates like explain-back
// and verified) while preserving other verification gates. Also generates a brief.
func (c *OrcCompleter) CompleteLight(beadsID, workdir string) error {
	args := []string{
		"complete", beadsID,
		"--headless",
	}
	if workdir != "" {
		args = append(args, "--workdir", workdir)
	}

	cmd := exec.Command("orch", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("orch complete (light) failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// HasEffortLabel checks if a set of labels contains a specific effort label.
func HasEffortLabel(labels []string, target string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, target) {
			return true
		}
	}
	return false
}

// HeadlessAutoCompleter extends LightAutoCompleter with headless completion support.
// Headless completion runs `orch complete --headless` to generate a brief without
// interactive gates. Used by the daemon to pre-generate briefs for label-ready-review
// completions so Dylan arrives to finished briefs instead of raw completions.
type HeadlessAutoCompleter interface {
	LightAutoCompleter
	// CompleteHeadless runs the completion pipeline in headless mode.
	// Skips interactive gates and auto-generates a brief to .kb/briefs/.
	// Fire-and-forget: errors are logged but don't block completion labeling.
	CompleteHeadless(beadsID, workdir string) error
}

// CompleteHeadless shells out to `orch complete <beadsID> --headless --workdir <workdir>`.
// Generates a brief without interactive gates (no explain-back, auto review tier).
func (c *OrcCompleter) CompleteHeadless(beadsID, workdir string) error {
	args := []string{"complete", beadsID, "--headless"}
	if workdir != "" {
		args = append(args, "--workdir", workdir)
	}

	cmd := exec.Command("orch", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("orch complete (headless) failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// IsEffortSmall returns true if the labels indicate light-tier work.
func IsEffortSmall(labels []string) bool {
	return HasEffortLabel(labels, LabelEffortSmall)
}
