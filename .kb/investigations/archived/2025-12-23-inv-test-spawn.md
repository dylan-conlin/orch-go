<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Spawn hangs fixed by adding 5-second timeout to kb context queries - root cause was discoverProjects() doing unbounded filepath.Walk() of ~/Documents.

**Evidence:** Reproduced hang with kb context commands, traced to discoverProjects() in kb-cli (line 142 search.go), confirmed `find ~/Documents` also hangs; timeout fix allows spawn to complete.

**Knowledge:** kb context --global searches large directories without timeout causing indefinite hangs; spawn fallback to global search amplified the issue.

**Next:** Consider fixing kb-cli's discoverProjects() to use registry-only or cached discovery; monitor for 400 error in prompt sending.

**Confidence:** High (90%) - Root cause confirmed via code trace and reproduction, fix tested and working

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

# Investigation: Test Spawn

**Question:** Does the orch-go spawn system correctly create agents with proper context loading? (Updated: Fixed KB context hang)

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** og-debug-test-spawn-23dec
**Phase:** Complete
**Next Step:** Monitor for additional spawn issues
**Status:** Complete
**Confidence:** High (90%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Spawn Hangs During KB Context Global Search

**Evidence:** 
- `orch spawn` without `--skip-artifact-check` hangs indefinitely during "broader search" step
- `kb context "verify"` hangs (local search)
- `kb context --global "verify"` hangs (global search)
- All kb context queries timeout after 30+ seconds
- `find ~/Documents -name ".kb"` also hangs (5+ seconds timeout)

**Source:**
- pkg/spawn/kbcontext.go:131 - calls `kb context --global` when local search has few results
- kb-cli/cmd/kb/context.go:181 - GetContextGlobal calls discoverProjects()
- kb-cli/cmd/kb/search.go:142 - discoverProjects() does filepath.Walk() of ~/Documents

**Significance:** The global KB context search hangs because discoverProjects() scans ~/Documents recursively (up to 3 levels deep), which is too large or contains problematic paths (symlinks, network mounts, permissions issues). This blocks spawn from completing.

---

### Finding 2: Spawn Works When KB Context Is Skipped

**Evidence:**
- `orch spawn --skip-artifact-check` completes successfully in <2 seconds
- Creates session, workspace, and SPAWN_CONTEXT.md properly
- Agent is spawned and functional

**Source:** Test spawn with flag: `./build/orch spawn --no-track --light --skip-artifact-check investigation "verify spawn works"`

**Significance:** The spawn mechanism itself works perfectly. The hang is ONLY in the KB context query step, specifically the global search fallback.

---

### Finding 3: Spawn Context Successfully Loaded (Previous Test)

**Evidence:** Agent received and parsed SPAWN_CONTEXT.md from workspace directory with complete skill guidance (feature-impl), beads integration instructions, and deliverables requirements.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-test-spawn-23dec/SPAWN_CONTEXT.md` - 586 lines loaded successfully

**Significance:** Confirms spawn system correctly generates and loads context files with skill content embedded. Agent has access to all required procedural guidance without additional tool invocations.

---

### Finding 2: Investigation File Creation Works

**Evidence:** Command `kb create investigation test-spawn` successfully created investigation file at expected location with complete template structure.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-test-spawn.md` created via kb CLI

**Significance:** Integration between spawn system and kb CLI is working. Agents can create investigation files as required by deliverables without manual intervention.

---

### Finding 3: Beads Integration Has Expected Limitations

**Evidence:** Commands `bd comment orch-go-untracked-1766504669 "..."` consistently returned "issue orch-go-untracked-1766504669 not found" error.

**Source:** Multiple bd comment attempts during phase reporting

**Significance:** For test spawns with non-existent beads issues, bd commands fail gracefully without blocking agent progress. This is expected behavior for isolated spawn testing.

---

## Synthesis

**Key Insights:**

1. **KB Context Global Search Hangs Indefinitely** - The kb context --global command hangs because discoverProjects() does filepath.Walk() of ~/Documents and other large directories without timeout. Even `find ~/Documents -name ".kb"` hangs, confirming the directory is too large or contains problematic paths (symlinks, network mounts, permissions issues).

2. **Spawn's Fallback Strategy Amplified the Problem** - When local KB context returns few results (<3 matches), spawn automatically falls back to --global search. This well-intentioned feature became a reliability issue because global search has no timeout and hangs indefinitely.

3. **Fix: Timeout Provides Graceful Degradation** - Adding a 5-second timeout to kb context command execution allows spawn to continue when KB context hangs. If timeout is reached, treat as "no context available" - spawn still works, just without KB context enrichment. This is better than hanging indefinitely.

**Answer to Investigation Question:**

Yes, the orch-go spawn system now correctly creates agents with proper context loading. The hang issue was caused by kb context --global scanning large directories without timeout. Fixed by adding 5-second timeout to kb context queries in pkg/spawn/kbcontext.go. Spawn now completes successfully even when global search times out, gracefully degrading to no KB context rather than hanging.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

This is a single test spawn with observable success in core functionality (context loading, file creation, agent execution). Confidence is high for the specific mechanics tested but not Very High because this is a single data point without testing edge cases, error conditions, or integration with real beads workflow.

**What's certain:**

- ✅ SPAWN_CONTEXT.md generation works correctly (586 lines loaded with complete skill content)
- ✅ kb CLI integration functional (investigation file created successfully)
- ✅ Agent receives and can parse spawn context (demonstrated by completing deliverables)

**What's uncertain:**

- ⚠️ Behavior with real beads issues (only tested with non-existent issue)
- ⚠️ Error handling for malformed context or missing dependencies
- ⚠️ Performance/reliability across multiple concurrent spawns

**What would increase confidence to Very High (95%+):**

- Test spawn with real beads issue to verify full bd comment workflow
- Test error conditions (missing workspace, malformed context, invalid skill)
- Spawn multiple agents concurrently to verify isolation and stability

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-test-spawn-23dec/SPAWN_CONTEXT.md` - Spawn context loaded by agent (586 lines)
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-test-spawn.md` - Investigation file created via kb CLI

**Commands Run:**
```bash
# Report phase to beads (failed as expected - issue doesn't exist)
bd comment orch-go-untracked-1766504669 "Phase: Planning - Testing spawn system with minimal task"

# Verify project location
pwd

# Create investigation file
kb create investigation test-spawn

# Report investigation path (failed as expected)
bd comment orch-go-untracked-1766504669 "investigation_path: ..."
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-feat-test-spawn-23dec/` - Workspace created by spawn command

---

## Investigation History

**2025-12-23:** Investigation started
- Initial question: Does the orch-go spawn system correctly create agents with proper context loading?
- Context: Testing spawn system functionality with minimal test task

**2025-12-23:** Findings documented
- Verified spawn context loading (586 lines), kb CLI integration, and workspace setup
- Observed expected bd comment failures for non-existent beads issue

**2025-12-23:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Spawn system verified working for basic use cases
