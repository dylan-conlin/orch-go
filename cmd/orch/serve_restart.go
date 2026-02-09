package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type serveRestartOutcome struct {
	Restarted bool
	Method    string
}

func runServeRestart() error {
	dir, err := currentProjectDir()
	if err != nil {
		return fmt.Errorf("failed to determine project directory: %w", err)
	}

	result, err := restartManagedOrchServe(dir)
	if err != nil {
		return err
	}

	if !result.Restarted {
		fmt.Println("No managed orch serve process found (overmind/launchd).")
		fmt.Println("Start services with: overmind start -D")
		return nil
	}

	fmt.Printf("Restarted orch serve via %s\n", result.Method)
	return nil
}

func restartManagedOrchServe(projectDir string) (serveRestartOutcome, error) {
	if projectDir == "" {
		dir, err := currentProjectDir()
		if err != nil {
			return serveRestartOutcome{}, fmt.Errorf("failed to determine project directory: %w", err)
		}
		projectDir = dir
	}

	if restarted, err := restartOrchServeViaOvermind(projectDir); err != nil {
		return serveRestartOutcome{}, err
	} else if restarted {
		return serveRestartOutcome{Restarted: true, Method: "overmind"}, nil
	}

	if restarted, label, err := restartOrchServeViaLaunchd(); err != nil {
		return serveRestartOutcome{}, err
	} else if restarted {
		return serveRestartOutcome{Restarted: true, Method: "launchctl:" + label}, nil
	}

	pids, err := findOrchServePIDs()
	if err != nil {
		return serveRestartOutcome{}, err
	}
	if len(pids) == 0 {
		return serveRestartOutcome{}, nil
	}

	return serveRestartOutcome{}, fmt.Errorf(
		"found unmanaged orch serve process(es): %s; restart via overmind ('overmind start -D' then 'orch serve restart')",
		strings.Join(pids, ", "),
	)
}

func restartOrchServeViaOvermind(projectDir string) (bool, error) {
	cmd := exec.Command("overmind", "status")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		return false, nil
	}

	restart := exec.Command("overmind", "restart", "api")
	restart.Dir = projectDir
	restart.Stdout = os.Stdout
	restart.Stderr = os.Stderr
	if err := restart.Run(); err != nil {
		return false, fmt.Errorf("overmind is running but failed to restart api: %w", err)
	}

	return true, nil
}

func restartOrchServeViaLaunchd() (bool, string, error) {
	uid := strconv.Itoa(os.Getuid())
	labels := []string{"com.overmind.orch-go", "com.orch.serve"}

	for _, label := range labels {
		loaded := exec.Command("launchctl", "print", fmt.Sprintf("gui/%s/%s", uid, label))
		if err := loaded.Run(); err != nil {
			continue
		}

		restart := exec.Command("launchctl", "kickstart", "-k", fmt.Sprintf("gui/%s/%s", uid, label))
		restart.Stdout = os.Stdout
		restart.Stderr = os.Stderr
		if err := restart.Run(); err != nil {
			return false, "", fmt.Errorf("launchd label %s is loaded but restart failed: %w", label, err)
		}

		return true, label, nil
	}

	return false, "", nil
}

func findOrchServePIDs() ([]string, error) {
	cmd := exec.Command("pgrep", "-f", "orch.*serve")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	pids := make([]string, 0, len(lines))
	self := strconv.Itoa(os.Getpid())
	for _, line := range lines {
		pid := strings.TrimSpace(line)
		if pid == "" || pid == self {
			continue
		}
		pids = append(pids, pid)
	}

	return pids, nil
}

func restartOrchServeProjectDir(projectDir string) string {
	if projectDir != "" {
		return projectDir
	}
	dir, err := currentProjectDir()
	if err == nil {
		return dir
	}
	if sourceDir != "" && sourceDir != "unknown" {
		return sourceDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Documents", "personal", "orch-go")
}
