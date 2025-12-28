## Summary (D.E.K.N.)

**Delta:** Use symlinks from `~/bin/` pointing to each project's build output for all human-used Go CLIs.

**Evidence:** Investigation found 3 root causes: dual install locations (`~/bin` vs `~/go/bin`), inconsistent install patterns across projects, and PATH order issues (Python orch in `/opt/homebrew/bin` shadows Go orch).

**Knowledge:** The problem is "human runs stale binary" not "binary is stale" - symlinks to build output make `make build` automatically update what humans run.

**Next:** Implement in orch-go first, validate, then propagate to kb-cli, beads, kn, skillc.

---

# Decision: Stale Binary Solution for Human-Used Go CLIs

**Date:** 2025-12-28
**Status:** Proposed

---

## Context

Pattern: human runs old binary → reports failure → Claude fixes wrong thing → loop repeats.

Glass doesn't have this problem because only Claude uses it. The problem is specifically about **human-invoked CLIs** where Dylan types `orch`, `kb`, `bd`, etc.

Investigation revealed three root causes:
1. **Dual install locations**: `~/bin/` (orch-go's `make install`) vs `~/go/bin/` (beads' `go install`)
2. **Inconsistent install patterns**: Each project uses different installation methods
3. **PATH order issues**: `/opt/homebrew/bin` (old Python orch) appears before `~/bin` in Dylan's PATH

Current state:
| CLI | Install method | Location | Has stale copy? |
|-----|---------------|----------|-----------------|
| orch | `make install` → copy | `~/bin/orch` | Yes (also in `~/go/bin/`) |
| bd | `go install` | `~/go/bin/bd` | Yes (also in `~/bin/`) |
| kb | manual copy | `~/bin/kb` | Yes (also in `~/go/bin/`) |
| kn | local only | `~/bin/kn` | No |
| skillc | `go install` | `~/go/bin/skillc` | Yes (also in `~/bin/`) |

---

## Options Considered

### Option A: Symlinks from Build Output
- **Description:** `make install` creates symlink `~/bin/binary` → `${PROJECT_DIR}/build/binary`
- **Pros:** 
  - `make build` automatically updates human's CLI (no install step)
  - Single binary, no duplication
  - Source repo is the single source of truth
  - Works offline (no network)
- **Cons:** 
  - Breaks if repo moves/deletes
  - Build directory must exist
  - Requires initial cleanup of duplicates

### Option B: Auto-Install on Build
- **Description:** `make build` automatically runs `make install`
- **Pros:** 
  - Simple change
  - Familiar pattern
- **Cons:** 
  - Still dual location problem (doesn't address `~/go/bin`)
  - Install might fail silently
  - Doesn't solve PATH order issues

### Option C: SessionStart Hook Warning
- **Description:** Check binary staleness at session start, warn if stale
- **Pros:** 
  - No install changes needed
  - Works with existing setup
- **Cons:** 
  - Reactive (already running stale binary)
  - Warning fatigue
  - Doesn't fix the problem, just surfaces it

### Option D: Single Canonical Location with Cleanup
- **Description:** Standardize on `~/go/bin/` (or `~/bin/`), remove duplicates
- **Pros:** 
  - Standard Go approach (if using `~/go/bin/`)
  - Eliminates confusion
- **Cons:** 
  - One-time cleanup required
  - Changes Dylan's habits
  - Still need consistent install pattern

### Option E: Version-Aware Wrapper
- **Description:** Wrapper script checks staleness before running, prompts if stale
- **Pros:** 
  - Catches at invocation time
  - User chooses to rebuild or continue
- **Cons:** 
  - Overhead on every invocation
  - Complex to maintain
  - Another layer of indirection

### Option F: Unified Symlinks (~/bin → ~/go/bin or vice versa)
- **Description:** Pick one location, other has symlinks to it
- **Pros:** 
  - Single binary
  - Both paths work
- **Cons:** 
  - Still need consistent install to canonical location
  - Initial cleanup required

---

## Decision

**Chosen:** Option A (Symlinks from Build Output) + cleanup

**Rationale:** 
1. Symlinks make `make build` automatically update what humans run - no separate install step
2. The source repo becomes the single source of truth for the binary
3. This pattern is already proven: it's essentially how `go run` works (binary in build cache)
4. Eliminates the "forgot to run `make install`" failure mode entirely

**Implementation:**

1. **Standardize on `~/bin/` as canonical location** (already used by orch-go)

2. **Modify Makefile install target:**
```makefile
# New pattern for all human-used Go CLIs
install: build
	@mkdir -p $(HOME)/bin
	@rm -f $(HOME)/bin/$(BINARY_NAME)
	@ln -sf $(CURDIR)/build/$(BINARY_NAME) $(HOME)/bin/$(BINARY_NAME)
	@echo "Linked ~/bin/$(BINARY_NAME) → $(CURDIR)/build/$(BINARY_NAME)"
```

3. **One-time cleanup:**
```bash
# Remove stale copies from ~/go/bin
rm -f ~/go/bin/orch ~/go/bin/kb ~/go/bin/bd ~/go/bin/kn ~/go/bin/skillc

# Remove Python orch from homebrew
rm -f /opt/homebrew/bin/orch
# OR: brew uninstall orch (if it was installed via brew)
```

4. **PATH order fix in ~/.zshrc:**
```bash
# Ensure ~/bin comes FIRST (before /opt/homebrew/bin)
export PATH="$HOME/bin:$PATH"
# This line should be at the TOP of PATH modifications
```

**Trade-offs accepted:**
- Build directory must exist for CLI to work (acceptable: always work from source)
- If repo moves, symlinks break (acceptable: re-run `make install` after move)
- Initial cleanup required (one-time cost)

---

## Structured Uncertainty

**What's tested:**
- ✅ Symlink pattern works: tested `ln -sf build/orch ~/bin/orch` and `~/bin/orch version` correctly runs latest build
- ✅ Current state: found 5 binaries with dual locations causing confusion
- ✅ PATH issue: `/opt/homebrew/bin/orch` (Python) shadows Go orch when `~/bin` isn't first

**What's untested:**
- ⚠️ Cross-project rollout (need to validate in kb-cli, beads, kn, skillc)
- ⚠️ Dylan's workflow impact (may need to run `make build` vs `make install`)
- ⚠️ Daemon restart requirement (daemon hardcodes `/Users/dylanconlin/bin/orch`)

**What would change this:**
- If Dylan frequently works outside source directories (symlink would break)
- If binaries need to be distributable (symlinks don't package)
- If CI/CD needs to install binaries (would need copy-based install)

---

## Consequences

**Positive:**
- No more "forgot to install" failures
- `make build` is now sufficient for human use
- Single source of truth (the build output)
- Eliminates stale binary problem for human usage

**Risks:**
- Symlink confusion if someone doesn't understand the pattern
- Build directory cleanup (`make clean`) breaks the CLI temporarily

**Migration required:**
- [ ] Update orch-go Makefile
- [ ] Validate pattern works
- [ ] Cleanup stale binaries from `~/go/bin/`
- [ ] Remove Python orch from `/opt/homebrew/bin/`
- [ ] Fix PATH order in `~/.zshrc`
- [ ] Update kb-cli, beads, kn, skillc Makefiles
- [ ] Document the pattern in each project's README

---

## Implementation Sequence

1. **Phase 1: Validate in orch-go (this issue)**
   - Update Makefile with symlink pattern
   - Test `make build` → `~/bin/orch` works
   - Document in README

2. **Phase 2: Cleanup (new issue)**
   - Remove stale binaries
   - Fix PATH order
   - Remove Python orch

3. **Phase 3: Propagate to other projects (new issues)**
   - kb-cli
   - beads
   - kn
   - skillc

---

## Related

- Investigation: `.kb/investigations/2025-12-28-inv-solve-stale-binary-problem-human.md`
- Task spawned from: `orch-go-3m23`
