## Summary (D.E.K.N.)

**Delta:** The 2 new "cli" investigations since Jan 6 synthesis (bd CLI launchd slowness, kb-cli reflect dedup) are NOT about orch-go CLI - they were mis-tagged due to containing "cli" in filename. No updates needed to `.kb/guides/cli.md`.

**Evidence:** Reviewed both new investigations: bd-cli-slow (about beads CLI daemon timeout), kb-cli-fix (about kb-cli repo code fix). Neither contains orch-go CLI knowledge.

**Knowledge:** kb reflect synthesis detection is overly broad - matches any investigation containing "cli" in filename/content, not just orch CLI investigations. This creates false synthesis triggers.

**Next:** Close - no action needed for orch CLI guide. Consider improving kb reflect's topic matching to reduce false positives.

**Promote to Decision:** recommend-no (observation about tooling behavior, not architectural choice)

---

# Investigation: Synthesize Cli Investigations 18 Synthesis

**Question:** Do the 2 new CLI investigations since Jan 6 contain knowledge that should be added to `.kb/guides/cli.md`?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** kb-reflect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** None - this validates the Jan 6 synthesis remains current

---

## Findings

### Finding 1: Prior synthesis (Jan 6) created authoritative CLI guide

**Evidence:** Investigation `2026-01-06-inv-synthesize-cli-investigations-16-synthesis.md` already:
- Consolidated 16 CLI investigations into `.kb/guides/cli.md`
- Identified 7 categories of investigations
- Found 2 duplicate pairs
- Marked the guide as "supersedes" all prior investigations

**Source:** `.kb/investigations/2026-01-06-inv-synthesize-cli-investigations-16-synthesis.md`, `.kb/guides/cli.md`

**Significance:** The orch-go CLI guide is current as of Jan 6. Any synthesis should only consider truly NEW orch CLI knowledge.

---

### Finding 2: New investigation #1 (bd-cli-slow) is about beads CLI, not orch CLI

**Evidence:** `2026-01-07-design-bd-cli-slow-launchd-env.md` investigates:
- **Topic:** bd CLI (beads CLI) running slow in launchd/minimal environments
- **Root cause:** `BEADS_NO_DAEMON=1` env var missing in launchd
- **Fix:** Set env var in orch-go's Fallback* functions when shelling out to bd

This is about **bd CLI** behavior, not orch CLI behavior. It affects orch-go code (pkg/beads/client.go) but is not CLI user-facing knowledge.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-design-bd-cli-slow-launchd-env.md`

**Significance:** This investigation should NOT trigger orch CLI synthesis - it's a different CLI tool.

---

### Finding 3: New investigation #2 (kb-cli-fix) is about kb-cli repo, not orch-go

**Evidence:** `2026-01-08-inv-kb-cli-fix-reflect-dedup.md` investigates:
- **Topic:** kb-cli reflect command's dedup functions
- **Root cause:** Error handling was fail-open instead of fail-closed
- **Fix:** Code changes in kb-cli repo (NOT orch-go)

This is entirely about **kb-cli** (knowledge base CLI), a different repository. It has zero relevance to orch-go CLI.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-kb-cli-fix-reflect-dedup.md`

**Significance:** This investigation is filed in orch-go but is about kb-cli repo. Should not trigger orch CLI synthesis.

---

### Finding 4: kb reflect synthesis detection is too broad

**Evidence:** The spawn context listed 18 "cli" investigations, but the topic matching appears to be:
- Matching any investigation with "cli" anywhere in filename or content
- Not distinguishing between orch CLI, bd CLI, kb CLI, or other CLI mentions

This caused two unrelated investigations to be grouped with orch CLI investigations.

**Source:** Spawn context listing, `kb chronicle "cli"` showing 425 entries (far too many for orch-go CLI specifically)

**Significance:** The synthesis trigger is a false positive. Improving topic detection would reduce noise.

---

## Synthesis

**Key Insights:**

1. **No new orch CLI knowledge to consolidate** - The 2 new investigations since Jan 6 are about other CLI tools (bd, kb), not orch-go CLI. The existing `.kb/guides/cli.md` remains authoritative.

2. **Topic matching produces false positives** - The "cli" synthesis trigger matched investigations containing "cli" in any context, not specifically orch CLI investigations. This wastes synthesis effort.

3. **Cross-repo investigations cause confusion** - The kb-cli-fix investigation is filed in orch-go's .kb/ but is about kb-cli repo code. This creates misleading synthesis signals.

**Answer to Investigation Question:**

**No, the 2 new investigations do not contain knowledge that should be added to `.kb/guides/cli.md`.**

- Investigation #1 (bd-cli-slow) is about beads CLI daemon timeout behavior. While it affects orch-go code (the Fallback* functions), it's not user-facing orch CLI knowledge.
- Investigation #2 (kb-cli-fix) is about kb-cli repo code, entirely unrelated to orch-go.

The existing CLI guide synthesized on Jan 6 remains current and complete for orch-go CLI documentation.

---

## Structured Uncertainty

**What's tested:**

- ✅ Both new investigations read in full (verified: file contents reviewed)
- ✅ Neither contains orch CLI user-facing knowledge (verified: bd CLI is beads, kb-cli is different repo)
- ✅ Prior synthesis guide exists at `.kb/guides/cli.md` (verified: file read, last updated Jan 6)

**What's untested:**

- ⚠️ Whether kb reflect can be improved to distinguish CLI topics (out of scope for this investigation)
- ⚠️ Whether other "cli" tagged investigations have same false-positive issue (not audited)

**What would change this:**

- Finding would be wrong if either new investigation actually contains orch CLI knowledge I missed
- Finding would be incomplete if there were more than 2 new investigations since Jan 6

---

## Implementation Recommendations

**Purpose:** This investigation found no action needed for the CLI guide, but identified tooling improvements.

### Recommended Approach ⭐

**No action needed for CLI guide** - The existing `.kb/guides/cli.md` is current. Close this synthesis issue.

**Why this approach:**
- Both "new" investigations are about other tools, not orch CLI
- The Jan 6 synthesis already consolidated all actual orch CLI knowledge
- Updating the guide with unrelated knowledge would add noise

**Trade-offs accepted:**
- The kb-reflect false positive issue remains unfixed (out of scope)
- Future "cli" synthesis triggers may repeat this pattern

### Alternative Approaches Considered

**Option B: Add cross-reference to bd CLI investigation in cli.md**
- **Pros:** Documents related CLI ecosystem knowledge
- **Cons:** Bloats guide with tangential info; bd CLI is not orch CLI
- **When to use instead:** If users frequently confuse bd and orch commands

**Rationale for recommendation:** The cli.md guide is specifically for orch-go CLI. Adding bd CLI or kb-cli knowledge would violate the guide's purpose.

---

## Proposed Actions

### Archive Actions
| ID | Target | Reason | Approved |
|----|--------|--------|----------|
| (none) | | | |

### Create Actions
| ID | Type | Title | Description | Approved |
|----|------|-------|-------------|----------|
| C1 | issue | "Improve kb reflect topic matching" | kb reflect synthesis detection matches any file containing "cli" - should distinguish orch CLI, bd CLI, kb CLI | [ ] |

### Promote Actions
| ID | kn-id | To | Reason | Approved |
|----|-------|-------|--------|----------|
| (none) | | | | |

### Update Actions
| ID | Target | Change | Reason | Approved |
|----|--------|--------|--------|----------|
| U1 | `.kb/guides/cli.md` | Update "Last verified" to Jan 8, 2026 | Confirm guide is still current after reviewing new investigations | [ ] |

**Summary:** 2 proposals (0 archive, 1 create, 0 promote, 1 update)
**High priority:** None - these are minor improvements

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-06-inv-synthesize-cli-investigations-16-synthesis.md` - Prior synthesis to understand baseline
- `.kb/investigations/2026-01-07-design-bd-cli-slow-launchd-env.md` - New investigation #1, found to be about bd CLI
- `.kb/investigations/2026-01-08-inv-kb-cli-fix-reflect-dedup.md` - New investigation #2, found to be about kb-cli repo
- `.kb/guides/cli.md` - Existing guide to verify currency

**Commands Run:**
```bash
# Create investigation file
kb create investigation synthesize-cli-investigations-18-synthesis

# Check topic evolution (too broad - 425 matches)
kb chronicle "cli"

# Check synthesis opportunities
kb reflect --type synthesis
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-synthesize-cli-investigations-16-synthesis.md` - Prior synthesis this validates
- **Guide:** `.kb/guides/cli.md` - The guide that remains current

---

## Investigation History

**2026-01-08:** Investigation started
- Initial question: Can 18 CLI investigations be consolidated? (2 new since Jan 6)
- Context: kb reflect synthesis trigger for "cli" topic

**2026-01-08:** Found false positive
- Both "new" investigations are about other CLI tools (bd, kb), not orch CLI
- No new orch CLI knowledge to consolidate

**2026-01-08:** Investigation completed
- Status: Complete
- Key outcome: No action needed - existing CLI guide remains current
