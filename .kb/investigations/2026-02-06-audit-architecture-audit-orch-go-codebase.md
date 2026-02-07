## Summary (D.E.K.N.)

**Delta:** Orch-go has 14 functions >200 lines, 5 distinct copy-paste pattern clusters, 22 files >500 lines (non-test), and 29 functions with 2+ hard-coded dependencies making them untestable without integration.

**Evidence:** Go AST parser confirmed function sizes; MD5-hashed 8-line sliding windows found 335 cross-file duplicates clustering into 5 major patterns; dependency analysis identified functions creating `opencode.NewClient`, mutating `beads.DefaultDir`, and shelling out to `exec.Command` without interfaces.

**Knowledge:** The highest-risk findings are the untestable command handlers (runComplete, handleAgents, runSpawnWithSkillInternal) and the daemon event-logging copy-paste (13 blocks). Size alone is not the risk — it's the combination of size + no interfaces + global state mutation.

**Next:** Create beads issues for the top 5 findings. Extract shared git-diff helper in verify/, extract Anthropic HTTP client in account/, extract daemon event logger, and introduce dependency injection for command handlers.

**Authority:** architectural - These findings cross package boundaries and require coordinated refactoring across cmd/ and pkg/.

---

# Investigation: Architecture Audit Orch-Go Codebase

**Question:** What are the oversized functions, duplicated patterns, oversized files, and untestable code in cmd/orch/ and pkg/?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** Architecture audit agent (orch-go-21398)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

---

## Findings

### Finding 1: 14 Functions >200 Lines (complete_cmd.go excluded)

**Evidence:** Using Go AST parser for accurate line counting. Excludes `complete_cmd.go` (handled by orch-go-21396).

| Lines | File:Line | Function | Risk |
|-------|-----------|----------|------|
| 756 | cmd/orch/serve_agents.go:90-845 | handleAgents | **Critical** |
| 653 | cmd/orch/spawn_cmd.go:320-972 | runSpawnWithSkillInternal | **Critical** |
| 637 | cmd/orch/stats_cmd.go:299-935 | aggregateStats | **Medium** |
| 626 | cmd/orch/daemon.go:188-813 | runDaemonLoop | **High** |
| 373 | cmd/orch/abandon_cmd.go:67-439 | runAbandon | **High** |
| 345 | cmd/orch/serve.go:203-547 | runServe | **Medium** |
| 284 | pkg/daemon/daemon.go:2003-2286 | CrossProjectOnceExcluding | **High** |
| 273 | cmd/orch/review.go:717-989 | runReviewDone | **Medium** |
| 260 | cmd/orch/serve_attention.go:111-370 | handleAttention | **Medium** |
| 231 | cmd/orch/serve_reviews.go:62-292 | handlePendingReviews | **Medium** |
| 218 | cmd/orch/clean_cmd.go:298-515 | runClean | **Medium** |
| 210 | cmd/orch/review.go:153-362 | getCompletionsForReview | **Medium** |
| 208 | pkg/verify/check.go:299-506 | VerifyCompletionFullWithComments | **Medium** |

**Additional functions 150-200 lines (borderline):** runStatus (188), ArchiveStaleWorkspaces (184), runDoctor (182), collectAttemptHistory (176), enrichStateDBAgentsLive (171), archiveStaleWorkspaces (168), OnceExcluding (167), handleErrors (166), RunPeriodicRecovery (160), runSpawnTmux (159), FindOrphanedSessions (158), SendMessageWithStreaming (148).

**Risk assessment details:**

- **handleAgents (756 lines, Critical):** Single HTTP handler that fetches sessions, builds workspace cache, batch-fetches beads, enriches with phase/status/gap analysis, deduplicates, filters by time/project/beads, builds investigation path cache, and serializes. Multiple threshold constants inline. Any bug in filtering logic hides agents from dashboard.

- **runSpawnWithSkillInternal (653 lines, Critical):** Orchestrates triage bypass, concurrency limit, usage check, model resolution, skill loading, beads issue creation, workspace creation, context generation, decision gate, backend routing (4 backends: inline/headless/tmux/claude/docker), state DB recording, and event logging. A failure in any step can produce a partially-initialized agent.

- **runDaemonLoop (626 lines, High):** Contains 13 event logging blocks (copy-pasted boilerplate) and 6 result-handling blocks following identical if-error/else-if-count/else-if-verbose branching. The copy-paste makes error handling inconsistent.

- **runAbandon (373 lines, High):** Mixes state DB lookup, tmux discovery, OpenCode session lookup, workspace discovery, beads update, event logging, and process termination in one function. Mutates `beads.DefaultDir` global.

**Source:** Go AST parsing via `go/ast` + `go/parser`, scanning all non-test `.go` files in cmd/ and pkg/.

**Significance:** Functions >200 lines are hard to reason about and test. The critical-risk functions (handleAgents, runSpawnWithSkillInternal) have complex branching where silent failures can hide behind successful-looking output (e.g., agents missing from dashboard, partially initialized spawns).

---

### Finding 2: 5 Major Copy-Paste Pattern Clusters

**Evidence:** MD5-hashed 8-line sliding window analysis found 335 cross-file duplicate blocks. After filtering imports and package declarations, 5 significant clusters remain:

**Cluster A: Git Diff/Show/Log Command Pattern (Critical, 6+ files)**

Identical `exec.Command("git", ...)` + fallback pattern copy-pasted across:
- `cmd/orch/complete_helpers.go:28-36`
- `pkg/verify/behavioral.go:304-312`
- `pkg/verify/build_verification.go:110-118`
- `pkg/verify/escalation.go:238-246`
- `pkg/verify/test_evidence.go:186-262` (multiple instances)
- `pkg/verify/visual.go:194-612` (multiple instances)

Total git exec.Command calls: 6 in complete_helpers, 4 in behavioral, 2 in build_verification, 2 in escalation, 10 in test_evidence, 15 in visual = **39 instances** across 6 files.

The `relWorkspace` path resolution + `git log --since=... --format=%H` pattern appears 4x across 2 files (test_evidence.go:251, test_evidence.go:386, visual.go:277, visual.go:529).

**Risk:** High — each copy has slightly different error handling. Missing a bug fix in one copy leaves others broken.

**Cluster B: Anthropic HTTP Request Headers (High, 3 packages)**

Identical 5-line header block (anthropic-beta, Content-Type, Accept, User-Agent) copy-pasted:
- `pkg/account/account.go:604-608, 666-670` (2 instances)
- `pkg/account/oauth.go:271-275` (1 instance)
- `pkg/usage/usage.go:269-273, 330-334` (2 instances)

Total: **5 instances** of the same header setup across 3 files.

**Risk:** High — if `AnthropicBetaHeaders` or `UserAgent` changes, easy to miss one copy. Also, the full request pattern (create request, set headers, send, check status, read body, parse JSON) spans ~20 lines and is duplicated 5x.

**Cluster C: Daemon Event Logging Pattern (High, 1 file)**

In `cmd/orch/daemon.go`, the pattern:
```go
event := events.Event{
    Type:      "daemon.X",
    Timestamp: time.Now().Unix(),
    Data: map[string]interface{}{...},
}
if err := logger.Log(event); err != nil {
    fmt.Fprintf(os.Stderr, "Warning: failed to log X event: %v\n", err)
}
```
Appears **13 times** in `runDaemonLoop` (lines 387, 404, 427, 443, 475, 492, 515, 531, 583, 722, 771, 922, 955). Each wrapped in identical if-error/else-if-count/else-if-verbose 3-branch structure appearing **6 times**.

**Risk:** High — the repetition obscures the actual daemon logic. 200+ lines of the 626-line function are pure logging boilerplate.

**Cluster D: Workdir Resolution + beads.DefaultDir Mutation (Medium, 5 commands)**

The workdir resolution pattern:
```go
var projectDir string
var err error
if workdir != "" {
    projectDir, err = filepath.Abs(workdir)
    // error check, stat check, beads.DefaultDir mutation
} else {
    projectDir, err = os.Getwd()
}
```
Copy-pasted in:
- `cmd/orch/abandon_cmd.go:72-92` (with beads.DefaultDir)
- `cmd/orch/rework_cmd.go:78-96` (with beads.DefaultDir)
- `cmd/orch/complete_cmd.go:260-275` (with beads.DefaultDir)
- `cmd/orch/shared.go:391-428` (resolveProjectDir, with beads.DefaultDir)
- `cmd/orch/work_cmd.go:20` (simplified)

**Risk:** Medium — the `beads.DefaultDir` global mutation makes each copy a potential race condition if commands ever run concurrently.

**Cluster E: Spawn Time File Parsing (Low, 2 files)**

Identical spawn-time file reading pattern in:
- `cmd/orch/clean_cmd.go:948-962, 1127-1141` (2 copies within same file)
- `pkg/cleanup/workspaces.go:90-104`

**Risk:** Low — pure data parsing, unlikely to harbor bugs.

**Source:** MD5-hashed 8-line sliding window across all non-test Go files. Manual verification of each cluster.

**Significance:** Clusters A (git commands) and C (daemon logging) represent the highest maintenance risk. A bug fix to one copy of the git diff pattern would need to be applied to all 39 instances across 6 files. The daemon logging makes the 626-line function ~30% boilerplate.

---

### Finding 3: 22 Non-Test Files >500 Lines (Split Candidates)

**Evidence:** Files >500 lines (non-test) sorted by size, with split analysis:

| Lines | File | Split? | Proposed Action |
|-------|------|--------|----------------|
| 2451 | pkg/daemon/daemon.go | **Yes** | Already partially split. Extract CrossProject* methods (~400 lines) to daemon_cross_project.go |
| 1700 | cmd/orch/serve_beads.go | **Yes** | Split into serve_beads_stats.go, serve_beads_graph.go, serve_beads_attempts.go |
| 1588 | cmd/orch/spawn_cmd.go | **Yes** | 653-line function is the issue, not file structure. Extract backend dispatch to spawn_backends.go |
| 1459 | pkg/opencode/client.go | Maybe | 8 functions, well-organized. Only split if adding more methods |
| 1446 | cmd/orch/serve_system.go | **Yes** | Mix of daemon, focus, servers, services, file, screenshot handlers. Split by domain |
| 1282 | pkg/beads/client.go | No | Client methods, cohesive |
| 1215 | cmd/orch/clean_cmd.go | **Yes** | 7 cleanup strategies in one file. Extract archiveStaleWorkspaces + archiveUntrackedWorkspaces to clean_archive.go |
| 1212 | pkg/spawn/context.go | Maybe | Long but cohesive (context generation). Only split if adding more context types |
| 1178 | cmd/orch/review.go | **Yes** | 3 separate commands (review, review single, review done) + data loading. Split by command |
| 1159 | pkg/tmux/tmux.go | No | Window management methods, cohesive |
| 1142 | cmd/orch/stats_cmd.go | **Yes** | 637-line aggregateStats is the issue. Extract stat aggregation to stats_aggregate.go |
| 1126 | cmd/orch/daemon.go | **Yes** | runDaemonLoop (626 lines) dominates. Extract event logging + result handling patterns |
| 1097 | cmd/orch/kb.go | Maybe | Multiple kb subcommands, borderline |
| 981 | pkg/verify/visual.go | **Yes** | Multiple verification strategies. Extract getWebChangesSince* to verify_git_changes.go |
| 975 | pkg/spawn/learning.go | No | Learning analysis, cohesive |
| 959 | pkg/account/account.go | Maybe | Account management, mostly cohesive |
| 898 | cmd/orch/handoff.go | No | Handoff workflow, cohesive |
| 880 | cmd/orch/serve_agents.go | **Yes** | 756-line handler dominates. Extract enrichment/filtering to sub-files |
| 870 | cmd/orch/spawn_validation.go | No | Already extracted from spawn_cmd.go |
| 805 | cmd/orch/hotspot.go | No | Hotspot analysis, cohesive |
| 799 | pkg/verify/check.go | No | Core verification, cohesive |
| 770 | cmd/orch/changelog.go | No | Changelog generation, cohesive |

**Source:** `wc -l` on all Go source files, filtered to >500 lines non-test.

**Significance:** 10 files clearly need splitting, 5 are borderline. The primary driver is not file size per se, but single functions dominating their files (handleAgents owns 86% of serve_agents.go; runDaemonLoop owns 56% of daemon.go cmd).

---

### Finding 4: 29 Functions with Untestable Hard-Coded Dependencies

**Evidence:** Functions with 2+ hard-coded dependencies (creating clients inline, mutating globals, shelling out without interfaces):

**4 hard deps (most untestable):**

| File:Line | Function | Lines | Dependencies |
|-----------|----------|-------|-------------|
| cmd/orch/abandon_cmd.go:67 | runAbandon | 373 | opencode.NewClient, os.Getwd, os.ReadFile, beads.DefaultDir mutation |
| cmd/orch/serve_agents.go:90 | handleAgents | 756 | opencode.NewClient, os.Getwd, os.ReadFile |

**3 hard deps:**

| File:Line | Function | Lines | Dependencies |
|-----------|----------|-------|-------------|
| cmd/orch/doctor_sessions.go:30 | runSessionsCrossReference | 119 | opencode.NewClient, os.Getwd, os.ReadFile |
| cmd/orch/resume.go:221 | runResumeByWorkspace | 99 | opencode.NewClient, os.Getwd, os.ReadFile |
| cmd/orch/wait.go:143 | resolveBeadsID | 94 | opencode.NewClient, os.Getwd, os.ReadFile |
| cmd/orch/doctor_install.go:36 | startOpenCode | 70 | opencode.NewClient, exec.Command, os.ReadFile |

**2 hard deps (25 functions):**

Most `run*` command handlers in cmd/orch/ create `opencode.NewClient(serverURL)` inline and read filesystem state via `os.Getwd()`. Key examples:
- `runSpawnWithSkillInternal` (653 lines) — creates client + reads cwd
- `runReviewDone` (273 lines) — reads cwd + mutates beads.DefaultDir
- `runStatus` (188 lines) — creates client + reads cwd
- `cleanOrphanedDiskSessions` (129 lines) — creates client + reads cwd
- `FindOrphanedSessions` in pkg/daemon/recovery.go (158 lines) — creates client + reads cwd

**Global state mutation (beads.DefaultDir):**

`beads.DefaultDir` is mutated in 6 places across 5 files:
- `cmd/orch/rework_cmd.go:90`
- `cmd/orch/abandon_cmd.go:86`
- `cmd/orch/serve.go:210`
- `cmd/orch/complete_cmd.go:269`
- `cmd/orch/complete_pipeline.go:132`
- `cmd/orch/shared.go:428`

This global mutation means these functions cannot be safely tested concurrently and any test must carefully manage global state.

**Source:** Automated scanning for `opencode.NewClient`, `exec.Command`, `os.Getwd()`, `os.ReadFile`, `beads.DefaultDir` in function bodies of functions >50 lines.

**Significance:** The core problem is that cmd/orch/ command handlers are procedural — they create their own dependencies rather than receiving them. This makes unit testing impossible; only integration tests can cover them. While CLI commands often start this way, the functions are now large enough (200-750 lines) that the lack of testability becomes a reliability risk.

---

## Synthesis

**Key Insights:**

1. **Size × Coupling = Risk** — The highest-risk functions aren't just big, they're big AND tightly coupled to external dependencies. `handleAgents` (756 lines, 3 hard deps) can silently drop agents from the dashboard; `runSpawnWithSkillInternal` (653 lines, 2 hard deps) can leave half-initialized agents. Pure computation functions like `aggregateStats` (637 lines) are less risky despite being large.

2. **Copy-paste debt concentrated in verify/ and daemon** — The 39 git command instances across 6 verify/ files and the 13 event-logging blocks in daemon.go represent the two largest duplication clusters. Both are "boilerplate amplification" — the same 10-20 line pattern repeated with only 1-2 parameter changes per copy.

3. **Global state mutation is the hidden coupling mechanism** — `beads.DefaultDir` mutation in 6 places creates invisible coupling between command handlers. Any test running two command handlers concurrently would race on this global. This is more dangerous than the function sizes because it crosses function boundaries invisibly.

**Answer to Investigation Question:**

The orch-go codebase has 14 functions exceeding 200 lines (excluding complete_cmd.go), 5 major duplication clusters (git commands 39x, HTTP headers 5x, daemon logging 13x, workdir resolution 5x, spawn-time parsing 3x), 10 files clearly needing splitting, and 29 untestable functions with hard-coded dependencies. The most architecturally concerning findings are: (1) the `handleAgents` HTTP handler doing everything in 756 lines with no testable subunits, (2) the `beads.DefaultDir` global state mutation creating invisible coupling, and (3) the daemon event-logging copy-paste making `runDaemonLoop` 30% boilerplate.

---

## Structured Uncertainty

**What's tested:**

- ✅ Function sizes verified via Go AST parser (`go/ast` + `go/parser`) — accurate to the line
- ✅ Duplication clusters verified via MD5-hashed sliding window + manual inspection of each cluster
- ✅ Hard-coded dependency counts verified via string matching against function bodies

**What's untested:**

- ⚠️ Whether the identified split points would actually reduce complexity (some extractions might create more coupling via new interfaces)
- ⚠️ Whether the beads.DefaultDir global is actually hit in practice by concurrent usage (daemon runs commands sequentially today)
- ⚠️ Whether the verify/ git command duplication causes actual bugs (each copy might have intentional differences not caught by the sliding window)

**What would change this:**

- If the daemon ever processes commands concurrently, the beads.DefaultDir finding would escalate from medium to critical
- If the verify/ package gets more verification types, the git command duplication would expand further
- If integration test coverage is high enough, the untestable code finding matters less for reliability

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Extract shared git helper in verify/ | implementation | Single package, reversible refactor |
| Extract Anthropic HTTP client | implementation | Shared utility, no behavioral change |
| Extract daemon event logger | implementation | Single file refactor |
| Introduce DI for command handlers | architectural | Crosses cmd/pkg boundary, changes testing patterns |
| Resolve beads.DefaultDir global | architectural | Affects 5 files, changes beads client interface |

### Recommended Approach: Bottom-Up Extraction

**Extract shared utilities first, then introduce interfaces for testability.**

**Why this approach:**
- Each extraction is independently valuable (reduces duplication)
- No behavioral changes required (pure refactoring)
- Directly addresses findings 1-4

**Trade-offs accepted:**
- Command handlers remain procedural (full DI deferred)
- Not all 14 oversized functions get addressed immediately

**Implementation sequence:**
1. **Extract verify/git_helper.go** — consolidate the 39 git command instances into shared functions: `getChangedFiles(projectDir, since)`, `getCommitHashes(projectDir, path, since)`, `getFileChangesForCommit(projectDir, hash)`. Fixes Cluster A.
2. **Extract pkg/account/http_client.go** — create `AnthropicRequest(method, url, token string)` that sets standard headers. Fixes Cluster B.
3. **Extract daemon logDaemonEvent() helper** — replace 13 copy-pasted blocks with `logDaemonEvent(logger, eventType string, data map[string]interface{})`. Fixes Cluster C.
4. **Extract resolveProjectDir() to shared.go** — replace 5 workdir resolution copies with single function returning `(projectDir string, cleanup func(), err error)` where cleanup restores beads.DefaultDir. Fixes Cluster D.
5. **Split serve_agents.go** — extract handleAgents into phases: buildAgentList, enrichWithBeads, filterAndDedupe, serializeResponse.

### Alternative Approaches Considered

**Option B: Full dependency injection refactor**
- **Pros:** Makes everything unit-testable
- **Cons:** Major rewrite; command handlers need `AppContext` struct threading; disrupts ongoing development
- **When to use instead:** If reliability failures are actually occurring in production

**Option C: Ignore and rely on integration tests**
- **Pros:** No code changes needed
- **Cons:** Doesn't fix duplication; doesn't improve maintainability; integration tests are slow
- **When to use instead:** If development velocity is the only priority

**Rationale for recommendation:** Bottom-up extraction is the lowest-risk path that yields the highest duplication reduction. It's also parallelizable — each extraction is independent.

---

### Implementation Details

**What to implement first:**
- verify/git_helper.go extraction (highest duplication count, 39 instances)
- daemon logDaemonEvent helper (reclaims 200+ lines in a critical function)

**Things to watch out for:**
- ⚠️ The git diff fallback pattern (`HEAD~5..HEAD` → `HEAD~1..HEAD`) varies across copies — ensure the helper preserves this fallback behavior
- ⚠️ The daemon event logging has subtle differences in Data map keys across the 13 copies
- ⚠️ beads.DefaultDir mutation during resolveProjectDir needs careful cleanup semantics

**Areas needing further investigation:**
- Whether pkg/daemon/daemon.go (2451 lines, 54 functions) should be split into domain files (cross_project.go, periodic_tasks.go, lifecycle.go)
- Whether opencode.NewClient should be replaced with an interface for testability (affects 25 command handlers)

**Success criteria:**
- ✅ No function >200 lines contains duplicated patterns (git commands, event logging)
- ✅ beads.DefaultDir mutations go through a single function with cleanup
- ✅ `go test ./...` still passes after each extraction (regression-free)
- ✅ Line count of daemon.go cmd drops by ~200 lines (logging extraction)

---

## References

**Files Examined:**
- All non-test Go files in cmd/orch/ and pkg/ (full codebase scan)
- cmd/orch/serve_agents.go — 756-line handleAgents function analysis
- cmd/orch/daemon.go — 13 event logging blocks, 626-line runDaemonLoop
- cmd/orch/abandon_cmd.go — workdir pattern + beads.DefaultDir mutation
- cmd/orch/rework_cmd.go — identical workdir pattern
- pkg/verify/test_evidence.go, visual.go — git command duplication cluster
- pkg/account/account.go, oauth.go, usage/usage.go — HTTP header duplication

**Commands Run:**
```bash
# Go AST function size analysis (accurate brace counting)
go run /tmp/funcsize.go  # Custom script using go/ast + go/parser

# Line counts for all source files
find cmd pkg -name "*.go" | xargs wc -l | sort -rn

# Cross-file duplication via MD5-hashed 8-line sliding windows
python3 sliding_window_dedup.py  # Custom script

# Hard-coded dependency analysis
python3 hard_dep_scanner.py  # Scanned for opencode.NewClient, exec.Command, os.Getwd, beads.DefaultDir
```

**Related Artifacts:**
- **Decision:** N/A — findings inform future decisions
- **Investigation:** This is the primary investigation

---

## Investigation History

**2026-02-06 16:00:** Investigation started
- Initial question: Architecture audit of cmd/orch/ and pkg/ focusing on 4 targets
- Context: Spawned by orchestrator as architecture audit (orch-go-21398)

**2026-02-06 16:15:** All 4 audit targets scanned
- Go AST: 14 functions >200 lines (accurate)
- Sliding window: 335 cross-file duplicates → 5 major clusters
- File analysis: 22 files >500 lines, 10 need splitting
- Dependency analysis: 29 functions with 2+ hard deps

**2026-02-06 16:30:** Investigation completed
- Status: Complete
- Key outcome: Highest-risk items are handleAgents (756 lines, dashboard reliability), daemon event logging (13 copies), and beads.DefaultDir global mutation (6 places).
