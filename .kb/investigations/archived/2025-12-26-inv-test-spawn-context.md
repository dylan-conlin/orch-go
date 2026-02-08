<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Spawn context template contains a bug - `bd comment` instructions are included for untracked spawns where the beads issue doesn't exist, causing failures.

**Evidence:** Spawned with `--no-track`, spawn context contains `bd comment orch-go-untracked-1766774790` but `bd show` returns "issue not found". Template at `pkg/spawn/context.go:16-196` has no conditional for untracked IDs.

**Knowledge:** Synthetic untracked beads IDs (format `{project}-untracked-{timestamp}`) are intentionally never created in beads - they're internal tracking only. Spawn context template needs conditional logic to skip beads instructions for untracked spawns.

**Next:** Create beads issue to fix template - add conditional in `SpawnContextTemplate` to omit beads tracking instructions when BeadsID contains "-untracked-".

**Confidence:** High (90%) - Verified by actual test of `bd comment` failing, traced to source code.

---

# Investigation: Test Spawn Context

**Question:** Does the spawn context template generate correct and complete spawn contexts for investigation skills, including proper handling of untracked spawns?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Untracked spawn contexts include invalid beads instructions

**Evidence:** 
- This spawn was created with `--no-track` flag
- Spawn context at `.orch/workspace/og-inv-test-spawn-context-26dec/SPAWN_CONTEXT.md` contains:
  ```
  bd comment orch-go-untracked-1766774790 "Phase: Planning..."
  ```
- Running `bd show orch-go-untracked-1766774790` returns: "Error resolving ID: no issue found"
- Running `bd comment orch-go-untracked-1766774790 "test"` returns: "issue not found"

**Source:** 
- `.orch/workspace/og-inv-test-spawn-context-26dec/SPAWN_CONTEXT.md` - lines 12, 23, 50, 94, etc.
- `pkg/spawn/context.go:16-196` - SpawnContextTemplate has no conditional for untracked IDs
- `cmd/orch/main.go:1672` - Synthetic ID generation: `fmt.Sprintf("%s-untracked-%d", projectName, time.Now().Unix())`

**Significance:** Agents spawned with `--no-track` are instructed to run commands that will always fail. The "🚨 CRITICAL - FIRST 3 ACTIONS" section tells agents to report via `bd comment` within first 3 tool calls, but this always fails for untracked spawns.

---

### Finding 2: Spawn context structure is otherwise complete

**Evidence:**
Verified spawn context contains all expected sections:
- ✅ TASK declaration
- ✅ SPAWN TIER (full)
- ✅ CRITICAL FIRST 3 ACTIONS section
- ✅ SESSION COMPLETE PROTOCOL
- ✅ CONTEXT section
- ✅ PROJECT_DIR (correct path)
- ✅ SESSION SCOPE
- ✅ AUTHORITY section (decides vs escalates)
- ✅ DELIVERABLES section
- ✅ STATUS UPDATES guidance
- ✅ BEADS PROGRESS TRACKING section
- ✅ SKILL GUIDANCE (investigation skill fully embedded)
- ✅ CONTEXT AVAILABLE references
- ✅ FINAL STEP reminder

**Source:** `.orch/workspace/og-inv-test-spawn-context-26dec/SPAWN_CONTEXT.md` - 429 lines

**Significance:** The template structure is comprehensive. The only issue is the unconditional inclusion of beads-specific instructions.

---

### Finding 3: Investigation skill is properly embedded

**Evidence:**
The investigation skill (lines 119-406 in spawn context) includes:
- Skill metadata and summary
- "The One Rule" - cannot conclude without testing
- Evidence Hierarchy table
- Workflow steps
- D.E.K.N. summary pattern
- Template example
- Common failure examples
- Self-Review checklist
- Leave it Better section
- Completion checklist

**Source:** `pkg/spawn/context.go:147-158` - SkillContent embedding logic

**Significance:** Skill embedding works correctly. Agents receive full skill guidance inline.

---

## Synthesis

**Key Insights:**

1. **Untracked spawns violate agent instructions** - The spawn context tells agents to report phase via `bd comment` as a critical first action, but this command fails for untracked spawns. This creates immediate confusion and wasted tool calls.

2. **Template needs conditional logic** - The fix requires detecting untracked BeadsIDs (containing "-untracked-") and either omitting beads instructions or replacing them with alternative tracking guidance.

3. **Workspace metadata is correctly written** - The `.tier` file (containing "full") and `.spawn_time` file work correctly for untracked spawns.

**Answer to Investigation Question:**

The spawn context template generates mostly correct spawn contexts, but has a significant bug: beads tracking instructions are included unconditionally, even for untracked spawns where the beads issue doesn't exist. This causes agents to fail on their first critical action. The fix requires adding conditional logic in `SpawnContextTemplate` to detect untracked IDs and provide alternative instructions (e.g., "Untracked spawn - beads tracking disabled, proceed with workspace-only status updates").

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Tested the actual behavior by attempting `bd comment` with the untracked ID - confirmed failure. Traced to source code to verify no conditional exists.

**What's certain:**

- ✅ `bd comment` fails for untracked beads IDs (tested)
- ✅ Template has no conditional for untracked IDs (verified in source)
- ✅ Untracked ID format is `{project}-untracked-{timestamp}` (verified in source)

**What's uncertain:**

- ⚠️ Whether this is considered a bug or intentional design (may be accepted that --no-track disables monitoring)
- ⚠️ What alternative instructions should be provided for untracked spawns

**What would increase confidence to Very High (95%+):**

- Confirmation from orchestrator/owner that this is indeed a bug to fix
- Review of intended design for --no-track workflows

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Add conditional in SpawnContextTemplate to handle untracked spawns**

**Why this approach:**
- Fixes the immediate issue of failing commands
- Minimal code change - add template conditional
- Maintains full tracking for normal spawns

**Trade-offs accepted:**
- Adds template complexity
- Untracked spawns lose beads-based monitoring (but that's the purpose of --no-track)

**Implementation sequence:**
1. Add `IsUntracked` field to `contextData` struct
2. Set based on BeadsID containing "-untracked-"
3. Wrap beads instructions in `{{if not .IsUntracked}}...{{end}}`
4. Add alternative instructions for untracked spawns

### Alternative Approaches Considered

**Option B: Fail spawn with --no-track if beads instructions would be generated**
- **Pros:** Forces explicit choice about tracking
- **Cons:** Too restrictive, breaks existing --no-track workflows
- **When to use instead:** If all spawns should require tracking

**Option C: Don't include beads instructions in spawn context at all**
- **Pros:** Simpler template
- **Cons:** Removes valuable phase tracking for tracked spawns
- **When to use instead:** If beads tracking is moved elsewhere

**Rationale for recommendation:** Option A is least disruptive and directly addresses the bug.

---

### Implementation Details

**What to implement first:**
- Add `IsUntracked` to `contextData` struct in `pkg/spawn/context.go`
- Compute value in `GenerateContext` function
- Add conditional wrapping in template

**Things to watch out for:**
- ⚠️ Template syntax - Go templates use `{{if not .IsUntracked}}`
- ⚠️ Ensure FINAL STEP section also gets conditional treatment
- ⚠️ Consider what guidance untracked agents should get instead

**Success criteria:**
- ✅ `bd comment` instructions not present when BeadsID contains "-untracked-"
- ✅ Untracked spawn context still has valid instructions for workspace-only tracking
- ✅ Tracked spawn context unchanged

---

## References

**Files Examined:**
- `.orch/workspace/og-inv-test-spawn-context-26dec/SPAWN_CONTEXT.md` - Actual spawn context being tested
- `pkg/spawn/context.go` - Template definition and generation
- `cmd/orch/main.go:1670-1682` - Untracked ID generation logic
- `cmd/orch/review.go:271-274` - Untracked ID detection helper

**Commands Run:**
```bash
# Attempted beads comment - failed
bd comment orch-go-untracked-1766774790 "Phase: Planning"
# Error: issue not found

# Verified issue doesn't exist
bd show orch-go-untracked-1766774790
# Error resolving ID: no issue found

# Created investigation file
kb create investigation test-spawn-context
# Created: .kb/investigations/2025-12-26-inv-test-spawn-context.md
```

**Related Artifacts:**
- None - this is a new investigation

---

## Investigation History

**2025-12-26 10:46:** Investigation started
- Initial question: Does spawn context work correctly for investigation skill?
- Context: Meta-test of spawn context template itself

**2025-12-26 10:47:** First critical finding
- Discovered `bd comment` fails because untracked beads ID doesn't exist
- Traced to source code to confirm no conditional logic

**2025-12-26 10:50:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Bug found - beads instructions included for untracked spawns

---

## Self-Review

- [x] Real test performed (not code review) - ran `bd comment` and `bd show` commands
- [x] Conclusion from evidence (not speculation) - based on actual command failures
- [x] Question answered - spawn context has a bug with untracked spawns
- [x] File complete - all sections filled

**Self-Review Status:** PASSED
