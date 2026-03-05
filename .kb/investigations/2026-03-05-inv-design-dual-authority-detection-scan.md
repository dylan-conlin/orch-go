# Design: Dual-Authority Detection Scan for orch doctor

**Date:** 2026-03-05
**Issue:** orch-go-71830
**Status:** Complete
**Type:** Architect design

## Problem

The "Deploy or Delete" principle identifies incomplete migrations as the root cause of 18 configuration-drift investigations in 30 days. The pattern: new system built, old system not removed, both claim authority over the same concern. Currently there is no mechanical gate — the principle exists only as prose.

## Prior Art

`orch doctor --defect-scan` (doctor_defect_scan.go, 652 lines) scans Go source for Class 2 (Multi-Backend Blindness) and Class 5 (Contradictory Authority Signals). It works by:
1. Parsing Go files into function-level `funcInfo` structs
2. Pattern-matching function bodies for known API calls
3. Flagging functions that use one backend without the other (Class 2) or read 3+ authority signals without explicit precedence (Class 5)
4. Allowlists for known-correct functions

This is the right model: mechanical, grep-based, allowlist-gated, runs in doctor.

## Design: `orch doctor --migration-scan`

### Concept: Migration Rules

A **migration rule** declares a dual-authority pattern:

```go
type MigrationRule struct {
    ID          string   // e.g., "stale-cli-bd-comment"
    Name        string   // Human-readable: "bd comment → bd comments add"
    Category    string   // "cli-reference", "config-schema", "prose-hook-overlap", "dead-code"
    OldPatterns []string // Regexes matching the OLD authority
    NewPatterns []string // Regexes matching the NEW authority (optional — for validation)
    ScanScope   []string // Glob patterns: "skills/src/**/*.md", "cmd/orch/**/*.go"
    Severity    string   // "high", "medium", "low"
    FixHint     string   // Actionable: "Replace 'bd comment X' with 'bd comments add X'"
}
```

A finding fires when:
- An `OldPattern` match is found in a file within `ScanScope`
- The match is NOT in an allowlisted context (comments, quoted examples in docs)

### Migration Rules (Initial Set)

Based on the 11 known incomplete migrations:

| ID | Category | Old Pattern | Scan Scope | Severity |
|----|----------|-------------|------------|----------|
| `stale-cli-bd-comment` | cli-reference | `bd comment [^s]` | `skills/src/**/*.md`, `~/.claude/skills/**/*.md` | high |
| `stale-cli-orch-frontier` | cli-reference | `orch (frontier\|reap\|health\|stability\|friction)` | `skills/src/**/*.md`, `~/.claude/skills/**/*.md` | high |
| `stale-cli-orch-flags` | cli-reference | `--bypass-triage\|--no-track\|--headless` | `skills/src/**/*.md` | medium |
| `dead-backup-files` | dead-code | `*.template.backup` | `skills/src/**/*` | low |
| `old-verifyspec-schema` | config-schema | `^level: V[0-3]` | `.orch/workspace/**/VERIFICATION_SPEC.yaml` | low |
| `daemon-default-config` | config-schema | `daemon\.DefaultConfig\(\)` (in non-test, non-plist code) | `cmd/orch/**/*.go` | medium |
| `prose-git-add-no-hook` | prose-hook-overlap | `NEVER.*git add -A\|NEVER.*git add \.` without matching PreToolUse hook | `skills/src/**/*.md` | medium |

### Architecture

```
cmd/orch/doctor_migration_scan.go   (~250 lines)
    ├── MigrationRule definitions (data)
    ├── MigrationFinding struct
    ├── runMigrationScan()          — entry point from doctor.go
    ├── scanMigrationRules()        — iterate rules, scan files, collect findings
    └── printMigrationReport()      — formatted output with fix hints
```

**File placement rationale:** Follows exact pattern of `doctor_defect_scan.go`. Single file, no new packages. Rules are data declarations at file scope.

### Output Format

```
orch doctor --migration-scan
Scanning for dual-authority patterns (incomplete migrations)...

Files scanned: 47
Rules checked: 7
Findings: 3

── cli-reference ──

  🔴 skills/src/worker/feature-impl/.skillc/SKILL.md.template.backup:42
      Rule: stale-cli-bd-comment
      Match: "bd comment orch-go-71830"
      Fix: Replace 'bd comment X' with 'bd comments add X'

── config-schema ──

  🟡 cmd/orch/daemon.go:281
      Rule: daemon-default-config
      Match: daemon.DefaultConfig()
      Fix: Use daemonconfig.FromUserConfig() as base, override with CLI flags

── dead-code ──

  🔵 skills/src/worker/feature-impl/.skillc/SKILL.md.template.backup
      Rule: dead-backup-files
      Fix: Delete backup file — source of stale reference findings
```

### Allowlist Mechanism

Same pattern as defect scan:

```go
var migrationAllowlist = map[string]map[string]bool{
    "stale-cli-bd-comment": {
        // Example in worker-base showing the OLD syntax for comparison
        "skills/src/shared/worker-base/.skillc/reference/migration-examples.md": true,
    },
    "daemon-default-config": {
        // Test files legitimately reference DefaultConfig
        "cmd/orch/doctor_test.go": true,
    },
}
```

### Integration with doctor.go

```go
// In doctor.go — add flag
var doctorMigrationScan bool

// In init()
doctorCmd.Flags().BoolVar(&doctorMigrationScan, "migration-scan", false,
    "Scan for dual-authority patterns (incomplete migrations)")

// In runDoctor()
if doctorMigrationScan {
    return runMigrationScan()
}
```

### Extensibility: Adding New Rules

When a new incomplete migration is discovered:
1. Add a `MigrationRule` to the `migrationRules` slice
2. Optionally add allowlist entries
3. No code changes needed — rules are data

When a migration is completed:
1. Remove the rule (or mark it `Enabled: false`)
2. The scan should output "0 findings" — confirming completion

### What This Does NOT Do

- **No auto-fix.** Findings are advisory. The fix hint tells you what to do.
- **No cross-file analysis.** Each file is scanned independently against each rule. No "file A has pattern X AND file B doesn't have pattern Y" logic. This keeps it simple and fast.
- **No AST parsing.** Pure regex on file content. The defect scan's function-level parsing is overkill for the patterns we're detecting here.
- **No deployed skill scanning by default.** Only scans source tree. Add `--include-deployed` flag later if needed.

### Relationship to Existing Scans

| Scan | What it detects | Scope |
|------|----------------|-------|
| `--defect-scan` | Code-level defect classes (backend blindness, authority conflicts) | Go source |
| `--migration-scan` | Incomplete migrations (dual authorities across any file type) | Skills, configs, Go source |
| `--config` | Plist vs config.yaml drift | Config files |

These are complementary. `--defect-scan` finds code bugs. `--migration-scan` finds systemic incompleteness. `--config` finds value-level drift.

### Estimated Implementation

~250 lines in `doctor_migration_scan.go` + ~50 lines in `doctor.go` for flag wiring + ~100 lines test file. Total ~400 lines. Straightforward feature-impl, no architectural decisions needed.

## Recommendation

Implement as a single feature-impl spawn. The design follows the established `doctor_defect_scan.go` pattern closely enough that no architect review is needed for implementation.

## Open Questions (for orchestrator)

1. **Should `--migration-scan` run as part of default `orch doctor` (no flags)?** The defect scan does NOT run by default. Recommend keeping migration scan opt-in as well, since it scans many files and takes longer than service health checks.

2. **Should findings block daemon spawns?** Currently the defect scan is informational only. The migration scan could optionally set exit code 1 when high-severity findings exist, allowing hooks to gate on it. Recommend: informational first, add gating later if findings are routinely ignored.
