// Package main provides the CLI entry point for orch-go.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/dylan-conlin/orch-go/pkg/config"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/spf13/cobra"
)

var (
	configDryRun bool // Preview without writing files
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Config-as-code management commands",
	Long: `Config-as-code management commands.

Provides generation and drift detection for external configuration files
(like launchd plists) from the declarative ~/.orch/config.yaml source.

Subcommands:
  generate  Generate config files from declarative source
  show      Show current config values`,
}

var configGenerateCmd = &cobra.Command{
	Use:   "generate [plist]",
	Short: "Generate config files from declarative source",
	Long: `Generate external configuration files from ~/.orch/config.yaml.

Supported targets:
  plist     Generate ~/Library/LaunchAgents/com.orch.daemon.plist

The generated files are derived from the declarative config in config.yaml.
Use --dry-run to preview the generated content without writing files.

Examples:
  orch config generate plist          # Generate daemon plist
  orch config generate plist --dry-run  # Preview plist without writing`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		switch target {
		case "plist":
			return runGeneratePlist()
		default:
			return fmt.Errorf("unknown target: %s (supported: plist)", target)
		}
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show [plist]",
	Short: "Show current config values",
	Long: `Show current configuration values from ~/.orch/config.yaml.

Optionally specify a target to see how config maps to that target.

Examples:
  orch config show         # Show all config
  orch config show plist   # Show daemon config as it would appear in plist`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return runShowConfig()
		}
		target := args[0]
		switch target {
		case "plist":
			return runShowPlistConfig()
		default:
			return fmt.Errorf("unknown target: %s (supported: plist)", target)
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a project config value",
	Long: `Set a configuration value in ~/.orch/config.yaml.

Supported keys:
  spawn_mode      Set spawn backend: "claude" or "opencode"
  default_model   Set default model for worker spawns (alias or provider/model)

Examples:
  orch config set spawn_mode claude       # Use Claude Code (tmux) for spawns
  orch config set spawn_mode opencode     # Use OpenCode (HTTP API) for spawns
  orch config set default_model gpt4o     # Use GPT-4o as default worker model
  orch config set default_model sonnet    # Use Sonnet as default worker model`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Load existing config or create new
		cfg, err := config.Load(cwd)
		if err != nil {
			// If config doesn't exist, create with defaults
			cfg = &config.Config{}
			cfg.ApplyDefaults()
		}

		// Set the value
		switch key {
		case "spawn_mode":
			if value != "claude" && value != "opencode" {
				return fmt.Errorf("invalid spawn_mode: %s (must be 'claude' or 'opencode')", value)
			}
			cfg.SpawnMode = value
			// Save project config
			if err := config.Save(cwd, cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		case "default_model":
			// default_model lives in user config (~/.orch/config.yaml)
			ucfg, err := userconfig.Load()
			if err != nil {
				return fmt.Errorf("failed to load user config: %w", err)
			}
			ucfg.DefaultModel = value
			if err := userconfig.Save(ucfg); err != nil {
				return fmt.Errorf("failed to save user config: %w", err)
			}
		default:
			return fmt.Errorf("unknown config key: %s (supported: spawn_mode, default_model)", key)
		}

		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a project config value",
	Long: `Get a configuration value from config.

Supported keys:
  spawn_mode      Current spawn backend ("claude" or "opencode")
  default_model   Default model for worker spawns

Examples:
  orch config get spawn_mode
  orch config get default_model`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Load config
		cfg, err := config.Load(cwd)
		if err != nil {
			// If config doesn't exist, show default
			cfg = &config.Config{}
			cfg.ApplyDefaults()
		}

		// Get the value
		switch key {
		case "spawn_mode":
			fmt.Println(cfg.SpawnMode)
		case "default_model":
			ucfg, err := userconfig.Load()
			if err != nil {
				return fmt.Errorf("failed to load user config: %w", err)
			}
			if ucfg.DefaultModel == "" {
				fmt.Println("(not set - using hardcoded default)")
			} else {
				fmt.Println(ucfg.DefaultModel)
			}
		default:
			return fmt.Errorf("unknown config key: %s (supported: spawn_mode, default_model)", key)
		}

		return nil
	},
}

func init() {
	configCmd.AddCommand(configGenerateCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)

	configGenerateCmd.Flags().BoolVar(&configDryRun, "dry-run", false, "Preview without writing files")

	rootCmd.AddCommand(configCmd)
}

// PlistData holds the template data for generating the plist file.
type PlistData struct {
	Label            string
	OrchPath         string
	PollInterval     int
	MaxAgents        int
	IssueLabel       string
	Verbose          bool
	ReflectIssues    bool
	LogPath          string
	WorkingDirectory string
	PATH             string
	Home             string
}

// plistTemplate is the launchd plist template for the orch daemon.
const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>

    <key>ProgramArguments</key>
    <array>
        <string>{{.OrchPath}}</string>
        <string>daemon</string>
        <string>run</string>
        <string>--poll-interval</string>
        <string>{{.PollInterval}}</string>
        <string>--max-agents</string>
        <string>{{.MaxAgents}}</string>
        <string>--label</string>
        <string>{{.IssueLabel}}</string>{{if .Verbose}}
        <string>--verbose</string>{{end}}
        <string>--reflect-issues={{.ReflectIssues}}</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>

    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>

    <key>WorkingDirectory</key>
    <string>{{.WorkingDirectory}}</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>{{.PATH}}</string>
        <key>BEADS_NO_DAEMON</key>
        <string>1</string>
    </dict>
</dict>
</plist>
`

func runGeneratePlist() error {
	cfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	data, err := buildPlistData(cfg)
	if err != nil {
		return fmt.Errorf("failed to build plist data: %w", err)
	}

	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	plistPath := getPlistPath()

	if configDryRun {
		fmt.Printf("# Would write to: %s\n", plistPath)
		fmt.Println("# Generated from: ~/.orch/config.yaml")
		fmt.Println()
		fmt.Println(buf.String())
		return nil
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Check if file exists and compare
	existingContent, err := os.ReadFile(plistPath)
	if err == nil && bytes.Equal(existingContent, buf.Bytes()) {
		fmt.Println("Plist is already up to date.")
		return nil
	}

	// Write the new plist
	if err := os.WriteFile(plistPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	fmt.Printf("Updated: %s\n", plistPath)
	fmt.Println()
	fmt.Println("To apply changes, restart the daemon:")
	fmt.Println("  launchctl kickstart -k gui/$(id -u)/com.orch.daemon")

	return nil
}

func buildPlistData(cfg *userconfig.Config) (*PlistData, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Find orch binary path
	orchPath := findOrchPath(home)

	// Build PATH from config
	pathDirs := cfg.DaemonPath()
	// Add system paths
	systemPaths := []string{"/usr/local/bin", "/usr/bin", "/bin"}
	allPaths := append(pathDirs, systemPaths...)
	pathStr := strings.Join(allPaths, ":")

	return &PlistData{
		Label:            "com.orch.daemon",
		OrchPath:         orchPath,
		PollInterval:     cfg.DaemonPollInterval(),
		MaxAgents:        cfg.DaemonMaxAgents(),
		IssueLabel:       cfg.DaemonLabel(),
		Verbose:          cfg.DaemonVerbose(),
		ReflectIssues:    cfg.DaemonReflectIssues(),
		LogPath:          filepath.Join(home, ".orch", "daemon.log"),
		WorkingDirectory: cfg.DaemonWorkingDirectory(),
		PATH:             pathStr,
		Home:             home,
	}, nil
}

func findOrchPath(home string) string {
	// Check common locations
	candidates := []string{
		filepath.Join(home, "bin", "orch"),
		filepath.Join(home, "go", "bin", "orch"),
		filepath.Join(home, ".bun", "bin", "orch"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fall back to which
	if path, err := exec.LookPath("orch"); err == nil {
		return path
	}

	// Default to ~/bin/orch
	return filepath.Join(home, "bin", "orch")
}

func getPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.orch.daemon.plist")
}

func runShowConfig() error {
	cfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("# ~/.orch/config.yaml (effective values)")
	fmt.Println()
	fmt.Printf("backend: %s\n", cfg.Backend)
	if cfg.DefaultModel != "" {
		fmt.Printf("default_model: %s\n", cfg.DefaultModel)
	} else {
		fmt.Printf("default_model: (not set - using hardcoded default)\n")
	}
	fmt.Printf("auto_export_transcript: %v\n", cfg.AutoExportTranscript)
	fmt.Printf("default_tier: %s\n", cfg.DefaultTier)
	fmt.Println()
	fmt.Println("notifications:")
	fmt.Printf("  enabled: %v\n", cfg.NotificationsEnabled())
	fmt.Println()
	fmt.Println("reflect:")
	fmt.Printf("  enabled: %v\n", cfg.ReflectEnabled())
	fmt.Printf("  interval_minutes: %d\n", cfg.ReflectIntervalMinutes())
	fmt.Printf("  create_issues: %v\n", cfg.ReflectCreateIssues())
	fmt.Println()
	fmt.Println("daemon:")
	fmt.Printf("  poll_interval: %d\n", cfg.DaemonPollInterval())
	fmt.Printf("  max_agents: %d\n", cfg.DaemonMaxAgents())
	fmt.Printf("  label: %s\n", cfg.DaemonLabel())
	fmt.Printf("  verbose: %v\n", cfg.DaemonVerbose())
	fmt.Printf("  reflect_issues: %v\n", cfg.DaemonReflectIssues())
	fmt.Printf("  working_directory: %s\n", cfg.DaemonWorkingDirectory())
	fmt.Println("  path:")
	for _, p := range cfg.DaemonPath() {
		fmt.Printf("    - %s\n", p)
	}

	return nil
}

func runShowPlistConfig() error {
	cfg, err := userconfig.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	data, err := buildPlistData(cfg)
	if err != nil {
		return fmt.Errorf("failed to build plist data: %w", err)
	}

	fmt.Println("# Daemon plist configuration (from ~/.orch/config.yaml)")
	fmt.Println()
	fmt.Printf("Label:             %s\n", data.Label)
	fmt.Printf("OrchPath:          %s\n", data.OrchPath)
	fmt.Printf("PollInterval:      %d seconds\n", data.PollInterval)
	fmt.Printf("MaxAgents:         %d\n", data.MaxAgents)
	fmt.Printf("IssueLabel:        %s\n", data.IssueLabel)
	fmt.Printf("Verbose:           %v\n", data.Verbose)
	fmt.Printf("ReflectIssues:     %v\n", data.ReflectIssues)
	fmt.Printf("WorkingDirectory:  %s\n", data.WorkingDirectory)
	fmt.Printf("LogPath:           %s\n", data.LogPath)
	fmt.Println()
	fmt.Printf("PATH:\n  %s\n", strings.ReplaceAll(data.PATH, ":", "\n  "))
	fmt.Println()
	fmt.Printf("Plist location:    %s\n", getPlistPath())

	return nil
}
