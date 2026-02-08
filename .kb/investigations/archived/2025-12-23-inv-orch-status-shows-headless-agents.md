<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode agents were incorrectly marked as phantom because the code checked beads issue status instead of recognizing that having an OpenCode session means the agent is running.

**Evidence:** `orch status --json` showed `is_phantom: true` AND `is_processing: true` for the same agent - logically impossible.

**Knowledge:** Phantom means "beads issue open but agent not running"; OpenCode agents have running sessions by definition, so they should never be phantom.

**Next:** Fix deployed and verified; headless agents now correctly show "running" or "idle" based on IsProcessing().

**Confidence:** High (95%) - fix tested with live headless agents, all showing correct status.

---

# Investigation: Orch Status Shows Headless Agents as Phantom

**Question:** Why does orch status show headless agents as 'phantom' instead of 'active/idle'?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: OpenCode agents set isPhantom based on beads issue existence

**Evidence:** In `cmd/orch/main.go:1940-1942`:
```go
// Check if beads issue is open (determines phantom status)
issue, issueExists := openIssues[oa.beadsID]
isPhantom := !issueExists // If issue is not in open list, it might be phantom
```

**Source:** cmd/orch/main.go:1940-1942

**Significance:** This logic is incorrect for OpenCode agents. The check assumes an agent without an open beads issue is "phantom", but OpenCode agents have running sessions - they ARE running regardless of beads status.

---

### Finding 2: Phantom definition contradicts OpenCode agent behavior

**Evidence:** From AgentInfo struct comment:
```go
IsPhantom    bool   `json:"is_phantom,omitempty"`    // True if beads issue open but agent not running
```

**Source:** cmd/orch/main.go:1755

**Significance:** Phantom means "beads issue open but agent NOT running". OpenCode agents HAVE running sessions, so by definition they cannot be phantom.

---

### Finding 3: IsProcessing was being set but overridden by phantom status

**Evidence:** From `orch status --json` output before fix:
```json
{
  "session_id": "ses_4b142aa3fffeS50lP04OQwue7m",
  "beads_id": "orch-go-2dkn",
  "is_phantom": true,
  "is_processing": true
}
```

**Source:** Running `orch status --all --json | jq '.agents[] | select(.session_id != "tmux-stalled")'`

**Significance:** Having both `is_phantom: true` AND `is_processing: true` is logically impossible - confirms the bug. The status display logic (lines 2185-2192) shows phantom if `IsPhantom` is true, even when `IsProcessing` is also true.

---

## Synthesis

**Key Insights:**

1. **Phantom only applies to stale references** - Phantom agents are tmux windows without corresponding OpenCode sessions. OpenCode agents always have sessions, so they're never phantom.

2. **IsProcessing was correctly implemented but masked** - The commit 8e52211 correctly added `IsSessionProcessing()`, but the phantom status override prevented it from affecting the displayed status.

3. **Simple fix: OpenCode agents always non-phantom** - Setting `isPhantom := false` for all OpenCode agents correctly reflects that they are running.

**Answer to Investigation Question:**

Headless agents showed as 'phantom' because the code incorrectly set `isPhantom` based on beads issue existence rather than recognizing that OpenCode agents are inherently running (they have active sessions). The fix sets `isPhantom = false` for all OpenCode agents since they are running by definition.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Fix has been deployed and tested with live headless agents showing correct status (running/idle).

**What's certain:**

- ✅ OpenCode agents now show "running" when processing, "idle" when complete
- ✅ Swarm counts correctly show active/processing/idle breakdown
- ✅ Default view (without --all) shows only active agents, hides phantoms

**What's uncertain:**

- ⚠️ Edge case where OpenCode session exists but is truly stale (>30 min idle)
- ⚠️ Interaction with tmux-based agents that also have OpenCode sessions

**What would increase confidence to Very High (100%):**

- Test overnight daemon run with multiple headless agents
- Verify behavior after agent exits (session becomes historical)

---

## Implementation

**Fix applied:** Changed `cmd/orch/main.go:1940-1948`

Before:
```go
// Check if beads issue is open (determines phantom status)
issue, issueExists := openIssues[oa.beadsID]
isPhantom := !issueExists // If issue is not in open list, it might be phantom
```

After:
```go
// OpenCode agents are NOT phantom because they have a running session.
// Phantom means "beads issue open but agent not running" - but these agents ARE running.
isPhantom := false

// Get issue for task info
issue := openIssues[oa.beadsID]
```

**Verification:**
```bash
$ orch status
SWARM STATUS: Active: 4 (running: 1, idle: 3), Phantom: 42 (use --all to show)

AGENTS
  BEADS ID           STATUS   PHASE        TASK                                SKILL              RUNTIME
  ---------------------------------------------------------------------------------------------------------
  orch-go-48bi       running  Planning     -                                   feature-impl       2m 40s
  orch-go-untracked-1766552655 idle     -            -                                   -                  7m 21s
  orch-go-2dkn       idle     Complete     -                                   feature-impl       8m 33s
  skillc-uh8         idle     -            -                                   feature-impl       18m 30s
```

---

## References

**Files Examined:**
- cmd/orch/main.go:1766-2037 - runStatus() function with agent classification
- cmd/orch/main.go:1724-1764 - AgentInfo and SwarmStatus structs
- pkg/opencode/client.go:315-345 - IsSessionProcessing() implementation

**Commands Run:**
```bash
# Check agents before fix
orch status --all --json | jq '.agents[] | select(.session_id != "tmux-stalled")'

# Check beads issue status
bd show orch-go-2dkn --json | jq '.[0].status'

# Verify fix
orch status  # Shows running/idle instead of phantom
```

**Related Artifacts:**
- **Commit:** 8e52211 - Added IsSessionProcessing() but result wasn't used correctly
