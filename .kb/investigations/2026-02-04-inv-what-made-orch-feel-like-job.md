# Investigation: What Made Orch Feel Like a Job

**Question:** What operational overhead makes orch feel like work instead of fun? What's essential vs accumulated? When did it cross from helpful to burdensome?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Worker agent (orch-go-21299)
**Phase:** Complete
**Status:** Complete

---

## Summary (D.E.K.N.)

**Delta:** Orch crossed from helpful to work when it accumulated mandatory ceremony at every touchpoint: 735 investigations, 75 decisions, 34 guides, SYNTHESIS.md templates, SPAWN_CONTEXT protocols, beads comments for phase tracking, kb quick for externalization, self-review checklists, harm assessments, and scope enumeration. Each individually reasonable, collectively exhausting.

**Evidence:** SPAWN_CONTEXT template alone has 400+ lines of instructions. Workers must remember: (1) report Phase: Planning within 3 tool calls, (2) create investigation file via `kb create`, (3) update investigation progressively, (4) run `kb quick` before completing, (5) pass self-review checklist, (6) report Phase: Complete, (7) call `/exit`. Feature-impl skill adds: scope enumeration, harm assessment, TDD discipline, visual verification for web changes, discovered work tracking.

**Knowledge:** The system optimized for preventing past mistakes rather than enabling flow. Each ceremony exists because an agent once failed without it - but the accumulated weight makes every spawn feel like paperwork. The core issue: **orch became process for AI workers, not a tool for humans**.

**Next:** Consider "trust tiers" - let experienced humans/orchestrators opt out of ceremony. The daemon path (headless, automated) should stay strict. The interactive path (Dylan spawning agents) should be lightweight. Fun looks like: `claude "fix the login bug"` → it fixes it → done.

---

## Findings

### Finding 1: The Ceremony Stack Is Massive

**Evidence:** A single feature-impl spawn receives:

| Layer | Content | Purpose |
|-------|---------|---------|
| SPAWN_CONTEXT.md | ~400 lines | Task, protocols, authority |
| worker-base skill | ~200 lines | Delegation rules, hard limits, beads tracking |
| feature-impl skill | ~800 lines | Phase guidance, TDD, validation |
| kb context | Variable | Prior decisions, constraints, investigations |
| Server context | ~20 lines | Local ports, commands |

**Total instructions:** 1400+ lines before the agent writes a single line of code.

**Source:** `pkg/spawn/context.go:54-404` (template), my own spawn context

**Significance:** This is designed for amnesia-resistant handoff between Claude instances. But when Dylan spawns an agent for a quick fix, the ceremony overhead dominates.

---

### Finding 2: Every Spawn Creates Artifacts

**Evidence:** Current artifact counts in orch-go:
- 735 investigation files
- 75 decision files
- 34 guide files
- Hundreds of workspace directories

The Jan 23 investigation found ~50% of investigations were unnecessary - implementation tasks that should have been commits with good messages.

**Source:** `ls .kb/investigations/*.md | wc -l` = 735

**Significance:** The system is drowning in artifacts. Each artifact represents overhead: creation time, maintenance burden, discovery friction. The `kb reflect` tool found "34 items need attention" on a routine startup.

---

### Finding 3: Workers Must Remember a Protocol Sequence

**Evidence:** From SPAWN_CONTEXT, a worker must:

1. **First 3 tool calls:** Report Phase: Planning via `bd comment`
2. **For investigations:** Run `kb create investigation`, report the path
3. **While working:** Update Status field, add findings progressively
4. **Before completing:** Run `kb quick decide/constrain/tried`
5. **Self-review:** Pass anti-pattern detection, security review, commit hygiene, test coverage, scope verification, discovered work checklist
6. **If web/ modified:** Capture visual verification screenshot
7. **Completion:** Report Phase: Complete, then `/exit`

Workers that skip steps get flagged as "unresponsive" or have their completion rejected.

**Source:** SPAWN_CONTEXT template lines 187-216, feature-impl skill self-review section

**Significance:** This is a 7-step protocol for every spawn. Each step exists because agents previously failed without it. But the aggregate cognitive load makes spawns feel like paperwork.

---

### Finding 4: The Friction Is Pre-Spawn and Post-Spawn

**Evidence:** From the Jan 30 worker observability investigation:

**Pre-spawn friction:**
- Multiple gates with slow feedback (status check, label check, dependency check)
- Each gate requires a fix-and-push cycle to resolve
- No preview of what will happen before commit

**During execution:**
- No progress visibility for headless agents
- Can't distinguish "working hard" from "stuck" from "dead"
- `orch status` doesn't show cross-project workers

**Post-spawn friction:**
- Status is stale - `in_progress` could mean active or zombie
- Completion detection requires spelunking through events.jsonl, workspace files, OpenCode API
- `orch complete` verification is complex and often fails

**Source:** `.kb/investigations/2026-01-30-inv-worker-observability-friction.md`

**Significance:** Even outside the agent, the human orchestrator faces ceremony. Releasing work to daemon isn't "tag and go" - it's a multi-step process with opaque feedback.

---

### Finding 5: The System Accumulated Defenses Against Past Failures

**Evidence:** Reading the SPAWN_CONTEXT and skills reveals why each rule exists:

| Rule | What Happened Without It |
|------|--------------------------|
| "First 3 actions report Phase" | Agents went silent, orchestrator couldn't track |
| "Create investigation file via kb create" | Agents made up filenames, files were inconsistent |
| "Run kb quick before completing" | Learnings were lost, same mistakes repeated |
| "Self-review: no hardcoded secrets" | Agents committed .env files |
| "Visual verification for web/" | Agents claimed "tests pass" but UI was broken |
| "Never run bd close" | Workers bypassed verification, tracking broke |
| "Never run git push" | Workers triggered deploys that disrupted production |

Each rule is a scar from a past failure. The system learned by accumulating constraints.

**Source:** SPAWN_CONTEXT template comments, skill guidance rationale sections

**Significance:** The ceremony isn't arbitrary - it's battle-tested. But the system never removes rules, only adds them. Technical debt in process form.

---

### Finding 6: Essential vs Accumulated Process

**Evidence:** Categorizing the ceremony:

**Essential (would break without):**
- Git commits with descriptive messages
- Tests passing
- Phase: Complete reporting (for daemon lifecycle)
- Beads issue tracking (for work discovery)

**Accumulated (defensive but removable for trusted actors):**
- First-3-actions phase reporting (unresponsive detection)
- Scope enumeration (section blindness prevention)
- Self-review checklists (anti-pattern detection)
- Harm assessment (ethics gate)
- `kb quick` before completing (externalization)
- Visual verification for web/ (UI regression prevention)
- SYNTHESIS.md (handoff document)
- Investigation file creation for non-investigation skills

**Source:** Analysis of failure modes and recovery needs

**Significance:** About 70% of the ceremony is defensive. For autonomous daemon workers, this defense is valuable. For Dylan spawning a quick fix, it's overhead.

---

### Finding 7: When It Crossed from Helpful to Work

**Evidence:** Timeline reconstruction from investigations and git history:

| Period | State | Evidence |
|--------|-------|----------|
| Oct 2025 | Helpful | `orch spawn` just worked, minimal ceremony |
| Nov 2025 | Growing | D.E.K.N. format introduced, investigation templates added |
| Dec 2025 | Complex | Skills system, worker-base, completion verification |
| Jan 2026 | Heavy | 17+ investigations in 36 hours, 735 total investigations |
| Feb 2026 | Job | 34 kb reflect items, 1400+ lines of spawn context |

The crossing point was **late December 2025** when the skill system and completion verification were formalized. Before that, spawns were lightweight. After, every spawn required reading and following extensive protocols.

**Source:** Investigation timestamps, git history, skill file dates

**Significance:** The system went from "helpful automation" to "process compliance" over ~2 months. The intent was good (reliability, handoff quality), but the accumulation was never pruned.

---

## Synthesis

**Core Issue:** Orch became a system optimized for AI workers, not a tool for humans.

The ceremony serves AI agents who have:
- No memory between sessions (need SYNTHESIS.md, investigation files)
- No implicit understanding of past failures (need explicit rules)
- No judgment about when to skip steps (need mandatory checklists)

But Dylan has:
- Memory across sessions
- Implicit understanding of context
- Judgment about what's necessary

**The weight is real:**
- 1400+ lines of instructions per spawn
- 7-step completion protocol
- 735 accumulated investigation files
- 34 items needing attention on startup

**What would "fun" look like:**

```bash
# Current (job-like)
bd create "fix login bug" --type bug -l triage:ready
orch daemon run  # wait for spawn
# agent runs through 1400 lines of protocol
# check `bd show` for completion
# run `orch complete` with verification
# deal with rejection if any step skipped

# Dream (fun)
claude "fix the login bug"
# it fixes it
# done
```

---

## Recommendations

### 1. Trust Tiers (Recommended)

Create two paths:
- **Daemon path (strict):** For autonomous agents, keep all ceremony
- **Interactive path (light):** For Dylan-initiated spawns, minimal ceremony

Implementation: `orch spawn --interactive` that skips defensive ceremony, trusts the human to verify.

### 2. Ceremony Sunset Review

Every 3 months, review ceremony with question: "Has this rule caught a real problem in the last month?"

Rules with no recent catches get demoted to "optional" or removed.

### 3. Zero-Ceremony Escape Hatch

Raw Claude without orch:
```bash
claude -p orch-go "fix the login bug in src/auth.go"
```

No beads, no SPAWN_CONTEXT, no phases. Just fix it.

### 4. Artifact Archival

Move old investigations to `.kb/investigations/archive/`. The 735 files are discovery friction.

### 5. Simplify Completion

Instead of `bd comment Phase: Complete` + `orch complete` + verification, make completion implicit:
- Agent commits with conventional message
- Hook detects and closes issue
- No manual reporting

---

## Answer to Investigation Questions

**1) What must users do/remember to use orch correctly?**

- Pre-spawn: Create issue, add triage:ready label, resolve dependencies
- Spawn: Choose skill, model, flags
- During: Monitor (if caring about progress)
- Post-spawn: Check completion, run `orch complete`, deal with verification failures
- Meta: Run `kb reflect`, action synthesis opportunities, maintain decisions/guides

**2) Essential vs accumulated process?**

- Essential (~30%): Git commits, tests passing, beads tracking, phase completion reporting
- Accumulated (~70%): Self-review checklists, scope enumeration, harm assessment, kb quick externalization, visual verification, SYNTHESIS.md, investigation file creation

**3) When did it cross from helpful to work?**

Late December 2025, when skills system and completion verification formalized. The accumulation was never pruned.

**4) What would fun look like instead?**

"Tell AI what to do, it does it, done."

No mandatory artifacts. No protocol sequences. No verification ceremonies. Trust the human to inspect results. Use ceremony only for autonomous/unattended work where reliability matters more than speed.

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - SPAWN_CONTEXT template (400+ lines)
- `.kb/investigations/2026-01-23-inv-so-many-investigations-created-root.md` - Investigation overhead analysis
- `.kb/investigations/2026-01-31-inv-investigation-churn-patterns-lineage.md` - Churn analysis
- `.kb/investigations/2026-01-30-inv-worker-observability-friction.md` - Pre/post spawn friction

**Counts:**
- Investigations: 735
- Decisions: 75
- Guides: 34
- SPAWN_CONTEXT template: 400+ lines
- Feature-impl skill content: ~800 lines

---

## Investigation History

**2026-02-04 18:35:** Started investigation
- Read prior investigations on overhead, churn, friction
- Analyzed SPAWN_CONTEXT template and skill content
- Counted artifacts

**2026-02-04 18:55:** Synthesized findings
- Identified ceremony stack as primary burden
- Distinguished essential from accumulated process
- Traced crossing point to Dec 2025
- Formulated "trust tiers" recommendation

**2026-02-04 19:00:** Completed
- Status: Complete
