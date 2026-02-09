package spawn

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const verificationSpecFileName = "VERIFICATION_SPEC.yaml"

type verificationSpecSkeleton struct {
	Version      int                           `yaml:"version"`
	Scope        verificationSpecScope         `yaml:"scope"`
	Verification []verificationSpecSkeletonRow `yaml:"verification"`
}

type verificationSpecScope struct {
	BeadsID   string `yaml:"beads_id"`
	Workspace string `yaml:"workspace"`
	Skill     string `yaml:"skill"`
}

type verificationSpecSkeletonRow struct {
	ID             string                         `yaml:"id"`
	Method         string                         `yaml:"method"`
	Tier           string                         `yaml:"tier"`
	Command        string                         `yaml:"command,omitempty"`
	CWD            string                         `yaml:"cwd,omitempty"`
	TimeoutSeconds int                            `yaml:"timeout_seconds,omitempty"`
	Expect         verificationSpecSkeletonExpect `yaml:"expect"`
}

type verificationSpecSkeletonExpect struct {
	ExitCode       int      `yaml:"exit_code"`
	StdoutContains []string `yaml:"stdout_contains,omitempty"`
}

// WriteVerificationSpecSkeleton writes a spawn-scoped VERIFICATION_SPEC.yaml skeleton.
func WriteVerificationSpecSkeleton(cfg *Config) error {
	workspacePath := cfg.WorkspacePath()
	specPath := filepath.Join(workspacePath, verificationSpecFileName)

	if _, err := os.Stat(specPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat verification spec: %w", err)
	}

	content, err := GenerateVerificationSpecSkeleton(cfg)
	if err != nil {
		return err
	}

	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write verification spec skeleton: %w", err)
	}

	return nil
}

// GenerateVerificationSpecSkeleton creates a tier/skill-specific proof spec skeleton.
func GenerateVerificationSpecSkeleton(cfg *Config) (string, error) {
	tier := normalizeVerificationTier(cfg.Tier, cfg.SkillName)
	skill := strings.TrimSpace(cfg.SkillName)
	if skill == "" {
		skill = "<fill-skill-name>"
	}

	beadsID := strings.TrimSpace(cfg.BeadsID)
	if beadsID == "" {
		beadsID = "<fill-beads-id>"
	}

	workspace := strings.TrimSpace(cfg.WorkspaceName)
	if workspace == "" {
		workspace = "<fill-workspace-name>"
	}

	entries := verificationEntriesForSkill(cfg, tier)
	if err := validateVerificationEntryCommands(entries); err != nil {
		return "", fmt.Errorf("validate verification spec skeleton commands: %w", err)
	}

	spec := verificationSpecSkeleton{
		Version: 1,
		Scope: verificationSpecScope{
			BeadsID:   beadsID,
			Workspace: workspace,
			Skill:     skill,
		},
		Verification: entries,
	}

	body, err := yaml.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshal verification spec skeleton: %w", err)
	}

	header := "# Spawn-time verification skeleton.\n" +
		"# Placeholder commands are syntax-valid and intentionally fail until replaced.\n" +
		"# Replace command and expect blocks before Phase: Complete.\n\n"

	return header + string(body), nil
}

func normalizeVerificationTier(configuredTier, skillName string) string {
	tier := strings.TrimSpace(configuredTier)
	if tier == "" {
		tier = strings.TrimSpace(DefaultTierForSkill(skillName))
	}
	if tier == "" {
		return TierFull
	}
	return tier
}

func verificationEntriesForSkill(cfg *Config, tier string) []verificationSpecSkeletonRow {
	skill := strings.ToLower(strings.TrimSpace(cfg.SkillName))

	switch {
	case isBrowserUISkill(skill):
		return browserVerificationEntries(tier)
	case isImplementationVerificationSkill(skill):
		entries := implementationVerificationEntries(tier)
		if strings.EqualFold(strings.TrimSpace(cfg.MCP), "playwright") {
			entries = append(entries, browserVerificationEntries(tier)...)
		}
		return entries
	case isArtifactVerificationSkill(skill):
		return artifactVerificationEntries(tier)
	default:
		return []verificationSpecSkeletonRow{
			{
				ID:      "verify-cli-command",
				Method:  "cli_smoke",
				Tier:    tier,
				Command: placeholderCommand("verification"),
				CWD:     ".",
				Expect: verificationSpecSkeletonExpect{
					ExitCode: 0,
				},
			},
		}
	}
}

func isImplementationVerificationSkill(skill string) bool {
	switch skill {
	case "feature-impl", "systematic-debugging", "reliability-testing":
		return true
	default:
		return false
	}
}

func isArtifactVerificationSkill(skill string) bool {
	switch skill {
	case "investigation", "architect", "research", "codebase-audit":
		return true
	default:
		return false
	}
}

func isBrowserUISkill(skill string) bool {
	if strings.Contains(skill, "ui") {
		return true
	}

	switch skill {
	case "design-session", "ui-design-session", "ui-mockup-generation":
		return true
	default:
		return false
	}
}

func implementationVerificationEntries(tier string) []verificationSpecSkeletonRow {
	return []verificationSpecSkeletonRow{
		{
			ID:      "verify-build",
			Method:  "cli_smoke",
			Tier:    tier,
			Command: placeholderCommand("build"),
			CWD:     ".",
			Expect: verificationSpecSkeletonExpect{
				ExitCode: 0,
			},
		},
		{
			ID:      "verify-test",
			Method:  "cli_smoke",
			Tier:    tier,
			Command: placeholderCommand("test"),
			CWD:     ".",
			Expect: verificationSpecSkeletonExpect{
				ExitCode: 0,
			},
		},
	}
}

func artifactVerificationEntries(tier string) []verificationSpecSkeletonRow {
	return []verificationSpecSkeletonRow{
		{
			ID:      "verify-artifact-exists",
			Method:  "static",
			Tier:    tier,
			Command: "test -f \"<path-to-artifact>\"",
			CWD:     ".",
			Expect: verificationSpecSkeletonExpect{
				ExitCode: 0,
			},
		},
	}
}

func browserVerificationEntries(tier string) []verificationSpecSkeletonRow {
	return []verificationSpecSkeletonRow{
		{
			ID:             "verify-browser",
			Method:         "browser",
			Tier:           tier,
			Command:        placeholderCommand("browser"),
			TimeoutSeconds: 45,
			Expect: verificationSpecSkeletonExpect{
				ExitCode: 0,
			},
		},
	}
}

func placeholderCommand(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "verification"
	}
	return fmt.Sprintf("echo \"TODO: replace %s command in VERIFICATION_SPEC.yaml\" >&2; exit 2", kind)
}

func validateVerificationEntryCommands(entries []verificationSpecSkeletonRow) error {
	errList := make([]string, 0)
	for i, entry := range entries {
		if strings.EqualFold(strings.TrimSpace(entry.Method), "manual") {
			continue
		}

		command := strings.TrimSpace(entry.Command)
		if command == "" {
			continue
		}

		if err := validateBashSyntax(command); err != nil {
			errList = append(errList, fmt.Sprintf("verification[%d].command (%s): %v", i, entry.ID, err))
		}
	}

	if len(errList) == 0 {
		return nil
	}

	return errors.New(strings.Join(errList, "; "))
}

func validateBashSyntax(command string) error {
	cmd := exec.Command("bash", "-n", "-c", command)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		text = err.Error()
	}

	return fmt.Errorf("invalid bash syntax: %s", text)
}
