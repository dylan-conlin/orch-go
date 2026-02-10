// Package main provides the healthcheck command for validating the spawn pipeline.
// The healthcheck spawns a trivial agent, polls its status, verifies it committed,
// then cleans up the workspace and branch.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/spf13/cobra"
)

const (
	// healthcheckWorkspacePrefix is the distinctive prefix for healthcheck workspaces.
	healthcheckWorkspacePrefix = "og-healthcheck-"

	// healthcheckDefaultTimeout is the maximum time to wait for the agent pipeline.
	healthcheckDefaultTimeout = 90 * time.Second

	// healthcheckModelTimeout is how long to wait for tokens > 0 (model responding).
	healthcheckModelTimeout = 30 * time.Second

	// healthcheckPollInterval is how often to check agent status.
	healthcheckPollInterval = 5 * time.Second

	// healthcheckTask is the prompt given to the canary agent.
	healthcheckTask = "Healthcheck: echo the current UTC timestamp to stdout, then exit immediately with /exit. Do NOT create or modify any files."
)

var (
	healthcheckTimeout time.Duration
	healthcheckModel   string
	healthcheckVerbose bool
)

var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Validate the spawn pipeline with a canary agent",
	Long: `Spawn a trivial agent to validate the full spawn pipeline is working.

The healthcheck:
1. Spawns a minimal headless agent (sonnet, --no-track, --light)
2. Polls agent status every 5s for up to 90s
3. Checks: tokens > 0 within 30s (model responding)
4. Checks: SPAWN_CONTEXT.md delivered to worktree
5. Cleans up: removes worktree + branch after validation
6. Reports pass/fail with diagnostics on failure

Exit codes:
  0 - Pipeline healthy (agent spawned, model responded)
  1 - Pipeline unhealthy (with diagnostic message)

Examples:
  orch healthcheck                  # Default check
  orch healthcheck --timeout 120s   # Custom timeout
  orch healthcheck --model opus     # Test specific model
  orch healthcheck --verbose        # Show polling details`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHealthcheck()
	},
}

func init() {
	healthcheckCmd.Flags().DurationVar(&healthcheckTimeout, "timeout", healthcheckDefaultTimeout, "Maximum time to wait for pipeline validation")
	healthcheckCmd.Flags().StringVar(&healthcheckModel, "model", "sonnet", "Model to test (default: sonnet)")
	healthcheckCmd.Flags().BoolVar(&healthcheckVerbose, "verbose", false, "Show detailed polling output")
}

// HealthcheckResult contains the outcome of a healthcheck run.
type HealthcheckResult struct {
	// Pass indicates whether the healthcheck succeeded.
	Pass bool
	// Message is a human-readable summary.
	Message string
	// Diagnostic provides details on failure.
	Diagnostic string
	// SessionID is the OpenCode session ID (if created).
	SessionID string
	// WorkspaceName is the workspace name used.
	WorkspaceName string
	// TokensObserved is the total tokens seen during polling.
	TokensObserved int
	// Elapsed is the total time taken.
	Elapsed time.Duration
}

// HealthcheckDeps holds injectable dependencies for testing.
type HealthcheckDeps struct {
	// Client is the OpenCode API client.
	Client opencode.ClientInterface
	// ProjectDir is the project root directory.
	ProjectDir string
	// ServerURL is the OpenCode server URL.
	ServerURL string
	// Now returns the current time.
	Now func() time.Time
	// Sleep pauses execution for the given duration.
	Sleep func(time.Duration)
	// SpawnHeadless spawns an agent and returns the session ID.
	SpawnHeadless func(client opencode.ClientInterface, serverURL, title, prompt, model, variant, runtimeDir string) (string, error)
	// Verbose controls detailed output.
	Verbose bool
}

// DefaultHealthcheckDeps creates HealthcheckDeps with production implementations.
func DefaultHealthcheckDeps(serverURL, projectDir string) *HealthcheckDeps {
	return &HealthcheckDeps{
		Client:     opencode.NewClient(serverURL),
		ProjectDir: projectDir,
		ServerURL:  serverURL,
		Now:        time.Now,
		Sleep:      time.Sleep,
		SpawnHeadless: func(client opencode.ClientInterface, serverURL, title, prompt, model, variant, runtimeDir string) (string, error) {
			resp, err := client.CreateSession(title, runtimeDir, model, variant, true)
			if err != nil {
				return "", fmt.Errorf("failed to create session: %w", err)
			}
			sessionID := strings.TrimSpace(resp.ID)
			if sessionID == "" {
				return "", fmt.Errorf("empty session ID returned")
			}
			if err := sendHeadlessPrompt(serverURL, sessionID, prompt, model, variant, runtimeDir); err != nil {
				return "", fmt.Errorf("failed to send prompt: %w", err)
			}
			return sessionID, nil
		},
		Verbose: healthcheckVerbose,
	}
}

// runHealthcheck is the CLI entry point.
func runHealthcheck() error {
	projectDir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to determine project directory: %w", err)
	}

	deps := DefaultHealthcheckDeps(serverURL, projectDir)
	deps.Verbose = healthcheckVerbose

	result := executeHealthcheck(context.Background(), deps, healthcheckTimeout, healthcheckModel)

	if result.Pass {
		fmt.Printf("PASS: %s (%s)\n", result.Message, formatDuration(result.Elapsed))
		return nil
	}

	fmt.Fprintf(os.Stderr, "FAIL: %s\n", result.Message)
	if result.Diagnostic != "" {
		fmt.Fprintf(os.Stderr, "  Diagnostic: %s\n", result.Diagnostic)
	}
	os.Exit(1)
	return nil
}

// executeHealthcheck runs the full healthcheck pipeline.
// It is separated from runHealthcheck for testability.
func executeHealthcheck(ctx context.Context, deps *HealthcheckDeps, timeout time.Duration, model string) HealthcheckResult {
	start := deps.Now()

	// Step 1: Generate workspace name and create worktree
	workspaceName := healthcheckWorkspacePrefix + deps.Now().Format("150405")
	worktreeDir, branch, err := spawn.CreateWorktree(deps.ProjectDir, workspaceName)
	if err != nil {
		return HealthcheckResult{
			Pass:       false,
			Message:    "Failed to create worktree",
			Diagnostic: err.Error(),
			Elapsed:    deps.Now().Sub(start),
		}
	}

	// Cleanup function — always remove worktree on success, leave on failure for debugging
	cleanup := func(pass bool) {
		if pass {
			if err := spawn.RemoveWorktree(deps.ProjectDir, workspaceName); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cleanup failed: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "  Leaving workspace for debugging: %s (branch: %s)\n", worktreeDir, branch)
		}
	}

	if deps.Verbose {
		fmt.Printf("  Worktree: %s\n", worktreeDir)
		fmt.Printf("  Branch:   %s\n", branch)
	}

	// Step 2: Write a minimal SPAWN_CONTEXT.md so the agent has instructions
	spawnCtxContent := fmt.Sprintf("TASK: %s\n\nThis is a healthcheck canary agent. Complete the task and exit.\n", healthcheckTask)
	spawnCtxPath := worktreeDir + "/SPAWN_CONTEXT.md"
	if err := os.WriteFile(spawnCtxPath, []byte(spawnCtxContent), 0644); err != nil {
		cleanup(false)
		return HealthcheckResult{
			Pass:          false,
			Message:       "Failed to write SPAWN_CONTEXT.md",
			Diagnostic:    err.Error(),
			WorkspaceName: workspaceName,
			Elapsed:       deps.Now().Sub(start),
		}
	}

	// Step 3: Verify SPAWN_CONTEXT.md exists in worktree
	if _, err := os.Stat(spawnCtxPath); os.IsNotExist(err) {
		cleanup(false)
		return HealthcheckResult{
			Pass:          false,
			Message:       "FAIL: spawn artifacts not delivered to worktree",
			Diagnostic:    fmt.Sprintf("SPAWN_CONTEXT.md not found at %s", spawnCtxPath),
			WorkspaceName: workspaceName,
			Elapsed:       deps.Now().Sub(start),
		}
	}

	// Step 4: Spawn headless agent
	resolvedModel := resolveModelForHealthcheck(model)
	sessionTitle := fmt.Sprintf("%s [healthcheck]", workspaceName)
	prompt := fmt.Sprintf("Read your spawn context from %s/SPAWN_CONTEXT.md and begin the task.", worktreeDir)

	sessionID, err := deps.SpawnHeadless(deps.Client, deps.ServerURL, sessionTitle, prompt, resolvedModel, "", worktreeDir)
	if err != nil {
		cleanup(false)
		return HealthcheckResult{
			Pass:          false,
			Message:       "Failed to spawn agent",
			Diagnostic:    err.Error(),
			WorkspaceName: workspaceName,
			Elapsed:       deps.Now().Sub(start),
		}
	}

	if deps.Verbose {
		fmt.Printf("  Session:  %s\n", sessionID)
		fmt.Printf("  Model:    %s\n", resolvedModel)
	}

	// Step 5: Poll for tokens > 0 (model responding)
	modelResponded := false
	var tokensObserved int

	deadline := deps.Now().Add(timeout)
	modelDeadline := deps.Now().Add(healthcheckModelTimeout)

	for deps.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			cleanup(false)
			return HealthcheckResult{
				Pass:          false,
				Message:       "Healthcheck cancelled",
				SessionID:     sessionID,
				WorkspaceName: workspaceName,
				Elapsed:       deps.Now().Sub(start),
			}
		default:
		}

		tokens, err := deps.Client.GetSessionTokens(sessionID)
		if err == nil && tokens != nil && tokens.TotalTokens > 0 {
			modelResponded = true
			tokensObserved = tokens.TotalTokens
			if deps.Verbose {
				fmt.Printf("  Tokens: %d (model responding)\n", tokens.TotalTokens)
			}
			break
		}

		if deps.Now().After(modelDeadline) {
			cleanup(false)
			return HealthcheckResult{
				Pass:          false,
				Message:       "FAIL: model not responding (check model config)",
				Diagnostic:    fmt.Sprintf("0 tokens after %s with model %s, session %s", healthcheckModelTimeout, resolvedModel, sessionID),
				SessionID:     sessionID,
				WorkspaceName: workspaceName,
				Elapsed:       deps.Now().Sub(start),
			}
		}

		if deps.Verbose {
			fmt.Printf("  Polling... (%s elapsed)\n", formatDuration(deps.Now().Sub(start)))
		}
		deps.Sleep(healthcheckPollInterval)
	}

	if !modelResponded {
		cleanup(false)
		return HealthcheckResult{
			Pass:          false,
			Message:       "FAIL: model not responding (check model config)",
			Diagnostic:    fmt.Sprintf("0 tokens after %s", timeout),
			SessionID:     sessionID,
			WorkspaceName: workspaceName,
			Elapsed:       deps.Now().Sub(start),
		}
	}

	// Step 6: Success — model responded and spawn pipeline worked
	elapsed := deps.Now().Sub(start)
	cleanup(true)

	return HealthcheckResult{
		Pass:           true,
		Message:        fmt.Sprintf("Pipeline healthy — agent spawned, model responded (%d tokens)", tokensObserved),
		SessionID:      sessionID,
		WorkspaceName:  workspaceName,
		TokensObserved: tokensObserved,
		Elapsed:        elapsed,
	}
}

// resolveModelForHealthcheck resolves a model alias to a full model spec string.
func resolveModelForHealthcheck(modelAlias string) string {
	spec := resolveModelWithConfig(modelAlias, "opencode", "", nil, nil)
	return spec.Format()
}
