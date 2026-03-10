package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
)

func runDaemonDryRun() error {
	config := daemonConfigFromFlags()
	d := daemon.NewWithConfig(config)

	// Initialize project registry for cross-project issue visibility
	if registry, err := daemon.NewProjectRegistry(); err == nil {
		d.ProjectRegistry = registry
	}

	// NOTE: Extraction system disabled. Hotspot checking not configured.
	// To re-enable, uncomment: d.HotspotChecker = daemon.NewGitHotspotChecker()

	// Wire focus-aware priority boost
	wireFocusBoost(d)

	// Seed verification tracker with unverified backlog
	seedVerificationTracker(d)

	result, err := d.Preview()
	if err != nil {
		return fmt.Errorf("preview error: %w", err)
	}

	// Show verification status in dry-run output
	if d.VerificationTracker != nil {
		verifyStatus := d.VerificationTracker.Status()
		if d.VerificationTracker.IsPaused() {
			breakdown := verificationBreakdown()
			fmt.Printf("[DRY-RUN] Verification pause: %d unverified completions, threshold is %d%s\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold, breakdown)
		} else if verifyStatus.IsEnabled() {
			fmt.Printf("[DRY-RUN] Verification check: %d/%d unverified completions\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
		}
	}

	// Get current directory for context
	projectDir, _ := os.Getwd()
	projectName := filepath.Base(projectDir)

	// Show queue summary: spawnable vs rejected counts
	spawnableCount := 0
	if result.Issue != nil {
		spawnableCount = 1
	}
	rejectedCount := len(result.RejectedIssues)
	fmt.Printf("[DRY-RUN] Queue: %d spawnable, %d rejected\n\n", spawnableCount, rejectedCount)

	if result.Issue != nil {
		fmt.Println("Next spawn:")
		fmt.Printf("  Project:  %s\n", projectName)
		fmt.Println(daemon.FormatPreview(result.Issue))
		fmt.Printf("\nInferred skill: %s\n", result.Skill)
		fmt.Printf("Inferred model: %s\n", result.Model)
		if result.ArchitectEscalated {
			fmt.Println("⚠️  Architect escalation: implementation skill escalated to architect (hotspot area)")
		}

		// Display hotspot warnings if any
		if result.HasHotspotWarnings() {
			fmt.Print(daemon.FormatHotspotWarnings(result.HotspotWarnings))
		}
	} else {
		fmt.Println("No spawnable issues in queue")
	}

	// Display rejected issues grouped by reason
	if len(result.RejectedIssues) > 0 {
		fmt.Print(daemon.FormatRejectedIssues(result.RejectedIssues))
	}

	fmt.Println("\nNo agents were spawned (dry-run mode).")

	return nil
}

func runDaemonOnce() error {
	config := daemonConfigFromFlags()
	d := daemon.NewWithConfig(config)

	// Initialize project registry for cross-project issue resolution
	registry, err := daemon.NewProjectRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: project registry unavailable: %v\n", err)
	} else {
		d.ProjectRegistry = registry
	}

	// Wire focus-aware priority boost
	wireFocusBoost(d)

	// Seed verification tracker with unverified backlog
	seedVerificationTracker(d)

	// Show verification status before spawning
	if d.VerificationTracker != nil {
		verifyStatus := d.VerificationTracker.Status()
		if d.VerificationTracker.IsPaused() {
			breakdown := verificationBreakdown()
			fmt.Printf("Verification pause: %d unverified completions, threshold is %d%s\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold, breakdown)
			fmt.Println("  Run 'orch daemon resume' after reviewing completed work to continue")
		} else if verifyStatus.IsEnabled() {
			fmt.Printf("Verification check: %d/%d unverified completions, proceeding\n",
				verifyStatus.CompletionsSinceVerification, verifyStatus.Threshold)
		}
	}

	result, err := d.Once()
	if err != nil {
		return fmt.Errorf("daemon error: %w", err)
	}

	if !result.Processed {
		fmt.Println(result.Message)
		return nil
	}

	fmt.Printf("Spawned: %s\n", result.Issue.ID)
	fmt.Printf("  Title:  %s\n", result.Issue.Title)
	fmt.Printf("  Type:   %s\n", result.Issue.IssueType)
	fmt.Printf("  Skill:  %s\n", result.Skill)
	fmt.Printf("  Model:  %s\n", result.Model)

	// Log the spawn
	logger := events.NewLogger(events.DefaultLogPath())
	event := events.Event{
		Type:      "daemon.once",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"beads_id": result.Issue.ID,
			"skill":    result.Skill,
			"title":    result.Issue.Title,
		},
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log event: %v\n", err)
	}

	return nil
}

func runDaemonPreview() error {
	config := daemonConfigFromFlags()
	d := daemon.NewWithConfig(config)

	// Initialize project registry for cross-project issue visibility
	if registry, err := daemon.NewProjectRegistry(); err == nil {
		d.ProjectRegistry = registry
	}

	// NOTE: Extraction system disabled. Hotspot checking not configured.
	// To re-enable, uncomment: d.HotspotChecker = daemon.NewGitHotspotChecker()

	// Wire focus-aware priority boost
	wireFocusBoost(d)

	// Seed verification tracker with unverified backlog
	seedVerificationTracker(d)

	result, err := d.Preview()
	if err != nil {
		return fmt.Errorf("preview error: %w", err)
	}

	// Get current directory for context
	projectDir, _ := os.Getwd()
	projectName := filepath.Base(projectDir)

	// Show queue summary: spawnable vs rejected counts
	spawnableCount := 0
	if result.Issue != nil {
		spawnableCount = 1
	}
	rejectedCount := len(result.RejectedIssues)
	fmt.Printf("Queue: %d spawnable, %d rejected\n\n", spawnableCount, rejectedCount)

	// Display focus status
	if result.FocusGoal != "" {
		fmt.Printf("Focus: %s\n", result.FocusGoal)
		if result.FocusBoosted {
			fmt.Println("  (priority boost applied to next issue)")
		}
		fmt.Println()
	}

	// Display spawnable issue if available
	if result.Issue != nil {
		fmt.Println("Next spawn:")
		fmt.Printf("  Project:  %s\n", projectName)
		fmt.Println(daemon.FormatPreview(result.Issue))
		fmt.Printf("\nInferred skill: %s\n", result.Skill)
		fmt.Printf("Inferred model: %s\n", result.Model)

		// Display hotspot warnings if any
		if result.HasHotspotWarnings() {
			fmt.Print(daemon.FormatHotspotWarnings(result.HotspotWarnings))
		}
	} else {
		fmt.Println(result.Message)
	}

	// Display rejected issues grouped by reason
	if len(result.RejectedIssues) > 0 {
		fmt.Print(daemon.FormatRejectedIssues(result.RejectedIssues))
	}

	if result.Issue != nil {
		fmt.Println("\nRun 'orch-go daemon once' to process this issue.")
	}

	return nil
}

func runDaemonReflect() error {
	fmt.Println("Running knowledge reflection analysis...")

	result := daemon.RunAndSaveReflection()
	if result.Error != nil {
		return fmt.Errorf("reflection error: %w", result.Error)
	}

	if result.Suggestions == nil || !result.Suggestions.HasSuggestions() {
		fmt.Println("No reflection suggestions found.")
		return nil
	}

	// Print summary
	fmt.Printf("\n%s\n", result.Suggestions.Summary())

	// Print details by category
	if len(result.Suggestions.Synthesis) > 0 {
		fmt.Printf("\nSynthesis Opportunities (%d):\n", len(result.Suggestions.Synthesis))
		for _, s := range result.Suggestions.Synthesis[:min(5, len(result.Suggestions.Synthesis))] {
			fmt.Printf("  - %s: %d investigations\n", s.Topic, s.Count)
		}
		if len(result.Suggestions.Synthesis) > 5 {
			fmt.Printf("  ... and %d more\n", len(result.Suggestions.Synthesis)-5)
		}
	}

	if len(result.Suggestions.Promote) > 0 {
		fmt.Printf("\nPromotion Candidates (%d):\n", len(result.Suggestions.Promote))
		for _, p := range result.Suggestions.Promote[:min(5, len(result.Suggestions.Promote))] {
			fmt.Printf("  - %s\n", truncateDaemonString(p.Content, 60))
		}
	}

	if len(result.Suggestions.Stale) > 0 {
		fmt.Printf("\nStale Decisions (%d):\n", len(result.Suggestions.Stale))
		for _, s := range result.Suggestions.Stale[:min(5, len(result.Suggestions.Stale))] {
			fmt.Printf("  - %s (%d days old)\n", filepath.Base(s.Path), s.Age)
		}
	}

	if len(result.Suggestions.Drift) > 0 {
		fmt.Printf("\nPotential Drifts (%d):\n", len(result.Suggestions.Drift))
		for _, d := range result.Suggestions.Drift[:min(5, len(result.Suggestions.Drift))] {
			fmt.Printf("  - %s\n", truncateDaemonString(d.Content, 60))
		}
	}

	if result.Saved {
		fmt.Printf("\nSuggestions saved to: %s\n", daemon.SuggestionsPath())
		fmt.Println("They will be surfaced at next session start.")
	}

	return nil
}

// runReflectionAnalysis runs kb reflect and saves suggestions.
// Called at the end of daemon processing to update reflection suggestions.
func runReflectionAnalysis(verbose bool) {
	if verbose {
		fmt.Println("Running reflection analysis...")
	}

	result := daemon.RunAndSaveReflection()
	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "Warning: reflection analysis failed: %v\n", result.Error)
		return
	}

	if result.Suggestions == nil || !result.Suggestions.HasSuggestions() {
		if verbose {
			fmt.Println("No reflection suggestions found.")
		}
		return
	}

	fmt.Printf("Reflection: %s\n", result.Suggestions.Summary())
	if result.Saved {
		if verbose {
			fmt.Printf("Suggestions saved to: %s\n", daemon.SuggestionsPath())
		}
	}
}

func runDaemonStatus() error {
	info := daemon.GetStatusInfo()

	// Clean up stale status file if detected
	if info.StaleFile {
		daemon.RemoveStatusFile()
	}

	fmt.Print(daemon.FormatStatusInfo(info))
	return nil
}

func runDaemonStop() error {
	pid := daemon.ReadPIDFromLockFile()
	if pid > 0 {
		fmt.Printf("Stopping daemon (PID %d)...\n", pid)
	} else {
		fmt.Println("Stopping daemon...")
	}

	err := daemon.StopDaemon(daemon.StopOptions{})
	if err == daemon.ErrNoDaemonRunning {
		fmt.Println("No daemon is currently running.")
		return nil
	}
	if err == daemon.ErrStopTimeout {
		return fmt.Errorf("daemon (PID %d) did not stop within timeout - it may need to be killed manually", pid)
	}
	if err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	fmt.Println("Daemon stopped.")
	return nil
}

func runDaemonRestart() error {
	// Try to stop existing daemon first (ignore "not running" error)
	pid := daemon.ReadPIDFromLockFile()
	if pid > 0 && daemon.IsProcessAlive(pid) {
		fmt.Printf("Stopping existing daemon (PID %d)...\n", pid)
		err := daemon.StopDaemon(daemon.StopOptions{})
		if err != nil && err != daemon.ErrNoDaemonRunning {
			return fmt.Errorf("failed to stop existing daemon: %w", err)
		}
		fmt.Println("Daemon stopped.")
	}

	fmt.Println("Starting new daemon...")
	return runDaemonLoop()
}

func runDaemonResume() error {
	fmt.Println("Sending resume signal to daemon...")

	if err := daemon.WriteResumeSignal(); err != nil {
		return fmt.Errorf("failed to write resume signal: %w", err)
	}

	fmt.Println("Resume signal sent")
	fmt.Println()
	fmt.Println("The daemon will detect the signal on the next poll cycle and resume operation.")
	fmt.Println("The verification counter will be reset, allowing the daemon to continue spawning.")

	return nil
}

// runDaemonCleanStale finds and optionally closes orphaned cross-project completions.
// These are issues from other projects that have Phase: Complete + daemon:ready-review
// but will never be resolved via orch complete (project merged/archived).
func runDaemonCleanStale(closeStale bool) error {
	registry, err := daemon.NewProjectRegistry()
	if err != nil {
		return fmt.Errorf("failed to load project registry: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	type staleIssue struct {
		ID         string
		Title      string
		ProjectDir string
		Project    string
	}

	var stale []staleIssue

	for _, proj := range registry.Projects() {
		issues, err := daemon.ListIssuesWithLabelForProject("daemon:ready-review", proj.Dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan %s: %v\n", proj.Dir, err)
			continue
		}

		for _, issue := range issues {
			// Only flag cross-project issues (not the current project)
			if proj.Dir == cwd {
				continue
			}
			stale = append(stale, staleIssue{
				ID:         issue.ID,
				Title:      issue.Title,
				ProjectDir: proj.Dir,
				Project:    proj.Prefix,
			})
		}
	}

	if len(stale) == 0 {
		fmt.Println("No stale cross-project completions found.")
		return nil
	}

	fmt.Printf("Found %d stale cross-project completion(s):\n\n", len(stale))
	for _, s := range stale {
		fmt.Printf("  %s: %s\n    Project: %s (%s)\n", s.ID, s.Title, s.Project, s.ProjectDir)
	}
	fmt.Println()

	if !closeStale {
		fmt.Println("Run with --close to close these issues.")
		return nil
	}

	closed := 0
	for _, s := range stale {
		if err := daemon.CloseIssueForProject(s.ID, s.ProjectDir, "Closed by orch daemon clean-stale: orphaned cross-project completion"); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to close %s: %v\n", s.ID, err)
			continue
		}
		fmt.Printf("  Closed: %s\n", s.ID)
		closed++
	}

	fmt.Printf("\nClosed %d/%d stale completions.\n", closed, len(stale))

	// Also send resume signal to unblock daemon
	if closed > 0 {
		if err := daemon.WriteResumeSignal(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to send resume signal: %v\n", err)
		} else {
			fmt.Println("Resume signal sent to daemon.")
		}
	}

	return nil
}
