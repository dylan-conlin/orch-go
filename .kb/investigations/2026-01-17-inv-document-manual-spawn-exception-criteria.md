<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Manual spawn exception criteria are scattered across 4 documents; synthesized into 5 categories: (1) Escape Hatch for critical infrastructure, (2) Interactive/Strategic skills, (3) Urgent items, (4) Complex/Ambiguous scope, (5) Skill inference override.

**Evidence:** Analyzed spawn.md, daemon.md, resilient-infrastructure-patterns.md, and 60% manual spawn investigation; verified spawn_cmd.go shows --bypass-triage friction but no exception criteria documentation in error message.

**Knowledge:** The "escape hatch" pattern (critical infrastructure work) is the most strategically important exception but is undocumented in spawn.md/daemon.md where orchestrators look; skill-based exceptions (design-session 100% manual) are implicit from data but not codified.

**Next:** Recommend updating spawn.md "Triage Bypass" section with comprehensive exception criteria matrix; this is the authoritative reference orchestrators consult.

**Promote to Decision:** recommend-yes - Establishes criteria that gates daemon autonomy; orchestrators need clear policy to avoid rationalizing "urgent" as default.

---

# Investigation: Document Manual Spawn Exception Criteria

**Question:** What are the complete, authoritative criteria for when manual spawn (with --bypass-triage) is appropriate vs when issues should flow through daemon?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Worker agent og-inv-document-manual-spawn-17jan-5269
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A - Creates new guidance, doesn't patch existing decision
**Extracted-From:** Synthesized from spawn.md, daemon.md, resilient-infrastructure-patterns.md, and prior investigations

---

## Findings

### Finding 1: Current Exception Criteria are Scattered Across 4 Documents

**Evidence:** Manual spawn exception criteria appear in fragments across:

1. **spawn.md:79-84** - 3 criteria:
   - Single urgent item requiring immediate attention
   - Complex/ambiguous task needing custom context
   - Skill selection requires orchestrator judgment

2. **daemon.md:511-520** - Same 3 criteria as spawn.md (duplicated)

3. **resilient-infrastructure-patterns.md:64-95** - Escape hatch pattern:
   - Building/fixing infrastructure the primary path (daemon) depends on
   - When OpenCode server can fail while agents are fixing it
   - Example: `orch spawn --bypass-triage --mode claude --model opus --tmux`

4. **60% manual spawn investigation** - Implicit skill-based criteria:
   - design-session: 100% manual (inherently interactive)
   - investigation: 90% manual (requires orchestrator framing)
   - systematic-debugging: 76% manual (often urgent)

**Source:**
- `.kb/guides/spawn.md:79-84`
- `.kb/guides/daemon.md:511-520`
- `.kb/guides/resilient-infrastructure-patterns.md:64-95`
- `.kb/investigations/2026-01-07-inv-investigate-60-manual-spawns-vs.md:68-78`

**Significance:** An orchestrator looking for exception criteria must read 4 documents to get the full picture. The escape hatch pattern (most strategically important) isn't in spawn.md or daemon.md where orchestrators naturally look for spawn guidance.

---

### Finding 2: Escape Hatch Pattern is the Most Strategically Important Exception

**Evidence:** The resilient-infrastructure-patterns guide documents a Jan 10, 2026 crisis where:
- OpenCode server crashed 3 times in 1 hour
- Each crash killed all active agents
- Agents were building the fixes for crash recovery
- Death spiral: can't fix crashes because crashes kill the fixes

**Resolution:** Used manual spawn with `--mode claude --tmux`:
```bash
orch spawn --bypass-triage --mode claude --model opus --tmux \
  feature-impl "P0: Implement orch doctor" --issue orch-go-uyveu
```

**Key insight from guide:** "Escape hatch broke the death spiral. The 'old spawn path' wasn't legacy - it was disaster recovery."

**Characteristics of escape hatch:**
1. **Independent:** Doesn't depend on what can fail (claude CLI ≠ opencode server)
2. **Visible:** Tmux window shows progress, can intervene
3. **Capable:** opus model for quality, full skill context

**Source:** `.kb/guides/resilient-infrastructure-patterns.md:266-292`

**Significance:** This exception category is categorically different from "urgent" or "complex". It's about system resilience, not task characteristics. Without this escape hatch, critical infrastructure work can enter death spirals.

---

### Finding 3: Skill-Based Exceptions are Data-Driven but Undocumented

**Evidence:** From 60% manual spawn investigation, skill distribution shows inherent manual spawn patterns:

| Skill | Manual % | Why |
|-------|----------|-----|
| design-session | 100% | Inherently interactive, requires real-time collaboration |
| orchestrator/meta-orchestrator | 100% | Coordination layer, not task-driven |
| investigation | 90% | Requires orchestrator framing the question |
| systematic-debugging | 76% | Often urgent, needs immediate attention |
| feature-impl | 37% | Mixed - some legitimate, some could be daemon |
| architect | 26% | Mostly daemon, occasional judgment needed |

The investigation concluded: "Most manual spawns fit documented exception categories: 'single urgent item', 'complex/ambiguous', or 'orchestrator judgment on skill/context needed'."

**Source:** `.kb/investigations/2026-01-07-inv-investigate-60-manual-spawns-vs.md:68-78, 119`

**Significance:** Some skills are NEVER suitable for daemon workflow (design-session, orchestrator). This should be explicit policy, not emergent behavior. Removing bypass friction for these skills would acknowledge their inherent nature.

---

### Finding 4: Exception Criteria Error Message Lacks Guidance

**Evidence:** When spawning without --bypass-triage, spawn_cmd.go shows:

```
╔═══════════════════════════════════════════════════════════════════════════╗
║                                                                           ║
║  Manual spawn requires --bypass-triage flag.                              ║
║                                                                           ║
║  The daemon-driven workflow is preferred:                                 ║
║    1. bd create "task description" --type <type> -l triage:ready         ║
║    2. orch daemon run (or let launchd-managed daemon handle it)          ║
║                                                                           ║
║  To proceed with manual spawn, add --bypass-triage:                       ║
║    orch spawn --bypass-triage %s "%s"                                    ║
║                                                                           ║
╚═══════════════════════════════════════════════════════════════════════════╝
```

**What's missing:** The error message tells you HOW to bypass but not WHEN it's appropriate. An orchestrator seeing this has no guidance on whether their case is a legitimate exception.

**Source:** `cmd/orch/spawn_cmd.go:2150-2173`

**Significance:** The friction is in place but lacks the discriminating guidance. This may lead to:
- Over-use: "urgent" becomes rationalization for everything
- Under-use: Legitimate escape hatches not taken because unclear

---

### Finding 5: Identify-Orchestrator-Value Investigation Requested This Documentation

**Evidence:** Investigation "Identify Orchestrator Value Add vs Routing" (2026-01-17) explicitly listed as action item:

> **Next:** Create issues for: ... (4) document manual spawn exception criteria.

The investigation found:
- "Exception criteria (when manual spawn is legitimate) must be clear - otherwise 'urgent' becomes default rationalization"
- "Clarify when manual spawn is actually needed (urgent, complex, interactive synthesis) vs workflow workaround"

**Source:** `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md:14, 320-321`

**Significance:** This investigation was explicitly requested as a deliverable from prior strategic analysis. The need is documented and the gap is recognized.

---

## Synthesis

**Key Insights:**

1. **Five Categories of Legitimate Manual Spawn Exceptions:**
   - **Escape Hatch (Critical Infrastructure):** Building/fixing infrastructure that daemon depends on; requires --mode claude --tmux to survive crashes
   - **Interactive/Strategic Skills:** design-session, orchestrator, meta-orchestrator - inherently require real-time collaboration
   - **Urgent Items:** Single issue requiring immediate attention (can't wait for daemon poll cycle)
   - **Complex/Ambiguous Scope:** Issue needs orchestrator framing, custom context, or skill clarification
   - **Skill Inference Override:** When daemon would infer wrong skill from issue type (e.g., "task" that actually needs feature-impl)

2. **Escape Hatch is a Distinct Category** - The other four exceptions are about task characteristics (urgent, complex, interactive). Escape hatch is about system resilience - it applies when the infrastructure itself can fail. This is strategically more important because it prevents death spirals.

3. **Documentation Gap is in the Wrong Place** - The escape hatch pattern is documented in resilient-infrastructure-patterns.md, but orchestrators naturally look in spawn.md for spawn guidance. The criteria need to be where orchestrators look.

4. **Skill-Based Exceptions Should Be Explicit** - design-session and orchestrator skills being 100% manual isn't documented policy - it's emergent data. Making this explicit would:
   - Acknowledge their inherent nature
   - Optionally remove bypass friction for these skills
   - Prevent orchestrators from feeling they're "breaking rules" when manual spawning them

**Answer to Investigation Question:**

The complete criteria for manual spawn exceptions are:

| Category | Criteria | Example | Command Variant |
|----------|----------|---------|-----------------|
| **Escape Hatch** | Building/fixing infrastructure that daemon depends on | OpenCode observability, crash recovery | `--mode claude --tmux` |
| **Interactive Skills** | Skills inherently requiring real-time collaboration | design-session, orchestrator | Standard |
| **Urgent** | Single item can't wait for daemon poll (60s cycle) | Production bug, time-sensitive fix | Standard |
| **Complex/Ambiguous** | Needs orchestrator framing, custom context | Vague requirements, multi-step exploration | Standard |
| **Skill Override** | Daemon would infer wrong skill from issue type | "task" that needs feature-impl | Standard |

**NOT legitimate exceptions (workflow workarounds):**
- Triage discipline gap (should label triage:ready, not manual spawn)
- Spawn reliability issues (fix daemon, don't work around)
- Batch processing (daemon handles this)
- "Easier than triage workflow" (this is the friction working)

---

## Structured Uncertainty

**What's tested:**

- ✅ spawn.md criteria documented (verified: read lines 79-84)
- ✅ daemon.md criteria match spawn.md (verified: read lines 511-520)
- ✅ Escape hatch pattern documented separately (verified: resilient-infrastructure-patterns.md)
- ✅ Skill distribution from events data (verified: read 60% manual spawn investigation)
- ✅ Error message lacks criteria guidance (verified: read spawn_cmd.go:2150-2173)

**What's untested:**

- ⚠️ Whether adding criteria to error message would change behavior (behavioral hypothesis)
- ⚠️ Whether removing bypass friction for interactive skills would be accepted (policy decision)
- ⚠️ How often orchestrators currently use each exception category (would need spawn event analysis with reason codes)

**What would change this:**

- If daemon poll interval reduced to <10s, "urgent" exception becomes less common
- If skill-based routing improved, "skill override" exception becomes less common
- If OpenCode server becomes highly reliable, "escape hatch" exception becomes rare
- If triage workflow becomes frictionless, more issues flow through daemon

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable documentation update.

### Recommended Approach ⭐

**Update spawn.md "Triage Bypass" section with comprehensive exception criteria matrix** - This is the authoritative reference orchestrators consult when deciding whether to manual spawn.

**Why this approach:**
- spawn.md is where orchestrators look for spawn guidance (Finding 1)
- Criteria are currently scattered, need consolidation (Finding 1)
- Error message lacks guidance, spawn.md is the reference doc (Finding 4)
- Escape hatch pattern is strategically critical but undiscovered in spawn.md (Finding 2)

**Trade-offs accepted:**
- Some duplication with daemon.md (acceptable: daemon.md references spawn.md)
- Doesn't change error message text (separate task if desired)
- Doesn't change code to auto-exempt interactive skills (separate task if desired)

**Implementation sequence:**
1. Replace spawn.md:79-84 "Manual spawn is for exceptions only" section with comprehensive matrix
2. Add escape hatch category with example command variant (`--mode claude --tmux`)
3. Add "NOT legitimate exceptions" section to prevent rationalization
4. Update daemon.md:511-520 to reference spawn.md instead of duplicating

### Alternative Approaches Considered

**Option B: Update error message in spawn_cmd.go**
- **Pros:** Guidance at point of friction; orchestrators see criteria when blocked
- **Cons:** Error messages have space constraints; full matrix won't fit; better as reference than inline
- **When to use instead:** If metrics show orchestrators not reading spawn.md

**Option C: Create separate decision document**
- **Pros:** Formal decision record for policy
- **Cons:** Another place to look; spawn.md is already the authoritative guide
- **When to use instead:** If criteria become contentious or require formal approval

**Rationale for recommendation:** spawn.md is the existing authoritative reference. Consolidating criteria there follows the "single source of truth" principle. The escape hatch pattern is too important to be buried in a separate patterns guide.

---

### Implementation Details

**What to implement first:**
- Update spawn.md "Triage Bypass" section with 5-category matrix
- Add escape hatch example with `--mode claude --tmux`
- Add "NOT legitimate exceptions" list

**Things to watch out for:**
- ⚠️ "Urgent" is the most abused exception - need clear definition (can't wait 60s for daemon poll)
- ⚠️ Escape hatch should be rare - only when infrastructure can fail, not routine caution
- ⚠️ Don't remove bypass friction entirely for interactive skills without explicit decision

**Success criteria:**
- ✅ spawn.md contains complete exception criteria matrix
- ✅ Escape hatch pattern documented in spawn.md (not just patterns guide)
- ✅ "NOT legitimate exceptions" provides clear anti-patterns
- ✅ daemon.md references spawn.md instead of duplicating

---

## References

**Files Examined:**
- `.kb/guides/spawn.md:71-94` - Current exception criteria section
- `.kb/guides/daemon.md:511-520` - Duplicate exception criteria
- `.kb/guides/resilient-infrastructure-patterns.md` - Escape hatch pattern documentation
- `.kb/investigations/2026-01-07-inv-investigate-60-manual-spawns-vs.md` - Skill distribution data
- `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md` - Request for this documentation
- `cmd/orch/spawn_cmd.go:2150-2173` - Error message text

**Commands Run:**
```bash
# Search for existing exception criteria
rg "bypass-triage|manual spawn" .kb/guides/

# Verify skill distribution data
cat .kb/investigations/2026-01-07-inv-investigate-60-manual-spawns-vs.md | grep -A10 "Manual spawns by skill"

# Check error message implementation
rg "Manual spawn requires" cmd/orch/
```

**Related Artifacts:**
- **Guide:** `.kb/guides/spawn.md` - Target for documentation update
- **Guide:** `.kb/guides/daemon.md` - References spawn for exception criteria
- **Guide:** `.kb/guides/resilient-infrastructure-patterns.md` - Escape hatch pattern
- **Investigation:** `.kb/investigations/2026-01-07-inv-investigate-60-manual-spawns-vs.md` - Skill data source
- **Investigation:** `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md` - Requested this work

---

## Investigation History

**[2026-01-17 15:30]:** Investigation started
- Initial question: What are complete criteria for manual spawn exceptions?
- Context: Requested by orch-go-yz17r to align orchestrator behavior with daemon autonomy goals

**[2026-01-17 15:35]:** Read spawn.md and daemon.md
- Found 3 exception categories documented in both
- Noted duplication between guides

**[2026-01-17 15:40]:** Read resilient-infrastructure-patterns.md
- Found escape hatch pattern as strategically important exception
- Noted this isn't documented in spawn.md where orchestrators look

**[2026-01-17 15:45]:** Read 60% manual spawn investigation
- Found skill distribution data showing inherent manual spawn patterns
- Noted design-session and orchestrator are 100% manual

**[2026-01-17 15:50]:** Verified spawn_cmd.go error message
- Confirmed friction exists but guidance is missing
- Error tells HOW to bypass, not WHEN appropriate

**[2026-01-17 15:55]:** Investigation completed
- Synthesized 5 exception categories
- Recommended updating spawn.md as authoritative reference
- Status: Complete
