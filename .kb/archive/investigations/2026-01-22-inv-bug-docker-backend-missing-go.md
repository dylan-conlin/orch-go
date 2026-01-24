<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Docker backend was missing Go, causing auto-rebuild hooks to fail when agents modify `.go` files.

**Evidence:** `~/.claude/hooks/post-tool-use.sh:46-53` runs `make install` on Go file changes; Dockerfile had Python/Ruby/Node but no Go.

**Knowledge:** The `~/.claude/docker-workaround/` directory and Dockerfile didn't exist - recreated with Go (`golang` package) added.

**Next:** Rebuild Docker image with `docker build -t claude-code-mcp ~/.claude/docker-workaround/` and test.

**Promote to Decision:** recommend-no - Bug fix, not architectural change.

---

# Investigation: Bug Docker Backend Missing Go

**Question:** Why do auto-rebuild hooks fail in Docker backend spawns?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None - Dockerfile created, ready for rebuild
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Auto-rebuild hook requires Go

**Evidence:** `~/.claude/hooks/post-tool-use.sh` lines 46-53:
```bash
# Automatically run 'make install' for orch-go if Go files were modified
if [[ "$COMMAND" =~ \.go ]]; then
    if [[ "$PWD" == *"/orch-go" ]]; then
        echo "🛠️  Go files modified in orch-go, running 'make install'..." >&2
        make install >/dev/null 2>&1 || echo "⚠️  'make install' failed" >&2
    fi
fi
```

**Source:** `~/.claude/hooks/post-tool-use.sh:46-53`

**Significance:** Any Bash command containing `.go` in the orch-go directory triggers `make install`, which requires Go compiler.

---

### Finding 2: Docker workaround directory didn't exist

**Evidence:** `ls ~/.claude/docker-workaround/` returned "No such file or directory"

**Source:** Bash check during investigation

**Significance:** The Dockerfile referenced in CLAUDE.md and prior investigations was never created or was deleted.

---

### Finding 3: Prior Dockerfile design had no Go

**Evidence:** Investigation `.kb/investigations/2025-12-12-claude-docker-mcp-setup.md` shows original Dockerfile with Python, Ruby, Node.js, Playwright deps, but no Go.

**Source:** `.kb/investigations/2025-12-12-claude-docker-mcp-setup.md:110-131`

**Significance:** Go was never in the original Docker image design - it was expected that cross-compiled binaries would be used via `~/.local/bin/linux-amd64`.

---

## Synthesis

**Key Insights:**

1. **Cross-compilation wasn't enough** - While `bd`, `orch`, `kb`, etc. are cross-compiled to `~/.local/bin/linux-amd64`, the auto-rebuild hook needs Go to run `make install` for orch-go changes.

2. **Docker workaround was never deployed** - The directory `~/.claude/docker-workaround/` didn't exist despite being referenced in multiple places.

3. **Simple fix** - Adding `golang` to apt-get install resolves the issue.

**Answer to Investigation Question:**

Auto-rebuild hooks fail because the Docker image lacks Go. The `post-tool-use.sh` hook runs `make install` when `.go` files are modified, requiring the Go compiler. Solution: Add `golang` package to the Dockerfile.

---

## Structured Uncertainty

**What's tested:**

- ✅ Dockerfile created at `~/.claude/docker-workaround/Dockerfile` (verified: file created)
- ✅ run.sh script created and made executable (verified: chmod +x successful)
- ✅ Root cause identified in post-tool-use.sh (verified: code inspection)

**What's untested:**

- ⚠️ Docker image build (cannot test in sandbox - no Docker)
- ⚠️ `make install` actually works in container (requires built image)
- ⚠️ Go version compatibility with orch-go (Debian golang package may be older)

**What would change this:**

- If orch-go requires Go 1.22+ and Debian package is older, need different Go install method
- If MCP server dependencies conflict with Go, may need multi-stage build

---

## Implementation Recommendations

### Recommended Approach: Add golang to apt-get

**Why this approach:**
- Minimal change to existing Dockerfile pattern
- Debian golang package is well-maintained
- Matches container's package manager conventions

**Trade-offs accepted:**
- Golang version determined by Debian repos (may be older)
- Slightly larger image size (~100MB)

**Implementation sequence:**
1. Dockerfile created with `golang` in apt-get
2. User rebuilds image: `docker build -t claude-code-mcp ~/.claude/docker-workaround/`
3. Test by spawning agent that modifies .go files

### Alternative Approaches Considered

**Option B: Install Go from official tarball**
- **Pros:** Exact version control (e.g., go1.23.x)
- **Cons:** More complex Dockerfile, manual version updates
- **When to use instead:** If Debian golang is too old for orch-go

---

## References

**Files Examined:**
- `~/.claude/hooks/post-tool-use.sh` - Auto-rebuild hook implementation
- `~/.claude/hooks/stale-binary-warning.sh` - Related binary staleness check
- `.kb/investigations/2025-12-12-claude-docker-mcp-setup.md` - Original Docker design
- `.kb/investigations/2026-01-20-inv-design-claude-docker-backend-integration.md` - Docker backend design
- `pkg/spawn/docker.go` - Docker spawn implementation

**Commands Run:**
```bash
# Check for existing Dockerfile
ls -la ~/.claude/docker-workaround/

# Search for Go-related hooks
grep -l "go build\|make install" ~/.claude/hooks/*
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-12-claude-docker-mcp-setup.md` - Original Docker MCP setup
- **Investigation:** `.kb/investigations/2026-01-20-inv-implement-docker-backend-orch-spawn.md` - Docker backend implementation

---

## Files Created

**`~/.claude/docker-workaround/Dockerfile`** - Docker image definition with Go added

**`~/.claude/docker-workaround/run.sh`** - Convenience wrapper script

---

## Rebuild Instructions

```bash
# Rebuild Docker image with Go
cd ~/.claude/docker-workaround
docker build -t claude-code-mcp .

# Or use convenience script
~/.claude/docker-workaround/run.sh --rebuild
```

---

## Investigation History

**2026-01-22 19:10:** Investigation started
- Initial question: Why do auto-rebuild hooks fail in Docker backend?
- Context: Bug report that Go binary missing in Docker image

**2026-01-22 19:15:** Root cause found
- `post-tool-use.sh` runs `make install` on .go file changes
- Docker workaround directory didn't exist

**2026-01-22 19:20:** Investigation completed
- Status: Complete
- Key outcome: Dockerfile and run.sh created with Go added
