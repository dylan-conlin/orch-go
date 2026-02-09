package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var verifyBatch bool

var verifyCmd = &cobra.Command{
	Use:   "verify [beads-id]",
	Short: "Run proof-carrying verification specs",
	Long: `Run proof-carrying verification specs from workspace VERIFICATION_SPEC.yaml files.

Batch mode discovers completed workspaces (active + archived), executes executable
entries (cli_smoke, integration, browser, static), marks manual entries pending
unless a human approval token is present, and prints aggregate pass-rate.

Single mode runs one spec by beads ID.

Examples:
  orch verify --batch
  orch verify orch-go-96prt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if verifyBatch {
			if len(args) != 0 {
				return fmt.Errorf("--batch does not accept beads IDs")
			}
			return runVerifyBatch()
		}

		if len(args) != 1 {
			return fmt.Errorf("single mode requires exactly one beads ID (or use --batch)")
		}

		return runVerifySingle(args[0])
	},
}

type verificationTarget struct {
	WorkspacePath string
	WorkspaceName string
	SpecPath      string
	BeadsID       string
	Tier          verify.VerificationTier
	HasSynthesis  bool
	IsArchived    bool
	ModTime       time.Time
}

func init() {
	verifyCmd.Flags().BoolVar(&verifyBatch, "batch", false, "Verify all completed workspaces with VERIFICATION_SPEC.yaml")
}

func runVerifyBatch() error {
	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	targets, err := discoverVerificationTargets(projectDir)
	if err != nil {
		return err
	}
	targets = filterCompletedVerificationTargets(targets)
	if len(targets) == 0 {
		fmt.Println("No completed workspaces with VERIFICATION_SPEC.yaml found")
		return nil
	}

	sort.Slice(targets, func(i, j int) bool {
		if targets[i].ModTime.Equal(targets[j].ModTime) {
			return targets[i].WorkspaceName < targets[j].WorkspaceName
		}
		return targets[i].ModTime.After(targets[j].ModTime)
	})

	commentsMap := verify.GetCommentsBatch(collectTargetBeadsIDs(targets))

	passCount := 0
	failCount := 0
	pendingCount := 0
	skippedCount := 0

	fmt.Printf("Verifying %d completed workspace specs\n\n", len(targets))

	for _, target := range targets {
		result := verify.ExecuteProofSpecInWorkspace(verify.ProofSpecRunnerOptions{
			WorkspacePath:     target.WorkspacePath,
			WorkspaceTier:     target.Tier,
			HasManualApproval: hasManualApprovalToken(target, commentsMap[target.BeadsID]),
		})
		if result.BeadsID == "" {
			result.BeadsID = target.BeadsID
		}

		printProofExecutionResult(target, result)

		switch result.Status {
		case verify.ProofStepStatusPass:
			passCount++
		case verify.ProofStepStatusFail:
			failCount++
		case verify.ProofStepStatusPending:
			pendingCount++
		case verify.ProofStepStatusSkipped:
			skippedCount++
		}
	}

	evaluated := passCount + failCount + pendingCount
	passRate := 0.0
	if evaluated > 0 {
		passRate = float64(passCount) / float64(evaluated) * 100
	}

	fmt.Printf("Aggregate: %d items | pass=%d fail=%d pending=%d skipped=%d | pass-rate=%.1f%% (%d/%d non-skipped)\n",
		len(targets),
		passCount,
		failCount,
		pendingCount,
		skippedCount,
		passRate,
		passCount,
		evaluated,
	)

	if failCount > 0 || pendingCount > 0 {
		return fmt.Errorf("batch verification not fully passing (%d fail, %d pending)", failCount, pendingCount)
	}

	return nil
}

func runVerifySingle(rawBeadsID string) error {
	resolvedBeadsID, err := resolveShortBeadsID(rawBeadsID)
	if err != nil {
		resolvedBeadsID = strings.TrimSpace(rawBeadsID)
	}

	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	targets, err := discoverVerificationTargets(projectDir)
	if err != nil {
		return err
	}

	matching := make([]verificationTarget, 0)
	for _, target := range targets {
		if target.BeadsID == resolvedBeadsID {
			matching = append(matching, target)
		}
	}

	if len(matching) == 0 {
		return fmt.Errorf("no VERIFICATION_SPEC.yaml workspace found for %s", resolvedBeadsID)
	}

	target := selectLatestVerificationTarget(matching)
	if len(matching) > 1 {
		fmt.Printf("Found %d matching workspaces for %s; verifying latest: %s\n\n", len(matching), resolvedBeadsID, target.WorkspaceName)
	}

	comments := []verify.Comment{}
	if target.BeadsID != "" {
		if fetched, fetchErr := verify.GetComments(target.BeadsID); fetchErr == nil {
			comments = fetched
		}
	}

	result := verify.ExecuteProofSpecInWorkspace(verify.ProofSpecRunnerOptions{
		WorkspacePath:     target.WorkspacePath,
		WorkspaceTier:     target.Tier,
		HasManualApproval: hasManualApprovalToken(target, comments),
	})
	if result.BeadsID == "" {
		result.BeadsID = target.BeadsID
	}

	printProofExecutionResult(target, result)

	if result.Status != verify.ProofStepStatusPass {
		return fmt.Errorf("verification status: %s", result.Status)
	}

	return nil
}

func discoverVerificationTargets(projectDir string) ([]verificationTarget, error) {
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")

	active, err := collectVerificationTargetsFromDir(workspaceDir, false)
	if err != nil {
		return nil, err
	}

	archived, err := collectVerificationTargetsFromDir(filepath.Join(workspaceDir, "archived"), true)
	if err != nil {
		return nil, err
	}

	targets := make([]verificationTarget, 0, len(active)+len(archived))
	targets = append(targets, active...)
	targets = append(targets, archived...)

	return targets, nil
}

func collectVerificationTargetsFromDir(dir string, archived bool) ([]verificationTarget, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read workspace directory %s: %w", dir, err)
	}

	targets := make([]verificationTarget, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspacePath := filepath.Join(dir, entry.Name())
		specPath := filepath.Join(workspacePath, verify.VerificationSpecFileName)
		if stat, statErr := os.Stat(specPath); statErr != nil || stat.IsDir() {
			continue
		}

		entryInfo, infoErr := entry.Info()
		modTime := time.Now()
		if infoErr == nil {
			modTime = entryInfo.ModTime()
		}

		targets = append(targets, verificationTarget{
			WorkspacePath: workspacePath,
			WorkspaceName: entry.Name(),
			SpecPath:      specPath,
			BeadsID:       detectTargetBeadsID(workspacePath),
			Tier:          normalizeTargetTier(verify.VerificationTier(verify.ReadTierFromWorkspace(workspacePath))),
			HasSynthesis:  workspaceHasNonEmptySynthesis(workspacePath),
			IsArchived:    archived,
			ModTime:       modTime,
		})
	}

	return targets, nil
}

func filterCompletedVerificationTargets(targets []verificationTarget) []verificationTarget {
	if len(targets) == 0 {
		return nil
	}

	lightIDs := make([]string, 0)
	seenLight := make(map[string]struct{})
	for _, target := range targets {
		if target.Tier != verify.VerificationTierLight || target.BeadsID == "" {
			continue
		}
		if _, exists := seenLight[target.BeadsID]; exists {
			continue
		}
		seenLight[target.BeadsID] = struct{}{}
		lightIDs = append(lightIDs, target.BeadsID)
	}

	commentsMap := verify.GetCommentsBatch(lightIDs)

	filtered := make([]verificationTarget, 0, len(targets))
	for _, target := range targets {
		switch target.Tier {
		case verify.VerificationTierLight:
			if target.BeadsID == "" {
				continue
			}
			phase := verify.ParsePhaseFromComments(commentsMap[target.BeadsID])
			if phase.Found && strings.EqualFold(phase.Phase, "Complete") {
				filtered = append(filtered, target)
			}
		default:
			if target.HasSynthesis {
				filtered = append(filtered, target)
			}
		}
	}

	return filtered
}

func detectTargetBeadsID(workspacePath string) string {
	if data, err := os.ReadFile(filepath.Join(workspacePath, ".beads_id")); err == nil {
		if beadsID := strings.TrimSpace(string(data)); beadsID != "" {
			return beadsID
		}
	}

	if beadsID := extractBeadsIDFromWorkspace(workspacePath); beadsID != "" {
		return beadsID
	}

	if spec, err := verify.LoadProofSpec(workspacePath); err == nil {
		return strings.TrimSpace(spec.Scope.BeadsID)
	}

	return ""
}

func normalizeTargetTier(tier verify.VerificationTier) verify.VerificationTier {
	switch tier {
	case verify.VerificationTierLight, verify.VerificationTierFull, verify.VerificationTierOrchestrator:
		return tier
	default:
		return verify.VerificationTierFull
	}
}

func workspaceHasNonEmptySynthesis(workspacePath string) bool {
	path := filepath.Join(workspacePath, "SYNTHESIS.md")
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Size() > 0
}

func collectTargetBeadsIDs(targets []verificationTarget) []string {
	ids := make([]string, 0, len(targets))
	seen := make(map[string]struct{})
	for _, target := range targets {
		if target.BeadsID == "" {
			continue
		}
		if _, exists := seen[target.BeadsID]; exists {
			continue
		}
		seen[target.BeadsID] = struct{}{}
		ids = append(ids, target.BeadsID)
	}
	return ids
}

func hasManualApprovalToken(target verificationTarget, comments []verify.Comment) bool {
	reviewState, err := verify.LoadReviewState(target.WorkspacePath)
	if err == nil && reviewState.IsApproved() {
		return true
	}

	if len(comments) == 0 && target.BeadsID != "" {
		if fetched, fetchErr := verify.GetComments(target.BeadsID); fetchErr == nil {
			comments = fetched
		}
	}

	hasApproval, _ := verify.HasHumanApproval(comments)
	return hasApproval
}

func selectLatestVerificationTarget(targets []verificationTarget) verificationTarget {
	latest := targets[0]
	for i := 1; i < len(targets); i++ {
		if targets[i].ModTime.After(latest.ModTime) {
			latest = targets[i]
		}
	}
	return latest
}

func printProofExecutionResult(target verificationTarget, result verify.ProofSpecExecutionResult) {
	beadsID := result.BeadsID
	if beadsID == "" {
		beadsID = target.BeadsID
	}

	status := strings.ToUpper(string(result.Status))
	fmt.Printf("[%s] %s", status, target.WorkspaceName)
	if beadsID != "" {
		fmt.Printf(" (%s)", beadsID)
	}
	if target.IsArchived {
		fmt.Print(" [archived]")
	}
	fmt.Println()

	fmt.Printf("  workspace: %s\n", target.WorkspacePath)
	if result.Error != "" {
		fmt.Printf("  error: %s\n", result.Error)
	}

	hash := result.Replay.SpecHash
	if len(hash) > 12 {
		hash = hash[:12]
	}
	if hash == "" {
		hash = "n/a"
	}

	fmt.Printf("  spec hash: %s\n", hash)

	fmt.Printf("  commands run (%d):\n", len(result.Replay.CommandsRun))
	if len(result.Replay.CommandsRun) == 0 {
		fmt.Println("    - none")
	} else {
		for _, cmd := range result.Replay.CommandsRun {
			fmt.Printf("    - %s\n", cmd)
		}
	}

	fmt.Printf("  expectations checked (%d):\n", len(result.Replay.ExpectationsChecked))
	if len(result.Replay.ExpectationsChecked) == 0 {
		fmt.Println("    - none")
	} else {
		for _, expect := range result.Replay.ExpectationsChecked {
			fmt.Printf("    - %s\n", expect)
		}
	}

	fmt.Printf("  failed step IDs: ")
	if len(result.Replay.FailedStepIDs) == 0 {
		fmt.Println("none")
	} else {
		fmt.Println(strings.Join(result.Replay.FailedStepIDs, ", "))
	}

	for _, step := range result.Steps {
		if step.Status == verify.ProofStepStatusPass || step.Status == verify.ProofStepStatusSkipped {
			continue
		}

		reason := step.Error
		if reason == "" && step.Status == verify.ProofStepStatusPending {
			reason = "waiting for human approval token"
		}
		if reason == "" {
			reason = "verification failed"
		}
		fmt.Printf("  step %s [%s]: %s\n", step.ID, step.Status, reason)
	}

	fmt.Println()
}
