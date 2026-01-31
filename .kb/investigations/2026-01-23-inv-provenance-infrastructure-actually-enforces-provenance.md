<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Provenance has NO infrastructure enforcing it - it's a principle without gates. The closest are test evidence and git diff gates, which verify narrow forms of "tracing to external reality" but don't detect closed-loop reasoning.

**Evidence:** Audited ~/.kb/principles.md (Provenance defined), pkg/verify/check.go (11 gates, none for Provenance), OpenCode plugins (Provenance Tracker designed Jan 8 but not built), skillc load_bearing (provenance is metadata, not enforcement).

**Knowledge:** Provenance is the most foundational principle but the least enforced. Current gates verify *specific* external traces (tests passed, visual verified, git diff matches) but can't detect the general failure mode: conclusions formed by reasoning alone without evidence gathering.

**Next:** Recommend implementing Provenance Tracker plugin from Jan 8 investigation design. This would observe evidence-gathering patterns (read/grep/bash) and flag sessions that conclude without gathering evidence.

**Promote to Decision:** Actioned - gap analysis, enforcement patterns in principles.md

---

# Investigation: Provenance as Infrastructure - What Actually Enforces It

**Question:** Is Provenance (conclusions must trace outside the conversation) also infrastructure? What actually enforces it? What gates or tooling could make violations detectable?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Worker agent (orch-go-usdaq)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Provenance is Defined as the Foundational Principle

**Evidence:** From `~/.kb/principles.md`:

> **Provenance (Foundational)**
>
> Every conclusion must trace to something outside the conversation.
>
> **The test:** "Can someone who wasn't here verify this?"
>
> **What this rejects:**
> - "Claude confirmed this" (closed loop - AI agreeing with itself)
> - "This feels significant" (feeling is not evidence)
> - "I wrote it down" (writing preserves the claim, not the proof)
>
> **Why this is foundational:** Session Amnesia tells you to externalize. Provenance tells you *how* - anchored to something that exists whether or not you talked about it.

**Source:** `~/.kb/principles.md:19-46`

**Significance:** Provenance is explicitly the most foundational principle ("Why this is foundational"). Other principles derive from it. Yet as a principle, it exists as documentation agents should follow - not infrastructure that enforces behavior.

---

### Finding 2: Verification Gates Focus on Specific Evidence Types, Not General Provenance

**Evidence:** `pkg/verify/check.go` defines 11 verification gates for `orch complete`:

| Gate | What It Checks | Provenance-Related? |
|------|---------------|---------------------|
| `GatePhaseComplete` | Phase: Complete reported | No - structure |
| `GateSynthesis` | SYNTHESIS.md exists | No - structure |
| `GateHandoffContent` | SYNTHESIS.md has content | No - structure |
| `GateConstraint` | Skill constraints pass | No - compliance |
| `GatePhaseGate` | Required phases reported | No - workflow |
| `GateSkillOutput` | Required outputs exist | No - structure |
| `GateVisualVerify` | Screenshot for web/ changes | **Partial** - specific external evidence |
| `GateTestEvidence` | Test execution output | **Partial** - specific external evidence |
| `GateGitDiff` | Git diff matches claims | **Partial** - reality check |
| `GateBuild` | Project builds | **Partial** - external validation |
| `GateDecisionPatchLimit` | Patch count < threshold | No - governance |

**Source:** `pkg/verify/check.go:13-25`

**Significance:** The "Partial" gates verify *specific* types of external anchoring:
- Test evidence: "Did tests actually pass?" (external: test runner output)
- Visual verification: "Did you see it working?" (external: screenshot)
- Git diff: "Does reality match claims?" (external: filesystem)
- Build: "Does code compile?" (external: compiler)

But none detect the *general* Provenance failure mode: conclusions formed by reasoning without evidence gathering.

---

### Finding 3: Provenance Tracker Plugin Was Designed But Not Built

**Evidence:** The Jan 8, 2026 investigation `2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md` designed a Provenance Tracker plugin:

```typescript
// Track evidence-gathering per session
const evidenceGathered = new Map<string, Set<string>>()

"tool.execute.after": async (input, output) => {
  const evidenceTools = ["read", "grep", "glob", "bash"]
  if (!evidenceTools.includes(input.tool)) return

  // Track that evidence was gathered
  const evidence = evidenceGathered.get(input.sessionID) || new Set()
  evidence.add(`${input.tool}:${output.args?.filePath || ...}`)
  evidenceGathered.set(input.sessionID, evidence)
}

// On session idle, check if conclusions exist without evidence
"event": async ({ event }) => {
  if (event.type !== "session.idle") return

  const evidence = evidenceGathered.get(sessionId)
  if (!evidence || evidence.size === 0) {
    // Session made conclusions without gathering evidence
    // Log warning
  }
}
```

The investigation's Next step was: "Implement 2-3 high-value plugins for principle mechanization: 'Coherence Over Patches' detector, 'Gate Over Remind' enforcement via beads, and **'Provenance' tracking for claims without evidence**."

**Source:** `.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md:326-366`

**Significance:** The infrastructure *capability* exists (OpenCode plugins can track tool calls and inject context). The design was done. But it was never built - Provenance remains unenforced.

---

### Finding 4: Skillc load_bearing Has "provenance" Field But It's Documentation Not Enforcement

**Evidence:** The skillc load_bearing data model includes a `provenance` field:

```yaml
load_bearing:
  - pattern: "ABSOLUTE DELEGATION RULE"
    provenance: "2025-11 orchestrator doing investigations led to 3-day derailment"
    severity: error
```

But this `provenance` field is metadata - it documents *why* the pattern matters (friction story). It's not enforcement of the Provenance *principle*. The skillc check verifies the pattern string exists; it doesn't verify that agents' conclusions trace to external evidence.

**Source:** `.kb/investigations/2026-01-08-inv-design-data-model-load-bearing.md`

**Significance:** Naming confusion: "provenance" in load_bearing means "origin story for this guidance" not "enforcement of the Provenance principle."

---

### Finding 5: Friction Capture Plugin Addresses Gate Over Remind, Not Provenance

**Evidence:** The friction-capture.ts plugin prompts on session idle:

> "Did you encounter any friction during this session?... capture it NOW (you'll rationalize it away later)"

It prompts for `kb quick` commands to capture decisions/constraints/failures.

**Source:** `~/.config/opencode/plugin/friction-capture.ts`

**Significance:** This plugin enforces *capture* (Session Amnesia / Gate Over Remind) but not *provenance*. An agent could respond with friction that's entirely fabricated or reasoned-into-existence. The prompt doesn't ask "What external evidence supports this claim?"

---

## Synthesis

**Key Insights:**

1. **The Foundational Principle Has No Foundational Enforcement** - Provenance is explicitly "why other principles derive from this" but has ZERO infrastructure enforcing it. There's no gate that says "this conclusion was formed from external evidence." The current gates verify *narrow* external traces (test output, screenshots, git diff) but not the *general* principle.

2. **The Gap is Observable in Practice** - Without Provenance enforcement, agents can:
   - Form conclusions via reasoning alone ("this feels significant")
   - Validate conclusions by asking another AI ("Claude confirmed this")
   - Write claims without evidence gathering (read/grep/bash)

   The existing investigation file template tries to address this with "Structured Uncertainty" sections (tested vs untested), but it's documentation, not a gate.

3. **Infrastructure Exists But Plugin Not Built** - OpenCode's plugin system can track evidence-gathering patterns (tool.execute.after for read/grep/glob/bash) and flag sessions that conclude without gathering evidence. The design exists from Jan 8. It was never implemented.

4. **Gate Over Remind Addresses When, Not What** - Current enforcement gates address *when* to capture (session end, session idle) and *that* certain artifacts exist. They don't address *quality* of what's captured - whether claims trace to external reality or are reasoning loops.

**Answer to Investigation Question:**

**Is Provenance also infrastructure?** No. Currently it's principle-only.

**What actually enforces it?** Nothing directly. The closest are:
- Test evidence gate (narrow: test runner output)
- Visual verification gate (narrow: screenshots)
- Git diff gate (narrow: filesystem match)

These verify *specific* external anchors but don't detect the general failure mode of reasoning without evidence.

**What gates or tooling could make violations detectable?**

1. **Provenance Tracker Plugin** (designed Jan 8, unbuilt):
   - Track evidence-gathering tool calls per session
   - Flag sessions that conclude without read/grep/glob/bash
   - Inject warning: "No evidence-gathering detected. What external source supports your conclusions?"

2. **Claim-Evidence Linking Gate**:
   - Parse SYNTHESIS.md or investigation files for claims
   - Check if corresponding evidence-gathering occurred
   - Require explicit source citations for conclusions

3. **Closed-Loop Detection**:
   - Detect patterns like "I reasoned that..." without prior evidence
   - Flag conversations where conclusions reference only other conclusions
   - This is harder - would require semantic analysis

---

## Structured Uncertainty

**What's tested:**

- ✅ Provenance defined as foundational in principles.md (verified: read file, line 19)
- ✅ pkg/verify/check.go has no GateProvenance (verified: read 600-line file)
- ✅ Provenance Tracker plugin designed but not in ~/.config/opencode/plugin/ (verified: ls directory)
- ✅ Skillc provenance field is documentation not enforcement (verified: read load_bearing investigation)

**What's untested:**

- ⚠️ Whether Provenance Tracker plugin design would actually catch violations (needs implementation + testing)
- ⚠️ Whether closed-loop detection is technically feasible (needs prototype)
- ⚠️ Performance impact of tracking all read/grep/glob/bash calls per session

**What would change this:**

- Finding would be wrong if there's a hidden Provenance gate I didn't find
- Finding would be wrong if plugins/coaching.ts implements Provenance checking
- Finding would be incomplete if there are enforcement mechanisms outside orch-go (e.g., in OpenCode itself)

---

## Implementation Recommendations

**Purpose:** Bridge investigation findings to actionable enforcement of Provenance.

### Recommended Approach ⭐

**Build Provenance Tracker Plugin** - Implement the Jan 8 design with observation-first approach.

**Why this approach:**
- Design already exists and was validated in investigation
- Plugin infrastructure is proven (7 other plugins in production)
- Observation-first generates data before adding hard gates
- Low risk (doesn't block operations, just surfaces violations)

**Trade-offs accepted:**
- Observation without blocking allows violations to continue
- Could be noisy if legitimate sessions have minimal evidence-gathering
- Requires analysis to tune thresholds before adding gates

**Implementation sequence:**
1. Implement basic evidence-gathering tracker (track read/grep/glob/bash per session)
2. On session.idle, if evidence.size === 0, log warning
3. After data collection (1-2 weeks), analyze patterns
4. Add soft guidance injection for zero-evidence sessions
5. Consider hard gate only for known-bad patterns

### Alternative Approaches Considered

**Option B: Add GateProvenance to orch complete**
- **Pros:** Hard enforcement, immediate effect
- **Cons:** Too blunt - some sessions legitimately need minimal evidence gathering (e.g., pure implementation from clear specs). Would need session-type discrimination.
- **When to use instead:** After Option A provides data on actual violation patterns

**Option C: Require Source Citations in SYNTHESIS.md**
- **Pros:** Makes provenance visible and auditable
- **Cons:** Can be gamed (fabricated citations). Adds friction to every session.
- **When to use instead:** For high-stakes sessions (architecture, decisions) where provenance is critical

**Rationale for recommendation:** Option A (Provenance Tracker Plugin) provides data-driven path to Option B (hard gates). Jumping to Option B without data risks false positives.

---

## References

**Files Examined:**
- `~/.kb/principles.md` - Provenance principle definition (lines 19-46)
- `pkg/verify/check.go` - Verification gates (11 gates, none for Provenance)
- `.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md` - Provenance Tracker design
- `~/.config/opencode/plugin/friction-capture.ts` - Existing plugin pattern
- `.kb/investigations/2026-01-14-inv-migration-tag-existing-hard-won.md` - Skillc provenance field

**Commands Run:**
```bash
# Search for provenance mentions in kb
grep -ri "provenance" /Users/dylanconlin/Documents/personal/orch-go/.kb/

# List existing plugins
ls -la ~/.config/opencode/plugin/

# Search verify package for evidence checks
grep -r "evidence\|verify" pkg/verify/
```

**Related Artifacts:**
- **Principle:** `~/.kb/principles.md` - Provenance as foundational principle
- **Investigation:** `.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md` - Plugin design including Provenance Tracker
- **Decision:** `.kb/decisions/2026-01-17-infrastructure-over-instruction.md` - Related principle about moving policy to infrastructure

---

## Investigation History

**2026-01-23 XX:XX:** Investigation started
- Initial question: Is Provenance infrastructure or just principle? What enforces it?
- Context: Spawned from orchestrator to understand governance gaps

**2026-01-23 XX:XX:** Core findings complete
- Discovered Provenance has no enforcement infrastructure
- Found existing gates verify narrow external traces only
- Located unbuilt Provenance Tracker plugin design from Jan 8

**2026-01-23 XX:XX:** Investigation completed
- Status: Complete
- Key outcome: Provenance is principle-only with no gates. Provenance Tracker plugin was designed but never built. Recommend implementing it.
