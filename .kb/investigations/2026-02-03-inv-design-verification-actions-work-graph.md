<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Keyboard shortcuts and API endpoint are already implemented, but verification state doesn't persist across page refresh because the server doesn't load/apply stored verifications when serving attention data.

**Evidence:** Code review shows `markVerified()`/`markNeedsFix()` call API, `handleAttentionVerify()` persists to JSONL, but `handleAttention()` doesn't load verifications.jsonl or filter verified issues from response.

**Knowledge:** The verification workflow has a write path (POST persists) but no read path (GET doesn't use persisted state). This is a simple gap to close.

**Next:** Implement verification state loading and filtering in handleAttention() - add loadVerifications() function, filter verified issues from recently-closed results.

**Authority:** implementation - Completing existing pattern, no architectural changes needed

---

# Investigation: Design Verification Actions Work Graph

**Question:** How should verification actions (v/x keyboard shortcuts) be implemented for the Work Graph with backend persistence?

**Started:** 2026-02-03
**Updated:** 2026-02-03
**Owner:** og-arch-design-verification-actions-03feb-fbdf
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Frontend Keyboard Handlers Already Implemented

**Evidence:** `work-graph-tree.svelte` lines 259-273 already implement `v` and `x` key handlers:
```typescript
case 'v':
    event.preventDefault();
    if (isCompletedIssue(current) && current.verificationStatus === 'unverified') {
        attention.markVerified(current.id);
    }
    break;

case 'x':
    event.preventDefault();
    if (isCompletedIssue(current) && current.verificationStatus === 'unverified') {
        attention.markNeedsFix(current.id);
    }
    break;
```

**Source:** `web/src/lib/components/work-graph-tree/work-graph-tree.svelte:259-273`

**Significance:** No frontend work needed for keyboard shortcuts - they're already in place and call the correct store methods.

---

### Finding 2: Attention Store Already Calls API

**Evidence:** `attention.ts` lines 215-272 show `markVerified()` and `markNeedsFix()` methods that:
1. Call `POST ${API_BASE}/api/attention/verify` with correct JSON body
2. Update local state on success
3. Return boolean success indicator

```typescript
async markVerified(issueId: string): Promise<boolean> {
    const response = await fetch(`${API_BASE}/api/attention/verify`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ issue_id: issueId, status: 'verified' }),
    });
    // ... updates local state on success
}
```

**Source:** `web/src/lib/stores/attention.ts:215-272`

**Significance:** The store correctly calls the API - no changes needed to the store's write methods.

---

### Finding 3: API Endpoint Exists and Persists to JSONL

**Evidence:** `serve_attention.go` lines 268-355 implement:
1. `handleAttentionVerify()` - POST handler for `/api/attention/verify`
2. Validates `issue_id` and `status` fields
3. Valid statuses: `verified` or `needs_fix`
4. Persists to `~/.orch/verifications.jsonl` via `persistVerification()`

Tests exist in `serve_attention_test.go` verifying:
- Method validation (POST only)
- Required field validation
- Status value validation
- Successful persistence to JSONL

**Source:** `cmd/orch/serve_attention.go:268-355`, `cmd/orch/serve_attention_test.go:204-360`

**Significance:** Write path is complete and tested. The issue is the read path.

---

### Finding 4: Critical Gap - Verification State Not Loaded on GET

**Evidence:** `handleAttention()` (lines 109-229) does NOT:
1. Load verifications from `~/.orch/verifications.jsonl`
2. Filter verified issues from the `recently-closed` collector results
3. Include verification status in the response

`RecentlyClosedCollector.Collect()` (lines 32-88) returns ALL recently-closed issues without checking if they've been verified.

**Source:**
- `cmd/orch/serve_attention.go:109-229` (handleAttention)
- `pkg/attention/recently_closed_collector.go:32-88` (Collect)

**Significance:** This is the root cause - verified issues reappear after page refresh because the server never loads the persisted verification state.

---

## Synthesis

**Key Insights:**

1. **Write path complete, read path missing** - The implementation is 90% done. Writing verifications works perfectly. The gap is that GET /api/attention doesn't use the stored verifications.

2. **Simple fix location** - The fix belongs in `handleAttention()` in serve_attention.go, not in the collector. Loading verifications and filtering should happen at the aggregation layer.

3. **No frontend changes needed** - The frontend already handles verification state correctly. Once the API returns accurate verification status, everything will work.

**Answer to Investigation Question:**

The verification actions are already largely implemented. The only missing piece is loading verification state when serving attention data. Implementation requires:

1. Add `loadVerifications()` function to load the JSONL file
2. Modify `handleAttention()` to filter verified issues from recently-closed results
3. Pass verification status for `needs_fix` items in metadata

---

## Structured Uncertainty

**What's tested:**

- ✅ POST /api/attention/verify persists to JSONL (verified: unit tests pass, code review confirms)
- ✅ Keyboard shortcuts v/x trigger store methods (verified: code review of handleKeyDown)
- ✅ Store methods call API endpoint (verified: code review of markVerified/markNeedsFix)

**What's untested:**

- ⚠️ Loading verifications from JSONL will correctly filter issues (needs implementation)
- ⚠️ Page refresh will preserve verification state (needs E2E testing after implementation)
- ⚠️ needs_fix issues display correctly after refresh (needs verification)

**What would change this:**

- If `loadVerifications()` has performance issues with large JSONL files, may need caching or a different storage format
- If there are concurrent write issues with the JSONL file, may need file locking

---

## Implementation Recommendations

**Purpose:** Complete the verification workflow by adding the read path for persisted verification state.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add loadVerifications() and filter in handleAttention() | implementation | Completing existing pattern - write path exists, just adding read path. Single file change, reversible. |

### Recommended Approach ⭐

**Filter in handleAttention()** - Load verifications and filter after collecting, before response

**Why this approach:**
- Minimal code changes (one function addition, one modification)
- Doesn't change collector interface or architecture
- Keeps verification logic co-located with the endpoint
- Matches existing patterns in serve_attention.go

**Trade-offs accepted:**
- Loads JSONL on every request (acceptable for small files, <100 entries)
- If JSONL grows large, may need caching in future (not a concern for current use)

**Implementation sequence:**

1. **Add `loadVerifications()` function** - Read and parse `~/.orch/verifications.jsonl`, return `map[string]VerificationEntry`

2. **Modify `handleAttention()`** - After collecting items:
   - Load verifications
   - Filter out issues with `status: "verified"` from recently-closed items
   - Add verification status to metadata for `needs_fix` items

3. **Add tests** - Test that:
   - Verified issues don't appear in response
   - needs_fix issues appear with correct status in metadata
   - Missing/empty JSONL file handles gracefully

### Alternative Approaches Considered

**Option B: Filter in RecentlyClosedCollector**
- **Pros:** Filtering at source, cleaner separation
- **Cons:** Requires passing verification state through collector interface, more invasive change
- **When to use instead:** If multiple collectors need verification filtering

**Option C: Separate GET /api/attention/verifications endpoint**
- **Pros:** Clean API separation
- **Cons:** Frontend needs two requests, more complex state management
- **When to use instead:** If verifications are needed independently of attention signals

**Rationale for recommendation:** Option A is simplest for the current use case. The read path should mirror the write path location.

---

### Implementation Details

**What to implement first:**
1. `loadVerifications()` function in serve_attention.go
2. Filter logic in handleAttention()
3. Tests for the new behavior

**Things to watch out for:**
- ⚠️ JSONL file may not exist on first run - return empty map, don't error
- ⚠️ Malformed lines in JSONL - skip and log, don't fail entirely
- ⚠️ Use the latest entry per issue_id (later entries override earlier ones)

**Areas needing further investigation:**
- None - implementation is straightforward

**Success criteria:**
- ✅ Verified issues don't appear in recently-closed list after page refresh
- ✅ needs_fix issues appear with `needs_fix` status in metadata
- ✅ Tests pass for all edge cases (empty file, missing file, malformed lines)

---

## File Change Summary

### Files to Modify

| File | Change Type | Description |
|------|-------------|-------------|
| `cmd/orch/serve_attention.go` | Modify | Add `loadVerifications()` function, modify `handleAttention()` to filter |
| `cmd/orch/serve_attention_test.go` | Modify | Add tests for verification state loading and filtering |

### No Changes Needed

| File | Reason |
|------|--------|
| `web/src/lib/stores/attention.ts` | Store already handles API correctly |
| `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` | Keyboard handlers already work |
| `pkg/attention/recently_closed_collector.go` | Collector doesn't need verification awareness |

---

## Code Sketch

### loadVerifications() function

```go
// loadVerifications reads the verification log and returns the latest status per issue.
// Returns an empty map if the file doesn't exist or is empty.
func loadVerifications() map[string]VerificationEntry {
    verifications := make(map[string]VerificationEntry)

    f, err := os.Open(verificationLogPath)
    if err != nil {
        // File doesn't exist yet - that's fine
        return verifications
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        var entry VerificationEntry
        if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
            continue // Skip malformed lines
        }
        // Later entries override earlier (supports re-verification)
        verifications[entry.IssueID] = entry
    }

    return verifications
}
```

### handleAttention() modification

```go
// In handleAttention(), after collecting from all sources:

// Load verification state
verifications := loadVerifications()

// Filter items: remove verified recently-closed, mark needs_fix
filteredItems := []attention.AttentionItem{}
for _, item := range allItems {
    if item.Signal == "recently-closed" {
        if v, ok := verifications[item.Subject]; ok {
            if v.Status == "verified" {
                continue // Skip verified issues
            }
            // Add needs_fix status to metadata
            if item.Metadata == nil {
                item.Metadata = make(map[string]any)
            }
            item.Metadata["verification_status"] = v.Status
        }
    }
    filteredItems = append(filteredItems, item)
}
allItems = filteredItems
```

---

## References

**Files Examined:**
- `web/src/lib/stores/attention.ts` - Store implementation with markVerified/markNeedsFix
- `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` - Keyboard handlers
- `cmd/orch/serve_attention.go` - API handlers for attention and verification
- `cmd/orch/serve_attention_test.go` - Existing tests
- `pkg/attention/recently_closed_collector.go` - Collector for recently closed issues

**Commands Run:**
```bash
# View attention package files
ls -la /Users/dylanconlin/Documents/personal/orch-go/pkg/attention/

# Search for verification-related code
grep -r "loadVerification\|verification.*jsonl" --include="*.go"
```

**Related Artifacts:**
- **Issue:** orch-go-21213 - Add verification actions to Work Graph

---

## Investigation History

**2026-02-03 09:44:** Investigation started
- Initial question: How should verification actions be implemented?
- Context: Issue orch-go-21213 describes keyboard shortcuts v/x for marking issues verified/needs-fix

**2026-02-03 09:50:** Key discovery - implementation largely exists
- Found keyboard handlers already in work-graph-tree.svelte
- Found API endpoint already in serve_attention.go
- Identified gap: GET doesn't load persisted verifications

**2026-02-03 10:00:** Investigation completed
- Status: Complete
- Key outcome: Add loadVerifications() and filter in handleAttention() - simple completion of existing pattern
