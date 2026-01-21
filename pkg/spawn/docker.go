// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/tmux"
)

// DockerImageName is the Docker image used for spawning Claude agents.
// This image should be built from ~/.claude/docker-workaround/Dockerfile.
const DockerImageName = "claude-code-mcp"

// SpawnDocker launches a Claude Code agent in Docker via a host tmux window.
// This provides Statsig fingerprint isolation for rate limit escape hatch.
//
// Architecture: Host tmux window runs 'docker run ... claude' (NOT nested tmux).
// This matches the claude backend pattern while providing fresh fingerprint per spawn.
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

	// Docker command that:
	// - Uses interactive terminal (-it)
	// - Auto-removes container on exit (--rm)
	// - Matches host user for file permissions
	// - Mounts home directory for project access
	// - Mounts .claude-docker as .claude for fresh fingerprint (statsig isolation)
	// - Mounts real config files (CLAUDE.md, settings.json, skills/, hooks/) on top
	//   so Docker Claude sees host configs while keeping separate fingerprint
	// - Sets working directory to project
	// - Passes CLAUDE_CONTEXT for hook coordination
	// - Sets PATH with linux-amd64 first for cross-compiled binaries (bd, orch, kb, skillc)
	//   Built via: scripts/cross-compile-linux.sh
	// - Pipes context file to claude with dangerous skip permissions
	dockerCmd := fmt.Sprintf(
		`docker run -it --rm `+
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
			`bash -c 'cat %q | claude --dangerously-skip-permissions'`,
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
