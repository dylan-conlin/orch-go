# Registry Removal Phase 4 - SYNTHESIS

## TLDR
Completed the removal of the global `~/.orch/agent-registry.json` file from orch-go. All commands now use workspace-local `.session_id` files and derive agent state from OpenCode API + tmux. The `pkg/registry/` package has been deleted.

## Outcome
success

## Recommendation
close

## Delta

### Files Deleted
- `pkg/registry/registry.go` - Global registry implementation
- `pkg/registry/registry_test.go` - Registry tests

### Files Modified
- `cmd/orch/main.go` - Already updated in prior phases (spawn, status, complete, clean, etc.)
- `cmd/orch/review.go` - Already updated to use workspace scanning
- `cmd/orch/review_test.go` - Rewritten to remove registry.Agent references
- `cmd/orch/status_test.go` - Rewritten to remove registry usage
- `cmd/orch/clean_test.go` - Rewritten to test workspace-based cleanup
- `cmd/orch/main_test.go` - Removed registry imports, kept helper function tests
- `cmd/orch/serve_test.go` - Fixed AgentWithSynthesis → AgentAPIResponse rename
- `cmd/orch/focus.go` - `getActiveIssues()` now uses OpenCode API instead of registry
- `pkg/opencode/service.go` - Removed registry dependency from CompletionService

## Evidence

### Build Verification
```
$ go build ./...
# Success - no errors
```

### Test Verification
```
$ go test ./...
ok  	github.com/dylan-conlin/orch-go	(cached)
ok  	github.com/dylan-conlin/orch-go/cmd/orch	0.166s
ok  	github.com/dylan-conlin/orch-go/legacy	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/account	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/capacity	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/daemon	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/events	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/focus	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/model	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/notify	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/opencode	0.619s
ok  	github.com/dylan-conlin/orch-go/pkg/port	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/question	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/skills	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/spawn	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/tmux	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/usage	(cached)
ok  	github.com/dylan-conlin/orch-go/pkg/verify	(cached)
```

### No More Registry Imports
```
$ rg '"github.com/dylan-conlin/orch-go/pkg/registry"' --type go -l
# No output - all imports removed
```

## Knowledge

### Key Design Decisions
1. **Session ID Storage:** Workspace file `.orch/workspace/{name}/.session_id` (already implemented in prior phases)
2. **Active Count:** OpenCode API `ListSessions()` for concurrency limiting
3. **Agent Lookup:** Workspace file first, then derived lookup (tmux + OpenCode title matching)
4. **Completed Detection:** SYNTHESIS.md existence in workspace
5. **No Registry:** Registry is completely removed, not minimized

### Migration Pattern
- Commands that wrote to registry now write session ID to workspace file
- Commands that read from registry now query OpenCode API or scan workspaces
- Tests that tested registry behavior now test workspace-based behavior

### What Changed for Each Command
| Command | Old Approach | New Approach |
|---------|--------------|--------------|
| `spawn` | Register agent in registry | Write session ID to workspace |
| `status` | Read active agents from registry | Query OpenCode ListSessions() |
| `complete` | Mark complete in registry | Check SYNTHESIS.md exists |
| `clean` | Remove from registry | Remove workspace directories |
| `focus/drift` | Query registry for active issues | Query OpenCode for sessions |

## Next Actions
- None - registry removal is complete
- Consider cleaning up any remaining references to "registry" in documentation
