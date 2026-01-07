# Session Synthesis

**Agent:** og-arch-design-beads-integration-25dec
**Issue:** orch-go-7yrh.10
**Duration:** 2025-12-25 07:27 → 2025-12-25 08:45
**Outcome:** success

---

## TLDR

Designed beads integration strategy for orch-go to handle machine-speed interaction (50+ agents, 2-5s polling). Recommend native Go RPC client connecting directly to beads daemon socket, replacing CLI subprocess calls.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-25-inv-design-beads-integration-strategy-orch.md` - Architecture investigation with trade-off analysis

### Files Modified
- `.orch/features.json` - Will be updated with implementation features

### Commits
- (To be committed with this synthesis)

---

## Evidence (What Was Observed)

- Beads provides 25+ RPC operations via Unix domain socket (protocol.go:8-40)
- Reference RPC client exists in beads `internal/rpc/client.go` with connection management
- Current orch-go uses only 7 bd commands: ready, show, list, stats, comments, close, create
- Daemon race condition is startup-specific issue in beads repo, not architecture problem
- SQLite locking handles concurrent daemon operations (30s busy timeout default)

### Commands Run
```bash
# Get beads daemon info
bd info
# Output: Mode: direct, Issue Count: 1208

# Check beads schema
sqlite3 .beads/beads.db ".schema issues"
# SQLite 3.x database with proper indexes
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-inv-design-beads-integration-strategy-orch.md` - Full architecture analysis

### Decisions Made
- Decision 1: Native Go RPC client (Approach A) because it eliminates subprocess overhead and provides type safety
- Decision 2: Phased migration starting with high-frequency ops (ready, show) because 7-command surface is tractable
- Decision 3: Maintain CLI fallback because daemon may be unavailable (BEADS_NO_DAEMON mode)

### Constraints Discovered
- Beads `internal/rpc` package is internal - may need to copy or request export
- Socket path follows convention: `.beads/bd.sock`
- Daemon startup race is separate issue (beads repo bug)

### Externalized via `kn`
- (No kn entries needed - findings in investigation artifact)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Spawn Follow-up

**Issues to create:**

1. **Title:** Implement pkg/beads Go RPC client
   **Skill:** feature-impl
   **Priority:** high
   **Context:**
   ```
   Create pkg/beads/client.go with connection management to beads daemon socket.
   Pattern after beads internal/rpc/client.go. Include health checks, reconnect logic.
   Socket at .beads/bd.sock. Types from beads/internal/rpc/protocol.go.
   ```

2. **Title:** Migrate daemon.ListReadyIssues to beads RPC
   **Skill:** feature-impl
   **Priority:** high
   **Context:**
   ```
   Replace pkg/daemon/daemon.go:ListReadyIssues() subprocess call with pkg/beads client.
   Use OpReady operation. Fallback to CLI if daemon unavailable.
   ```

3. **Title:** Migrate verify.GetIssue to beads RPC  
   **Skill:** feature-impl
   **Priority:** medium
   **Context:**
   ```
   Replace pkg/verify/check.go:GetIssue() subprocess call with pkg/beads client.
   Use OpShow operation. Handle comments via OpCommentList.
   ```

4. **Title:** Migrate serve beads calls to RPC
   **Skill:** feature-impl
   **Priority:** medium
   **Context:**
   ```
   Replace cmd/orch/serve.go handleBeads() subprocess call with pkg/beads client.
   Use OpStats operation. Update BeadsAPIResponse to match.
   ```

---

## Unexplored Questions

**Questions that emerged during this session:**
- How should connection pooling work for pkg/beads client? (single vs per-request)
- Should beads maintainer be asked to export internal/rpc package?

**Areas worth exploring further:**
- Stress testing beads daemon with 100+ concurrent requests
- Connection lifecycle management (reconnect strategies)

**What remains unclear:**
- Daemon stability under heavy orch-go load (not stress-tested)
- Whether Go module import of beads works cleanly

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-beads-integration-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-design-beads-integration-strategy-orch.md`
**Beads:** `bd show orch-go-7yrh.10`
