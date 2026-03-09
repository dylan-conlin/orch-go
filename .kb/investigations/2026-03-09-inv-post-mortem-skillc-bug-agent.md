## Summary (D.E.K.N.)

**Delta:** A literal `~/.skillc/test-env/` directory was created in orch-go repo root by `skillc test` during test-isolation development (March 3-4); a human then accidentally deleted their home directory by running `rm -rf ~/` instead of `rm -rf ./~/`.

**Evidence:** Session `e8ae39d9` transcript shows the `~/` directory contained Claude Code config artifacts (`plugins/blocklist.json`, `.claude.json`, `backups/`, `debug/`). Code audit of `skillc test_cmd.go` reveals `expandHome()` is called on `scenariosDir`, `variantPath`, `transcriptsDir` but NOT on `configDir` (line 75). `SetupAuth()` and `SetupHooks()` use raw `configDir` which can contain unexpanded `~`.

**Knowledge:** Go's `os.MkdirAll("~/.skillc/test-env", 0755)` creates a literal `~` directory. `expandHome()` must be called on ALL user-provided paths containing `~` BEFORE any file operations. The `testEnv()` function only expands `~/` for the `CLAUDE_CONFIG_DIR` env var, not for `SetupAuth`/`SetupHooks` which operate on the filesystem.

**Next:** Create CROSS_REPO_ISSUE for skillc to add `expandHome(configDir)` and path validation. Add `.gitignore` entry and pre-commit guard in orch-go. Route through architect for implementation.

**Authority:** architectural - Cross-repo fix (skillc) + orch-go protection layers require orchestrator coordination

---

# Investigation: Post-Mortem — skillc ./~/ Bug → Agent Deleted Home Directory

**Question:** How did a literal `~/` directory get created in the orch-go repo root, and what prevention gates should exist to prevent this class of incident?

**Started:** 2026-03-09
**Updated:** 2026-03-09
**Owner:** orch-go-7r4l3
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-02-25-inv-architect-skillc-deploy-silent-failures.md | extends | yes | - |
| .kb/investigations/2026-02-18-inv-audit-full-skillc-pipeline.md | extends | pending | - |

---

## Findings

### Finding 1: The `~/` directory contained Claude Code config artifacts from `skillc test` isolation

**Evidence:** Session `e8ae39d9` (March 8, 2026) agent listed the contents:
```
/Users/dylanconlin/Documents/personal/orch-go/~/.skillc/test-env/plugins/blocklist.json
/Users/dylanconlin/Documents/personal/orch-go/~/.skillc/test-env/.claude.json
/Users/dylanconlin/Documents/personal/orch-go/~/.skillc/test-env/backups/.claude.json.backup.1772653415028
/Users/dylanconlin/Documents/personal/orch-go/~/.skillc/test-env/debug/2f26e22a-4abf-4d1e-899c-3064f268459d.txt
```
Directory created: `Wed Mar 4 11:43:35 2026`. Contents are standard Claude Code config directory structure: `.claude.json`, `plugins/`, `backups/`, `debug/`.

**Source:** Session transcript extraction: `python3 -c "..." e8ae39d9-*.jsonl`

**Significance:** Confirms the `~/` directory was created by Claude Code itself initializing a config directory at a literal `~` path. The `.skillc/test-env/` subpath matches the `defaultIsolationDir()` function used by `skillc test`.

---

### Finding 2: `expandHome()` is NOT called on `--config-dir` in `test_cmd.go`

**Evidence:** In `skillc/cmd/skillc/test_cmd.go`, `expandHome()` is called on:
- `scenariosDir` (line 119)
- `variantPath` (line 121)
- `transcriptsDir` (line 183)

But NOT on `configDir` (line 75 — raw `os.Args[i+1]` assignment).

The `testEnv()` function in `runner.go:443-448` expands `~/` for `CLAUDE_CONFIG_DIR` env var, but `SetupAuth(configDir)` and `SetupHooks(configDir, hooksDir)` receive the RAW `configDir` and perform filesystem operations (symlinks, `os.MkdirAll`) at that literal path.

**Source:**
- `skillc/cmd/skillc/test_cmd.go:75` — raw configDir assignment
- `skillc/cmd/skillc/test_cmd.go:119,121,183` — expandHome on other paths
- `skillc/pkg/scenario/runner.go:112,122` — SetupAuth uses raw configDir
- `skillc/pkg/scenario/runner.go:232,245` — SetupHooks/copyDir calls os.MkdirAll

**Significance:** This is the code-level vulnerability. If `skillc test` is invoked via `exec.Command` (no shell expansion) or with quoted tilde, `configDir` retains the literal `~`.

---

### Finding 3: The incident timeline spans March 3-8, 2026

**Evidence:**
- **March 3:** Commits `88afe64` (default isolation to `~/.skillc/test-env/`) and `a541f1f` (behavioral gate in deploy) introduced test isolation
- **March 3-4:** Multiple rapid-fire fixes for auth in isolation dir (commits `bfdd176`, `8c72d75`, `2ee3f91`, `f2b87ec`)
- **March 4 11:43:** The literal `~/.skillc/test-env/` directory created in orch-go root
- **March 4:** Commit `2ee3f91` removes default isolation because "it breaks keychain auth"
- **March 8 ~16:00:** Home directory deleted via `rm -rf ~/`
- **March 8 16:15:** Session `e8ae39d9` — recovery session, agent safely removes `~/` dir
- **March 8 17:21:** Commit `639b6bef7` — reconstruct lost files post-deletion

**Source:** `git log` for skillc repo, `ls -la` of session files, commit `639b6bef7` message

**Significance:** The `~/` directory sat unnoticed for 4 days (March 4-8) before the destructive `rm -rf ~/` was run. The rapid development of test isolation (6 commits in 2 days) created conditions for the bug. Session logs from March 3-4 were lost in the deletion, preventing exact root cause identification.

---

### Finding 4: `defaultIsolationDir()` uses `os.UserHomeDir()` correctly

**Evidence:** Both `defaultIsolationDir()` and `defaultBehavioralIsolationDir()` call `os.UserHomeDir()`, which on macOS uses `getpwuid_r()` (syscall, not `$HOME`). Test confirmed `os.UserHomeDir()` returns `/Users/dylanconlin`.

When the default path is used (no `--config-dir` flag), the configDir is an absolute path and the bug doesn't trigger. The bug only manifests when `--config-dir` receives an unexpanded tilde.

**Source:**
- `skillc/cmd/skillc/test_cmd.go:502-514` — defaultIsolationDir
- `skillc/cmd/skillc/deploy.go:597-609` — defaultBehavioralIsolationDir
- Go test `/tmp/test_tilde.go` — verified expandHome and testEnv behavior

**Significance:** The default code path is safe. The vulnerability is in the `--config-dir` flag handling, which is the most likely path used by agents invoking `skillc test` programmatically.

---

### Finding 5: Two-stage failure — creation + deletion

**Evidence:** The incident required TWO failures:
1. **Creation:** `skillc test` (or a similar process) created `./~/` at the orch-go root
2. **Deletion:** Human ran `rm -rf ~/` instead of `rm -rf ./~/`

The agent in session `e8ae39d9` correctly identified the danger and suggested: `rm -rf '/Users/dylanconlin/Documents/personal/orch-go/~'` (using the full quoted absolute path).

**Source:** Session `e8ae39d9` transcript

**Significance:** Both stages need prevention gates. The creation bug is in skillc, but the deletion mistake is a universal shell safety issue that also needs defense.

---

## Synthesis

**Key Insights:**

1. **Go treats `~` as a literal character** — Unlike shell, Go's `os.MkdirAll`, `os.WriteFile`, etc. do not expand `~`. Every path that might contain `~` must pass through `expandHome()` before ANY filesystem operation. The `testEnv()` function only expanded `~/` for env var injection, not for the filesystem operations that ran before it.

2. **The asymmetry between `testEnv` and `SetupAuth` is the root cause** — `testEnv()` correctly expands `~/` for `CLAUDE_CONFIG_DIR`, but `SetupAuth()` and `SetupHooks()` receive the raw configDir. This means Claude CLI gets the correct path via env var, but filesystem setup operations use the literal path, creating the `~` directory.

3. **`defaultIsolationDir()` is safe; `--config-dir` is not** — The default path through `os.UserHomeDir()` always produces an absolute path. But when `--config-dir` is explicitly set (likely by an agent invoking `skillc test` programmatically via `exec.Command`), tilde expansion depends on shell, which may not be involved.

**Answer to Investigation Question:**

The `~/` directory was most likely created when `skillc test` was invoked (either directly or via `skillc deploy --behavioral`) with a `--config-dir` argument containing `~/.skillc/test-env/` where the tilde was not shell-expanded. This occurred during rapid development of test isolation (March 3-4). The `expandHome()` function is not called on the `configDir` parameter, causing `SetupAuth()`, `SetupHooks()`, and Claude CLI to create files at a literal `~` path relative to CWD. The exact invocation cannot be determined because session logs from March 3-4 were lost in the deletion. The home directory deletion occurred 4 days later when a human ran `rm -rf ~/` to clean up the stray directory.

---

## Structured Uncertainty

**What's tested:**

- ✅ `expandHome()` correctly handles `~/.skillc/test-env/` → `/Users/dylanconlin/.skillc/test-env` (verified: Go test at `/tmp/test_tilde.go`)
- ✅ `testEnv()` correctly expands `~/` for CLAUDE_CONFIG_DIR env var (verified: Go test)
- ✅ `os.UserHomeDir()` returns correct path on macOS (verified: Go test — returns `/Users/dylanconlin`)
- ✅ `expandHome` is NOT called on configDir in test_cmd.go (verified: code audit of lines 75 vs 119,121,183)
- ✅ The `~/` directory contained Claude Code config artifacts from test isolation (verified: session e8ae39d9 transcript)

**What's untested:**

- ⚠️ Exact invocation that created the `~/` directory (session logs lost in deletion)
- ⚠️ Whether `SetupAuth` or `SetupHooks` actually created the `~` directory vs Claude CLI itself (both are plausible — SetupHooks calls os.MkdirAll, Claude CLI also creates config dirs)
- ⚠️ Whether the `--config-dir` flag was explicitly passed or if there's another code path

**What would change this:**

- Finding session logs from March 3-4 (unlikely — destroyed in deletion, may exist in Time Machine)
- Discovering another process that creates `~/.skillc/test-env/` without proper expansion
- Evidence that `defaultIsolationDir()` failed (would require `os.UserHomeDir()` failure on macOS)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Fix `expandHome` on configDir in skillc | architectural | Cross-repo change (skillc), affects all skillc test users |
| Add path validation guard in skillc | architectural | New pattern for all path-accepting functions |
| Add `.gitignore` entry in orch-go | implementation | Project-local, reversible, no cross-boundary impact |
| Add pre-commit guard for `~` dirs | implementation | Project-local safety net |

### Recommended Approach ⭐

**Multi-layer prevention: fix source + guard repo + constrain shell** — Fix the skillc bug at the source AND add defensive layers in orch-go.

**Why this approach:**
- Fixes the root cause in skillc (missing `expandHome`)
- Adds defense-in-depth in orch-go (gitignore, pre-commit)
- Existing constraint already covers shell safety (`rm -rf` rule)

**Trade-offs accepted:**
- Cannot verify fix against the exact scenario (logs lost)
- Cross-repo issue requires separate skillc work

**Implementation sequence:**
1. **orch-go: `.gitignore` entry** — Add `~` to `.gitignore` so literal `~` directories are never tracked (immediate, low risk)
2. **skillc: `expandHome(configDir)`** — Add `expandHome` call on `configDir` after flag parsing (CROSS_REPO_ISSUE)
3. **skillc: path validation** — Add `validateNoLiteralTilde(path)` check before `os.MkdirAll` calls in `SetupAuth`, `SetupHooks`, `copyDir` (CROSS_REPO_ISSUE)
4. **skillc: test coverage** — Add test case for `skillc test --config-dir "~/.skillc/test-env/"` with unexpanded tilde

### Alternative Approaches Considered

**Option B: Only fix skillc, no orch-go guards**
- **Pros:** Fixes root cause, no orch-go changes needed
- **Cons:** No defense-in-depth; other tools could create `~` directories
- **When to use instead:** If orch-go changes are too disruptive

**Option C: Shell-level `rm` alias/function**
- **Pros:** Prevents the deletion regardless of source
- **Cons:** Fragile (subshells, scripts, non-interactive shells bypass aliases)
- **When to use instead:** As supplementary protection alongside other fixes

---

### Implementation Details

**What to implement first:**
- `.gitignore` entry for `~` — immediate, zero risk
- CROSS_REPO_ISSUE for skillc fix — enables tracking

**Things to watch out for:**
- ⚠️ The `expandHome` function handles `~` at position 0; paths like `./~/` or `path/~/` need different handling
- ⚠️ `testEnv` already expands `~/` for env var but NOT for filesystem; the fix must be BEFORE any filesystem use

**Success criteria:**
- ✅ `skillc test --config-dir "~/.skillc/test-env/"` (quoted) creates files at expanded path, not literal `~`
- ✅ `git status` never shows `~` directory as untracked
- ✅ Pre-commit hook blocks commits containing `~` paths

---

## References

**Files Examined:**
- `skillc/cmd/skillc/deploy.go` — Full deploy flow including behavioral gate
- `skillc/cmd/skillc/test_cmd.go:75,119,121,183` — expandHome calls (and missing call on configDir)
- `skillc/cmd/skillc/util.go:12-21` — expandHome implementation
- `skillc/pkg/scenario/runner.go:96-150,219-277,423-452,454-498` — SetupAuth, SetupHooks, testEnv, Run functions
- `skillc/pkg/compiler/compiler.go:44-90,741-791` — CompileForDeploy and header generation
- `orch-go/.git/hooks/pre-commit` — Current pre-commit hook
- Session `e8ae39d9` transcript — Recovery session showing `~/` directory contents

**Commands Run:**
```bash
# Extract Bash commands from recovery session
python3 -c "..." e8ae39d9-*.jsonl

# Find sessions with rm -rf ~/
grep -c 'rm -rf ~/' *.jsonl

# Test tilde expansion behavior
go run /tmp/test_tilde.go

# Check skillc commits around incident
git log --oneline --since="2026-03-03" --until="2026-03-05"
```

**Related Artifacts:**
- **Constraint:** `kb-59dd35` — Open question about `./~/` directory origin (now answered)
- **Constraint:** Existing CLAUDE.md/kb constraint — "Never rm -rf ~/ or rm -rf without ./ prefix"
