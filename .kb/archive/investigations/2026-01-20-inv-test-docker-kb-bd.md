# Investigation: Test Docker with kb/bd

**TLDR:** Docker backend spawns cannot use host kb/bd CLIs - they are macOS binaries incompatible with the Linux container environment.

**Status:** Complete
**Date:** 2026-01-20
**Type:** Investigation

## Question

Can agents spawned via Docker backend use the kb and bd CLI tools?

## What I Tried

1. Spawned agent via `orch spawn --backend docker hello "test docker with kb bd"`
2. Attempted to run `kb create investigation test-docker-kb-bd`
3. Checked binary paths and architecture

## What I Observed

1. `kb create` failed with: `cannot execute binary file: Exec format error`
2. `file /Users/dylanconlin/.local/bin/kb` - "kb not found" (not mounted in container)
3. `uname -a` shows: `Linux ... aarch64 GNU/Linux` (container runs Linux ARM64)

The kb and bd binaries are:
- Either macOS-compiled binaries (Mach-O) which can't run on Linux
- Or not mounted into the Docker container at all

## Test Performed

```bash
# Inside Docker container
pwd  # /Users/dylanconlin/Documents/personal/orch-go (mounted correctly)
kb create investigation test-docker-kb-bd  # Exit code 126: cannot execute binary file
uname -a  # Linux aarch64 (not Darwin)
```

## Conclusion

**Docker backend spawns are incompatible with kb/bd CLI tools.**

To support kb/bd in Docker spawns, options include:
1. Build Linux ARM64 versions of kb/bd and include in Docker image
2. Build Linux x86_64 versions (requires emulation on ARM Macs)
3. Use kb/bd as a service/API instead of CLI tools
4. Document that Docker spawns cannot use kb/bd and adjust spawn context accordingly

## Impact

Spawns using `--backend docker` that require kb/bd operations (like creating investigation files, bd comment updates, bd close) will fail. The spawn context should likely omit kb/bd instructions for Docker backend spawns.

## Recommendation

For now, use `--backend claude` (tmux) when kb/bd integration is required. Docker backend is best for:
- Simple verification tasks (like this hello skill)
- Rate limit bypass scenarios where kb/bd isn't needed
- Tasks that don't require beads issue tracking
