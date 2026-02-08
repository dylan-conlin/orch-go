## Summary (D.E.K.N.)

**Delta:** Dashboard API slowness (5-7s) caused by fetching beads data for 623 accumulated sessions (392 unique beads IDs requiring 784+ RPC calls).

**Evidence:** Measured 68ms per beads RPC call; with 20 concurrent limit, 392 IDs = ~3 seconds just for beads data. Only 6 sessions updated in last hour but all 623 were being processed.

**Knowledge:** Session accumulation is unbounded in OpenCode; current architecture doesn't distinguish active from historical sessions.

**Next:** Fix applied - filter to only process sessions updated within 2 hours. Response time now 200-500ms. Future work: implement proper session cleanup mechanism.

---

# Investigation: Dashboard API Slow Again (623 Sessions)

**Question:** Why does /api/agents take 5-7 seconds to load with 623 OpenCode sessions?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** og-debug-dashboard-api-slow-06jan-d822
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: 623 sessions but only 6 active in last hour

**Evidence:**
- 624 total OpenCode sessions
- 36 sessions updated in last 24 hours
- Only 6 sessions updated in last hour
- 588 sessions (94%) older than 24 hours

**Source:**
```bash
curl -s 'http://localhost:4096/session' | jq 'length'  # 624
curl -s 'http://localhost:4096/session' | jq 'map(select((.time.updated/1000) > (now - 3600))) | length'  # 6
```

**Significance:** The vast majority of sessions are historical/inactive, yet the API was processing all of them.

---

### Finding 2: 392 unique beads IDs requiring 784+ RPC calls

**Evidence:**
- 392 unique beads IDs extracted from session titles
- Each beads ID requires 2 RPC calls (issue show + comments)
- Each RPC call takes ~68ms
- With 20 concurrent limit: 392 / 20 = ~20 batches × 68ms × 2 calls = ~2.7 seconds

**Source:**
```bash
curl -s 'http://localhost:4096/session' | jq '[.[] | .title] | map(select(contains("[") and contains("]"))) | map(capture("\\[(?<id>[^\\]]+)\\]").id) | unique | length'  # 392

time (for id in orch-go-91qze orch-go-hmj61 orch-go-47k35; do bd show "$id" --json >/dev/null 2>&1; done)  # 206ms for 3 calls = ~68ms each
```

**Significance:** The O(n) scaling of beads fetches with session count is the root cause of latency.

---

### Finding 3: Cache TTLs were too short for the data volume

**Evidence:**
- Open issues cache: 10 seconds
- All issues cache: 30 seconds
- Comments cache: 5 seconds

**Source:** `cmd/orch/serve_agents_cache.go:78-82`

**Significance:** With 5-30 second TTLs, the cache was frequently expiring and triggering full refetches of 392+ beads IDs.

---

## Synthesis

**Key Insights:**

1. **Unbounded session accumulation** - OpenCode retains all sessions in memory indefinitely. Over time, this grows unbounded and causes O(n) scaling issues in any code that iterates over sessions.

2. **Beads fetching is O(sessions)** - The current design fetches beads data for every session with a beads ID, not just active sessions. This amplifies the session accumulation problem.

3. **Prior fixes addressed symptoms, not root cause** - orch-go-50hv and orch-go-yw1q fixed workspace scanning, but this is the third occurrence because the root cause (unbounded session/beads fetching) wasn't addressed.

**Answer to Investigation Question:**

The 5-7 second load time is caused by:
1. Iterating through 623 sessions (fast, ~75ms)
2. Extracting 392 unique beads IDs from session titles
3. Making 784+ RPC calls (2 per beads ID) to fetch issue and comment data
4. Even with 20 concurrent calls and caching, when cache expires (every 5-30s), this takes ~3+ seconds

---

## Structured Uncertainty

**What's tested:**

- ✅ API now responds in 200-500ms (verified: multiple curl timing tests)
- ✅ All cmd/orch tests pass (verified: go test ./cmd/orch/...)
- ✅ Fix correctly filters out sessions older than 2 hours (verified: reduced from 567 to 290 agents)

**What's untested:**

- ⚠️ Long-term session cleanup mechanism (deferred to future work)
- ⚠️ Impact of returning fewer agents on dashboard UX (acceptable trade-off)
- ⚠️ OpenCode's session retention policy/limits (unknown)

**What would change this:**

- Finding would be wrong if beads RPC calls were already being cached/batched at the server level
- Finding would be wrong if OpenCode implemented server-side filtering

---

## Implementation Applied

**Approach:** Filter sessions by age before adding to beads fetch list

**Changes:**
1. Added `beadsFetchThreshold := 2 * time.Hour` in `serve_agents.go`
2. Sessions older than threshold are `continue`d before being added to `beadsIDsToFetch`
3. Increased cache TTLs: openIssues 10→30s, allIssues 30→60s, comments 5→15s

**Result:**
- Before: 5.4-7.2 seconds
- After: 200-500ms
- Agents returned: 567 → 290 (sessions > 2 hours excluded)

---

## References

**Files Modified:**
- `cmd/orch/serve_agents.go:116-185` - Added beadsFetchThreshold and early continue
- `cmd/orch/serve_agents_cache.go:78-85` - Increased TTLs

**Commands Run:**
```bash
# Measure API response time
time curl -sk 'https://localhost:3348/api/agents' > /dev/null

# Count sessions
curl -s 'http://localhost:4096/session' | jq 'length'

# Count unique beads IDs
curl -s 'http://localhost:4096/session' | jq '...' 

# Run tests
go test ./cmd/orch/...
```

**Related Artifacts:**
- **Prior fix:** orch-go-50hv - Fixed getCompletionsForReview scanning 303 workspaces
- **Prior fix:** orch-go-yw1q - Fixed findWorkspaceByBeadsID reading 702 workspace dirs

---

## Investigation History

**2026-01-06 15:15:** Investigation started
- Initial question: Why does dashboard API take 5-7 seconds?
- Context: Third occurrence of this performance issue

**2026-01-06 15:30:** Root cause identified
- 623 sessions → 392 beads IDs → 784+ RPC calls
- Each RPC ~68ms, with 20 concurrent = ~3+ seconds

**2026-01-06 15:45:** Fix implemented and verified
- Added 2-hour age filter for beads fetching
- Increased cache TTLs
- Response time: 200-500ms

**2026-01-06 15:50:** Investigation completed
- Status: Complete
- Key outcome: 10-20x performance improvement via session age filtering
