package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
)

// FrustrationBoundaryAgent represents an agent that reported Phase: Boundary.
type FrustrationBoundaryAgent struct {
	BeadsID      string
	Title        string
	Summary      string
	ArtifactPath string
	OriginalTask string
	AttemptCount int
	ProjectDir   string
	Transitioned bool
}

// FrustrationBoundaryResult summarizes one boundary handling scan.
type FrustrationBoundaryResult struct {
	HandledCount int
	SkippedCount int
	Agents       []FrustrationBoundaryAgent
	Error        error
	Message      string
}

// ShouldRunFrustrationBoundary returns true if boundary handling is due.
func (d *Daemon) ShouldRunFrustrationBoundary() bool {
	return d.Scheduler.IsDue(TaskFrustrationBoundary)
}

// RunPeriodicFrustrationBoundary finds workers in Phase: Boundary and respawns them.
func (d *Daemon) RunPeriodicFrustrationBoundary() *FrustrationBoundaryResult {
	if !d.ShouldRunFrustrationBoundary() {
		return nil
	}

	agentDiscoverer := d.Agents
	if agentDiscoverer == nil {
		agentDiscoverer = &defaultAgentDiscoverer{}
	}

	agents, err := agentDiscoverer.GetActiveAgents()
	if err != nil {
		return &FrustrationBoundaryResult{
			Error:   err,
			Message: fmt.Sprintf("Frustration boundary scan failed to list agents: %v", err),
		}
	}

	projectDir := ""
	if d.ProjectRegistry != nil {
		projectDir = d.ProjectRegistry.CurrentDir()
	}
	if projectDir == "" {
		if cwd, cwdErr := os.Getwd(); cwdErr == nil {
			projectDir = cwd
		}
	}

	if d.boundaryHandled == nil {
		d.boundaryHandled = make(map[string]time.Time)
	}

	handled := 0
	skipped := 0
	processed := make([]FrustrationBoundaryAgent, 0)

	for _, agent := range agents {
		phaseName, summary := splitPhase(agent.Phase)
		if !strings.EqualFold(phaseName, "boundary") {
			skipped++
			continue
		}
		if agent.BeadsID == "" {
			skipped++
			continue
		}
		if _, alreadyHandled := d.boundaryHandled[agent.BeadsID]; alreadyHandled {
			skipped++
			continue
		}

		workspacePath := findWorkspaceForIssue(agent.BeadsID, "", projectDir)
		if workspacePath == "" {
			skipped++
			continue
		}

		comments, _ := verify.GetComments(agent.BeadsID, projectDir)
		originalTask := extractTaskFromSpawnContext(filepath.Join(workspacePath, "SPAWN_CONTEXT.md"))
		attempts := collectBoundaryAttempts(comments)
		artifactPath, artifactErr := writeFrustrationBoundaryArtifact(workspacePath, agent.BeadsID, originalTask, summary, attempts)
		if artifactErr != nil {
			skipped++
			continue
		}

		feedback := buildBoundaryFeedback(originalTask, summary, attempts, artifactPath)
		transitioner := d.BoundaryTransitioner
		if transitioner == nil {
			transitioner = &defaultBoundaryTransitioner{}
		}
		if err := transitioner.Transition(agent.BeadsID, feedback, projectDir); err != nil {
			return &FrustrationBoundaryResult{
				HandledCount: handled,
				SkippedCount: skipped,
				Agents:       processed,
				Error:        err,
				Message:      fmt.Sprintf("Frustration boundary transition failed for %s: %v", agent.BeadsID, err),
			}
		}

		d.boundaryHandled[agent.BeadsID] = time.Now()
		handled++
		processed = append(processed, FrustrationBoundaryAgent{
			BeadsID:      agent.BeadsID,
			Title:        agent.Title,
			Summary:      summary,
			ArtifactPath: artifactPath,
			OriginalTask: originalTask,
			AttemptCount: len(attempts),
			ProjectDir:   projectDir,
			Transitioned: true,
		})
	}

	cleanBoundaryHandled(d.boundaryHandled, agents)
	d.Scheduler.MarkRun(TaskFrustrationBoundary)

	return &FrustrationBoundaryResult{
		HandledCount: handled,
		SkippedCount: skipped,
		Agents:       processed,
		Message:      fmt.Sprintf("Frustration boundary: %d handled, %d skipped", handled, skipped),
	}
}

// TransitionFrustratedWorker abandons the current worker session and respawns via orch rework.
func TransitionFrustratedWorker(beadsID, feedback, workdir string) error {
	abandonArgs := []string{"abandon", beadsID, "--force"}
	if workdir != "" {
		abandonArgs = append(abandonArgs, "--workdir", workdir)
	}
	if output, err := exec.Command("orch", abandonArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("orch abandon failed: %w\nOutput: %s", err, string(output))
	}

	reworkArgs := []string{"rework", beadsID, feedback}
	if workdir != "" {
		reworkArgs = append(reworkArgs, "--workdir", workdir)
	}
	if output, err := exec.Command("orch", reworkArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("orch rework failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func splitPhase(phase string) (string, string) {
	parts := strings.SplitN(phase, " - ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(phase), ""
}

func cleanBoundaryHandled(boundaryHandled map[string]time.Time, agents []ActiveAgent) {
	if boundaryHandled == nil {
		return
	}
	current := make(map[string]bool)
	for _, agent := range agents {
		phaseName, _ := splitPhase(agent.Phase)
		if strings.EqualFold(phaseName, "boundary") {
			current[agent.BeadsID] = true
		}
	}
	for beadsID := range boundaryHandled {
		if !current[beadsID] {
			delete(boundaryHandled, beadsID)
		}
	}
}

func extractTaskFromSpawnContext(spawnContextPath string) string {
	data, err := os.ReadFile(spawnContextPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "TASK:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "TASK:"))
		}
	}
	return ""
}

func collectBoundaryAttempts(comments []verify.Comment) []string {
	attempts := make([]string, 0)
	seen := make(map[string]bool)
	for _, comment := range comments {
		phaseStatus := verify.ParsePhaseFromComments([]verify.Comment{comment})
		if !phaseStatus.Found {
			continue
		}
		phaseName := strings.ToLower(phaseStatus.Phase)
		if phaseName == "planning" || phaseName == "boundary" || phaseName == "complete" {
			continue
		}
		summary := strings.TrimSpace(phaseStatus.Summary)
		if summary == "" || seen[summary] {
			continue
		}
		seen[summary] = true
		attempts = append(attempts, summary)
	}
	return attempts
}

func writeFrustrationBoundaryArtifact(workspacePath, beadsID, task, summary string, attempts []string) (string, error) {
	if task == "" {
		task = beadsID
	}
	attemptLines := []string{"- Investigate the prior phase comments if more context is needed."}
	if len(attempts) > 0 {
		attemptLines = attemptLines[:0]
		for _, attempt := range attempts {
			attemptLines = append(attemptLines, "- "+attempt)
		}
	}
	if summary == "" {
		summary = "Frustration compound detected before meaningful progress was re-established."
	}
	content := fmt.Sprintf(`# Frustration Boundary

**Trigger:** frustration compound detected in headless worker
**Session:** %s
**Duration before boundary:** unknown

## The Question

%s

## What Was Tried

%s

## Why It Didn't Work

%s

## Suggested Fresh Angle

Start from the original question again. Use the failed attempts only as anti-patterns, not as the frame for the next session.

## Do Not Repeat

- Continue from the degraded frame once compound frustration has already triggered.
- Re-run the same thrashing path without first checking the boundary diagnosis.
`, beadsID, task, strings.Join(attemptLines, "\n"), summary)

	artifactPath := filepath.Join(workspacePath, "FRUSTRATION_BOUNDARY.md")
	if err := os.WriteFile(artifactPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write frustration boundary artifact: %w", err)
	}
	return artifactPath, nil
}

func buildBoundaryFeedback(task, summary string, attempts []string, artifactPath string) string {
	parts := []string{"Frustration boundary triggered for the prior worker."}
	if task != "" {
		parts = append(parts, fmt.Sprintf("Original question: %s", task))
	}
	if len(attempts) > 0 {
		parts = append(parts, "What was tried:")
		for _, attempt := range attempts {
			parts = append(parts, "- "+attempt)
		}
	}
	if summary != "" {
		parts = append(parts, fmt.Sprintf("What did not work: %s", summary))
	}
	if artifactPath != "" {
		parts = append(parts, fmt.Sprintf("See %s for the failure-path handoff.", artifactPath))
	}
	parts = append(parts, "Start fresh from the original question instead of continuing the prior frame.")
	return strings.Join(parts, "\n")
}
