# Probe: Agent Lifecycle State Model — Major Restructuring Drift

**Model:** agent-lifecycle-state-model
**Date:** 2026-02-20
**Status:** Complete

---

## Question

The model references 4 primary evidence files (lines 314-317):
- `cmd/orch/serve_agents.go` (~1400 lines) — Status calculation implementation
- `pkg/session/registry.go` — Session registry structure
- `pkg/verify/check.go` — Phase parsing from beads comments
- `.beads/issues.jsonl` — Canonical completion source

The model also references `~/.orch/registry.json` as a "fifth layer" (line 29) and describes a "Priority Cascade" for status calculation. With 334 commits since last update and multiple deleted files, **which model claims are still accurate, which are stale, and what new architecture has replaced the deleted components?**

---

## What I Tested

### 1. File existence verification
```bash
# Check which referenced files still exist
ls cmd/orch/serve_agents.go        # NOT FOUND — extracted into 8 files
ls pkg/session/registry.go          # NOT FOUND — deleted entirely
ls pkg/verify/check.go              # EXISTS — still present, greatly expanded
ls .beads/issues.jsonl               # EXISTS — still present

# What replaced serve_agents.go (commit a7b6b38df, 2026-02-18)
ls cmd/orch/serve_agents_*.go
# serve_agents_activity.go, serve_agents_cache.go, serve_agents_cache_handler.go,
# serve_agents_cache_test.go, serve_agents_discovery.go, serve_agents_events.go,
# serve_agents_gap.go, serve_agents_handlers.go, serve_agents_status.go,
# serve_agents_types.go (+ tests)
```

### 2. Registry removal verification
```bash
# Commit a9ec5cbf2 (2026-02-18): "feat: remove dead agent registry plumbing"
git show --stat a9ec5cbf2
# Deleted: pkg/session/registry.go (308 lines), pkg/session/registry_test.go (550 lines)
# Modified: spawn_cmd.go, complete_cmd.go, abandon_cmd.go, status_cmd.go, doctor.go — all de-registried

# Check for any remaining registry references in Go code
grep -r "registry.json" cmd/ pkg/ --include="*.go"
# ZERO RESULTS in cmd/ or pkg/ Go source files
```

### 3. Current architecture verification — `queryTrackedAgents()` in `cmd/orch/query_tracked.go`
```go
// The new single-pass query engine (replaces registry-based lookups)
func queryTrackedAgents(projectDirs []string) ([]AgentStatus, error) {
    // Step 1: Start from beads (source of truth for what work exists)
    issues, err := listTrackedIssues()
    // Step 2: Batch lookup workspace bindings (AGENT_MANIFEST.json)
    manifests, err := lookupManifestsAcrossProjects(projectDirs, beadsIDs)
    // Step 3: OpenCode liveness (session status)
    // Step 4: Join with explicit reason codes
}
```

### 4. Priority Cascade verification — `cmd/orch/serve_agents_status.go:53-104`
```go
func determineAgentStatus(issueClosed bool, phaseComplete bool, workspacePath string, sessionStatus string) string {
    // Priority 1: Beads issue closed → completed
    // Priority 2: Phase: Complete AND session dead → awaiting-cleanup
    // Priority 3: Phase: Complete → completed
    // Priority 4: SYNTHESIS.md exists AND session dead → awaiting-cleanup
    // Priority 5: SYNTHESIS.md exists → completed
    // Priority 6: Session activity → sessionStatus (fallback)
}
```

### 5. Two-Lane ADR verification — `.kb/decisions/2026-02-18-two-lane-agent-discovery.md`
```
Decision: Two-Lane Agent Discovery Architecture (accepted 2026-02-18)
Supersedes: registry-is-spawn-cache.md, registry-contract-spawn-cache-only.md

Deleted: ~/.orch/registry.json, ~/.orch/sessions.json, ~/.orch/state.db, pkg/session/registry.go
Prohibited: pkg/state/, pkg/registry/, pkg/cache/ for lifecycle state
Enforced by: architecture_lint_test.go (CI gate)
```

### 6. Architecture lint test verification
```go
// cmd/orch/architecture_lint_test.go
// Tests enforce: no new pkg/registry/, pkg/cache/ files
// Tests enforce: no forbidden imports from cmd/orch/
// Tests check: no stale files in ~/.orch/ (advisory)
```

### 7. pkg/session/ package evolution
```go
// pkg/session/session.go — NO longer contains registry
// Now contains: orchestrator session management (goal tracking, spawn recording, checkpoints)
// SpawnRecord has no Status field — "status is NOT stored here - derived at query time"
// GetSpawnStatuses() queries pkg/state.GetLiveness() at query time
```

### 8. verify/check.go evolution
```go
// Still exists but vastly expanded from "Phase parsing" to full verification suite
// Now contains 14 gate constants (GatePhaseComplete, GateSynthesis, GateGitDiff, etc.)
// VerifyCompletionFull() runs 10 verification gates
// Supports orchestrator tier (SESSION_HANDOFF.md instead of SYNTHESIS.md)
```

---

## What I Observed

### Model Claims That Still Hold (Confirmed)

1. **Four-layer state model is accurate** — Beads, workspace files, OpenCode sessions, tmux windows are still the four sources
2. **Beads is canonical for completion** — `determineAgentStatus()` checks `issueClosed` first (Priority 1)
3. **Phase: Complete is agent's declaration** — Only agents set Phase, checked via `ParsePhaseFromComments()`
4. **Session existence ≠ agent still working** — Confirmed by Priority Cascade fallback logic
5. **Status checks don't mutate state** — `determineAgentStatus()` is pure function
6. **State vs Infrastructure distinction** — Maintained and reinforced by two-lane architecture
7. **Priority Cascade** — Still exists, now expanded with `awaiting-cleanup` status and SYNTHESIS.md check

### Model Claims That Are Stale/Wrong (Contradicted)

1. **"Registry (~/.orch/registry.json) was a fifth layer"** (line 29) — **Registry no longer exists.** Deleted Feb 18, 2026 with architectural lint preventing recreation.

2. **"Registry demoted to metadata only"** (line 236, 258, 297) — **Wrong.** Registry was fully eliminated, not demoted. The two-lane ADR (Feb 18) supersedes the earlier "registry as spawn cache" decision.

3. **Primary Evidence section (lines 314-317) lists 3/4 wrong files:**
   - `cmd/orch/serve_agents.go` → Now extracted to `cmd/orch/serve_agents_*.go` (8+ files)
   - `pkg/session/registry.go` → **Deleted entirely.** Replaced by `cmd/orch/query_tracked.go` + workspace manifests
   - `pkg/verify/check.go` → Still exists but scope vastly expanded (14 verification gates, orchestrator support)

4. **"Why Registry Caused Drift" section (lines 224-237)** — Registry no longer exists. This section is historical context only. The architectural lint test (`architecture_lint_test.go`) now structurally prevents registry recreation.

5. **State Transitions diagram (lines 69-93)** — Shows "Registry entry created (Status: running)" as a spawn step. **Registry entries are no longer created.** Spawn now writes AGENT_MANIFEST.json to workspace instead.

6. **Failure Mode 2 (lines 141-152)** — References "session cleanup" triggering cascade to "dead" state. The `determineAgentStatus()` now handles this better with `awaiting-cleanup` status, distinguishing completed-but-orphaned from truly dead.

### New Architecture Not Covered by Model (Extends)

1. **Two-Lane Split** — Tracked work (orch status, beads-based) vs untracked sessions (orch sessions, OpenCode-based). The model doesn't describe this distinction.

2. **Single-Pass Query Engine** — `queryTrackedAgents()` replaces ad-hoc multi-source reconciliation. Uses `AgentStatus` struct with explicit reason codes (MissingBinding, MissingSession, SessionDead, MissingPhase).

3. **AGENT_MANIFEST.json** — Replaces registry entries as the workspace-local binding between beads_id, session_id, and project_dir. Written atomically during spawn.

4. **Architecture Lint Tests** — CI-level structural enforcement preventing lifecycle state package recreation (`architecture_lint_test.go`).

5. **Expanded Priority Cascade** — Now includes `awaiting-cleanup` status (Phase: Complete + session dead) and SYNTHESIS.md-based completion detection.

6. **Expanded Verification Suite** — `check.go` now has 14 gate types including git diff verification, accretion checks, build verification, visual verification, test evidence, and orchestrator-specific handoff validation.

7. **In-Memory Caching Only** — Per two-lane ADR, only process-local, short-TTL caching is allowed. Disk-backed persistent caches are prohibited.

---

## Model Impact

- [x] **Contradicts** invariants: Registry references throughout model are stale (registry no longer exists). State Transitions diagram shows registry step that no longer happens. Primary Evidence section lists 3/4 wrong files.
- [x] **Extends** model with: Two-lane architecture, single-pass query engine with reason codes, AGENT_MANIFEST.json binding, architecture lint enforcement, expanded Priority Cascade with awaiting-cleanup, expanded verification suite (14 gates).

**Verdict: CONTRADICTS + EXTENDS** — The core mechanism (four-layer model, beads as canonical, Priority Cascade) is confirmed, but the model has significant stale content (registry references, wrong file paths, outdated state transitions) and is missing the major Feb 18 architectural restructuring (two-lane split, query engine, lint enforcement).

**Recommended model update priority: HIGH** — The stale registry references and wrong file paths actively mislead any agent reading this model.

---

## Specific Update Recommendations

1. **Remove all registry references** — Lines 29, 224-237, 258, 297. Registry is structurally prohibited now.
2. **Update Primary Evidence** — Replace 3 stale file paths with current files
3. **Update State Transitions** — Remove "Registry entry created" step, add AGENT_MANIFEST.json write
4. **Add Two-Lane Architecture section** — Reference the ADR decision
5. **Update Priority Cascade** — Add awaiting-cleanup status and SYNTHESIS.md-based detection
6. **Update Failure Mode 2** — The cascade now handles this via awaiting-cleanup
7. **Add Evolution entry for Feb 18, 2026** — Major restructuring event
8. **Update References** — Add two-lane ADR, architecture lint test, query_tracked.go

---

## Notes

- The model's *core principles* are remarkably stable — four-layer model, beads canonical, Phase: Complete protocol, state vs infrastructure distinction all survived 334 commits unchanged.
- The *implementation details* underwent radical restructuring: registry eliminated, single-pass query engine built, architecture lint enforcement added.
- The model would benefit from separating "timeless principles" from "implementation details" to reduce future drift.
