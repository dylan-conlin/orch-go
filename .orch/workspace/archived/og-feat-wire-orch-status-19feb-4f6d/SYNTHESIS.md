# Synthesis: Wire orch status CLI and dashboard /api/agents to new query path

## TLDR
Rewired both `orch status` CLI and dashboard `/api/agents` to use `queryTrackedAgents` single-pass query engine, replacing ~1000 lines of ad-hoc 3-phase discovery with ~600 lines using the cached query engine. Added 3s TTL cache for dashboard, injectable `queryTrackedAgentsFn` for testing, and `Reason` field for surfacing partial-metadata reason codes.

## Outcome: success

## Recommendation: close

## Delta

### Files Modified
- `cmd/orch/status_cmd.go` — Replaced `runStatus()` discovery (~470 lines of workspace scan → tmux → sessions) with single `queryTrackedAgents()` call + `agentStatusToAgentInfo()` conversion. Removed `spawn` and `tmux` imports.
- `cmd/orch/serve_agents_handlers.go` — Replaced `handleAgents()` discovery (~600 lines of beads-first + workspace scan + session cross-ref) with cached `queryTrackedAgents`. Added `agentStatusToAPIResponse()` conversion. Removed `agent` import and `issueHasAgentMetadata` function.
- `cmd/orch/serve_agents_cache.go` — Added `trackedAgentsCache` struct (3s TTL, RWMutex, project-dirs match validation), `globalTrackedAgentsCache` singleton, and injectable `queryTrackedAgentsFn` variable.
- `cmd/orch/serve_agents_cache_handler.go` — Added `globalTrackedAgentsCache.invalidate()` to cache invalidation handler.
- `cmd/orch/serve_agents_types.go` — Added `Reason` field to `AgentAPIResponse` for surfacing reason codes.
- `cmd/orch/serve_agents_handlers_test.go` — Rewrote 5 failing tests to mock `queryTrackedAgentsFn` instead of old discovery functions. Added test helper `setupHandlerTest`/`restore` pattern. Enhanced `newTestOpenCodeServer` to handle `/session/status` and list endpoints.

### Key Design Decisions
1. **CLI gets no cache** (short-lived process, full query each time) vs **dashboard gets 3s TTL cache** (long-lived server, 30s poll interval)
2. **Injectable function variable** (`queryTrackedAgentsFn`) follows existing codebase pattern (`getIssuesBatch`, `listOpenIssues`) for test mockability
3. **Enrichment preserved** — Both CLI and dashboard keep all enrichment layers (tokens, activity, investigation paths, synthesis, gap analysis, stall detection) on top of the query engine results
4. **agentStatusToAPIResponse maps status codes**: active→active, idle→dead, retrying→active, unknown→dead (with reason codes)

## Evidence
- `go build ./cmd/orch/` — clean
- `go vet ./cmd/orch/` — clean
- `go test ./cmd/orch/` — all tests pass (including 7 handler tests, 11 query engine tests)

## Next Actions
- Monitor dashboard latency in production to verify <500ms target
- Consider removing now-unused functions from `serve_agents_discovery.go` if they're no longer called anywhere
