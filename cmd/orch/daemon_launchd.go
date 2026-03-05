// Package main provides the CLI entry point for orch-go.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/dylan-conlin/orch-go/pkg/daemonconfig"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/spf13/cobra"
)

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install daemon as a launchd service",
	Long: `Generate the launchd plist from ~/.orch/config.yaml and load it into launchd.

This command:
1. Generates ~/Library/LaunchAgents/com.orch.daemon.plist from config
2. Loads the service via launchctl bootstrap
3. The daemon starts immediately (RunAtLoad) and auto-restarts on crash (KeepAlive)

If the service is already loaded, use --force to unload and reload it.

After installation, the daemon survives terminal close, logout, and system sleep.
Use 'orch daemon uninstall' to remove it.

Examples:
  orch daemon install          # Install and start daemon
  orch daemon install --force  # Reinstall (unload + reload)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		return runDaemonInstall(force)
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall daemon from launchd",
	Long: `Stop the daemon and remove it from launchd.

This command:
1. Unloads the service via launchctl bootout
2. Optionally removes the plist file (--remove-plist)

The daemon will stop immediately and will not auto-restart.

Examples:
  orch daemon uninstall                # Unload from launchd (keep plist)
  orch daemon uninstall --remove-plist # Unload and delete plist file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		removePlist, _ := cmd.Flags().GetBool("remove-plist")
		return runDaemonUninstall(removePlist)
	},
}

func init() {
	daemonInstallCmd.Flags().Bool("force", false, "Unload and reload if already installed")
	daemonUninstallCmd.Flags().Bool("remove-plist", false, "Also delete the plist file")

	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)
}

// getUID returns the current user's UID as a string.
func getUID() (string, error) {
	return strconv.Itoa(os.Getuid()), nil
}

// isServiceLoaded checks if com.orch.daemon is loaded in launchd.
func isServiceLoaded() bool {
	cmd := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/com.orch.daemon", os.Getuid()))
	return cmd.Run() == nil
}

func runDaemonInstall(force bool) error {
	plistPath := daemonconfig.GetPlistPath()

	// Check if already loaded
	if isServiceLoaded() {
		if !force {
			fmt.Println("Daemon is already installed in launchd.")
			fmt.Println("Use --force to reinstall, or 'orch daemon uninstall' first.")
			return nil
		}
		// Unload first for reinstall
		fmt.Println("Unloading existing service...")
		uid, err := getUID()
		if err != nil {
			return fmt.Errorf("failed to get UID: %w", err)
		}
		unloadCmd := exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%s/com.orch.daemon", uid))
		if out, err := unloadCmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: bootout failed (may already be unloaded): %s\n", string(out))
		}
	}

	// Generate plist from config
	cfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	content, err := daemonconfig.GeneratePlist(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate plist: %w", err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Write plist
	if err := os.WriteFile(plistPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}
	fmt.Printf("Wrote plist: %s\n", plistPath)

	// Load into launchd via bootstrap
	uid, err := getUID()
	if err != nil {
		return fmt.Errorf("failed to get UID: %w", err)
	}
	loadCmd := exec.Command("launchctl", "bootstrap", fmt.Sprintf("gui/%s", uid), plistPath)
	if out, err := loadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bootstrap service: %s\n%w", string(out), err)
	}

	fmt.Println("Daemon installed and started via launchd.")
	fmt.Println()
	fmt.Println("The daemon will:")
	fmt.Println("  - Start automatically on login (RunAtLoad)")
	fmt.Println("  - Auto-restart on crash (KeepAlive)")
	fmt.Println("  - Log to ~/.orch/daemon.log")
	fmt.Println()
	fmt.Println("Useful commands:")
	fmt.Println("  orch daemon status                    # Check daemon status")
	fmt.Println("  tail -f ~/.orch/daemon.log            # Watch daemon logs")
	fmt.Println("  make install-restart                  # Restart after rebuild")
	fmt.Println("  orch daemon uninstall                 # Remove from launchd")

	return nil
}

func runDaemonUninstall(removePlist bool) error {
	plistPath := daemonconfig.GetPlistPath()

	if !isServiceLoaded() {
		fmt.Println("Daemon is not loaded in launchd.")
		if removePlist {
			if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove plist: %w", err)
			}
			fmt.Printf("Removed: %s\n", plistPath)
		}
		return nil
	}

	// Bootout from launchd
	uid, err := getUID()
	if err != nil {
		return fmt.Errorf("failed to get UID: %w", err)
	}
	bootoutCmd := exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%s/com.orch.daemon", uid))
	if out, err := bootoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bootout service: %s\n%w", string(out), err)
	}

	fmt.Println("Daemon unloaded from launchd.")

	if removePlist {
		if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove plist: %w", err)
		}
		fmt.Printf("Removed: %s\n", plistPath)
	} else {
		fmt.Printf("Plist retained at: %s\n", plistPath)
		fmt.Println("Use --remove-plist to also delete the plist file.")
	}

	return nil
}
