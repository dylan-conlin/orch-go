package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationRuleDefinitions(t *testing.T) {
	// Verify all rules have required fields
	for _, rule := range migrationRules {
		if rule.ID == "" {
			t.Error("Rule missing ID")
		}
		if rule.Name == "" {
			t.Errorf("Rule %s missing Name", rule.ID)
		}
		if rule.Category == "" {
			t.Errorf("Rule %s missing Category", rule.ID)
		}
		if len(rule.OldPatterns) == 0 {
			t.Errorf("Rule %s has no OldPatterns", rule.ID)
		}
		if len(rule.ScanScope) == 0 {
			t.Errorf("Rule %s has no ScanScope", rule.ID)
		}
		if rule.Severity == "" {
			t.Errorf("Rule %s missing Severity", rule.ID)
		}
		if rule.FixHint == "" {
			t.Errorf("Rule %s missing FixHint", rule.ID)
		}
	}
}

func TestMigrationRuleCount(t *testing.T) {
	if len(migrationRules) < 7 {
		t.Errorf("Expected at least 7 migration rules, got %d", len(migrationRules))
	}
}

func TestScanMigrationRule_MatchesOldPattern(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "src", "test")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write file with deprecated bd comment syntax
	content := `# Worker Instructions
Report progress: bd comment orch-go-123 "Phase: Planning"
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rule := MigrationRule{
		ID:          "stale-cli-bd-comment",
		Name:        "bd comment → bd comments add",
		Category:    "cli-reference",
		OldPatterns: []string{`bd comment [^s]`},
		ScanScope:   []string{"skills/src/**/*.md"},
		Severity:    "high",
		FixHint:     "Replace 'bd comment X' with 'bd comments add X'",
	}

	findings, _ := scanMigrationRule(rule, tmpDir)
	if len(findings) == 0 {
		t.Fatal("Expected at least 1 finding for deprecated bd comment syntax")
	}
	if findings[0].RuleID != "stale-cli-bd-comment" {
		t.Errorf("Expected rule ID stale-cli-bd-comment, got %s", findings[0].RuleID)
	}
	if findings[0].Severity != "high" {
		t.Errorf("Expected high severity, got %s", findings[0].Severity)
	}
}

func TestScanMigrationRule_NoMatchForNewPattern(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "src", "test")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write file with correct syntax (bd comments add)
	content := `# Worker Instructions
Report progress: bd comments add orch-go-123 "Phase: Planning"
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rule := MigrationRule{
		ID:          "stale-cli-bd-comment",
		Name:        "bd comment → bd comments add",
		Category:    "cli-reference",
		OldPatterns: []string{`bd comment [^s]`},
		ScanScope:   []string{"skills/src/**/*.md"},
		Severity:    "high",
		FixHint:     "Replace 'bd comment X' with 'bd comments add X'",
	}

	findings, _ := scanMigrationRule(rule, tmpDir)
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for correct syntax, got %d", len(findings))
	}
}

func TestScanMigrationRule_Allowlist(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "src", "test")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write file with deprecated syntax
	content := `bd comment orch-go-123 "example"`
	if err := os.WriteFile(filepath.Join(skillDir, "allowed.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rule := MigrationRule{
		ID:          "test-rule",
		Name:        "test",
		Category:    "cli-reference",
		OldPatterns: []string{`bd comment [^s]`},
		ScanScope:   []string{"skills/src/**/*.md"},
		Severity:    "high",
		FixHint:     "fix it",
	}

	// Add to allowlist
	origAllowlist := migrationAllowlist
	migrationAllowlist = map[string]map[string]bool{
		"test-rule": {
			"skills/src/test/allowed.md": true,
		},
	}
	defer func() { migrationAllowlist = origAllowlist }()

	findings, _ := scanMigrationRule(rule, tmpDir)
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for allowlisted file, got %d", len(findings))
	}
}

func TestScanMigrationRule_DeadBackupFiles(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "src", "test")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a backup file (the old pattern matches filenames, not content)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md.template.backup"), []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	rule := MigrationRule{
		ID:          "dead-backup-files",
		Name:        "Dead backup files",
		Category:    "dead-code",
		OldPatterns: []string{`\.template\.backup$`},
		ScanScope:   []string{"skills/src/**/*"},
		Severity:    "low",
		FixHint:     "Delete backup file",
		MatchMode:   "filename",
	}

	findings, _ := scanMigrationRule(rule, tmpDir)
	if len(findings) == 0 {
		t.Fatal("Expected finding for backup file")
	}
}

func TestScanMigrationRule_GoFileExcludesTests(t *testing.T) {
	tmpDir := t.TempDir()
	cmdDir := filepath.Join(tmpDir, "cmd", "orch")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `package main
func init() {
	daemon.DefaultConfig()
}
`
	// Write to test file — should be excluded
	if err := os.WriteFile(filepath.Join(cmdDir, "daemon_test.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rule := MigrationRule{
		ID:          "daemon-default-config",
		Name:        "daemon.DefaultConfig()",
		Category:    "config-schema",
		OldPatterns: []string{`daemon\.DefaultConfig\(\)`},
		ScanScope:   []string{"cmd/orch/**/*.go"},
		Severity:    "medium",
		FixHint:     "Use daemonconfig.FromUserConfig()",
		ExcludeTest: true,
	}

	findings, _ := scanMigrationRule(rule, tmpDir)
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for test file with ExcludeTest, got %d", len(findings))
	}
}

func TestMigrationFindingFields(t *testing.T) {
	f := MigrationFinding{
		RuleID:   "stale-cli-bd-comment",
		RuleName: "bd comment → bd comments add",
		Category: "cli-reference",
		File:     "skills/src/worker/feature-impl/SKILL.md",
		Line:     42,
		Match:    `bd comment orch-go-71830`,
		Severity: "high",
		FixHint:  "Replace 'bd comment X' with 'bd comments add X'",
	}

	if f.RuleID != "stale-cli-bd-comment" {
		t.Error("RuleID field")
	}
	if f.Line != 42 {
		t.Error("Line field")
	}
}

func TestRunMigrationScanIntegration(t *testing.T) {
	// Create a temp directory with test files
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "src", "test")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	// File with deprecated syntax
	content := `# Instructions
Run: bd comment orch-go-123 "Phase: Planning"
Also: orch frontier
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := scanAllMigrationRules(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if report.RulesChecked != len(migrationRules) {
		t.Errorf("Expected %d rules checked, got %d", len(migrationRules), report.RulesChecked)
	}

	// Should find at least the bd comment and orch frontier findings
	foundBdComment := false
	foundOrchFrontier := false
	for _, f := range report.Findings {
		if f.RuleID == "stale-cli-bd-comment" {
			foundBdComment = true
		}
		if f.RuleID == "stale-cli-orch-frontier" {
			foundOrchFrontier = true
		}
	}

	if !foundBdComment {
		t.Error("Expected to find stale-cli-bd-comment finding")
	}
	if !foundOrchFrontier {
		t.Error("Expected to find stale-cli-orch-frontier finding")
	}
}

func TestMigrationReportGroupsByCategory(t *testing.T) {
	findings := []MigrationFinding{
		{RuleID: "r1", Category: "cli-reference", Severity: "high"},
		{RuleID: "r2", Category: "dead-code", Severity: "low"},
		{RuleID: "r3", Category: "cli-reference", Severity: "medium"},
	}

	grouped := groupFindingsByCategory(findings)
	if len(grouped["cli-reference"]) != 2 {
		t.Errorf("Expected 2 cli-reference findings, got %d", len(grouped["cli-reference"]))
	}
	if len(grouped["dead-code"]) != 1 {
		t.Errorf("Expected 1 dead-code finding, got %d", len(grouped["dead-code"]))
	}
}

func TestScanMigrationRule_MultipleOldPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "src", "test")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `Use orch frontier to check status
Or try orch health for monitoring
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rule := MigrationRule{
		ID:          "stale-cli-orch-frontier",
		Name:        "Removed orch subcommands",
		Category:    "cli-reference",
		OldPatterns: []string{`orch frontier`, `orch health`},
		ScanScope:   []string{"skills/src/**/*.md"},
		Severity:    "high",
		FixHint:     "Remove references to removed commands",
	}

	findings, _ := scanMigrationRule(rule, tmpDir)
	if len(findings) < 2 {
		t.Errorf("Expected at least 2 findings for multiple old patterns, got %d", len(findings))
	}
}

func TestCommentLinesSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "src", "test")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Go comment referencing old pattern should not be flagged
	cmdDir := filepath.Join(tmpDir, "cmd", "orch")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `package main
// daemon.DefaultConfig() was the old way
func init() {}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "test.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rule := MigrationRule{
		ID:            "daemon-default-config",
		Name:          "daemon.DefaultConfig()",
		Category:      "config-schema",
		OldPatterns:   []string{`daemon\.DefaultConfig\(\)`},
		ScanScope:     []string{"cmd/orch/**/*.go"},
		Severity:      "medium",
		FixHint:       "Use daemonconfig.FromUserConfig()",
		ExcludeTest:   true,
		SkipGoComment: true,
	}

	findings, _ := scanMigrationRule(rule, tmpDir)
	if len(findings) != 0 {
		for _, f := range findings {
			t.Logf("Unexpected finding at line %d: %s", f.Line, f.Match)
		}
		t.Errorf("Expected 0 findings for Go comment line, got %d", len(findings))
	}
}

func TestSeverityIconReuse(t *testing.T) {
	// severityIcon is shared with defect scan — verify it works for migration too
	if severityIcon("high") != "🔴" {
		t.Error("high")
	}
	if severityIcon("medium") != "🟡" {
		t.Error("medium")
	}
	if severityIcon("low") != "🔵" {
		t.Error("low")
	}
}

func TestMigrationRuleCategories(t *testing.T) {
	validCategories := map[string]bool{
		"cli-reference":      true,
		"config-schema":      true,
		"dead-code":          true,
		"prose-hook-overlap": true,
	}

	for _, rule := range migrationRules {
		if !validCategories[rule.Category] {
			t.Errorf("Rule %s has invalid category: %s", rule.ID, rule.Category)
		}
	}
}

func TestMigrationRuleUniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, rule := range migrationRules {
		if seen[rule.ID] {
			t.Errorf("Duplicate rule ID: %s", rule.ID)
		}
		seen[rule.ID] = true
	}
}

func TestStaleOrchFlagsRule(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "src", "test")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `Use --bypass-triage to skip triage
Use --no-track for untracked spawns
Use --headless for background execution
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Find the stale-cli-orch-flags rule
	var rule MigrationRule
	for _, r := range migrationRules {
		if r.ID == "stale-cli-orch-flags" {
			rule = r
			break
		}
	}
	if rule.ID == "" {
		t.Fatal("stale-cli-orch-flags rule not found")
	}

	findings, _ := scanMigrationRule(rule, tmpDir)
	if len(findings) == 0 {
		t.Error("Expected findings for stale orch flags")
	}

	// Check that at least one finding contains one of the stale flags
	found := false
	for _, f := range findings {
		if strings.Contains(f.Match, "--bypass-triage") ||
			strings.Contains(f.Match, "--no-track") ||
			strings.Contains(f.Match, "--headless") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected finding matching one of the stale flags")
	}
}
