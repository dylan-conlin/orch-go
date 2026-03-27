## Summary (D.E.K.N.)

**Delta:** Designed the composition layer — the missing step between individual brief generation and thread-level understanding. Composition is an orchestrator-session act (not daemon automation), triggered by accumulation threshold at session start, producing digest artifacts with draft thread proposals that Dylan reviews conversationally.

**Evidence:** 61 briefs examined, 4 cross-cutting clusters manually identified in prior session (identity gap × 8, epistemic dishonesty × 5, model-routing × 6, production-exceeding-comprehension × 3). Signal-to-design loop model provides the stage framework. Comprehension queue infrastructure already supports the trigger mechanism.

**Knowledge:** Composition is the clustering stage of the signal-to-design loop applied to briefs. The key design constraint is that composition must provoke, not replace, the synthesis stage — which requires Dylan's engagement per the Understanding Through Engagement principle. The output is a "digest" that surfaces clusters and harvests tension sections, not a summary that feels complete.

**Next:** 3 implementation phases. Phase 1: `orch compose` CLI + digest artifact format. Phase 2: session-start hook integration. Phase 3: tension harvesting and thread proposal flow.

**Authority:** architectural — Crosses comprehension queue, thread system, orient session-start, and orchestrator workflow boundaries. Multiple valid approaches with significant trade-offs.

---

# Investigation: Design Brief Composition Layer

**Question:** What should sit between individual brief generation and thread-level understanding? Who composes, when, what's the output, and how does it avoid becoming a closed loop?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-c5ha1 (architect)
**Phase:** Complete
**Next Step:** Implementation via decomposed issues
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2026-03-24-inv-design-brief-md-comprehension-artifact.md` | extends | Yes — BRIEF.md template is the atom this composes | None |
| `2026-03-24-inv-design-briefs-reading-queue-persistent.md` | extends | Yes — `/briefs` page is individual reading; composition is cross-brief | None |
| `2026-03-26-design-thread-first-home-surface.md` | extends | Yes — composition output (digests) feed the thread-first surface | None |
| `.kb/global/models/signal-to-design-loop.md` | constrains | Yes — composition is Stage 3 (clustering) of this loop | None |
| `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` | constrains | Yes — composition is core product, not infrastructure | None |

---

## Problem Framing

**The gap:** Agent briefs are atoms of understanding. Each one captures what a single agent learned. But the most valuable insights — cross-cutting patterns that only become visible when reading briefs together — live in the relationships BETWEEN briefs. Today's evidence: 8 briefs independently discovered the identity gap, 5 share the epistemic dishonesty pattern, 6 touch model-routing-as-boundary. No individual brief names these patterns. The system produces the atoms but has no mechanism to compose them.

**Success criteria:**
1. Cross-cutting patterns across briefs become visible without Dylan reading every brief individually
2. Tension sections (orphaned knowledge seeds) are structurally harvested, not archived
3. Composition provokes conversation, not replaces it — per comprehension artifacts thread
4. The closed-loop risk (system coherent with itself, disconnected from reality) is explicitly managed
5. The composition integrates with the existing comprehension queue lifecycle

**Constraints:**
- **Understanding Through Engagement** (principle): Synthesis requires the vantage point only the integrating level has. Composition cannot be fully automated synthesis.
- **Asymmetric Velocity** (principle): Briefs can accumulate faster than Dylan can read. Composition is a gate in this dimension.
- **Provenance** (principle): Composition must trace to the briefs it clusters, not produce free-standing claims.
- **Comprehension artifacts thread** (2026-03-24): Briefs must provoke conversation, not replace it. Composition that feels complete kills the reactive moment.
- **Epistemic dishonesty thread** (2026-03-26): Must not conflate "I clustered these" with "I understood these" — would be 6th instance of the system treating absence of verification as presence of understanding.

**Scope:**
- IN: Design of composition step, trigger mechanism, output artifact, queue interaction, closed-loop management
- OUT: Implementation (separate issues), dashboard UI changes, thread system modifications

---

## Findings

### Finding 1: Composition Is the Clustering Stage of the Signal-to-Design Loop

**Evidence:** The signal-to-design loop model (`.kb/global/models/signal-to-design-loop.md`) describes five stages: Signal Capture → Accumulation → Clustering → Synthesis → Design Response. Briefs are Stage 1 (signal capture) and Stage 2 (accumulation in `.kb/briefs/`). What's missing is Stage 3 — clustering related signals by shared metadata.

The model's clustering stage specifies: "Requires explicit, machine-readable clustering key. Resolution hierarchy: explicit tag > threshold count > lexical proximity." Briefs don't have explicit tags or structured metadata beyond Frame/Resolution/Tension. Clustering must work via content analysis (lexical proximity + semantic overlap), which is the weakest clustering resolution per the model.

However, briefs have a structural advantage: the Tension section is a constrained vocabulary of open questions. Tensions cluster better than Frames because they're about what's unresolved, and unresolved questions often point at the same gap from different angles.

**Significance:** This grounds the design in an existing, validated model. Composition is not a new pattern — it's the missing stage of a loop that's already partially operating.

**Defect class exposure:** Class 5 (Contradictory Authority Signals) — if composition and individual brief reading both claim to represent "what the system learned," they'll diverge. Mitigation: composition explicitly labels itself as "draft" requiring Dylan's engagement.

---

### Finding 2: Who Composes — Orchestrator Session Is the Only Viable Actor

**Evidence:** Three actors were considered:

**Option A: Daemon second pass (automated).** The daemon could run a composition pass after brief generation. Pro: no human gating. Con: Violates Understanding Through Engagement — synthesis requires cross-agent vantage. The daemon has no context about Dylan's current thinking, active threads, or what he cares about. It would cluster by lexical similarity alone. The closed-loop risk thread (2026-03-10, resolved) identified this exact failure mode: automated composition without external validation produces a system coherent with itself but disconnected from reality.

**Option B: Dedicated composition agent (spawned).** Spawn a composition agent when N briefs accumulate. Pro: agent has full context window for comparison. Con: Still an automated actor — it has no access to Dylan's mental model. The Understanding Through Engagement principle explicitly rejects "spawning architects to think for me." More importantly, the comprehension artifacts thread says the reactive moment where insight happens requires Dylan's participation. A composition agent that produces finished clusters removes that moment.

**Option C: Orchestrator session (conversation-gated).** Composition happens during the orchestrator's session-start orientation, when the orchestrator reads accumulated briefs and surfaces patterns conversationally to Dylan. Pro: The orchestrator has cross-agent context AND Dylan is present for the reactive moment. This is exactly how the 4 clusters were discovered today — in a conversation. Con: Gated on session start, so composition doesn't happen between sessions.

**Fork resolution:** Option C. The con (session-gated) is actually a feature — composition without Dylan present is the closed-loop risk. The session gate IS the human-in-the-loop constraint.

**Substrate trace:** Understanding Through Engagement ("synthesis requires direct engagement"), Comprehension artifacts thread ("provoke, don't replace"), Closed-loop risk thread ("automated composition without external validation").

---

### Finding 3: When to Compose — Accumulation Threshold at Session Start

**Evidence:** Three triggers were considered:

**Trigger A: On every brief.** Daemon recomposes after each new brief. Pro: always current. Con: Thrashing — clusters change every hour. Also, composition of 2 briefs isn't meaningful. The signal-to-design loop model says clustering needs "RecurrenceThreshold = 3" — you need enough signals for a pattern to emerge.

**Trigger B: Schedule (e.g., daily at 6am).** Pro: predictable, aligns with morning coffee reading. Con: Arbitrary timing — may fire with 0 new briefs or miss a burst of 15.

**Trigger C: Accumulation threshold at session start.** When the orchestrator session begins and N+ unprocessed briefs exist, composition runs as part of orientation. Pro: natural integration point (orient already surfaces threads, models, ready work); only fires when there's material to compose; Dylan is present for the reactive moment. Con: Composition quality depends on orchestrator context window.

**Fork resolution:** Trigger C, with N=5 (matching the existing comprehension queue threshold). The orient command already collects active threads, model freshness, and ready issues. Adding a composition step when briefs exceed threshold is a natural extension.

**Implementation shape:** `orch compose` as a standalone CLI command (callable by orient or orchestrator). Reads all briefs from `.kb/briefs/`, clusters by content analysis, produces a digest. Orient calls this when brief count exceeds threshold and injects the digest into session-start context.

---

### Finding 4: Output Artifact — The Digest

**Evidence:** Three output formats were considered:

**Format A: Draft thread entries.** Composition directly writes to `.kb/threads/`. Pro: threads are the primary artifact per the product decision. Con: Writing to threads without Dylan's review is the closed-loop risk — the system would be editing its own understanding artifacts. Also violates the epistemic dishonesty constraint: clustered ≠ understood.

**Format B: Cluster summaries.** Markdown file listing clusters with member briefs and a generated summary. Pro: simple, readable. Con: Summaries create false comprehension per the comprehension artifacts thread. A summary that feels complete kills the reactive moment.

**Format C: Digest with draft proposals.** A new artifact type — the "digest" — that lives in `.kb/digests/`. Contains:
1. **Clusters** — groups of related briefs with the clustering rationale (not a summary)
2. **Tension harvest** — collected Tension sections from member briefs, grouped by cluster
3. **Thread proposals** — "these briefs seem related to thread X" or "these briefs may warrant a new thread about Y" — proposals, not actions
4. **Epistemic label** — explicit statement of what the digest IS (clustering by observed content similarity) and what it IS NOT (understanding, synthesis, verified pattern)

Pro: Provokes conversation ("do you see these clusters differently?"), harvests orphaned tensions structurally, proposes but doesn't execute thread updates. Con: New artifact type adds complexity.

**Fork resolution:** Format C. The epistemic label is the structural guard against the 6th instance of the system treating "I processed this" as "I understood this." The thread proposals are conversation starters, not actions.

**Artifact location:** `.kb/digests/YYYY-MM-DD-digest.md` — one per day maximum. Digests are ephemeral by design — they're input to a conversation, not durable knowledge. Old digests can be cleaned up aggressively (7-day retention).

---

### Finding 5: Comprehension Queue Interaction — Composition Does Not Drain the Queue

**Evidence:** The comprehension queue has two states: `comprehension:unread` (daemon completed, orchestrator hasn't reviewed) and `comprehension:processed` (orchestrator reviewed, Dylan hasn't read brief). The queue throttles spawning when unread count exceeds 5.

Two interaction models were considered:

**Model A: Composition drains the queue.** After composing briefs into a digest, mark constituent briefs as processed. Pro: clears the backlog, unblocks spawning. Con: Conflates "clustered" with "comprehended." This is exactly the epistemic dishonesty the system already has in 5 other places. A brief that's been clustered hasn't been understood — the clustering is a navigation aid, not a comprehension act.

**Model B: Composition adds a layer above the queue.** Briefs retain their comprehension state. The digest is a lens ON the queue, not a replacement for it. Dylan can still read individual briefs. The digest helps him decide which ones to engage with first. Pro: honest about what composition achieved. Con: Doesn't unblock spawning — the queue stays full.

**Fork resolution:** Model B. The queue throttle is doing its job — it's an Asymmetric Velocity gate. If the system is producing faster than Dylan can comprehend, composition should help him triage, not pretend he's caught up. The throttle pressure is actually valuable: it creates back-pressure on spawning, which is the system's way of saying "slow down, the human can't keep up."

However, the trigger should consider `comprehension:processed` briefs (which haven't been composed into a digest yet) as well as unread ones. Composition operates on briefs Dylan hasn't ENGAGED with, regardless of whether the orchestrator has mechanically processed them.

---

### Finding 6: Closed-Loop Risk Management — Epistemic Labeling

**Evidence:** The closed-loop risk thread (resolved 2026-03-17) established that automated processing without external validation produces systems coherent with themselves but disconnected from reality. The epistemic dishonesty thread (2026-03-26) found 5 instances of the system conflating "didn't check" with "nothing there."

Composition has the same risk: an automated system reads briefs and declares "these are related" — but the relationship may be coincidental, or the real pattern may be something the system can't see because it lacks Dylan's context.

**Mitigation design (three layers):**

1. **Epistemic label on every digest.** Each digest opens with:
   ```
   ## Epistemic Status
   This digest clusters briefs by observed content similarity.
   It has NOT been verified by a human. Clusters may be:
   - Coincidental (briefs happen to use similar words)
   - Incomplete (the real pattern includes briefs not in this cluster)
   - Wrong (the briefs are related, but not for the reason shown)
   ```

2. **Draft thread proposals, not thread edits.** The digest proposes "brief X seems related to thread Y" but does NOT modify threads. Dylan or the orchestrator must explicitly accept the proposal. This preserves the human judgment step.

3. **Provenance links.** Every cluster cites the specific brief IDs and the specific content that triggered the clustering. Dylan can verify the reasoning, not just the conclusion. This satisfies the Provenance principle.

**What this sacrifices:** Speed. An automated system could update threads directly, and Dylan would arrive to a fully organized knowledge graph. But that speed comes at the cost of Dylan's participation in the sense-making. The comprehension artifacts thread is clear: the reactive moment IS the product. Composition should set up that moment, not skip it.

---

## Synthesis

**Key Insights:**

1. **Composition is Stage 3 (clustering) of the signal-to-design loop, applied to briefs.** The loop already operates for defect classes and investigations. Briefs are the newest signal type. The same stage structure applies.

2. **The actor question resolved cleanly: orchestrator session, not daemon.** Understanding Through Engagement makes this nearly tautological — synthesis requires the integrating level's vantage point and direct human engagement. The daemon can prepare the digest; the orchestrator presents it; Dylan engages with it.

3. **The output must provoke, not summarize.** This is the hardest constraint to hold. Every instinct says "make it easy — produce a finished summary." But the comprehension artifacts thread is clear: false comprehension is worse than no comprehension. The digest presents clusters and proposals. The understanding happens in conversation.

4. **Tension harvesting is the highest-leverage innovation.** Every brief ends with an open question. These are forming questions for next-round work. Right now they're archived with the brief. Composition should structurally collect them, group them by cluster, and surface them as thread-entry candidates. This is the mechanism by which "every spawn composes knowledge" becomes true — the tension sections feed forward into the thread graph.

5. **The queue interaction was the most dangerous fork.** The temptation to have composition drain the queue is strong — it would unblock spawning and feel productive. But it would be the 6th instance of epistemic dishonesty. Composition is a navigation aid, not a comprehension act. The queue stays honest.

**Answer to Investigation Question:**

Composition is an orchestrator-session act that clusters accumulated briefs, harvests their tension sections, and produces digest artifacts with draft thread proposals — all presented conversationally to Dylan, who engages with the clusters and decides what becomes thread-level understanding. It fires at session start when 5+ unprocessed briefs exist. It does not modify threads or drain the comprehension queue. It labels its own epistemic status explicitly to avoid being the 6th instance of the system conflating processing with understanding.

---

## Structured Uncertainty

**What's tested:**
- Verified: comprehension queue has two-state lifecycle with threshold at 5 (`pkg/daemon/comprehension_queue.go`)
- Verified: orient command already collects threads, models, ready work at session start (`cmd/orch/orient_cmd.go`)
- Verified: 61 briefs exist in `.kb/briefs/`, averaging ~15 lines each (943 total lines)
- Verified: thread system supports Create, Append, LinkWork operations (`pkg/thread/`)
- Verified: signal-to-design loop model describes clustering as Stage 3 with explicit requirements

**What's untested:**
- Whether content-based clustering produces meaningful clusters without explicit metadata tags
- Whether digest artifacts actually provoke conversation or get skipped like other system output
- Whether 5 is the right threshold — may be too low (noisy) or too high (delayed)
- Whether daily digest retention (7 days) is right, or whether digests should be even more ephemeral

**What would change this:**
- If content-based clustering fails consistently, may need explicit clustering metadata on briefs (a tag or category field)
- If Dylan never reads digests, the artifact is waste — same risk as with briefs themselves
- If the orchestrator can't produce good digests within session-start context, may need a pre-session composition agent after all (with explicit Dylan review gate)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Digest artifact format and location | architectural | New artifact type in the knowledge system |
| `orch compose` CLI command | implementation | Extends existing CLI patterns |
| Orient integration (session-start hook) | architectural | Changes session-start behavior |
| Thread proposal flow | strategic | Changes how threads get updated — value judgment about automation boundary |
| Comprehension queue interaction | architectural | Affects queue semantics and spawning throttle |

### Recommended Approach: Three-Phase Implementation

**Phase 1: `orch compose` CLI + digest format** (implementation)

Create `orch compose` command that:
1. Scans `.kb/briefs/` for all briefs
2. Reads each brief's Frame, Resolution, and Tension sections
3. Clusters by content similarity (start with simple keyword overlap; upgrade to semantic later if needed)
4. Collects Tension sections grouped by cluster
5. Identifies potential thread connections (match cluster content against `.kb/threads/` titles and entries)
6. Writes digest to `.kb/digests/YYYY-MM-DD-digest.md`

**Digest format:**
```markdown
---
date: 2026-03-26
briefs_composed: 33
clusters_found: 4
epistemic_status: unverified-clustering
---

## Epistemic Status
This digest clusters briefs by observed content similarity.
It has NOT been verified by a human. [standard label]

## Cluster 1: [Generated cluster name]
**Briefs:** orch-go-f8y50, orch-go-1r7ih, orch-go-z4h7s, ...
**Why clustered:** [specific content overlap that triggered grouping]
**Thread connection:** May relate to thread "threads-as-primary-artifact-thinking" (forming)

### What the cluster shows (not a summary — a question)
[The composition insight: what becomes visible only when seeing these together]

### Harvested tensions
- (orch-go-f8y50) [tension text]
- (orch-go-1r7ih) [tension text]

### Draft thread proposal
**Proposal:** Append to thread "threads-as-primary-artifact-thinking":
> [Draft entry text]

**Or new thread:** "Identity gap — every surface still introduces the system as something it isn't"

---

[Repeat for each cluster]

## Unclustered Briefs
[Briefs that didn't fit any cluster — listed for individual reading]

## Tension Orphans
[Tensions from unclustered briefs — open questions with no structural home]
```

**Phase 2: Orient integration** (architectural)

Add to `orch orient` / session-start hook:
1. Count unprocessed briefs (unread + processed-but-undigested)
2. If count >= 5, run `orch compose` and include digest summary in orientation
3. Orient output adds: "Digest available: 4 clusters across 33 briefs. Key finding: [cluster 1 name]. Read full digest or discuss?"

**Phase 3: Tension harvesting + thread proposal flow** (strategic)

Build the thread proposal acceptance flow:
1. `orch compose --accept cluster-1` — orchestrator/Dylan approves a thread proposal from a digest
2. Accepted proposals execute: `thread append` or `thread new` with the draft content
3. Accepted briefs get a `composed:digest-YYYY-MM-DD` label (tracking, not comprehension state change)
4. Tension orphans surface in orient as "N open questions without thread homes"

### Alternative Approaches Considered

**Option B: Daemon-automated composition**
- **Pros:** Always current, no session dependency
- **Cons:** Violates Understanding Through Engagement, creates closed-loop risk, can't access Dylan's mental model
- **When to use instead:** Never for composition. The daemon can trigger `orch compose` on schedule for pre-computation, but the digest must still be presented conversationally.

**Option C: LLM-powered semantic clustering**
- **Pros:** Better clustering quality than keyword overlap
- **Cons:** Requires API call (cost), adds latency to session start, the LLM may hallucinate connections
- **When to use instead:** Phase 2 upgrade if keyword clustering proves too noisy. Important: the LLM would do clustering, not synthesis — the distinction matters.

**Rationale for recommendation:** The three-phase approach starts with the minimum viable composition (keyword clustering + digest format), integrates with the existing session-start flow, and defers the strategic question (thread proposal acceptance) to Phase 3 where it can be informed by actual usage of Phases 1-2.

### Implementation Details

**What to implement first:**
- `orch compose` command in `cmd/orch/compose_cmd.go`
- Digest template in `.orch/templates/DIGEST.md`
- Brief parser (extract Frame/Resolution/Tension from markdown) in `pkg/comprehension/` or `pkg/brief/`

**Things to watch out for:**
- Defect class exposure: Class 3 (Stale Artifact Accumulation) — digests need cleanup lifecycle (7-day retention, `orch clean` integration)
- Defect class exposure: Class 5 (Contradictory Authority Signals) — digest clusters and individual brief reading may tell different stories. The epistemic label mitigates this.
- Defect class exposure: Class 0 (Scope Expansion) — keyword clustering may over-cluster (everything touches "comprehension") or under-cluster. Start conservative (require 3+ keyword overlap).

**Areas needing further investigation:**
- Keyword extraction strategy — stopword removal + TF-IDF-like weighting, or simpler?
- Thread matching algorithm — exact title match, or fuzzy content match against thread entries?
- Digest UI presentation — dashboard route, or orient-only?

**Success criteria:**
- `orch compose` produces a digest from 10+ briefs with at least 2 meaningful clusters
- Digest clusters match what Dylan would identify reading the same briefs
- Session-start orientation surfaces digest when threshold is met
- Dylan reports the digest helped him decide what to engage with (not that it replaced engagement)

---

## Composition Claims

| ID | Claim | Components Involved | How to Verify |
|----|-------|--------------------|----|
| CC-1 | "Composition never modifies threads directly" | compose_cmd + thread proposals | Review compose_cmd: no `thread.Append()` calls; proposals are text only |
| CC-2 | "Queue state is unchanged by composition" | compose_cmd + comprehension_queue | Verify: no calls to `TransitionToProcessed` or `RemoveComprehensionUnread` in compose path |
| CC-3 | "Every digest includes epistemic status label" | compose_cmd + digest template | Verify: template includes label; compose_cmd does not strip it |
| CC-4 | "Digest clusters trace to specific brief IDs" | compose_cmd + digest format | Verify: every cluster lists member brief IDs with content rationale |

---

## References

**Files Examined:**
- `pkg/daemon/comprehension_queue.go` — two-state comprehension lifecycle, threshold mechanism
- `cmd/orch/orient_cmd.go` — session-start orientation, data collection pattern
- `cmd/orch/complete_brief.go` — brief generation from SYNTHESIS.md
- `pkg/daemon/coordination.go` — daemon completion routing, headless completion firing
- `cmd/orch/serve_briefs.go` — brief API endpoints, read state tracking
- `pkg/thread/thread.go` — thread data structures, lifecycle operations
- `.kb/global/models/signal-to-design-loop.md` — five-stage loop model
- `.kb/global/principles.md` — Understanding Through Engagement, Asymmetric Velocity, Provenance

**Briefs Examined (representative sample):**
- orch-go-pityd — boundary classification (identity gap cluster)
- orch-go-wgkj4 — codebase inventory (identity gap cluster)
- orch-go-ey4py — daemon classification (identity gap cluster)
- orch-go-bw0y6 — architecture guide rewrite (identity gap cluster)
- orch-go-vo51p — dashboard classification (identity gap cluster)
- orch-go-z4h7s — README rewrite (identity gap cluster)
- orch-go-o5uih — stall tracker (epistemic dishonesty cluster)
- orch-go-k6c0v — kb timeout (epistemic dishonesty cluster)
- orch-go-fsikn — verification spec (epistemic dishonesty cluster)
- orch-go-n4uwb — synthesis compliance (epistemic dishonesty cluster)

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-26-every-spawn-composes-knowledge-task.md`
- **Thread:** `.kb/threads/2026-03-26-epistemic-dishonesty-system-conflates-didn.md`
- **Thread:** `.kb/threads/2026-03-24-artifact-attractors-agents-naturally-externalize.md`
- **Decision:** `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md`
