<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Agent spawn inherits minimal PATH from OpenCode parent process, causing bd/go/orch commands to fail.

**Evidence:** PATH in spawned agent is `/Users/dylanconlin/.cargo/bin:/Users/dylanconlin/claude-npm-global/bin:/Users/dylanconlin/.bun/bin:/usr/local/bin:/usr/bin:/bin` - missing ~/bin, /opt/homebrew/bin, ~/go/bin.

**Knowledge:** OpenCode spawns agents with minimal environment; orch spawn uses os.Environ() which inherits this limited PATH. Commands like bd, go, orch are in ~/bin but unreachable.

**Next:** Create follow-up issue to add PATH augmentation in orch-go spawn code.

---

# Investigation: Cleanup Stale Binaries + PATH Fix

**Question:** What stale binaries exist at project root and why do agents fail to find bd/orch commands?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Worker agent (orch-go-niyj)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Stale binaries at project root

**Evidence:** Three executable files found at project root:
- `orch` (20MB) - development build
- `orch-go` (20MB) - development build
- `gendoc` (3.4MB) - documentation generator

These are ignored by .gitignore but consume disk space.

**Source:** `ls -la | grep -E '^-.*x'`

**Significance:** Already handled by `make clean-all` target. Cleaned successfully.

---

### Finding 2: Agent PATH is minimal

**Evidence:** Agent's PATH contains only:
```
/Users/dylanconlin/.cargo/bin
/Users/dylanconlin/claude-npm-global/bin
/Users/dylanconlin/.bun/bin
/usr/local/bin
/usr/bin
/bin
```

Missing:
- `~/bin` (where orch, bd installed)
- `/opt/homebrew/bin` (where go installed)
- `~/go/bin` (Go binaries)
- `~/.local/bin` (other local binaries)

**Source:** `echo $PATH | tr ':' '\n'`

**Significance:** Commands like `bd comment`, `go build`, `orch spawn` fail with "command not found" in agent sessions.

---

### Finding 3: PATH is configured in .zshrc but not inherited

**Evidence:** `~/.zshrc` has:
```bash
export PATH=$HOME/bin:$PATH
```

But OpenCode spawns agents with a minimal environment that doesn't source .zshrc.

**Source:** `cat ~/.zshrc | tail -30`

**Significance:** Shell configuration exists but isn't applied to agent processes.

---

### Finding 4: orch spawn uses os.Environ()

**Evidence:** In `cmd/orch/main.go:1545`:
```go
cmd.Env = append(os.Environ(), "ORCH_WORKER=1")
```

This inherits parent environment, which is already minimal.

**Source:** cmd/orch/main.go:1545

**Significance:** orch-go correctly passes environment but the parent (OpenCode) doesn't provide a full PATH.

---

## Synthesis

**Key Insights:**

1. **Cascading minimal environment** - OpenCode spawns orchestrator with minimal PATH, orchestrator spawns workers, workers inherit minimal PATH.

2. **Two-layer fix needed** - Either OpenCode needs to provide full PATH, or orch-go needs to augment PATH before spawning.

3. **Workaround available** - Commands can be called with full path (e.g., `~/bin/bd`) but this is tedious.

**Answer to Investigation Question:**

Stale binaries (orch, orch-go, gendoc) were successfully cleaned via `rm -f`. The PATH issue is a fundamental environment inheritance problem where OpenCode spawns agents with minimal PATH that doesn't include ~/bin or /opt/homebrew/bin. The proper fix requires either OpenCode changes or PATH augmentation in orch-go spawn code.

---

## Structured Uncertainty

**What's tested:**

- ✅ Stale binaries exist at project root (verified: ls command)
- ✅ PATH in agent is minimal (verified: echo $PATH)
- ✅ ~/bin/bd works with full path (verified: ~/bin/bd comment worked)
- ✅ orch spawn uses os.Environ() (verified: read main.go:1545)

**What's untested:**

- ⚠️ Adding PATH augmentation in orch-go spawn would fix issue (not implemented)
- ⚠️ OpenCode has configuration for environment (not investigated deeply)

**What would change this:**

- Finding would be wrong if OpenCode has a hidden PATH config option
- Finding would be wrong if agents run in a different shell mode that sources zshrc

---

## Implementation Recommendations

### Recommended Approach: Add PATH augmentation to orch-go spawn

Modify spawn code to augment PATH with essential directories before creating worker processes.

**Why this approach:**
- orch-go already controls the spawn environment (main.go:1545)
- Can read user's full PATH and pass it to workers
- Self-contained fix without requiring OpenCode changes

**Trade-offs accepted:**
- Duplicates PATH knowledge (hardcoded paths or read from user shell)
- May diverge from actual user PATH if not carefully synced

**Implementation sequence:**
1. In spawn code, read current PATH and append essential dirs (~/bin, /opt/homebrew/bin)
2. Or: Run `zsh -i -c 'echo $PATH'` to get user's actual PATH
3. Set augmented PATH in cmd.Env

### Alternative Approaches Considered

**Option B: Fix in OpenCode**
- **Pros:** Fixes all agents, not just orch-spawned ones
- **Cons:** Requires changes to external project
- **When to use instead:** If orch-go fix proves insufficient

---

## References

**Files Examined:**
- `cmd/orch/main.go:1545` - spawn environment setup
- `pkg/opencode/client.go` - OpenCode client implementation
- `~/.zshrc` - user shell configuration
- `.gitignore` - build artifact ignoring

**Commands Run:**
```bash
# Check stale binaries
ls -la | grep -E '^-.*x'

# Clean binaries
rm -f orch orch-go gendoc

# Check PATH
echo $PATH | tr ':' '\n'

# Check shell config
cat ~/.zshrc | tail -30
```

---

## Investigation History

**2025-12-28 21:35:** Investigation started
- Initial question: What stale binaries exist and why do agents fail to find commands?
- Context: Follow-up from orch-go-3m23, environment prep task

**2025-12-28 21:45:** Stale binaries cleaned
- Successfully removed orch, orch-go, gendoc from project root

**2025-12-28 21:55:** PATH issue identified
- Root cause: OpenCode spawns with minimal environment
- Workaround: Use full paths (~/bin/bd)

**2025-12-28 22:00:** Investigation completed
- Status: Complete
- Key outcome: Binaries cleaned, PATH fix needs follow-up issue
