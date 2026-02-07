# Guide: Claude Code Sandbox Architecture

**Created:** 2026-01-21
**Purpose:** Understanding and configuring Claude Code's Linux sandbox for orch-go development

## Overview

Claude Code runs commands in a **Linux sandbox** using bubblewrap (bwrap), even on macOS. This has implications for binary compatibility, network access, and service management.

## Key Concepts

### Platform Reality

```bash
# Inside Claude Code agent:
$ uname -a
Linux ... aarch64 GNU/Linux

# Your macOS terminal:
$ uname -a
Darwin ... arm64
```

Agents run in Linux, not macOS. This affects which binaries can execute.

### Binary Compatibility

| Binary Type | Works in Sandbox? | Example |
|-------------|-------------------|---------|
| Linux ARM64 | ✅ Yes | `/usr/local/bin/orch` (cross-compiled) |
| Linux AMD64 | ✅ Yes (emulated) | `~/.local/bin/linux-amd64/*` |
| macOS ARM64 | ❌ No | `~/.bun/bin/opencode` → "Exec format error" |
| Shell scripts | ✅ Yes | Most utility scripts |
| Node.js/Python | ✅ Yes | Interpreted, platform-independent |

### Network Isolation

The sandbox has **network isolation** from the host:

```bash
# From sandbox - FAILS:
$ curl http://localhost:4096
# connection refused

# From macOS terminal - WORKS:
$ curl http://localhost:4096
# {"sessions": [...]}
```

Agents cannot reach services running on the macOS host via localhost.

## Configuration: excludedCommands

Commands in `excludedCommands` bypass the sandbox and run directly on macOS:

```json
// ~/.claude/settings.json
{
  "sandbox": {
    "excludedCommands": [
      "orch",           // Needs OpenCode API access
      "bd",             // Beads commands
      "orch-dashboard", // Service management
      "overmind",       // Process manager
      "docker",         // Needs Docker socket
      "colima",         // Container runtime
      "open",           // Open URLs/files in macOS
      "osascript"       // macOS automation
    ]
  }
}
```

**Restart Claude Code after changing settings.**

## Cross-Compilation for Sandbox

To make Go binaries available in the sandbox, cross-compile for Linux:

```bash
# Build Linux binaries
make cross-compile-linux

# Output: ~/.local/bin/linux-amd64/
#   - orch
#   - bd
#   - kb
```

These are then available via PATH in the sandbox.

### How Binaries Get into /usr/local/bin/

The sandbox mounts certain host directories. Binaries in `/usr/local/bin/` that are Linux builds become available to agents. The exact provisioning mechanism depends on Claude Code version.

## Common Issues

### "Exec format error"

```
/bin/bash: ~/.bun/bin/opencode: cannot execute binary file: Exec format error
```

**Cause:** Trying to run a macOS binary in the Linux sandbox.
**Fix:** Use a Linux build or add to `excludedCommands`.

### "connection refused" to localhost

```
dial tcp 127.0.0.1:4096: connect: connection refused
```

**Cause:** Sandbox network isolation.
**Fix:** Add the command to `excludedCommands` so it runs on host.

### orch serve fails with "unknown/pkg/certs/cert.pem"

```
Error: open unknown/pkg/certs/cert.pem: no such file or directory
```

**Cause:** Binary built without ldflags (`sourceDir: unknown`).
**Fix:** Use `make install` not `go build` directly.

## Implications for orch-go

1. **Agents can edit files** - Filesystem is accessible via mounts
2. **Agents can run tests** - Go/Node available in sandbox
3. **Agents cannot start services** - Use `excludedCommands` for service management
4. **Agents cannot query OpenCode directly** - Unless `orch` is in `excludedCommands`
5. **Dashboard provides visibility** - Use browser dashboard since agents can't query status

## References

- Settings file: `~/.claude/settings.json`
- Cross-compile script: `scripts/cross-compile-linux.sh`
- Sandbox docs: https://docs.anthropic.com/claude-code/sandbox
