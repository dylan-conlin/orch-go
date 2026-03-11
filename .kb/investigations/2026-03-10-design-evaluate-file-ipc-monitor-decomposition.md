## Summary (D.E.K.N.)

**Delta:** All three Dispatch-inspired patterns (file IPC, auto-surfacing monitor, task decomposition) are either already present in orch under different names, or solve problems orch doesn't empirically have — none warrant adoption.

**Evidence:** Mapped actual infrastructure dependencies for orch send (3-tier: OpenCode API → OpenCode lookup → tmux, no beads RPC), orch question (4-tier: workspace+OpenCode → tmux discovery → OpenCode matching → tmux scraping, no beads RPC), and daemon question detection (beads RPC required). Cross-referenced against Dispatch's file IPC protocol, monitor script, and decomposition model. Reviewed context exhaustion data showing 91% completion for largest-context agents.

**Knowledge:** The task premise overstates orch's infrastructure dependencies — orch send and orch question already have transport-agnostic fallbacks that work without beads daemon. The daemon's question detection is periodic but already auto-surfaces via desktop notifications. Context exhaustion is empirically not orch's failure mode. Dispatch's patterns are optimized for interactive parallelization with a human present; orch's overnight autonomous model has different pressure points.

**Next:** Close. No implementation issues created — all three recommendations are "don't adopt." One tuning suggestion (daemon poll interval for question detection) is the only actionable item, and it's a minor config change not an architectural decision.

**Authority:** strategic - All three decisions involve architectural direction and value judgments about orch's operating model trajectory.

---

# Investigation: Evaluate File-Based IPC, Auto-Surfacing Monitor, and Task Decomposition Patterns from Dispatch

**Question:** Should orch-go adopt three patterns from Dispatch: (1) file-based IPC as fallback for agent Q&A, (2) auto-surfacing monitor for agent questions, (3) task decomposition with fresh context windows?

**Started:** 2026-03-10
**Updated:** 2026-03-10
**Owner:** spawned agent (architect)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-10-inv-compare-dispatch-to-orch-go.md | extends | yes | Corrects premise — orch send/question don't require beads RPC |
| .kb/investigations/2026-03-04-inv-context-volume-stall-rate.md (referenced, not read directly) | informs | partially — used finding via prior knowledge | none |

---

## Findings

### Finding 1: Task Premise Overstates Orch's Infrastructure Dependencies

**Evidence:** The task states "orch send requires beads RPC and orch question requires tmux scraping." Code inspection reveals:

- **orch send** (`cmd/orch/send_cmd.go`) has a 3-tier fallback: (1) OpenCode API via workspace session ID, (2) OpenCode API via session lookup, (3) tmux send-keys as last resort. Beads RPC is never called — session IDs come from workspace files.
- **orch question** (`cmd/orch/question_cmd.go`) has a 4-tier fallback: (1) workspace file + OpenCode API, (2) tmux window discovery, (3) OpenCode session matching, (4) tmux pane scraping. Beads RPC is never called.
- **Daemon question detection** (`pkg/daemon/question_detector.go`) DOES require beads RPC — it polls `in_progress` issues for Phase: QUESTION comments.

**Source:** `cmd/orch/send_cmd.go`, `cmd/orch/question_cmd.go`, `pkg/daemon/question_detector.go`, `pkg/daemon/recovery.go`

**Significance:** The user-facing commands (send, question) are already transport-agnostic. Only the daemon's automated detection depends on beads RPC. This narrows the "file IPC as fallback" question to: does the daemon need a beads-free detection path? Since beads is a core dependency expected to always run (same as tmux), the answer is likely no.

---

### Finding 2: Orch Already Has Auto-Surfacing — The Gap Is Latency, Not Capability

**Evidence:** The daemon's `RunPeriodicQuestionDetection()` runs on the `PhaseTimeoutInterval`, scans all active agents for Phase: QUESTION comments, tracks previously-notified agents in `d.questionNotified` (prevents duplicates), and triggers `notify.QuestionPending()` — a macOS desktop notification.

Dispatch's monitor polls at 3-second intervals per worker and triggers Claude Code's task-notification system. The key difference: Dispatch needs sub-5s latency because a human is waiting at the terminal. Orch's overnight fleets don't have a human waiting.

**Source:** `pkg/daemon/question_detector.go` (RunPeriodicQuestionDetection), Dispatch `docs/ipc-protocol.md` (3s poll interval)

**Significance:** Orch already auto-surfaces questions. If daytime latency matters, reducing the daemon poll interval from N minutes to 30 seconds achieves the same result without new infrastructure. One centralized monitor (daemon) scales better than N per-worker monitors at orch's 10+ concurrent agent scale.

---

### Finding 3: Context Exhaustion Is Not Orch's Empirical Failure Mode

**Evidence:** Prior investigation (2026-03-04) found that agents with the LARGEST contexts (Q4, 65K+ tokens) had 91% completion rate, while agents with the SMALLEST contexts (Q1, <45K) had only 21% completion. The true stall rate is 2-4% and is unrelated to context size. The actual problem is protocol weight — 97% of feature-impl agents skip SYNTHESIS.md due to ceremony overhead at session end.

Orch already has context risk monitoring (`pkg/verify/context_risk.go`: 150K warning, 180K critical) and the rework pattern (`cmd/orch/rework_cmd.go`) for recovery when agents do fail. Context validation at spawn time (`pkg/spawn/tokens.go`) blocks spawns that would exceed 150K tokens.

**Source:** Prior knowledge from .kb/investigations/2026-03-04 (context-volume-stall-rate), `pkg/verify/context_risk.go`, `pkg/spawn/tokens.go`, `cmd/orch/rework_cmd.go`

**Significance:** Dispatch decomposes because it's optimized for interactive parallelization ("work on 3 things at once while I watch"). Orch's daemon already parallelizes by spawning independent agents for independent beads issues. Adding upfront decomposition logic (planning step, coordination mechanism, synthesis merge) is significant complexity for a problem the data says orch doesn't have.

---

### Finding 4: File-Based IPC Creates Defect Class 5 Risk (Contradictory Authority Signals)

**Evidence:** Orch agents currently communicate state via beads comments (`bd comments add <id> "Phase: X"`). The daemon, orch status, orch complete, and orch review all read beads comments as the authoritative channel. Adding file-based IPC creates a second communication channel. When two channels exist, they can disagree — a question written to a file but not reflected in beads comments, or vice versa.

This is textbook Defect Class 5 (Contradictory Authority Signals): "Multiple sources of truth disagree, fixes oscillate." The fix pattern for Class 5 is "single canonical derivation" — the opposite of adding a parallel channel.

**Source:** `.kb/models/defect-class-taxonomy/model.md` (Class 5), `pkg/daemon/question_detector.go` (beads as canonical channel)

**Significance:** File-based IPC would be a second IPC protocol alongside beads comments. Orch already has 77K+ lines and 7 documented defect classes. Adding a pattern that maps to a known defect class is the wrong direction.

---

### Finding 5: Orch's Multi-Agent Model IS Task Decomposition (at Planning Level)

**Evidence:** Dispatch decomposes at execution time: one dispatcher session creates a plan, then spawns workers for each checklist item. Orch decomposes at planning time: the architect skill produces issues (one per component), the daemon spawns independent agents for each issue, each agent works a self-contained scope.

The spawned-orchestrator pattern (`.kb/guides/spawned-orchestrator-pattern.md`) already exists for cases where a task needs further decomposition — an architect is spawned to break work into sub-issues. This is the same logical flow as Dispatch's decomposition, just mediated by beads issues instead of plan.md checklists.

**Source:** `.kb/guides/spawned-orchestrator-pattern.md`, `pkg/daemon/daemon.go` (spawns from triage:ready issues), architect skill (Phase 5d: Decomposition)

**Significance:** Formal execution-time decomposition (splitting a running agent's context into sub-contexts) would duplicate what the planning-level decomposition already provides. The daemon + beads + architect pattern is orch's decomposition. It works at a different layer than Dispatch's, but achieves the same outcome: independent work units with fresh context.

---

## Synthesis

**Key Insights:**

1. **Operating model dictates pattern value** — Dispatch optimizes for "human at terminal, wants fast parallel work." Orch optimizes for "fleet runs overnight, human reviews in morning." File IPC's 3-second latency matters for the first model; it's irrelevant for the second. The patterns are correct for Dispatch but wrong for orch.

2. **Orch's actual gaps are not what the premise assumes** — orch send/question already work without beads RPC. Auto-surfacing already exists via daemon + notifications. Context exhaustion is empirically not the failure mode. The real gaps (protocol weight causing SYNTHESIS.md skips, daemon poll latency for daytime use) are much smaller and don't require new architectural patterns.

3. **Adding infrastructure to solve non-problems is how 77K lines happened** — Every pattern from Dispatch is sensible in isolation. But orch's complexity budget is already stretched. The discipline is saying "no" to patterns that solve hypothetical problems or problems that belong to a different operating model. Principle: "evolve by distinction" — Dispatch's distinction is interactive parallelization; orch's is autonomous fleet management. The patterns should stay distinct.

**Answer to Investigation Question:**

**Decision 1 (File-based IPC as fallback): No.** The premise is incorrect — orch send/question don't require beads RPC. Only daemon detection does, and beads is a core always-on dependency. Adding file IPC creates Defect Class 5 risk (contradictory authority signals) with a second communication channel. No implementation justified.

**Decision 2 (Auto-surfacing monitor): Already exists, minor tuning only.** The daemon's `RunPeriodicQuestionDetection()` + desktop notifications already auto-surface. If daytime latency matters, reduce the daemon poll interval to 30s. One centralized monitor > N per-worker monitors at orch's scale. No new pattern needed.

**Decision 3 (Task decomposition): No.** Empirical data shows context exhaustion isn't the failure mode (91% completion for largest contexts). Orch already decomposes at the planning level via architect → sub-issues → daemon spawns. Formal execution-time decomposition adds planning cost, coordination complexity, and synthesis overhead for a problem the data says doesn't exist.

---

## Structured Uncertainty

**What's tested:**
- ✅ orch send does not require beads RPC (verified: read cmd/orch/send_cmd.go, traced all code paths)
- ✅ orch question does not require beads RPC (verified: read cmd/orch/question_cmd.go, traced all code paths)
- ✅ Daemon question detection requires beads RPC (verified: read pkg/daemon/question_detector.go)
- ✅ Dispatch IPC uses atomic file writes with 3s poll (verified: read docs/ipc-protocol.md and SKILL.md)
- ✅ Context exhaustion doesn't correlate with stalls (verified via prior investigation findings)

**What's untested:**
- ⚠️ Whether daemon poll interval is actually the bottleneck for daytime Q&A latency (not measured — would need timing data from actual question-detection events)
- ⚠️ Whether the spawned-orchestrator pattern effectively replaces Dispatch-style decomposition at scale (pattern exists but usage frequency unknown)
- ⚠️ Whether beads daemon actually has availability gaps in practice (assumed always-on, not measured)

**What would change this:**
- Evidence of frequent beads daemon outages causing missed questions would justify file-based IPC
- Evidence of agents exhausting context at high rates on complex tasks would justify decomposition
- Evidence of questions sitting undetected for hours during daytime use would justify faster polling or a new monitor pattern

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Don't adopt file-based IPC | strategic | Cross-system architectural direction; involves orch's operating model identity |
| Don't adopt new monitor pattern | strategic | Involves whether orch should shift toward interactive mode |
| Don't architect formal decomposition | strategic | Would reshape spawn/daemon/verify pipeline; value judgment about where orch invests |
| Reduce daemon poll interval for daytime use (optional) | implementation | Config knob within existing architecture, fully reversible |

### Recommended Approach: Maintain Current Architecture

**No new patterns** — Orch's existing infrastructure already provides the capabilities that Dispatch's patterns target, just at different layers appropriate to orch's operating model.

**Why this approach:**
- Avoids Defect Class 5 (contradictory authority signals) from dual IPC channels
- Respects empirical evidence that context exhaustion isn't the failure mode
- Keeps complexity budget for actual gaps (protocol weight, SYNTHESIS skip rate)

**Trade-offs accepted:**
- Daytime question detection has higher latency than Dispatch's 3s (acceptable for overnight fleet model)
- No automatic decomposition means complex tasks may need manual architect → sub-issue routing (acceptable because the spawned-orchestrator pattern handles this)

**Optional tuning (implementation authority):**
1. Reduce daemon question-detection poll interval to 30s for daytime responsiveness
2. This is a config change in `pkg/daemon/daemon.go`, not an architectural change

### Alternative Approaches Considered

**Option B: File-based IPC as beads fallback**
- **Pros:** Zero-dependency communication; crash-recoverable
- **Cons:** Creates second IPC channel (Defect Class 5); requires all consumers (daemon, orch status, orch complete) to check both channels; beads daemon is rarely unavailable
- **When to use instead:** If beads is deprecated or reliability drops below 95%

**Option C: Per-worker monitor scripts (Dispatch-style)**
- **Pros:** Sub-5s question detection; per-worker isolation
- **Cons:** Doesn't scale to 10+ agents (10 bash loops); requires Claude Code task-notification integration that orch's tmux backend doesn't have; duplicates daemon capability
- **When to use instead:** If orch pivots to interactive use (human-at-terminal model)

**Option D: Formal execution-time decomposition**
- **Pros:** Fresh context per subtask; natural parallelization
- **Cons:** Requires planning step (consumes tokens), coordination mechanism (new infrastructure), synthesis step (merge results); solves a problem data says doesn't exist
- **When to use instead:** If stall rate data changes — if context exhaustion becomes a real failure mode (currently 2-4%)

### Defect Class Exposure

- **Class 5 (Contradictory Authority Signals):** File-based IPC + beads comments = two channels that can disagree. Mitigated by: not adopting.
- **Class 2 (Multi-Backend Blindness):** Decomposition would need to work for both OpenCode and Claude CLI backends. Mitigated by: not adopting.
- **Class 1 (Filter Amnesia):** New IPC channel would need filtering in all existing consumers (status, complete, review). Mitigated by: not adopting.

---

## References

**Files Examined:**

Orch-go:
- `cmd/orch/send_cmd.go` - Send command infrastructure dependencies (3-tier fallback)
- `cmd/orch/question_cmd.go` - Question extraction infrastructure (4-tier fallback)
- `pkg/daemon/question_detector.go` - Periodic question detection (beads RPC required)
- `pkg/daemon/recovery.go` - Daemon recovery and health
- `pkg/verify/context_risk.go` - Context exhaustion thresholds (150K warning, 180K critical)
- `pkg/spawn/tokens.go` - Spawn-time context validation
- `cmd/orch/rework_cmd.go` - Rework pattern for agent recovery
- `pkg/spawn/rework.go` - Rework context injection
- `pkg/session/session.go` - Session management and checkpoint thresholds

Dispatch:
- `~/Documents/personal/dispatch/docs/ipc-protocol.md` - File IPC specification
- `~/Documents/personal/dispatch/skills/dispatch/SKILL.md` - Core implementation (v2.0.0)
- `~/Documents/personal/dispatch/docs/architecture.md` - Architecture and flow

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-10-inv-compare-dispatch-to-orch-go.md` - Comparative analysis this design extends
- **Model:** `.kb/models/defect-class-taxonomy/model.md` - Defect Class 5 (Contradictory Authority Signals)
- **Guide:** `.kb/guides/spawned-orchestrator-pattern.md` - Existing decomposition pattern

---

## Investigation History

**2026-03-10:** Investigation started
- Question: Should orch adopt file-based IPC, auto-surfacing monitor, and task decomposition from Dispatch?
- Context: Spawned from Dispatch comparison investigation to make three concrete architectural decisions

**2026-03-10:** Infrastructure dependency mapping complete
- orch send: 3-tier fallback, no beads RPC required (corrects task premise)
- orch question: 4-tier fallback, no beads RPC required (corrects task premise)
- Daemon detection: beads RPC required (confirmed)
- Context exhaustion: 91% completion for largest contexts (contradicts decomposition premise)

**2026-03-10:** Investigation completed
- Status: Complete
- Key outcome: All three patterns declined — orch already has equivalent capabilities at appropriate layers, and the problems Dispatch's patterns solve aren't orch's empirical failure modes
