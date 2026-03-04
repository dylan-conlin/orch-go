# Design: Simplify Orchestrator Skill

**Date:** 2026-03-04
**Status:** Active — deployed, testing blocked
**Phase:** Validation (blocked)
**Triggered by:** Task orch-go-w7xe — skill at 2,368 lines, behavioral testing shows constraints don't work, hooks now enforce boundaries

## TLDR

Strip orchestrator skill from 2,368→~450 lines. Keep knowledge transfer (routing tables, vocabulary, intent distinctions — ~47 items that survive dilution). Keep ≤4 behavioral norms (knowledge framing, not prohibition). Remove ~20 behavioral constraints now enforced by 6 infrastructure hooks. Consider Claim 4 separation (grammar, routing table, legibility protocol).

## Question

What is the minimum viable orchestrator skill that preserves behavioral lift on knowledge-transfer scenarios while eliminating the dead weight of behavioral constraints?

## Evidence Base

### Behavioral Testing (Mar 1 baseline)
- Both skill variants (v2.1 259 lines, 7686b131 457 lines) score ~22/56 vs bare 17/56
- 5 of 7 scenarios are bare-parity violations
- Knowledge sticks: routing tables (+4/8), framing vocabulary (+2/8), intent distinction (+2/8)
- Constraints don't: delegation speed (1/8=bare), reconnection (0-1/8=bare), anti-sycophancy (3/8=bare)

### Constraint Dilution (Mar 1-2 probes)
- Behavioral constraint budget: ≤4 co-resident constraints before dilution
- At 10+ constraints: bare parity (zero behavioral return)
- Knowledge constraint budget: ≤50 (functional at 10+)
- Emphasis language provides +1-2 positions on dilution curve but can't overcome fundamental limit

### Hook Infrastructure (6/7 active, verified)
| Hook | Enforces | Status |
|------|----------|--------|
| bash-write gate | No Edit/Write tools | Working (308 tests) |
| git-remote gate | No git push | Working (64 tests) |
| bd-close gate | Only orch complete closes | Working (37 tests) |
| investigation-drift nudge | Don't investigate, spawn | Working (38 tests) |
| spawn-ceremony nudge | Run kb context before spawn | Working (40 tests) |
| spawn-context validation | --issue and --intent required | Working (68 tests) |
| code-access gate | Coaching on code reads | NOT REGISTERED |

## Design Approach

### Principle: Knowledge framing, not prohibition framing

Current: "NEVER do X" repeated 20+ times → dilution → bare parity
New: "The system works like Y" → knowledge transfer → measurable lift

### What stays (~47 knowledge items)

**Routing/Decision (highest value):**
- Fast path surface table (13 rows)
- Skill decision tree (10 mappings)
- Intent clarification (experiential/production/comparative)
- Strategic-first for hotspots
- Stall triage failure modes
- Investigation/probe outcome classification

**Vocabulary/Concepts:**
- Three jobs (COMPREHEND, TRIAGE, SYNTHESIZE)
- Tool ecosystem (orch, bd, kb, skillc, opencode)
- Work pipeline (release to daemon)
- Session model, label taxonomy, spawn modes

**Dylan Interface:**
- Signal prefixes, mode declarations
- Three-layer reconnection
- Frustration protocol, epic model phases
- Observability architecture

### What goes (enforced by hooks, dead weight)
- All "NEVER use Edit/Write" text (~50 lines) → bash-write gate
- All "NEVER git push" text (~30 lines) → git-remote gate
- All "NEVER bd close" text (~20 lines) → bd-close gate
- Pre-spawn kb context ceremony (~40 lines) → spawn-ceremony nudge
- --issue/--intent requirement text (~30 lines) → spawn-context validation gate
- Multiple repetitions of delegation rule (~100 lines) → investigation-drift nudge
- All anti-pattern recognition tables for hook-enforced behaviors (~80 lines)
- Detailed protocol checklists for hook-enforced behaviors (~100 lines)

### ≤4 Behavioral Norms (reformulated)
1. **Delegation norm:** "You never implement. You COMPREHEND, TRIAGE, SYNTHESIZE."
2. **Filter before presenting:** "Only present options you'd recommend."
3. **Act by default:** "If obvious next step, act without asking."
4. **Answer the question asked:** "Investigation findings ≠ design decision."

### Claim 4 Separation (Partial)

The model says skills fuse three artifacts: grammar, routing table, legibility protocol.

**Decision:** Partial separation.
- Grammar (command vocabulary) → already in reference/tools-and-commands.md
- Routing table → stays in core (highest value, always-loaded)
- Legibility protocol → stays in core as labeled section (Dylan interface knowledge)

Full separation would degrade always-available Dylan communication knowledge. The remaining content after constraint removal IS naturally separated into routing + legibility.

## What I tried

1. **Quality review** — Verified v4 template (422 lines, 4,830 tokens) against design criteria:
   - All ~47 knowledge items present (routing tables, vocabulary, Dylan interface)
   - Exactly 4 behavioral norms (delegation, filter, act-by-default, answer-the-question)
   - No prohibition framing — knowledge framing throughout
   - No hook-enforced constraint text (all 6 hooks handle their own enforcement)
   - Load-bearing patterns: all 4 present (skillc check passes)

2. **Build & deploy** — `skillc deploy --target ~/.claude/skills/ --cleanup`
   - Deployed 448-line v4 to `~/.claude/skills/skills/src/meta/orchestrator/SKILL.md`
   - Cleaned 10 orphaned files including old 2,368-line version at `~/.claude/skills/meta/orchestrator/SKILL.md`

3. **Behavioral testing (blocked)** — `skillc test` requires nested `claude --print` calls
   - From spawned agent: CLAUDECODE env var blocks nested sessions
   - `skillc test` env stripping doesn't work in current version (returns 0/0 scores)
   - Must run from terminal: `skillc test --scenarios skills/src/meta/orchestrator/.skillc/tests/scenarios/ --variant skills/src/meta/orchestrator/SKILL.md --model sonnet`

## What I observed

- v4 template is well-structured: 16 sections covering routing, spawning, completion, Dylan interface, knowledge capture
- Token reduction: 27,200 → 4,830 (82% reduction)
- Line reduction: 2,368 → 448 (81% reduction)
- Claim 4 partial separation: grammar in reference/, routing table and legibility protocol in core (correct decision per design)
- Deploy path changed from `meta/orchestrator/` to `skills/src/meta/orchestrator/` — skill still discoverable

## Conclusion

**v4 deployed but behavioral gate pending.** The skill meets all design criteria:
- Knowledge transfer preserved (~47 items)
- ≤4 behavioral norms (knowledge framing)
- Hook-enforced constraints removed
- 82% token reduction

**Remaining gate:** `skillc test` bare-parity regression check must run from a non-Claude-Code terminal session. Run:
```bash
cd ~/Documents/personal/orch-go
skillc test --scenarios skills/src/meta/orchestrator/.skillc/tests/scenarios/ --variant skills/src/meta/orchestrator/SKILL.md --model sonnet
skillc test --scenarios skills/src/meta/orchestrator/.skillc/tests/scenarios/ --bare --model sonnet
skillc compare <v4-result.json> <bare-result.json>
```

If v4 scores ≥ bare on knowledge-transfer scenarios (01-intent-clarification, 03-complex-architectural-routing), the simplification is validated.
