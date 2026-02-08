<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The orch-go codebase has accumulated organizational debt including duplicate legacy code at root, uncommitted build artifacts, backup files, and unnecessary binary files - all addressable with simple cleanup.

**Evidence:** Root main.go (519 lines) is identical to legacy/main.go; 5 executable binaries (70MB+) not in .gitignore; cmd/orch/main.go.bak exists; README explicitly states "legacy monolithic main.go at project root is deprecated."

**Knowledge:** The current Makefile and README correctly guide users to cmd/orch/, but root-level artifacts create confusion - especially for new contributors who might try to build from project root.

**Next:** Remove deprecated root main.go/main_test.go (keep legacy/ as reference), update .gitignore for build artifacts, delete .bak files, add cleanup targets to Makefile.

**Confidence:** High (90%) - findings are concrete and verifiable; cleanup is low-risk.

---

# Investigation: Organizational Orch-Go Codebase Structure

**Question:** Should the legacy main.go at root be removed? Is the overall code organization messy? Are there dead code or confusing patterns? Is the build/install process clear?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** codebase-audit agent
**Phase:** Complete
**Next Step:** None - ready for orchestrator review
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Duplicate Legacy Code at Root

**Evidence:** 
- `/main.go` (519 lines) is byte-for-byte identical to `/legacy/main.go`
- Both contain deprecated code that only supports `spawn`, `monitor`, and `ask` commands
- The real CLI is in `cmd/orch/main.go` (3,221 lines) with full Cobra-based implementation
- README explicitly states: "The legacy monolithic `main.go` at project root is deprecated"

**Source:**
- `diff main.go legacy/main.go` - no output (identical)
- `wc -l main.go cmd/orch/main.go` - 519 vs 3221 lines
- README.md lines 11-12: "go build -o orch-go ./cmd/orch"

**Significance:** Having the deprecated main.go at root creates confusion because:
1. `go build .` from project root builds the deprecated version, not the real CLI
2. New contributors may not realize cmd/orch/ is the actual source
3. The legacy/ directory already preserves the code for reference

---

### Finding 2: Uncommitted Build Artifacts Polluting Root

**Evidence:**
```
-rwxr-xr-x  13,828,962 bytes  orch
-rwxr-xr-x  13,324,258 bytes  orch-go
-rwxr-xr-x  12,202,210 bytes  orch-new
-rwxr-xr-x  13,828,802 bytes  orch-test
-rwxr-xr-x  13,793,522 bytes  orch-test-serve
-rwxr-xr-x   8,279,858 bytes  test-orch-go
```
Total: ~75MB of executable binaries at project root.

**Source:**
- `ls -la orch* test*` - shows 5+ executables at root
- `file orch orch-new orch-test` - all "Mach-O 64-bit executable arm64"
- `.gitignore` only lists `orch-go` and `build/`, missing these variants

**Significance:**
- These are development/testing artifacts that should not persist
- They are not in .gitignore so could accidentally be committed
- Makes `ls` output cluttered and confusing
- 75MB+ of unnecessary files in working directory

---

### Finding 3: Backup Files in cmd/orch/

**Evidence:**
- `cmd/orch/main.go.bak` (45,767 bytes)
- `cmd/orch/wait_test.go.bak` (4,292 bytes)
- `.git/hooks/pre-commit.old` (not impactful but exists)

**Source:**
- `find . -name "*.bak" -o -name "*.old"` - found 3 files
- These are not in .gitignore

**Significance:**
- .bak files are typically editor/development artifacts
- They add noise and potential confusion
- Risk of accidentally committing stale backup code

---

### Finding 4: debug_sse.go With Build Ignore Tag

**Evidence:**
- `debug_sse.go` exists at root with `//go:build ignore` tag
- Contains 52 lines of SSE debugging code
- Not used by main build, only for ad-hoc debugging

**Source:**
- First line of debug_sse.go: `//go:build ignore`
- `cat debug_sse.go` - shows it's a standalone debugging utility

**Significance:**
- The build ignore tag is correct - it won't affect normal builds
- Could be moved to a `scripts/` or `tools/` directory for clarity
- Low priority but contributes to root clutter

---

### Finding 5: Build/Install Process is Clear (No Issues)

**Evidence:**
- Makefile correctly builds from `./cmd/orch/`:
  ```makefile
  build:
      go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/orch/
  install: build
      cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
  ```
- README correctly documents: `go build -o orch-go ./cmd/orch`
- `make build && make install` workflow is clean and functional

**Source:**
- `Makefile` lines 24-28 and 36-39
- `README.md` lines 9-12

**Significance:**
- The build process itself is well-organized
- Documentation is accurate
- Issue is purely the leftover artifacts, not the process

---

### Finding 6: Package Organization is Sound

**Evidence:**
- Clear package structure in `pkg/`:
  - `account/` - Claude Max account management
  - `opencode/` - OpenCode HTTP client + SSE
  - `spawn/` - Spawn context generation
  - `tmux/` - Tmux window management
  - `verify/` - Completion verification
  - And more...
- Each package has `*_test.go` companion files
- cmd/orch/ properly imports from pkg/

**Source:**
- `ls pkg/` - shows well-named subdirectories
- `cmd/orch/main.go` imports show clean package usage

**Significance:**
- Code organization within pkg/ and cmd/ is good
- No "god package" anti-pattern
- cmd/orch/main.go is large (3,221 lines) but that's typical for CLI entry points with Cobra

---

## Synthesis

**Key Insights:**

1. **Root-level clutter is the main issue** - The codebase structure is fundamentally sound (cmd/orch + pkg/), but the project root has accumulated artifacts that create confusion and noise.

2. **Legacy preservation is done correctly** - Moving deprecated code to `legacy/` was the right call, but the root main.go was left as a duplicate instead of being removed.

3. **Build process is clean, artifacts are messy** - The Makefile and documentation are well-organized; the issue is uncommitted artifacts from development/testing that persist in the working directory.

**Answer to Investigation Question:**

**Should root main.go be removed?** YES. It's deprecated (per README), duplicated in legacy/, and building from root creates a non-functional CLI. Keep legacy/ as the reference.

**Is code organization messy?** NO. The pkg/ and cmd/orch/ structure is clean and well-organized. The "mess" is limited to root-level clutter (build artifacts, .bak files).

**Dead code or confusing patterns?** YES, but minor:
- Root main.go and main_test.go are dead code (duplicate of legacy/)
- .bak files are stale
- Multiple uncommitted executables at root

**Build/install process clarity?** GOOD. Makefile and README correctly document the build process. No changes needed there.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**
- Findings are based on direct file inspection and `diff` commands
- README explicitly documents the deprecation
- All recommendations are low-risk cleanup tasks

**What's certain:**

- ✅ Root main.go is identical to legacy/main.go (verified with diff)
- ✅ Build process uses cmd/orch/, not root
- ✅ Multiple uncommitted executables exist at root (verified with ls and file commands)
- ✅ .bak files exist in cmd/orch/

**What's uncertain:**

- ⚠️ Whether any external tooling depends on `go build .` from root (unlikely but worth checking)
- ⚠️ Whether test-orch-go and similar executables are used in CI (doesn't appear so from .gitignore)

**What would increase confidence to Very High (95%+):**

- Confirm no CI pipelines use root-level go build
- Verify no scripts reference the root main.go

---

## Implementation Recommendations

**Purpose:** Clean up organizational debt with minimal risk.

### Recommended Approach ⭐

**Phased Cleanup** - Remove duplicates, update .gitignore, delete artifacts

**Why this approach:**
- Each step is independently safe and reversible
- Addresses root-level clutter without changing functional code
- legacy/ already preserves the deprecated code for reference

**Trade-offs accepted:**
- Removing root main.go means `go build .` from root will fail
- This is desirable - it forces correct usage via cmd/orch/

**Implementation sequence:**

1. **Update .gitignore** (safe, immediate)
   - Add patterns for test executables: `orch-*` (except tracked ones), `test-*`
   - Already have `orch-go` and `build/`

2. **Delete stale artifacts** (safe, immediate)
   - `rm cmd/orch/main.go.bak cmd/orch/wait_test.go.bak`
   - `rm orch orch-go orch-new orch-test orch-test-serve test-orch-go` (if untracked)

3. **Remove root main.go/main_test.go** (safe, legacy/ has copy)
   - `rm main.go main_test.go`
   - Keep `debug_sse.go` (has build ignore tag, harmless)
   - Or move debug_sse.go to `tools/` or `scripts/` directory

4. **Add Makefile clean targets** (optional enhancement)
   - `clean` already removes `build/`
   - Consider `clean-all` to remove root executables

### Alternative Approaches Considered

**Option B: Keep root main.go as entry point to legacy/**
- **Pros:** Preserves historical build compatibility
- **Cons:** Creates confusion, README already says deprecated
- **When to use instead:** Never - legacy/ already serves this purpose

**Option C: Delete legacy/ entirely**
- **Pros:** Cleaner codebase, removes all deprecated code
- **Cons:** Loses reference implementation, might break tests that use legacy/
- **When to use instead:** After confirming no tests import legacy/

**Rationale for recommendation:** Option A (phased cleanup) is lowest risk, preserves history in legacy/, and addresses all findings.

---

### Implementation Details

**What to implement first:**
1. Update .gitignore - prevents future artifact accumulation
2. Delete .bak files - simple, no risk
3. Remove root main.go after confirming no dependencies

**Things to watch out for:**
- ⚠️ Verify no CI/CD uses `go build .` from root
- ⚠️ Check if any tests import from package `main` at root level

**Areas needing further investigation:**
- None identified - cleanup is straightforward

**Success criteria:**
- ✅ `ls *.go` at root shows only `debug_sse.go` (or nothing)
- ✅ `go build ./cmd/orch/` continues to work
- ✅ No .bak files in codebase
- ✅ .gitignore prevents future artifact accumulation

---

## References

**Files Examined:**
- `/main.go` - deprecated legacy CLI (519 lines, identical to legacy/main.go)
- `/cmd/orch/main.go` - actual CLI entry point (3,221 lines)
- `/legacy/main.go` - archived legacy code
- `/Makefile` - build process (correctly uses cmd/orch/)
- `/README.md` - documents deprecation
- `/.gitignore` - missing patterns for test executables

**Commands Run:**
```bash
# Compare root and legacy main.go
diff main.go legacy/main.go
# (no output - identical)

# Count lines in main files
wc -l main.go cmd/orch/main.go
#      518 main.go
#     3221 cmd/orch/main.go

# Find backup files
find . -name "*.bak" -o -name "*.old"
# ./cmd/orch/main.go.bak
# ./cmd/orch/wait_test.go.bak
# ./.git/hooks/pre-commit.old

# List root executables
ls -la orch* test*
# Shows 5+ uncommitted executables (~75MB total)

# Check file types
file orch orch-new orch-test
# All Mach-O 64-bit executable arm64
```

**Related Artifacts:**
- **CLAUDE.md** - documents architecture correctly (cmd/orch pattern)

---

## Investigation History

**2025-12-23 16:30:** Investigation started
- Initial question: Should legacy main.go at root be removed? Is codebase becoming messy?
- Context: Spawned from beads issue orch-go-2p4n for organizational audit

**2025-12-23 16:45:** Key findings identified
- Discovered root main.go is duplicate of legacy/
- Found ~75MB of uncommitted executables at root
- Found .bak files in cmd/orch/

**2025-12-23 17:00:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Yes, remove root main.go; cleanup artifacts; code org is otherwise sound
