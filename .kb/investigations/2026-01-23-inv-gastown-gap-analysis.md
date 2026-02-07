<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Gastown and orch-go have fundamentally different approaches to knowledge externalization and human oversight. Gastown encodes knowledge in workflow templates (formulas) and automates verification, while orch-go builds persistent knowledge artifacts (.kb) and requires human-gated pushes. Gastown has design tooling, seancing for predecessor queries, and automated conflict resolution, but lacks explicit principles, decisions, or investigation patterns.

**Evidence:** Examined Gastown source: no .kb directory (verified), no CLAUDE.md files (verified), no principles.md (verified). Found design.formula.toml with 6-leg parallel analysis, gt seance command for predecessor communication, mol-polecat-conflict-resolve.formula.toml for automated reimagination.

**Knowledge:** The gap reveals philosophy differences: Gastown optimizes for velocity (autonomous agents, automated merging, ephemeral knowledge in wisps), orch-go optimizes for durability (persistent artifacts, human oversight, accumulated knowledge). Neither is wrong - they solve different problems at different scales.

**Next:** Document patterns worth cross-pollinating. Consider: (1) seance-style predecessor queries for orch-go, (2) kb-cli style knowledge artifacts for Gastown users, (3) design formula patterns as skill inspiration.

**Promote to Decision:** recommend-no - This is comparative analysis. Philosophy differences are deliberate, not deficiencies.

---

# Investigation: Gastown Gap Analysis - What They DON'T Have That We Do

**Question:** What does orch-go/kb-cli have that Gastown lacks, and what does that reveal about philosophy differences?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Investigation worker
**Phase:** Complete
**Status:** Complete

---

## Findings

### Finding 1: No Knowledge Management System

**Test performed:**
```bash
ls /Users/dylanconlin/Documents/personal/gastown/.kb
# Error: No such file or directory

kb create investigation test
# Error: no .kb directory found
```

**Observation:** Gastown has no .kb directory, no investigations, no decisions, no guides, no models. Knowledge externalization happens through:
- Formulas (TOML workflow templates in `.beads/formulas/`)
- Role templates (`templates/polecat-CLAUDE.md`, `templates/witness-CLAUDE.md`)
- Beads themselves (issues as work units)
- Wisps (ephemeral beads that don't persist)

**Gap analysis:**

| orch-go Has | Gastown Has | Philosophy Difference |
|-------------|-------------|----------------------|
| `.kb/investigations/` | None | orch-go builds understanding, Gastown executes |
| `.kb/decisions/` | None | orch-go preserves choices, Gastown ephemeral |
| `.kb/guides/` | Formulas | orch-go teaches patterns, Gastown encodes steps |
| `.kb/models/` | None | orch-go synthesizes, Gastown doesn't |
| `kb quick` entries | Beads | orch-go captures learning, Gastown tracks work |
| Principles.md | None | orch-go encodes values, Gastown implicit |

**Significance:** Gastown is execution-focused. Knowledge lives in agents' heads and dies with sessions. The seance mechanism (`gt seance`) partially compensates by allowing queries to predecessor sessions, but this is synchronous and ephemeral - not persistent artifacts.

---

### Finding 2: No CLAUDE.md Files in Repository

**Test performed:**
```bash
find /Users/dylanconlin/Documents/personal/gastown -name "CLAUDE.md" -type f
# No results

cat /Users/dylanconlin/Documents/personal/gastown/AGENTS.md
# "See CLAUDE.md for complete agent context..."
```

**Observation:** AGENTS.md references CLAUDE.md, but none exist. Context is injected dynamically via `gt prime`:

```go
// internal/session/startup.go:69-71
// For handoff, cold-start, and attach, add explicit instructions so the agent knows
// what to do even if hooks haven't loaded CLAUDE.md yet
```

**Gap analysis:**

| orch-go Has | Gastown Has | Philosophy Difference |
|-------------|-------------|----------------------|
| `~/.claude/CLAUDE.md` | Dynamic injection | orch-go: static, versioned. Gastown: computed |
| Project CLAUDE.md | Role templates | orch-go: one file per project. Gastown: per role |
| Principles with teeth | GUPP/NDI/MEOW concepts | orch-go: explicit constraints. Gastown: philosophy embedded |

**Significance:** Gastown embeds agent instructions in role templates (`polecat-CLAUDE.md`, `witness-CLAUDE.md`) rather than a single CLAUDE.md. This enables role-specific context but loses the single-source-of-truth pattern.

---

### Finding 3: Verification is Automated, Not Human-Gated

**Evidence from templates/witness-CLAUDE.md:89-127:**

```markdown
## Pre-Kill Verification Checklist
[ ] 1. gt polecat check-recovery {{RIG}}/<name>  # Must be SAFE_TO_NUKE
[ ] 2. gt polecat git-state <name>               # Must be clean
[ ] 3. Verify issue closed: bd show <issue-id>   # Should show 'closed'
[ ] 4. Verify PR submitted (if applicable)
```

**Evidence from mol-refinery-patrol.formula.toml:**

```toml
# Refinery runs tests and merges autonomously
[[steps]]
id = "run-tests"
description = "Run the test suite... ALL TESTS MUST PASS."

[[steps]]
id = "merge-push"
description = "Merge to main and push."
```

**Gap analysis:**

| orch-go Has | Gastown Has | Philosophy Difference |
|-------------|-------------|----------------------|
| Human-gated push | Refinery auto-merge | orch-go: human reviews before push |
| `orch complete` verification | Witness pre-kill check | orch-go: orchestrator validates. Gastown: agents verify each other |
| Coaching plugin | None | orch-go: catches issues during session |

**Significance:** Gastown trusts agents to verify each other. Witness checks Polecats, Refinery checks code. Human is notified via escalation only on failures. orch-go requires human review before code reaches remote.

---

### Finding 4: Design Tooling EXISTS (Contrary to Appleton's Concern)

**Evidence from design.formula.toml:**

```toml
description = """
Structured design exploration via parallel specialized analysts.
"""
type = "convoy"

[[legs]]
id = "api"
title = "API & Interface Design"

[[legs]]
id = "data"
title = "Data Model Design"

[[legs]]
id = "ux"
title = "User Experience Analysis"

[[legs]]
id = "scale"
title = "Scalability Analysis"

[[legs]]
id = "security"
title = "Security Analysis"

[[legs]]
id = "integration"
title = "Integration Analysis"
```

**Gap analysis:**

| orch-go Has | Gastown Has | Comparison |
|-------------|-------------|------------|
| `architect` skill | `design.formula.toml` | Both structured. Gastown parallel, orch-go sequential |
| `design-session` skill | Same formula | Both interactive exploration |
| `feature-impl` investigation | Formula steps | Both phase-based |

**Significance:** Appleton noted design as the bottleneck. Gastown actually has design tooling - it spawns 6 parallel polecats to analyze different dimensions and synthesizes results. This is MORE structured than orch-go's single-agent architect skill.

---

### Finding 5: Seance Enables Predecessor Queries (We Don't Have)

**Evidence from internal/cmd/seance.go:**

```go
// gt seance --talk <session-id>
// Spawns: claude --fork-session --resume <id>

// "Where did you put the stuff you left for me?" - The #1 handoff question.
// Instead of parsing logs, seance spawns a Claude subprocess that resumes
// a predecessor session with full context.
```

**Gap analysis:**

| orch-go Has | Gastown Has | Philosophy Difference |
|-------------|-------------|----------------------|
| SYNTHESIS.md (static) | Seance (dynamic) | orch-go: write artifacts. Gastown: query sessions |
| SESSION_HANDOFF.md | gt handoff + mail | orch-go: static handoff. Gastown: mail + respawn |
| /handoff skill | gt seance command | Gastown can ASK predecessors questions |

**Significance:** Seance is a genuine innovation. When an agent inherits context but can't find expected state, it can literally ask its predecessor "where is the stuff you left?" This is more powerful than static handoff documents - but ephemeral.

---

### Finding 6: Conflict Reimagination is Automated

**Evidence from mol-polecat-conflict-resolve.formula.toml:**

```toml
## Key Differences from Regular Polecat Work
| Aspect | Regular Work | Conflict Resolution |
| Merge path | Submit to queue via `gt done` | Push directly to main |
| Serialization | None | Merge-slot gate required |

# Resolve conflicts using your judgment
# If stuck on a conflict: escalate to Witness
```

**Gap analysis:**

| orch-go Has | Gastown Has | Philosophy Difference |
|-------------|-------------|----------------------|
| Human decides on conflicts | Fresh polecat reimagines | orch-go: human judgment. Gastown: agent re-implementation |
| Re-spawn manually | Refinery spawns resolution task | Gastown: automated workflow |
| No merge-slot | bd merge-slot acquire | Gastown: serializes conflict resolution |

**Significance:** When Refinery can't mechanically rebase, it creates a conflict-resolution task. A fresh polecat "reimagines" the change by understanding original intent and resolving conflicts. This is autonomous - human only involved on escalation.

---

## Gap Table: What We Have That They Don't

| Capability | orch-go/kb-cli | Gastown | Impact |
|------------|----------------|---------|--------|
| **Persistent Knowledge** | `.kb/` with investigations, decisions, guides, models | None | Gastown loses learnings between sessions |
| **Explicit Principles** | `principles.md` with provenance and teeth | GUPP/NDI/MEOW as philosophy | Gastown principles are implicit, not enforceable |
| **Project CLAUDE.md** | Static, versioned, single source of truth | Dynamic injection per role | Gastown: more flexible. orch-go: more traceable |
| **Human-Gated Push** | Workers commit locally, orchestrator reviews | Refinery auto-merges | orch-go: safer. Gastown: faster |
| **kb quick Entries** | Capture learning in real-time | Beads track work, not knowledge | Gastown: work-focused. orch-go: knowledge-focused |
| **Skill System** | Procedural skills with dependencies | Formulas | Similar patterns, different vocabulary |
| **Investigation Discipline** | Test before concluding | Execute and escalate | orch-go: understand then act. Gastown: act then escalate |

## What They Have That We Don't

| Capability | Gastown | orch-go | Impact |
|------------|---------|---------|--------|
| **Seance** | Query predecessor sessions | Static SYNTHESIS.md | Gastown: dynamic context. orch-go: static artifacts |
| **Wisps** | Ephemeral beads, no Git pollution | All beads persisted | Gastown: cleaner history. orch-go: full audit trail |
| **Parallel Design** | 6-leg convoy formula | Single architect agent | Gastown: broader analysis. orch-go: deeper focus |
| **Automated Conflict Resolution** | Fresh polecat reimagines | Human decides | Gastown: faster. orch-go: more controlled |
| **Role Templates** | Per-role CLAUDE.md equivalent | Single project CLAUDE.md | Gastown: role-specific. orch-go: project-wide |
| **Merge Slot** | Serializes conflict resolution | N/A (human gated) | Gastown: prevents races |

---

## Philosophy Insights

### 1. Execution vs Understanding

**Gastown:** Optimizes for execution velocity. Agents run, complete, die. Knowledge lives in predecessor sessions (queryable via seance) and workflow templates (formulas). When agents get stuck, they escalate to Witness who nudges or respawns.

**orch-go:** Optimizes for understanding accumulation. Agents investigate, document, hand off. Knowledge persists in `.kb/` artifacts. When agents complete work, they externalize learnings.

**Insight:** Gastown assumes knowledge is cheap to regenerate (just seance your predecessor). orch-go assumes knowledge is expensive and should be preserved.

### 2. Trust Model

**Gastown:** Agents trust each other. Witness trusts Polecats did work. Refinery trusts tests. Human intervenes on escalation only.

**orch-go:** Orchestrator trusts workers selectively. Human reviews before push. Coaching plugin catches issues during session. Verification is built into workflow.

**Insight:** Gastown is designed for high-trust, high-velocity teams. orch-go is designed for safety-first, learning-focused work.

### 3. Memory Architecture

**Gastown:** Short-term memory in sessions, queryable via seance. Medium-term in beads/molecules. Long-term in formulas (patterns). No persistent knowledge layer.

**orch-go:** Short-term in session. Medium-term in SYNTHESIS.md/SESSION_HANDOFF.md. Long-term in `.kb/` with investigations, decisions, guides, models, principles.

**Insight:** orch-go has a richer memory hierarchy. Gastown relies on session resumption as the memory mechanism.

---

## Conclusion

The gap is philosophical, not deficiency. Gastown optimizes for:
- **Execution velocity** (agents complete work fast)
- **Session recovery** (seance, GUPP, handoffs)
- **Automated verification** (Witness, Refinery)

orch-go optimizes for:
- **Knowledge accumulation** (.kb artifacts)
- **Human oversight** (gated push, coaching)
- **Understanding preservation** (investigations, decisions)

Neither is wrong. They serve different purposes:
- **Gastown:** 20-30 agents working in parallel on a mature codebase you trust
- **orch-go:** 1-5 agents working carefully on code that matters, with full audit trail

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/gastown/templates/polecat-CLAUDE.md` - Polecat contract
- `/Users/dylanconlin/Documents/personal/gastown/templates/witness-CLAUDE.md` - Witness responsibilities
- `/Users/dylanconlin/Documents/personal/gastown/internal/cmd/seance.go` - Seance implementation
- `/Users/dylanconlin/Documents/personal/gastown/.beads/formulas/design.formula.toml` - Design convoy
- `/Users/dylanconlin/Documents/personal/gastown/.beads/formulas/mol-polecat-conflict-resolve.formula.toml` - Conflict resolution
- `/Users/dylanconlin/Documents/personal/gastown/.beads/formulas/mol-refinery-patrol.formula.toml` - Refinery patrol
- `/Users/dylanconlin/Documents/personal/gastown/docs/concepts/propulsion-principle.md` - GUPP
- `/Users/dylanconlin/Documents/personal/gastown/docs/glossary.md` - Terminology
- `/Users/dylanconlin/Documents/personal/gastown/docs/design/escalation-system.md` - Escalation design

**Prior Investigation:**
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-23-inv-gastown-orchestration-system-analysis-compare.md`

**External Analysis:**
- `~/Documents/personal/blog/sources/gastown-maggie-appleton.md` - Appleton's analysis

---

## Investigation History

**2026-01-23 19:30:** Investigation started
- Question: What does orch-go have that Gastown lacks?
- Approach: Search for .kb, CLAUDE.md, principles, then examine workflows

**2026-01-23 19:45:** Core gaps identified
- No .kb directory (verified)
- No CLAUDE.md files (verified)
- No principles.md (verified)
- Dynamic context injection via gt prime

**2026-01-23 20:00:** Counter-findings discovered
- Design formula EXISTS (contrary to Appleton's concern)
- Seance is a genuine innovation we don't have
- Conflict resolution is more automated than expected

**2026-01-23 20:15:** Investigation completed
- Gap table produced
- Philosophy differences documented
- Status: Complete
