# Session Synthesis

**Agent:** og-inv-attention-system-dashboard-16feb-b422
**Issue:** orch-go-992
**Outcome:** success

---

## Plain-Language Summary

The attention system has a fully implemented backend with 11 real collectors that query beads, git, agent APIs, and event files — none are stubs. However, the dashboard currently shows **zero correct badges**. Only 2 of 11 collectors are producing output (issue-ready and recently-closed), and both map to false "Awaiting verification" badges because the frontend's `mapSignalToBadge()` function defaults unmapped signal types to `'verify'`. Three badge types (`decide`, `escalate`, `crashed`) exist only in the frontend type system with no backend collector that could ever emit them. The system is architecturally sound but needs the frontend mapping fixed and the signal-to-badge gap closed before any attention badges provide real value.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root.

---

## Delta (What Changed)

### Files Created
- `.kb/models/dashboard-architecture/probes/2026-02-16-attention-pipeline-full-audit.md` — Comprehensive probe covering all 11 collectors and 9 badge types

### Files Modified
- None (audit only)

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

### Live API Results
- `/api/attention` returns 59 items: 33 `issue-ready` + 26 `recently-closed`. All 11 collector sources registered. Only 2 produced output.
- `/api/attention/likely-done` returns 0 signals (no open issues with commits and no active workspace)
- `/api/agents?since=all` shows 1341 agents: 1332 completed, 9 dead, 0 active/idle, 0 awaiting-cleanup
- `~/.orch/verify-failed.jsonl` has 52 entries but all >72h old (cutoff is 72h)
- `~/.orch/verifications.jsonl` has hundreds of entries but none match recently-closed item subjects — all 26 recently-closed items show "unverified"

### Code Review
- All 11 collectors in `pkg/attention/` have real detection logic (not stubs)
- `mapSignalToBadge()` in `attention.ts:110-140` maps 5 signals explicitly, defaults all others to `'verify'`
- 5 backend signals (`issue-ready`, `stale`, `duplicate-candidate`, `competing`, `epic-orphaned`) have no frontend badge mapping
- 3 badge types (`decide`, `escalate`, `crashed`) have no backend collector

---

## Knowledge (What Was Learned)

### Key Findings

1. **Backend is real, frontend mapping is broken.** All 11 collectors read real data from real sources. The problem is entirely in the frontend signal-to-badge translation layer.

2. **Three badge types are aspirational.** `decide`, `escalate`, and `crashed` exist as TypeScript types and badge config entries but have zero backend support. No collector emits these signals.

3. **The default fallback is the root cause of all false badges.** `attention.ts:138` `default: return 'verify'` causes every unmapped signal to show "Awaiting verification". This single line produces 100% of the false positive badges.

4. **Verification status matching may be broken.** Despite hundreds of entries in `verifications.jsonl`, all 26 recently-closed items show `unverified`. This suggests the lookup key (`item.Subject` = beads ID) doesn't match the verification log entries, or the annotations are being overwritten.

5. **Most collectors don't fire due to current project state, not bugs.** StuckCollector needs active agents. LikelyDone needs open issues with commits. StaleCollector needs old issues. These are all legitimate "nothing to report" conditions.

### Constraints Discovered
- `CompetingCollector` requires `area:` labels on beads issues — without systematic labeling, it will never fire
- `VerifyFailedCollector` has a 72h lookback window — verification failures older than 3 days disappear silently

### Externalized via `kb quick`
- (see below)

---

## Next (What Should Happen)

**Recommendation:** close (audit complete) + spawn follow-ups for the fixes

### To Make Attention Badges Work (priority order)

1. **Fix `mapSignalToBadge()` default** — change `default: return 'verify'` to `default: return null` and filter null badges from the signals map. This eliminates all false positive badges immediately.

2. **Add badge mappings for 5 unmapped signals** — `issue-ready` (no badge or new "ready" badge), `stale` (new badge), `duplicate-candidate` (new badge), `competing` (new badge), `epic-orphaned` (new badge). Or filter these signals out before they reach the frontend.

3. **Investigate verification status key mismatch** — why do 26 recently-closed items show "unverified" when `verifications.jsonl` has hundreds of entries? The lookup in `serve_attention.go:295` uses `item.Subject` but verification entries use `issue_id` — are these the same key?

4. **Decide fate of `decide`, `escalate`, `crashed` badge types** — either create collectors for them or remove them from the type system to prevent confusion.

---

## Unexplored Questions

- **Verification key mismatch:** Do the `issue_id` values in `verifications.jsonl` match the `item.Subject` values used for lookup in `serve_attention.go`? If not, verified issues would always show as "unverified".
- **Performance of DuplicateCandidateCollector:** With O(n^2) title comparison on all open issues, this could become slow with hundreds of issues.
- **Is `issue-ready` even useful as an attention signal?** It duplicates information already available in the work-graph tree (open issues are inherently visible). It may add noise without value.

---

## Session Metadata

**Skill:** investigation
**Workspace:** `.orch/workspace/og-inv-attention-system-dashboard-16feb-b422/`
**Probe:** `.kb/models/dashboard-architecture/probes/2026-02-16-attention-pipeline-full-audit.md`
**Beads:** `bd show orch-go-992`
