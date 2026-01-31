# Decision: Decidability Graph Substrate Options

**Date:** 2026-01-30
**Status:** Accepted
**Context:** Synthesized from investigation 2026-01-19-design-investigate-substrate-options-decidability-graphs.md

## Summary

Extend beads fork with three targeted additions for decidability semantics: (1) wire `answered` status to unblock dependencies, (2) add `authority` field to dependency edges (daemon/orchestrator/dylan), (3) add `resolution_type` to questions (factual/judgment/framing). Beads provides 85% of decidability substrate needs; gaps are semantic, not structural. Rebuild cost for alternatives: 2-4x higher with no clear advantage.

## The Problem

Decidability graphs require substrate that:
- Tracks work items (issues, questions, gates)
- Models dependencies (blocking relationships)
- Calculates frontier (what's decidable now)
- Enforces gates (questions block dependent work)

Beads provides most of this, but has three semantic gaps:

**Gap 1: answered status doesn't unblock**
- Questions need: Open→Investigating→Answered→Closed
- Current: Only `closed` status unblocks dependencies
- Constraint kb-fe6173: `answered` status exists but doesn't unblock

**Gap 2: No authority encoding on edges**
- Dependencies are just "blocks" or "parent-child"
- Decidability needs: daemon/orchestrator/dylan authority levels
- Edge traversal rules depend on WHO is traversing
- Example: daemon can traverse factual question blocks, orchestrator can traverse judgment blocks

**Gap 3: No resolution typing for questions**
- Questions have type=question but no subtype
- Decidability needs: factual/judgment/framing distinction
- Determines WHO can resolve the question
- Enables routing: factual→daemon, judgment→orchestrator, framing→dylan

## The Decision

### Extend Beads Fork with Three Additions

**Option chosen:** Extend beads (not replace with markdown files or build purpose-designed substrate)

**Why beads:**
- 217K lines of existing, tested code
- Efficient queries, concurrent access safety, dashboard integration
- Frontier calculation already implemented (`bd ready`)
- RPC client, CLI fallback, dependency resolution machinery exists

**Implementation: Three targeted changes in beads**

**1. Wire `answered` Status to Unblock Dependencies**

Modify `GetBlockingDependencies()` to treat `answered` like `closed`:

```go
// Current: only checks closed
if dep.Status == "closed" { skip }

// Updated: check answered OR closed
if dep.Status == "answered" || dep.Status == "closed" { skip }
```

Effect: Questions can be marked `answered` and unblock dependent work without forcing immediate close.

**2. Add Authority Field to Dependency Edges**

Schema addition:
```go
type Dependency struct {
    From      string  // Issue ID
    To        string  // Issue ID
    Type      string  // "blocks", "parent-child"
    Authority string  // "daemon", "orchestrator", "dylan"
}
```

CLI usage:
```bash
bd dep add <epic-id> <question-id> --authority orchestrator
```

Query:
```bash
bd ready --authority daemon  # Only daemon-traversable work
```

Default: `authority = "daemon"` for backward compatibility

**3. Add resolution_type to Questions**

Schema addition to question-type issues:
```go
type Question struct {
    // ... existing fields
    ResolutionType string  // "factual", "judgment", "framing", ""
}
```

CLI usage:
```bash
bd create --type question --resolution-type judgment "Should we build X?"
bd list --type question --resolution-type factual
```

Optional field, nullable. Can be set after creation.

### Alternatives Rejected

**Option B: Directory of Markdown Files**
- Cost: 2,000-4,000 lines new code
- Loses: Efficient queries, concurrent access, existing dashboard integration
- Gains: Simplicity, session-amnesia friendly
- Rejected: Rebuilds beads machinery for philosophical purity, no practical advantage

**Option C: Purpose-Designed Substrate**
- Cost: 3,000-6,000 lines new code
- Includes: All of Option B PLUS SQLite/hybrid storage, graph traversal, migration
- Gains: First-class decidability concepts
- Rejected: 2-4x cost with no evidence beads design is fundamentally wrong

## Why This Design

### Principle: Compose Over Monolith

From principles.md: "Extend Unix-style with composable utilities, not monolithic frameworks."

Beads is Unix-style: one tool for work tracking. Extending with decidability semantics follows the same pattern - one concept per dimension.

### Principle: Evolve by Distinction

The gaps are SEMANTIC (what data means), not STRUCTURAL (how it's stored). Beads stores:
- Issues ✅
- Dependencies ✅
- Statuses ✅
- Frontier queries ✅

What's missing is interpretation layer: 
- What `answered` means (unblocks)
- What edges encode (authority)
- What question subtypes mean (resolution type)

Adding semantics to existing structure vs. rebuilding structure.

### Key Insight: Frontier-Awareness is Substrate Concern

`bd ready` already implements frontier calculation:
- Excludes blocked issues
- Respects status
- Handles parent-child vs blocks distinction

Moving this to orchestrator would duplicate graph traversal logic. Right fix: extend beads with authority-aware queries.

### Lesson: GLM Research Worker Validated Need

From investigation 2026-01-22: Research worker cited stale decision (Jan 9) when superseded (Jan 18). Evidence: **Decisions are decidability nodes** with supersedes edges, authority levels, frontier queries.

Decision lifecycle is validated use case, not theoretical.

## Trade-offs

**Accepted:**
- Beads fork divergence (acceptable: already fork for custom features)
- Three targeted changes vs "clean" design (acceptable: perfect is enemy of good)
- Schema migrations may need careful handling

**Rejected:**
- Hybrid approach (beads storage + orch-go interpretation): Two sources of truth, duplicated traversal logic
- Waiting for perfect design: Current gaps block decidability implementation

## Constraints

1. **Authority field defaults to "daemon"** - Backward compatibility for existing dependencies
2. **answered status only for questions** - Other issue types use standard lifecycle (open→in_progress→closed)
3. **Resolution type is optional** - Can create question without resolution type, set later
4. **bd ready respects authority** - Query filtering by authority level must be efficient

## Implementation Notes

**Timeline:**
- Day 1: Wire `answered` unblocking
- Day 2: Add authority field
- Day 3: Add resolution_type field

**Phase 1: Wire answered Unblocking (Highest Value)**

File: beads fork `internal/storage/sqlite/dependencies.go`
```go
// Modify GetBlockingDependencies()
// Change: only closed → answered OR closed
```

Tests: blocked-by-question unblocks when answered

**Phase 2: Add Authority Field**

Files:
- Schema: `internal/storage/sqlite/schema.sql` - Add authority column
- CLI: `cmd/bd/dep.go` - Add --authority flag
- Query: `cmd/bd/ready.go` - Add --authority filter
- API: `internal/rpc/server.go` - Include authority in responses

**Phase 3: Add resolution_type Field**

Files:
- Schema: `internal/storage/sqlite/schema.sql` - Add resolution_type column
- CLI: `cmd/bd/create.go` - Add --resolution-type flag for questions
- Query: `cmd/bd/list.go` - Add --resolution-type filter
- API: Include in question responses

**Success Criteria:**
- `bd ready --authority daemon` returns only daemon-traversable work
- Closing question with `answered` status unblocks dependent work
- Questions can have `resolution_type` set and queried
- Dashboard shows "blocked by question" distinct from "blocked by work"

**Migration Path:**
- Existing dependencies: default authority = "daemon"
- Existing questions: resolution_type = null (can be set later)
- No breaking changes to existing workflows

## References

**Investigation:**
- `.kb/investigations/2026-01-19-design-investigate-substrate-options-decidability-graphs.md` - Substrate options analysis
- `.kb/investigations/2026-01-22-inv-research-glm-ai-orchestration-context.md` - GLM research validation

**Files:**
- `pkg/beads/interface.go` - BeadsClient interface (10 operations used)
- `pkg/beads/types.go:195-224` - GetBlockingDependencies implementation

**Models:**
- `.kb/models/decidability-graph.md:136-146` - Node taxonomy
- `.kb/models/decidability-graph.md:386-396` - Friction discovered

**Decisions:**
- `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` - Question entity foundation

**Principles:**
- Compose Over Monolith - `~/.kb/principles.md`
- Evolve by Distinction - Implicit in semantic vs structural gap analysis
- Local-First - Beads is SQLite-based, no external dependencies
- Share Patterns Not Tools - Beads is Dylan's fork, sharing pattern of issue tracking
