## Summary (D.E.K.N.)

**Delta:** 10+ principles remain as instructions when they should be infrastructure; highest-impact gaps are Provenance verification, Capture at Context, Evidence Hierarchy, and Strategic-First enforcement.

**Evidence:** Systematic review of 27 principles against coaching plugin, orch complete gates, hooks, and kb tools. Found ~40% have infrastructure, ~30% partial, ~30% instruction-only.

**Knowledge:** Principles that describe **what agents should check** (Provenance, Evidence Hierarchy) have no infrastructure, while principles about **what tools should surface** (Surfacing Over Browsing, Authority is Scoping) are well-implemented.

**Next:** Create beads issues for top 5 infrastructure gaps: Provenance verification gate, Capture at Context temporal triggers, Evidence Hierarchy grep-before-claim, Friction capture automation, Strategic-First mandatory gate.

**Promote to Decision:** recommend-no (audit findings, not architectural decision)

---

# Investigation: Audit Governance Principles Become Infrastructure

**Question:** Which principles in principles.md are still instructions that should become enforced infrastructure?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Spawned worker (orch-go-xuejv)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current Infrastructure Landscape

**Evidence:** The system has four infrastructure layers for principle enforcement:

1. **Tool Layer** (OpenCode)
   - Task tool disabled via `.opencode/opencode.json` - forces orch spawn delegation
   - Coaching plugin (`plugins/coaching.ts`) detects 6+ behavioral patterns

2. **Gate Layer** (orch complete verification in `pkg/verify/`)
   - `check.go`: 11 gate constants (phase_complete, synthesis, test_evidence, visual_verification, etc.)
   - `phase_gates.go`: Verifies required phases reported
   - `test_evidence.go`: Requires actual test output, not claims
   - `visual.go`: Requires screenshot evidence for web changes
   - `constraint.go`: Verifies spawn context constraints

3. **Hook Layer**
   - `session-start.sh`: Injects session resume context
   - `.beads/hooks/on_close`: Emits completion events

4. **Knowledge Layer**
   - `kb quick`: Capture decisions, constraints, attempts
   - `kb context`: Surface relevant knowledge
   - `bd ready`: Surface unblocked work

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/*.go`
- `~/.claude/hooks/session-start.sh`
- `/Users/dylanconlin/Documents/personal/orch-go/.beads/hooks/on_close`

**Significance:** Strong foundation exists for infrastructure enforcement. Gaps are specific - not architectural.

---

### Finding 2: Principles by Enforcement Level

**Evidence:** Systematic mapping of 27 principles to infrastructure:

| Level | Count | Examples |
|-------|-------|----------|
| **HIGH** (Infrastructure) | 8 | Surfacing Over Browsing, Pain as Signal, Authority is Scoping, Local-First, Observation Infrastructure |
| **MEDIUM** (Partial) | 9 | Gate Over Remind, Track Actions, Coherence Over Patches, Perspective is Structural, Premise Before Solution |
| **LOW/NONE** (Instruction) | 10 | Provenance, Evidence Hierarchy, Capture at Context, Friction is Signal, Escalation is Information Flow |

**Source:** Comparison of `~/.kb/principles.md` against infrastructure inventoried in Finding 1.

**Significance:** ~40% well-enforced, ~30% partial, ~30% instruction-only. The instruction-only principles are predominantly **epistemic** (how to verify claims) rather than **operational** (how to work).

---

### Finding 3: Coaching Plugin Coverage

**Evidence:** The coaching plugin implements detection for:

| Pattern | Principle Covered | Enforcement Type |
|---------|------------------|------------------|
| Analysis paralysis | Gate Over Remind | Warning injection |
| Frame collapse | Perspective is Structural | Warning injection |
| Behavioral variation | Track Actions, Not Just State | Warning + metric |
| Circular patterns | Coherence Over Patches (partial) | Warning injection |
| Premise-skipping | Premise Before Solution | Warning injection |

What's NOT covered:
- **Provenance** - No verification that claims trace to evidence
- **Evidence Hierarchy** - No check that grep/search precedes claims
- **Capture at Context** - No temporal triggering of knowledge capture
- **Verification Bottleneck** - No rate limiting of changes per verification

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts` lines 1-1846

**Significance:** Coaching plugin handles behavioral patterns but not epistemic verification.

---

### Finding 4: orch complete Gate Coverage

**Evidence:** Current gates in `pkg/verify/check.go`:

| Gate | What It Verifies | Principle Covered |
|------|------------------|------------------|
| GatePhaseComplete | "Phase: Complete" reported | Gate Over Remind |
| GateSynthesis | SYNTHESIS.md exists | Session Amnesia |
| GateHandoffContent | SYNTHESIS.md has content | Session Amnesia |
| GateTestEvidence | Actual test output present | Evidence Hierarchy (partial) |
| GateVisualVerify | Screenshot evidence for web changes | Evidence Hierarchy (partial) |
| GateGitDiff | Claimed changes match actual diff | Provenance (partial) |
| GateBuild | Project builds successfully | - |
| GateDecisionPatchLimit | Not too many patches to one decision | Coherence Over Patches |

What's NOT gated:
- **General Provenance** - No check that conclusions trace to verifiable evidence
- **Capture at Context** - Capture only at session end, not at phase transitions
- **Friction capture** - No requirement to document friction when failures detected
- **Strategic-First** - No blocking gate for HOTSPOT areas

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go` lines 12-25

**Significance:** Gates focus on artifacts and test evidence, not on reasoning quality.

---

### Finding 5: Highest-Impact Infrastructure Gaps

**Evidence:** Based on principle importance (foundational, frequently violated, high-cost failures):

| Gap | Principle | Why High Impact | Proposed Infrastructure |
|-----|-----------|-----------------|------------------------|
| 1. Provenance verification | **Provenance** | Foundational principle; closed loops caused system spirals | Gate: Require evidence trail for conclusions |
| 2. Temporal capture triggers | **Capture at Context** | Context decays; session-end gates capture reconstructed, not observed | Hook: Inject kb quick prompts at phase transitions |
| 3. Grep-before-claim check | **Evidence Hierarchy** | Agents claim "X exists/doesn't exist" without searching | Plugin: Detect existence claims without prior search |
| 4. Friction capture automation | **Friction is Signal** | Friction disappears without capture; same bugs recur | Gate: Require friction section when failures detected |
| 5. Strategic-First mandatory | **Strategic-First Orchestration** | HOTSPOT areas get patched repeatedly; 8 bugs + 2 abandons on coaching plugin | Gate: Block tactical debugging in HOTSPOT areas |

**Source:** Cross-referencing principle provenance table with current infrastructure gaps.

**Significance:** These five gaps represent the highest-leverage improvements. All have clear implementation paths.

---

## Synthesis

**Key Insights:**

1. **Operational principles are well-covered; epistemic principles are not** - The system enforces *what to produce* (SYNTHESIS.md, test evidence, screenshots) but not *how to reason* (tracing claims to evidence, verifying before concluding).

2. **Gates fire at wrong times** - Capture at Context identified that end-of-session gates capture reconstructed context, not observed. Current implementation: session-end only. Needed: phase-transition triggers.

3. **Coaching plugin is behavioral, not epistemic** - Detects patterns like analysis paralysis and frame collapse, but not reasoning flaws like conclusions without evidence.

4. **Strategic-First is advisory when it should be mandatory** - HOTSPOT warnings exist but don't block. Evidence: coaching plugin had 8 bugs + 2 abandonments from tactical approaches before strategic (architect) approach succeeded.

**Answer to Investigation Question:**

10+ principles remain as instructions that should become infrastructure:

**High Priority (clear path, high impact):**
1. Provenance - Add verification that conclusions trace to evidence
2. Capture at Context - Add phase-transition triggers for knowledge capture
3. Evidence Hierarchy - Detect claims without prior search/verification
4. Friction is Signal - Automate friction capture when failures detected
5. Strategic-First Orchestration - Block tactical debugging in HOTSPOT areas

**Medium Priority (important but harder):**
6. Verification Bottleneck - Rate limit changes per human verification
7. Escalation is Information Flow - Track escalation patterns for visibility
8. Understanding Lag - Require "newly visible vs new problem" annotation

---

## Structured Uncertainty

**What's tested:**

- ✅ Coaching plugin source reviewed - patterns documented match code
- ✅ pkg/verify gates enumerated - gate constants match implementation
- ✅ Hooks inventoried - session-start and beads on_close confirmed
- ✅ kb quick capabilities confirmed via help output

**What's untested:**

- ⚠️ Provenance gate feasibility - would need design exploration
- ⚠️ Phase-transition hook implementation - may require OpenCode API changes
- ⚠️ Grep-before-claim detection accuracy - may produce false positives

**What would change this:**

- If Provenance verification is infeasible at tool layer, would need skill-level enforcement
- If phase-transition hooks can't inject prompts, would need different temporal trigger mechanism

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Incremental infrastructure promotion** - Implement the five high-priority gaps as separate beads issues, starting with Strategic-First mandatory gate (most urgent, clearest path).

**Why this approach:**
- Each gap is independent - no dependencies between implementations
- Incremental allows learning from each implementation
- Strategic-First has clearest path: already have HOTSPOT detection, just need to change from warning to block

**Trade-offs accepted:**
- Epistemic gates (Provenance, Evidence Hierarchy) are harder to implement accurately
- May need iteration on detection accuracy

**Implementation sequence:**
1. **Strategic-First mandatory gate** - Change HOTSPOT warning to blocking gate in coaching plugin
2. **Friction capture automation** - Add friction section requirement when failures detected (gate in orch complete)
3. **Capture at Context temporal triggers** - Add phase-transition hooks (requires OpenCode exploration)
4. **Evidence Hierarchy grep-before-claim** - Add detection to coaching plugin
5. **Provenance verification** - Design exploration needed first

### Alternative Approaches Considered

**Option B: Skill-level enforcement only**
- Pros: No infrastructure changes needed
- Cons: Instructions fail under cognitive load (the problem we're solving)
- When to use instead: If infrastructure changes prove infeasible

**Option C: Big-bang infrastructure overhaul**
- Pros: Comprehensive coverage
- Cons: High risk, long timeline, harder to iterate
- When to use instead: Never - incremental is safer

---

## References

**Files Examined:**
- `~/.kb/principles.md` - All 27 principles reviewed
- `/Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts` - Coaching plugin patterns
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/*.go` - Verification gates
- `~/.claude/hooks/session-start.sh` - Session resume hook
- `/Users/dylanconlin/Documents/personal/orch-go/.beads/hooks/on_close` - Completion event hook

**Commands Run:**
```bash
# List verify package contents
ls -la /Users/dylanconlin/Documents/personal/orch-go/pkg/verify/

# Check kb capabilities
kb --help
```

**Related Artifacts:**
- **Decision:** `~/.kb/decisions/2026-01-17-infrastructure-over-instruction.md` - The principle this audit validates
- **Investigation:** `.kb/investigations/2026-01-22-audit-feature-impl-skill-constitutional-constraints.md` - Related skill audit

---

## Investigation History

**2026-01-23 01:30:** Investigation started
- Initial question: Which principles are instructions that should become infrastructure?
- Context: Infrastructure Over Instruction principle says policy should be code

**2026-01-23 01:45:** Infrastructure inventory complete
- Documented four layers: tool, gate, hook, knowledge
- Identified coaching plugin and orch complete as main enforcement points

**2026-01-23 02:00:** Principle mapping complete
- Classified 27 principles by enforcement level
- Identified 10+ instruction-only principles

**2026-01-23 02:15:** Investigation completed
- Status: Complete
- Key outcome: Five high-priority infrastructure gaps identified with implementation recommendations
