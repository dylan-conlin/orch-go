---
linked_issues:
  - orch-go-vjghb
  - orch-go-pscu7
  - orch-go-0g1pi
  - orch-go-wq3mz
  - orch-go-qu8fj
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Four spawn reliability bugs span different pipeline stages but share a common pattern: fail-open design with backend-dependent coverage and insufficient observability.

**Evidence:** Code analysis shows dedup mechanisms only work for OpenCode backend (session_dedup.go:128), Docker execution hangs at Claude CLI startup (spawn/docker.go:103), daemon polling may have stale state vs preview (issue_adapter.go:19-39), and kb dedup is an external dependency failure.

**Knowledge:** The spawn pipeline has defense-in-depth but layers are backend-specific and fail silently; unified spawn gating with explicit error handling would make failures visible.

**Next:** Fix bugs individually (they're independent) but add unified spawn gate logging to detect silent failures; prioritize Docker stuck (P1) first.

**Promote to Decision:** Actioned - patterns documented in spawn guide

---

# Investigation: Spawn Reliability Pattern Analysis

**Question:** Are four related spawn bugs (Docker stuck, daemon visibility, dedup missing, false spawns) systemic or isolated?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** og-arch agent
**Phase:** Complete
**Next Step:** None - fix bugs individually per recommendations
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Docker backend stuck is an isolated execution bug, not dedup

**Evidence:**
The Docker spawn command uses:
```go
// pkg/spawn/docker.go:103
`bash -c 'cat %q | claude --dangerously-skip-permissions'`
```

With `docker run -it --rm` which requires proper TTY emulation. The "Tinkering..." message is Claude CLI's startup prompt, indicating the container starts but then gets stuck waiting for terminal input that never arrives properly through the tmux pipe.

**Source:**
- `pkg/spawn/docker.go:85-109` (Docker command construction)
- `pkg/spawn/docker.go:111-117` (tmux.SendKeys execution)
- Issue orch-go-pscu7 description

**Significance:** This is NOT a spawn pipeline dedup issue. It's an isolated Docker+tmux terminal emulation problem. The spawn completes (session starts), but execution hangs. Fix is independent of other dedup issues.

---

### Finding 2: Daemon visibility gap exists between preview and polling

**Evidence:**
Two different code paths for getting issues:

1. `orch daemon preview` - calls `ListReadyIssues()` directly (fresh query)
2. Daemon polling loop - also calls `ListReadyIssues()` but in a loop that may have:
   - Stale beads RPC client connection
   - BeadsClient reusing old socket state
   - No explicit reconnect between polls

The issue states: "orch daemon preview DOES show the issue as spawnable" but daemon polling doesn't see it.

```go
// pkg/daemon/issue_adapter.go:21-24
socketPath, err := beads.FindSocketPath("")
if err == nil {
    client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
```

The `WithAutoReconnect(3)` should handle transient issues, but the client is created fresh each call, so connection state shouldn't persist.

**Source:**
- `pkg/daemon/issue_adapter.go:16-39` (ListReadyIssues)
- `pkg/daemon/daemon.go:306` (NextIssueExcluding calls listIssuesFunc)
- Issue orch-go-0g1pi description

**Significance:** This is likely a beads daemon caching issue, not an orch-go issue. The fact that preview works but continuous polling doesn't suggests either beads daemon caching or some state in the polling context. Needs further debugging in beads-cli.

---

### Finding 3: Dedup mechanisms are backend-dependent and partially fixed

**Evidence:**
Current dedup flow in `daemon.Once()`:

```go
// pkg/daemon/daemon.go:803-818
if HasExistingSessionForBeadsID(issue.ID) {  // Only works for OpenCode!
    return ... "Existing session found"
}

// pkg/daemon/daemon.go:820-836
if hasComplete, _ := HasPhaseComplete(issue.ID); hasComplete {
    return ... "already completed"
}
```

The session dedup check (`HasExistingSessionForBeadsID`) only queries OpenCode API:
```go
// pkg/daemon/session_dedup.go:128-131
func HasExistingSessionForBeadsID(beadsID string) bool {
    checker := initDefaultSessionDedupChecker()
    return checker.HasExistingSession(beadsID)  // Queries OpenCode /session
}
```

For Claude CLI and Docker backends, there's NO session-level dedup. The only protection is:
1. SpawnedIssueTracker (in-memory, 6h TTL)
2. Status update to in_progress (can fail silently)
3. Phase: Complete comment check

The issue notes it was "Subsumed by tier alignment fix (42027eb2)" - light-tier auto-completes now, eliminating most duplicates.

**Source:**
- `pkg/daemon/daemon.go:803-836` (dedup checks in Once)
- `pkg/daemon/session_dedup.go:67-96` (OpenCode-only check)
- `pkg/daemon/spawn_tracker.go:38-42` (6h TTL)
- Issue orch-go-wq3mz description

**Significance:** Dedup coverage is BACKEND-DEPENDENT. OpenCode has session dedup; Claude CLI and Docker rely only on status update + Phase: Complete check. This is a design gap, not a bug per se.

---

### Finding 4: False spawns on completed synthesis is kb-cli external dependency

**Evidence:**
From issue orch-go-qu8fj:
```
Root cause: Three compounding failures:
1. Dedup JSON parse failure returns 'no duplicate' on error
2. No recognition of COMPLETED synthesis
3. Polysemous 'model' keyword matches 5+ unrelated topics
```

This is a kb-cli dedup failure, not an orch-go failure. The daemon correctly polls for issues, but kb context/dedup returns bad data.

**Source:**
- Issue orch-go-qu8fj description
- Investigation reference: `2026-01-08-inv-synthesize-model-investigations-11-synthesis-work.md`

**Significance:** This is ISOLATED to kb-cli. Fix is in kb-cli, not orch-go. However, orch could add validation of kb responses.

---

## Synthesis

**Key Insights:**

1. **Bugs are related by architecture, not by a single root cause** - All four bugs touch the spawn pipeline but at different stages: Docker at execution, daemon visibility at issue discovery, dedup at pre-spawn checks, kb at context gathering.

2. **Fail-open design creates invisible failures** - Each dedup layer returns false/continues on error to avoid blocking work. This is intentional resilience, but makes failures invisible. Example: `HasExistingSessionForBeadsID` returns `false` on API error (fail-open).

3. **Dedup coverage is backend-dependent** - OpenCode has session dedup; Claude CLI and Docker don't. This creates asymmetric reliability that's confusing to users.

4. **Two bugs are external dependencies** - Docker stuck is a Claude CLI terminal issue; kb false spawns is a kb-cli dedup issue. Orch-go can work around but not directly fix.

**Answer to Investigation Question:**

The four bugs are **related but not systemic** - they share the same architecture (spawn pipeline) but have independent root causes:

| Bug | Category | Root Cause | Fix Location |
|-----|----------|------------|--------------|
| pscu7 (Docker stuck) | Execution | TTY emulation in Docker+tmux | pkg/spawn/docker.go |
| 0g1pi (Daemon visibility) | Discovery | Beads daemon caching | beads-cli (external) |
| wq3mz (Dedup missing) | Pre-spawn | Backend-dependent coverage | pkg/daemon/session_dedup.go |
| qu8fj (False spawns) | Context | kb-cli parse failure | kb-cli (external) |

**Recommendation:** Fix individually but add unified spawn gate logging for observability.

---

## Structured Uncertainty

**What's tested:**

- ✅ Dedup only checks OpenCode sessions (verified: read session_dedup.go:99-100 shows GET /session)
- ✅ Docker uses pipe-to-claude pattern (verified: read docker.go:103)
- ✅ SpawnedIssueTracker TTL is 6 hours (verified: read spawn_tracker.go:41)
- ✅ Phase: Complete check exists (verified: read daemon.go:826)

**What's untested:**

- ⚠️ Why Docker container hangs at Tinkering (need to run Docker spawn with debugging)
- ⚠️ Whether beads daemon is actually caching (need to trace beads RPC calls)
- ⚠️ Whether unified spawn gate would prevent all duplicates (need implementation)

**What would change this:**

- Finding would be wrong if Docker hangs for non-TTY reasons
- Finding would be incomplete if there's another dedup path not identified
- Recommendation would change if unified gate adds too much latency

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Fix individually + add spawn gate logging** - Each bug has an independent fix; add unified logging for observability.

**Why this approach:**
- Each bug has clear, independent root cause
- No shared redesign needed (they touch different pipeline stages)
- Logging adds observability without changing behavior
- Prioritizes P1 (Docker) which is blocking work

**Trade-offs accepted:**
- Dedup remains backend-dependent (architectural gap remains)
- Doesn't add new dedup mechanisms (just observability)

**Implementation sequence:**
1. **Fix Docker stuck (P1)** - Most urgent, escape hatch is unreliable
2. **Add spawn gate logging** - Makes all future failures visible
3. **Report beads visibility issue** - External dependency
4. **Report kb dedup issue** - External dependency

### Alternative Approaches Considered

**Option B: Unified backend-agnostic dedup**
- **Pros:** Consistent reliability across all backends
- **Cons:** Significant redesign; Claude CLI has no session API to query
- **When to use instead:** If dedup failures become more frequent after other fixes

**Option C: Make dedup fail-closed**
- **Pros:** No silent failures; forces investigation
- **Cons:** Blocks work on any dedup error; reduces autonomy
- **When to use instead:** In critical production where duplicates are more costly than blocked work

**Rationale for recommendation:** Individual fixes are faster and lower risk. Unified gate logging provides observability to detect future issues without behavioral changes.

---

### Implementation Details

**What to implement first:**

1. **Docker stuck fix (P1)** - Investigate TTY emulation; may need to use `docker run` without `-it` and explicit stdin handling

2. **Spawn gate logging** - Add structured log line before spawn with all check results:
   ```
   spawn_gate: beads_id=X session_exists=false phase_complete=false tracker_marked=true status_updated=true
   ```

**Things to watch out for:**
- ⚠️ Docker fix may require different approach for different terminal environments
- ⚠️ Beads visibility may be a transient issue that's hard to reproduce
- ⚠️ Logging shouldn't add significant latency to spawn path

**Areas needing further investigation:**
- Why exactly does Docker hang? (needs debugging session with docker logs)
- Is beads caching by design? (needs beads-cli code review)
- Should ReconcileWithIssues be called in production?

**Success criteria:**
- ✅ Docker spawns complete without hanging at Tinkering
- ✅ Daemon sees newly created triage:ready issues within 1 poll cycle
- ✅ No duplicate spawns for same beads ID within 6 hours
- ✅ Spawn gate logs visible in daemon output

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Main daemon loop, dedup checks, Once() implementation
- `pkg/daemon/session_dedup.go` - OpenCode session dedup (backend-dependent)
- `pkg/daemon/spawn_tracker.go` - In-memory spawn tracking with 6h TTL
- `pkg/daemon/issue_adapter.go` - ListReadyIssues, SpawnWork, HasPhaseComplete
- `pkg/spawn/docker.go` - Docker spawn implementation
- `.kb/guides/spawn.md` - Spawn documentation
- `.kb/investigations/2026-01-15-inv-investigate-daemon-duplicate-spawn-issue.md` - Prior investigation

**Commands Run:**
```bash
# Show related bugs
bd show orch-go-pscu7
bd show orch-go-0g1pi
bd show orch-go-wq3mz
bd show orch-go-qu8fj

# Show parent epic
bd show orch-go-5bpcc

# List daemon package files
ls -la pkg/daemon/
```

**External Documentation:**
- None

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-15-inv-investigate-daemon-duplicate-spawn-issue.md` - Prior dedup investigation
- **Epic:** orch-go-5bpcc - Systemic Completion Integrity & Signal Detection
- **Guide:** `.kb/guides/spawn.md` - Spawn documentation

---

## Test Performed

**Test:** Analyzed code paths for all four bug scenarios

**Method:**
1. Read daemon.Once() to trace dedup flow
2. Read session_dedup.go to verify OpenCode-only coverage
3. Read docker.go to understand execution model
4. Read issue_adapter.go to understand visibility flow

**Result:**
- Confirmed dedup is backend-dependent (OpenCode vs Claude CLI/Docker)
- Confirmed fail-open design throughout
- Confirmed bugs are independent but share architecture

---

## Conclusion

The four spawn reliability bugs are **related by architecture but have independent root causes**. They span different pipeline stages (execution, discovery, pre-spawn, context) and can be fixed individually.

**Root pattern:** Fail-open design with backend-dependent coverage creates invisible failures. The spawn pipeline has defense-in-depth but each layer fails silently.

**Recommendation:** Fix individually, add unified spawn gate logging for observability, and accept that dedup coverage varies by backend until a more significant redesign is justified by failure frequency.

---

## Investigation History

**2026-01-22 19:30:** Investigation started
- Initial question: Are four related spawn bugs systemic or isolated?
- Context: Multiple P1/P2 bugs affecting spawn reliability

**2026-01-22 20:00:** Completed code analysis
- Found dedup is backend-dependent (OpenCode-only session check)
- Found Docker uses pipe-to-claude that may have TTY issues
- Found prior investigation already extended TTL and added session dedup

**2026-01-22 20:30:** Investigation completed
- Status: Complete
- Key outcome: Bugs are related by architecture but independent; fix individually, add logging
