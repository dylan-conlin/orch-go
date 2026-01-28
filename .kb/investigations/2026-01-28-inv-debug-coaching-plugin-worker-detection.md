<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Coaching plugin is disabled because it uses broken metadata-based worker detection (line 2028 checks `sessionMetadata.role === "worker"` but metadata field doesn't exist in session.created events).

**Evidence:** File is named `coaching.ts.disabled`; code at line 2028 uses metadata approach; investigation 21001 confirmed metadata field is always empty object in session.created events; investigation 21001 verified title-based detection works.

**Knowledge:** The plugin was "upgraded" from working title-based to broken metadata-based detection without validating that metadata is available in event hooks; title-based pattern (`hasBeadsId && !isOrchestrator`) is proven working and ready to use.

**Next:** Replace line 2028 with title-based detection, remove debug logging, rename to `coaching.ts`, test with worker spawn.

**Promote to Decision:** recommend-no - This is a bug fix reverting to proven working approach, not a new architectural decision.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Debug Coaching Plugin Worker Detection

**Question:** Is the coaching plugin worker detection functioning correctly, and what is the current state of worker vs orchestrator detection?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** og-inv-debug-coaching-plugin-28jan-3d11
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Coaching Plugin is Currently Disabled

**Evidence:** The coaching plugin file is named `coaching.ts.disabled` rather than `coaching.ts`. OpenCode plugins must have `.ts` extension to be loaded - the `.disabled` suffix prevents plugin loading.

**Source:** 
- File listing: `/Users/dylanconlin/Documents/personal/orch-go/.opencode/plugins/coaching.ts.disabled` exists
- Attempted to read `coaching.ts` failed with "file not found"
- Directory listing shows only `coaching.ts.disabled` (no active `coaching.ts`)

**Significance:** This means the coaching plugin is not currently running at all, so worker detection issues cannot be actively occurring. This investigation needs to determine whether the plugin was disabled due to worker detection problems, or for another reason.

---

### Finding 2: Plugin Was Disabled After Metadata-Based Detection Failed

**Evidence:** Investigation `2026-01-28-inv-debug-coaching-plugin-still-fires.md` (issue 21001, completed 13:43) found that the coaching plugin was "upgraded" from title-based to metadata-based worker detection, but session.created events do NOT include the metadata field. This caused worker detection to fail completely. The plugin was disabled at 13:39, during or shortly after this investigation.

**Source:**
- `.orch/workspace/og-inv-debug-coaching-plugin-28jan-b245/SYNTHESIS.md` - Details the root cause
- `stat` output showing coaching.ts.disabled modified at 13:39
- Timeline: audit investigation completed 13:35, plugin disabled 13:39, verify investigation completed 13:43

**Significance:** The plugin is disabled because the metadata-based detection approach doesn't work (metadata not available in session.created events). The recommended fix is to revert to title-based detection, which was proven working in prior investigations.

---

### Finding 3: Title-Based Detection Pattern Is Proven to Work

**Evidence:** Investigation `2026-01-28-inv-verify-coaching-plugin-worker-detection.md` tested two separate worker sessions and confirmed zero coaching alerts were fired despite 10+ tool calls each. The title-based pattern used was: `hasBeadsId && !isOrchestratorTitle` where beads ID matches `/\[[\w-]+-\d+\]/` and orchestrator pattern is `/-orch-/`.

**Source:**
- `.kb/investigations/2026-01-28-inv-verify-coaching-plugin-worker-detection.md` - Verification testing
- Tested sessions: `ses_3f9d325bbffetxp88HZ2YFlWhq` and `ses_3f9d0c828ffeGIx3oua2PzXlnx`
- Both had titles like `og-inv-verify-coaching-plugin-28jan-5e08 [orch-go-20993]`

**Significance:** This provides strong evidence that title-based detection is reliable for standard worker spawns with beads tracking. This is the fallback approach that should be used since metadata-based detection is not available.

---

### Finding 4: Current Code Uses Broken Metadata-Based Detection

**Evidence:** The disabled coaching plugin at line 2028 uses `sessionMetadata.role === "worker"` for worker detection. However, investigation 21001 confirmed that `info.metadata` is always an empty object (`{}`) in session.created events, so `sessionMetadata.role` is always undefined, making this check always false.

**Source:**
- `.opencode/plugins/coaching.ts.disabled:2028` - `const isWorker = sessionMetadata.role === "worker"`
- `.opencode/plugins/coaching.ts.disabled:2017` - `const sessionMetadata = info.metadata || {}`
- Investigation finding: session.created events have `properties.info.{id, title, directory, ...}` but NO metadata field

**Significance:** This confirms the plugin cannot work in its current state. It needs to be modified to use title-based detection (pattern: `hasBeadsId && !isOrchestrator`) before being re-enabled.

---

## Synthesis

**Key Insights:**

1. **The plugin is disabled because the detection logic is broken** - The coaching plugin was "upgraded" from working title-based detection to broken metadata-based detection (Finding 4). Since session.created events don't include metadata (Finding 2), worker detection never succeeds, causing workers to incorrectly receive coaching alerts. The plugin was disabled as a temporary measure.

2. **Title-based detection is proven and ready to use** - Investigation 21001 verified that title-based detection (`hasBeadsId && !isOrchestratorTitle`) works correctly across multiple worker sessions (Finding 3). This approach is reliable for the common case of worker spawns with beads tracking.

3. **The fix is straightforward** - Replace line 2028 (`sessionMetadata.role === "worker"`) with title-based pattern matching on `sessionTitle`. The pattern is well-documented from prior investigations and has proven test results.

**Answer to Investigation Question:**

The coaching plugin worker detection is currently **not functioning** because:
1. The plugin file is disabled (`.ts.disabled` extension prevents loading) - Finding 1
2. Even if enabled, the detection logic is broken (metadata-based approach fails because metadata field doesn't exist in session.created events) - Finding 4

To fix this: Replace the metadata-based detection at line 2028 with title-based detection using the pattern `/\[[\w-]+-\d+\]/` (has beads ID) AND NOT `/-orch-/` (not orchestrator). This pattern was proven working in investigation 2026-01-28-inv-verify-coaching-plugin-worker-detection.md (Finding 3).

---

## Structured Uncertainty

**What's tested:**

- ✅ Plugin file is disabled (verified: file is named `coaching.ts.disabled`, not `coaching.ts`)
- ✅ Current code uses metadata-based detection (verified: read line 2028 showing `sessionMetadata.role === "worker"`)
- ✅ Title-based detection pattern works (verified: investigation 2026-01-28-inv-verify-coaching-plugin-worker-detection.md tested two sessions)
- ✅ session.created events lack metadata field (verified: investigation 21001 examined actual event structure via event-test.jsonl)
- ✅ Timeline of when plugin was disabled (verified: stat showed modification at 13:39, between investigations 21000 and 21001)

**What's untested:**

- ⚠️ Whether fixing detection and re-enabling will work (implementation not performed)
- ⚠️ Whether there are other issues preventing the plugin from working
- ⚠️ Whether ad-hoc spawns (without beads IDs) need different handling
- ⚠️ Impact on orchestrator sessions if detection pattern is wrong

**What would change this:**

- Finding would be wrong if metadata IS actually available in session.created events (contradicts investigation 21001)
- Finding would be wrong if plugin file is actually active (contradicts file listing showing `.disabled` suffix)
- Finding would be wrong if title-based detection failed in production (contradicts verified test results)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Fix detection logic and re-enable plugin** - Replace metadata-based detection (line 2028) with title-based pattern, remove debug logging, rename file to `.ts` to enable.

**Why this approach:**
- Title-based detection is proven working (Finding 3 - two verified worker sessions)
- Addresses root cause (Finding 4 - metadata approach doesn't work)
- Simple, low-risk change (single line of logic + cleanup)
- Gets coaching plugin back online for orchestrators

**Trade-offs accepted:**
- Title-based detection doesn't cover edge cases (ad-hoc spawns without beads IDs)
- Relies on proper session titling conventions
- Won't work if titles are manually changed mid-session

**Implementation sequence:**
1. Replace line 2028 with title-based detection: `const hasBeadsId = /\[[\w-]+-\d+\]/.test(sessionTitle); const isOrchestrator = /-orch-/.test(sessionTitle); const isWorker = hasBeadsId && !isOrchestrator;`
2. Remove enhanced debug logging (lines 1998-1999, 2008, 2011, 2019, 2030, 2034-2039)
3. Rename `coaching.ts.disabled` to `coaching.ts`
4. Test with worker spawn to verify zero coaching alerts
5. Test with orchestrator session to verify coaching still fires

### Alternative Approaches Considered

**Option B: Keep plugin disabled until OpenCode adds metadata support**
- **Pros:** Waits for proper architectural fix (metadata in events)
- **Cons:** Orchestrators lose coaching functionality indefinitely; upstream fix timeline unknown
- **When to use instead:** If title-based detection proves unreliable in production

**Option C: Move coaching logic to orch-go process**
- **Pros:** orch-go CAN see ORCH_WORKER env var; eliminates architectural gap
- **Cons:** Loses OpenCode plugin hooks; significant refactoring; coupling increases
- **When to use instead:** If OpenCode plugin architecture proves fundamentally unsuitable

**Rationale for recommendation:** Title-based detection is proven working (Finding 3), the fix is simple (one line + cleanup), and coaching provides value to orchestrators. Option B leaves coaching broken indefinitely. Option C is over-engineering for a problem that has a working solution.

---

### Implementation Details

**What to implement first:**
- Fix the detection logic at line 2028 (this is the core bug)
- Remove debug logging (prevents console noise)
- Rename file to enable plugin (makes fix active)

**Things to watch out for:**
- ⚠️ Verify OpenCode server restarts after renaming file (plugins load at server startup)
- ⚠️ Test with REAL worker spawn, not just code review (verify in production)
- ⚠️ Check that orchestrator coaching still works (don't break existing functionality)
- ⚠️ Ad-hoc spawns without beads IDs will still receive coaching (known limitation)

**Areas needing further investigation:**
- Whether OpenCode will eventually add metadata to session.created events
- Whether ad-hoc worker sessions (without beads tracking) need different handling
- Whether to add fallback detection for edge cases
- Whether to add telemetry to track detection accuracy over time

**Success criteria:**
- ✅ Worker sessions (with beads IDs) have zero coaching alerts in coaching-metrics.jsonl
- ✅ Orchestrator sessions continue receiving coaching (action_ratio, analysis_paralysis, etc.)
- ✅ No debug logging appears in console during normal operation
- ✅ Plugin file exists as `coaching.ts` (not `.disabled`)

---

## References

**Files Examined:**
- `.opencode/plugins/coaching.ts.disabled` - Current (broken) plugin code
- `.kb/investigations/2026-01-28-inv-verify-coaching-plugin-worker-detection.md` - Proof that title-based detection works
- `.kb/investigations/2026-01-28-inv-audit-opencode-plugins-worker-detection.md` - Audit of plugin detection approaches
- `.orch/workspace/og-inv-debug-coaching-plugin-28jan-b245/SYNTHESIS.md` - Root cause analysis from issue 21001

**Commands Run:**
```bash
# Check plugin file status
ls -la /Users/dylanconlin/Documents/personal/orch-go/.opencode/plugins/

# Check when plugin was disabled
stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" /Users/dylanconlin/Documents/personal/orch-go/.opencode/plugins/coaching.ts.disabled

# Search for detection patterns in plugin code
grep -n "sessionMetadata.role" .opencode/plugins/coaching.ts.disabled

# List coaching-related workspaces to understand investigation timeline
ls -lt /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/ | grep "coaching"
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-28-inv-verify-coaching-plugin-worker-detection.md` - Proves title-based detection works
- **Investigation:** `.kb/investigations/2026-01-28-inv-audit-opencode-plugins-worker-detection.md` - Identifies metadata approach as reference
- **Investigation:** `.kb/investigations/2026-01-28-inv-orchestrator-coaching-plugin-cannot-reliably.md` - Architectural analysis
- **Workspace:** `.orch/workspace/og-inv-debug-coaching-plugin-28jan-b245/` - Issue 21001 root cause investigation

---

## Investigation History

**2026-01-28 14:59:** Investigation started
- Initial question: Is the coaching plugin worker detection functioning correctly?
- Context: Spawned as worker to debug coaching plugin worker detection issues

**2026-01-28 15:00:** First finding - plugin is disabled
- Discovered coaching.ts.disabled instead of coaching.ts
- Created investigation file and committed initial checkpoint

**2026-01-28 15:05:** Timeline and root cause uncovered
- Read prior investigations (21000, 21001) and SYNTHESIS.md
- Found that plugin was disabled at 13:39 after metadata-based detection failed
- Confirmed title-based detection is proven working

**2026-01-28 15:10:** Investigation completed
- Status: Complete
- Key outcome: Plugin is disabled due to broken metadata-based detection; fix requires reverting to title-based pattern at line 2028
