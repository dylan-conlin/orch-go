<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Headless spawn works for agent execution but auto-generated untracked issue IDs don't persist to beads database, breaking progress tracking.

**Evidence:** Agent spawned successfully, executed commands, and created artifacts; `bd comment` failed with "issue not found"; `bd list` confirmed issue ID absent from database.

**Knowledge:** Headless spawns with `--no-track` (or auto-untracked mode) reference non-existent beads issues in spawn context, creating orchestrator blind spot during execution.

**Next:** Escalate to orchestrator - either fix untracked issue persistence OR update spawn context to not reference beads for untracked spawns.

**Confidence:** High (85%) - reproduced issue directly, clear evidence, but didn't investigate root cause in spawn code.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Test Headless Spawn - Say Hello

**Question:** Does headless spawn work correctly, and does beads tracking function as expected?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Spawned agent (headless test)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Headless spawn creates session successfully

**Evidence:** Successfully spawned as a headless agent, working directory correctly set to `/Users/dylanconlin/Documents/personal/orch-go`, investigation file created at expected path.

**Source:**

- Command: `pwd` returned `/Users/dylanconlin/Documents/personal/orch-go`
- Command: `kb create investigation test-headless-spawn-say-hello` successfully created investigation file
- File created: `.kb/investigations/2025-12-21-inv-test-headless-spawn-say-hello.md`

**Significance:** Headless spawn mode is functional - agent is properly initialized and can execute commands in the correct project directory.

---

### Finding 2: Beads tracking issue ID doesn't exist in database

**Evidence:** The spawn context references issue `orch-go-untracked-1766305160`, but attempting to comment fails with "issue orch-go-untracked-1766305160 not found". Running `bd list` shows 168+ issues in the database, but this ID is not among them.

**Source:**

- Command: `bd comment orch-go-untracked-1766305160 "Phase: Planning..."` → Error: "issue orch-go-untracked-1766305160 not found"
- Command: `bd list` shows all issues in database (searched output, ID not present)
- The issue ID format `orch-go-untracked-1766305160` suggests auto-generated untracked issue

**Significance:** Headless spawns with auto-tracking may not properly persist the issue to the beads database, breaking the beads progress tracking protocol. This means orchestrators cannot monitor progress via `bd show <id>`.

---

### Finding 3: Investigation file creation works despite tracking failure

**Evidence:** Despite beads tracking failing, the `kb create investigation` command worked correctly and created the investigation artifact at the expected path with proper template structure.

**Source:**

- Successfully created: `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-test-headless-spawn-say-hello.md`
- Template properly filled with D.E.K.N. structure and investigation sections

**Significance:** The investigation workflow can proceed independently of beads tracking, but without beads comments, orchestrators lose real-time progress visibility.

---

## Synthesis

**Key Insights:**

1. **Headless spawn is functional but has tracking gaps** - The core headless spawn mechanism works (agent spawns, correct directory, can execute commands), but the beads progress tracking integration has issues with auto-generated issue IDs not being persisted to the database.

2. **Workflow can proceed independently of beads tracking** - Investigation file creation and agent execution work correctly even when beads tracking fails, demonstrating that the workflow doesn't hard-depend on beads but loses orchestrator visibility.

3. **"Untracked" issue IDs aren't actually tracked** - The issue ID pattern `orch-go-untracked-{timestamp}` suggests these are meant to be temporary/auto-generated, but they're referenced in spawn context as if they exist in beads, creating a mismatch.

**Answer to Investigation Question:**

Headless spawn works correctly for agent execution (Finding 1), and I successfully said "Hello" via test command. However, beads tracking does NOT function as expected (Finding 2) - the auto-generated issue ID `orch-go-untracked-1766305160` doesn't exist in the beads database, preventing progress comments from being recorded. This breaks the orchestrator's ability to monitor progress via `bd show` while the agent is running. The investigation workflow can still complete (Finding 3), but without real-time progress visibility.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Direct reproduction of the issue with clear error messages and verification via beads database query. The symptom is certain, but I didn't investigate the root cause in spawn.go code, so there's minor uncertainty about the exact implementation fix needed.

**What's certain:**

- ✅ Headless spawn creates functional agent session (verified via pwd, kb create, echo test)
- ✅ Issue ID `orch-go-untracked-1766305160` doesn't exist in beads database (verified via bd list)
- ✅ Beads comment protocol fails for untracked spawns (verified via error message)

**What's uncertain:**

- ⚠️ Exact code location in spawn.go where untracked issue ID is generated
- ⚠️ Whether this affects regular tracked spawns or only --no-track mode
- ⚠️ If there's an intentional reason for including beads protocol in untracked spawn context

**What would increase confidence to Very High (95%+):**

- Read spawn.go code to understand issue ID generation logic
- Test regular tracked spawn to confirm beads protocol works correctly
- Review git history for when untracked mode was added and original intent

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Conditional Beads Tracking** - Only include beads tracking protocol in spawn context when a real beads issue exists.

**Why this approach:**

- Prevents agents from attempting to comment on non-existent issues
- Maintains tracking for normal spawns while gracefully handling untracked mode
- Minimal code change (conditional template inclusion in spawn.go)

**Trade-offs accepted:**

- Untracked spawns lose progress visibility (but that's expected for untracked mode)
- Two different spawn context templates to maintain

**Implementation sequence:**

1. Update spawn.go to check if issue exists in beads before including tracking section
2. Create spawn-context-untracked.md template without beads protocol
3. Test both tracked and untracked spawn modes

### Alternative Approaches Considered

**Option B: Always create real beads issue**

- **Pros:** Uniform tracking for all spawns
- **Cons:** Clutters beads database with throwaway test issues; defeats purpose of --no-track flag
- **When to use instead:** Never - --no-track exists for a reason

**Option C: Make untracked IDs actually work**

- **Pros:** Maintains progress visibility even for untracked spawns
- **Cons:** Complex implementation (in-memory registry?); violates semantic meaning of "untracked"
- **When to use instead:** If orchestrator truly needs progress visibility for all spawns regardless of tracking preference

**Rationale for recommendation:** Option A respects the semantic intent of "untracked" while preventing agent confusion from referencing non-existent issues.

---

### Implementation Details

**What to implement first:**

- Add conditional logic in spawn.go to detect untracked spawns (check issue ID pattern or --no-track flag)
- Create separate spawn context template for untracked mode without beads protocol section
- Update spawn context generation to use appropriate template

**Things to watch out for:**

- ⚠️ Issue ID format might change (currently `orch-go-untracked-{timestamp}`) - make detection robust
- ⚠️ Agents spawned before this fix will still reference non-existent issues - acceptable for tests
- ⚠️ Ensure orchestrator knows untracked spawns won't have beads progress visibility

**Areas needing further investigation:**

- How are untracked issue IDs generated? (search spawn.go for timestamp generation)
- Should untracked spawns even attempt to create issues or skip entirely?
- Is there value in logging untracked spawn progress somewhere else (events.jsonl only)?

**Success criteria:**

- ✅ Untracked spawns don't reference beads in spawn context
- ✅ Tracked spawns continue to work with beads protocol
- ✅ Test both modes: `orch spawn --no-track` and `orch spawn --issue <real-id>`

---

## References

**Files Examined:**

- `.orch/workspace/og-inv-test-headless-spawn-21dec/SPAWN_CONTEXT.md` - Spawn context with beads issue reference
- `.kb/investigations/2025-12-21-inv-test-headless-spawn-say-hello.md` - This investigation file

**Commands Run:**

```bash
# Report phase to beads (failed - issue not found)
bd comment orch-go-untracked-1766305160 "Phase: Planning - Testing headless spawn workflow"

# Verify working directory
pwd

# Create investigation file
kb create investigation test-headless-spawn-say-hello

# List all beads issues to verify absence
bd list

# Test basic functionality
echo "Hello from headless spawn test! This agent is working correctly." > /tmp/headless-test-output.txt && cat /tmp/headless-test-output.txt
```

**External Documentation:**

- None

**Related Artifacts:**

- **Spawn Context:** `.orch/workspace/og-inv-test-headless-spawn-21dec/SPAWN_CONTEXT.md` - Contains reference to non-existent issue ID

---

## Investigation History

**2025-12-21:** Investigation started

- Initial question: Does headless spawn work correctly, and does beads tracking function as expected?
- Context: Testing headless spawn mode as requested via spawn command

**2025-12-21:** Discovered beads tracking mismatch

- Found that spawn context references issue `orch-go-untracked-1766305160` but issue doesn't exist in beads database
- Confirmed via `bd list` and `bd comment` error message

**2025-12-21:** Investigation completed

- Final confidence: High (85%)
- Status: Complete
- Key outcome: Headless spawn works for execution but auto-generated untracked issue IDs break beads progress tracking protocol.

---

## Self-Review

- [x] Real test performed (echo test to verify agent functionality)
- [x] Conclusion from evidence (`bd comment` error + `bd list` verification)
- [x] Question answered (headless spawn works, beads tracking doesn't)
- [x] File complete (all sections filled, D.E.K.N. summary complete)
- [x] Test is real (ran actual commands: pwd, bd comment, bd list, echo)
- [x] Evidence concrete (error messages, command outputs, file paths)
- [x] Conclusion factual (based on observed beads database query results)
- [x] No speculation (recommendations based on findings, not guesses)
- [x] D.E.K.N. filled (summary section complete at top)

**Self-Review Status:** PASSED

**Discovered Work:**
- Bug found: Untracked spawn contexts reference non-existent beads issues
- Recommendation: Create beads issue for fixing spawn context template logic

