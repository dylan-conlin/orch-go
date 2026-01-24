## Summary (D.E.K.N.)

**Delta:** Beads already has 80% of the decidability substrate (ResolutionType, Authority on edges, HOP entity tracking) built into the schema but NOT exposed via CLI - architect investigation missed this by examining issues.jsonl instead of types.go source.

**Evidence:** Examined beads fork types.go (lines 35-117, 429-575, 934-1047) - found ResolutionType enum (factual/judgment/framing), Authority enum on dependencies (daemon/orchestrator/human), and full HOP entity tracking (Creator, Validations, EntityRef).

**Knowledge:** The gap is CLI exposure and tooling integration, not schema design. Beads has the right data model; it just isn't queryable yet.

**Next:** Create issue for beads CLI exposure (--resolution-type, --domain, --authority flags), then reassess architect's Phase 1 (may be mostly done).

**Promote to Decision:** recommend-yes - Establishes that decidability infrastructure exists in beads, redirects implementation effort from schema design to CLI exposure.

---

# Investigation: Reconcile Architect Accountability Architecture Proposal

**Question:** How does the architect's accountability architecture proposal align with existing beads decidability infrastructure? What's built, what aligns, what conflicts, what's missing?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Worker agent (investigation skill)
**Phase:** Complete
**Next Step:** None (reconciliation complete)
**Status:** Complete

**Extracted-From:** Epic orch-go-kz7zr (Governance Infrastructure for Human-AI Systems)

---

## Findings

### Finding 1: Beads Schema Already Has ResolutionType and Domain Fields

**Evidence:** The beads types.go file defines ResolutionType as a first-class field on the Issue struct:

```go
// ===== Decidability Fields (substrate extensions) =====
// ResolutionType indicates how questions should be resolved:
// - factual: Can be answered by research/testing
// - judgment: Requires orchestrator/human judgment call
// - framing: Meta-question about how to think about a problem
ResolutionType ResolutionType `json:"resolution_type,omitempty"`
// Domain categorizes decisions for efficient frontier queries
// Examples: model-selection, spawn-architecture, api-design
Domain string `json:"domain,omitempty"`
```

And the ResolutionType enum (lines 429-454):
```go
const (
    ResolutionFactual ResolutionType = "factual"
    ResolutionJudgment ResolutionType = "judgment"
    ResolutionFraming ResolutionType = "framing"
)
```

**Source:** `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go:35-44, 429-454`

**Significance:** This is EXACTLY what the architect proposed as Phase 1 ("Structured authority in skills") - but it already exists in beads! The architect investigation found "decidability not encoded" by looking at actual issues, not the schema that supports encoding it.

---

### Finding 2: Authority Field Already Exists on Dependencies

**Evidence:** The Dependency struct has an Authority field:

```go
type Dependency struct {
    IssueID     string         `json:"issue_id"`
    DependsOnID string         `json:"depends_on_id"`
    Type        DependencyType `json:"type"`
    CreatedAt   time.Time      `json:"created_at"`
    CreatedBy   string         `json:"created_by,omitempty"`
    // Authority indicates who can traverse/resolve this edge
    // Used for decidability substrate: daemon can only traverse daemon-level edges
    Authority Authority `json:"authority,omitempty"`
    // ...
}
```

And the Authority enum (lines 549-575):
```go
const (
    // AuthorityDaemon indicates the dependency can be resolved by automated processes
    AuthorityDaemon Authority = "daemon"

    // AuthorityOrchestrator indicates the dependency requires orchestrator judgment
    AuthorityOrchestrator Authority = "orchestrator"

    // AuthorityHuman indicates the dependency requires human decision
    AuthorityHuman Authority = "human"
)
```

**Source:** `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go:532-575`

**Significance:** The decidability graph model's "edge authority" concept is already implemented in beads schema. The architect proposal for "authority on edges" is already built - just not exposed via CLI.

---

### Finding 3: HOP Entity Tracking Provides "Consequence Bearer" Infrastructure

**Evidence:** The Issue struct has Creator and Validations fields for entity tracking:

```go
// ===== HOP Fields (entity tracking for CV chains) =====
Creator     *EntityRef   `json:"creator,omitempty"`     // Who created (human, agent, or org)
Validations []Validation `json:"validations,omitempty"` // Who validated/approved
```

EntityRef is a full structured reference (lines 934-992):
```go
type EntityRef struct {
    Name     string `json:"name,omitempty"`     // Human-readable identifier
    Platform string `json:"platform,omitempty"` // Execution context (gastown, github)
    Org      string `json:"org,omitempty"`      // Organization
    ID       string `json:"id,omitempty"`       // Unique identifier
}
```

With URI support: `entity://hop/<platform>/<org>/<id>`

**Source:** `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go:94-96, 934-1047`

**Significance:** The architect identified "consequence bearer is implicitly Dylan, never explicit" as a gap. But beads already has the Creator field with full entity tracking! This is the infrastructure for making consequence bearers explicit. The gap is that it's not being used/exposed.

---

### Finding 4: CLI Does NOT Expose Decidability Fields

**Evidence:** Checked `bd create --help` and `bd dep add --help`:

```
bd create --help
# No --resolution-type flag
# No --domain flag

bd dep add --help
# No --authority flag
# Types limited to: blocks|tracks|related|parent-child|discovered-from
```

**Source:** CLI help output, commands checked 2026-01-22

**Significance:** The schema exists but isn't accessible. Users and tools can't set resolution_type, domain, or authority via CLI. This is the real gap - not schema design, but CLI exposure.

---

### Finding 5: Architect Investigation Looked at Wrong Source

**Evidence:** The architect investigation examined:
- `.beads/issues.jsonl` (actual issue data)
- `pkg/spawn/context.go` (spawn templates)
- `~/.claude/skills/` (skill files)
- `.kb/guides/decision-authority.md` (documentation)

But did NOT examine:
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go` (the actual schema)

From architect investigation Finding 1 table:
> | **Beads issues** | Priority, type, dependencies, labels | Structured (JSON) | Yes |

This shows fields observed in actual issues, not fields available in schema.

**Source:** Architect investigation `.kb/investigations/2026-01-22-inv-design-accountability-architecture-first-class.md:32-51`

**Significance:** The investigation's conclusions were based on observed data (what's actually stored) not potential data (what can be stored). The schema supports decidability; it's just not being populated.

---

## Synthesis

**Key Insights:**

1. **Schema exists, exposure doesn't** - Beads has ResolutionType, Domain, Authority, and HOP entity tracking built into the schema. The architect proposal for "make decidability queryable" is already structurally possible - the fields exist, they're just not exposed via CLI or being populated.

2. **The architect's proposed Phase 1 is already done (in beads)** - "Structured authority in skills" proposed adding frontmatter like `resolution_type: judgment`. But this field already exists in beads issues. The question is whether authority belongs in skills (orch-knowledge) or in beads issues (where it already exists).

3. **The real work is different than proposed** - Instead of:
   - Phase 1: Design decidability schema ❌ (already done)
   - Phase 2: Inject into spawn ❌ (spawn can already reference beads fields)

   The actual work is:
   - Phase 1: Add CLI flags to bd (`--resolution-type`, `--domain`, `--authority`) ✅
   - Phase 2: Start populating these fields on actual issues ✅
   - Phase 3: Build query interface (`bd ready --authority daemon`) ✅

**Answer to Investigation Question:**

| Architect Proposal | Status | Action Needed |
|-------------------|--------|---------------|
| ResolutionType field | **BUILT** (types.go:35-44) | Add CLI flag |
| Domain field | **BUILT** (types.go:43) | Add CLI flag |
| Authority on edges | **BUILT** (types.go:538-575) | Add CLI flag |
| Consequence bearer | **BUILT** as Creator (types.go:95) | Add CLI flag, start using |
| `.kb/governance/` | NOT BUILT | May not be needed (use beads instead?) |
| `kb authority` commands | NOT BUILT | Could query beads instead |
| Dashboard authority panel | NOT BUILT | Still needed |

**What conflicts:** Nothing - architect proposal aligns perfectly with beads schema. The schema was apparently designed with decidability in mind (see the explicit "Decidability Fields" comment in types.go).

**What's missing:** CLI exposure and adoption. The infrastructure is there; it's not being used.

---

## Structured Uncertainty

**What's tested:**

- ✅ Beads schema has ResolutionType field (verified: read types.go)
- ✅ Beads schema has Authority on Dependency (verified: read types.go)
- ✅ Beads schema has Creator EntityRef (verified: read types.go)
- ✅ CLI does NOT expose these fields (verified: ran bd create --help, bd dep add --help)

**What's untested:**

- ⚠️ Whether `bd ready` respects Authority field on dependencies (not tested)
- ⚠️ Whether frontier queries can be built using existing schema (not tested)
- ⚠️ Whether HOP entity tracking (Creator, Validations) works end-to-end (not tested)

**What would change this:**

- If beads team says these fields are deprecated/experimental, would need new approach
- If Authority field isn't wired into `bd ready` query logic, would need schema changes
- If CLI exposure is blocked by architectural concerns, would need alternative

---

## Implementation Recommendations

**Purpose:** Redirect implementation from schema design to CLI exposure.

### Recommended Approach: Expose Existing Schema ⭐

**Focus on CLI exposure, not new schema design.**

**Why this approach:**
- Schema already exists and is well-designed
- CLI exposure is lower risk than schema changes
- Can ship incrementally (one flag at a time)
- Validates the schema design through actual use

**Trade-offs accepted:**
- May discover schema gaps during exposure (acceptable - iterate)
- Beads fork must accept these changes (already Dylan's fork)

**Implementation sequence:**
1. **Add `--resolution-type` to `bd create`** - Enable setting factual/judgment/framing
2. **Add `--domain` to `bd create`** - Enable categorization
3. **Add `--authority` to `bd dep add`** - Enable edge-level authority
4. **Add `--authority` filter to `bd ready`** - Enable frontier queries
5. **Update daemon to respect Authority** - Only traverse daemon-level edges

### Alternative Approaches Considered

**Option B: Implement architect's Phase 1 (skills frontmatter)**
- **Pros:** Keeps authority close to skill definitions
- **Cons:** Duplicates what beads already has; skills aren't queryable
- **When to use instead:** If beads can't be extended (unlikely - it's Dylan's fork)

**Option C: Build `.kb/governance/` as new artifact type**
- **Pros:** Clean separation, governance-specific tooling
- **Cons:** Yet another place for authority info; inconsistent with beads
- **When to use instead:** If authority spans beyond issues (org-level governance)

**Rationale for recommendation:** Beads already has the infrastructure. Exposing it is faster and more consistent than building parallel structures.

---

### Implementation Details

**What to implement first:**
- `--resolution-type` flag on `bd create` - Highest value, enables question typing
- `--authority` filter on `bd ready` - Enables daemon-safe queries

**Things to watch out for:**
- ⚠️ `bd ready` may need query logic changes to filter by dependency authority
- ⚠️ Existing issues have no resolution_type set - need backfill strategy
- ⚠️ Dashboard may need API changes to expose these fields

**Areas needing further investigation:**
- How should existing issues be backfilled with resolution_type?
- Should `bd ready` default to `--authority daemon` for safety?
- How does Authority interact with labels like `triage:ready`?

**Success criteria:**
- ✅ Can create issue with `bd create --type question --resolution-type judgment`
- ✅ Can add dependency with `bd dep add X Y --authority human`
- ✅ Can query with `bd ready --authority daemon` (daemon-safe frontier)
- ✅ Dashboard shows authority state on issues

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go` - Beads schema (primary source)
- `.kb/investigations/2026-01-22-inv-design-accountability-architecture-first-class.md` - Architect investigation
- `.kb/models/decidability-graph.md` - Decidability model documentation
- `.kb/guides/decision-authority.md` - Authority guidance

**Commands Run:**
```bash
# Found beads fork location
ls -la /Users/dylanconlin/bin/bd
# -> /Users/dylanconlin/Documents/personal/beads/build/bd

# Checked CLI flags
bd create --help
bd dep add --help
bd ready --help
```

**Related Artifacts:**
- **Epic:** orch-go-kz7zr - Governance Infrastructure for Human-AI Systems
- **Architect Investigation:** `.kb/investigations/2026-01-22-inv-design-accountability-architecture-first-class.md`
- **Model:** `.kb/models/decidability-graph.md` - Documents how these fields should be used
- **Guide:** `.kb/guides/decision-authority.md` - Documents authority levels (matches schema)

---

## Investigation History

**2026-01-22 23:30:** Investigation started
- Initial question: Reconcile architect proposal with existing beads infrastructure
- Context: Part of Governance Infrastructure epic (orch-go-kz7zr)

**2026-01-22 23:45:** Major discovery
- Found beads types.go has ResolutionType, Authority, HOP entity tracking
- Realized architect investigation looked at issues.jsonl, not schema source
- The infrastructure exists; it's just not exposed

**2026-01-22 23:55:** Investigation completed
- Status: Complete
- Key outcome: 80% of decidability substrate already built in beads schema; redirect effort from schema design to CLI exposure
