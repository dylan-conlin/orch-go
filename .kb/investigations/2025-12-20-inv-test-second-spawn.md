**TLDR:** Question: Can orch-go spawn multiple agents concurrently? Answer: Yes - fire-and-forget tmux spawn design successfully supports concurrent agent execution with separate windows/workspaces. High confidence (85%) - directly tested with 2 concurrent spawns, though upper limits untested.

---

# Investigation: Test Second Spawn - Concurrent Agent Support

**Question:** Can orch-go successfully spawn a second (or nth) agent while other agents are already running in tmux windows?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: orch-go successfully spawns concurrent agents

**Evidence:**
- First spawn command created window 7: `og-inv-test-concurrent-spawn-20dec`
- Workspace created at `.orch/workspace/og-inv-test-concurrent-spawn-20dec/` with SPAWN_CONTEXT.md
- OpenCode session started successfully and agent began executing
- Second spawn command created window 9: `og-inv-test-third-concurrent-20dec` while window 7 was still active
- Both windows show active OpenCode sessions running concurrently

**Source:**
- Command: `./orch-go spawn investigation "test concurrent spawn capability"`
- Command: `./orch-go spawn investigation "test third concurrent spawn"`
- tmux output: `tmux list-windows -t workers-orch-go`
- tmux capture: `tmux capture-pane -t workers-orch-go:7 -p`

**Significance:** Confirms that orch-go's fire-and-forget tmux spawn design allows multiple agents to be spawned and run concurrently without blocking.

---

### Finding 2: Beads ID parsing issue confirmed

**Evidence:**
Both spawn commands showed warning: `Warning: failed to update beads issue status: failed to update issue status: exit status 1: Error resolving ID open: operation failed: failed to resolve ID: no issue found matching "open"`

Spawn output showed: `Beads ID: open` instead of proper issue ID format (e.g., `orch-go-xyz`)

**Source:**
- stderr output from `./orch-go spawn investigation` commands
- Known bug orch-go-c4r: "Fix bd create output parsing - captures 'open' instead of issue ID"

**Significance:** While spawn functionality works correctly, the beads tracking integration has a parsing bug that prevents proper issue status updates. This is already tracked and being fixed in window 6.

---

## Synthesis

**Key Insights:**

1. **Fire-and-forget spawn design enables concurrency** - The tmux-based spawn implementation doesn't block or wait for session IDs, allowing the orchestrator to spawn multiple agents in rapid succession without coordination overhead.

2. **Concurrent execution is stable** - Multiple agents can run simultaneously in separate tmux windows, each with their own OpenCode session and workspace, without interference.

3. **Beads tracking fails but spawn succeeds** - The issue ID parsing bug prevents proper beads integration, but doesn't block agent spawning or execution. The core functionality works despite the tracking failure.

**Answer to Investigation Question:**

Yes, orch-go successfully supports spawning multiple agents concurrently. The fire-and-forget tmux spawn design allows spawning subsequent agents while previous ones are still running, with each agent getting its own tmux window and workspace. Testing confirmed two consecutive spawns (windows 7 and 9) both started successfully and ran concurrently. The only limitation is the beads ID parsing bug that prevents proper issue tracking, but this doesn't affect the core spawning capability.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Direct testing confirmed concurrent spawning works with observable evidence (tmux windows created, OpenCode sessions running, workspaces generated). The test is simple, reproducible, and shows clear success. Confidence is not "Very High" because testing was limited to 2 concurrent spawns - haven't tested limits like 10+ concurrent agents or resource constraints.

**What's certain:**

- ✅ Multiple agents can be spawned concurrently (verified by spawning 2 agents back-to-back)
- ✅ Each spawn gets its own tmux window, workspace, and OpenCode session (verified by checking tmux list and workspace directories)
- ✅ Fire-and-forget design doesn't block subsequent spawns (verified by immediate return from spawn commands)

**What's uncertain:**

- ⚠️ Upper limit of concurrent spawns (how many can run before hitting resource constraints)
- ⚠️ Behavior under high load (e.g., 10+ agents spawning simultaneously)
- ⚠️ Impact of beads tracking bug on orchestrator workflows

**What would increase confidence to Very High (95%+):**

- Test with 5-10 concurrent spawns to verify scalability
- Monitor resource usage (CPU, memory, OpenCode server limits)
- Verify orch status and orch complete commands work correctly with concurrent agents

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## References

**Files Examined:**
- `.orch/workspace/og-inv-test-concurrent-spawn-20dec/SPAWN_CONTEXT.md` - Verified workspace creation
- `.orch/workspace/og-inv-test-third-concurrent-20dec/SPAWN_CONTEXT.md` - Verified second concurrent workspace

**Commands Run:**
```bash
# Test first concurrent spawn
./orch-go spawn investigation "test concurrent spawn capability"

# Verify tmux window creation
tmux list-windows -t workers-orch-go

# Check agent execution
tmux capture-pane -t workers-orch-go:7 -p

# Test second concurrent spawn
./orch-go spawn investigation "test third concurrent spawn"

# Verify both concurrent spawns running
tmux list-windows -t workers-orch-go | grep -E "concurrent|third"
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Knowledge:** kn-34d52f - "orch-go tmux spawn is fire-and-forget - no session ID capture"
- **Investigation:** `.kb/investigations/2025-12-19-inv-test-tmux-spawn.md` - Related tmux spawn testing
- **Workspace:** `.orch/workspace/og-inv-test-second-spawn-20dec/SPAWN_CONTEXT.md` - This investigation's workspace

---

## Investigation History

**[2025-12-20 08:20]:** Investigation started
- Initial question: Can orch-go spawn multiple agents concurrently?
- Context: Testing fire-and-forget tmux spawn design with existing running agents

**[2025-12-20 08:21]:** First concurrent spawn test
- Spawned window 7: og-inv-test-concurrent-spawn-20dec
- Verified workspace creation and OpenCode session startup

**[2025-12-20 08:22]:** Second concurrent spawn test
- Spawned window 9: og-inv-test-third-concurrent-20dec while window 7 still active
- Confirmed both agents running simultaneously in separate tmux windows

**[2025-12-20 08:25]:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Concurrent spawning confirmed working, limited only by untested upper bounds

---

## Self-Review

- [x] Real test performed (spawned 2 concurrent agents, verified with tmux commands)
- [x] Conclusion from evidence (observed tmux windows, workspaces, and OpenCode sessions)
- [x] Question answered (confirmed concurrent spawning works)
- [x] File complete (all sections filled)
- [x] TLDR filled (replaced placeholder with actual summary)

**Self-Review Status:** PASSED
