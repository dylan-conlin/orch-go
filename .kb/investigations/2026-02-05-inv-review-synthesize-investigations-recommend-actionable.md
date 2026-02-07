## Summary (D.E.K.N.)

**Delta:** Six investigations reveal one root pattern: the system accumulated defenses against past failures without pruning, creating ~70% overhead that serves AI amnesia but burdens the human who created it.

**Evidence:** 94KB orchestrator skill (only 5 patterns load-bearing), 35+ checkpoints (40% theater), 30+ dead commands (8K LOC), 735 investigations, 1400+ lines instructions per spawn—all incident-driven accumulation without sunset reviews.

**Knowledge:** The friction isn't random—it's a design tension between AI-first (amnesia-resistant) and human-first (judgment-trusting) workflows. Ceremony that helps autonomous workers hurts interactive Dylan. The crossing point was Dec 2025 when skills/verification formalized.

**Next:** Implement tiered simplification: (1) Quick wins now—skill split, archive investigations, hide dead commands. (2) Larger refactor—trust tiers separating daemon path (strict) from interactive path (light).

**Authority:** strategic - This is a premise-level question about whether orch serves AI workers or Dylan. Simplification involves removing defenses built from real failures.

---

# Investigation: Synthesis of 6 Friction Point Investigations

**Question:** What are the common root causes across these 6 investigations, what's causing the most friction, and what concrete actions should we take?

**Started:** 2026-02-05
**Updated:** 2026-02-05
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-04-inv-escape-hatch-architecture-complexity.md | synthesized | yes | None |
| 2026-02-04-inv-analyze-meta-orchestrator-look-claude.md | synthesized | yes | None |
| 2026-02-04-inv-analyze-checkpoint-rituals-session-start.md | synthesized | yes | None |
| 2026-02-04-inv-analyze-94kb-orchestrator-skill-claude.md | synthesized | yes | None |
| 2026-02-04-inv-what-made-orch-feel-like-job.md | synthesized | yes | None |
| 2026-02-04-inv-orch-commands-usage-analysis.md | synthesized | yes | None |

---

## Findings

### Finding 1: One Root Pattern - Incident-Driven Accumulation Without Pruning

**Evidence:** Every investigation found the same growth pattern:

| System | How It Grew | Current State |
|--------|-------------|---------------|
| Orchestrator skill | Each incident added content | 12K→24K tokens in 5 weeks, reduced twice but kept growing back |
| Checkpoints | Each failure added a rule | 35+ checkpoints, 40% theater |
| Commands | Features added but never removed | 30+ dead commands, 8K LOC unused |
| Investigations | Every task creates one | 735 files, 50% unnecessary |
| Spawn modes | Each failure mode added escape hatch | 3 backends × 3 modes |

**Source:** All 6 investigations document this pattern with specific evidence.

**Significance:** The system learned by accumulating constraints but never unlearned. No sunset review process exists. Two reduction attempts on the orchestrator skill were outpaced by new additions.

---

### Finding 2: AI-First vs Human-First Design Tension

**Evidence:** The ceremony serves AI workers who have:
- No memory between sessions → need SYNTHESIS.md, investigation files
- No implicit understanding → need explicit rules for past failures
- No judgment → need mandatory checklists

But Dylan has:
- Memory across sessions
- Implicit understanding of context
- Judgment about what's necessary

**Source:** What-made-orch-feel-like-job investigation analysis.

**Significance:** The system optimized for the wrong user. ~70% of ceremony is defensive measures for AI amnesia. For autonomous daemon workers, this is valuable. For Dylan spawning a quick fix, it's overhead.

---

### Finding 3: Theater vs Gates - Only Automated Enforcement Works

**Evidence:** From checkpoint analysis:

| Category | Count | Enforcement | Real Value |
|----------|-------|-------------|------------|
| Hard gates (automated) | 11 | orch complete fails | High |
| Soft signals (visibility) | 6 | Context injection | Medium |
| Manual checklists | 11 | None - agent compliance | Low (easily gamed) |
| Session hooks | 7 | Auto-fire | Medium |

Manual checklists have escape hatches: "N/A", "No discovered work", "straightforward investigation". These are used liberally.

**Source:** Checkpoint rituals investigation, tested by observing actual agent behavior.

**Significance:** ~40% of process is unenforceable theater. The friction-to-value ratio is inverted—highest-friction checkpoints (11-item self-review) are easiest to bypass; lowest-friction (Phase: Complete) are most valuable.

---

### Finding 4: Core Essence Is Tiny, Bloat Is Vast

**Evidence:**

| Asset | Current Size | Core Essence | Bloat % |
|-------|--------------|--------------|---------|
| Orchestrator skill | 24K tokens | ~300 tokens (3 roles, 1 rule, 5 patterns) | 99% |
| Commands | 50+ | 8 core | 84% |
| Checkpoints | 35+ | ~12 (automated gates + hooks) | 66% |
| Spawn context | 1400+ lines | ~50 lines (task, skill, context) | 96% |

The orchestrator skill's core essence:
- 3 roles: COMPREHEND → TRIAGE → SYNTHESIZE
- 1 absolute rule: Never do spawnable work
- 5 load-bearing patterns (protected in skill.yaml)

Everything else is reference material (25%) or edge-case handling (15%).

**Source:** 94KB orchestrator skill investigation, command usage analysis.

**Significance:** Dramatic simplification is possible without losing load-bearing functionality.

---

### Finding 5: Escape Hatches Are Load-Bearing (Exception to Simplification)

**Evidence:**
- 19% of spawns use escape hatches (596/3143)
- Jan 10 incident: OpenCode crashed 3x while agents fixed observability—claude escape hatch saved the work
- Docker backend only 4% usage (marginal)
- Complexity is defensive (failure handling) not offensive (new capabilities)

**Source:** Escape hatch architecture investigation.

**Significance:** Unlike other accumulated complexity, escape hatches have proven value. The dual-mode architecture (headless + tmux, opencode + claude) should remain. Docker backend is the one exception—4% usage doesn't justify its complexity.

---

### Finding 6: The Crossing Point Was December 2025

**Evidence:** Timeline reconstruction:

| Period | State | Key Change |
|--------|-------|------------|
| Oct 2025 | Helpful | `orch spawn` just worked |
| Nov 2025 | Growing | D.E.K.N. format, investigation templates |
| Dec 2025 | Complex | Skills system, worker-base, completion verification |
| Jan 2026 | Heavy | 17+ investigations in 36 hours |
| Feb 2026 | Job | 34 kb reflect items, 1400+ lines spawn context |

**Source:** What-made-orch-feel-like-job investigation timeline analysis.

**Significance:** The formalization was well-intentioned (reliability, handoff quality) but crossed a threshold. The accumulation was never pruned because no pruning mechanism exists.

---

## Synthesis

**Key Insights:**

1. **One Root Cause: Accumulation without pruning** - Every investigation found systems that grew via incident response and never shrank. This is the meta-pattern. The fix isn't patching individual systems—it's establishing a sunset review process.

2. **Design tension is real and solvable** - The AI-first vs human-first split isn't philosophical—it maps to daemon path vs interactive path. Strict ceremony for autonomous workers, light ceremony for Dylan-initiated spawns. Trust tiers.

3. **40% of process is theater** - Manual checklists that agents game don't improve quality. Either automate them as hard gates or remove them. The middle ground (soft checklists) doesn't work.

4. **Core functionality is tiny** - 8 commands handle 80%+ of usage. 5 skill patterns are load-bearing. 12 checkpoints are gates. Everything else could be pruned or moved to reference files.

5. **Exception: Escape hatches stay** - Unlike other complexity, escape hatches have documented incidents proving their value. The 19% usage justifies the architecture. Docker backend (4%) is the one candidate for removal.

**Answer to Investigation Question:**

The common root cause is **incident-driven accumulation without sunset reviews**. The most friction comes from:

1. **Orchestrator skill bloat** (loads every session, dominates context)
2. **1400+ line spawn context** (ceremony before any work)
3. **735 investigations** (discovery friction, kb reflect noise)
4. **40% theater checkpoints** (cognitive load without enforcement)

Concrete actions below, prioritized by impact.

---

## Structured Uncertainty

**What's tested:**

- ✅ Orchestrator skill grew 12K→24K tokens in 5 weeks (verified: read stats.json)
- ✅ 35+ checkpoints exist, categorized by enforcement (verified: read hooks, skill templates)
- ✅ 30+ commands have zero events (verified: analyzed events.jsonl)
- ✅ 19% of spawns use escape hatches (verified: event analysis in escape hatch investigation)
- ✅ Manual checklists are bypassed with "N/A" (verified: observed agent behavior)

**What's untested:**

- ⚠️ Whether trust tiers would actually reduce friction (hypothesis, not implemented)
- ⚠️ Whether skill split would maintain effectiveness (hypothesis, not tested)
- ⚠️ Whether archiving old investigations would help or hurt discovery (hypothesis)

**What would change this:**

- If skill split causes orchestrator failures → bloat was load-bearing despite not being marked as such
- If removing theater checkpoints causes quality degradation → manual enforcement was working
- If trust tiers lead to more interactive-path bugs → ceremony was preventing issues Dylan didn't notice

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Quick wins (skill split, archive, hide commands) | architectural | Cross-component changes, reversible |
| Trust tiers | strategic | Changes fundamental system assumptions about who orch serves |
| Remove theater checkpoints | strategic | Removes defenses built from real failures, value judgment |

### Prioritized Actions

#### Tier 1: Quick Wins (High impact, low risk)

**Q1. Split orchestrator skill into core + reference files**

*Impact:* Context budget reduction from 24K to ~8-10K tokens
*Effort:* Medium (1-2 hours)
*Risk:* Low (reference files remain accessible)

```
Core skill (~8-10K tokens):
- Fast Path Surface Table
- Pre-Response Gates
- Context Detection
- ABSOLUTE DELEGATION RULE
- Triage Protocol (condensed)
- Links to reference files

Reference files:
- reference/model-selection.md
- reference/spawn-checklist.md
- reference/completion-workflow.md
- reference/daemon-operations.md
```

---

**Q2. Archive old investigations**

*Impact:* Reduce discovery friction, kb reflect noise
*Effort:* Low (1 command with date filter)
*Risk:* Very low (archived, not deleted)

```bash
# Move investigations older than 60 days to archive
mkdir -p .kb/investigations/archive/
find .kb/investigations/*.md -mtime +60 -exec mv {} .kb/investigations/archive/ \;
```

---

**Q3. Hide dead commands from help**

*Impact:* Cleaner UX, reduced cognitive load
*Effort:* Low (add Hidden: true to Cobra commands)
*Risk:* Very low (commands still work, just not advertised)

Commands to hide (zero events):
`attach`, `changelog`, `claim`, `docs`, `emit`, `fetch-md`, `guarded`, `history`, `learn`, `mode`, `model`, `patterns`, `reconcile`, `sessions`, `test-report`, `tokens`, `transcript`

---

**Q4. Make theater checkpoints explicitly optional**

*Impact:* Reduce skill template size, stop pretending
*Effort:* Low (edit skill templates)
*Risk:* Low (already optional in practice)

Convert from:
```markdown
## Self-Review Checklist (REQUIRED)
- [ ] Prior-Work acknowledged
- [ ] Cited claims verified
...
```

To:
```markdown
## Self-Review (Optional for experienced agents)
Consider reviewing: prior work, cited claims, test evidence
```

---

#### Tier 2: Larger Refactors (High impact, requires design)

**L1. Trust Tiers - Two paths through the system**

*Impact:* Fundamental friction reduction for interactive use
*Effort:* High (spawn context changes, skill flags, documentation)
*Risk:* Medium (could miss issues ceremony was preventing)

| Path | When | Ceremony Level |
|------|------|----------------|
| Daemon | Autonomous agents via `triage:ready` | Full (SYNTHESIS.md, phases, verification) |
| Interactive | Dylan spawns directly | Light (commit message, tests pass) |

Implementation:
- Add `--interactive` flag to spawn
- Skip: Phase reporting, SYNTHESIS.md requirement, investigation file creation
- Keep: Git commits, tests passing, beads tracking

---

**L2. Sunset review process**

*Impact:* Prevents future accumulation
*Effort:* Medium (process, not code)
*Risk:* Low (review, not automatic removal)

Quarterly review:
- For each checkpoint: "Did this catch a real problem in the last 90 days?"
- For each command: "Was this used in the last 90 days?"
- For each skill section: "Is this load-bearing or reference?"

Rules with no catches → demoted to optional → removed after second failure.

---

**L3. Consolidate spawn variants**

*Impact:* Simpler mental model
*Effort:* Medium (code refactoring)
*Risk:* Medium (behavior changes)

Current: `spawn`, `work`, `swarm` (3 commands)
Proposed: `spawn` with modes (`spawn --from-issue ID`, `spawn --batch`)

---

#### Tier 3: Aggressive Pruning (If Tiers 1-2 insufficient)

**A1. Remove Docker backend**

*Impact:* Simplify spawn architecture
*Evidence:* Only 4% usage (133/3143 spawns)
*Risk:* Rate limit scenarios require alternative solution

---

**A2. Delete dead command code**

*Impact:* ~8,000 LOC removed, faster builds, simpler codebase
*Effort:* Medium (verify each command is truly unused before deletion)
*Risk:* Medium (some commands may have undocumented users)

---

**A3. Convert soft signals to hard gates or remove**

Current soft signals (inject context but don't block):
- kb reflect suggestions
- Session duration warnings
- Pre-commit reminders

Either: Make them blocking gates OR remove them entirely. The middle ground (soft reminders) adds noise without enforcement.

---

### Alternative Approaches Considered

**Option B: Keep All Ceremony, Add Navigation**
- **Pros:** No risk of losing important defenses
- **Cons:** Doesn't reduce friction, just documents it better
- **When to use:** If trust tiers lead to quality problems

**Option C: Full Reset to Minimal Orch**
- **Pros:** Dramatic simplification
- **Cons:** Loses battle-tested defenses, requires rebuilding from scratch
- **When to use:** If incremental pruning proves impossible

---

## References

**Investigations Synthesized:**
1. `.kb/investigations/2026-02-04-inv-escape-hatch-architecture-complexity.md` - 19% escape hatch usage, Docker 4%
2. `.kb/investigations/2026-02-04-inv-analyze-meta-orchestrator-look-claude.md` - Collapsed correctly, principles survived
3. `.kb/investigations/2026-02-04-inv-analyze-checkpoint-rituals-session-start.md` - 35+ checkpoints, 40% theater
4. `.kb/investigations/2026-02-04-inv-analyze-94kb-orchestrator-skill-claude.md` - 24K tokens, 5 load-bearing patterns
5. `.kb/investigations/2026-02-04-inv-what-made-orch-feel-like-job.md` - 1400+ lines per spawn, Dec 2025 crossing
6. `.kb/investigations/2026-02-04-inv-orch-commands-usage-analysis.md` - 8 core, 30+ dead

**Key Files:**
- `~/.claude/skills/meta/orchestrator/SKILL.md` - 2,181 lines, 94KB
- `pkg/spawn/context.go` - SPAWN_CONTEXT template
- `~/.claude/settings.json` - 21 hooks
- `~/.orch/events.jsonl` - 19,172 events

---

## Investigation History

**2026-02-05:** Investigation started
- Initial question: Synthesize 6 investigations, prioritize recommendations
- Context: Orchestrator requested cross-investigation synthesis

**2026-02-05:** All 6 investigations read and analyzed
- Identified common root pattern: incident-driven accumulation
- Found design tension: AI-first vs human-first
- Quantified: 40% theater, 99% skill bloat, 84% command bloat

**2026-02-05:** Investigation completed
- Status: Complete
- Key outcome: Tiered simplification—quick wins (skill split, archive, hide) + larger refactor (trust tiers, sunset process)
