<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** CLI binaries get shadowed because Makefiles install to different locations (~/bin vs ~/go/bin) and ~/go/bin has PATH priority.

**Evidence:** 3 of 6 CLI tools use `go install` → ~/go/bin; ~/.zshrc prepends ~/go/bin to PATH before ~/bin.

**Knowledge:** Standardize on ~/bin with explicit install target (orch-go pattern): build → copy → codesign. Avoid `go install`.

**Next:** Update beads, skillc, agentlog Makefiles. Clean stale binaries from ~/go/bin. Consider moving ~/go/bin lower in PATH.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Standardize Binary Installation Location Across CLI Tools

**Question:** Why do CLI binaries (orch, kb, bd) get shadowed by stale versions? How should we standardize installation?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent (orch-go-23fh)
**Phase:** Complete
**Next Step:** Orchestrator to create follow-up issues for beads, skillc, agentlog Makefile updates
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Dual installation locations cause shadowing

**Evidence:** 
- `~/bin/` contains: orch (Dec 26), kb (Dec 26), bd (Dec 25), kn (Dec 6)
- `~/go/bin/` contains: orch (Dec 23), kb (Dec 23), bd (Dec 24), agentlog, skillc

These are different timestamps - ~/go/bin has stale versions.

**Source:** 
- `ls -la ~/bin/` and `ls -la ~/go/bin/`
- Beads issue description: "stale binaries mask new ones"

**Significance:** When `go install` puts binaries in ~/go/bin and manual installs go to ~/bin, PATH order determines which runs. Stale ~/go/bin versions can shadow current ~/bin versions.

---

### Finding 2: PATH order causes ~/go/bin to shadow ~/bin

**Evidence:**
- `~/.zshrc` line 783-784: `export PATH="/Users/dylanconlin/go/bin:$PATH"`
- This PREPENDS ~/go/bin to PATH, making it higher priority than ~/bin
- `~/bin` is added at lines 414, 512, 544, 792 but ~/go/bin:$PATH prepends

**Source:** `~/.zshrc` (lines 783-784, 414, 512, 544, 792)

**Significance:** Root cause of shadowing. Even when ~/bin has the correct binary, ~/go/bin is checked first.

---

### Finding 3: Makefiles use inconsistent install targets

**Evidence:**
| Project | Makefile Install Target | Binary Location |
|---------|------------------------|-----------------|
| orch-go | `~/bin` (explicit INSTALL_DIR) | ~/bin/orch |
| beads | `go install` (GOPATH/bin) | ~/go/bin/bd |
| skillc | `go install` (GOPATH/bin) | ~/go/bin/skillc |
| agentlog | `go install` (GOPATH/bin) | ~/go/bin/agentlog |
| kb-cli | No Makefile (manual build) | ~/bin/kb (copied manually) |
| kn | No Makefile (manual build) | ~/bin/kn (copied manually) |

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/Makefile:35-41`
- `/Users/dylanconlin/Documents/personal/beads/Makefile:37-41`
- `/Users/dylanconlin/Documents/personal/skillc/Makefile:12-13`
- `/Users/dylanconlin/Documents/personal/agentlog/Makefile:4-5`

**Significance:** 3 of 6 CLI tools use `go install` which puts binaries in ~/go/bin. This is the source of the dual-location problem.

---

## Synthesis

**Key Insights:**

1. **Standard should be ~/bin** - orch-go already uses this correctly. All CLI tools should follow the same pattern: build locally, copy to ~/bin, codesign.

2. **`go install` is the anti-pattern** - It installs to ~/go/bin which has PATH priority, shadowing ~/bin. Beads, skillc, and agentlog use this.

3. **Two-pronged fix needed** - Update Makefiles AND remove ~/go/bin from PATH (or reorder).

**Answer to Investigation Question:**

CLI binaries get shadowed because:
1. Some projects use `go install` (→ ~/go/bin) while others use manual install (→ ~/bin)
2. ~/.zshrc prepends ~/go/bin to PATH, giving stale versions priority

**Solution:** Standardize all Makefiles to install to `~/bin` using the orch-go pattern:
```makefile
INSTALL_DIR=$(HOME)/bin
install: build
    cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
    codesign --force --sign - $(INSTALL_DIR)/$(BINARY_NAME)
```

This investigation is in orch-go repo, so I will:
1. Verify orch-go's Makefile is correct (it is)
2. Create standardized Makefile templates for other projects
3. Document the standard in this investigation

---

## Structured Uncertainty

**What's tested:**

- ✅ ~/bin contains current versions: `ls -la ~/bin/{orch,kb,bd,kn}` shows recent timestamps
- ✅ ~/go/bin contains stale versions: `ls -la ~/go/bin/{orch,kb,bd}` shows older timestamps
- ✅ PATH order verified: grep of ~/.zshrc shows ~/go/bin prepended before ~/bin

**What's untested:**

- ⚠️ Removing ~/go/bin from PATH doesn't break other Go tools (gopls, dockfmt may need it)
- ⚠️ Other users' PATH configurations may differ

**What would change this:**

- Finding would be wrong if ~/go/bin is NOT actually in PATH at runtime (check `echo $PATH`)
- Finding would be wrong if another tool legitimately needs ~/go/bin in PATH

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Standardize on ~/bin with explicit install target** - All CLI Makefiles should use the orch-go pattern: build to local dir, copy to ~/bin, codesign.

**Why this approach:**
- ~/bin is already in PATH at high priority (added multiple times in .zshrc)
- orch-go already uses this pattern successfully
- Explicit copy prevents go install's default behavior
- Codesign prevents macOS Gatekeeper issues

**Trade-offs accepted:**
- Requires updating 3 Makefiles (beads, skillc, agentlog)
- Manual copy vs `go install` convenience
- Must remember to run `make install` not `go install`

**Implementation sequence:**
1. Document standard Makefile pattern (this investigation)
2. Update beads/Makefile - highest priority (bd is heavily used)
3. Update skillc/Makefile and agentlog/Makefile
4. Add Makefiles to kb-cli and kn (optional - currently manual)
5. Clean stale binaries from ~/go/bin (or move ~/go/bin lower in PATH)

### Alternative Approaches Considered

**Option B: Reorder PATH to put ~/bin before ~/go/bin**
- **Pros:** No Makefile changes needed
- **Cons:** Fragile - .zshrc is complex, may be overridden; doesn't prevent future issues
- **When to use instead:** Quick temporary fix

**Option C: Symlink ~/go/bin binaries to ~/bin**
- **Pros:** Both locations work
- **Cons:** Adds complexity; symlinks can cause codesign issues on macOS
- **When to use instead:** Never recommended

**Rationale for recommendation:** Fixing at source (Makefiles) prevents future issues. PATH reordering is fragile.

---

### Implementation Details

**What to implement first:**
- This investigation documents the standard (done)
- beads/Makefile is highest priority (bd is core orchestration tool)

**Things to watch out for:**
- ⚠️ macOS requires codesign for binaries (add to all install targets)
- ⚠️ `go install` will still work but install to wrong location - document this
- ⚠️ Existing stale binaries in ~/go/bin should be removed

**Areas needing further investigation:**
- Should ~/go/bin be removed from PATH entirely? (gopls, other go tools may need it)
- Should we add a staleness check to `make install`?

**Success criteria:**
- ✅ `which orch` returns `~/bin/orch` (not ~/go/bin/orch)
- ✅ `which bd` returns `~/bin/bd`
- ✅ All Makefiles use `INSTALL_DIR=$(HOME)/bin` pattern
- ✅ No CLI binaries in ~/go/bin (except gopls, dockfmt)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/Makefile` - Correct pattern to follow
- `/Users/dylanconlin/Documents/personal/beads/Makefile` - Uses `go install` (needs update)
- `/Users/dylanconlin/Documents/personal/skillc/Makefile` - Uses `go install` (needs update)
- `/Users/dylanconlin/Documents/personal/agentlog/Makefile` - Uses `go install` (needs update)
- `~/.zshrc` - PATH configuration showing ~/go/bin prepended

**Commands Run:**
```bash
# Check binary locations
ls -la ~/bin/{orch,kb,bd,kn}
ls -la ~/go/bin/

# Check PATH configuration
grep -n "PATH" ~/.zshrc | grep -E "bin"
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Decision:** None yet - this investigation produces the recommendation
- **Investigation:** N/A
- **Workspace:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-standardize-binary-installation-26dec/`

---

## Investigation History

**2025-12-26 22:35:** Investigation started
- Initial question: Why do CLI binaries get shadowed by stale versions?
- Context: Hit with orch and kb - stale binaries in ~/go/bin masking current ~/bin versions

**2025-12-26 22:40:** Root cause identified
- Dual installation locations: ~/bin (manual) vs ~/go/bin (go install)
- PATH order: ~/go/bin prepended before ~/bin in .zshrc
- 3 of 6 CLI tools use `go install` (beads, skillc, agentlog)

**2025-12-26 22:45:** Investigation synthesized
- Status: Complete
- Key outcome: Standardize all Makefiles to use `INSTALL_DIR=$(HOME)/bin` pattern

---

## Follow-Up Issues (for orchestrator to create)

These issues should be created in their respective repos:

### 1. beads: Update Makefile to install to ~/bin
**Priority:** P1 (bd is core orchestration tool)
**Description:** Replace `go install` with explicit install to `~/bin`:
```makefile
INSTALL_DIR=$(HOME)/bin
BUILD_DIR=build

install: build
    @mkdir -p $(INSTALL_DIR)
    cp $(BUILD_DIR)/bd $(INSTALL_DIR)/bd
    @codesign --force --sign - $(INSTALL_DIR)/bd
```
**Rationale:** Prevents PATH shadowing when ~/go/bin has stale version.

### 2. skillc: Update Makefile to install to ~/bin
**Priority:** P2
**Description:** Same pattern as beads.

### 3. agentlog: Update Makefile to install to ~/bin
**Priority:** P2
**Description:** Same pattern as beads.

### 4. (Optional) Clean stale binaries from ~/go/bin
**Priority:** P3
**Description:** Remove orch, kb, bd from ~/go/bin to prevent future confusion:
```bash
rm ~/go/bin/{orch,kb,bd}
```

### 5. (Optional) Reorder PATH in ~/.zshrc
**Priority:** P3
**Description:** Move ~/go/bin AFTER ~/bin in PATH, or remove entirely if no other Go tools need it.
