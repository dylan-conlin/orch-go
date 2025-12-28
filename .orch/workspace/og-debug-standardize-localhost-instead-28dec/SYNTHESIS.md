# Session Synthesis

**Agent:** og-debug-standardize-localhost-instead-28dec
**Issue:** orch-go-nqoi
**Duration:** 2025-12-28
**Outcome:** success

---

## TLDR

Standardized all user-facing URLs from `127.0.0.1` to `localhost` across the orch-go codebase because localhost works while 127.0.0.1 didn't work for the user. Updated ~50 occurrences across CLI defaults, web frontend, documentation, and tests.

---

## Delta (What Changed)

### Files Modified

**CLI Core:**
- `cmd/orch/main.go` - Default `--server` flag: `http://127.0.0.1:4096` → `http://localhost:4096`
- `cmd/orch/serve.go` - Server status output, startup messages, pprof comment
- `cmd/orch/doctor.go` - Health check URLs
- `cmd/orch/focus.go` - OpenCode client URL
- `cmd/gendoc/main.go` - Generated docs use localhost

**Package Defaults:**
- `pkg/daemon/daemon.go` - Default serverURL fallback
- `pkg/daemon/completion.go` - DefaultCompletionServiceConfig
- `pkg/tmux/tmux.go` - OpencodeAttachConfig comment

**Web Frontend (all stores):**
- `web/src/lib/stores/agents.ts` - API_BASE
- `web/src/lib/stores/agentlog.ts` - API_BASE
- `web/src/lib/stores/beads.ts` - API_BASE
- `web/src/lib/stores/config.ts` - API_BASE
- `web/src/lib/stores/daemon.ts` - API_BASE
- `web/src/lib/stores/focus.ts` - API_BASE
- `web/src/lib/stores/pending-reviews.ts` - API_BASE
- `web/src/lib/stores/servers.ts` - API_BASE
- `web/src/lib/stores/usage.ts` - API_BASE
- `web/vite.config.ts` - Proxy target URLs

**Documentation:**
- `README.md` - All API pattern examples
- `CLAUDE.md` - Architecture overview
- `docs/cli/*.md` - All 30 generated CLI docs

**Tests:**
- `pkg/opencode/client_test.go` - Standard client URLs
- `pkg/opencode/sse_test.go` - SSE client URL
- `pkg/tmux/tmux_test.go` - Config URLs
- `pkg/state/reconcile_test.go` - Server URLs
- `pkg/daemon/completion_test.go` - Default config assertion
- `web/tests/load-test.spec.ts` - API base URL
- `web/test-race-condition.html` - API URLs

**Legacy:**
- `legacy/main.go` - Default ServerURL
- `legacy/main_test.go` - Test URLs

### Commits
- (to be committed)

---

## Evidence (What Was Observed)

- Initial grep found ~100 occurrences of `127.0.0.1`
- User reported `127.0.0.1:5188` didn't work but `localhost:5188` does
- CORS handler in serve.go correctly accepts both `localhost` AND `127.0.0.1` origins (kept as-is)

### Categorization of 127.0.0.1 occurrences:

| Category | Action | Count |
|----------|--------|-------|
| User-facing defaults | Changed to localhost | ~30 |
| Web frontend API_BASE | Changed to localhost | 9 |
| Documentation | Changed to localhost | ~20 |
| Tests (standard flow) | Changed to localhost | ~15 |
| CORS handling | Keep both (correct) | 1 |
| Tests (invalid port 99999/9999) | Keep (error testing) | ~13 |
| httptest.Server parsing | Keep (Go stdlib uses 127.0.0.1) | 2 |
| Generated logs | Will update on next run | 1 |

### Tests Run
```bash
# Go build is not available in this environment - requires verification by orchestrator
# All changes are text substitutions that should not break compilation
```

---

## Knowledge (What Was Learned)

### Key Finding
- `localhost` and `127.0.0.1` are NOT always interchangeable
- Some network configurations, DNS, or application settings may resolve them differently
- Using `localhost` is more user-friendly and often more reliable

### Constraints Discovered
- CORS handler must accept BOTH `localhost` and `127.0.0.1` origins (browsers may send either)
- Go's httptest.Server always uses 127.0.0.1 format (cannot change)
- Tests for error handling can use arbitrary URLs (no functional difference)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [ ] Tests passing (requires orchestrator to run `go test ./...`)
- [x] Changes consistent across codebase
- [ ] Ready for `orch complete orch-go-nqoi`

### Post-merge verification
After building and installing the new binary:
1. `orch serve` should show `http://localhost:3348`
2. `orch status` should work with the new default URL
3. Web dashboard at `http://localhost:5188` should connect successfully

---

## Unexplored Questions

- Why did 127.0.0.1 not work while localhost did? (Could be DNS, hosts file, or browser behavior)
- Should the CORS handler use a configurable list of allowed origins?

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** (orchestrator context)
**Workspace:** `.orch/workspace/og-debug-standardize-localhost-instead-28dec/`
**Beads:** `bd show orch-go-nqoi`
