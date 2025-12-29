## Summary (D.E.K.N.)

**Delta:** Symlink pattern implemented in orch-go Makefile - `make install` now creates symlink `~/bin/orch → build/orch` instead of copying.

**Evidence:** Tested `make build` → `orch version` showed new build time without running `make install` (build time changed from 05:31:28Z to 05:31:40Z).

**Knowledge:** Codesign must run on build output before creating symlink (macOS requirement); `ln -sf` with absolute path works correctly.

**Next:** Close this issue; propagate pattern to kb-cli, beads, kn, skillc in follow-up issues (orch-go-niyj, orch-go-jj4i).

---

# Investigation: Implement Symlink Pattern in orch-go

**Question:** Can we implement the symlink-based install pattern in orch-go to solve the stale binary problem?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** N/A (this is implementation of decision from orch-go-3m23)

---

## Findings

### Finding 1: Symlink Pattern Works Correctly

**Evidence:** 
```bash
# After make install:
$ ls -la ~/bin/orch
lrwxr-xr-x  1 dylanconlin  staff  56 Dec 28 21:31 /Users/dylanconlin/bin/orch -> /Users/dylanconlin/Documents/personal/orch-go/build/orch

$ orch version
orch version 47fd75d7-dirty
build time: 2025-12-29T05:31:28Z

# After make build (no install):
$ orch version
orch version 47fd75d7-dirty
build time: 2025-12-29T05:31:40Z
```

**Source:** `make install` and `make build` commands

**Significance:** Confirms that `make build` automatically updates what humans run - the core goal of this change.

---

### Finding 2: Codesign Must Run on Build Output

**Evidence:**
Original implementation signed the installed copy. With symlinks, we must sign the source:
```makefile
# Old (copy-based):
cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
codesign --force --sign - $(INSTALL_DIR)/$(BINARY_NAME)

# New (symlink-based):
codesign --force --sign - $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME)
ln -sf $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
```

**Source:** Makefile implementation

**Significance:** macOS requires binary signatures; signing the symlink target works correctly.

---

### Finding 3: CURDIR Provides Absolute Path

**Evidence:**
```bash
$ make -n install
ln -sf /Users/dylanconlin/Documents/personal/orch-go/build/orch /Users/dylanconlin/bin/orch
```

**Source:** `make -n install` dry-run output

**Significance:** $(CURDIR) in Make provides absolute paths automatically - no custom logic needed.

---

## Synthesis

**Key Insights:**

1. **Symlinks solve the "forgot to install" failure mode** - After initial `make install`, subsequent `make build` commands automatically update the CLI.

2. **Codesign placement is critical** - Must codesign the build output before creating symlink, not the symlink itself.

3. **Make's CURDIR handles path resolution** - No need for custom absolute path logic; CURDIR is already absolute.

**Answer to Investigation Question:**

Yes, the symlink pattern works correctly in orch-go. The implementation required:
1. Moving codesign to run on build output instead of installed binary
2. Using `ln -sf` with $(CURDIR) for absolute paths
3. Removing existing file/symlink before creating new symlink

The pattern is validated and ready for propagation to other projects.

---

## Structured Uncertainty

**What's tested:**

- ✅ Symlink created correctly: `ls -la ~/bin/orch` shows symlink to build/orch
- ✅ Binary runs via symlink: `orch version` returns expected output
- ✅ Auto-update on build: build time changed after `make build` without `make install`
- ✅ Codesign works: no signature errors when running via symlink

**What's untested:**

- ⚠️ Daemon restart behavior (requires orchestrator to restart daemon)
- ⚠️ Cross-project propagation (kb-cli, beads, kn, skillc pending)
- ⚠️ `make clean` behavior (would break CLI temporarily until next build)

**What would change this:**

- If macOS rejects symlinked binaries in some contexts (not observed)
- If launchd/daemon can't follow symlinks (unlikely, but needs testing)

---

## Implementation Summary

**Changes made:**

1. **Makefile install target** - Changed from copy to symlink:
   - Codesign runs on `$(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME)`
   - `rm -f` removes existing file/symlink
   - `ln -sf` creates symlink with absolute path

2. **Makefile help** - Updated to clarify "(symlink to build output)"

3. **README.md** - Updated Installation section to document:
   - Symlink pattern explanation
   - First-time setup instructions
   - Why only `make build` is needed after initial install

---

## References

**Files Modified:**
- `Makefile` - Updated install target and help text
- `README.md` - Updated installation documentation

**Commands Run:**
```bash
# Validate dry-run
make -n install

# Execute install
make install

# Verify symlink
ls -la ~/bin/orch

# Test auto-update
make build && orch version
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-28-stale-binary-solution.md` - The decision this implements
- **Investigation:** `.kb/investigations/2025-12-28-inv-solve-stale-binary-problem-human.md` - Prior investigation

---

## Investigation History

**2025-12-28 21:29:** Investigation started
- Task: Implement symlink pattern in orch-go Makefile
- Context: Phase 1 of stale binary solution

**2025-12-28 21:31:** Implementation complete
- Makefile updated with symlink pattern
- README updated with documentation
- Pattern validated with build/version tests
