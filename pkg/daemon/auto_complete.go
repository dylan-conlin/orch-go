package daemon

import (
	"fmt"
	"os/exec"
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
