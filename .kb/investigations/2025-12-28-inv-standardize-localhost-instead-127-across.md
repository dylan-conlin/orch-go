<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The orch-go codebase already uses `localhost` consistently in production code; only test files and historical workspace artifacts contain `127.0.0.1`.

**Evidence:** Grep found 17 occurrences in .go files - 14 in test files (intentional fake ports), 2 in serve_test.go (Go httptest.Server behavior), 1 in CORS middleware (correctly accepts both).

**Knowledge:** Production code uses `localhost`. Test files legitimately use `127.0.0.1:9999` for fake/unreachable servers. CORS must accept both origins. The real issue was in `features.json` documentation.

**Next:** Close - only `features.json` needed updating (done). Cross-repo skill files in orch-knowledge should be updated separately.

---

# Investigation: Standardize Localhost Instead of 127.0.0.1 Across Orch Ecosystem

**Question:** Where are `127.0.0.1` references that should be `localhost` in the orch-go codebase?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Production Code Already Uses localhost

**Evidence:** All user-facing defaults and output messages use `localhost`:
- `cmd/orch/main.go:64` - `--server` flag defaults to `http://localhost:4096`
- `cmd/orch/serve.go:103,134,261` - All serve status/startup messages use localhost
- `pkg/daemon/completion.go:78` - ServerURL default is `http://localhost:4096`
- `pkg/spawn/context.go:884` - Server URLs in spawn context use localhost

**Source:** `grep -r "localhost" --include="*.go" /Users/dylanconlin/Documents/personal/orch-go`

**Significance:** No production code changes needed - the codebase is already standardized on localhost.

---

### Finding 2: Test Files Use 127.0.0.1 for Fake/Unreachable Servers (Intentional)

**Evidence:** Test files use `127.0.0.1:9999` or similar for intentionally invalid/unreachable servers:
- `pkg/daemon/completion_test.go` - 12 occurrences of `http://127.0.0.1:9999` for fake servers
- `pkg/opencode/client_test.go:496` - `http://127.0.0.1:99999` for invalid port test
- `pkg/opencode/sse_test.go:365` - `http://127.0.0.1:99999/event` for unreachable SSE test

**Source:** `grep -r "127\.0\.0\.1" --include="*.go" /Users/dylanconlin/Documents/personal/orch-go`

**Significance:** These are intentional - using `127.0.0.1:9999` (a typically unbound port) for testing error handling. No change needed.

---

### Finding 3: Go httptest.Server Uses 127.0.0.1 Format (SDK Behavior)

**Evidence:** In `cmd/orch/serve_test.go:318-320`:
```go
// The URL is in format http://127.0.0.1:PORT (httptest.Server always uses 127.0.0.1)
_, err := fmt.Sscanf(server.URL, "http://127.0.0.1:%d", &testPort)
```

**Source:** `cmd/orch/serve_test.go:318-320`

**Significance:** Go's `httptest.Server` always returns URLs with `127.0.0.1`. This is standard library behavior and cannot/should not be changed.

---

### Finding 4: CORS Middleware Correctly Accepts Both (Keep As-Is)

**Evidence:** In `cmd/orch/serve.go:181`:
```go
if origin == "" || strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
```

**Source:** `cmd/orch/serve.go:181`

**Significance:** CORS must accept both `localhost` and `127.0.0.1` origins because browsers may send either. This is correct behavior and should not be changed.

---

### Finding 5: features.json Had Outdated 127.0.0.1 Reference (Fixed)

**Evidence:** `.orch/features.json` line 186 contained:
```
http://127.0.0.1:4096/healthz ... http://127.0.0.1:3348/api/sessions
```

**Source:** `.orch/features.json:186`

**Significance:** This was documentation text in a feature description. Updated to use `localhost`.

---

### Finding 6: Cross-Repo Skill Files Have 127.0.0.1 References (Out of Scope)

**Evidence:** Skill files in orch-knowledge repo contain `127.0.0.1:3333` references:
- `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/SKILL.md:360,367,432`
- `/Users/dylanconlin/orch-knowledge/skills/src/policy/orchestrator/SKILL.md:190`

**Source:** `grep -r "127\.0\.0\.1" /Users/dylanconlin/orch-knowledge/skills`

**Significance:** These are in a different repo (orch-knowledge) and should be updated separately. Filed as discovered work.

---

## Synthesis

**Key Insights:**

1. **Codebase is already standardized** - Production code consistently uses `localhost`. The perceived issue was likely from historical workspace artifacts or external documentation.

2. **Test file usage is intentional** - Using `127.0.0.1:9999` for unreachable/fake servers is a valid testing pattern. These should not be changed to `localhost`.

3. **CORS must accept both** - Web browsers may send either `localhost` or `127.0.0.1` as the Origin header. The CORS middleware correctly handles both.

**Answer to Investigation Question:**

The orch-go codebase has minimal 127.0.0.1 references that need changing:
- ✅ **features.json** - Fixed (documentation text)
- ⏸️ **orch-knowledge skills** - Out of scope (different repo)
- ❌ **Test files** - Intentional, no change needed
- ❌ **CORS middleware** - Correct behavior, no change needed
- ❌ **serve_test.go** - Go stdlib behavior, cannot change

---

## Structured Uncertainty

**What's tested:**

- ✅ Production code uses localhost (verified: grep found 50+ localhost occurrences in .go files)
- ✅ Test files use 127.0.0.1 for fake servers (verified: all 14 occurrences are in test files with port 9999)
- ✅ CORS accepts both origins (verified: code review of serve.go:181)

**What's untested:**

- ⚠️ Why localhost works but 127.0.0.1 didn't for the user (could be DNS, hosts file, or browser configuration)

**What would change this:**

- Finding production code that outputs 127.0.0.1 to users would require changes
- Finding localhost failures in tests would require reconsideration

---

## Implementation Recommendations

### Recommended Approach ⭐

**Minimal changes - only fix documentation** - The codebase is already standardized on localhost. Only update documentation artifacts that reference 127.0.0.1.

**Why this approach:**
- Production code already correct
- Test patterns are intentional and valid
- CORS behavior is correct

**Implementation sequence:**
1. ✅ Fix features.json description (done)
2. Create follow-up issue for orch-knowledge skill files

---

## References

**Files Examined:**
- `cmd/orch/serve.go` - Main serve command, CORS middleware
- `cmd/orch/main.go` - Root command, server URL flag
- `pkg/daemon/completion.go` - Default ServerURL
- All test files containing 127.0.0.1

**Commands Run:**
```bash
# Find 127.0.0.1 in Go files
grep -r "127\.0\.0\.1" --include="*.go" .

# Find localhost in Go files
grep -r "localhost" --include="*.go" .

# Find in .orch directory
grep -r "127\.0\.0\.1" .orch/
```

---

## Investigation History

**2025-12-28:** Investigation started
- Initial question: Where are 127.0.0.1 references that need updating?
- Context: User reported localhost:5188 works but 127.0.0.1:5188 doesn't

**2025-12-28:** Investigation completed
- Status: Complete
- Key outcome: Codebase already uses localhost; only features.json documentation needed updating
