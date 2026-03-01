# Probe: Hotspot System Blind Spot Analysis — Five Dimensions

**Model:** completion-verification
**Date:** 2026-03-01
**Status:** Complete

---

## Question

The hotspot system measures bloat-size, fix-density, investigation-cluster, and coupling-cluster — all retrospective file-level signals. Does this miss significant structural problems? Testing five hypothesized blind spots with concrete codebase evidence.

---

## What I Tested

Ran 5 parallel analysis agents, each covering one blind spot dimension, with git log analysis, grep-based code analysis, import graph tracing, and manual code review. Independently validated the highest-severity findings.

### 1. Cold Spots
```bash
git log --since="28 days ago" --name-only --pretty=format: | sort | uniq -c | sort -rn
# Cross-referenced imports from top-10 churn files against modification frequency
```

### 2. Semantic Complexity Without Size
```bash
# Analyzed all Go files under 800 lines for:
# - Nesting depth, error branches, goroutine/channel/mutex/select usage
# - sync.Cond patterns, state machines, bidirectional resolution logic
```

### 3. Implicit Coupling
```bash
grep -rn "^var " pkg/ cmd/ --include="*.go" | grep -v _test.go  # global state
# Traced 10+ string-literal coupling protocols (skill names, status strings, labels)
# Verified git co-change patterns for coupled files
```

### 4. Scattered Duplication
```bash
grep -rn "beads.FindSocketPath" --include="*.go" | wc -l  # 58 occurrences
grep -rn 'exec.Command("bd"' --include="*.go" | grep -v _test.go  # 22 direct calls
```

### 5. Dead Features
```bash
# Traced all package imports, checked for zero external callers
# Verified pkg/capacity/, pkg/shell/, pkg/certs/ have zero non-test imports
# Confirmed cmd/orch/orch (21MB binary) tracked in git
```

---

## What I Observed

### Blind Spot 1: Cold Spots — CONFIRMED, REAL PAIN

**Top finding:** `pkg/opencode/service.go` and `pkg/opencode/monitor.go` (467 lines combined) haven't been touched since Dec 2025, yet `pkg/opencode/` is imported by 46 files. They contain dead code from disabled SSE completion detection (Dec 2025) including duplicate `extractBeadsIDFromTitle` logic. The `client.go` in the same package had 9 commits — clear split personality.

**Also found:** `pkg/state/reconcile.go` (1 commit in 28 days) is the single source of truth for agent liveness, depended on by the #1 and #2 highest-churn files. Its heuristic beads ID extraction logic hasn't been validated against current naming conventions. High-churn callers may be working around its bugs rather than fixing them.

**Verdict:** Cold spot detection would catch real problems. The opencode SSE subsystem is frozen dead code adding coupling weight; reconcile.go is frozen infrastructure that may be silently wrong.

### Blind Spot 2: Semantic Complexity — CONFIRMED, HIGH SEVERITY

**Top finding:** 4 files under 800 lines with HIGH complexity:

1. **`cmd/orch/swarm.go`** (667 lines): 9 goroutines, 13 channels, 9 mutexes, 6 selects. The goroutine+channel+mutex interactions have implicit ordering constraints. The classic Go `break`-in-`select` trap exists (line 352-358).

2. **`pkg/capacity/manager.go`** (323 lines): Uses `sync.Cond` (notoriously tricky) with goroutine + context cancellation. The `Wait()`/`Broadcast()` interaction requires understanding 3 concurrent control flows. Silent error swallowing at line 307 makes accounts invisible on transient failures.

3. **`cmd/orch/serve_agents_cache.go`** (718 lines): 34 mutex uses, TOCTOU race in check-then-act pattern (lines 68-89), non-deterministic "first writer wins" cache merge (lines 540-612).

4. **`pkg/spawn/resolve.go`** (710 lines): Bidirectional model<->backend resolution with 7 precedence levels and conditional bypasses. Not concurrent but combinatorially complex.

**Verdict:** Semantic complexity is invisible to the hotspot system. These files would produce subtly wrong agent code due to concurrency hazards and implicit ordering constraints.

### Blind Spot 3: Implicit Coupling — CONFIRMED, 3 CRITICAL + 1 BUG FOUND

**Critical couplings that never co-change:**

1. **`beads.DefaultDir` global mutation** (CRITICAL): `complete_pipeline.go` sets it at line 103 without defer-restore. Error paths leave it pointing at the wrong project, causing all subsequent beads operations to target wrong issue database. The `spawn_cmd.go` version uses defer correctly.

2. **Skill name strings** (CRITICAL): 10 independent maps across 7 packages (769 string occurrences) that must agree on valid skill names. Zero shared constants. These maps do NOT co-change in git. Adding a new skill requires updating all 10 locations.

3. **Status string filtering gap** (CRITICAL + existing bug): `verify/beads_api.go` filters `"open" || "in_progress" || "blocked"` but `serve_agents_discovery.go` filters only `"open" || "in_progress"`. **Blocked agents are invisible on the dashboard.** Confirmed at serve_agents_discovery.go:305 and :322.

4. **SYNTHESIS.md Recommendation regex bug** (confirmed): `synthesis_parser.go:14` uses `\w+` which can't match "spawn-follow-up" (hyphen breaks word characters). But `escalation.go:175` checks for `rec == "spawn-follow-up"`. This means spawn-follow-up recommendations are silently dropped. Confirmed by reading both files.

**Verdict:** Implicit coupling is the most dangerous blind spot. It produces bugs that are invisible at the file level — the coupling is between string protocols that span packages without type safety.

### Blind Spot 4: Scattered Duplication — CONFIRMED, 1 CRITICAL PATTERN

**Top finding:** The beads RPC-first/CLI-fallback pattern is hand-written **58 times** across 24 files. Implementations vary in subtle ways: some use `WithAutoReconnect(3)`, some don't. Some check `beads.DefaultDir`, some skip it. Some set `BEADS_NO_DAEMON=1`, some don't. A `CLIClient` exists in `pkg/beads/cli_client.go` but ~40 call sites bypass it.

**Also found:**
- 6 structurally identical Event struct definitions for parsing events.jsonl
- 5 independent `~/.orch/events.jsonl` path constructions
- 4 copies of `git diff --name-only HEAD~5..HEAD` pattern
- 3 copies of `formatSessionTitle()`
- 4 duplicate Issue struct definitions with field-by-field conversion functions

**Verdict:** Scattered duplication is a real problem, especially the beads pattern. No single file is a hotspot, but the pattern's inconsistencies produce subtle behavioral differences.

### Blind Spot 5: Dead Features — CONFIRMED, ~7,900 LINES

**Confirmed dead (safe to remove):**
- `pkg/capacity/` (717 lines) — zero non-test imports, superseded by `pkg/account/`
- `pkg/shell/` (972 lines) — zero non-test imports, codebase uses `exec.Command` directly
- `pkg/certs/` (~5KB) — zero references, **private key in source control**
- `cmd/orch/orch` (21MB binary) — tracked in git due to `.gitignore` covering `/orch` not `cmd/orch/orch`
- `session.go.bak2` + `spawn_cmd.go.bak2` (4,187 lines) — tracked because `.gitignore` covers `*.bak` not `*.bak2`
- `pkg/daemon/completion.go` (308 lines) — only referenced in tests

**Likely dead:**
- OpenCode SSE subsystem (695 lines across 3 files) — disabled since Dec 2025
- `orch deploy` (521 lines) — superseded by `orch-dashboard` script
- `orch mode` (213 lines) — enforcement mechanism never implemented
- `orch guarded` (230 lines) — purely informational, no callers

**Verdict:** Dead features are a genuine blind spot. The 21MB binary in git and private key in source control are particularly concerning. Zero fix-density means the hotspot system doesn't flag these.

---

## Model Impact

- [x] **Extends** model with: Five concrete blind spot categories with evidence, ranked by severity

The hotspot system's four metrics (bloat-size, fix-density, investigation-cluster, coupling-cluster) are necessary but insufficient. Three blind spots cause real pain TODAY:

1. **Implicit coupling** (CRITICAL) — The skill-name string protocol and status-string filtering gap are existing bugs caused by undetected coupling. The Recommendation regex bug silently drops escalation recommendations.

2. **Semantic complexity** (HIGH) — `sync.Cond` patterns and bidirectional resolution logic in files under 800 lines will reliably produce subtly wrong agent code. The hotspot system sees these as small, well-factored files.

3. **Dead features** (HIGH) — 7,900+ lines of dead code + 21MB binary bloating context windows and git clones. Private key in source control is a security issue.

Two blind spots are real but lower severity:
4. **Cold spots** (MEDIUM) — Frozen infrastructure (opencode SSE, reconcile.go) may silently degrade
5. **Scattered duplication** (MEDIUM) — The beads RPC/CLI fallback pattern's inconsistencies cause behavioral differences across 58 call sites

---

## Notes

### Existing Bugs Found During This Investigation

1. **Blocked agents invisible on dashboard** — `serve_agents_discovery.go` doesn't include "blocked" status
2. **Recommendation regex can't match "spawn-follow-up"** — `\w+` doesn't match hyphens
3. **`beads.DefaultDir` not defer-restored in complete_pipeline.go** — error paths can target wrong project
4. **Private key tracked in source control** — `pkg/certs/key.pem`
5. **21MB compiled binary tracked in git** — `cmd/orch/orch`

### Detection Recommendations (ranked by ROI)

1. **String protocol coupling detector** — Scan for bare string literals used across >2 packages without shared constants. Would catch skill names, status strings, labels, phase strings.
2. **Semantic complexity scorer** — Count goroutine/channel/mutex/select density per function. Flag files with high concurrency complexity regardless of line count.
3. **Dead code scanner** — Check for packages with zero non-test imports. Check for git-tracked files matching `.gitignore` siblings.
4. **Cold spot detector** — Flag files with 0-1 commits in 28 days that are imported by files with >10 commits.
5. **Duplication pattern detector** — Find functions with similar structure across >3 files (structural clone detection).
