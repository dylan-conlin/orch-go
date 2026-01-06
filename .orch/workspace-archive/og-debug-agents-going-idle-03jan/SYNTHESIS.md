# Session Synthesis

**Agent:** og-debug-agents-going-idle-03jan
**Issue:** orch-go-rzch
**Duration:** 2026-01-03 20:49 → 2026-01-03 21:45
**Outcome:** success

---

## TLDR

Investigated why agents go idle without Phase: Complete and why orch complete --force fails silently. Found three root causes: (1) SSE idle detection is distinct from Phase: Complete by design, (2) FallbackClose doesn't respect beads.DefaultDir for cross-project operations, (3) CloseIssue RPC client doesn't use WithCwd. Recommended fixes for both CLI and RPC paths.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-03-inv-agents-going-idle-without-phase.md` - Full investigation with findings and recommendations

### Files Modified
- None (investigation-only session)

### Commits
- None (no code changes - investigation deliverable is the investigation file)

---

## Evidence (What Was Observed)

- SSE idle detection in `pkg/opencode/monitor.go:172-181` triggers on busy→idle transition, not Phase: Complete
- `FallbackClose` in `pkg/beads/client.go:738-747` uses `exec.Command("bd", args...)` without `cmd.Dir`
- `CloseIssue` in `pkg/verify/check.go:570-583` creates RPC client without `WithCwd` option
- Manual test: `cd /tmp && bd close orch-go-rzch` fails with "no beads database found" (exit code 1)
- Issues 80tq and 57dn are now both closed - can't reproduce original failure

### Tests Run
```bash
# Verified bd close fails from wrong directory
cd /tmp && bd close orch-go-rzch --reason "test" 2>&1; echo "Exit code: $?"
# Result: Error: no beads database found, Exit code: 1

# Verified issue states
bd show 57dn --json | jq -r '.[] | "Status: \(.status)"'
# Result: Status: closed
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-03-inv-agents-going-idle-without-phase.md` - Full root cause analysis

### Decisions Made
- SSE idle ≠ Phase: Complete is BY DESIGN, not a bug - this is correctly documented in prior constraints
- Cross-project context loss is the likely cause of "silent failure" when closing issues

### Constraints Discovered
- FallbackClose must respect beads.DefaultDir for cross-project operations
- RPC client should use WithCwd when beads.DefaultDir is set
- Error visibility needed for CLI fallback (use CombinedOutput, not Run)

### Externalized via `kn`
- Not applicable - findings captured in investigation file

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Fix FallbackClose and CloseIssue for cross-project directory context
**Skill:** feature-impl
**Context:**
```
FallbackClose in pkg/beads/client.go:738-747 needs cmd.Dir set to DefaultDir.
CloseIssue in pkg/verify/check.go:570-583 needs WithCwd option on RPC client.
See investigation: .kb/investigations/2026-01-03-inv-agents-going-idle-without-phase.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why do some agents exhaust context before reporting Phase: Complete?
- Is there a way to enforce Phase: Complete before session end?

**Areas worth exploring further:**
- Context exhaustion detection and early warning
- Pre-exit phase reporting enforcement in spawn template

**What remains unclear:**
- Exact sequence of events that caused 57dn's original failure (can't reproduce - issue now closed)
- Whether beads daemon validates Cwd field for close operations

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus
**Workspace:** `.orch/workspace/og-debug-agents-going-idle-03jan/`
**Investigation:** `.kb/investigations/2026-01-03-inv-agents-going-idle-without-phase.md`
**Beads:** `bd show orch-go-rzch`
