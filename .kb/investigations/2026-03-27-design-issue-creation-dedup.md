## Summary (D.E.K.N.)

**Delta:** Architect skill creates follow-up issues via `bd create` with zero pre-creation dedup — all dedup happens downstream at spawn-time via CommitDedupGate, leaving zombie issues in beads.

**Evidence:** Architect SKILL.md.template Phase 5d had 4 `bd create` call sites with no dedup guidance. CommitDedupGate (pkg/daemon/prior_art_dedup.go) catches duplicates at spawn-time but issues already exist in beads. Duplicate extraction provenance trace (2026-02-16) documented 9+ zombie issues from this pattern.

**Knowledge:** Dedup is a layered concern: (1) issue-creation-time in skills, (2) spawn-time in daemon gates, (3) execution-time via session dedup. Layer 1 was missing entirely. Adding a Prior Art Check procedure to the architect skill template catches duplicates before they enter beads.

**Next:** Skill template updated and deployed. Worker-base discovered-work section also needs this check but is governance-protected — follow-up issue created.

**Authority:** implementation - Skill text change within existing patterns, no architectural impact

---

# Investigation: Issue Creation Dedup

**Question:** Should the architect skill check for already-committed work before creating follow-up issues via `bd create`? Where should this check live (skill text vs daemon code)?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** architect

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-16 duplicate extraction provenance trace (probe) | extends | Yes — confirmed zombie issue pattern persists | None |
| 2026-03-26 daemon duplicate spawn detection (prior_art_dedup.go) | extends | Yes — CommitDedupGate exists but is spawn-time only | None |

---

## Findings

### Finding 1: Zero pre-creation dedup in architect skill

**Evidence:** The architect SKILL.md.template Phase 5d (Decomposition) contained 4 `bd create` call sites with no dedup guidance:
- Line 244: `bd create` for component issues
- Line 245: `bd create` for integration issues
- Line 251: `bd create` for single-component issues
- Line 171: `bd create` for question entities (Phase 3)

**Source:** `skills/src/worker/architect/.skillc/SKILL.md.template:244-251`

**Significance:** Every follow-up issue is created blind. Architects have no instruction to check whether the work they're proposing has already been done by another agent.

---

### Finding 2: CommitDedupGate is downstream bandaid

**Evidence:** The daemon's SpawnPipeline has a CommitDedupGate (L6) at `pkg/daemon/prior_art_dedup.go:47-124` that catches already-committed work at spawn-time. Two checks:
1. Issue's own beads ID in git commits
2. Referenced beads IDs with commits (with title similarity filtering)

But this gate operates AFTER issue creation. The issue exists in beads as a zombie — created, never spawned, never closed.

**Source:** `pkg/daemon/prior_art_dedup.go:47-124`, `pkg/daemon/spawn_gate.go:107-149`

**Significance:** Spawn-time dedup prevents wasted agent spawns but doesn't prevent wasted issues. The create-then-filter pattern leaves zombie issues in beads.

---

### Finding 3: Worker-base is governance-protected

**Evidence:** The worker-base discovered-work section (`skills/src/shared/worker-base/.skillc/discovered-work.md`) has `bd create` templates for ALL skills with no dedup guidance. But worker-base is governance-protected:
> `skills/src/shared/worker-base` — worker base skill (shared protocols)
> **Instead:** Escalate to orchestrator — worker-base skill can only be modified in direct sessions

**Source:** Governance-protected paths in spawn context

**Significance:** The broader fix (adding dedup to all skills via worker-base) requires escalation to orchestrator. The architect skill can be fixed independently.

---

## Synthesis

**Key Insights:**

1. **Dedup is a layered concern** — The system had layers 2 (spawn-time) and 3 (execution-time/session dedup) but was entirely missing layer 1 (creation-time). This probe confirms Defect Class 6 (Duplicate Action) exposure in the architect skill's issue creation path.

2. **Filter-then-create beats create-then-filter** — Moving the dedup check upstream from spawn-time to creation-time prevents zombie issues from accumulating in beads. The cost of the check is low (2 shell commands: `git log` + `bd list`).

3. **Skill-level dedup is instructional, not mechanical** — Unlike daemon spawn gates (Go code), skill-level dedup is behavioral guidance in the skill template. Agents follow the procedure because it's in their loaded context. This matches the established pattern for skill constraints.

**Answer to Investigation Question:**

Yes, the architect skill should check for already-committed work before creating follow-up issues. The check belongs in skill text (SKILL.md.template Phase 5d) as a "Prior Art Check" procedure that agents follow before every `bd create` call. This is upstream of the daemon's CommitDedupGate and prevents zombie issues from being created.

---

## Structured Uncertainty

**What's tested:**

- ✅ Architect skill template had zero pre-creation dedup guidance (verified: read SKILL.md.template, grep for dedup patterns)
- ✅ CommitDedupGate operates at spawn-time only (verified: read prior_art_dedup.go, spawn_gate.go)
- ✅ Worker-base is governance-protected (verified: governance-protected paths list)

**What's untested:**

- ⚠️ Effectiveness of skill-level dedup guidance (not measured — requires observing architect agents after deploy)
- ⚠️ False positive rate of git log keyword matching (not benchmarked — keywords may match unrelated commits)

**What would change this:**

- If agents routinely ignore skill text guidance, this approach would fail and a code-level gate would be needed
- If git log keyword matching produces >20% false positives, the procedure would need tighter matching criteria

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add Prior Art Check to architect skill template | implementation | Skill text change within existing patterns |
| Add Prior Art Check to worker-base discovered-work | architectural | Crosses governance boundary, requires orchestrator session |

### Recommended Approach ⭐

**Prior Art Check in Architect Skill Template** - Add a "5d. Prior Art Check" phase before issue creation in the architect skill's externalization workflow.

**Why this approach:**
- Catches duplicates at the earliest possible point (creation-time vs spawn-time)
- Low cost: 2 shell commands per proposed issue
- Follows existing skill guidance patterns (instructional, not mechanical)

**Trade-offs accepted:**
- Relies on agent compliance with skill text (soft gate, not hard gate)
- Only covers architect skill — other skills still create blind (worker-base is governance-protected)

**Implementation sequence:**
1. Add "5d. Prior Art Check" section to SKILL.md.template ✅ Done
2. Wire prior art check into Phase 5e (Decomposition) steps ✅ Done
3. Deploy via skillc ✅ Done
4. Create follow-up issue for worker-base change (requires orchestrator)

### Enforcement Mechanisms

| Mechanism | Type | Consequence Sensor |
|-----------|------|--------------------|
| Prior Art Check (skill text) | advisory | DEDUP comments in beads — searchable via `bd list \| grep DEDUP` |
| CommitDedupGate (daemon) | gate | events.jsonl spawn.gate_decision — fire rate via orch stats |

---

## References

**Files Examined:**
- `skills/src/worker/architect/.skillc/SKILL.md.template` - Architect skill template (edited)
- `pkg/daemon/prior_art_dedup.go` - CommitDedupGate implementation
- `pkg/daemon/spawn_gate.go` - SpawnPipeline architecture
- `skills/src/shared/worker-base/.skillc/discovered-work.md` - Worker-base bd create templates
- `.kb/models/architect/model.md` - Architect model (updated with probe)

**Related Artifacts:**
- **Probe:** `.kb/models/architect/probes/2026-03-27-probe-issue-creation-dedup-effectiveness.md`
- **Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-16-duplicate-extraction-provenance-trace.md`
