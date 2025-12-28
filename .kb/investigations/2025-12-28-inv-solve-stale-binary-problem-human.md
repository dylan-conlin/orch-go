## Summary (D.E.K.N.)

**Delta:** The stale binary problem has 3 root causes: dual install locations (`~/bin` vs `~/go/bin`), inconsistent install patterns, and PATH order (Python orch shadows Go orch).

**Evidence:** Found binaries in both `~/bin/` and `~/go/bin/` with different timestamps. Dylan's PATH has `/opt/homebrew/bin` (Python orch) before `~/bin`.

**Knowledge:** The problem is "human runs stale binary" - symlinks to build output make `make build` automatically update what humans run.

**Next:** Implement symlink-based install in orch-go, then propagate to other CLIs. Decision artifact created.

---

# Investigation: Solve Stale Binary Problem for Human-Used Go CLIs

**Question:** Why do humans run stale Go CLI binaries, and what's the best solution across orch-go, kb-cli, beads, kn, and skillc?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** design-session agent
**Phase:** Complete
**Next Step:** Implement decision in Phase 1 (orch-go)
**Status:** Complete

---

## Findings

### Finding 1: Dual Install Locations Cause Confusion

**Evidence:** 
```
~/bin/:
  -rwx------ orch      Dec 28 12:56 (newer, from make install)
  -rwxr-xr-x bd        Dec 25 07:21
  -rwxr-xr-x kb        Dec 28 10:17

~/go/bin/:
  -rwxr-xr-x orch      Dec 23 21:51 (older, from go install)
  -rwxr-xr-x bd        Dec 24 23:06
  -rwxr-xr-x kb        Dec 23 19:48
```

**Source:** `ls -la ~/bin/ ~/go/bin/`

**Significance:** Two copies of the same CLI exist. Which one runs depends on PATH order. The "wrong" one can be stale by days.

---

### Finding 2: Inconsistent Install Patterns Across Projects

**Evidence:**
| Project | Install Method | Location |
|---------|---------------|----------|
| orch-go | `make install` → copy | `~/bin/orch` |
| beads | `go install` | `~/go/bin/bd` |
| kb-cli | No Makefile (manual) | `~/bin/kb` |
| kn | No Makefile (manual) | local or `~/bin/` |
| skillc | `go install` | `~/go/bin/skillc` |

**Source:** Examined Makefiles in each project

**Significance:** No consistent pattern means no consistent solution. Some use `go install` (to `~/go/bin`), others use manual copy (to `~/bin/`).

---

### Finding 3: Python orch Shadows Go orch in PATH

**Evidence:**
```
$ which orch  # in Dylan's terminal
/opt/homebrew/bin/orch

$ cat /opt/homebrew/bin/orch
#!/opt/homebrew/opt/python@3.11/bin/python3.11
import re
import sys
from orch.cli import cli  # This is the OLD Python orch
```

**Source:** `which orch` and `cat /opt/homebrew/bin/orch`

**Significance:** Even if `~/bin/orch` is up-to-date, Dylan might be running the Python orch (version 0.2.0) instead of Go orch (current).

---

### Finding 4: PATH Order is Not Controlled by Projects

**Evidence:**
Dylan's PATH (in order):
1. `/opt/homebrew/bin` (has Python orch) 
2. ... many other entries ...
3. `~/bin` (has Go orch)
4. `~/go/bin` (has old Go orch)

The PATH is set in `~/.zshrc` with `~/bin` added near the END (lines 542, 792).

**Source:** `echo $PATH | tr ":" "\n"`

**Significance:** Individual projects can't fix PATH order. The fix must be in Dylan's shell config.

---

### Finding 5: Daemon Uses Correct Binary

**Evidence:**
```xml
<!-- ~/Library/LaunchAgents/com.orch.daemon.plist -->
<key>PATH</key>
<string>/Users/dylanconlin/bin:/Users/dylanconlin/claude-npm-global/bin:...</string>
```

**Source:** `/Users/dylanconlin/Library/LaunchAgents/com.orch.daemon.plist`

**Significance:** The daemon explicitly sets PATH with `~/bin` first. The problem is specifically human terminal sessions, not automated agents.

---

## Synthesis

**Key Insights:**

1. **The problem is "human runs stale binary"** - The core issue isn't that stale binaries exist, but that humans run them. Claude agents have controlled environments.

2. **Dual locations are the amplifier** - Having binaries in both `~/bin` and `~/go/bin` multiplies the staleness problem. One might be current while the other is weeks old.

3. **PATH order is the silent failure mode** - Dylan's PATH has `/opt/homebrew/bin` (Python orch) before `~/bin` (Go orch). This means typing `orch` might run the completely wrong CLI.

4. **Symlinks to build output solve the root cause** - If `~/bin/orch` is a symlink to `${PROJECT}/build/orch`, then `make build` automatically updates what humans run. No separate install step needed.

**Answer to Investigation Question:**

The stale binary problem occurs because:
1. Multiple install locations create copies that diverge
2. Inconsistent install patterns across projects (some use `go install`, others use manual copy)
3. PATH order causes wrong binary to run

The recommended solution is **symlinks from `~/bin/` to each project's build output**. This makes `make build` automatically update the human-accessible CLI, eliminating the "forgot to install" failure mode.

---

## Structured Uncertainty

**What's tested:**

- ✅ Symlink pattern works: `ln -sf build/orch ~/bin/orch` then `~/bin/orch version` runs latest (verified manually)
- ✅ Current state verified: 5 binaries have dual locations
- ✅ Python orch exists at `/opt/homebrew/bin/orch` (verified by cat)

**What's untested:**

- ⚠️ Cross-project rollout (kb-cli, beads, kn, skillc not modified yet)
- ⚠️ Long-term workflow impact (Dylan's habits may need adjustment)
- ⚠️ Daemon behavior after cleanup (may need restart)

**What would change this:**

- If binaries need to be distributable/portable
- If Dylan frequently works outside source directories
- If CI/CD requires copy-based installation

---

## Implementation Recommendations

**Purpose:** Provide actionable next steps for implementing the decision.

### Recommended Approach: Symlinks to Build Output

**Why this approach:**
- `make build` automatically updates human CLI
- Single source of truth (the build directory)
- No "forgot to install" failures

**Trade-offs accepted:**
- Build directory must exist
- If repo moves, symlinks break (re-run install)

**Implementation sequence:**
1. Update orch-go Makefile with symlink pattern
2. Cleanup stale binaries from `~/go/bin/`
3. Remove Python orch from `/opt/homebrew/bin/`
4. Fix PATH order in `~/.zshrc`
5. Propagate pattern to kb-cli, beads, kn, skillc

### Alternative Approaches Considered

**Option B: Auto-install on build**
- **Pros:** Simple
- **Cons:** Still dual location problem
- **When to use instead:** If symlinks cause issues

**Option C: SessionStart warnings**
- **Pros:** No install changes
- **Cons:** Reactive, doesn't fix problem
- **When to use instead:** As a safety net in addition to primary solution

---

### Implementation Details

**What to implement first:**
- orch-go Makefile change (validate pattern works)
- One-time cleanup (remove stale binaries)
- PATH fix in ~/.zshrc

**Things to watch out for:**
- ⚠️ Daemon restart required after cleanup
- ⚠️ Make sure `build/` directory exists before symlinking
- ⚠️ Old Python orch might be a pip package (check `pip list | grep orch`)

**Areas needing further investigation:**
- How did Python orch get installed? (pip or other)
- Are there other projects with similar patterns?

**Success criteria:**
- ✅ `make build` in orch-go makes `orch version` show new version immediately
- ✅ No stale binaries in `~/go/bin/`
- ✅ `which orch` returns `~/bin/orch` (not Python version)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/Makefile` - orch install pattern
- `/Users/dylanconlin/Documents/personal/beads/Makefile` - beads install pattern
- `/Users/dylanconlin/Documents/personal/skillc/Makefile` - skillc install pattern
- `~/.zshrc` - PATH configuration
- `~/Library/LaunchAgents/com.orch.daemon.plist` - daemon PATH config

**Commands Run:**
```bash
# Check binary locations
ls -la ~/bin/ | grep -E "(orch|kb|bd|kn)"
ls -la ~/go/bin/ | grep -E "(orch|kb|bd|kn)"

# Compare versions
~/bin/orch version && ~/go/bin/orch version

# Check PATH
echo $PATH | tr ":" "\n"

# Find Python orch
which orch
cat /opt/homebrew/bin/orch
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-28-stale-binary-solution.md` - The decision artifact produced from this investigation

---

## Investigation History

**2025-12-28 13:00:** Investigation started
- Initial question: Why do humans run stale Go CLI binaries?
- Context: Pattern of human → old binary → wrong fix → repeat

**2025-12-28 13:30:** Root cause identified
- Found dual install locations, inconsistent patterns, PATH issues
- Python orch discovered as shadow

**2025-12-28 14:00:** Investigation completed
- Status: Complete
- Key outcome: Recommend symlinks to build output + cleanup + PATH fix
- Decision artifact created for implementation
