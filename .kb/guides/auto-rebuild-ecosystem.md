# Auto-Rebuild Patterns Across Orch Ecosystem

**Purpose:** How Go binary rebuilds work across ecosystem repos, where the gaps are, and how to diagnose stale binaries.

**Last verified:** 2026-02-07

---

## Quick Reference

```bash
# Check if binary is stale
orch version          # shows git hash — compare to `git rev-parse --short HEAD`
kb version            # shows git hash with dirty flag
bd version            # shows git hash and branch
skillc version        # shows git hash and build time

# Manual rebuild any Go repo
make install          # standard pattern across repos

# Force rebuild + restart orch serve
make install && launchctl kickstart -k gui/$(id -u)/com.orch.serve
```

---

## Three Rebuild Mechanisms

### 1. Post-Commit Hooks (40% of repos)

Automatic rebuild triggered when Go files change on commit. The gold standard — no discipline required.

| Repo | Hook Pattern | Build Command |
|------|-------------|---------------|
| **orch-go** | Detects `cmd\|pkg/*.go` changes | `make install` |
| **kb-cli** | Detects `*.go` changes | `make build` |
| **agentlog** | Unconditional (always rebuilds) | `go install ./cmd/agentlog` |
| **kn** | Detects `*.go` changes | `go build -o kn ./cmd/kn` |

### 2. `orch complete` Auto-Rebuild (Agent workflow only)

During `orch complete`, `rebuildGoProjectsIfNeeded()` runs BEFORE verification to ensure gates test fresh binaries.

- **Repos covered:** orch-go, kb-cli (cross-project support)
- **Trigger:** Detects Go file changes in recent commits via `git diff --name-only HEAD~5..HEAD`
- **Extras:** Restarts `orch serve` if orch-go was rebuilt
- **Limitation:** Only triggers during agent completion, not manual CLI usage

**Evolution:**
1. **Dec 24, 2025** — Initial implementation, ran AFTER verification (wrong timing)
2. **Jan 17, 2026** — Fixed lock file deadlock (stale PIDs not validated)
3. **Jan 23, 2026** — Moved to run BEFORE verification + cross-project support
4. **Jan 28, 2026** — Ecosystem audit identified coverage gaps

### 3. Manual Discipline (60% of repos)

Developer must remember `make install` after committing changes.

| Repo | Why No Hook | Risk |
|------|------------|------|
| **beads** | Upstream OSS (Dylan doesn't modify) | Low — uses releases |
| **glass** | Missing | **High** — no version command either, zero staleness detection |
| **skillc** | Missing (has own auto-rebuild on version check) | Medium — skills compiled with stale binary |
| **orch-cli** | Python, no build step | N/A |
| **orch-knowledge** | Skills compiled via `skillc deploy` | Medium — depends on skillc freshness |
| **beads-ui-svelte** | Vite dev server auto-reloads | Low — only production build is manual |

---

## Lock File Pattern

Auto-rebuild uses `.autorebuild.lock` to prevent concurrent builds:

```
PID is written to lock file on build start
→ Subsequent rebuild attempts check:
  1. Does lock file exist?
  2. Is the PID still alive? (kill -0)
  3. If PID dead → stale lock → remove and proceed
  4. If PID alive → skip (build in progress)
```

**Gotcha:** Go's `os.FindProcess().Signal(0)` returns `"os: process already finished"` (string error), not `syscall.ESRCH`. Both must be handled.

---

## Staleness Detection

| CLI | Version Command | Detectable? |
|-----|----------------|-------------|
| orch | `orch version` → git hash + build time | ✅ Yes |
| kb | `kb version` → git hash + dirty flag | ✅ Yes |
| bd | `bd version` → git hash + branch | ✅ Yes |
| skillc | `skillc version` → git hash + build time | ✅ Yes |
| glass | ❌ No version command | ❌ **No** |
| agentlog | ❌ No version command | ❌ **No** |

**To check if a binary is stale:**
```bash
# Compare binary git hash to repo HEAD
BINARY_HASH=$(orch version 2>&1 | grep -o '[a-f0-9]\{7,\}' | head -1)
REPO_HASH=$(git -C ~/Documents/personal/orch-go rev-parse --short HEAD)
[ "$BINARY_HASH" = "$REPO_HASH" ] && echo "FRESH" || echo "STALE"
```

---

## Critical Gaps (Open Work)

1. **glass** — No version command AND no post-commit hook. Zero staleness detection. Highest priority gap.
2. **agentlog** — No version command. Has post-commit hook (rebuilds) but can't verify freshness.
3. **skillc** — No post-commit hook. Has its own auto-rebuild-on-version-check mechanism (race condition observed).

**Adding version commands** requires ldflags pattern:
```go
// In main.go
var (
    version   = "dev"
    commit    = "unknown"
    buildTime = "unknown"
)
```
```makefile
# In Makefile
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"
```

---

## Failure Mode: "Fixed Bug Still Reproduces"

The most common symptom of stale binaries: an agent commits a fix, `orch complete` succeeds, but the bug still reproduces because the running binary wasn't rebuilt.

**Diagnosis:**
1. Check binary version: `orch version`
2. Compare to repo HEAD: `git rev-parse --short HEAD`
3. If mismatched: `make install` in the repo
4. If `orch serve` is running: restart it after rebuild

---

## Synthesized From

- `.kb/investigations/2025-12-24-inv-auto-rebuild-after-go-changes.md` — Original implementation
- `.kb/investigations/2026-01-17-inv-investigate-auto-rebuild-reports-already.md` — Lock file deadlock fix
- `.kb/investigations/2026-01-23-inv-auto-rebuild-go-binaries-during.md` — Timing fix (before verification)
- `.kb/investigations/2026-01-28-inv-audit-rebuild-change-mechanisms-across.md` — Ecosystem audit
