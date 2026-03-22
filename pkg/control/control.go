// Package control manages control plane immutability via macOS chflags uchg.
//
// The control plane consists of settings.json and enforcement hook scripts
// (PreToolUse, Stop events). These files define agent constraints and must
// be protected from agent modification using OS-level immutability.
package control

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// enforcementEvents are hook events that can deny agent actions.
// Only these hooks are control plane; informational hooks (SessionStart,
// PostToolUse, etc.) are data plane.
var enforcementEvents = map[string]bool{
	"PreToolUse": true,
	"Stop":       true,
}

// Status represents the lock state of a control plane file.
type Status struct {
	Path   string
	Exists bool
	Locked bool
}

// DiscoverControlPlaneFiles reads settings.json and returns all control plane
// file paths: settings.json itself plus all enforcement hook scripts that
// exist on disk. Environment variables in hook commands (e.g., $HOME) are
// expanded.
func DiscoverControlPlaneFiles(settingsPath string) ([]string, error) {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("reading settings: %w", err)
	}

	var settings struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}

	files := []string{settingsPath}
	seen := map[string]bool{settingsPath: true}

	for event, groups := range settings.Hooks {
		if !enforcementEvents[event] {
			continue
		}
		for _, group := range groups {
			for _, hook := range group.Hooks {
				cmd := hook.Command
				if cmd == "" {
					continue
				}
				// Expand ~ and environment variables
				path := expandPath(cmd)
				// Resolve to absolute path
				if !filepath.IsAbs(path) {
					continue
				}
				if seen[path] {
					continue
				}
				// Only include files that exist on disk
				if _, err := os.Stat(path); err != nil {
					continue
				}
				seen[path] = true
				files = append(files, path)
			}
		}
	}

	return files, nil
}

// DefaultSettingsPath returns the default path to ~/.claude/settings.json.
func DefaultSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

// FileStatus returns the lock status of a single file.
func FileStatus(path string) (Status, error) {
	s := Status{Path: path}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	s.Exists = true

	// Check for uchg flag using ls -lO
	out, err := exec.Command("/bin/ls", "-lO", path).Output()
	if err != nil {
		return s, fmt.Errorf("checking flags: %w", err)
	}
	_ = info
	// ls -lO output format: "-rw-------  1 user  staff  uchg 7840 Feb 28 19:09 file"
	// The flags field appears after group, before size. If "uchg" appears, file is locked.
	s.Locked = strings.Contains(string(out), "uchg")

	return s, nil
}

// Lock applies chflags uchg to the given files, making them immutable.
// Files that are git-tracked are skipped with a warning, because uchg
// prevents git pull/merge/checkout from replacing the file (git needs
// to unlink+create, which fails with "Operation not permitted" on uchg files).
func Lock(files []string) error {
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("control plane file missing: %s", f)
			}
			return err
		}
		if IsGitTracked(f) {
			fmt.Fprintf(os.Stderr, "  skip %s (git-tracked, uchg would block git pull)\n", f)
			continue
		}
		if err := exec.Command("chflags", "uchg", f).Run(); err != nil {
			return fmt.Errorf("locking %s: %w", f, err)
		}
	}
	return nil
}

// EnsureLocked discovers control plane files and locks any that are unlocked.
// Returns the number of files locked and any error. If settings.json doesn't
// exist, returns (0, nil) — the control plane is optional.
func EnsureLocked() (int, error) {
	sp := DefaultSettingsPath()
	if _, err := os.Stat(sp); os.IsNotExist(err) {
		return 0, nil
	}

	files, err := DiscoverControlPlaneFiles(sp)
	if err != nil {
		return 0, fmt.Errorf("discovering control plane: %w", err)
	}

	locked := 0
	for _, f := range files {
		status, err := FileStatus(f)
		if err != nil || !status.Exists || status.Locked {
			continue
		}
		if IsGitTracked(f) {
			continue
		}
		if err := exec.Command("chflags", "uchg", f).Run(); err != nil {
			return locked, fmt.Errorf("locking %s: %w", f, err)
		}
		locked++
	}
	return locked, nil
}

// Unlock removes the uchg flag from the given files, allowing modification.
func Unlock(files []string) error {
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			if os.IsNotExist(err) {
				continue // Skip missing files during unlock
			}
			return err
		}
		if err := exec.Command("chflags", "nouchg", f).Run(); err != nil {
			return fmt.Errorf("unlocking %s: %w", f, err)
		}
	}
	return nil
}

// UnlockMarkerPath returns the path to the unlock marker file.
// When this file exists, the pre-commit hook skips immutability verification.
func UnlockMarkerPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orch", "harness-unlocked")
}

// WriteUnlockMarker creates the unlock marker file to signal intentional unlock.
func WriteUnlockMarker() error {
	path := UnlockMarkerPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating marker dir: %w", err)
	}
	return os.WriteFile(path, []byte("intentional unlock\n"), 0644)
}

// RemoveUnlockMarker removes the unlock marker file.
func RemoveUnlockMarker() error {
	path := UnlockMarkerPath()
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing marker: %w", err)
	}
	return nil
}

// IsUnlockMarkerPresent returns true if the unlock marker file exists.
func IsUnlockMarkerPresent() bool {
	_, err := os.Stat(UnlockMarkerPath())
	return err == nil
}

// VerifyLocked checks that all control plane files have uchg set.
// Returns a list of unlocked files. If all files are locked, returns nil.
// If settings.json doesn't exist, returns nil (control plane is optional).
func VerifyLocked() (unlocked []string, err error) {
	sp := DefaultSettingsPath()
	if _, err := os.Stat(sp); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := DiscoverControlPlaneFiles(sp)
	if err != nil {
		return nil, fmt.Errorf("discovering control plane: %w", err)
	}

	for _, f := range files {
		status, err := FileStatus(f)
		if err != nil {
			return nil, fmt.Errorf("checking %s: %w", f, err)
		}
		if !status.Exists {
			continue
		}
		if !status.Locked {
			unlocked = append(unlocked, f)
		}
	}
	return unlocked, nil
}

// expandPath expands ~ prefix and environment variables in a path.
// Shell tilde expansion (~/) is not handled by os.ExpandEnv, so we
// handle it explicitly before expanding $VAR references.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	return os.ExpandEnv(path)
}

// IsGitTracked returns true if the file is tracked by git. Git-tracked files
// should not have uchg applied because git pull/merge/checkout needs to
// unlink+replace them, which uchg prevents.
func IsGitTracked(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	dir := filepath.Dir(absPath)
	base := filepath.Base(absPath)
	cmd := exec.Command("git", "-C", dir, "ls-files", "--error-unmatch", base)
	return cmd.Run() == nil
}

// DenyRules returns the deny rules that should be present in settings.json
// to prevent agents from editing control plane files. These provide
// defense-in-depth on top of chflags uchg.
func DenyRules() []string {
	return []string{
		"Edit(~/.claude/settings.json)",
		"Edit(~/.claude/settings.local.json)",
		"Write(~/.claude/settings.json)",
		"Write(~/.claude/settings.local.json)",
		"Edit(~/.orch/hooks/**)",
		"Write(~/.orch/hooks/**)",
	}
}
