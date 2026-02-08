## Summary (D.E.K.N.)

**Delta:** Interactive orchestrators serve three legitimate functions that daemon cannot replicate: (1) goal refinement through conversation, (2) real-time frame correction, and (3) synthesis of worker results across a focus block—they are not purely compensation for daemon gaps.

**Evidence:** 1,468 total spawns, 382 daemon-driven, 21 orchestrator sessions. Daemon handles batch work well (triage:ready → auto-spawn). But goal refinement ("work on orch-go" → "complete daemon reliability epic") and synthesis require conversational interaction that daemon lacks by design.

**Knowledge:** The hypothesis conflated two valid observations: (a) daemon underutilization (26% daemon-driven) and (b) interactive orchestrators serving no purpose. Finding: the first is true, the second is false.

**Next:** Increase daemon utilization through better triage discipline, but preserve interactive orchestrators for their legitimate functions. Create separate issues for daemon underutilization patterns.

---

# Investigation: Interactive Orchestrators as Compensation Pattern

**Question:** Are interactive orchestrators compensation for daemon/system gaps, or do they serve legitimate coordination functions?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Daemon handles batch work effectively when used

**Evidence:** 
- 382 daemon.spawn events in events.jsonl (26% of 1,468 total spawns)
- Daemon correctly infers skill from issue type, respects dependencies, handles rate limiting
- pkg/daemon/daemon.go shows sophisticated queue management: priority sorting, blocking dependency checks, rate limiting (20/hour), worker pool management

**Source:** 
- `~/.orch/events.jsonl` - grep for "daemon.spawn" vs "session.spawned"
- `pkg/daemon/daemon.go` - Lines 160-251 (NextIssue logic), 351-421 (Preview)

**Significance:** The daemon is capable, but underutilized. This is a discipline problem (not using triage:ready workflow) rather than a capability gap.

---

### Finding 2: Interactive orchestrators serve three functions daemon cannot

**Evidence:**
From meta-orchestrator skill (SKILL.md lines 69-79):
1. **Goal refinement** - "work on orch-go" → "complete daemon reliability epic" requires conversation
2. **Real-time frame correction** - catching when orchestrators "do spawnable work" 
3. **Post-mortem perspective** - seeing patterns orchestrators can't see about themselves

The meta-orchestrator skill explicitly states: "Meta-orchestrator is a thinking partner with Dylan, not another autonomous execution layer" (line 61).

**Source:**
- `~/.claude/skills/meta/meta-orchestrator/SKILL.md` - Lines 59-79 (The Conversational Frame section)
- Orchestrator skill shows synthesis as core responsibility that requires context

**Significance:** These functions are inherently conversational. Daemon automates "what to work on next" but cannot automate "what should we focus on" or "what does this pattern mean."

---

### Finding 3: The 26% daemon utilization reveals friction in triage workflow

**Evidence:**
- 1,468 total spawns, 382 daemon-driven = 26% daemon utilization
- --bypass-triage flag added to create "friction" (cmd/orch/spawn_cmd.go line 177)
- Comment: "This creates friction to encourage the daemon-driven workflow" (line 1643)
- 151 untracked spawns (no_track=true) - ad-hoc work bypassing both daemon and issue tracking

**Source:**
- `~/.orch/events.jsonl` - spawn event counts
- `cmd/orch/spawn_cmd.go` - Lines 69-74, 177, 1642-1669

**Significance:** The system has mechanisms to encourage daemon use, but they're not working at target levels. The question "why do 74% of spawns bypass daemon" deserves its own investigation.

---

### Finding 4: Orchestrator sessions (21 total) are distinct from manual worker spawns

**Evidence:**
- 21 orchestrator sessions in events.jsonl (skill="orchestrator")
- These are not "manual spawns bypassing daemon" - they're a different entity type
- Orchestrators spawn workers (who ARE tracked in beads), monitor them, and synthesize results
- meta-orchestrator skill creates orchestrator sessions, not workers

**Source:**
- `~/.orch/events.jsonl` - grep for skill="orchestrator"
- Meta-orchestrator skill shows orchestrator session lifecycle

**Significance:** The "94% manual spawn" framing may have conflated orchestrator sessions with manual worker spawns. Orchestrators are a coordination layer, not a workaround.

---

### Finding 5: What would break without interactive orchestrators

**Evidence:** Analyzing orchestrator responsibilities from skill:

| Function | Can Daemon Handle? | Why/Why Not |
|----------|-------------------|-------------|
| Cross-agent synthesis | ❌ No | Requires reading multiple worker outputs, connecting insights |
| Knowledge integration | ❌ No | Deciding what becomes a decision record requires judgment |
| Meta-level evaluation | ❌ No | Evaluating orchestration system requires external perspective |
| Work prioritization | ⚠️ Partial | Daemon picks next issue by priority; focus alignment needs conversation |
| Interactive synthesis | ❌ No | Conversational analysis with Dylan is inherently interactive |
| Conflict resolution | ❌ No | Reconciling contradictory findings requires reasoning |

**Source:**
- Orchestrator skill "Orchestrator Core Responsibilities" section
- Comparison with daemon.go capabilities

**Significance:** Removing interactive orchestrators would break synthesis, knowledge integration, and focus alignment. These are not daemon gaps—they're categorically different from queue processing.

---

## Synthesis

**Key Insights:**

1. **Two valid observations were conflated** - Daemon underutilization (26% vs target) and interactive orchestrator value are separate questions. The investigation hypothesis combined them into "interactive orchestrators as compensation."

2. **Daemon automates dispatch, not direction** - Daemon answers "what spawnable issue is next." Interactive orchestrators answer "what should we focus on" and "what does this mean." These are complementary, not competing.

3. **The proposed evolution misses a layer** - "Dylan → Meta-orch → Issues → Daemon → Workers" removes the orchestrator→synthesis step. Workers produce artifacts; someone must synthesize them. That's the orchestrator's job.

**Answer to Investigation Question:**

Interactive orchestrators are NOT primarily compensation for daemon gaps. They serve three legitimate functions:
1. Goal refinement (converting vague strategic intent to actionable orchestrator goals)
2. Real-time frame correction (catching level drops)
3. Synthesis (combining worker results into decisions/knowledge)

The daemon underutilization (26% vs target) IS a real problem, but the solution is better triage discipline, not removing interactive orchestrators.

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon spawn count vs total spawn count (verified: grep on events.jsonl)
- ✅ Orchestrator session count (verified: grep for skill="orchestrator")
- ✅ Daemon capability (verified: read pkg/daemon/daemon.go source)

**What's untested:**

- ⚠️ Whether increased daemon use would actually reduce need for interactive orchestrators (hypothesis, not measured)
- ⚠️ Whether goal refinement could be front-loaded to issue creation (proposed but not validated)
- ⚠️ Whether synthesis could be automated (would require testing with actual worker outputs)

**What would change this:**

- Finding that synthesis adds no value (workers could self-synthesize)
- Finding that goal refinement happens once and never needs adjustment
- Finding that frame correction isn't actually needed (orchestrators don't drift)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Preserve interactive orchestrators, increase daemon utilization through discipline** - Address the 26% daemon utilization separately from orchestrator value.

**Why this approach:**
- Interactive orchestrators serve legitimate functions that daemon can't replicate
- Daemon underutilization is a separate problem (triage discipline, bypass-triage friction)
- The proposed "remove interactive orchestrators" would break synthesis workflow

**Trade-offs accepted:**
- Maintaining two coordination layers (meta-orchestrator + orchestrator + daemon)
- Complexity of knowing when to use each

**Implementation sequence:**
1. Create issue for "daemon underutilization investigation" - why 74% bypass daemon
2. Strengthen triage:ready workflow guidance in orchestrator skill
3. Consider metrics/alerts for daemon utilization percentage

### Alternative Approaches Considered

**Option B: Merge orchestrator into meta-orchestrator**
- **Pros:** One fewer layer
- **Cons:** Meta-orchestrator skill says "meta-orchestrators spawn orchestrators, not workers" - the separation is intentional
- **When to use instead:** If orchestrator sessions consistently provide no value beyond what daemon provides

**Option C: Automate synthesis**
- **Pros:** Could eliminate need for interactive orchestrator synthesis
- **Cons:** Synthesis requires judgment about what matters, what connects, what to capture
- **When to use instead:** If synthesis becomes formulaic (just extract/combine, no judgment)

---

## References

**Files Examined:**
- `~/.orch/events.jsonl` - Spawn event history
- `pkg/daemon/daemon.go` - Daemon implementation
- `~/.claude/skills/meta/meta-orchestrator/SKILL.md` - Meta-orchestrator responsibilities
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator responsibilities
- `cmd/orch/spawn_cmd.go` - Bypass-triage friction implementation

**Commands Run:**
```bash
# Count spawns by type
cat ~/.orch/events.jsonl | grep '"type":"session.spawned"' | wc -l  # 1468
cat ~/.orch/events.jsonl | grep '"type":"daemon.spawn"' | wc -l     # 382
cat ~/.orch/events.jsonl | grep '"skill":"orchestrator"' | wc -l    # 21
cat ~/.orch/events.jsonl | grep '"no_track":true' | wc -l           # 151
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-11-26-orchestrator-autonomy-pattern.md`
- **Skill:** `~/.claude/skills/meta/orchestrator/SKILL.md`

---

## Investigation History

**2026-01-06 08:10:** Investigation started
- Initial question: Are interactive orchestrators compensation for daemon gaps?
- Context: Hypothesis from meta-orchestrator session suggesting 94% manual spawn rate

**2026-01-06 08:30:** Key finding - spawn ratio is 26% daemon, not 6%
- Recalculated using events.jsonl grep counts
- Original 94% figure may have been from different measurement

**2026-01-06 08:45:** Investigation completed
- Status: Complete
- Key outcome: Interactive orchestrators serve legitimate functions (goal refinement, synthesis, frame correction) that daemon cannot replicate. Daemon underutilization is a separate problem.
