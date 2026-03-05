package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// MigrationRule declares a dual-authority pattern from an incomplete migration.
type MigrationRule struct {
	ID          string   // e.g., "stale-cli-bd-comment"
	Name        string   // Human-readable: "bd comment → bd comments add"
	Category    string   // "cli-reference", "config-schema", "prose-hook-overlap", "dead-code"
	OldPatterns []string // Regexes matching the OLD authority
	ScanScope   []string // Glob patterns: "skills/src/**/*.md", "cmd/orch/**/*.go"
	Severity    string   // "high", "medium", "low"
	FixHint     string   // Actionable fix description

	// Optional behavior flags
	MatchMode     string // "content" (default) or "filename" — match against filename instead of content
	ExcludeTest   bool   // Skip _test.go files
	SkipGoComment bool   // Skip lines starting with // in .go files
}

// MigrationFinding represents a single migration rule match.
type MigrationFinding struct {
	RuleID   string `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Category string `json:"category"`
	File     string `json:"file"`
	Line     int    `json:"line"` // 0 for filename-only matches
	Match    string `json:"match"`
	Severity string `json:"severity"`
	FixHint  string `json:"fix_hint"`
}

// MigrationScanReport contains the results of a migration scan.
type MigrationScanReport struct {
	Findings     []MigrationFinding `json:"findings"`
	FilesScanned int                `json:"files_scanned"`
	RulesChecked int                `json:"rules_checked"`
}

// migrationRules defines the initial set of dual-authority detection rules.
var migrationRules = []MigrationRule{
	{
		ID:          "stale-cli-bd-comment",
		Name:        "bd comment → bd comments add",
		Category:    "cli-reference",
		OldPatterns: []string{`bd comment [^s]`},
		ScanScope:   []string{"skills/src/**/*.md"},
		Severity:    "high",
		FixHint:     "Replace 'bd comment X' with 'bd comments add X'",
	},
	{
		ID:          "stale-cli-orch-frontier",
		Name:        "Removed orch subcommands",
		Category:    "cli-reference",
		OldPatterns: []string{`orch frontier`, `orch reap`, `orch health[^-]`, `orch stability`, `orch friction`},
		ScanScope:   []string{"skills/src/**/*.md"},
		Severity:    "high",
		FixHint:     "Remove references to removed orch subcommands",
	},
	{
		ID:          "stale-cli-orch-flags",
		Name:        "Removed orch spawn flags",
		Category:    "cli-reference",
		OldPatterns: []string{`--bypass-triage`, `--no-track`, `--headless`},
		ScanScope:   []string{"skills/src/**/*.md"},
		Severity:    "medium",
		FixHint:     "Remove references to removed spawn flags",
	},
	{
		ID:          "dead-backup-files",
		Name:        "Dead backup files",
		Category:    "dead-code",
		OldPatterns: []string{`\.template\.backup$`},
		ScanScope:   []string{"skills/src/**/*"},
		Severity:    "low",
		FixHint:     "Delete backup file — source of stale reference findings",
		MatchMode:   "filename",
	},
	{
		ID:          "old-verifyspec-schema",
		Name:        "Old VERIFICATION_SPEC schema",
		Category:    "config-schema",
		OldPatterns: []string{`^level: V[0-3]`},
		ScanScope:   []string{".orch/workspace/**/VERIFICATION_SPEC.yaml"},
		Severity:    "low",
		FixHint:     "Update VERIFICATION_SPEC.yaml to current schema",
	},
	{
		ID:            "daemon-default-config",
		Name:          "daemon.DefaultConfig() in non-test code",
		Category:      "config-schema",
		OldPatterns:   []string{`daemon\.DefaultConfig\(\)`},
		ScanScope:     []string{"cmd/orch/**/*.go"},
		Severity:      "medium",
		FixHint:       "Use daemonconfig.FromUserConfig() as base, override with CLI flags",
		ExcludeTest:   true,
		SkipGoComment: true,
	},
	{
		ID:          "prose-git-add-no-hook",
		Name:        "Prose 'NEVER git add -A' without hook enforcement",
		Category:    "prose-hook-overlap",
		OldPatterns: []string{`NEVER.*git add -A`, `NEVER.*git add \.`},
		ScanScope:   []string{"skills/src/**/*.md"},
		Severity:    "medium",
		FixHint:     "Ensure a PreToolUse hook enforces this constraint mechanically",
	},
}

// migrationAllowlist maps rule IDs to files that are known-correct exceptions.
var migrationAllowlist = map[string]map[string]bool{
	"daemon-default-config": {
		// This file contains the rule definition itself (as a string pattern)
		"cmd/orch/doctor_migration_scan.go": true,
	},
}

// runMigrationScan performs the migration scan and prints results.
func runMigrationScan() error {
	fmt.Println("orch doctor --migration-scan")
	fmt.Println("Scanning for dual-authority patterns (incomplete migrations)...")
	fmt.Println()

	report, err := scanAllMigrationRules(".")
	if err != nil {
		return fmt.Errorf("migration scan error: %w", err)
	}

	fmt.Printf("Files scanned: %d\n", report.FilesScanned)
	fmt.Printf("Rules checked: %d\n", report.RulesChecked)
	fmt.Printf("Findings: %d\n", len(report.Findings))
	fmt.Println()

	if len(report.Findings) == 0 {
		fmt.Println("✓ No dual-authority patterns detected")
		return nil
	}

	printMigrationReport(report.Findings)
	return nil
}

// scanAllMigrationRules runs all migration rules against the given root directory.
func scanAllMigrationRules(rootDir string) (*MigrationScanReport, error) {
	report := &MigrationScanReport{
		Findings:     make([]MigrationFinding, 0),
		RulesChecked: len(migrationRules),
	}

	scannedFiles := make(map[string]bool)

	for _, rule := range migrationRules {
		findings, files := scanMigrationRule(rule, rootDir)
		report.Findings = append(report.Findings, findings...)
		for _, f := range files {
			scannedFiles[f] = true
		}
	}

	report.FilesScanned = len(scannedFiles)

	// Sort findings by category, then severity, then file
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Category != report.Findings[j].Category {
			return report.Findings[i].Category < report.Findings[j].Category
		}
		if report.Findings[i].Severity != report.Findings[j].Severity {
			return severityRank(report.Findings[i].Severity) < severityRank(report.Findings[j].Severity)
		}
		return report.Findings[i].File < report.Findings[j].File
	})

	return report, nil
}

// scanMigrationRule scans files matching the rule's scope for old pattern matches.
// Returns findings and the list of files that were scanned.
func scanMigrationRule(rule MigrationRule, rootDir string) ([]MigrationFinding, []string) {
	var findings []MigrationFinding
	var scannedFiles []string

	// Collect all files matching the scan scope globs using doublestar for ** support
	var files []string
	for _, pattern := range rule.ScanScope {
		matches, err := doublestar.Glob(os.DirFS(rootDir), pattern)
		if err != nil {
			continue
		}
		for _, m := range matches {
			files = append(files, filepath.Join(rootDir, m))
		}
	}

	// Compile old patterns
	var compiledPatterns []*regexp.Regexp
	for _, p := range rule.OldPatterns {
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		compiledPatterns = append(compiledPatterns, re)
	}

	matchMode := rule.MatchMode
	if matchMode == "" {
		matchMode = "content"
	}

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}

		// Compute relative path for display and allowlist lookup
		relPath, err := filepath.Rel(rootDir, file)
		if err != nil {
			relPath = file
		}

		scannedFiles = append(scannedFiles, relPath)

		// Check allowlist
		if allowedFiles, ok := migrationAllowlist[rule.ID]; ok {
			if allowedFiles[relPath] {
				continue
			}
		}

		// Skip test files if ExcludeTest is set
		if rule.ExcludeTest && strings.HasSuffix(file, "_test.go") {
			continue
		}

		if matchMode == "filename" {
			// Match against the filename itself
			basename := filepath.Base(file)
			for _, re := range compiledPatterns {
				if re.MatchString(basename) {
					findings = append(findings, MigrationFinding{
						RuleID:   rule.ID,
						RuleName: rule.Name,
						Category: rule.Category,
						File:     relPath,
						Line:     0,
						Match:    basename,
						Severity: rule.Severity,
						FixHint:  rule.FixHint,
					})
				}
			}
			continue
		}

		// Content matching: scan line by line
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		isGoFile := strings.HasSuffix(file, ".go")
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			// Skip Go comments if configured
			if rule.SkipGoComment && isGoFile && strings.HasPrefix(trimmed, "//") {
				continue
			}

			for _, re := range compiledPatterns {
				if match := re.FindString(line); match != "" {
					findings = append(findings, MigrationFinding{
						RuleID:   rule.ID,
						RuleName: rule.Name,
						Category: rule.Category,
						File:     relPath,
						Line:     lineNum,
						Match:    strings.TrimSpace(match),
						Severity: rule.Severity,
						FixHint:  rule.FixHint,
					})
				}
			}
		}
		f.Close()
	}

	return findings, scannedFiles
}

// groupFindingsByCategory groups findings by their category for display.
func groupFindingsByCategory(findings []MigrationFinding) map[string][]MigrationFinding {
	grouped := make(map[string][]MigrationFinding)
	for _, f := range findings {
		grouped[f.Category] = append(grouped[f.Category], f)
	}
	return grouped
}

// printMigrationReport prints findings grouped by category.
func printMigrationReport(findings []MigrationFinding) {
	grouped := groupFindingsByCategory(findings)

	// Sort categories for deterministic output
	categories := make([]string, 0, len(grouped))
	for cat := range grouped {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	for _, cat := range categories {
		fmt.Printf("── %s ──\n\n", cat)
		for _, f := range grouped[cat] {
			icon := severityIcon(f.Severity)
			if f.Line > 0 {
				fmt.Printf("  %s %s:%d\n", icon, f.File, f.Line)
			} else {
				fmt.Printf("  %s %s\n", icon, f.File)
			}
			fmt.Printf("      Rule: %s\n", f.RuleID)
			fmt.Printf("      Match: %s\n", f.Match)
			fmt.Printf("      Fix: %s\n", f.FixHint)
			fmt.Println()
		}
	}
}
