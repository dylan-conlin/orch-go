## Summary (D.E.K.N.)

**Delta:** No single tool dominates all memory types; Claude Code leads explicit persistent instruction memory, while GitHub Copilot Memory currently leads validated episodic memory for coding agents.

**Evidence:** I tested vendor docs and changelogs directly (Claude, Cursor, Windsurf, Codex, Devin, Gemini, Copilot) and mapped each system against declarative/procedural/episodic memory behaviors.

**Knowledge:** Orch is already strong in declarative and procedural memory via models/investigations/decisions/templates, but remains comparatively weak in automated episodic memory capture and replay.

**Next:** Close this research issue and use findings to scope an architectural follow-up for first-class episodic memory in orch.

**Authority:** architectural - recommendations span multiple orch subsystems (spawn, verification, session state, and artifact lifecycle).

---

# Investigation: Session Memory Landscape 2026 Who

**Question:** Which AI coding tools currently provide the best session memory, by memory type (declarative/procedural/episodic), and how that compares to orch's artifact stack.

**Started:** 2026-02-08
**Updated:** 2026-02-08
**Owner:** OpenCode worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
| ------------- | ------------ | -------- | --------- |
| N/A           | -            | -        | -         |

---

## Findings

### Finding 1: Claude Code has the strongest explicit instruction stack among coding CLIs

**Evidence:** Claude Code documents layered memory types (`CLAUDE.md`, `.claude/rules`, user/project/local scopes, managed org policy) plus auto memory persisted under `~/.claude/projects/<project>/memory/` with startup loading of `MEMORY.md` head and on-demand topic files.

**Source:** `https://docs.anthropic.com/en/docs/claude-code/memory`.

**Significance:** This is high-quality declarative + procedural memory with clear precedence and operational ergonomics; episodic memory exists but is mostly note-centric rather than action-log-centric.

---

### Finding 2: Cursor and Windsurf converge on rules-first memory, with partial episodic capabilities

**Evidence:** Cursor docs state "large language models don't retain memory between completions" and position `.cursor/rules`, `AGENTS.md`, and user/team rules as persistent prompt-level memory; Cursor also supports session resume, cloud handoff, and enterprise shared transcripts/insights. Windsurf explicitly separates auto-generated workspace memories from global/workspace/system rules.

**Source:** `https://cursor.com/docs/context/rules`; `https://cursor.com/docs/cli/overview`; `https://cursor.com/changelog/enterprise-dec-2025`; `https://docs.windsurf.com/windsurf/cascade/memories`.

**Significance:** Both are strong for declarative/procedural memory. Episodic is present through transcripts and summaries, but less formalized than a validated reusable memory object model.

---

### Finding 3: Codex has robust transcript continuity and instruction layering, with episodic memory emerging

**Evidence:** Codex provides resumable CLI sessions, archived threads, cloud task history, AGENTS.md layering, and app/IDE/CLI shared config. Changelog notes show recent "memory plumbing" for thread memory summaries, indicating active investment in episodic memory beyond raw transcript persistence.

**Source:** `https://developers.openai.com/codex/cli/features`; `https://developers.openai.com/codex/guides/agents-md`; `https://developers.openai.com/codex/app/settings`; `https://developers.openai.com/codex/changelog`.

**Significance:** Codex is strong and improving across all three memory categories, but episodic memory reuse semantics are still less explicit in public docs than Copilot Memory.

---

### Finding 4: Devin has strong organizational memory and explicit session retrospection

**Evidence:** Devin exposes reusable Knowledge (triggered retrieval, repo pinning, auto-generated knowledge), Playbooks for procedural templates, and Session Insights that analyze timelines, issues, and prompt improvements after runs.

**Source:** `https://docs.devin.ai/product-guides/knowledge`; `https://docs.devin.ai/onboard-devin/knowledge-onboarding`; `https://docs.devin.ai/product-guides/creating-playbooks`; `https://docs.devin.ai/product-guides/session-insights`.

**Significance:** Devin is very strong on declarative/procedural memory and offers better explicit episodic analysis than most peers, though it emphasizes retrospective coaching more than low-latency episodic retrieval during every turn.

---

### Finding 5: GitHub Copilot Memory currently sets the strongest public bar for episodic memory correctness

**Evidence:** Copilot Memory stores repository-scoped memories with citations, validates memories against current code before use, and applies TTL deletion (28 days) to reduce stale memory risk; memories are also reviewable/deletable by repo owners.

**Source:** `https://docs.github.com/en/copilot/concepts/agents/copilot-memory`; `https://docs.github.com/en/copilot/how-tos/use-copilot-agents/copilot-memory`.

**Significance:** This is the most explicit "track actions/observations, validate before reuse, expire stale state" pattern among surveyed tools and directly addresses the weak point called out in this task.

---

### Finding 6: Gemini Code Assist has strong repository retrieval memory, weak explicit episodic memory

**Evidence:** Gemini Code Assist supports local codebase awareness and enterprise code customization (indexed private repos, 24h reindexing, seat-managed retrieval), but docs do not describe persistent action-history memory objects equivalent to Copilot Memory.

**Source:** `https://docs.cloud.google.com/gemini/docs/codeassist/overview`; `https://docs.cloud.google.com/gemini/docs/codeassist/configure-local-codebase-awareness`; `https://docs.cloud.google.com/gemini/docs/codeassist/code-customization-overview`.

**Significance:** Gemini is strong for contextual retrieval (declarative memory substrate) but currently less mature in explicit episodic session memory.

---

### Finding 7: Orch has excellent explicit artifact memory, but episodic memory remains mostly manual

**Evidence:** Orch's stack includes models (`.kb/models/`), investigations (`.kb/investigations/`), decisions (`.kb/decisions/`), quick entries (`.kb/quick/entries.jsonl`), spawn context injection (`SPAWN_CONTEXT.md`), and handoff/synthesis templates with D.E.K.N.; this is explicitly represented in spawned context and local KB structure.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-research-session-memory-landscape-08feb-9f82/SPAWN_CONTEXT.md`; `/Users/dylanconlin/Documents/personal/orch-go/.kb/models/README.md`; `/Users/dylanconlin/Documents/personal/orch-go/.kb/models/context-injection.md`; `/Users/dylanconlin/Documents/personal/orch-go/.kb/quick/entries.jsonl`; `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md`; `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SESSION_HANDOFF.md`.

**Significance:** Orch is already differentiated on human-readable durable knowledge artifacts, but episodic state still depends heavily on manual synthesis quality and late-session recall instead of always-on validated action-memory primitives.

---

## Synthesis

**Key Insights:**

1. **"Best" depends on memory type** - Claude/Cursor/Windsurf win on explicit instruction memory ergonomics, Copilot currently wins on validated episodic memory semantics, and Devin wins on post-session reflective analysis.

2. **Episodic maturity requires three mechanics together** - citation/provenance, validation-before-reuse, and lifecycle expiry; most tools provide only 1-2 of these.

3. **Orch already has high-quality substrate for declarative/procedural memory** - the gap is not representation depth but automated capture/retrieval of what happened during execution.

**Answer to Investigation Question:**

There is no universal single winner across memory categories. If the question is "best overall session memory for coding workflows," the leading pattern today is split: Claude Code for layered persistent instruction memory, and GitHub Copilot Memory for explicit episodic memory safety (cited, validated, TTL). Compared to that landscape, orch is ahead on durable knowledge architecture but behind on low-friction episodic memory capture/reuse, aligning with the principle concern that actions are less systematically tracked than state.

---

## Structured Uncertainty

**What's tested:**

- ✅ Claude, Cursor, Windsurf, Codex, Devin, Gemini, and Copilot memory behavior claims are based on direct doc/changelog fetches executed in this session.
- ✅ Orch artifact-stack comparison is based on direct local file reads plus spawned-context constraints.
- ✅ Episodic-memory gap claim is validated by absence of first-class auto-validated episodic object model in orch docs/templates and reliance on manual D.E.K.N./handoff narratives.

**What's untested:**

- ⚠️ Production-scale effectiveness differences between vendors were not benchmarked with identical tasks.
- ⚠️ Closed-beta/enterprise-only memory features may exist beyond public docs.
- ⚠️ User satisfaction with each memory system was not measured.

**What would change this:**

- If orch already has a hidden validated episodic memory subsystem (event-to-memory pipeline with retrieval and TTL), this conclusion changes.
- If vendor docs are stale and product behavior diverges materially in current releases, rankings would need refresh.
- If Copilot Memory validation/TTL behavior proves unreliable in practice, its episodic lead would weaken.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation                                                         | Authority      | Rationale                                                                          |
| ---------------------------------------------------------------------- | -------------- | ---------------------------------------------------------------------------------- |
| Add first-class episodic memory pipeline to orch                       | architectural  | Cross-cuts spawn runtime, event logging, verification, retrieval, and KB surfacing |
| Keep D.E.K.N. artifacts but backfill them from machine-captured events | implementation | Can be phased by improving synthesis generation and template prefill               |
| Add memory lifecycle controls (validation + TTL + owner curation)      | architectural  | Requires new retention, indexing, and UI/CLI governance paths                      |

### Recommended Approach ⭐

**Evidence-backed episodic memory layer** - Introduce an explicit `action-memory` artifact class generated from runtime events, with citations, validation-before-reuse, and expiry.

**Why this approach:**

- Directly addresses the "Track Actions Not Just State" weakness.
- Reuses existing orch strengths (D.E.K.N., models, decisions, context injection).
- Mirrors proven external pattern (Copilot cited + validated + TTL memory).

**Trade-offs accepted:**

- Higher implementation complexity in exchange for lower manual cognitive burden.
- Additional storage/indexing overhead to gain better reliability and cross-session continuity.

**Implementation sequence:**

1. Define `action-memory` schema (action, outcome, evidence pointer, confidence, expiry).
2. Add event->memory extraction at key lifecycle boundaries (spawn, command, verification, completion).
3. Gate reuse through revalidation against current workspace/code/session state before injection into prompts.

### Alternative Approaches Considered

**Option B: Improve templates only**

- **Pros:** Fast to ship.
- **Cons:** Still manual; does not solve episodic capture/reuse quality.
- **When to use instead:** If near-term goal is only better handoff readability.

**Option C: Keep status quo + enforce stricter manual reporting**

- **Pros:** Minimal engineering effort.
- **Cons:** Increases process overhead and still fails under session amnesia pressure.
- **When to use instead:** Short-term while architecture work is queued.

**Rationale for recommendation:** Option A is the only path that materially closes the episodic memory gap while preserving orch's existing artifact philosophy.

---

### Implementation Details

**What to implement first:**

- Memory schema + storage location conventions.
- Extraction hooks from existing event stream.
- Retrieval API for prompt injection with validation checks.

**Things to watch out for:**

- ⚠️ Avoid turning noisy low-value events into memory spam.
- ⚠️ Ensure memory expiry and revalidation are deterministic and observable.
- ⚠️ Keep prompt injection budgets bounded (do not regress context bloat constraints).

**Areas needing further investigation:**

- Retention policy by memory type (short-lived vs long-lived episodic entries).
- Best retrieval ranking signal (recency vs relevance vs confidence).
- UX for curator controls (approve/delete/edit memories).

**Success criteria:**

- ✅ Workers receive relevant action-history context automatically without manual restating.
- ✅ Completion/handoff quality becomes less sensitive to end-of-session narrative quality.
- ✅ Stale episodic memories are auto-pruned or fail validation before use.

---

## References

**Files Examined:**

- `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-research-session-memory-landscape-08feb-9f82/SPAWN_CONTEXT.md` - Task requirements, constraints, and orch memory stack framing.
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/models/README.md` - Model role in orch knowledge architecture.
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/models/context-injection.md` - Injection architecture and context/role mechanics.
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/quick/entries.jsonl` - Quick-entry memory patterns and D.E.K.N. metadata.
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md` - Worker synthesis artifact shape.
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SESSION_HANDOFF.md` - Cross-session handoff artifact shape.

**Commands Run:**

```bash
# Verify task project directory
pwd

# Phase reporting
orch phase orch-go-21507 Planning "Researching AI coding tool session memory landscape and comparing to orch"
orch phase orch-go-21507 Implementing "Collecting external docs and drafting comparative memory landscape analysis"

# Create investigation and report path
kb create investigation session-memory-landscape-2026-who
bd comments add orch-go-21507 "investigation_path: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-08-inv-session-memory-landscape-2026-who.md"
```

**External Documentation:**

- `https://docs.anthropic.com/en/docs/claude-code/memory` - Claude memory hierarchy and auto-memory behavior.
- `https://cursor.com/docs/context/rules` - Cursor rules model and `.cursorrules` deprecation.
- `https://cursor.com/docs/cli/overview` - Cursor session resume and CLI flow.
- `https://cursor.com/changelog/enterprise-dec-2025` - Cursor shared transcripts/insights.
- `https://docs.windsurf.com/windsurf/cascade/memories` - Windsurf memories/rules architecture.
- `https://developers.openai.com/codex/cli/features` - Codex resume and transcript continuity.
- `https://developers.openai.com/codex/guides/agents-md` - Codex layered instruction discovery.
- `https://developers.openai.com/codex/changelog` - Codex memory plumbing trajectory.
- `https://docs.devin.ai/product-guides/knowledge` - Devin persistent knowledge model.
- `https://docs.devin.ai/product-guides/session-insights` - Devin episodic retrospective analysis.
- `https://docs.devin.ai/product-guides/creating-playbooks` - Devin procedural memory/playbooks.
- `https://docs.cloud.google.com/gemini/docs/codeassist/overview` - Gemini feature scope.
- `https://docs.cloud.google.com/gemini/docs/codeassist/configure-local-codebase-awareness` - Gemini local awareness behavior.
- `https://docs.cloud.google.com/gemini/docs/codeassist/code-customization-overview` - Gemini enterprise code customization memory.
- `https://docs.github.com/en/copilot/concepts/agents/copilot-memory` - Copilot memory model and validation semantics.
- `https://docs.github.com/en/copilot/how-tos/use-copilot-agents/copilot-memory` - Copilot memory controls and lifecycle.

**Related Artifacts:**

- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-08-inv-session-memory-landscape-2026-who.md` - This artifact.
- **Workspace:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-research-session-memory-landscape-08feb-9f82/` - Session workspace.

---

## Investigation History

**[2026-02-08 10:00]:** Investigation started

- Initial question: Who has the best session memory across coding tools, and how does orch compare?
- Context: Spawned issue `orch-go-21507` requesting cross-tool memory landscape analysis with emphasis on episodic memory.

**[2026-02-08 10:20]:** Source sweep completed

- Collected and compared docs for Claude Code, Cursor, Windsurf, Codex, Devin, Gemini Code Assist, and GitHub Copilot Memory.

**[2026-02-08 10:35]:** Orchard comparison synthesized

- Mapped orch artifact stack to declarative/procedural/episodic taxonomy and identified episodic automation gap.

**[2026-02-08 10:45]:** Investigation completed

- Status: Complete
- Key outcome: Copilot currently leads explicit episodic memory mechanics; orch should add first-class validated action-memory while preserving existing artifact strengths.
