## Summary (D.E.K.N.)

**Delta:** On-demand resume model is the right session lifecycle for headless orchestrator — stateless per wakeup, lower blast radius, no context window accumulation risk. But the explain-back gate is the real design fork: headless orchestrator must produce its own explain-back text, creating a new comprehension tier between "daemon auto-closed" and "Dylan reviewed live."

**Evidence:** Current completion pipeline (`complete_pipeline.go`) has 7 interactive advisory gates that require stdin/stdout; stream-JSON pipe (orch-go-zxe2j) and session resume both work in Claude Code v2.1.83; DFM engine session proves orchestrator intermediary can invert meaning when forwarding without source verification.

**Knowledge:** The headless orchestrator is a comprehension amplifier, not an automation layer. Its value is producing briefs that meet Norm 2 ("translate, don't forward") — going to SYNTHESIS.md and source data, not parroting agent summaries. The design must preserve Dylan's final judgment authority while removing the bottleneck of conversation-gated review.

**Next:** Implement in three phases: (1) `orch complete --headless` mode that skips interactive gates and writes explain-back from SYNTHESIS + source, (2) daemon integration to wake headless orchestrator on Phase: Complete, (3) brief quality feedback loop. Three blocking questions created for Dylan's judgment.

**Authority:** architectural — Cross-component design affecting daemon, completion pipeline, orchestrator skill, and comprehension queue.

---

# Investigation: Design Daemon-Triggered Headless Orchestrator

**Question:** How should the daemon wake a headless orchestrator to perform completion review (read SYNTHESIS, decide follow-ups, connect to threads, write brief) without Dylan present?

**Started:** 2026-03-25
**Updated:** 2026-03-25
**Owner:** Architect agent (orch-go-exgxp)
**Phase:** Complete
**Next Step:** Dylan reviews blocking questions, then implementation issues created
**Status:** Complete
**Model:** orchestrator-session-lifecycle

**Patches-Decision:** N/A (new capability)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| orch-go-zxe2j (Claude Code external wakeup mechanisms) | extends | Yes — confirmed stream-JSON and resume both work | None |
| Comprehension artifacts thread (2026-03-24) | extends | Yes — brief-as-comprehension-artifact design is prerequisite | None |
| DFM engine session (orch-go-hjllu) | constrains | Yes — intermediary inversion pattern confirmed | None |
| Plan mode incompatibility decision (2026-02-26) | confirms | Yes — headless orchestrator faces same interactive gate problem | None |
| Completion workflow guide | extends | Yes — four-phase pipeline architecture confirmed | None |
| Orchestrator skill Norm 2 | constrains | Yes — "translate, don't forward" is the quality bar | None |

---

## Findings

### Finding 1: The completion pipeline has two distinct layers — mechanical and judgmental

**Evidence:** The four-phase pipeline in `complete_pipeline.go` separates cleanly:

- **Phase 1 (Target Resolution)** and **Phase 2 (Verification Gates)** are fully mechanical — find workspace, run V0-V3 gates, check SYNTHESIS.md exists. No human judgment required.
- **Phase 3 (Advisories)** is where ALL interactive judgment lives: discovered work disposition, probe verdict processing, knowledge maintenance, explain-back gate (Gate 1), behavioral verification (Gate 2).
- **Phase 4 (Lifecycle Transition)** is mechanical — close beads, archive workspace, update handoff.

The daemon already runs Phases 1, 2, and 4 via `ProcessCompletion()` in `completion_processing.go`. It skips Phase 3 entirely, instead labeling with `daemon:ready-review` and `comprehension:pending`.

**Source:** `cmd/orch/complete_pipeline.go` (four-phase architecture), `pkg/daemon/completion_processing.go:355-381` (ProcessCompletion)

**Significance:** The headless orchestrator's job is specifically Phase 3 — the advisory layer. This is a well-bounded scope. We don't need to redesign the whole pipeline, just make Phase 3 executable without stdin/stdout.

---

### Finding 2: Stream-JSON pipe vs session resume have fundamentally different failure modes

**Evidence:** From orch-go-zxe2j investigation:

| Property | Stream-JSON Pipe | Session Resume |
|----------|-----------------|----------------|
| Context | Accumulates across wakeups (warm) | Fresh per wakeup (cold) |
| Process | Single long-lived process | New process per wakeup |
| Failure blast radius | Pipe break kills ALL accumulated context | Single wakeup fails, next is clean |
| Context window | Grows unboundedly with each completion | Fixed per completion |
| Concurrency | Single-threaded (one pipe) | Naturally parallelizable |
| Daemon integration | Must manage pipe lifecycle | Fire-and-forget per completion |

**Source:** `.kb/briefs/orch-go-zxe2j.md`, Claude Code `--input-format stream-json` flag, `claude -p --resume <id>` command

**Significance:** The persistent pipe sounds attractive (warm context, no resume overhead) but has worse failure modes for daemon integration. A pipe break is catastrophic — all accumulated context from prior completions is lost. Resume is stateless; each completion is independent. This maps cleanly to the daemon's fire-and-forget completion processing model.

---

### Finding 3: The explain-back gate is the hardest design fork — it exists to verify HUMAN comprehension

**Evidence:** The explain-back gate (`pkg/orch/completion.go`) requires `--explain` with non-empty text. Its purpose is NOT to verify agent quality — it's to verify that Dylan (via orchestrator) has comprehended what the agent built. The gate stores the explanation as a beads comment and checkpoint.

Current gate behavior:
```
Gate 1 (explain-back): Orchestrator explains what agent built → proves comprehension
Gate 2 (behavioral): Orchestrator confirms behavior verified → proves testing
```

A headless orchestrator can read SYNTHESIS.md and produce an explain-back text. But this changes the gate's meaning: instead of "a human comprehended this," it becomes "an AI read the synthesis and summarized it." This is the intermediary inversion risk from the DFM engine session — the headless orchestrator could parrot agent summaries without actually comprehending them.

**Source:** `pkg/orch/completion.go` (RunExplainBackGate), `.kb/briefs/orch-go-hjllu.md` (DFM intermediary inversion), orchestrator skill Norm 2

**Significance:** This is a framing question, not an implementation question. Two valid interpretations:
1. Explain-back gate verifies HUMAN comprehension → headless orchestrator cannot satisfy it → gate must be deferred for Dylan
2. Explain-back gate verifies QUALITY comprehension → headless orchestrator can satisfy it IF it goes to source data (Norm 2) → gate can be satisfied headlessly

---

### Finding 4: The comprehension queue already has the right throttling mechanism

**Evidence:** `pkg/daemon/comprehension_queue.go` implements a simple but effective throttle:
- `comprehension:pending` label marks work not yet comprehended
- `CheckComprehensionThrottle()` returns false when count >= threshold (default 5)
- Daemon pauses spawning when threshold exceeded

The headless orchestrator's brief-writing should decrement this counter. But "brief written by headless orchestrator" is different from "Dylan read and understood." The current counter conflates two things: (1) mechanical processing status and (2) human comprehension status.

**Source:** `pkg/daemon/comprehension_queue.go:94-109`, comprehension artifacts thread (2026-03-24)

**Significance:** The comprehension queue needs a two-state model: `comprehension:processed` (headless orchestrator reviewed, brief written) and `comprehension:unread` (Dylan hasn't read the brief yet). The throttle gates on `unread`, not `processed`.

---

### Finding 5: The same orchestrator skill constraint applies — no separate headless skill

**Evidence:** The spawn context explicitly states: "The headless orchestrator runs the same skill as the live orchestrator. No separate 'headless skill.'" This is correct per the spawned orchestrator pattern guide: orchestrators use the same skill regardless of spawn mode.

The orchestrator skill's three jobs map to headless completion:
1. **COMPREHEND** — Read SYNTHESIS.md, go to source data, validate claims → yes, headless can do this
2. **TRIAGE** — Review discovered work, decide follow-ups → yes, headless can do this
3. **SYNTHESIZE** — Connect findings to threads, write brief → yes, headless can do this

The skill's Norm 2 ("translate, don't forward") is the quality bar. The headless orchestrator must:
- Read SYNTHESIS.md AND go to source files referenced in Delta section
- Validate that the synthesis matches actual code changes (git diff)
- Produce brief in Frame/Resolution/Tension format using writing primers
- NOT just summarize SYNTHESIS.md (that's forwarding)

**Source:** Orchestrator skill SKILL.md (Norm 2), spawned orchestrator pattern guide, spawn context constraint

**Significance:** No new skill infrastructure needed. The challenge is not capability but quality — can the orchestrator skill produce good synthesis without Dylan's real-time feedback? The brief quality becomes the retroactive signal.

---

### Finding 6: Defect class exposure analysis

**Evidence:** Analyzing the proposed design against the defect class taxonomy:

| Class | Exposure | Mitigation |
|-------|----------|------------|
| Class 0 (Scope Expansion) | LOW — headless completion is a new code path, not widening an existing scanner | N/A |
| Class 2 (Multi-Backend Blindness) | MEDIUM — headless orchestrator uses Claude CLI backend only, not OpenCode API | Ensure resume injection works regardless of backend |
| Class 3 (Stale Artifact Accumulation) | HIGH — headless orchestrator sessions accumulate if daemon fails to clean up | Session TTL + cleanup in daemon periodic tasks |
| Class 5 (Contradictory Authority) | HIGH — two sources of "comprehended" status (headless orchestrator vs Dylan) | Two-state label model (processed vs unread) |
| Class 6 (Duplicate Action) | MEDIUM — daemon could wake multiple headless orchestrators for same completion | Dedup via `daemon:ready-review` label check before wakeup |

**Source:** `.kb/models/defect-class-taxonomy/model.md`

**Significance:** Class 5 (Contradictory Authority) is the highest risk — the system needs to distinguish "orchestrator reviewed" from "Dylan comprehended" or we get oscillating state between two authority sources.

---

## Synthesis

**Key Insights:**

1. **The headless orchestrator is Phase 3 of the completion pipeline, externalized** — Instead of building a new system, we're making the existing advisory layer (discovered work, probe verdicts, knowledge maintenance, explain-back) executable without stdin/stdout. The daemon already handles Phases 1, 2, and 4. The gap is specifically Phase 3.

2. **On-demand resume beats persistent pipe for daemon integration** — The fire-and-forget model (resume per completion) matches the daemon's existing architecture. No pipe lifecycle management, no context window accumulation, natural parallelism, bounded blast radius. The warm-context benefit of persistent pipes is illusory — each completion is independent work that benefits more from fresh perspective than accumulated context (DFM engine evidence: fresh session produced better comprehension than context-laden one).

3. **The explain-back gate must split into two tiers** — Tier A: "AI orchestrator reviewed and produced brief" (headless can satisfy). Tier B: "Human confirmed comprehension" (Dylan satisfies by reading brief). This isn't a compromise — it's a more accurate model. Currently the gate conflates "orchestrator typed an explanation" with "Dylan comprehended." Making this explicit is better.

4. **Brief quality is the retroactive quality signal** — Without Dylan present, there's no real-time quality check. But briefs land in the reading queue. If Dylan reads a brief and finds it shallow/wrong, that's a quality signal that can feed back into skill guidance. This is the same pattern as completion verification: trust the process, measure retroactively, improve iteratively.

**Answer to Investigation Question:**

The daemon should wake a headless orchestrator using Claude Code session resume (`claude -p --resume <id> "completion event"`) on each Phase: Complete detection. The headless orchestrator:

1. Reads SYNTHESIS.md and goes to source files (Norm 2 compliance)
2. Makes discovered work disposition decisions (auto-file for mechanical findings, escalate ambiguous ones)
3. Produces explain-back text from source-verified understanding
4. Writes BRIEF.md in Frame/Resolution/Tension format
5. Connects findings to active threads
6. Labels `comprehension:processed` (replaces `comprehension:pending`)
7. Dylan later reads brief, marks `comprehension:read` (or doesn't — the brief is there)

The key constraint: the headless orchestrator runs the SAME skill as the live orchestrator. If it can't produce good synthesis without Dylan, that's a skill quality problem to fix, not a mode to work around.

---

## Structured Uncertainty

**What's tested:**

- ✅ Stream-JSON and session resume both work in Claude Code v2.1.83 (verified: orch-go-zxe2j investigation with actual CLI tests)
- ✅ Daemon completion processing already handles Phases 1, 2, 4 mechanically (verified: read `completion_processing.go`)
- ✅ BRIEF.md template and writing primers produce good briefs (verified: two briefs produced and quality confirmed per comprehension artifacts thread)
- ✅ Comprehension queue throttle works (verified: `comprehension_queue.go` implementation read)

**What's untested:**

- ⚠️ Session resume for orchestrator skill specifically (tested for worker skills, not orchestrator)
- ⚠️ Headless orchestrator quality without Dylan's real-time feedback (hypothesis: brief quality is measurable retroactively)
- ⚠️ Concurrent resume safety (what if daemon tries to resume same session twice)
- ⚠️ Context window sufficiency — orchestrator skill + completion context + source files may exceed limits
- ⚠️ Thread connection quality — headless orchestrator must map findings to active threads without conversation

**What would change this:**

- If session resume can't load the orchestrator skill context (skill too large for resume), persistent pipe becomes necessary
- If headless brief quality is consistently shallow (measured over 10+ briefs), the design needs a quality gate before publishing
- If Claude Code removes or changes `--resume` behavior, fallback to stream-JSON pipe required

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| On-demand resume as session lifecycle | architectural | Cross-component: daemon lifecycle, Claude CLI integration, completion pipeline |
| Two-state comprehension label model | architectural | Changes semantics of comprehension:pending across daemon and orchestrator |
| Explain-back gate tier split | strategic | Changes what "comprehension" means in the system — value judgment |
| Headless mode for orch complete | implementation | New flag on existing command, within completion pipeline scope |
| Brief quality feedback mechanism | architectural | Crosses dashboard, reading queue, and skill guidance boundaries |

### Recommended Approach: On-Demand Resume with Deferred Human Gate

**One sentence:** Daemon detects Phase: Complete, wakes headless orchestrator via `claude -p --resume`, orchestrator produces brief + explain-back, daemon labels `comprehension:processed`, Dylan reads brief async and confirms comprehension.

**Why this approach:**
- Matches daemon's existing fire-and-forget model (Finding 2)
- Isolates headless orchestrator scope to Phase 3 advisories (Finding 1)
- Preserves Dylan's comprehension authority via two-state label model (Finding 4)
- Uses existing injection mechanisms already proven to work (Finding 2)
- No new skill needed — same orchestrator skill, same behavioral norms (Finding 5)

**Trade-offs accepted:**
- Headless orchestrator may produce lower-quality synthesis than live conversation (accepted: measurable retroactively via brief quality)
- Dylan still needs to read briefs — system doesn't eliminate human comprehension step (accepted: this is correct — the goal is to move comprehension from synchronous to async)
- Concurrent wakeup race possible if daemon polls faster than orchestrator completes (accepted: mitigated by label-based dedup, same pattern as spawn dedup L1)

**Implementation sequence:**

1. **`orch complete --headless` mode** — New flag that executes Phase 3 advisories non-interactively. Discovered work: auto-file all (no prompt). Probe verdicts: auto-update uncontested, escalate contested. Knowledge maintenance: skip (not actionable without conversation). Explain-back: generate from SYNTHESIS.md + source verification. Behavioral verification: skip (deferred to Dylan).

2. **Daemon headless orchestrator integration** — On Phase: Complete detection, instead of just labeling `daemon:ready-review`, daemon calls `claude -p --resume <orch-session-id> "Completion event: <beads-id> Phase: Complete. Run orch complete --headless <beads-id>"`. This wakes the orchestrator, which runs the headless completion and writes BRIEF.md.

3. **Two-state comprehension labels** — Replace single `comprehension:pending` with `comprehension:processed` (headless orchestrator reviewed) and `comprehension:unread` (Dylan hasn't read brief). Throttle gates on `unread` count. `orch comprehension review <id>` transitions `unread` → `read`.

### Alternative Approaches Considered

**Option B: Persistent Stream-JSON Pipe**
- **Pros:** Warm context (remembers prior completions), single process management, potentially faster per-wakeup
- **Cons:** Unbounded context window growth, catastrophic pipe break failure, single-threaded bottleneck, complex lifecycle management for daemon
- **When to use instead:** If completions are highly correlated (e.g., all from same epic) and cross-completion context would improve synthesis quality

**Option C: Dedicated Synthesis Agent (not orchestrator)**
- **Pros:** Could be lighter weight, purpose-built for brief generation, no orchestrator skill overhead
- **Cons:** Violates "same skill" constraint from spawn context, duplicates orchestrator capability, creates maintenance burden for two synthesis paths
- **When to use instead:** If orchestrator skill proves too heavy for headless completion (context window limits)

**Option D: Wait for Claude Code Channels MCP**
- **Pros:** Purpose-designed push mechanism, likely better API stability, Anthropic-supported
- **Cons:** Research preview only, unknown timeline, may never ship or may ship differently
- **When to use instead:** If Channels MCP ships with stable API before implementation begins

**Rationale for recommendation:** Option A (on-demand resume) wins on failure isolation, daemon integration simplicity, and adherence to existing patterns. The DFM engine evidence (Finding 3) specifically supports stateless per-wakeup: fresh perspective produced better comprehension than accumulated context.

---

### Implementation Details

**What to implement first:**
- `orch complete --headless` flag — this is the foundational capability, testable independently of daemon integration
- Headless advisory execution — non-interactive versions of discovered work disposition and explain-back generation
- Brief writing from headless context — same BRIEF.md template, source-verified

**Things to watch out for:**
- ⚠️ Context window limits — orchestrator skill (~1251 lines) + SYNTHESIS.md + source files may approach token limits. Test with real completions before daemon integration.
- ⚠️ Concurrent wakeup race — if daemon polls every 60s and headless completion takes >60s, daemon may try to wake again. Need label-based dedup before wakeup.
- ⚠️ Quality regression signal lag — if headless briefs are consistently bad, Dylan won't notice until reading queue builds up. Consider adding a quality self-check (brief length, tension section non-empty, source references present).
- ⚠️ Thread connection quality — mapping findings to active threads requires reading thread state. Thread files may be large; may need truncated thread summaries.

**Areas needing further investigation:**
- What's the actual context window cost of orchestrator skill + completion context + source files?
- Can session resume load the orchestrator skill reliably? (Needs practical test with `claude -p --resume`)
- What's the right quality threshold for brief self-check before publishing?

**Success criteria:**
- ✅ `orch complete --headless <id>` produces BRIEF.md that passes the "could Dylan act on this?" test
- ✅ Daemon-triggered headless completion runs end-to-end without human intervention
- ✅ Two-state comprehension labels correctly gate spawning on Dylan's reading speed, not daemon processing speed
- ✅ Brief quality (measured by Dylan's reaction) is >= current orchestrator live synthesis quality
- ✅ No increase in "comprehension inversion" incidents (DFM-style meaning flip)

---

## Blocking Questions (Created as Beads Entities)

### Q1: Session Lifecycle (orch-go-e9zpl)
**Should the headless orchestrator use persistent stream-JSON pipe (warm context) or on-demand resume (stateless per wakeup)?**
- Authority: architectural
- Subtype: judgment
- Recommendation: On-demand resume (see Finding 2)
- What changes: If persistent pipe, need pipe lifecycle management in daemon, bounded context window strategy, pipe-break recovery. If resume, simpler daemon integration but cold-start overhead per completion.

### Q2: Explain-Back Gate (orch-go-fc1xv)
**Should the headless orchestrator bypass explain-back gate (Gate 1) or produce its own explanation text?**
- Authority: strategic
- Subtype: judgment
- Recommendation: Produce its own explain-back text with source verification, creating a new "AI-reviewed" comprehension tier
- What changes: If bypass, headless completions are explicitly "unreviewed" — Dylan must do full review later. If produce, headless completions have intermediate trust level — Dylan reviews brief, not raw synthesis.

### Q3: Comprehension Interaction (orch-go-y9ey6)
**Does headless completion review count as "comprehended" or does Dylan still mark-as-read separately?**
- Authority: strategic
- Subtype: framing
- Recommendation: Two-state model — `comprehension:processed` (headless reviewed) and `comprehension:unread` (Dylan hasn't read brief). Throttle on unread.
- What changes: If headless = comprehended, Dylan's reading becomes optional and comprehension quality may degrade. If two-state, Dylan still gates but from briefs instead of raw synthesis — faster but still human-verified.

---

## References

**Files Examined:**
- `pkg/daemon/completion_processing.go` — daemon completion loop, ProcessCompletion, CompletionOnce
- `pkg/daemon/comprehension_queue.go` — comprehension:pending label management, throttle
- `cmd/orch/complete_pipeline.go` — four-phase completion pipeline
- `cmd/orch/complete_verification.go` — verification gate dispatcher
- `pkg/verify/check.go` — V0-V3 verification gates
- `pkg/orch/completion.go` — explain-back gate implementation
- `skills/src/meta/orchestrator/SKILL.md` — orchestrator behavioral norms, Norm 2
- `.kb/guides/headless.md` — headless spawn mode reference
- `.kb/guides/completion.md` — completion workflow reference
- `.kb/guides/daemon.md` — daemon autonomous operation reference
- `.kb/guides/spawned-orchestrator-pattern.md` — hierarchical completion model
- `.kb/guides/orchestrator-session-management.md` — orchestrator session architecture

**External Documentation:**
- None (all evidence from codebase and internal knowledge base)

**Related Artifacts:**
- **Brief:** `.kb/briefs/orch-go-zxe2j.md` — Claude Code external wakeup mechanisms
- **Thread:** `.kb/threads/2026-03-24-comprehension-artifacts-async-synthesis-delivery.md` — Async brief system design
- **Brief:** `.kb/briefs/orch-go-hjllu.md` — DFM engine session failure (intermediary inversion evidence)
- **Decision:** `.kb/decisions/2026-02-26-plan-mode-incompatible-with-daemon-spawned-agents.md` — Interactive gate incompatibility precedent
- **Question:** orch-go-e9zpl — Session lifecycle fork
- **Question:** orch-go-fc1xv — Explain-back gate fork
- **Question:** orch-go-y9ey6 — Comprehension interaction fork

---

## Investigation History

**2026-03-25 T0:** Investigation started
- Initial question: How to design daemon-triggered headless orchestrator for automatic completion review
- Context: Spawned as architect from orchestrator session, full tier

**2026-03-25 T1:** Context gathering complete
- Read headless guide, completion guide, daemon guide, orchestrator skill, comprehension queue code
- Read prior art: orch-go-zxe2j (wakeup mechanisms), comprehension artifacts thread, DFM engine session
- Read completion pipeline code: four-phase architecture, advisory gates, verification gates

**2026-03-25 T2:** Fork navigation complete
- 6 forks identified, 3 navigated with substrate, 3 surfaced as blocking questions
- Key insight: headless orchestrator = Phase 3 advisories externalized
- DFM evidence strongly supports stateless (resume) over stateful (pipe) model

**2026-03-25 T3:** Investigation complete
- Produced investigation with 6 findings, 3 blocking questions, implementation sequence
- Status: Complete, pending Dylan's judgment on blocking questions
