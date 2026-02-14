# Synthesis: Eliminate pkg/registry

## What Changed
- Deleted `pkg/registry/` (529 lines) entirely
- Replaced all 4 consumers with workspace file reads (AGENT_MANIFEST.json, .session_id)
- Added `Model` field to `AgentManifest` struct to close the last data gap
- Removed `registerAgent()` from spawn_cmd.go (4 call sites + function definition)
- Replaced status_cmd.go Phase 1 with workspace directory scanning
- Removed `cleanInactiveRegistryEntries()` from clean_cmd.go

## Verification
- `go build ./cmd/orch/` -- passes
- `go vet ./...` -- passes
- `go test ./...` -- no new failures (only pre-existing pkg/model test failures)
- `grep "pkg/registry"` -- zero matches remaining
