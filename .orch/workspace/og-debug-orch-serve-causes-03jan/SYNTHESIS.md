# SYNTHESIS: orch serve CPU spike fix

## TLDR

Fixed CPU spike in `orch serve` by adding TTL-based caching for beads data and workspace metadata, plus limiting beads fetches to active agents only. Response time dropped from **90+ seconds to 0.13s** (cached), bd processes from **20+ to 1-2**.

## Problem

When the dashboard polled `/api/agents`, orch serve was spawning bd processes for every workspace (600+) without any caching:
- 20+ concurrent `bd` processes per request
- 90+ second response times
- High CPU usage during dashboard polling

## Root Cause

Three compounding issues in `cmd/orch/serve_agents.go`:

1. **No caching** - Every request re-fetched all beads data
2. **All workspaces scanned** - Including 500+ historical/completed ones
3. **No activity filtering** - Beads fetched even for agents inactive for days

## Solution

### Changes Made

1. **TTL-based beads cache** (`beadsCache` struct):
   - Open issues: 10s TTL
   - All issues: 30s TTL  
   - Comments: 5s TTL

2. **Workspace metadata cache** (`globalWorkspaceCacheType`):
   - SPAWN_CONTEXT.md parsing: 30s TTL

3. **Active-agent filtering**:
   - Only fetch beads for agents updated in last 10 minutes
   - Skip token fetching for idle/completed agents

4. **Test fix**:
   - Initialize `globalBeadsCache` in test setup

### Files Modified

- `cmd/orch/serve.go` - Initialize globalBeadsCache in runServe()
- `cmd/orch/serve_agents.go` - Add caching infrastructure and optimizations
- `cmd/orch/serve_agents_test.go` - Initialize cache in test

## Results

| Metric | Before | After |
|--------|--------|-------|
| Response time (first) | 90+ seconds | ~15 seconds |
| Response time (cached) | 90+ seconds | **0.13 seconds** |
| bd processes per request | 20+ concurrent | 1-2 |
| CPU during polling | High spike | Minimal |

## Verification

```bash
# Response time test
time curl -s localhost:5188/api/agents | jq '.agents | length'
# First: ~15s, Subsequent: 0.13s

# bd process count
ps aux | grep bd | wc -l
# During request: 1-2 (was 20+)

# Tests pass
go test ./cmd/orch/... -run TestHandleAgents
```

## Commit

```
dd76a979 fix: add TTL caching to orch serve to prevent CPU spike from bd process spawning
```

## Next

- [x] Code changes committed
- [x] Investigation updated
- [x] SYNTHESIS.md created
- [ ] Report Phase: Complete
