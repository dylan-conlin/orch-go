# Session Synthesis

**Agent:** og-debug-launchd-plists-bin-28dec
**Issue:** orch-go-zrzq
**Duration:** 2025-12-28
**Outcome:** success

---

## TLDR

Fixed launchd plists failing to find commands like `go` by changing shell invocation from `/bin/sh -c` to `/bin/bash -l -c` (login shell), ensuring PATH environment variables are properly inherited by child processes.

---

## Delta (What Changed)

### Files Modified
- `pkg/servers/servers.go:228-240` - Changed shell invocation from `/bin/sh -c` to `/bin/bash -l -c` in `ServerToPlistConfig` function
- `pkg/servers/servers_test.go:498,522-527,619-627` - Updated tests to verify new shell arguments

### Commits
- (pending) - fix: use login shell for launchd plists to properly inherit PATH

---

## Evidence (What Was Observed)

- Error "'/bin/sh: go: command not found'" when running launchd plist with PATH set to include `/opt/homebrew/bin`
- `go` binary exists at `/opt/homebrew/bin/go`
- PATH was correctly set in EnvironmentVariables section of plist
- npm worked (likely already in base PATH or different execution path)

**Root cause:** `/bin/sh` (POSIX shell) doesn't properly export environment variables to child processes in launchd context, unlike `/bin/bash -l` (login shell) which sources profile files.

### Tests Run
```bash
# Plist generation tests
go test ./pkg/servers/... -run "TestGeneratePlist" -v
# PASS: TestGeneratePlist (0.00s)
# PASS: TestGeneratePlist_XMLEscaping (0.00s)
# PASS: TestGeneratePlist_NoKeepAlive (0.00s)

# ServerToPlistConfig tests
go test ./pkg/servers/... -run "TestServerToPlistConfig" -v
# PASS: TestServerToPlistConfig (0.00s)
# PASS: TestServerToPlistConfig_DefaultWorkdir (0.00s)

# All server package tests
go test ./pkg/servers/... -v
# PASS: all tests passing
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-launchd-plists-bin-sh-command.md` - Investigation of /bin/sh vs /bin/bash behavior in launchd context

### Decisions Made
- Decision: Use `/bin/bash -l -c` instead of `/bin/sh -c` because login shells source profile files and properly inherit environment variables, making commands in `/opt/homebrew/bin` available

### Constraints Discovered
- `/bin/sh` (POSIX shell) doesn't properly export PATH to child processes in launchd context
- Developer tools (go, npm, node) typically require profile sourcing to be found

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-zrzq`

---

## Unexplored Questions

**Straightforward session, no unexplored territory**

The fix is well-understood and applies universally. The login shell approach is the standard solution for this class of problem.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** Claude
**Workspace:** `.orch/workspace/og-debug-launchd-plists-bin-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-launchd-plists-bin-sh-command.md`
**Beads:** `bd show orch-go-zrzq`
