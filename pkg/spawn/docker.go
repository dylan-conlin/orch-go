// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// DockerImageName is the Docker image used for spawning Claude agents.
// This image should be built from ~/.claude/docker-workaround/Dockerfile.
const DockerImageName = "claude-code-mcp"

// ContainerNamePrefix is the prefix used for orch-managed Docker containers.
const ContainerNamePrefix = "orch-"

// SpawnDocker launches a Claude Code agent in Docker via a host tmux window.
// This provides Statsig fingerprint isolation for rate limit escape hatch.
//
// Architecture: Host tmux window runs 'docker run ... claude' (NOT nested tmux).
// This matches the claude backend pattern while providing fresh fingerprint per spawn.
//
// Container tracking: The container name is written to .container_id in the workspace
// for cleanup by `orch complete` and `orch abandon`.
func SpawnDocker(cfg *Config) (*tmux.SpawnResult, error) {
	// 1. Ensure appropriate tmux session exists
	// Docker mode is an escape hatch, so both orchestrators and workers go into workers session
	sessionName, err := tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure tmux session: %w", err)
	}

	// 2. Build window name with emoji and beads ID
	windowName := tmux.BuildWindowName(cfg.WorkspaceName, cfg.SkillName, cfg.BeadsID)

	// 3. Create detached window and get its target and ID
	windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, cfg.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux window: %w", err)
	}

	// 4. Build Docker command
	contextPath := cfg.ContextFilePath()

	// Determine CLAUDE_CONTEXT env var to signal hooks to skip duplicate injection
	claudeContext := inferDockerClaudeContext(cfg)

	// Ensure the docker-specific claude config directory exists
	// This provides fresh Statsig fingerprint isolation
	dockerConfigDir := filepath.Join(os.Getenv("HOME"), ".claude-docker")
	if err := os.MkdirAll(dockerConfigDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create docker config dir: %w", err)
	}

	// 4a. Generate container name for tracking and cleanup
	// Format: orch-{workspace-name} (sanitized for Docker naming rules)
	containerName := ContainerNamePrefix + sanitizeContainerName(cfg.WorkspaceName)

	// 4b. Write container name to workspace for cleanup by orch complete/abandon
	workspacePath := filepath.Join(cfg.ProjectDir, ".orch", "workspace", cfg.WorkspaceName)
	containerIDFile := filepath.Join(workspacePath, ".container_id")
	if err := os.WriteFile(containerIDFile, []byte(containerName), 0644); err != nil {
		// Non-fatal - cleanup will just skip this container if file is missing
		fmt.Fprintf(os.Stderr, "Warning: failed to write container ID file: %v\n", err)
	}

	// Docker command that:
	// - Uses interactive terminal (-it)
	// - Auto-removes container on exit (--rm)
	// - Names the container for tracking and cleanup (--name)
	// - Limits memory to 6GB to prevent OOM kills in Colima VM
	//   (Colima VM has 12GB total; 6GB per container allows 2 concurrent agents)
	//   Setting --memory-swap equal to --memory disables swap (prevents disk thrashing)
	// - Matches host user for file permissions
	// - Mounts home directory for project access
	// - Mounts .claude-docker as .claude for fresh fingerprint (statsig isolation)
	// - Mounts real config files (CLAUDE.md, settings.json, skills/, hooks/) on top
	//   so Docker Claude sees host configs while keeping separate fingerprint
	// - Sets working directory to project
	// - Passes CLAUDE_CONTEXT for hook coordination
	// - Sets PATH with linux-amd64 first for cross-compiled binaries (bd, orch, kb, skillc)
	//   Built via: scripts/cross-compile-linux.sh
	// - Uses stdin redirection (not pipe) to feed context file to claude
	//   Redirection works more reliably than pipes with Docker's -it flag
	dockerCmd := fmt.Sprintf(
		`docker run -it --rm `+
			`--name %q `+
			`--memory 6g --memory-swap 6g `+
			`--user "$(id -u):$(id -g)" `+
			`-v "$HOME":"$HOME" `+
			`-v "$HOME/.claude-docker":"$HOME/.claude" `+
			`-v "$HOME/.claude/CLAUDE.md":"$HOME/.claude/CLAUDE.md":ro `+
			`-v "$HOME/.claude/settings.json":"$HOME/.claude/settings.json":ro `+
			`-v "$HOME/.claude/skills":"$HOME/.claude/skills":ro `+
			`-v "$HOME/.claude/hooks":"$HOME/.claude/hooks":ro `+
			`-v "$HOME/.orch/hooks":"$HOME/.orch/hooks":ro `+
			`-w %q `+
			`-e HOME="$HOME" `+
			`-e CLAUDE_CONTEXT=%s `+
			`-e TERM=xterm-256color `+
			`-e BEADS_NO_DAEMON=1 `+
			`-e PATH="$HOME/.local/bin/linux-amd64:$HOME/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" `+
			`%s `+
			`bash -c 'claude --dangerously-skip-permissions < %q'`,
		containerName,
		cfg.ProjectDir,
		claudeContext,
		DockerImageName,
		contextPath,
	)

	// 5. Send command to tmux window
	if err := tmux.SendKeys(windowTarget, dockerCmd); err != nil {
		return nil, fmt.Errorf("failed to send docker command: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return nil, fmt.Errorf("failed to send enter: %w", err)
	}

	return &tmux.SpawnResult{
		Window:        windowTarget,
		WindowID:      windowID,
		WindowName:    windowName,
		WorkspaceName: cfg.WorkspaceName,
	}, nil
}

// inferDockerClaudeContext determines the CLAUDE_CONTEXT value for Docker spawns.
func inferDockerClaudeContext(cfg *Config) string {
	switch {
	case cfg.IsMetaOrchestrator:
		return "meta-orchestrator"
	case cfg.IsOrchestrator:
		return "orchestrator"
	default:
		return "worker"
	}
}

// MonitorDocker captures the current content of the Docker agent's tmux pane.
// Uses the same mechanism as claude mode since Docker runs in host tmux.
func MonitorDocker(windowTarget string) (string, error) {
	return tmux.GetPaneContent(windowTarget)
}

// SendDocker sends keys to the Docker agent's tmux pane, followed by Enter.
// Uses the same mechanism as claude mode since Docker runs in host tmux.
func SendDocker(windowTarget, keys string) error {
	if err := tmux.SendKeysLiteral(windowTarget, keys); err != nil {
		return fmt.Errorf("failed to send keys: %w", err)
	}
	if err := tmux.SendEnter(windowTarget); err != nil {
		return fmt.Errorf("failed to send enter: %w", err)
	}
	return nil
}

// AbandonDocker kills the tmux window running the Docker agent.
// This will also terminate the Docker container (it runs with --rm).
func AbandonDocker(windowTarget string) error {
	if err := tmux.KillWindow(windowTarget); err != nil {
		return fmt.Errorf("failed to kill tmux window: %w", err)
	}
	return nil
}

// CheckDockerImage verifies the Docker image exists.
// Returns an error with helpful instructions if the image is not found.
func CheckDockerImage() error {
	// We could shell out to `docker image inspect` but that adds dependency on docker CLI
	// For now, just return nil and let the spawn fail with docker's error message
	// The error from docker run will be clear enough
	return nil
}

// sanitizeContainerName converts a workspace name to a valid Docker container name.
// Docker container names must match [a-zA-Z0-9][a-zA-Z0-9_.-]*.
// We replace invalid characters with hyphens.
func sanitizeContainerName(name string) string {
	var result []byte
	for i, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result = append(result, byte(c))
		} else if c == '_' || c == '.' || c == '-' {
			// These are valid in Docker names (except at position 0)
			if i > 0 {
				result = append(result, byte(c))
			} else {
				result = append(result, 'x') // Replace leading special char
			}
		} else {
			// Replace other chars with hyphen
			result = append(result, '-')
		}
	}
	// Ensure name starts with alphanumeric
	if len(result) > 0 && !((result[0] >= 'a' && result[0] <= 'z') || (result[0] >= 'A' && result[0] <= 'Z') || (result[0] >= '0' && result[0] <= '9')) {
		result[0] = 'x'
	}
	return string(result)
}

// CleanupDockerContainer stops and removes a Docker container by name.
// This is called by orch complete and orch abandon to clean up containers
// that were spawned with the docker backend.
//
// Returns nil if:
// - Container was successfully stopped/removed
// - Container doesn't exist (already cleaned up or never created)
// - Container already stopped (--rm flag auto-removed it)
//
// Only returns error for unexpected failures.
func CleanupDockerContainer(containerName string) error {
	if containerName == "" {
		return nil
	}

	// Try to stop the container (will fail gracefully if already stopped/removed)
	// docker stop sends SIGTERM, waits, then SIGKILL
	stopCmd := exec.Command("docker", "stop", containerName)
	stopOutput, stopErr := stopCmd.CombinedOutput()

	if stopErr != nil {
		// Check if error is "container not found" - that's fine
		outputStr := string(stopOutput)
		if strings.Contains(outputStr, "No such container") ||
			strings.Contains(outputStr, "not found") {
			// Container already gone - this is expected with --rm flag
			return nil
		}
		// Log warning but don't fail - container might be in weird state
		fmt.Fprintf(os.Stderr, "Warning: docker stop %s: %s\n", containerName, outputStr)
	}

	// Try to remove the container (may already be removed by --rm flag)
	rmCmd := exec.Command("docker", "rm", "-f", containerName)
	rmOutput, rmErr := rmCmd.CombinedOutput()

	if rmErr != nil {
		outputStr := string(rmOutput)
		if strings.Contains(outputStr, "No such container") ||
			strings.Contains(outputStr, "not found") {
			// Container already gone - expected with --rm flag
			return nil
		}
		// Log warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: docker rm %s: %s\n", containerName, outputStr)
	}

	return nil
}

// ReadContainerID reads the container name from a workspace's .container_id file.
// Returns empty string if file doesn't exist or can't be read.
func ReadContainerID(workspacePath string) string {
	if workspacePath == "" {
		return ""
	}
	containerIDFile := filepath.Join(workspacePath, ".container_id")
	data, err := os.ReadFile(containerIDFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
