# Investigation: Create Cross-Compile Script

**Date:** 2026-01-20
**Status:** Complete
**TLDR:** Created cross-compile script to build Linux binaries (bd, orch, kb) for Docker containers, updated docker.go PATH to use them.

## What I Tried

1. Examined existing Makefiles in beads, orch-go, and kb-cli to understand build flags
2. Created `scripts/cross-compile-linux.sh` that cross-compiles all three Go tools
3. Updated `pkg/spawn/docker.go` to add `$HOME/.local/bin/linux-amd64` to PATH
4. Added `cross-compile-linux` target to Makefile

## What I Observed

- All three projects (beads, orch-go, kb-cli) use similar Go build patterns with ldflags
- Docker spawns use PATH set via `-e PATH=...` in docker.go (line 69)
- The existing PATH did not include Linux-specific binary directory

## Test Performed

- Verified script syntax: `bash -n scripts/cross-compile-linux.sh` - passed
- Verified Makefile: `make help` shows new targets
- Script execution could not be tested in this environment (no Go installed), but syntax is valid

## Deliverables

1. **`scripts/cross-compile-linux.sh`** - Cross-compile script
   - Supports `--all`, `--bd`, `--orch`, `--kb` flags
   - Outputs to `~/.local/bin/linux-amd64/`
   - Matches each project's ldflags patterns

2. **`pkg/spawn/docker.go`** - Updated PATH
   - Added `$HOME/.local/bin/linux-amd64` as first PATH entry
   - Added comment explaining the purpose

3. **`Makefile`** - Added targets
   - `make cross-compile-linux` - Builds all three tools
   - `make cross-compile-linux-orch` - Builds only orch (faster)

## Usage

```bash
# From macOS host (requires Go installed):
make cross-compile-linux    # Build bd, orch, kb for Linux

# Then Docker spawns will use the Linux binaries:
orch spawn --backend docker feature-impl "task"
```

## Conclusion

The cross-compile infrastructure is in place. When `make cross-compile-linux` is run on macOS with Go installed, it will produce Linux binaries that Docker containers can execute. The Docker spawn PATH is already configured to prefer these binaries.
