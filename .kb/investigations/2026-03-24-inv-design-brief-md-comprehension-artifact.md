## Summary (D.E.K.N.)

**Delta:** BRIEF.md is a 3-section comprehension artifact (Frame/Resolution/Tension) produced by full-tier agents alongside SYNTHESIS.md, written for Dylan using the 4 writing primers, delivered to `.kb/briefs/` on completion.

**Evidence:** Current system has all pieces — SYNTHESIS.md (orchestrator-facing), comprehension:pending label (14 gates model, gate 13 explain-back), 4 writing primers (story first, earn abstractions, say what it felt like, the turn is the piece), daemon completion hooks — but comprehension is conversation-gated, not reading-gated.

**Knowledge:** The brief must provoke conversation, not replace it. Ending with a tidy summary creates false comprehension — Dylan reads, marks as read, never has the reactive moment. Briefs end with Tension (open question requiring judgment), not Conclusion.

**Next:** 3 implementation issues created. Implement in sequence: skill protocol change, daemon delivery, dashboard reading queue.

**Authority:** architectural — Crosses skill protocol, daemon completion, and dashboard boundaries; establishes new artifact type in the system.

---

# Investigation: Design BRIEF.md Comprehension Artifact

**Question:** What should BRIEF.md look like (template, style, length), how does it integrate with the skill protocol and completion flow, and how does the dashboard deliver it as a reading queue?

**Started:** 2026-03-24
**Updated:** 2026-03-24
**Owner:** orch-go-o7c0u
**Phase:** Complete
**Next Step:** None — design complete, 3 implementation issues created
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/plans/2026-03-05-synthesis-as-comprehension.md` | extends | Yes — plan added Thread→Insight→Position to orchestrator debriefs; BRIEF.md moves comprehension to agent output | None |
| `.kb/models/writing-style/model.md` | extends | Yes — 4 primers confirmed INERT; BRIEF.md is first structural application | None |
| `.kb/models/completion-verification/model.md` | constrains | Yes — gate 13 (explain-back) is judgment-based; BRIEF.md is evidence-based | See Finding 3 |
| `.kb/threads/2026-03-24-comprehension-artifacts-async-synthesis-delivery.md` | source | Yes — thread establishes problem, convergence, and false-comprehension risk | None |
| `.kb/investigations/2026-03-20-inv-design-writing-skill-technical-blog.md` | informs | Yes — composition self-audit pattern applies; quote-based evidence approach | None |

---

## Findings

### Finding 1: Comprehension is Conversation-Gated, Not Reading-Gated

**Evidence:** The current completion flow requires an orchestrator conversation to achieve comprehension:
1. Daemon auto-completes agents and adds `comprehension:pending` label (`pkg/daemon/coordination.go:294`)
2. `comprehension-queue-count.sh` hook injects queue count into orchestrator sessions
3. Orchestrator runs `orch complete` which triggers gate 13 (explain-back): orchestrator must explain what was built and why
4. `orch complete` removes `comprehension:pending` label (`cmd/orch/complete_lifecycle.go:201`)

Dylan cannot process completions without being in a conversation with the orchestrator. System throughput is bounded by conversation time, not Dylan's reading speed.

**Source:** `pkg/daemon/comprehension_queue.go`, `cmd/orch/complete_lifecycle.go:190-210`, `.claude/hooks/comprehension-queue-count.sh`, `pkg/verify/unverified.go:84-96`

**Significance:** This is the structural bottleneck. BRIEF.md converts comprehension from a synchronous conversation act to an asynchronous reading act. Dylan reads briefs over coffee, then conversations start from shared understanding.

---

### Finding 2: The Agent Has Maximum Context for Brief Generation

**Evidence:** At completion time, the agent has:
- Full task context (SPAWN_CONTEXT.md, codebase reads, investigation work)
- All decisions made and trade-offs accepted
- Test evidence and verification results
- The "story" of what happened — wrong turns, surprises, discoveries

No other actor (daemon, orchestrator, synthesis agent) has this full context. The daemon only sees beads metadata. The orchestrator would need to read SYNTHESIS.md + code changes to reconstruct context. A dedicated synthesis agent would need an API call to re-process.

**Source:** Thread convergence analysis, worker-base completion protocol (`skills/src/shared/worker-base/.skillc/completion.md`)

**Significance:** The completing agent is the only zero-cost producer of the brief. This eliminates infrastructure options (daemon transform, synthesis agent) and satisfies the "max plan — no API calls" constraint.

---

### Finding 3: Briefs Must Provoke, Not Summarize — The False Comprehension Risk

**Evidence:** Thread paragraph 3 identifies the risk: "good briefs create false comprehension. Dylan reads, marks as read, moves on — but never has the reactive moment where one insight triggers a strategic reframe." The highest-value output in the thread's own session emerged from Dylan reacting to synthesized material in real time — that reactive, connective thinking doesn't happen from reading a tidy summary.

The synthesis-as-comprehension plan (`2026-03-05`) faced the same challenge at orchestrator level: debriefs that summarized events vs debriefs that synthesized Thread→Insight→Position. The BRIEF.md template must encode this structurally.

**Source:** Thread update 2026-03-24 (paragraph 3), `.kb/plans/2026-03-05-synthesis-as-comprehension.md`

**Significance:** The template's final section must be **Tension** (open question requiring Dylan's judgment), not **Conclusion** (summary that feels complete). This is the structural gate against false comprehension. The brief is setup; the conversation is where the thinking happens.

---

### Finding 4: SYNTHESIS.md and BRIEF.md Serve Different Audiences

**Evidence:** Current SYNTHESIS.md sections (TLDR, Delta, Evidence, Architectural Choices, Knowledge, Next, Unexplored Questions, Friction, Session Metadata) are structured for orchestrator machine-consumption:
- TLDR is for `orch complete` verification (gate 2)
- Delta is for git diff cross-referencing (gate 9)
- Evidence is for test evidence validation (gate 8)
- Friction is for system improvement tracking

None of these serve Dylan's reading experience. They're operational, not comprehension-oriented. The "Plain-Language Summary" in the completion protocol is the closest to Dylan-facing, but it's 2-4 sentences — a blurb, not a brief.

BRIEF.md is additive. SYNTHESIS.md continues to exist for orchestrator/pipeline consumption. BRIEF.md exists for Dylan's comprehension.

**Source:** `.orch/templates/SYNTHESIS.md`, `skills/src/shared/worker-base/.skillc/completion.md:47-49`

**Significance:** No sections need to be removed from or shared with SYNTHESIS.md. Clean separation of concerns: SYNTHESIS.md → orchestrator pipeline, BRIEF.md → Dylan's reading queue.

---

### Finding 5: Existing Dashboard Infrastructure Supports a Reading Queue

**Evidence:** The dashboard already has:
- `review-queue-section` component that shows comprehension:pending items with gate status
- `completion-review` component with expand/collapse, acknowledge, batch-acknowledge
- `pending-reviews` store that tracks review state
- `/api/pending-reviews` endpoint that scans workspaces for SYNTHESIS.md

The review queue currently shows issue titles and TLDR snippets. Adding BRIEF.md content rendering is a UI extension, not new infrastructure. The acknowledge flow (mark as read → remove comprehension:pending) already exists.

**Source:** `web/src/lib/components/review-queue-section/review-queue-section.svelte`, `web/src/lib/components/completion-review/completion-review.svelte`, `cmd/orch/serve_reviews.go`

**Significance:** Dashboard delivery requires extending the existing review queue to render BRIEF.md content, not building a new reading queue from scratch. `.kb/briefs/` as the file delivery location lets the API serve brief content alongside existing review metadata.

---

## Synthesis

**Key Insights:**

1. **The brief is a reading artifact, not a verification artifact.** SYNTHESIS.md is for the pipeline. BRIEF.md is for Dylan. They have different audiences, different styles, different purposes. Mixing them would compromise both.

2. **Style injection must be stance-level, not behavioral.** The writing-style model established that 4 attention primers > 20 rules. The skill protocol should inject the primers as attention context, not as MUST/NEVER rules. The template structure does the behavioral work (sections enforce frame/resolution/tension); the primers shift what the agent notices while writing.

3. **The template's Tension section is the structural gate against false comprehension.** Without it, briefs become summaries Dylan can passively consume. The Tension section requires the agent to surface what it couldn't resolve — the judgment call, the strategic question, the surprising implication. This is what makes the brief setup for conversation, not a replacement for it.

**Answer to Investigation Question:**

BRIEF.md is a 3-section artifact (Frame / Resolution / Tension) produced by full-tier agents as a final act before Phase: Complete, written using Dylan's 4 writing primers as stance guidance. It's delivered to `.kb/briefs/{beads-id}.md` by the daemon on completion. The dashboard renders briefs in the existing review queue with mark-as-read. `orch complete` remains the sole way to clear the comprehension gate — mark-as-read confirms reading, not comprehension. Two blocking questions remain: (1) whether mark-as-read should partially decrement the comprehension threshold (`orch-go-c29fl`), and (2) how much thread context the agent needs for placement writing (`orch-go-1sqgz`).

---

## Structured Uncertainty

**What's tested:**

- Verified: SYNTHESIS.md sections are orchestrator-facing, not Dylan-facing (read template, completion protocol)
- Verified: Agent has full context at completion time (examined worker-base completion flow)
- Verified: Dashboard review queue already supports acknowledge/expand patterns (read svelte components)
- Verified: comprehension:pending lifecycle — daemon adds, orch complete removes (read Go source)

**What's untested:**

- Whether agents can produce good briefs with stance primers alone (no writing skill test run)
- Whether the Tension section actually provokes conversation vs becomes formulaic
- Whether `.kb/briefs/` as persistent storage creates accumulation problems (no cleanup lifecycle)
- Whether BRIEF.md adds meaningful completion time cost (estimated: 1-2 min of agent time)

**What would change this:**

- If agents produce formulaic briefs despite primers, may need composition self-audit (from writing skill design)
- If BRIEF.md consistently takes >5 min of agent time, may need to restrict to architect/investigation skills only
- If Dylan never reads briefs, the entire artifact is waste — need measurement within 2 weeks of launch

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| BRIEF.md template and skill protocol integration | architectural | Crosses skill protocol, daemon, and dashboard boundaries |
| Comprehension gate feedback (mark-as-read → comprehension:pending) | strategic | Changes the meaning of "comprehended" — value judgment about what constitutes understanding |
| Thread context injection into SPAWN_CONTEXT | architectural | Requires changes to spawn flow and orchestrator context management |

### Recommended Approach: Agent-Generated Brief with Stance-Primed Template

**Why this approach:**
- Zero new infrastructure — agent session is the compute, no API calls
- Agent has maximum context — no other actor can produce this quality
- Template structure (Frame/Resolution/Tension) encodes the false-comprehension guard
- Stance primers align with writing-style model's validated approach

**Trade-offs accepted:**
- Brief quality depends on agent compliance with style guidance (but template structure is the real gate, not style)
- No thread context means agents can't write precise "placement" (where this fits in Dylan's thinking) — but Frame section can approximate this from SPAWN_CONTEXT orientation
- Adding ~1-2 min to agent completion time (acceptable given the comprehension value)

**Implementation sequence:**

1. **Skill protocol change** (worker-base completion) — Add BRIEF.md as required artifact for full-tier, with template and style primers. This is the foundational change.
2. **Daemon delivery** — On completion, copy BRIEF.md from workspace to `.kb/briefs/{beads-id}.md` before archiving workspace. Add to lifecycle manager effects.
3. **Dashboard reading queue** — Extend review-queue-section to render BRIEF.md content with mark-as-read. Serve brief content from `/api/briefs/{beads-id}`.

### Alternative Approaches Considered

**Option B: Daemon-generated brief (post-completion transform)**
- **Pros:** Agents don't need skill protocol changes; centralized quality control
- **Cons:** Daemon has no task context — would need API call to read SYNTHESIS.md and produce brief; violates max-plan constraint; adds infrastructure
- **When to use instead:** If agent-generated briefs prove consistently low quality

**Option C: Dedicated synthesis agent (spawned on completion)**
- **Pros:** Can be a specialized writing agent with full style skill
- **Cons:** Requires API call (violates max plan); adds completion latency; complex orchestration
- **When to use instead:** Never — this is over-engineering for the problem

**Option D: BRIEF.md replaces SYNTHESIS.md**
- **Pros:** Simpler — one artifact instead of two
- **Cons:** SYNTHESIS.md serves the verification pipeline (gates 2, 8, 9); replacing it breaks existing infrastructure
- **When to use instead:** Never — different audiences, different purposes

**Rationale for recommendation:** Option A is the only approach that satisfies all constraints: zero new infrastructure, no API calls, agent has full context, style primers from validated model, template structure guards against false comprehension.

---

### Implementation Details

**What to implement first:**
- BRIEF.md template file at `.orch/templates/BRIEF.md`
- Skill protocol addition in `skills/src/shared/worker-base/.skillc/completion.md`
- Style guidance injection (4 primers, inline, not referenced)

**Things to watch out for:**
- Governance: `skills/src/shared/worker-base` is a governance-protected path — these changes must be done by orchestrator in direct session
- Defect class exposure: Class 3 (Stale Artifact Accumulation) — `.kb/briefs/` needs cleanup lifecycle, or briefs accumulate indefinitely
- Constraint dilution: Adding BRIEF.md instructions to completion protocol increases ceremony — keep the addition to <15 lines to stay within behavioral grammars budget

**Areas needing further investigation:**
- Thread context injection mechanism (blocked on `orch-go-1sqgz`)
- Comprehension gate feedback loop (blocked on `orch-go-c29fl`)
- Brief quality measurement — need a heuristic similar to `orch debrief --quality`

**Success criteria:**
- Full-tier agents produce BRIEF.md alongside SYNTHESIS.md
- Briefs land in `.kb/briefs/` and render in dashboard
- Dylan reports reading briefs async (over coffee, between meetings)
- Conversations start from "here's what I think about what I read" instead of narration

---

## BRIEF.md Template

See `.orch/templates/BRIEF.md` (created as part of this investigation).

### Template Design Rationale

**Three sections, not more:**

1. **Frame** (2-3 sentences) — What was the question, why did it matter? Maps to Three-Layer Reconnection's "frame." Written as story, not metadata. The reader should care about the answer before seeing it.

2. **Resolution** (1-2 paragraphs) — What did the agent find/build/decide? Maps to "resolution." Earned abstractions only — every framework term follows a concrete experience. Say what it felt like when relevant (wrong turns, surprises).

3. **Tension** (1-3 sentences) — What couldn't be resolved? What needs Dylan's judgment? Maps to "placement" but inverted: instead of the agent placing the work in Dylan's context (which it can't do), it surfaces the open question that Dylan's context would answer. This is the structural false-comprehension guard.

**Why not more sections:**
- Behavioral grammars: constraint dilution at 5+ items
- Writing-style model: 4 primers, not 20 rules
- The goal is a half-page reading artifact, not a comprehensive report

**Style guidance (injected as stance primers, not rules):**
- Story first, framework after
- Earn the abstraction
- Say what it felt like
- The turn is the piece

---

## References

**Files Examined:**
- `pkg/daemon/comprehension_queue.go` — comprehension:pending lifecycle
- `cmd/orch/complete_lifecycle.go:190-210` — orch complete removes comprehension:pending
- `pkg/verify/unverified.go:84-96` — tier-based verification (gate1=comprehension, gate2=behavioral)
- `.claude/hooks/comprehension-queue-count.sh` — hook injecting queue count
- `skills/src/shared/worker-base/.skillc/completion.md` — current completion protocol
- `.orch/templates/SYNTHESIS.md` — current synthesis template
- `.kb/models/writing-style/model.md` — 4 stance primers, INERT status
- `.kb/models/completion-verification/model.md` — 14 gates, V0-V3 levels
- `.kb/plans/2026-03-05-synthesis-as-comprehension.md` — prior comprehension infrastructure
- `web/src/lib/components/review-queue-section/review-queue-section.svelte` — dashboard review queue
- `web/src/lib/components/completion-review/completion-review.svelte` — dashboard completion review
- `cmd/orch/serve_reviews.go` — pending reviews API
- `.kb/global/principles.md` — Gate Over Remind, Asymmetric Velocity, Session Amnesia

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-24-comprehension-artifacts-async-synthesis-delivery.md`
- **Blocking Question:** `orch-go-c29fl` — mark-as-read vs orch complete for comprehension gate
- **Blocking Question:** `orch-go-1sqgz` — thread context injection for placement writing
- **Investigation:** `.kb/investigations/2026-03-20-inv-design-writing-skill-technical-blog.md` — composition self-audit pattern
