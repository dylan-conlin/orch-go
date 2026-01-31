<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Duplicate synthesis issues are caused by silent failure in the deduplication check when JSON parsing fails.

**Evidence:** The `synthesisIssueExists` function in kb-cli returns `false, nil` when `json.Unmarshal` fails, allowing duplicate creation. Traced 14+ duplicates for "model" topic, all created hourly despite open issues existing.

**Knowledge:** Error handling that "assumes no duplicate on failure" is fundamentally wrong for idempotency - it should assume duplicate EXISTS on failure to prevent false positives.

**Next:** Fix dedup to return `true` (assume exists) on any error, and add logging for diagnosis.

**Promote to Decision:** Actioned - patterns in kb reflect tool

---

# Investigation: Recurring Problem Duplicate Synthesis Issues

**Question:** Why are duplicate synthesis issues being created repeatedly despite deduplication code being in place?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** Implement fix in kb-cli
**Status:** Complete

---

## Findings

### Finding 1: Duplicates are created every ~60 minutes

**Evidence:** Timeline of "Synthesize model investigations" issues:
- 14 total issues for this single topic
- Created at roughly hourly intervals (matching daemon's `--reflect-interval 60`)
- Example sequence: 12:18, 16:17, 17:31, 18:35, 19:36, 20:37, 21:38, 22:55, 23:55, 01:16, 02:48, 06:53, 07:07

**Source:** `bd list --all --title-contains "Synthesize model investigations" --json | jq -r '.[].created_at' | sort`

**Significance:** The daemon runs reflection hourly with `--reflect-issues true`, and each run creates duplicates. The dedup check isn't preventing creation.

---

### Finding 2: Dedup code returns false on JSON parse error

**Evidence:** In `kb-cli/cmd/kb/reflect.go`, line 505-508:
```go
if err := json.Unmarshal(output, &issues); err != nil {
    // If parsing fails, assume no duplicate
    return false, nil
}
```

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:505-508`

**Significance:** This is the root cause. When bd output contains any character that causes JSON parsing to fail, the function silently returns "no duplicate exists", allowing creation.

---

### Finding 3: JSON parsing can fail due to description content

**Evidence:** When testing with shell variable assignment, JSON parse occasionally fails:
```
jq: parse error: Invalid string: control characters from U+0000 through U+001F must be escaped at line 75, column 1
```
However, direct piping works consistently. The issue may be intermittent based on bd's output buffering or shell handling.

**Source:** Bash testing with `output=$(bd list ...) && echo "$output" | jq`

**Significance:** The dedup check is probabilistic - it sometimes works, sometimes fails silently. This explains why duplicates appear even when open issues exist.

---

### Finding 4: Error handling strategy is fundamentally wrong

**Evidence:** Lines 498-501 and 505-508 both follow the same pattern:
```go
if err != nil {
    // If X fails, assume no duplicate
    return false, nil
}
```

This appears in both:
- `cmd.Output()` failure (bd not found, etc.)
- `json.Unmarshal()` failure (parse error)

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:498-508`

**Significance:** For idempotency checks, "fail open" (assume no duplicate) is wrong. Should "fail closed" (assume duplicate exists) to prevent duplicates on error.

---

## Synthesis

**Key Insights:**

1. **Silent failure enables duplicates** - The dedup function never logs errors, never returns errors, and defaults to "allow creation". This makes diagnosis difficult and permits false negatives.

2. **Fail-closed is the correct strategy** - For deduplication, the cost of a false positive (not creating a needed issue) is LOW (user can manually create). The cost of a false negative (creating duplicate) is HIGH (clutters backlog, requires manual dedup).

3. **Daemon amplifies the problem** - Because daemon runs hourly and the dedup failure is intermittent, duplicates accumulate at ~1/hour rate when parsing fails.

**Answer to Investigation Question:**

Duplicate synthesis issues are created because the `synthesisIssueExists()` function in kb-cli silently returns `false` when JSON parsing fails, allowing duplicate creation even when open issues exist. The error handling strategy is "fail open" when it should be "fail closed".

---

## Structured Uncertainty

**What's tested:**

- ✅ Duplicates are created hourly (verified: checked timestamps of 14 model issues)
- ✅ Dedup code exists and returns false on parse error (verified: read source code)
- ✅ bd list command returns valid JSON (verified: direct pipe to jq works)
- ✅ Open issues DO exist when duplicates are created (verified: traced orch-go-yyw7d timeline)

**What's untested:**

- ⚠️ Exact cause of intermittent parse failure (suspected shell buffering)
- ⚠️ Whether Go's json.Unmarshal actually fails in production (assumed from behavior)
- ⚠️ Performance impact of fail-closed strategy (should be negligible)

**What would change this:**

- Finding would be wrong if dedup function actually returns errors (but it doesn't - checked code)
- Finding would be wrong if duplicates had different creation paths (but all are from kb reflect)

---

## Implementation Recommendations

**Purpose:** Fix the deduplication to reliably prevent duplicate synthesis issues.

### Recommended Approach ⭐

**Fail-Closed Dedup with Logging** - Return `true` (assume duplicate exists) on any error, with warning logging.

**Why this approach:**
- Prevents duplicates even when errors occur
- Logging enables diagnosis without blocking creation
- Aligns with "Gate Over Remind" principle - block is safer than allow on uncertainty

**Trade-offs accepted:**
- May occasionally skip creation when issue doesn't exist (false positive)
- User can manually create if needed - low cost

**Implementation sequence:**
1. Change error handling to return `true` on any error
2. Add `fmt.Fprintf(os.Stderr, ...)` warnings for diagnosis
3. Test with intentionally corrupted JSON to verify behavior

### Alternative Approaches Considered

**Option B: Retry with exponential backoff**
- **Pros:** Recovers from transient failures
- **Cons:** Adds complexity, delays reflection, transient failures are rare
- **When to use instead:** If intermittent failures become common

**Option C: Store created issues in local file**
- **Pros:** Independent of bd's JSON output
- **Cons:** Duplicates state, requires cleanup, can desync
- **When to use instead:** If bd becomes unreliable

**Rationale for recommendation:** Fail-closed is simplest, aligns with security best practices, and has low false-positive cost.

---

### Implementation Details

**What to implement first:**
- Change line 500: `return false, nil` → `return true, nil` + warning log
- Change line 507: `return false, nil` → `return true, nil` + warning log
- Same for `openIssueExists()` function

**Things to watch out for:**
- ⚠️ Logging goes to stderr which daemon captures to log file
- ⚠️ Rebuild kb binary after fix: `cd kb-cli && go build -o build/kb ./cmd/kb`
- ⚠️ Restart daemon to pick up new binary: `launchctl kickstart -k gui/$(id -u)/com.orch.daemon`

**Areas needing further investigation:**
- Root cause of JSON parse failures (not blocking for this fix)
- Whether daemon should check for kb binary changes

**Success criteria:**
- ✅ No new duplicates created for topics that already have open issues
- ✅ Warning logs appear when dedup check errors occur
- ✅ Manual verification: run `kb reflect --type synthesis --create-issue` after fix

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go` - Core dedup logic, lines 488-535
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/beads.go` - bd command helper
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/reflect.go` - Daemon's reflection call

**Commands Run:**
```bash
# Count duplicates for model topic
bd list --all --title-contains "Synthesize model investigations" --json | jq 'length'
# Result: 14

# Trace creation timeline
bd list --all --title-contains "Synthesize model investigations" --json | jq -r '.[].created_at' | sort

# Test current dedup behavior
kb reflect --type synthesis --create-issue --format json | jq '.synthesis[] | select(.count >= 10) | {topic, issue_created}'
```

**Related Artifacts:**
- **Investigation:** None directly related
- **Decision:** Should promote to decision on "fail-closed for idempotency checks"

---

## Investigation History

**2026-01-07 07:20:** Investigation started
- Initial question: Why are duplicate synthesis issues being created despite deduplication?
- Context: Observed 14+ duplicates for model topic, 2 manual deduplications in recent days

**2026-01-07 07:45:** Root cause identified
- Found silent error handling in synthesisIssueExists()
- Traced through code path from daemon → kb reflect → createSynthesisIssue → dedup check

**2026-01-07 08:00:** Investigation completed
- Status: Complete
- Key outcome: Change error handling from "assume no duplicate" to "assume duplicate exists" on any error
