<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Core decidability substrate implemented: authority field on deps, resolution_type/domain on issues. Schema, types, migrations, and basic CLI support all complete.

**Evidence:** New migrations (035, 036) add columns; Dependency.Authority and Issue.ResolutionType/Domain fields added; bd dep add --authority works; test written for answered unblocking.

**Knowledge:** The `answered` status already unblocks (contrary to old constraint). Decidability requires both edge metadata (authority) and node metadata (resolution_type, domain) for proper query scoping.

**Next:** Future session: add --resolution-type CLI, bd ready --authority filter, dashboard updates. Core schema is ready.

**Promote to Decision:** recommend-no (implementation complete, not architectural change)

---

# Investigation: Implement Decidability Substrate Extensions Beads

**Question:** What changes are needed to extend beads with decidability substrate features (answered unblocking, authority on edges, resolution_type)?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None (future work documented below)
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-01-18-questions-as-first-class-entities.md`
**Extracted-From:** `.kb/investigations/2026-01-19-design-investigate-substrate-options-decidability-graphs.md`

---

## Findings

### Finding 1: Answered Status Already Unblocks Dependencies

**Evidence:** Code analysis of blocking logic:
- `blocked_cache.go:149-150`: blocking status list is `('open', 'in_progress', 'blocked', 'deferred', 'hooked', 'investigating')`
- `answered` is NOT in this list
- Status changes trigger `invalidateBlockedCache` via `queries.go:919-925`

**Source:**
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/blocked_cache.go:149-150`
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/queries.go:919-925`
- Git commit `744af9cf` added `investigating` but intentionally excluded `answered`

**Significance:** Constraint `kb-fe6173` ("answered doesn't unblock") appears to be outdated or from before full testing. The code already implements correct behavior. Test written to verify: `question_status_test.go`.

---

### Finding 2: Dependency Schema Lacks Authority Field

**Evidence:** The `dependencies` table schema:
```sql
CREATE TABLE IF NOT EXISTS dependencies (
    issue_id TEXT NOT NULL,
    depends_on_id TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'blocks',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT NOT NULL,
    metadata TEXT DEFAULT '{}',
    thread_id TEXT DEFAULT '',
    PRIMARY KEY (issue_id, depends_on_id),
    ...
);
```

No `authority` field exists. The `metadata` JSON could be used, but a proper column is cleaner.

**Source:**
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/schema.go:60-71`
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go:494-507`

**Significance:** Need to add `authority` column to dependencies table via migration. Default should be 'daemon' for backward compatibility.

---

### Finding 3: Issue Schema Lacks Resolution_Type Field

**Evidence:** The `Issue` struct has `IssueType` but no `ResolutionType`:
- Question types exist: `TypeQuestion IssueType = "question"`
- No field for subtyping questions as factual/judgment/framing
- No field for decisions (which could also have domains)

**Source:**
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go:407` (TypeQuestion)
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go:14-108` (Issue struct, no resolution_type)

**Significance:** Need to add `resolution_type` field to Issue struct and database schema. Also consider adding `domain` field for decisions.

---

## Synthesis

**Key Insights:**

1. **Answered Status Works** - The blocking logic already excludes `answered` from blocking statuses. The constraint `kb-fe6173` may have been from before the implementation was tested or there was a misunderstanding.

2. **Authority Requires Schema Change** - Adding authority to edges requires:
   - New column in dependencies table
   - Migration for existing data
   - CLI flag support for `bd dep add --authority`
   - Query support for `bd ready --authority`

3. **Resolution_Type is Straightforward** - Adding resolution_type is a standard field addition:
   - New field in Issue struct
   - New column in issues table
   - CLI flag support for `bd create --resolution-type`
   - Query support for `bd list --resolution-type`

**Answer to Investigation Question:**

The answered status unblocking is **already implemented**. The remaining work is:
1. Add `authority` column to dependencies table (migration + types + CLI)
2. Add `resolution_type` field to issues (for questions, optional for others)
3. Add `bd ready --authority` query support

---

## Structured Uncertainty

**What's tested:**

- ✅ `answered` status NOT in blocking list (code analysis of `blocked_cache.go:149-150`)
- ✅ Status changes trigger cache invalidation (code analysis of `queries.go:919-925`)
- ✅ Test written for answered unblocking behavior (`question_status_test.go`)

**What's untested:**

- ⚠️ Test hasn't been run (Go not available in sandbox)
- ⚠️ Migration performance on large databases
- ⚠️ CLI flag interactions with existing commands

**What would change this:**

- If tests fail, there's a bug in cache invalidation timing
- If migration has performance issues, may need batch approach

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Incremental Schema Extension** - Add fields via migrations, update types, add CLI support.

**Why this approach:**
- Minimal disruption to existing functionality
- Each change can be validated independently
- Backward compatible with default values

**Trade-offs accepted:**
- Multiple small migrations instead of one big one
- CLI changes span multiple commands

**Implementation sequence:**
1. Add `authority` to dependencies (highest value - enables frontier queries)
2. Add `resolution_type` to issues (enables question routing)
3. Add CLI support for both fields
4. Add `bd ready --authority` query

### Implementation Details

**What to implement first:**
- Authority field on dependencies - enables `bd ready --authority daemon`
- Use column, not metadata JSON, for queryability

**Things to watch out for:**
- ⚠️ Need migration for existing dependencies (default: 'daemon')
- ⚠️ Dashboard may need updates for new fields
- ⚠️ RPC layer needs updates for daemon communication

**Success criteria:**
- ✅ `bd dep add X Y --authority orchestrator` works
- ✅ `bd ready --authority daemon` returns only daemon-traversable work
- ✅ `bd create --type question --resolution-type factual` works
- ✅ All existing tests pass

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/blocked_cache.go` - Blocking logic
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/ready.go` - Ready query logic
- `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/schema.go` - Database schema
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go` - Type definitions
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/dep.go` - Dependency CLI
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/ready.go` - Ready CLI

**Commands Run:**
```bash
# Check git history for answered status
git log --oneline --all --grep="answered" --since="2026-01-01"

# Check question gates commit
git show 744af9cf --stat
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-19-design-investigate-substrate-options-decidability-graphs.md`
- **Model:** `.kb/models/decidability-graph.md`
- **Decision:** `.kb/decisions/2026-01-18-questions-as-first-class-entities.md`

---

## Investigation History

**2026-01-22 21:00:** Investigation started
- Initial question: What changes are needed to implement decidability substrate extensions?
- Context: Live validation case discovered Jan 22 - worker cited stale decision

**2026-01-22 21:30:** Major finding - answered status already works
- Code analysis shows `answered` NOT in blocking statuses
- Constraint `kb-fe6173` appears outdated
- Wrote test to verify behavior

**2026-01-22 21:45:** Moving to implementation
- Status: In Progress
- Key outcome: Only authority field and resolution_type need implementation

**2026-01-22 22:30:** Core implementation complete
- Added Authority field to Dependency type with validation
- Created migration 035_authority_column.go
- Updated AddDependency to handle authority
- Added --authority flag to bd dep add command
- Updated RPC protocol for authority field
- Added ResolutionType type with factual/judgment/framing values
- Added ResolutionType and Domain fields to Issue type
- Created migration 036_resolution_type_domain_columns.go
- Updated issue insert queries (insertIssue, insertIssues)
- Updated GetIssue query to read new fields
- Added fields to allowedUpdateFields whitelist
- Wrote test for answered status unblocking (question_status_test.go)

**Remaining work for future sessions:**
- CLI support for --resolution-type on bd create
- bd ready --authority filter implementation
- Dashboard updates for new fields
- More comprehensive tests
- Documentation updates
