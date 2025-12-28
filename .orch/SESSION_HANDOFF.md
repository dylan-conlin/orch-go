# Session Handoff - 28 Dec 2025 (Evening)

## TLDR

**Stabilization + knowledge infrastructure session.** Completed P0 verification fixes from prior session, investigated knowledge fragmentation (500 investigations with 0 lineage links), shipped 3 knowledge linkage improvements, and finally solved the "OpenCode redirect loop" mystery (it's expected behavior, not a bug).

---

## D.E.K.N. Summary

### Delta (What Changed)
- **Completed** P0 verification fixes (`orch-go-ik77`, `orch-go-bn9y`) - agents now blocked without test evidence
- **Shipped** 3 knowledge linkage improvements:
  - `orch-go-89y0` - kb reflect semantic clustering (in kb-cli)
  - `orch-go-87rn` - Spawn context includes related investigations
  - `orch-go-i5s5` - Lineage reminder in spawn template
- **Solved** OpenCode "redirect loop" mystery - NOT a bug, use `/session` not `/health`
- **Closed** 10+ zombie/verification-only agents
- **Created** `orch-go-x7vn` bug: visual verification checks project git history, not agent-specific changes

### Evidence (Proof of Work)
- `go test ./pkg/verify/...` passes with 47 new test cases for evidence verification
- Commits: `a6214ce7` (test evidence), `fa77b8d5` (lineage reminder), `fcb5de77` (related investigations)
- Investigation: `.kb/investigations/2025-12-28-inv-knowledge-fragmentation-433-investigations-days.md`
- Investigation: `.kb/investigations/2025-12-28-inv-opencode-redirect-loop-health-sessions.md`

### Knowledge (What Was Learned)

**1. Knowledge fragmentation is a linkage problem, not duplication**
500 investigations exist but 0 have lineage references (Supersedes/Extracted-From). Investigations are distinct - each explores different aspect. The fix is better linkage, not synthesis passes.

**2. OpenCode "redirect loop" is expected behavior**
OpenCode has NO `/health` endpoint. Unknown routes proxy to `desktop.opencode.ai` (web app), causing auth redirects. Use `/session` for health checks. This has been investigated 4 times - knowledge surfacing is the real issue.

**3. Dashboard/CLI status mismatch remains**
API marks `Phase: Complete` agents as "completed", CLI keeps them "active" until `orch complete`. This is semantic difference, not a bug - but causes confusion.

### Next (Recommended Actions)

**Resume MCP investigation:**
`orch-go-xnqg` - "MCP vs CLI: What is MCP's actual value proposition?" - 3 attempts failed (stuck in Planning). Needs narrower scope or investigation into why sessions die.

**Auto-switch account:**
`orch-go-bwrm` - "Auto-switch account failing silently" - not urgent at 35% usage, but will bite when rate limited.

---

## What Actually Happened This Session

1. Read prior handoff - crisis response session had identified P0s
2. Completed P0 verification fixes (test evidence requirements now enforced)
3. Completed 6 zombie agents (prior work, verification-only sessions)
4. Investigated knowledge fragmentation → found linkage is the problem
5. Spawned 3 agents to fix knowledge linkage → all completed successfully
6. Investigated OpenCode redirect loop → solved (use `/session` not `/health`)
7. MCP investigation failed 3x → abandoned, needs different approach

### Key Commits This Session
- `a6214ce7` - feat(verify): require test execution evidence
- `fa77b8d5` - feat(spawn): add lineage reminder to template
- `fcb5de77` - feat(spawn): include Delta for investigations in context
- `57939c5d` - inv: OpenCode redirect loop is expected behavior

---

## Agents Still Running
None - all completed or abandoned.

---

## Local State

**Branch:** master  
**Uncommitted:** Workspace files only (ephemeral)

```bash
# Sync and push
bd sync && git add .beads/ .kn/ && git commit -m "chore: sync beads and kn" && git push
```

**Note:** `git push` was failing with "Repository not found" - may need SSH key refresh or repo access check.

---

## Open Issues Worth Noting

| Issue | Type | Summary |
|-------|------|---------|
| `orch-go-xnqg` | investigation | MCP vs CLI value proposition (3 failed attempts) |
| `orch-go-bwrm` | investigation | Auto-switch account failing silently |
| `orch-go-x7vn` | bug | Visual verification checks project history, not agent changes |
| `orch-go-bgf5` | investigation | Dashboard API shows 0 active (status mismatch) |

---

## What NOT To Do

1. **Don't call `/health` on OpenCode** - use `/session` instead
2. **Don't investigate redirect loop again** - it's been solved 4 times, answer is in kn-2a4e34
3. **Don't force complete agents without checking tests** - new verification will catch this

---

## Key Investigations to Read

| File | Summary |
|------|---------|
| `.kb/investigations/2025-12-28-inv-knowledge-fragmentation-433-investigations-days.md` | 500 investigations have 0 lineage links - fix is linkage not synthesis |
| `.kb/investigations/2025-12-28-inv-opencode-redirect-loop-health-sessions.md` | Redirect loop is expected - OpenCode proxies unknown routes to web app |
| `.kb/investigations/2025-12-28-inv-verification-system-audit-verification-theater.md` | Why agents could claim "tests pass" without evidence (now fixed) |

---

## Verification System Now Enforces

Agents are blocked from completion if:
1. Skill is `feature-impl`, `systematic-debugging`, or `reliability-testing`
2. Code files (`.go`, `.ts`, `.py`, etc.) were modified
3. No test execution evidence in beads comments

Valid evidence example:
```
bd comment <id> 'Tests: go test ./pkg/... - PASS (12 tests in 0.8s)'
```

---

## Session Metadata

**Generated:** 28 Dec 2025 ~15:40 PST  
**Duration:** ~2 hours  
**Focus:** Stabilization + knowledge infrastructure  
**Outcome:** P0s complete, knowledge linkage improved, redirect loop solved
