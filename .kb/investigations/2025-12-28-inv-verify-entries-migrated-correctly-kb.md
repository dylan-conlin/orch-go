## Summary (D.E.K.N.)

**Delta:** Migration from .kn to .kb/quick preserved data correctly, but kb context returns duplicates because it reads from both locations.

**Evidence:** Tested "tiered spawn" query - same entry appears twice with different ID prefixes (kn-87329c vs kb-87329c); entries exist in both .kn/entries.jsonl and .kb/quick/entries.jsonl.

**Knowledge:** kb context searches BOTH legacy .kn and new .kb/quick locations, causing duplicates when entries exist in both; post-migration kn usage is adding entries only to .kn (not synced to .kb/quick).

**Next:** Need deduplication logic in kb context OR need to ensure .kn is fully deprecated and removed.

---

# Investigation: Verify Entries Migrated Correctly with kb context

**Question:** Did the migration from .kn to .kb/quick preserve entries correctly, and does kb context return them accurately?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Agent (orch-go-hop2)
**Phase:** Complete
**Next Step:** None - findings documented, issue creation recommended
**Status:** Complete

---

## Findings

### Finding 1: Data Preserved Correctly During Migration

**Evidence:** Compared entries between .kn/entries.jsonl and .kb/quick/entries.jsonl:
- ID prefix changed from `kn-` to `kb-` (e.g., `kn-87329c` → `kb-87329c`)
- All other fields identical: type, content, status, created_at, updated_at, reason, ref_count

**Source:**
```bash
grep -i "tiered spawn protocol" .kn/entries.jsonl
# {"id":"kn-87329c","type":"decision","content":"Tiered spawn protocol uses .tier file..."...}

grep -i "tiered spawn protocol" .kb/quick/entries.jsonl
# {"id":"kb-87329c","type":"decision","content":"Tiered spawn protocol uses .tier file..."...}
```

**Significance:** Migration correctly preserved all entry data. The only change was the ID prefix, which is intentional for namespace separation.

---

### Finding 2: kb context Returns Duplicates

**Evidence:** Running `kb context "tiered spawn"` shows the same entry twice:
```
## DECISIONS (from kn)

- Tiered spawn protocol uses .tier file in workspace for orch complete
  Reason: Allows VerifyCompletion to read tier...
- Tiered spawn protocol uses .tier file in workspace for orch complete
  Reason: Allows VerifyCompletion to read tier...
```

**Source:**
- Ran: `kb context "tiered spawn"`
- Code analysis: kb-cli/cmd/kb/context.go:140-146 shows both sources are searched and merged:
  ```go
  // Search kn entries from legacy .kn/ location
  knEntries, _ := searchKnEntries(projectDir, queryLower, opts)
  categorizeKnEntries(&result, knEntries, opts.Limit)
  
  // Search quick entries from new .kb/quick/ location  
  quickEntries, _ := SearchQuickEntries(projectDir, queryLower)
  categorizeKnEntries(&result, quickEntries, opts.Limit)
  ```

**Significance:** Both legacy .kn and new .kb/quick locations are searched without deduplication, causing identical entries to appear twice in output.

---

### Finding 3: Post-Migration Entry Count Divergence

**Evidence:** Entry counts differ between locations:
| Type       | .kn | .kb/quick | Missing |
|------------|-----|-----------|---------|
| constraint | 69  | 66        | 3       |
| decision   | 328 | 297       | 31      |
| attempt    | 23  | 21        | 2       |
| question   | 7   | 7         | 0       |

Timeline analysis:
- .kb/quick/entries.jsonl last modified: Dec 28 11:20
- .kn/entries.jsonl last modified: Dec 28 21:46

**Source:**
```bash
wc -l .kn/entries.jsonl .kb/quick/entries.jsonl
# 438 .kn/entries.jsonl
# 391 .kb/quick/entries.jsonl
```

**Significance:** Users continued using the deprecated `kn` command after migration, adding ~47 entries that only exist in .kn. The deprecation warning shows but doesn't prevent usage.

---

### Finding 4: kn Deprecation Notice Active

**Evidence:** Running `kn --help` shows:
```
DEPRECATION NOTICE: kn is deprecated and will be removed in a future release.

Quick knowledge entries have been merged into the kb CLI:
  - Use 'kb quick decide' instead of 'kn decide'
  - Use 'kb quick tried' instead of 'kn tried'
  - Use 'kb quick constrain' instead of 'kn constrain'
  - Use 'kb quick question' instead of 'kn question'
```

**Source:** `which kn` → `/Users/dylanconlin/bin/kn`

**Significance:** Deprecation warning exists but kn continues to function, adding entries to .kn only.

---

## Synthesis

**Key Insights:**

1. **Migration data integrity is good** - All entries that were migrated preserved their content, timestamps, and metadata correctly. Only the ID prefix changed.

2. **Dual-source search causes duplicates** - kb context searches both .kn and .kb/quick without checking for duplicates, leading to double entries in output.

3. **Deprecation isn't enforced** - The kn CLI shows a warning but continues to work, allowing users to add entries that bypass the new system.

**Answer to Investigation Question:**

The migration itself worked correctly - data was preserved accurately. However, kb context now returns duplicate entries because:
1. It reads from both legacy (.kn) and new (.kb/quick) locations
2. No deduplication logic exists
3. Users continue using deprecated kn command, adding entries only to .kn

---

## Structured Uncertainty

**What's tested:**

- ✅ Migration preserved entry data (verified: compared JSON fields between both files)
- ✅ kb context returns duplicates (verified: ran `kb context "tiered spawn"` and observed same entry twice)
- ✅ Entry count divergence exists (verified: grep + wc on both files)
- ✅ kn deprecation warning shows (verified: ran `kn --help`)

**What's untested:**

- ⚠️ Performance impact of searching both locations (not benchmarked)
- ⚠️ Global search (`kb context --global`) duplicate behavior (not tested)
- ⚠️ Whether other kb commands (promote, reflect) handle duplicates correctly

**What would change this:**

- Finding would be wrong if context.go has deduplication logic I missed
- Finding would be wrong if .kn entries are removed as part of a cleanup not yet run

---

## Implementation Recommendations

### Recommended Approach ⭐

**Deduplicate by content hash or remove .kn search** - Either add deduplication logic to kb context, or fully remove .kn search now that migration is complete.

**Why this approach:**
- Duplicates in kb context waste agent context tokens
- Users will see the same constraint/decision twice, potentially causing confusion
- Once .kn is deprecated, keeping the search adds maintenance burden

**Trade-offs accepted:**
- Removing .kn search immediately may lose entries added post-migration
- Deduplication adds complexity to context.go

**Implementation sequence:**
1. Run `kb migrate kn` again to sync any new .kn entries to .kb/quick
2. Add deduplication logic to kb context (match on content+type)
3. Consider removing .kn search entirely in next release

### Alternative Approaches Considered

**Option B: Re-run migration and remove .kn**
- **Pros:** Clean break, no dedup logic needed
- **Cons:** Requires careful verification; may miss entries
- **When to use instead:** If .kn entry count is manageable

**Option C: Keep both searches, accept duplicates**
- **Pros:** No code changes needed
- **Cons:** Duplicates waste context; confusing output
- **When to use instead:** If transitional period is expected to be short

---

### Implementation Details

**What to implement first:**
- Re-run `kb migrate kn` to capture post-migration entries
- Verify entry counts match after migration

**Things to watch out for:**
- ⚠️ Edge case: entries with same content but different IDs may be legitimate separate entries
- ⚠️ Entries with `status: superseded` should remain separate

**Areas needing further investigation:**
- Check if kb promote, kb reflect also have duplicate issues
- Verify global search behavior

**Success criteria:**
- ✅ `kb context "tiered spawn"` returns entry exactly once
- ✅ All entry types (constraint, decision, attempt, question) return without duplicates
- ✅ No data loss from migration cleanup

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/context.go` - Search logic for both sources
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/quick.go` - QuickStore and SearchQuickEntries
- `.kn/entries.jsonl` - Legacy entry storage
- `.kb/quick/entries.jsonl` - New entry storage

**Commands Run:**
```bash
# Count entries in both locations
wc -l .kn/entries.jsonl .kb/quick/entries.jsonl

# Check for duplicates
kb context "tiered spawn"

# Compare specific entry
grep -i "tiered spawn protocol" .kn/entries.jsonl
grep -i "tiered spawn protocol" .kb/quick/entries.jsonl

# Check kn deprecation
kn --help
```

---

## Investigation History

**2025-12-28 21:50:** Investigation started
- Initial question: Verify entries migrated correctly with kb context queries
- Context: Follow-up from Phase 1 (kb absorbs kn) consolidation effort

**2025-12-28 22:00:** Key findings discovered
- Found duplicate entries in kb context output
- Identified dual-source search as root cause
- Confirmed post-migration .kn usage continues

**2025-12-28 22:10:** Investigation completed
- Status: Complete
- Key outcome: Migration data is correct but kb context has duplicate problem due to dual-source search
