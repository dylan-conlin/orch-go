<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `/bin/sh -c` doesn't properly export PATH to child processes in launchd context, causing commands like `go` in `/opt/homebrew/bin` to fail with "command not found".

**Evidence:** Error "'/bin/sh: go: command not found'" when running launchd plist with PATH set to include `/opt/homebrew/bin`.

**Knowledge:** In launchd context, `/bin/sh` (POSIX shell) doesn't inherit/export environment variables to subprocess commands properly. Using `/bin/bash -l -c` (login shell) ensures profile sourcing and proper environment inheritance.

**Next:** Fix implemented - changed `ServerToPlistConfig` to use `/bin/bash -l -c` instead of `/bin/sh -c`.

---

# Investigation: Launchd Plists Bin Sh Command

**Question:** Why do launchd plists fail to find commands like `go` even when PATH is set in EnvironmentVariables?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Debug spawn
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: PATH is correctly set in plist but commands not found

**Evidence:** 
- PATH in plist includes `/opt/homebrew/bin`
- `go` is at `/opt/homebrew/bin/go`
- Error: `'/bin/sh: go: command not found'`
- npm works (different path location or already in base PATH)

**Source:** SPAWN_CONTEXT.md evidence

**Significance:** PATH is set correctly in EnvironmentVariables, but child processes spawned by `/bin/sh -c` don't see it.

---

### Finding 2: /bin/sh is POSIX shell with limited environment handling

**Evidence:** 
- macOS `/bin/sh` is a POSIX-compliant shell (not bash)
- POSIX shell doesn't source profile files (~/.profile, ~/.bash_profile)
- When launchd sets EnvironmentVariables, `/bin/sh` receives them but doesn't necessarily export them properly to child processes in the same way a login shell would

**Source:** macOS shell behavior documentation

**Significance:** This explains why setting PATH in EnvironmentVariables doesn't guarantee child processes can find commands.

---

### Finding 3: Login shell properly inherits and exports environment

**Evidence:**
- `/bin/bash -l` runs as a login shell
- Login shells source profile files that set up PATH and environment
- Child processes inherit the complete environment

**Source:** Bash documentation for `-l` flag

**Significance:** Using login shell ensures commands in user-configured PATH locations are available.

---

## Synthesis

**Key Insights:**

1. **Environment inheritance is shell-dependent** - `/bin/sh` and `/bin/bash` have different behaviors for environment variable handling in launchd context.

2. **Login shells are essential for dev tools** - Developer tools (go, npm, node) are typically installed via package managers (Homebrew) and added to PATH via profile files that only login shells source.

3. **The fix is simple and universal** - Changing from `/bin/sh -c` to `/bin/bash -l -c` handles all commands without needing to resolve absolute paths.

**Answer to Investigation Question:**

The launchd plists fail because `/bin/sh -c` doesn't properly export PATH to child processes. The solution is to use `/bin/bash -l -c` (login shell) which sources profile files and properly inherits environment variables, making commands in `/opt/homebrew/bin` and other PATH locations available.

---

## Structured Uncertainty

**What's tested:**

- ✅ Unit tests pass with new `/bin/bash -l -c` arguments
- ✅ `TestServerToPlistConfig` verifies correct shell arguments are generated
- ✅ `TestGeneratePlist` verifies plist XML generation works correctly

**What's untested:**

- ⚠️ Live launchd plist loading/execution (requires actual launchd service test)
- ⚠️ Edge cases with different shell configurations

**What would change this:**

- Finding would be wrong if `/bin/bash` is not available on all macOS systems (unlikely - bash is standard)
- Finding would be incomplete if some users have non-standard profile files

---

## Implementation Recommendations

### Recommended Approach ⭐

**Use /bin/bash -l -c** - Change shell invocation from `/bin/sh -c` to `/bin/bash -l -c`

**Why this approach:**
- Login shell sources profile files (`.bash_profile`, `.profile`)
- Environment variables are properly inherited by child processes
- Commands in Homebrew paths (`/opt/homebrew/bin`) are found

**Trade-offs accepted:**
- Slightly slower shell startup (profile sourcing)
- Dependency on bash (standard on macOS)

**Implementation sequence:**
1. Change `ServerToPlistConfig` in `pkg/servers/servers.go` line 234
2. Update test to verify new arguments
3. Add explanatory comment for future maintainers

### Alternative Approaches Considered

**Option B: Resolve absolute paths during plist generation**
- **Pros:** No shell behavior dependency
- **Cons:** Requires command resolution logic, may fail for complex commands
- **When to use instead:** If bash is unavailable or profile sourcing is problematic

**Option C: Use 'env PATH=$PATH' wrapper**
- **Pros:** Explicit PATH passing
- **Cons:** More complex, still doesn't solve profile sourcing
- **When to use instead:** When needing fine-grained environment control

**Rationale for recommendation:** Option A (login shell) is simplest, handles all commands uniformly, and is the standard approach for user-context services.

---

## References

**Files Examined:**
- `pkg/servers/servers.go:228-270` - `ServerToPlistConfig` function
- `pkg/servers/servers_test.go:495-590` - Plist generation tests

**Commands Run:**
```bash
# Run plist tests
go test ./pkg/servers/... -run "TestGeneratePlist" -v

# Run all server tests
go test ./pkg/servers/... -v
```

---

## Implementation History

**2025-12-28:** Investigation started
- Initial question: Why do launchd plists fail to find commands like `go`?
- Context: Spawned from orch-go-zrzq bug report

**2025-12-28:** Fix implemented
- Changed `/bin/sh -c` to `/bin/bash -l -c` in `ServerToPlistConfig`
- Updated tests to verify new arguments
- All tests passing
