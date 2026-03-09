# Session Synthesis

**Agent:** og-inv-post-mortem-skillc-09mar-e14c
**Issue:** orch-go-7r4l3
**Duration:** 2026-03-09
**Outcome:** success

---

## Plain-Language Summary

On March 4, a literal `~` directory was created inside the orch-go repo root by `skillc test`'s isolation directory feature. The directory contained Claude Code config files (`.claude.json`, `plugins/`, `debug/`), meaning the Claude CLI wrote its config to `./~/.skillc/test-env/` instead of `/Users/dylanconlin/.skillc/test-env/`. This happened because Go treats `~` as a literal character — unlike a shell. Four days later, a human tried to clean it up with `rm -rf ~/`, which the shell expanded to the home directory, deleting everything and requiring a Time Machine restore.

The root cause is in `skillc/cmd/skillc/test_cmd.go`: the `--config-dir` flag value is used without calling `expandHome()`, while all other path flags (`--scenarios`, `--variant`, `--transcripts`) do call it. When `skillc test` is invoked programmatically (via `exec.Command` or quoted tilde), the literal `~/.skillc/test-env/` path reaches `SetupAuth()` and `SetupHooks()` which perform filesystem operations at that literal path. I added a `.gitignore` entry for `~` in orch-go as an immediate safety net, and documented CROSS_REPO_ISSUE for the skillc fix.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification steps.

Key outcomes:
- Root cause identified: missing `expandHome(configDir)` in `skillc/test_cmd.go:75`
- `.gitignore` entry added to prevent `~` directories from being tracked
- CROSS_REPO_ISSUE documented for skillc fix
- Open question `kb-59dd35` answered

---

## TLDR

Post-mortem traced the `./~/` directory to a missing `expandHome()` call on `skillc test`'s `--config-dir` flag. Go's filesystem APIs treat `~` literally, creating a `~` directory at CWD. Added `.gitignore` guard; skillc fix needs a CROSS_REPO_ISSUE.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-09-inv-post-mortem-skillc-bug-agent.md` - Full investigation with D.E.K.N., 5 findings, prevention recommendations

### Files Modified
- `.gitignore` - Added `~` entry to prevent literal tilde directories from being tracked

### Commits
- (pending)

---

## Evidence (What Was Observed)

- Session `e8ae39d9` transcript confirmed `~/` directory contained Claude Code config artifacts created March 4
- `skillc/cmd/skillc/test_cmd.go:75` assigns `configDir = os.Args[i+1]` without `expandHome()`
- Lines 119, 121, 183 DO call `expandHome()` on other path flags
- `testEnv()` in `runner.go:443-448` expands `~/` for `CLAUDE_CONFIG_DIR` env var but NOT for `SetupAuth`/`SetupHooks` filesystem operations
- Go test confirmed `os.UserHomeDir()` returns correct `/Users/dylanconlin` (so `defaultIsolationDir()` default path is safe)
- `defaultIsolationDir()` and `defaultBehavioralIsolationDir()` both use `os.UserHomeDir()` correctly — bug only triggers via explicit `--config-dir` with unexpanded tilde

### Tests Run
```bash
# Verified tilde expansion behavior
go run /tmp/test_tilde.go
# expandHome("~/.skillc/test-env/") → "/Users/dylanconlin/.skillc/test-env"
# testEnv("~/.skillc/test-env/") → "CLAUDE_CONFIG_DIR=/Users/dylanconlin/.skillc/test-env"
```

---

## Architectural Choices

### Investigation mode vs probe mode
- **What I chose:** Full investigation (no injected model-claim markers in SPAWN_CONTEXT)
- **What I rejected:** Probe against spawn-architecture or orchestrator-session-lifecycle models
- **Why:** Post-mortem is a novel investigation question, not testing a model claim

### Defense-in-depth vs single fix
- **What I chose:** Multi-layer (fix skillc source + orch-go .gitignore + constraint)
- **What I rejected:** Only fixing skillc
- **Why:** The root cause is in skillc but orch-go can defend itself independently; other tools may also create `~` directories
- **Risk accepted:** `.gitignore` hides rather than prevents — but it stops the `~` directory from appearing in `git status`, which is where humans notice it

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-09-inv-post-mortem-skillc-bug-agent.md` - Root cause analysis of `~/` directory creation and home directory deletion

### Constraints Discovered
- Go's `os.MkdirAll`, `os.WriteFile`, `os.Symlink` treat `~` as a literal character — every path with potential `~` must be expanded before filesystem operations
- `expandHome()` must be called on ALL user-provided paths, not just some — the asymmetry between `configDir` and other paths was the vulnerability

---

## Next (What Should Happen)

**Recommendation:** close (with CROSS_REPO_ISSUE for skillc)

### If Close
- [x] Investigation file complete with D.E.K.N. and 5 findings
- [x] `.gitignore` entry added for `~`
- [x] CROSS_REPO_ISSUE documented in Phase: Complete comment
- [x] Answers open question `kb-59dd35`

---

## Unexplored Questions

- **Exact session that created the `~/` directory** — Session logs from March 3-4 were destroyed in the deletion; may exist in Time Machine backup but low priority to recover
- **Whether other tools in the ecosystem have similar tilde bugs** — Any Go tool accepting `~` paths via flags (not shell) could have this issue; orch-go's own Go code uses `os.UserHomeDir()` correctly

---

## Friction

- No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-post-mortem-skillc-09mar-e14c/`
**Investigation:** `.kb/investigations/2026-03-09-inv-post-mortem-skillc-bug-agent.md`
**Beads:** `bd show orch-go-7r4l3`
