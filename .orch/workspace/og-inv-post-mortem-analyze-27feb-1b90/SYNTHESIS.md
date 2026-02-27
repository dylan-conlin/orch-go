# Session Synthesis

**Agent:** og-inv-post-mortem-analyze-27feb-1b90
**Issue:** orch-go-ge7i
**Duration:** 2026-02-27
**Outcome:** success

---

## Plain-Language Summary

Analyzed 3 session transcripts where Dylan experienced total communication breakdown between orchestrators in the Toolshed and Price Watch projects. The core incident: OshCut shipping data showed "Not collected" despite being fully available in Price Watch — a stale knowledge model caused orchestrators in both repos to misdiagnose the problem, while Dylan manually relayed findings between them for 60+ minutes.

Found 21 individual communication failures organized into 7 categories: stale context (root cause), frame guard rigidity (orchestrators couldn't read code to debug), user as message bus (Dylan relaying between orchestrators), CLI/gate cascade (repeated orch complete failures), premature action, role confusion, and follow-through gaps. The `kb agreements` system directly addresses the root cause (stale context = stale-shadow failure mode) and would have prevented the cascade — but 5 of 7 failure categories are in-session behavioral issues that agreements can't catch. Recommendations: orchestrator diagnostic mode for debugging, all-at-once gate reporting for orch complete, and a hard rule that agreement checks must use consumer-facing interfaces (API) not source-of-truth (DB).

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace for verification evidence.

---

## TLDR

Post-mortem of 3 broken sessions found 21 failures across 7 categories. The agreement system catches the root cause (stale cross-project context) that triggered the entire cascade, but 5 of 7 downstream failure categories (frame guard rigidity, user as message bus, CLI cascade, role confusion, follow-through gaps) represent gaps agreements won't address.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-27-postmortem-communication-breakdown-sessions.md` — Full post-mortem with per-session breakdowns, 7-category taxonomy, and gap analysis
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-27-probe-communication-breakdown-postmortem-3-sessions.md` — Model probe extending orchestrator-session-lifecycle with real-world behavioral evidence
- `.orch/workspace/og-inv-post-mortem-analyze-27feb-1b90/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-post-mortem-analyze-27feb-1b90/VERIFICATION_SPEC.yaml` — Verification spec

---

## Evidence (What Was Observed)

- 3 full session transcripts read and analyzed: 267 lines (Session 1), ~887 lines (Session 2), 873 lines (Session 3)
- 21 individual failure instances cataloged with line-number references
- 7 distinct failure categories identified, mapped against agreement system capabilities
- Agreement system reviewed: 5 built-in + 5 custom agreements, non-blocking spawn gate, stale-shadow detection designed

### Key Observations
1. Stale context was the ROOT CAUSE: if the OshCut HTTP migration had been reflected in Toolshed's PW integration model, most downstream failures wouldn't have occurred
2. Frame guard blocked code reading 4 times across sessions — each time forcing agent spawn with 2-5 min delay
3. Dylan copy-pasted 5+ relay messages between orchestrators in Session 2
4. orch complete hit 4+ sequential gate failures per session (8+ retries total)
5. Production API served null shipping while DB had shipping populated — agreement checks must use consumer-facing interface

---

## Architectural Choices

No architectural choices — this was analysis/investigation work, not implementation.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-27-postmortem-communication-breakdown-sessions.md` — Full post-mortem analysis
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-27-probe-communication-breakdown-postmortem-3-sessions.md` — Model probe

### Constraints Discovered
- Agreement checks for data contracts MUST run against consumer-facing interface (API endpoint), not source of truth (DB). DB being right is meaningless if API serves null.
- Frame guard needs a diagnostic mode — read-only code access during active debugging is legitimate orchestrator work, distinct from implementation.
- orch complete should report all gate failures at once, not sequentially.

### Externalized via `kb quick`
- See completion section — kb quick constraint for agreement check interface requirement

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + probe + synthesis)
- [x] Investigation file has analysis complete
- [x] Ready for `orch complete orch-go-ge7i`

### Recommended Follow-Up Issues
1. **Orch complete all-at-once gate reporting** — Report all unmet gates in single failure message (addresses CLI cascade category)
2. **Orchestrator diagnostic mode design** — Time-limited frame guard relaxation for read-only code access during active debugging (addresses frame guard rigidity)
3. **Agreement check constraint: consumer-facing interfaces** — Hard rule that data contract checks must use API endpoints not DB queries

---

## Unexplored Questions

- **Promise tracking for orchestrators:** How would session-level commitment tracking work? When orchestrator says "I'll do X", how to reliably track and surface unfulfilled promises?
- **Cross-repo context bridge:** Can `kb context` with cross-project filtering provide real-time dependency context at session start? How would this interact with spawn context generation?
- **Frame collapse risk of diagnostic mode:** If we add read-only code access during debugging, how do we prevent orchestrators from sliding into implementer behavior? What's the right boundary?

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-post-mortem-analyze-27feb-1b90/`
**Investigation:** `.kb/investigations/2026-02-27-postmortem-communication-breakdown-sessions.md`
**Beads:** `bd show orch-go-ge7i`
