# Session Synthesis

**Agent:** og-audit-architecture-audit-orch-06feb-297f
**Issue:** orch-go-21398
**Duration:** 2026-02-06 ~16:10 UTC
**Outcome:** success

---

## TLDR

Architecture audit of orch-go cmd/orch/ and pkg/ found 14 functions >200 lines, 5 copy-paste pattern clusters (39 git command instances, 13 daemon logging blocks, 5 HTTP header blocks), 10 files needing splitting, and 29 untestable functions with hard-coded dependencies. Highest-risk items: handleAgents (756 lines, dashboard reliability), daemon event logging (30% boilerplate), beads.DefaultDir global mutation (6 places).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-06-audit-architecture-audit-orch-go-codebase.md` - Complete architecture audit investigation with 4 findings, risk ratings, and implementation recommendations

### Files Modified
- None (audit-only session, no code changes)

### Commits
- (pending — investigation file to be committed)

---

## Evidence (What Was Observed)

- Go AST parser (`go/ast` + `go/parser`) confirmed 14 functions >200 lines with accurate line counts
- MD5-hashed 8-line sliding window analysis found 335 cross-file duplicates, clustering into 5 major patterns
- Cluster A (git commands): 39 `exec.Command("git", ...)` instances across 6 files in verify/ and complete_helpers
- Cluster B (HTTP headers): 5 identical Anthropic header blocks across account/, oauth/, usage/
- Cluster C (daemon logging): 13 copy-pasted event logging blocks in cmd/orch/daemon.go
- Cluster D (workdir resolution): 5 copies of workdir + beads.DefaultDir mutation across 5 cmd files
- Dependency analysis: 29 functions create `opencode.NewClient` inline, mutate `beads.DefaultDir`, or shell out without interfaces
- handleAgents at 756 lines is the single largest non-excluded function, doing session fetch + workspace cache + beads batch + enrichment + filtering + serialization in one HTTP handler

### Tests Run
```bash
# No code changes, no tests needed
# Verification via Go AST parsing and automated pattern analysis
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-06-audit-architecture-audit-orch-go-codebase.md` - Full architecture audit

### Decisions Made
- Risk rating uses Size x Coupling metric (not size alone): 637-line aggregateStats is medium-risk (pure computation) while 373-line runAbandon is high-risk (4 hard deps + global mutation)

### Constraints Discovered
- `beads.DefaultDir` is a global variable mutated in 6 places — this creates invisible coupling and race condition potential if any command handlers ever run concurrently
- Go brace-counting scripts for function size are unreliable — `{` and `}` inside string literals confuse them. Must use `go/ast` for accurate results.

### Externalized via `kn`
- (see Leave it Better section — kb quick commands to be run)

---

## Issues Created

Issues for discovered work to be created by orchestrator based on investigation findings. Top 5 recommendations:

1. Extract verify/git_helper.go — consolidate 39 git command instances (Cluster A)
2. Extract daemon logDaemonEvent() — remove 200+ lines of boilerplate from runDaemonLoop (Cluster C)
3. Extract Anthropic HTTP client helper — consolidate 5 header blocks (Cluster B)
4. Consolidate workdir resolution + beads.DefaultDir mutation (Cluster D)
5. Split handleAgents into phases (756 lines → testable subunits)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with all 4 audit targets)
- [x] Tests passing (audit-only, no code changes)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-21398`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should pkg/daemon/daemon.go (2451 lines, 54 functions) be split by domain? It has grown into a god package with cross-project, periodic tasks, lifecycle, and pool management all in one file.
- Should opencode.NewClient be replaced with an interface for testability? Affects 25 command handlers but would be a significant refactor.
- Are the verify/ git command copies intentionally different or accidental divergence? The sliding window found them identical at 8-line granularity, but there may be subtle differences in surrounding code.

**Areas worth exploring further:**
- Integration test coverage analysis — how much of the untestable code is actually covered by integration tests?
- Whether `beads.DefaultDir` can be replaced with explicit directory parameters (would require beads client interface change)

**What remains unclear:**
- Whether the recommended extractions (verify/git_helper, daemon logger) would pass `go test ./...` without behavioral changes — each extraction should be validated individually

---

## Session Metadata

**Skill:** codebase-audit
**Model:** opus
**Workspace:** `.orch/workspace/og-audit-architecture-audit-orch-06feb-297f/`
**Investigation:** `.kb/investigations/2026-02-06-audit-architecture-audit-orch-go-codebase.md`
**Beads:** `bd show orch-go-21398`
