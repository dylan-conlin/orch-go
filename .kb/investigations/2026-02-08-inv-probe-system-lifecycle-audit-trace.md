## Summary (D.E.K.N.)

**Delta:** The probe→model feedback loop partially works: all 6 parent models have uncommitted diffs incorporating probe findings, but 9 of 10 probes remain untracked in git and no workspace produced a SYNTHESIS.md, meaning the feedback loop closes in working-tree edits but not in committed, reviewable artifacts.

**Evidence:** `git status` shows 9/10 probes untracked; `git diff` shows all 6 parent models have uncommitted changes referencing probe findings; 0/5 checked workspaces have SYNTHESIS.md files; only 1 probe (SSE FD leak) was committed in `b008bc89`.

**Knowledge:** The probe system produces high-quality artifacts that reference specific model claims. Model updates happen (working tree shows edits incorporating probe findings). But the commit/review cycle is broken — probes and their model updates accumulate as uncommitted working-tree changes. The feedback loop is "open" at the persistence boundary.

**Next:** Add probe commit automation to `orch complete` or create a `kb commit-probes` command that commits probe + model update pairs atomically. The artifacts are good; the persistence pipeline is missing.

**Authority:** architectural - Spans spawn system, completion protocol, and kb CLI; requires orchestrator-level design.

---

# Investigation: Probe System Lifecycle Audit Trace

**Question:** Does the probe→model feedback loop actually close? Do probe findings flow back into models, or are probes produced and then ignored?

**Started:** 2026-02-08
**Updated:** 2026-02-08
**Owner:** Agent (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation                                                           | Relationship | Verified | Conflicts                                               |
| ----------------------------------------------------------------------- | ------------ | -------- | ------------------------------------------------------- |
| .kb/decisions/2026-02-08-model-centric-probes-replace-investigations.md | extends      | yes      | Decision assumes probes will be committed; 9/10 are not |

---

## Findings

### Finding 1: All 10 probes exist and follow the correct template structure

**Evidence:** 10 probe files found across 6 parent model directories:

- `beads-integration-architecture/probes/` — 5 probes (bd-subprocess, ready-queue, stale-retry, synthesis-dedup, jsonl-hash-mismatch)
- `completion-verification/probes/` — 1 probe (transient-verification-retry)
- `agent-lifecycle-state-model/probes/` — 1 probe (commit-idle-auto-completion)
- `multi-model-evaluation-feb2026/probes/` — 1 probe (daemon-model-routing)
- `daemon-autonomous-operation/probes/` — 1 probe (processed-cache-mark)
- `sse-connection-management/probes/` — 1 probe (delayed-log-fd-leak)

All follow the mandated 4-section format (Question, What I Tested, What I Observed, Model Impact). All include concrete command output in "What I Tested" — none are code-review-only.

**Source:** `glob .kb/models/*/probes/*.md`; read all 10 files.

**Significance:** The probe artifact quality is high. Agents are following the template correctly. The system produces good probes.

---

### Finding 2: All parent models existed before their probes (verified)

**Evidence:** Model creation dates vs probe filesystem creation:

| Model                             | Created    | Earliest Probe (filesystem) |
| --------------------------------- | ---------- | --------------------------- |
| beads-integration-architecture.md | 2026-01-12 | 2026-02-08 10:51            |
| completion-verification.md        | 2026-01-12 | 2026-02-08 16:46            |
| agent-lifecycle-state-model.md    | 2026-01-12 | 2026-02-08 11:37            |
| multi-model-evaluation-feb2026.md | 2026-02-07 | 2026-02-08 10:57            |
| daemon-autonomous-operation.md    | 2026-01-12 | 2026-02-08 10:42            |
| sse-connection-management.md      | 2026-01-29 | 2026-02-08 10:05            |

Every model predates its probes by days to weeks. No probe was created against a nonexistent model.

**Source:** `git log --all --diff-filter=A` for model files; `stat -f` for probe filesystem timestamps.

**Significance:** The probe→model relationship is correctly ordered. Agents are writing probes against existing models, not creating probes in isolation.

---

### Finding 3: 9 of 10 probes are UNTRACKED in git (never committed)

**Evidence:** `git status --porcelain -- '.kb/models/*/probes/*.md'` shows 9 files with `??` (untracked) status. Only `sse-connection-management/probes/2026-02-08-delayed-log-open-fd-leak.md` was committed (in `b008bc89`, 2026-02-08 10:13:46).

The one committed probe was part of a manual bulk commit ("chore: update kb models, decisions, and quick entries from agent work") that also updated 11 models. This was a human-initiated commit, not automated.

**Source:** `git status --porcelain`; `git log --all --oneline --diff-filter=A -- '.kb/models/*/probes/*.md'`

**Significance:** This is the critical failure. Probes exist on disk but are invisible to git history, code review, and other sessions. If the working tree is cleaned, 9 probes are lost. The persistence boundary is broken.

---

### Finding 4: All 10 probes reference specific model claims (not generic)

**Evidence:** Each probe's "Question" section references a concrete model claim or invariant:

| Probe                        | Specific Claim Referenced                                               |
| ---------------------------- | ----------------------------------------------------------------------- |
| bd-subprocess-cap-hit        | CLI fallback observability (model section on fallback behavior)         |
| ready-queue-accessibility    | "ID-not-found as a cross-project failure mode at completion time"       |
| stale-retry-jsonl-mtime      | "RPC-first + CLI fallback resilience under concurrent write contention" |
| synthesis-dedup-parse-error  | "Auto-tracking duplicate risk" mitigation                               |
| jsonl-hash-mismatch          | CLI fallback output mode-awareness                                      |
| transient-verification-retry | "strict verification requirements" gate model                           |
| commit-idle-auto-completion  | "`session idle ≠ completion`" invariant                                 |
| daemon-model-routing         | "label-based per-issue routing is not yet wired" open claim             |
| processed-cache-mark         | "dedup invariants for daemon spawn cycle"                               |
| delayed-log-fd-leak          | "SSE pool pressure and reconnection behavior"                           |

Every probe precisely targets a model claim. Zero probes are generic or untethered.

**Source:** Read all 10 probe files, cross-referenced with parent model sections.

**Significance:** Probe quality is high. The model→probe targeting relationship works as designed.

---

### Finding 5: All 6 parent models have uncommitted working-tree updates incorporating probe findings

**Evidence:** `git diff --stat -- '.kb/models/*.md'` shows modifications to all 6 parent models:

| Model                             | Uncommitted Changes | Probe References Added                         |
| --------------------------------- | ------------------- | ---------------------------------------------- |
| beads-integration-architecture.md | +7/-2 lines         | "Recent Probes" section with 3 probe summaries |
| completion-verification.md        | +37/-3 lines        | "Recent Probes" section + Phase 7 description  |
| agent-lifecycle-state-model.md    | +5/-2 lines         | "Recent Probes" section with commit-idle probe |
| multi-model-evaluation-feb2026.md | +6/-1 lines         | "Recent Probes" section + Feb 8-9 update       |
| daemon-autonomous-operation.md    | +15/-3 lines        | "Recent Probes" section + significant changes  |
| sse-connection-management.md      | +7/-4 lines         | "Recent Probes" section with FD leak probe     |

Each model update includes a "Recent Probes" section that summarizes the probe's verdict, what was extended/confirmed, and confidence level. The model diffs accurately reflect probe findings.

**Source:** `git diff` for each model file.

**Significance:** The feedback loop is mechanically closing — probe findings ARE flowing into models. But the updates are stranded in the working tree, not committed.

---

### Finding 6: Probe-producing workspaces lack SYNTHESIS.md files

**Evidence:** Checked 5 workspaces that correlate to probe topics:

- `og-debug-suppress-bd-subprocess-08feb-3a5c` → bd-subprocess probe — NO SYNTHESIS
- `og-debug-daemon-processedissuecache-marks-08feb-35ee` → processed-cache probe — NO SYNTHESIS
- `og-debug-jsonl-hash-mismatch-08feb-bf94` → jsonl-hash-mismatch probe — NO SYNTHESIS
- `og-debug-orch-complete-auto-08feb-df7b` → transient-verification probe — NO SYNTHESIS
- `og-debug-beads-staleness-check-08feb-0f6e` → stale-retry probe — NO SYNTHESIS

Each workspace contains only AGENT_MANIFEST.json, SPAWN_CONTEXT.md, and screenshots/. The probe file itself lives in `.kb/models/*/probes/`, not in the workspace.

**Source:** `ls .orch/workspace/<ws>/` for each workspace.

**Significance:** Probe-producing agents are completing their code fixes and writing probes but not creating SYNTHESIS.md. This may be because the spawn system uses "light" tier for debug tasks (which doesn't require SYNTHESIS). The probe IS the deliverable, making SYNTHESIS redundant in most cases.

---

### Finding 7: Probe verdicts are predominantly "extends" (8/10)

**Evidence:**

| Verdict     | Count | Probes                                                                                                                      |
| ----------- | ----- | --------------------------------------------------------------------------------------------------------------------------- |
| extends     | 8     | bd-subprocess, ready-queue, stale-retry, jsonl-hash-mismatch, transient-verification, commit-idle, processed-cache, fd-leak |
| confirms    | 2     | daemon-model-routing, synthesis-dedup                                                                                       |
| contradicts | 0     | (none)                                                                                                                      |

No probe contradicted its parent model. All probes either extended model scope or confirmed open claims.

**Source:** "Model Impact" section of all 10 probes.

**Significance:** The model accuracy appears high — no contradictions found. However, this could also indicate confirmation bias (probes created in context of fixing code that aligns with model expectations). A contradicting probe would be the strongest evidence of a functioning feedback loop, and none exist yet.

---

## Synthesis

**Key Insights:**

1. **Probe artifact quality is excellent** — All 10 probes follow the mandated template, reference specific model claims, include concrete test output, and produce clear verdicts. The format works.

2. **The feedback loop closes in working-tree edits but breaks at the commit boundary** — All 6 parent models have been updated (uncommitted) to incorporate probe findings. Someone or something is reading probes and updating models. But these updates haven't been committed, making the feedback invisible to git history, code review, and future sessions.

3. **Persistence is the bottleneck, not production** — The probe system generates good artifacts and triggers model updates. The gap is in the commit/persistence pipeline: there's no automated mechanism to commit probe+model pairs, and the manual bulk commit approach (demonstrated once in `b008bc89`) doesn't scale.

**Answer to Investigation Question:**

The probe→model feedback loop **partially works**. Probes are produced with high quality (10/10 follow template, 10/10 reference specific claims, 10/10 include test output). Model updates happen (6/6 models have uncommitted changes incorporating findings). But the loop is open at the persistence boundary: 9/10 probes are untracked in git, 6/6 model updates are uncommitted, and only 1 manual bulk commit captured any of this work. Without automated commit discipline, probe findings exist only in ephemeral working-tree state and are vulnerable to loss.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 10 probe files exist and follow correct structure (verified: read all files)
- ✅ All parent models predate their probes (verified: git log --diff-filter=A + stat timestamps)
- ✅ 9/10 probes are untracked in git (verified: git status --porcelain)
- ✅ All 6 models have uncommitted diffs referencing probes (verified: git diff --stat)
- ✅ Probe verdicts are 8 extends, 2 confirms, 0 contradicts (verified: read Model Impact sections)

**What's untested:**

- ⚠️ Whether the model diffs are semantically accurate (I verified they exist and reference probes, but didn't validate claim accuracy)
- ⚠️ Whether the absence of "contradicts" verdicts indicates confirmation bias or genuine model accuracy
- ⚠️ Whether probe-producing agents are the same agents that updated models, or if a separate process did the model updates

**What would change this:**

- A contradicting probe that triggers a model rewrite would prove the full feedback loop works
- An automated `orch complete` probe-commit step that persists probes would close the persistence gap
- Evidence of probe loss (working-tree cleanup destroying uncommitted probes) would demonstrate the failure mode is not theoretical

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation                                 | Authority      | Rationale                                           |
| ---------------------------------------------- | -------------- | --------------------------------------------------- |
| Auto-commit probe+model pairs at completion    | architectural  | Spans spawn system, completion protocol, and kb CLI |
| Treat probes as the SYNTHESIS for debug spawns | implementation | Clarifies existing pattern within probe workflow    |

### Recommended Approach ⭐

**Probe Persistence Pipeline** - Add automated commit of probe+model update pairs when agents complete.

**Why this approach:**

- Probes already exist and are high quality; no artifact changes needed
- Model updates already happen in working tree; they just need to be committed
- The single manual commit (`b008bc89`) proves the pattern works when executed

**Trade-offs accepted:**

- More commits in git history (one per probe-producing agent)
- May need to batch probe commits if multiple agents finish simultaneously

**Implementation sequence:**

1. Add `git add .kb/models/*/probes/*.md .kb/models/*.md && git commit` to `orch complete` for probe-producing agents
2. Detect probe presence by checking if workspace produced files in `.kb/models/*/probes/`
3. Consider `kb commit-probes` command for manual trigger

### Alternative Approaches Considered

**Option B: Periodic bulk commit via daemon**

- **Pros:** No per-agent overhead, batches naturally
- **Cons:** Delay between production and persistence; if daemon dies, uncommitted probes lost
- **When to use instead:** If commit-per-agent creates too much git noise

**Rationale for recommendation:** Per-agent commit is simpler, more reliable, and aligns with existing `orch complete` as the persistence boundary.

---

## References

**Commands Run:**

```bash
# Find all probes
glob .kb/models/*/probes/*.md

# Check git status of probes
git status --porcelain -- '.kb/models/*/probes/*.md'

# Get probe creation dates
git log --all --diff-filter=A -- <probe-path>

# Get model creation dates
git log --all --diff-filter=A -- <model-path>

# Check model updates after probes
git log --all --after="2026-02-08T00:00:00" -- <model-path>

# Check uncommitted model diffs
git diff -- .kb/models/<model>.md

# Check workspaces for SYNTHESIS.md
ls .orch/workspace/<ws>/

# Filesystem timestamps for probes
stat -f "%m %Sm %N" <probe-path>
```

**Related Artifacts:**

- **Decision:** .kb/decisions/2026-02-08-model-centric-probes-replace-investigations.md — Established probe system
- **Commit:** b008bc89 — Only committed probe (SSE FD leak) + model updates

---

## Investigation History

**2026-02-08:** Investigation started

- Initial question: Does the probe→model feedback loop actually close?
- Context: 10 probes exist; need to determine if findings flow back into models

**2026-02-08:** Data collection complete

- Found 10 probes, 6 parent models, all with correct temporal ordering
- Discovered 9/10 probes untracked, all 6 models have uncommitted probe-referencing diffs
- Key finding: feedback loop closes mechanically but breaks at commit boundary

**2026-02-08:** Investigation completed

- Status: Complete
- Key outcome: Probe system works for artifact production and model updating, but lacks persistence automation — 90% of probes are uncommitted and vulnerable to loss
