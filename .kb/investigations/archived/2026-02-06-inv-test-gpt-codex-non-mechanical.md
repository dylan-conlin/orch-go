## Summary (D.E.K.N.)

**Delta:** The three sessions were interrupted in a single 09:31 OpenCode server-restart event, not by long-running agent instability; one proposed fix landed, one partially landed (different root cause), and one did not land.

**Evidence:** All three issues received DEAD SESSION comments at 2026-02-06 09:31, archived activity logs show SERVER RECOVERY interruption notices at the same minute, and direct code checks show `21315` fix present while `21294` and the later `21275` root-cause fix are missing.

**Knowledge:** Failure mode is primarily control-plane reliability (restart/recovery + completion-comment persistence/reconciliation), not runtime duration; session death classification can diverge from actual work/commit state.

**Next:** Reopen or follow up unresolved fixes (`21294`, `21275` stale-client path), and harden dead-session detection with restart-aware grace plus read-after-write verification for `Phase: Complete` comments.

**Authority:** architectural - Fix spans daemon dead-session policy, recovery flow, and beads persistence verification across components.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Test Gpt Codex Non Mechanical

**Question:** Why did sessions `orch-go-21294`, `orch-go-21315`, and `orch-go-21275` die without completing, is this an OpenCode long-running stability issue, and what pattern explains the failures?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** openai/gpt-5.3-codex
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation                                                                  | Relationship | Verified | Conflicts                                                                                                               |
| ------------------------------------------------------------------------------ | ------------ | -------- | ----------------------------------------------------------------------------------------------------------------------- |
| `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md`    | confirms     | yes      | No conflict; this investigation found the same restart-interruption class and extends with issue-level landing analysis |
| N/A - novel synthesis across issue comments + workspace activity + git history | -            | -        | -                                                                                                                       |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: The three target sessions were interrupted as one clustered restart event

**Evidence:** Dead-session comments for `orch-go-21294`, `orch-go-21315`, and `orch-go-21275` were all created at `2026-02-06T09:31`, and archived activity logs include explicit `SERVER RECOVERY: The OpenCode server was restarted and your session was interrupted` messages at `09:31-09:32`.

**Source:** `.beads/issues.jsonl:471`, `.beads/issues.jsonl:490`, `.beads/issues.jsonl:511`, `.orch/workspace/archived/og-debug-daemon-spawned-agents-05feb-c361/ACTIVITY.json:10293`, `.orch/workspace/archived/og-debug-bd-label-remove-05feb-23a8/ACTIVITY.json:9340`, command: `python3` dead-comment clustering script (`2026-02-06T09:31` count=3).

**Significance:** This points to a shared infrastructure interruption, not three independent task-specific crashes.

---

### Finding 2: This is not a long-running-agent stability pattern

**Evidence:** Session timestamp output shows target sessions were created around `23:48-23:49` and last updated around `23:56` (about 7-8 minutes active), then later interrupted by morning restart.

**Source:** `.orch/workspace/archived/og-debug-daemon-spawned-agents-05feb-c361/ACTIVITY.json:4573`, command in log: `curl -s "http://127.0.0.1:4096/session" ...`.

**Significance:** The evidence does not support "long-running sessions become unstable" for this incident.

---

### Finding 3: Root-cause-to-code landing status is mixed across the three issues

**Evidence:**

- `orch-go-21315`: landed (`c80ba366`) and code uses `Registry.Purge + SaveSkipMerge` in both daemon cleanup and `orch clean` paths.
- `orch-go-21294`: not landed for the stated root cause; `ProcessCompletion` still closes issue without `RemoveTriageReadyLabel`.
- `orch-go-21275`: first attempt fix landed (`fa08bef9`, verification filter scope), but later dead-session root-cause comment about stale `beadsClient` in `serve_attention.go` is not landed (no socket-existence guard there).

**Source:** `git show c80ba366`, `pkg/daemon/cleanup.go:71`, `cmd/orch/clean_cmd.go:1270`, `pkg/daemon/completion_processing.go:264`, `git show fa08bef9`, `cmd/orch/serve_attention.go:154`, command: python code-presence check (FOUND/MISSING matrix).

**Significance:** "Root cause identified in comments" did not consistently mean "fix landed for that root cause," which explains confusion in backlog state.

---

### Finding 4: Completion-state persistence/reconciliation is a likely contributing reliability gap

**Evidence:** In `orch-go-21315` activity, the agent ran `bd comments add ... "Phase: Complete ..."` and got `Comment added`, but current issue record contains no `Phase: Complete` comment; prior issue `orch-go-21112` already documents this exact silent non-persistence failure mode.

**Source:** `.orch/workspace/archived/og-debug-daemon-spawned-agents-05feb-c361/ACTIVITY.json:10123`, `.orch/workspace/archived/og-debug-daemon-spawned-agents-05feb-c361/ACTIVITY.json:10127`, `.beads/issues.jsonl:511`, `.beads/issues.jsonl:299`.

**Significance:** Dead-session detection that depends on `Phase: Complete` comment presence can false-negative completed work if comment persistence is unreliable.

---

## Synthesis

**Key Insights:**

1. **Shared outage signature** - The three target sessions line up to a single restart/recovery window, so this is one incident affecting multiple sessions.

2. **Control-plane over runtime failure** - Evidence points to restart/recovery plus state reconciliation problems (session tracking + comment persistence), not long-running agent degradation.

3. **Issue lifecycle drift** - Closed/open transitions and comment histories can diverge from actual code-landed state, so "issue closed" is insufficient evidence that the specific proposed fix shipped.

**Answer to Investigation Question:**

These sessions died primarily because of an OpenCode server restart event that interrupted active workers, not because of long-running session instability. The pattern is clustered interruption + imperfect recovery/accounting (dead-session marking and `Phase: Complete` persistence), with inconsistent fix landing relative to comment-level root-cause statements. `orch-go-21315`'s fix is landed in code, while `orch-go-21294` and the later `orch-go-21275` stale-client root-cause fix are not landed. Limitation: I did not live-reproduce comment persistence loss or stale-client behavior in a fresh runtime; conclusions rely on repository artifacts, activity logs, and commit/code verification.

---

## Structured Uncertainty

**What's tested:**

- ✅ Dead-session clustering is real (verified with Python parsing of `.beads/issues.jsonl`: 3 dead comments at `2026-02-06T09:31`).
- ✅ Landing status matrix is accurate (verified via `git show` for `c80ba366`/`fa08bef9` and direct code-presence checks in current files).
- ✅ Sessions were short active bursts before interruption (verified from session created/updated timestamps in archived activity output).

**What's untested:**

- ⚠️ Direct runtime reproduction of `bd comments add` silent non-persistence in current environment.
- ⚠️ Direct runtime reproduction of stale `beadsClient` behavior in `serve_attention` after socket disappearance.
- ⚠️ Whether dead-session detector currently includes any hidden restart grace not visible from artifacts.

**What would change this:**

- If a persisted issue history source shows `Phase: Complete` comments for these exact sessions that were omitted from `.beads/issues.jsonl`.
- If fresh reproduction shows `serve_attention` already handles socket-loss fallback correctly despite missing explicit guards.
- If server restart timeline proves these sessions died before (not during/after) the 09:31 recovery window.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation                                                                               | Authority      | Rationale                                                                                                   |
| -------------------------------------------------------------------------------------------- | -------------- | ----------------------------------------------------------------------------------------------------------- |
| Add restart-aware dead-session grace and comment-persistence verification in completion flow | architectural  | Touches daemon detection policy, recovery coordination, and beads write/read verification across subsystems |
| Land unresolved root-cause fixes for `21294` and `21275` stale-client path                   | implementation | Localized code changes in existing files without changing overall architecture                              |

**Authority Levels:**

- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"

- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Restart-Aware Completion Reliability** - Make dead-session marking contingent on restart context and verified completion-comment persistence.

**Why this approach:**

- Prevents false dead-session marking immediately after server recovery events.
- Aligns issue lifecycle state with actual completed work by verifying `Phase: Complete` persistence.
- Directly addresses Findings 1 and 4 (clustered restart + persistence gap).

**Trade-offs accepted:**

- Slightly delayed dead-session detection due to grace window.
- Additional beads read-after-write overhead during completion; acceptable because completion frequency is low relative to polling.

**Implementation sequence:**

1. Add a short post-restart grace path in dead-session detection before adding DEAD SESSION comment.
2. Add completion comment read-after-write verification (or retry) before considering session unresolved.
3. Land missing targeted code fixes (`21294` completion label removal, `21275` attention socket-loss fallback).

### Alternative Approaches Considered

**Option B: Keep current detector and only improve docs/process**

- **Pros:** Minimal code change.
- **Cons:** Does not address observed restart clustering or comment persistence mismatch; likely repeat incidents.
- **When to use instead:** Only if production evidence disproves technical persistence/recovery faults.

**Option C: Disable automatic dead-session marking entirely**

- **Pros:** Eliminates false-positive dead comments.
- **Cons:** Reintroduces zombie `in_progress` issues and manual cleanup burden.
- **When to use instead:** Temporary emergency mode during known outage windows.

**Rationale for recommendation:** Option A is the smallest change set that addresses all high-signal failure points observed in this investigation.

---

### Implementation Details

**What to implement first:**

- Dead-session detector restart grace and recovery awareness.
- Completion comment persistence confirmation before dead marking.
- `pkg/daemon/completion_processing.go` label-removal parity with `orch complete`.

**Things to watch out for:**

- ⚠️ Race between server recovery and detector cycle can still misclassify if grace period is too short.
- ⚠️ Mixed data sources (OpenCode session state vs beads comments) can drift unless reconciliation order is explicit.
- ⚠️ Retrying comment writes must avoid duplicate/spam comment behavior.

**Areas needing further investigation:**

- Why successful `bd comments add` can be absent from persisted issue JSON in some sessions.
- Whether session activity persistence differs between archived and non-archived workspaces (e.g., missing `ACTIVITY.json` for `og-feat-orch-go-systematic-05feb-3478`).
- Whether issue closure at 09:40 without `Phase: Complete` should be treated as policy exception or tooling bug.

**Success criteria:**

- ✅ Post-restart dead-session false positives drop to zero in sampled incidents.
- ✅ `Phase: Complete` comments used by detectors always match persisted issue state.
- ✅ Reproductions for `21294` and `21275` root causes pass with explicit tests.

---

## References

**Files Examined:**

- `.beads/issues.jsonl` - Canonical issue/comment timeline, dead-session clustering, and status transitions.
- `.orch/workspace/archived/og-debug-daemon-spawned-agents-05feb-c361/ACTIVITY.json` - Server recovery event, session timestamps, and completion-comment attempt evidence.
- `.orch/workspace/archived/og-debug-bd-label-remove-05feb-23a8/ACTIVITY.json` - Second interrupted session evidence for `21294`.
- `pkg/daemon/completion_processing.go` - Verify whether `21294` proposed label-removal fix landed.
- `pkg/daemon/cleanup.go` and `cmd/orch/clean_cmd.go` - Verify whether `21315` locking fix landed.
- `cmd/orch/serve_attention.go` - Verify whether later `21275` stale-client root-cause fix landed.

**Commands Run:**

```bash
# Verify issue state and comments
bd show orch-go-21294 --json
bd show orch-go-21315 --json
bd show orch-go-21275 --json

# Verify code-landed status against root-cause claims
python3 - <<'PY'
from pathlib import Path
checks = {
  '21294_completion_label_removal': ('pkg/daemon/completion_processing.go', 'RemoveTriageReadyLabel(agent.BeadsID)'),
  '21315_cleanup_locking': ('pkg/daemon/cleanup.go', 'SaveSkipMerge()'),
  '21315_clean_cmd_locking': ('cmd/orch/clean_cmd.go', 'SaveSkipMerge()'),
  '21275_attention_socket_check': ('cmd/orch/serve_attention.go', 'socketExists'),
}
for name,(rel,needle) in checks.items():
    text=Path(rel).read_text()
    print(f"{name}: {'FOUND' if needle in text else 'MISSING'}")
PY

# Verify commit-level landing evidence
git show --oneline fa08bef9 -- cmd/orch/serve_attention.go cmd/orch/serve_attention_test.go
git show --oneline c80ba366 -- pkg/daemon/cleanup.go cmd/orch/clean_cmd.go

# Verify dead-session clustering
python3 - <<'PY'
import json
from collections import Counter
from pathlib import Path
rows=[]
for line in Path('.beads/issues.jsonl').read_text().splitlines():
    if not line.strip():
        continue
    rec=json.loads(line)
    for c in rec.get('comments',[]):
        if c.get('text','').startswith('DEAD SESSION:'):
            rows.append(c['created_at'][:16])
print(Counter(rows))
PY
```

**External Documentation:**

- N/A

**Related Artifacts:**

- **Decision:** `kb-25dc4c` - Captures the reusable pattern that clustered dead-session comments map to restart incidents, not long-running degradation.
- **Investigation:** `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md` - Prior restart/interruption model this investigation validates at issue level.
- **Issue:** `.beads/issues.jsonl:299` (`orch-go-21112`) - Prior documented "Phase: Complete comment fails to persist" pattern matching observed evidence.
- **Workspace:** `.orch/workspace/archived/og-debug-daemon-spawned-agents-05feb-c361/` - Primary activity evidence for `21315`.
- **Workspace:** `.orch/workspace/archived/og-debug-bd-label-remove-05feb-23a8/` - Primary activity evidence for `21294`.

---

## Investigation History

**[2026-02-06 09:56]:** Investigation started

- Initial question: Why these three sessions died and whether this indicates long-running OpenCode instability.
- Context: Spawned non-mechanical synthesis task to evaluate GPT-5.3-codex judgment quality.

**[2026-02-06 10:15]:** Restart clustering and landing-status matrix established

- Verified shared 09:31 interruption pattern and code-level landed/unlanded split across the three issues.

**[2026-02-06 10:26]:** Investigation completed

- Status: Complete
- Key outcome: Incident is a restart/reconciliation reliability pattern, not long-running instability; unresolved fixes remain for `21294` and later `21275` root-cause claim.
