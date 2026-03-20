# Probe: Prompt Context as Accreting Substrate

**Model:** knowledge-accretion
**Date:** 2026-03-20
**Status:** Complete
**claim:** extends substrate-generalization
**verdict:** extends

---

## Question

The knowledge-accretion model's substrate generalization table covers code, knowledge (.kb/), runtime behavior, OPSEC, and four hypothesized substrates. It does NOT cover **prompt context** — the text injected into every agent's context window (CLAUDE.md, skill files, SPAWN_CONTEXT.md). Is prompt context an accreting substrate? Does it meet the five conditions? What are its unique accretion signatures?

---

## What I Tested

### 1. CLAUDE.md Growth Trajectory

```bash
git log --all --follow --format="%h %as" -- CLAUDE.md | while read hash date; do
  size=$(git show "$hash:CLAUDE.md" 2>/dev/null | wc -l)
  echo "$date $hash $size"
done
```

### 2. Staleness/Contradiction Detection

Cross-referenced CLAUDE.md claims against actual source code:
- `DefaultModel` value in `pkg/model/model.go`
- Guide file existence checks
- Event type coverage (CLAUDE.md vs `pkg/events/`)
- Command existence (`kb audit decisions`)

### 3. Content Type Analysis

Manually classified every CLAUDE.md section as DIRECTIVE (agents must act on) vs REFERENCE (passive context).

### 4. Skill File Growth

```bash
git log --all --format="%h %as" -- "skills/src/shared/worker-base/.skillc/*" | while read hash date; do
  total=0; for f in $(git ls-tree -r --name-only "$hash" -- "skills/src/shared/worker-base/.skillc/"); do
    lines=$(git show "$hash:$f" | wc -l); total=$((total + lines))
  done; echo "$date $total"
done
```

### 5. SPAWN_CONTEXT.md Size Distribution

```bash
for f in .orch/workspace/*/SPAWN_CONTEXT.md; do
  lines=$(wc -l < "$f"); echo "$lines $f"
done | sort -rn
```

### 6. Total Context Budget Per Agent

Summed CLAUDE.md + worker-base skill + skill-specific SKILL.md + SPAWN_CONTEXT.md for each skill type.

---

## What I Observed

### CLAUDE.md Growth: 8x in 3 months

| Date | Lines | Delta | Milestone |
|------|-------|-------|-----------|
| 2025-12-19 | 93 | — | Initial creation |
| 2025-12-28 | 304 | +211 | Early expansion |
| 2026-01-10 | 478 | +174 | Major structure additions |
| 2026-01-21 | 391 | -87 | Pruning event (rare) |
| 2026-02-07 | 527 | +136 | Sustained growth |
| 2026-02-12 | 333 | -194 | Major pruning event (also rare) |
| 2026-02-14 | 352 | +19 | Re-growth begins immediately |
| 2026-03-14 | 485 | +133 | Steady accretion |
| 2026-03-20 | 753 | +268 | Current (+55% in 6 days) |

**Key pattern:** Two pruning events (Jan 21, Feb 12) temporarily reduced size. Both were followed by immediate re-growth. The file has never stayed pruned — accretion is a ratchet.

**Growth rate:** 93→753 lines over 91 days = **7.2 lines/day average**. But the rate is accelerating: Mar 14-20 alone added 268 lines (38 lines/day) — 5x the average.

**Contributors:** 48 commits by "Test User" (agent-spawned), 29 by Dylan Conlin. **62% of CLAUDE.md modifications are by agents**, not humans. The file's primary authors are amnesiac.

**Commit classification:** 27 of 77 total commits (35%) are tagged "docs:", "sync", or "artifact drift" — automated/semi-automated sync processes that add content without human review.

### Contradiction Found: Three Different Default Models

CLAUDE.md contains two contradictory claims about the default model:
- Line 267 (Key Packages section): `Default: google/gemini-3-flash-preview`
- Line 561 (Gotchas section): `Model default: Opus (Max subscription), not Gemini`

Actual code (`pkg/model/model.go:19`): `claude-sonnet-4-5-20250929`

**All three disagree.** This is a textbook accretion signature: different agents updated different sections at different times, each locally correct, but the file never had a coherence check.

### Staleness Found

1. **`kb audit decisions` still listed** in Commands (line 433) despite commit "fix: delete kb audit decisions — 54% false positive rate." The code still exists (the commit didn't delete it, just the decision to deprecate was recorded), but the command reference doesn't note the 54% false positive rate or any deprecation status.

2. **Event tracking table** (lines 623-714): 91 rows listing 85+ event types. Many events in the table (e.g., `session.labeled`, `session.send`, `agent.wait.complete`, `focus.set`) are NOT found in the events package source, suggesting they're documented but not emitted, or emitted from cmd/orch directly. The table grows monotonically — no event has ever been removed.

3. **SSE references**: Line 260 says "SSEClient for real-time event streaming" and line 562 says "SSE parsing: Event type is inside JSON data, not event: prefix." But the SSE client code has a comment: "Replaced SSE-based implementation with polling for simplicity." The SSE gotcha is about a system that no longer exists.

### Content Type Distribution: 92% Reference, 8% Directive

| Content Type | Lines | % | Description |
|-------------|-------|---|-------------|
| Architecture tree | 117 | 15.5% | File/package listing |
| Commands reference | 166 | 22.0% | Every orch subcommand |
| Event types table | 91 | 12.1% | All 85+ event types |
| Key packages | 96 | 12.7% | Package descriptions |
| Spawn backends | 57 | 7.6% | Backend architecture |
| All other reference | 164 | 21.8% | Common commands, development, related |
| **Directive content** | **62** | **8.2%** | Accretion boundaries, constraints, gotchas, tab editing |

**The top 5 reference sections account for 527 lines (70% of the file).** Only 62 lines contain content that actively shapes agent behavior (the accretion boundary rule, the no-local-state constraint, the gotchas, and the tab-editing workarounds).

**Implication:** Agents receive ~700 lines of passive reference material in every session. Whether they use it depends on task relevance — an agent doing a daemon fix doesn't need the event tracking table, the account management section, or the OpenCode fork section.

### Skill File Accretion

| Skill | Lines | Type |
|-------|-------|------|
| ux-audit | 2,827 | Outlier — 3.3x next largest |
| meta-orchestrator | 875 | Meta skill |
| design-session | 838 | Worker |
| orchestrator | 512 | Meta skill |
| research | 506 | Worker |
| worker-base | 429 | Shared (injected into ALL workers) |
| investigation | 267 | Worker |
| feature-impl | 273 | Worker |

**Worker-base growth:** 693→874 lines over 18 days (Mar 2-20). This is the shared base injected into every worker agent, growing at 10 lines/day.

**The stacking problem:** A worker agent receives CLAUDE.md (753) + worker-base (429) + skill-specific (267-326) + SPAWN_CONTEXT.md (954 avg) = **2,400-2,500 lines of injected context** before it reads a single line of the task's code. This is the total prompt context budget:

| Agent Type | CLAUDE.md | Worker-Base | Skill | SPAWN_CONTEXT | **Total** |
|------------|-----------|-------------|-------|---------------|-----------|
| investigation | 753 | 429 | 267 | 954 | **2,403** |
| feature-impl | 753 | 429 | 273 | 954 | **2,409** |
| architect | 753 | 429 | 326 | 954 | **2,462** |
| orchestrator | 753 | — | 512 | 954 | **2,219** |

### SPAWN_CONTEXT.md: Accretes Over Time But Not Across Sessions

SPAWN_CONTEXT.md is generated fresh per-agent, so it doesn't accrete within a file. But the **template** that generates it accretes:

| Date | Template Lines | Worker Template Lines |
|------|---------------|----------------------|
| Early | — | ~200 (estimated from Jan 08 workspace: 757 lines total) |
| Mar 20 | — | 477 lines |

The template (`worker_template.go`) is 477 lines. But SPAWN_CONTEXT size depends heavily on KBContext injection — the `kb context` output that varies by task. Range across 129 workspaces: **418-1,204 lines** (avg 954).

**Crucially:** SPAWN_CONTEXT.md size grows over time because the KB grows. More models, more constraints, more guides = more context injected. Average size by date:

| Date Range | Avg Size | Count |
|------------|----------|-------|
| Jan 08 | 757 | 1 |
| Mar 15 | 1,016 | 5 |
| Mar 17 | 913 | 31 |
| Mar 18 | 960 | 41 |
| Mar 19 | 953 | 32 |
| Mar 20 | 1,030 | 14 |

The upward trend (757→1,030) over 71 days = **+36%** growth in average SPAWN_CONTEXT size, driven by KB growth, not template growth.

---

## Model Impact

- [x] **Extends** model with: Prompt context as a confirmed accreting substrate with unique properties

### Prompt Context Substrate Properties

| Property | Value |
|----------|-------|
| **Mutable** | Yes — agents and humans modify CLAUDE.md |
| **Shared** | Yes — injected into every agent's context window |
| **Compositional** | Partially — sections must be internally consistent (e.g., default model) but are compositionally weak (most sections are independent) |
| **Amnesiac** | Yes — 62% of CLAUDE.md commits are by agents with no memory of prior state |

**Meets 4 of 4 minimal substrate properties.** Prompt context is a confirmed accreting substrate.

### Unique Properties (Differentiating From Other Substrates)

1. **Read amplification** — Every token of accretion is read by every agent in every session. Code accretion (daemon.go +892 lines) only impacts agents working on daemon code. CLAUDE.md accretion impacts ALL agents regardless of task. The blast radius is total.

2. **Signal-to-noise ratio degrades with size** — Only 8.2% of CLAUDE.md is directive content. The remaining 91.8% is reference material that may or may not be relevant to any given agent. As the file grows, the ratio gets worse: new content is predominantly reference (event types, commands, package descriptions), while directive content stays roughly constant.

3. **Re-accretion after pruning** — Two pruning events (Jan 21, Feb 12) were both followed by immediate re-growth. Without a gate preventing additions, the substrate behaves as a ratchet. This matches the code accretion pattern: `spawn_cmd.go` shrank -1,755 then regrew +483 in 3 weeks.

4. **Automated accretion** — 35% of CLAUDE.md commits are automated "artifact drift" syncs. The daemon runs `orch artifact-sync`, which adds new packages, events, and commands to CLAUDE.md without human review of whether those additions are useful context for agents.

5. **Multiplicative cost** — CLAUDE.md's 753 lines are consumed by ~40-100 agents per week. At ~1.5 tokens/line, that's ~1,130 tokens × ~50 agents/week = ~56,500 tokens/week of pure consumption. The event tracking table alone (91 lines, ~137 tokens) costs ~6,850 tokens/week across all agents — for content that no agent action has been traced to.

### Proposed Addition to Substrate Table

| Substrate | Accretion | Attractors | Gates | Entropy Signal | Status |
|-----------|-----------|-----------|-------|----------------|--------|
| **Prompt context** (CLAUDE.md, skills, SPAWN_CONTEXT.md) | 93→753 lines in 91 days (8x); 3 contradictory claims; 92% passive reference; 35% auto-committed | Sections act as attractors — "Event Tracking" pulls new event rows, "Commands" pulls new subcommands | **None** — no size limit, no staleness check, no consistency validation, no relevance filtering | Contradiction count, directive-to-reference ratio, section-level staleness | **Confirmed** |

### Proposed Gate Mechanisms

1. **Section-level TTL** — Mark each CLAUDE.md section with a last-verified date. Artifact-sync should not add content without verifying existing content is still accurate.

2. **Contradiction detection** — Automated check that scans CLAUDE.md for claims about defaults, paths, and tool names, and cross-references against actual code. Run as pre-commit or daemon periodic task.

3. **Size budget** — Cap CLAUDE.md at a target (e.g., 500 lines for directive+key reference). The event tracking table and full command reference could be moved to `.kb/guides/` and injected only when relevant.

4. **Relevance filtering** — Instead of injecting all of CLAUDE.md into every agent, inject a core section (directives + gotchas, ~62 lines) plus task-relevant sections based on skill type.

---

## Notes

- The SPAWN_CONTEXT.md constraint "SPAWN_CONTEXT injection volume is a false signal for KB atom read rate" (from prior knowledge) is confirmed and extended: it's not just that injection volume doesn't correlate with read rate — the injection volume itself is an accreting substrate whose growth is driven by KB growth. As the KB gets bigger, every agent gets a bigger context window consumed by context, leaving less room for the actual task.

- The ux-audit skill at 2,827 lines is a dramatic outlier. If that skill is used, the total context budget would be ~4,963 lines — over twice the typical agent. This warrants investigation: is the skill actually used? If so, do agents degrade when consuming that much context?

- The worker-base skill (429 lines, injected into every worker) is a secondary CLAUDE.md — it exhibits the same shared-mutable-amnesiac properties. Its growth rate (693→874 in 18 days) should be tracked alongside CLAUDE.md's.
