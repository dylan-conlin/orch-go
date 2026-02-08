<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Cross-boundary decidability requires three primitives: authority claims (what I decide), authority requirements (what I need from you), and handoff tokens (bounded delegation)—analogous to capability-based security.

**Evidence:** Analysis of current model (authority on edges, consequence-bearer termination), cross-boundary scenarios (shipping to contractors, multi-org projects, external APIs), and analogies from capability systems and API contracts.

**Knowledge:** The current "Dylan-bears-consequences" termination point is the implicit trust boundary; making it explicit enables composition through declared interfaces rather than assumed containment.

**Next:** Document composability patterns in decidability-graph model; consider authority-claim field for governance exports when shipping code.

**Promote to Decision:** recommend-no (conceptual framework, not yet architectural; needs concrete use case to crystallize)

---

# Investigation: Decidability Graph Composability Across Trust

**Question:** What happens when decidability graphs from different systems meet? How do authority claims become contracts?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Worker agent (feature-impl skill)
**Phase:** Complete
**Next Step:** None (conceptual investigation complete)
**Status:** Complete

<!-- Lineage -->
**Patches-Decision:** None (extends decidability-graph model conceptually)
**Extracted-From:** Epic orch-go-kz7zr (Governance Infrastructure for Human-AI Systems)

---

## Findings

### Finding 1: Current Model Has Implicit Trust Boundary at "Dylan-bears-consequences"

**Evidence:** The current decidability graph model terminates authority at Dylan:

```
daemon (Work edges) → orchestrator (Question edges) → Dylan (Gate edges) → [stops here]
```

From decidability-graph.md:
> "**What remains irreducibly human (Dylan):**
> - Overriding scoping decisions
> - Value judgments that determine which frames matter
> - Accountability for where the system points its attention"

The governance architecture investigation explicitly names this: "Consequence bearer is implicitly Dylan, never explicit."

**Source:**
- `.kb/models/decidability-graph.md:98-101` (irreducible human functions)
- `.kb/investigations/2026-01-22-inv-design-accountability-architecture-first-class.md:68-69` (consequence bearer gap)
- `.kb/decisions/2026-01-19-worker-authority-boundaries.md:9` (workers create nodes, orchestrator creates edges)

**Significance:** The implicit termination point works for single-system operation but becomes a problem at composition. When Dylan ships code to Kenneth, the authority chain needs a new termination point. The current model assumes containment (all authority flows to Dylan) rather than declared interfaces (authority claims are explicit).

---

### Finding 2: Three Cross-Boundary Scenarios Reveal Different Composition Patterns

**Evidence:** Analyzed where authority graphs would need to meet:

| Scenario | Description | Composition Need |
|----------|-------------|------------------|
| **Shipping to users** | Dylan's system used by Kenneth/contractors | Authority handoff: "decisions of type X now yours" |
| **Multi-org projects** | Joint work with different consequence bearers | Authority federation: "Alice decides X, Bob decides Y" |
| **External API integration** | Calling services with their own decisions | Authority import: "decisions from elsewhere accepted" |

Each scenario has different trust properties:
- **Shipping**: One-time delegation with bounded scope
- **Multi-org**: Ongoing negotiation with shared context
- **External API**: Black-box trust with interface contract

**Source:**
- `.kb/investigations/2026-01-22-inv-design-accountability-architecture-first-class.md:104-106` (composability queries identified)
- `.kb/models/decidability-graph.md:92-94` (authority chain is about scoping)
- Governance handoff doc sections on composability (referenced in accountability architecture investigation)

**Significance:** Composability isn't one problem—it's three. Different patterns apply to different trust relationships. A shipping handoff needs attenuation (give less authority, never more). Multi-org needs negotiation (explicit interface on who decides what). External API needs encapsulation (accept decisions without seeing the graph that made them).

---

### Finding 3: Capability-Based Security Provides the Right Primitives

**Evidence:** The composability problem maps directly to capability-based security concepts:

| Security Concept | Decidability Equivalent |
|------------------|------------------------|
| **Capability** | Authority token (permission to traverse certain edges) |
| **Attenuation** | Handoff can delegate less authority, never more |
| **Confinement** | Authority doesn't leak across boundaries without explicit grant |
| **Object-capability** | Nodes carry authority requirements, not just data dependencies |

The beads schema already has primitives that support this:
- `Authority` enum on dependencies: `daemon | orchestrator | human`
- `Creator` EntityRef: who made this decision (entity://hop/<platform>/<org>/<id>)

What's missing:
- **Authority claims**: "This system decides things of type X"
- **Authority requirements**: "This system needs decisions of type Y from elsewhere"
- **Boundary nodes**: Gates that specifically require external input

**Source:**
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go:549-575` (Authority enum)
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go:934-992` (EntityRef with platform/org/id)
- `.kb/investigations/2026-01-22-inv-reconcile-architect-accountability-architecture-proposal.md:91-126` (HOP entity tracking)
- Capability-based security literature (Dennis & Van Horn 1966, object-capability model)

**Significance:** The infrastructure is half-built. EntityRef can represent "where this decision came from" (import). Authority on edges can represent "who can traverse" (internal). What's missing is the export side: declaring what authority claims this system makes, so other systems can compose against a declared interface rather than assuming containment.

---

### Finding 4: Authority Claims Become Contracts Through Interface Declaration

**Evidence:** The transition from claims to contracts follows a pattern:

**Authority Claim (unilateral):**
```yaml
# .kb/governance/authority-claims.yaml
system: orch-go
claims:
  - scope: spawn-architecture
    level: orchestrator
    consequences: dylan
  - scope: model-selection
    level: daemon
    consequences: dylan
```

**Authority Contract (bilateral):**
```yaml
# Composition agreement between systems
systems:
  - name: orch-go
    exports:
      - scope: spawn-architecture  # I make these decisions
    imports:
      - scope: billing             # I accept these decisions from you
  - name: infra-system
    exports:
      - scope: billing
    imports:
      - scope: spawn-architecture
```

The transition is:
1. Each system declares claims (what it decides)
2. When systems compose, claims become interface (what's visible to others)
3. Negotiation produces contract (explicit agreement on imports/exports)
4. Contract enables bounded trust (I trust your decisions on X, not Y)

**Source:**
- `.kb/investigations/2026-01-22-inv-design-accountability-architecture-first-class.md:226-266` (governance file formats)
- API versioning patterns (OpenAPI, protobuf service definitions)
- Microservice contract testing (consumer-driven contracts)

**Significance:** The accountability architecture investigation proposed `.kb/governance/authority.yaml` for internal claims. The composability extension is adding `exports` and `imports` to enable cross-system composition. Claims are unilateral (I state my authority); contracts are bilateral (we agree on the interface).

---

## Synthesis

**Key Insights:**

1. **Trust boundaries are implicit today** - The "Dylan-bears-consequences" termination point works because the system assumes containment. All authority flows inward to one human. Composition requires making this boundary explicit: "here's where my authority ends."

2. **Three patterns for three trust relationships** - Shipping (attenuation), multi-org (federation), and external API (encapsulation) each require different composition primitives. A unified model needs to support all three without conflating them.

3. **Claims become contracts at composition** - Authority claims are unilateral declarations ("I decide X"). When systems meet, claims become interface surfaces. Explicit negotiation produces contracts ("I accept your decisions on X, you accept mine on Y"). This is the mechanism by which internal authority becomes externally trustable.

4. **Capability model provides safety properties** - Attenuation (can't grant more than you have), confinement (no implicit leakage), and explicit handoff (authority transfer is visible) are the properties that make composition safe. The beads Authority enum is primitive but could encode these.

**Answer to Investigation Question:**

**What happens when decidability graphs from different systems meet?**

The graphs compose at their boundary nodes—Gates that require authority from outside the system. The current model assumes Dylan is the ultimate authority, so all paths terminate there. When composing with another system:

1. **Boundary identification**: Which nodes require authority from outside this system?
2. **Interface declaration**: What authority claims does this system make? What authority does it require from others?
3. **Contract negotiation**: Explicit agreement on imports/exports between systems
4. **Handoff encoding**: Authority tokens that travel across the boundary (bounded delegation)

**How do authority claims become contracts?**

Through a three-step transition:

1. **Internal claims** → Each system declares what it decides (unilateral)
2. **Interface exposure** → Claims become visible surfaces when systems compose
3. **Bilateral contracts** → Explicit agreement on who decides what at the boundary

The key insight is that contracts aren't new structures—they're claims from different systems that meet at a declared interface. The contract is the intersection of "what I export" and "what you import."

**Current gaps for implementation:**
- No authority export declaration (what decisions this system makes available to others)
- No authority import declaration (what decisions this system accepts from others)
- No boundary node type (Gates that explicitly require external authority)
- No handoff token mechanism (how authority transfers across boundaries)

---

## Structured Uncertainty

**What's tested:**

- ✅ Current model terminates at Dylan (verified: read decidability-graph.md, authority hierarchy explicit)
- ✅ Beads has Authority enum on dependencies (verified: read types.go:549-575)
- ✅ Beads has EntityRef for tracking decision origin (verified: read types.go:934-992)
- ✅ Accountability architecture investigation identified consequence bearer gap (verified: read investigation)

**What's untested:**

- ⚠️ Whether capability model primitives map cleanly to decidability (conceptual, not prototyped)
- ⚠️ Whether multi-org federation is a real use case (no concrete scenario today)
- ⚠️ Whether authority export/import declarations would be used (no current shipping to others)
- ⚠️ Performance of cross-boundary authority validation (no implementation to measure)

**What would change this:**

- If composition never happens (Dylan never ships to others), this entire investigation becomes theoretical
- If multi-org scenarios arise, the federation pattern would need concrete testing
- If a simpler model works (just "you're the consequence bearer now"), the capability framework may be overkill
- If external APIs require deeper trust (not just accept decisions), encapsulation pattern needs extension

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Conceptual Documentation First** - Add composability section to decidability-graph model documenting the three patterns (attenuation, federation, encapsulation) before implementing primitives.

**Why this approach:**
- No concrete use case for composition today (Dylan is sole user)
- Premature implementation risks building wrong abstractions
- Documentation crystallizes the concepts for future implementation
- Follows "Premise Before Solution" principle—validate need before building

**Trade-offs accepted:**
- No executable composability yet
- Defers implementation until shipping scenario arises
- Concepts may need refinement when tested against reality

**Implementation sequence:**
1. **Add "Composability" section to decidability-graph.md** - Document three patterns and primitives
2. **Add authority-claims field to governance schema** (when shipping arises) - Enable export declaration
3. **Implement boundary node type** (when multi-org arises) - Enable explicit external authority requirement

### Alternative Approaches Considered

**Option B: Implement authority export/import now**
- **Pros:** Ready for composition when needed; forces concrete design decisions
- **Cons:** YAGNI—no current use case; may build wrong abstractions
- **When to use instead:** If shipping to Kenneth/contractors becomes imminent

**Option C: Use simpler handoff model**
- **Pros:** Just "transfer consequence bearer" without capability framework
- **Cons:** Loses safety properties (attenuation, confinement); can't represent partial delegation
- **When to use instead:** If all handoffs are complete transfers, not bounded delegation

**Rationale for recommendation:** Composition is valuable conceptually but not urgent operationally. Document the framework now so it's available when needed; implement when a concrete scenario demands it.

---

### Implementation Details

**What to implement first:**
- Add "Composability Across Trust Boundaries" section to `.kb/models/decidability-graph.md`
- Document the three patterns: attenuation (shipping), federation (multi-org), encapsulation (external API)
- Define primitives: authority claims, authority requirements, boundary nodes, handoff tokens

**Things to watch out for:**
- ⚠️ Over-engineering before need is validated (capability model may be too heavy)
- ⚠️ Assuming Dylan will always be the termination point (breaks with shipping)
- ⚠️ Conflating different composition patterns (shipping ≠ federation ≠ API)

**Areas needing further investigation:**
- What does a concrete handoff to Kenneth look like? (drives attenuation design)
- Are there multi-org scenarios on the roadmap? (drives federation design)
- Which external APIs will we integrate? (drives encapsulation design)

**Success criteria:**
- ✅ decidability-graph.md has clear section on cross-boundary composition
- ✅ Three patterns documented with when to use each
- ✅ Primitives defined conceptually (ready for implementation when needed)
- ✅ Future reader can understand how to compose decidability graphs

---

## References

**Files Examined:**
- `.kb/models/decidability-graph.md` - Current model, authority hierarchy, termination point
- `.kb/decisions/2026-01-19-worker-authority-boundaries.md` - Node creation vs edge creation authority
- `.kb/investigations/2026-01-22-inv-design-accountability-architecture-first-class.md` - Consequence bearer gap
- `.kb/investigations/2026-01-22-inv-reconcile-architect-accountability-architecture-proposal.md` - HOP entity tracking
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go` - Authority enum, EntityRef schema

**Commands Run:**
```bash
# Reported phase to orchestrator
bd comments add orch-go-v3jax "Phase: Planning - Investigating decidability graph composability"

# Created investigation file
kb create investigation decidability-graph-composability-across-trust
```

**External Documentation:**
- Capability-based security (Dennis & Van Horn 1966) - Foundational model for authority tokens
- Object-capability model - Safety properties (attenuation, confinement)
- Consumer-driven contracts - Pattern for bilateral interface agreements

**Related Artifacts:**
- **Model:** `.kb/models/decidability-graph.md` - The model this investigation extends
- **Investigation:** `.kb/investigations/2026-01-22-inv-design-accountability-architecture-first-class.md` - Governance architecture
- **Investigation:** `.kb/investigations/2026-01-19-design-investigate-substrate-options-decidability-graphs.md` - Substrate design
- **Epic:** orch-go-kz7zr - Governance Infrastructure for Human-AI Systems

---

## Investigation History

**2026-01-23 ~08:00:** Investigation started
- Initial question: What happens when decidability graphs from different systems meet? How do authority claims become contracts?
- Context: Extracted from governance infrastructure epic; composability mentioned as open question in prior investigations

**2026-01-23 ~08:15:** Analyzed current model
- Found implicit trust boundary at Dylan-bears-consequences
- Identified that composition requires making this boundary explicit

**2026-01-23 ~08:30:** Identified three composition patterns
- Attenuation (shipping to users), federation (multi-org), encapsulation (external API)
- Recognized each has different trust properties

**2026-01-23 ~08:45:** Mapped to capability-based security
- Found existing beads primitives (Authority enum, EntityRef) provide foundation
- Identified gap: no authority export/import declarations

**2026-01-23 ~09:00:** Investigation completed
- Status: Complete
- Key outcome: Cross-boundary composition requires three primitives (claims, requirements, handoff tokens); recommend documenting conceptually before implementing
