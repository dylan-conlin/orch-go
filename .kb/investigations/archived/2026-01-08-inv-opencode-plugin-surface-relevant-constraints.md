<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Enhanced existing guarded-files plugin to surface kb/kn constraints when editing guarded files (principles.md, SKILL.md, etc.).

**Evidence:** Tested with bun - principles.md shows principle-addition protocol + kb constraint; SKILL.md shows skillc protocol + skillc constraints.

**Knowledge:** Path-based detection must have higher priority than content-based to avoid false positives (principles.md mentions "AUTO-GENERATED" in an example).

**Next:** Plugin is ready for production use. Restart OpenCode server to load changes.

**Promote to Decision:** recommend-no (enhancement to existing plugin, not architectural)

---

# Investigation: Opencode Plugin Surface Relevant Constraints

**Question:** How can we surface relevant kb/kn constraints when agents edit guarded files?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing guarded-files plugin structure

**Evidence:** A `guarded-files.ts` plugin already existed at `~/.config/opencode/plugin/` with:
- `tool.execute.before` hook on Edit tool
- Pattern registry for guarded files (skillc, principles.md, CLAUDE.md, decisions)
- `client.session.prompt` injection with `noReply: true`
- Per-file session caching to avoid spam

**Source:** `/Users/dylanconlin/.config/opencode/plugin/guarded-files.ts`, `/Users/dylanconlin/.config/opencode/lib/guarded-files.ts`

**Significance:** The infrastructure was already in place - just needed to add kb context integration.

---

### Finding 2: kb context returns constraints and decisions

**Evidence:** `kb context "principle" --global` returns:
```
## CONSTRAINTS (from kn)
- [orch-go] Principle changes require following ~/.kb/guides/principle-addition.md protocol
  Reason: Forgot protocol exists, edited without checking skills/guides
```

**Source:** `kb context "principle" --global`, `kb context "skillc" --global`

**Significance:** kb context provides exactly the constraints we need - we can query by keyword and get relevant constraints.

---

### Finding 3: Priority ordering prevents false positives

**Evidence:** principles.md contains "AUTO-GENERATED" in an example text, which was matching the skillc detection pattern. Fixed by giving path-based patterns (principles.md: priority 110) higher priority than content-based patterns (skillc: priority 100).

**Source:** `grep -n "AUTO-GENERATED" ~/.kb/principles.md` - line 77

**Significance:** Content-based detection should have lower priority than path-based detection to avoid false positives.

---

## Synthesis

**Key Insights:**

1. **Static + Dynamic Context** - The enhanced plugin combines static protocols (hardcoded guidance) with dynamic kb context (kn constraints), providing both evergreen documentation and session-specific learnings.

2. **Keyword-Based Retrieval** - Each guarded file pattern can specify a `kbKeyword` which is used to query `kb context --global --limit 5`. Only constraints are extracted for brevity.

3. **Priority-Based Pattern Matching** - Using numeric priority allows flexible ordering of detection patterns. Path-based patterns should have higher priority than content-based to avoid false positives.

**Answer to Investigation Question:**

The existing guarded-files plugin was enhanced to surface kb/kn constraints by:
1. Adding `kbKeyword` field to GuardedFile interface
2. Adding `getKbContext()` function that runs `kb context` and filters to constraints
3. Adding `getGuardedFileProtocolWithContext()` that combines static protocol with dynamic constraints
4. Updating plugin to use the enhanced function

---

## Structured Uncertainty

**What's tested:**

- ✅ principles.md shows correct protocol with kb constraint (verified: bun test)
- ✅ SKILL.md shows skillc protocol with skillc constraints (verified: bun test)
- ✅ kb context query returns constraints within 5 second timeout (verified: bun test)

**What's untested:**

- ⚠️ Performance impact in real agent sessions (no latency benchmarks)
- ⚠️ Behavior when kb command not available (error handling exists but not tested)
- ⚠️ Integration with OpenCode's session.prompt API (requires server restart to verify)

**What would change this:**

- Finding would be wrong if kb command is slow (>5s) in production, causing timeout
- Finding would be wrong if session.prompt injection breaks agent flow

---

## Implementation Recommendations

### Recommended Approach ⭐

**Enhance existing plugin** - Add kb context integration to existing guarded-files plugin rather than creating new plugin.

**Why this approach:**
- Avoids duplicate detection logic
- Leverages existing infrastructure (caching, injection)
- Single point of maintenance

**Trade-offs accepted:**
- Constraints shown are limited to 5 per keyword (configurable via --limit)
- Only constraints section is extracted (not decisions/guides)

**Implementation sequence:**
1. Add kbKeyword to GuardedFile interface ✅
2. Add getKbContext() function ✅
3. Add getGuardedFileProtocolWithContext() function ✅
4. Update plugin to use enhanced function ✅

---

### Implementation Details

**What to implement first:**
- ✅ All implementation complete

**Things to watch out for:**
- ⚠️ kb command must be in PATH (added ~/.bun/bin to PATH env)
- ⚠️ 5 second timeout may be too short for cold kb index
- ⚠️ Path-based patterns need higher priority than content-based

**Success criteria:**
- ✅ Editing principles.md shows principle constraint
- ✅ Editing SKILL.md shows skillc constraints
- ✅ No false positives (principles.md doesn't show as skillc)

---

## References

**Files Examined:**
- `/Users/dylanconlin/.config/opencode/plugin/guarded-files.ts` - Main plugin
- `/Users/dylanconlin/.config/opencode/lib/guarded-files.ts` - Helpers
- `/Users/dylanconlin/.config/opencode/plugin/bd-close-gate.ts` - Reference plugin pattern
- `/Users/dylanconlin/.config/opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts` - Plugin API types

**Commands Run:**
```bash
# Test kb context queries
kb context "principle" --global
kb context "skillc" --global

# Test plugin functions
cd /Users/dylanconlin/.config/opencode && bun -e "
import { getGuardedFileProtocolWithContext } from './lib/guarded-files.ts'
const protocol = await getGuardedFileProtocolWithContext('/Users/dylanconlin/.kb/principles.md')
console.log(protocol)
"
```

**Related Artifacts:**
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-epic-mechanize-principles-via-opencode.md` - Parent epic
- **Guide:** `~/.kb/guides/principle-addition.md` - Protocol for principles.md

---

## Investigation History

**2026-01-08 10:20:** Investigation started
- Initial question: How to surface kb/kn constraints when editing guarded files?
- Context: Agent edited principles.md without seeing principle-addition protocol constraint

**2026-01-08 10:25:** Found existing plugin
- Discovered guarded-files.ts plugin already existed with infrastructure

**2026-01-08 10:35:** Enhanced plugin
- Added kbKeyword field and kb context integration
- Fixed priority ordering for path-based vs content-based detection

**2026-01-08 10:40:** Investigation completed
- Status: Complete
- Key outcome: Plugin enhanced to surface kb constraints alongside static protocols
