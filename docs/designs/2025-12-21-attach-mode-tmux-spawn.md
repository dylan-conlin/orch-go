# Design: Attach Mode for Tmux Spawn

**Problem:** When spawning an agent in tmux using `orch spawn --tmux`, the agent starts in a new tmux window, but the user's terminal remains in the current session/shell. The user has to manually switch to the new window to see the agent's progress.

**Success Criteria:**
- Adding `--attach` to `orch spawn` automatically attaches the user's terminal to the newly created tmux window.
- Works correctly whether the user is already inside a tmux session or not.
- `--attach` implies `--tmux`.

## Proposed Changes

### 1. pkg/tmux/tmux.go
Add an `Attach` function that handles both `switch-client` (if inside tmux) and `attach-session` (if outside tmux).

```go
func Attach(windowTarget string) error {
	var cmd *exec.Cmd
	if os.Getenv("TMUX") != "" {
		// Inside tmux: switch client to the new window
		cmd = exec.Command("tmux", "switch-client", "-t", windowTarget)
	} else {
		// Outside tmux: attach to the session/window
		// Use -t to target the specific window
		cmd = exec.Command("tmux", "attach-session", "-t", windowTarget)
	}
	
	// We want tmux to take over the terminal, so we should connect its
	// stdin/stdout/stderr to the current process.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}
```

### 2. cmd/orch/main.go
- Add `spawnAttach` boolean flag.
- Update `spawnCmd` to include `--attach` flag.
- Update `runSpawnWithSkill` to accept `attach` parameter.
- Update `runSpawnTmux` to accept `attach` parameter and call `tmux.Attach` if true.

## Testing Strategy
- **Manual Test:** Run `orch spawn --attach investigation "test attach mode"`.
- Verify it attaches correctly when run from a regular shell.
- Verify it switches correctly when run from within a tmux session.
- **Unit Test:** Add a test case in `pkg/tmux/tmux_test.go` to verify the `Attach` command construction (though testing the actual attachment is hard in unit tests).

## Alternatives Considered
- **Always use attach-session:** Might not work as expected if already inside tmux (could create nested sessions).
- **Just print the command to attach:** Less convenient for the user.
