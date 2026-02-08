<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Untracked spawns generate fake beads IDs that cause `bd comment` failures - spawn context instructs agents to use `bd comment` with non-existent issue IDs.

**Evidence:** Running `bd comment orch-go-untracked-1766774810 "..."` returns "issue not found" error; beads ID pattern `{project}-untracked-{timestamp}` is synthetic and never created in beads database.

**Knowledge:** The spawn context template doesn't distinguish between tracked and untracked spawns - it always includes `bd comment` instructions regardless of whether the beads ID is real or fake.

**Next:** Either (1) conditionally skip beads instructions in spawn context for untracked spawns, or (2) always create a real beads issue even for "untracked" spawns with a special label.

**Confidence:** High (90%) - Direct testing confirms the bug; code path is clear.

---

# Investigation: Test Orch Spawn Context

**Question:** Does the orch spawn context work correctly for agents, particularly around beads tracking?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Untracked spawn IDs are synthetic and don't exist in beads

**Evidence:** 
- The spawn context contains BeadsID: `orch-go-untracked-1766774810`
- Running `bd show orch-go-untracked-1766774810` returns: "no issue found matching"
- Running `bd comment orch-go-untracked-1766774810 "Phase: Planning..."` fails with: "issue not found"

**Source:** 
- `cmd/orch/main.go:1671-1673` - ID generation for untracked spawns
- Test command: `bd show orch-go-untracked-1766774810`

**Significance:** Agents spawned with `--no-track` (or when beads creation fails) receive instructions to use `bd comment` with IDs that don't exist, causing immediate failures on first tool call.

---

### Finding 2: Spawn context template doesn't conditionally handle untracked spawns

**Evidence:**
The template at `pkg/spawn/context.go:18-196` always includes:
```
🚨 CRITICAL - FIRST 3 ACTIONS:
You MUST do these within your first 3 tool calls:
1. Report via `bd comment {{.BeadsID}} "Phase: Planning - [brief description]"`
```

There's no conditional logic to skip these instructions when `BeadsID` is an untracked pattern.

**Source:**
- `pkg/spawn/context.go:32-39` - Template always includes `bd comment` instructions
- `pkg/spawn/context.go:116-145` - Beads progress tracking section (unconditional)

**Significance:** The template design assumes all spawns have valid beads IDs, but untracked spawns have synthetic IDs that beads doesn't recognize.

---

### Finding 3: Untracked ID detection exists but isn't used during spawn

**Evidence:**
The system knows how to detect untracked IDs:
```go
// cmd/orch/review.go:271-274
func isUntrackedBeadsID(beadsID string) bool {
    return strings.Contains(beadsID, "-untracked-")
}
```

This function is used by `orch review` to filter out untracked agents, but the spawn context generation doesn't use this pattern to conditionally exclude beads instructions.

**Source:**
- `cmd/orch/review.go:271-274` - Detection function exists
- `cmd/orch/review_test.go:432-444` - Test cases for detection

**Significance:** The infrastructure to detect untracked spawns exists; it just needs to be applied during context generation.

---

## Synthesis

**Key Insights:**

1. **Design gap in spawn context generation** - The spawn context template was designed with the assumption that all spawns have valid beads issues. When `--no-track` was added, the template wasn't updated to handle the case where beads operations will fail.

2. **Untracked is intentional but incomplete** - The `--no-track` feature is documented and intentional (for ad-hoc work without beads tracking), but the implementation only generates a fake ID without adjusting the agent's instructions.

3. **Fix is straightforward** - The `isUntrackedBeadsID` function already exists; applying it in the template generation to conditionally skip beads instructions would resolve the issue.

**Answer to Investigation Question:**

The orch spawn context has a bug when used with untracked spawns (either via `--no-track` flag or when beads issue creation fails silently): the spawn context instructs agents to use `bd comment` with fake beads IDs that don't exist. This causes immediate failures when agents attempt their first required action ("Report via bd comment..."). The fix is to conditionally exclude beads-related instructions when the BeadsID contains the `-untracked-` pattern.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Direct testing confirms the failure mode. Code analysis shows the root cause clearly. The gap between expectation (beads instructions work) and reality (they fail for untracked) is unambiguous.

**What's certain:**

- ✅ Untracked BeadsIDs (`*-untracked-*`) cause `bd comment` failures (tested directly)
- ✅ The spawn context template doesn't conditionally handle untracked spawns (code analysis confirms)
- ✅ Detection function for untracked IDs exists and is tested (`isUntrackedBeadsID`)

**What's uncertain:**

- ⚠️ Whether this spawn was intentionally untracked or if beads creation failed
- ⚠️ How often beads issue creation silently fails (could mask the bug)
- ⚠️ Whether the fix should skip all beads instructions or just the comment ones

**What would increase confidence to Very High (95%+):**

- Review logs from the spawn that created this agent to confirm if `--no-track` was used
- Test the proposed fix with both tracked and untracked spawns
- Review if there are other beads-dependent instructions that need conditional handling

---

## Implementation Recommendations

### Recommended Approach ⭐

**Conditional template rendering** - Detect untracked spawns and render alternative instructions that skip beads operations.

**Why this approach:**
- Minimal change to existing code structure
- Uses existing `isUntrackedBeadsID` pattern
- Preserves beads tracking for normal spawns

**Trade-offs accepted:**
- Untracked agents lose phase tracking via beads
- Orchestrator has reduced visibility into untracked agent progress

**Implementation sequence:**
1. Add `IsUntracked bool` field to `contextData` struct in `pkg/spawn/context.go`
2. Detect untracked pattern before template execution
3. Add conditionals in template to skip beads instructions when untracked

### Alternative Approaches Considered

**Option B: Always create real beads issues**
- **Pros:** All spawns have working beads tracking
- **Cons:** Defeats the purpose of `--no-track`; creates beads clutter
- **When to use instead:** If we decide untracked spawns shouldn't exist

**Option C: Create beads issue but label as "ephemeral"**
- **Pros:** Tracking works; issue can be auto-cleaned
- **Cons:** More complex; requires beads changes
- **When to use instead:** If visibility into all agents is required

**Rationale for recommendation:** Option A (conditional template) is the simplest fix that preserves the original intent of `--no-track` while preventing agent failures.

---

### Implementation Details

**What to implement first:**
- Add `IsUntracked` field to `contextData` struct
- Populate field using `strings.Contains(cfg.BeadsID, "-untracked-")`
- Add `{{if not .IsUntracked}}...{{end}}` around beads instructions

**Things to watch out for:**
- ⚠️ Ensure both "CRITICAL - FIRST 3 ACTIONS" and "BEADS PROGRESS TRACKING" sections are conditionally rendered
- ⚠️ Consider what agents should do instead of `bd comment` for progress reporting
- ⚠️ Update the skill self-review checklists that reference beads

**Areas needing further investigation:**
- Should untracked agents still create investigation files (currently instructed to)?
- Is there a fallback progress mechanism for untracked agents?
- Should `orch complete` handle untracked agents differently?

**Success criteria:**
- ✅ Untracked spawns don't include `bd comment` instructions
- ✅ Tracked spawns continue to work as before
- ✅ Agents spawned with `--no-track` don't fail on first action

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - Spawn context template and generation logic
- `cmd/orch/main.go:1661-1717` - BeadsID determination and issue creation
- `cmd/orch/review.go:271-274` - Untracked ID detection function
- `.orch/workspace/og-inv-test-orch-spawn-26dec/SPAWN_CONTEXT.md` - This agent's spawn context

**Commands Run:**
```bash
# Verify untracked issue doesn't exist
bd show orch-go-untracked-1766774810
# Result: no issue found matching "orch-go-untracked-1766774810"

# Attempt to comment on untracked issue
bd comment orch-go-untracked-1766774810 "Phase: Planning..."
# Result: issue not found

# Count untracked pattern references in codebase
rg "\-untracked\-" --type go -c
# Result: 4 files with references
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-inv-test-orch-spawn-26dec/` - This investigation's spawn context

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-26 ~00:00:** Investigation started
- Initial question: Does orch spawn context work correctly?
- Context: Agent was spawned to test spawn context; immediately encountered `bd comment` failure

**2025-12-26 ~00:15:** Key finding discovered
- Untracked spawn IDs are synthetic and cause beads failures
- Spawn context template doesn't handle this case

**2025-12-26 ~00:30:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Bug confirmed - untracked spawns include beads instructions that fail
