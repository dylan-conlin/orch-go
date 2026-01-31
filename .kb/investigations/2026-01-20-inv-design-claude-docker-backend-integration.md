<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Docker backend provides Statsig fingerprint isolation but requires significant architectural changes for full integration with orch-go's monitoring/lifecycle systems.

**Evidence:** Existing Docker workaround works for fingerprint isolation but lacks: session tracking, dashboard visibility, registry integration, and lifecycle management. Claude CLI backend pattern provides implementation template.

**Knowledge:** Docker backend is most valuable as a "double escape hatch" for rate limit scenarios, not as a primary spawn path. Integration should be minimal - leverage tmux-in-tmux (host tmux attaching to container) rather than dashboard SSE.

**Next:** Implement minimal viable integration: `--backend docker` flag that wraps claude CLI spawn in Docker container with fresh fingerprint.

**Promote to Decision:** Actioned - decision exists (dual-spawn-mode-architecture)

---

# Investigation: Design Claude Docker Backend Integration

**Question:** How should Docker be integrated as a third backend option for orch spawn, addressing session management, dashboard integration, credential handling, concurrency, and tmux integration?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None - Ready for implementation
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-01-09-dual-spawn-mode-architecture.md
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Docker Workaround Already Exists and Works

**Evidence:**
- Dockerfile at `~/.claude/docker-workaround/Dockerfile` with Claude Code, tmux, MCP dependencies
- `run.sh` script supports three modes: `--shell`, `--tmux`, and direct claude
- Investigation `2025-11-30-claude-code-cross-account-rate-limit-bug.md` confirms Docker provides fresh Statsig fingerprint

**Source:**
- `~/.claude/docker-workaround/Dockerfile` (59 lines)
- `~/.claude/docker-workaround/run.sh` (98 lines)
- `~/.kb/investigations/2025-11-30-claude-code-cross-account-rate-limit-bug.md:129` - "CONFIRMED WORKING"

**Significance:** We don't need to build Docker infrastructure from scratch. The existing setup is proven. Integration is about connecting it to orch-go's lifecycle management.

---

### Finding 2: Current Dual Backend Pattern Provides Integration Template

**Evidence:**
- `cmd/orch/backend.go` has clean priority chain: flags > config > defaults
- `pkg/spawn/claude.go` shows tmux integration pattern: create window, send command via tmux keys
- Registry tracks agents with `mode` field ("claude" | "opencode") and mode-specific fields (`SessionID` for opencode, `TmuxWindow` for claude)

**Source:**
- `cmd/orch/backend.go:19-83` - `resolveBackend()` function
- `pkg/spawn/claude.go:10-70` - `SpawnClaude()` implementation
- `cmd/orch/spawn_cmd.go:1862-1924` - `runSpawnClaude()` with registry integration

**Significance:** Docker can follow the same pattern - add "docker" as third mode value, add docker-specific fields to registry, implement `SpawnDocker()` function similar to `SpawnClaude()`.

---

### Finding 3: Dashboard SSE Integration Is Infeasible for Docker

**Evidence:**
- Dashboard relies on OpenCode HTTP API (`/sessions/:id`, `/api/events` SSE)
- Docker runs isolated Claude CLI - no OpenCode server inside container
- Claude CLI doesn't expose SSE or HTTP interface for monitoring
- Escape Hatch Visibility Architecture model explicitly states visibility comes from tmux, not dashboard

**Source:**
- `cmd/orch/serve_agents.go` - Queries OpenCode sessions for agent status
- `.kb/models/escape-hatch-visibility-architecture.md:36-45` - Independence/Visibility/Capability triad
- `.kb/models/dashboard-architecture.md:46` - Dashboard architecture depends on OpenCode API

**Significance:** Docker backend cannot have dashboard visibility. This is acceptable because Docker is an escape hatch - escape hatches trade dashboard visibility for independence. Use tmux for visibility.

---

### Finding 4: tmux-in-tmux Problem Has Solution

**Evidence:**
- Existing Docker workaround uses tmux inside container (`claude-docker --tmux`)
- Creates nested tmux session (host tmux → container bash → container tmux)
- Alternative: Use host tmux to spawn Docker container, container runs claude directly (no nested tmux)

**Source:**
- `~/.claude/docker-workaround/run.sh:81-82` - tmux mode spawns internal tmux session
- `pkg/spawn/claude.go:31-34` - Host tmux window creation pattern
- `.kb/investigations/2025-12-12-claude-docker-mcp-setup.md:163` - "Separate tmux from host" listed as known limitation

**Significance:** Two approaches possible:
1. **Host-tmux approach**: Create host tmux window, run `docker run ... claude` inside it - simpler, no nested sessions
2. **Nested approach**: Run docker with internal tmux - more complex, harder to monitor

Host-tmux is better - matches existing claude backend pattern.

---

### Finding 5: Credential Handling Strategy

**Evidence:**
- Existing workaround mounts `~/.claude-docker/` as `~/.claude` inside container
- OAuth tokens stored in `.credentials.json`
- Fresh container = fresh Statsig fingerprint = new "device" to rate limit system
- Rate limit is per-device, not per-account

**Source:**
- `~/.claude/docker-workaround/run.sh:58-66` - Volume mounting configuration
- `~/.kb/investigations/2025-11-30-claude-code-cross-account-rate-limit-bug.md:101-121` - Why Docker works
- `~/.kb/investigations/2025-11-30-claude-code-cross-account-rate-limit-bug.md:150-153` - Separate `.claude-docker` config dir

**Significance:** Use separate config directory (`~/.claude-docker/`) to maintain fresh fingerprint. First run requires `claude login`, subsequent runs use cached OAuth. Option to wipe fingerprint for true fresh start.

---

### Finding 6: Container Lifecycle Decision Point

**Evidence:**
- Docker startup adds ~2-5 seconds overhead (image load, container creation)
- Persistent container would need orchestration (keep-alive, health checks)
- Current Claude backend creates fresh tmux window per spawn - simple, works well
- Docker escape hatch used rarely (rate limit scenarios) - startup overhead acceptable

**Source:**
- `pkg/spawn/claude.go:18-25` - Each spawn creates fresh tmux session/window
- `.kb/models/model-access-spawn-paths.md:217-222` - "Reserve escape hatch for critical work due to ergonomic overhead"

**Significance:** One container per spawn is simpler and sufficient. Persistent container pool adds complexity for minimal benefit given low expected usage.

---

### Finding 7: Backend Selection Priority Must Not Auto-Override

**Evidence:**
- Infrastructure detection in `resolveBackend()` is advisory-only (warns, doesn't override)
- Decision explicitly rejects auto-override pattern: "NEVER overrides the backend - warnings only"
- Docker should follow same pattern - advisory when rate-limited, never forced

**Source:**
- `cmd/orch/backend.go:85-99` - `addInfrastructureWarning()` is advisory
- `cmd/orch/spawn_cmd.go:179` - `--backend` flag comment: "Overrides config and auto-selection"

**Significance:** Docker backend should be explicit opt-in (`--backend docker`), not auto-selected. May add rate-limit advisory warning similar to infrastructure warning.

---

## Synthesis

**Key Insights:**

1. **Docker as Third Escape Hatch** - Docker isn't replacing claude or opencode backends; it's a "double escape hatch" for when claude backend hits rate limits. Low usage frequency means simplicity > optimization.

2. **Host tmux, Not Nested tmux** - Run Docker container inside host tmux window (like claude backend), not tmux inside Docker. Matches existing pattern, avoids nested session complexity.

3. **No Dashboard, tmux for Visibility** - Accept that Docker backend has no dashboard visibility. This matches escape hatch philosophy - trade convenience for independence. Use host tmux pane for monitoring.

4. **Fresh Fingerprint is the Value** - The entire point of Docker is Statsig fingerprint isolation. Each spawn should get fresh fingerprint. One container per spawn, separate config directory.

**Answer to Investigation Question:**

Docker should be integrated as a minimal third backend that:
- Uses `--backend docker` explicit flag (no auto-selection)
- Creates host tmux window, runs `docker run ... claude` inside
- Uses separate `~/.claude-docker/` config directory for fingerprint isolation
- Tracks in registry with `mode: "docker"` and `TmuxWindow` field (same as claude)
- Has no dashboard visibility (tmux provides visibility)
- One container per spawn (simplicity over optimization)
- Same lifecycle commands: `orch status`, `orch complete`, `orch abandon` via tmux

---

## Structured Uncertainty

**What's tested:**

- ✅ Docker workaround provides fresh Statsig fingerprint (verified: working since Dec 2025)
- ✅ Existing run.sh successfully mounts volumes and runs claude (verified: documented in investigation)
- ✅ Host tmux window creation pattern works for claude backend (verified: production usage)

**What's untested:**

- ⚠️ Container startup overhead in practice (estimated 2-5s, not benchmarked)
- ⚠️ OAuth token persistence across container restarts
- ⚠️ MCP servers work correctly inside container (documented as working, not recently tested)
- ⚠️ Host tmux + Docker interaction (would container inherit host terminal properly?)

**What would change this:**

- If Docker startup overhead is >10s, consider persistent container approach
- If OAuth tokens don't persist, need volume mount for keychain
- If Anthropic changes Statsig fingerprinting to detect Docker, escape hatch loses value

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach: Host tmux + Docker Container

**Approach Name:** Minimal Docker Backend as Third Escape Hatch

**Why this approach:**
- Matches existing claude backend pattern (host tmux window, send command)
- Simplest implementation (reuse 90% of claude spawn logic)
- Fresh fingerprint per spawn (one container per spawn)
- tmux provides visibility without dashboard integration complexity

**Trade-offs accepted:**
- No dashboard visibility (acceptable for escape hatch)
- 2-5s startup overhead per spawn (acceptable for rare usage)
- No persistent container optimization (simplicity over performance)

**Implementation sequence:**

1. **Add backend enum value** - Add "docker" to valid backends in `backend.go`
2. **Create SpawnDocker function** - Similar to SpawnClaude, but runs `docker run` instead of `claude` directly
3. **Update registry schema** - Docker mode uses same TmuxWindow field as claude
4. **Add --backend docker flag** - Wire through spawn command
5. **Test with existing Docker setup** - Verify integration with `~/.claude/docker-workaround/`

### Alternative Approaches Considered

**Option B: Nested tmux (Docker runs internal tmux)**
- **Pros:** Existing run.sh already supports this mode
- **Cons:** Confusing keybindings, harder to monitor from host, doesn't match claude pattern
- **When to use instead:** If host tmux + docker interaction proves problematic

**Option C: Full dashboard integration via proxy**
- **Pros:** Docker agents visible in dashboard
- **Cons:** Requires running OpenCode server inside container, complex networking, defeats independence criterion
- **When to use instead:** Never - violates escape hatch principles

**Option D: Persistent container pool**
- **Pros:** No startup overhead
- **Cons:** Complex orchestration (health checks, cleanup), overkill for rare usage
- **When to use instead:** If spawn frequency dramatically increases

**Rationale for recommendation:** Option A (host tmux + single container) matches existing patterns, minimizes implementation effort, and correctly positions Docker as escape hatch rather than primary path.

---

### Implementation Details

**What to implement first:**
1. `pkg/spawn/docker.go` - SpawnDocker function (modeled on claude.go)
2. `cmd/orch/backend.go` - Add "docker" to valid backend values
3. Wire `--backend docker` flag through spawn command

**Things to watch out for:**
- ⚠️ Docker image must exist (`claude-code-mcp`) - add check with helpful error
- ⚠️ Container user permissions - match host user (existing run.sh handles this)
- ⚠️ Working directory - mount project dir correctly inside container
- ⚠️ CLAUDE_CONTEXT env var - pass through to container (for hook coordination)

**Areas needing further investigation:**
- MCP server functionality inside container (may need testing)
- Git credential passthrough for agents that commit
- Performance benchmarking for startup overhead

**Success criteria:**
- ✅ `orch spawn --backend docker investigation "test"` creates tmux window with docker-spawned claude
- ✅ `orch status` shows docker-spawned agents with correct mode
- ✅ `orch complete` works for docker agents (same as claude backend)
- ✅ Fresh Statsig fingerprint per spawn (verify with rate limit test)

---

## Implementation Code Sketch

```go
// pkg/spawn/docker.go

package spawn

import (
    "fmt"
    "github.com/dylan-conlin/orch-go/pkg/tmux"
)

// SpawnDocker launches a Claude Code agent in Docker via host tmux window.
// This provides Statsig fingerprint isolation for rate limit escape hatch.
func SpawnDocker(cfg *Config) (*tmux.SpawnResult, error) {
    // 1. Ensure tmux session exists (same as claude mode)
    sessionName, err := tmux.EnsureWorkersSession(cfg.Project, cfg.ProjectDir)
    if err != nil {
        return nil, fmt.Errorf("failed to ensure tmux session: %w", err)
    }

    // 2. Build window name
    windowName := tmux.BuildWindowName(cfg.WorkspaceName, cfg.SkillName, cfg.BeadsID)

    // 3. Create detached window
    windowTarget, windowID, err := tmux.CreateWindow(sessionName, windowName, cfg.ProjectDir)
    if err != nil {
        return nil, fmt.Errorf("failed to create tmux window: %w", err)
    }

    // 4. Build Docker command
    // Uses existing ~/.claude/docker-workaround/ setup
    contextPath := cfg.ContextFilePath()
    dockerCmd := fmt.Sprintf(
        "docker run -it --rm "+
            "--user \"$(id -u):$(id -g)\" "+
            "-v \"$HOME\":\"$HOME\" "+
            "-v \"$HOME/.claude-docker\":\"$HOME/.claude\" "+
            "-w %q "+
            "-e HOME=\"$HOME\" "+
            "-e CLAUDE_CONTEXT=%s "+
            "-e TERM=xterm-256color "+
            "claude-code-mcp "+
            "bash -c 'cat %q | claude --dangerously-skip-permissions'",
        cfg.ProjectDir,
        inferClaudeContext(cfg),
        contextPath,
    )

    // 5. Send command to tmux window
    if err := tmux.SendKeys(windowTarget, dockerCmd); err != nil {
        return nil, fmt.Errorf("failed to send docker command: %w", err)
    }
    if err := tmux.SendEnter(windowTarget); err != nil {
        return nil, fmt.Errorf("failed to send enter: %w", err)
    }

    return &tmux.SpawnResult{
        Window:        windowTarget,
        WindowID:      windowID,
        WindowName:    windowName,
        WorkspaceName: cfg.WorkspaceName,
    }, nil
}

func inferClaudeContext(cfg *Config) string {
    switch {
    case cfg.IsMetaOrchestrator:
        return "meta-orchestrator"
    case cfg.IsOrchestrator:
        return "orchestrator"
    default:
        return "worker"
    }
}
```

---

## References

**Files Examined:**
- `cmd/orch/backend.go` - Backend selection priority chain
- `cmd/orch/spawn_cmd.go:1-200` - Spawn command flags and modes
- `cmd/orch/spawn_cmd.go:800-1200` - Backend resolution and mode handling
- `cmd/orch/spawn_cmd.go:1862-1924` - runSpawnClaude implementation
- `pkg/spawn/claude.go` - Claude backend tmux integration
- `~/.claude/docker-workaround/Dockerfile` - Existing Docker image
- `~/.claude/docker-workaround/run.sh` - Existing Docker runner script

**Commands Run:**
```bash
# Check existing Docker workaround
ls -la ~/.claude/docker-workaround/
```

**External Documentation:**
- `~/.kb/investigations/2025-11-30-claude-code-cross-account-rate-limit-bug.md` - Docker workaround origin
- `~/.kb/investigations/2025-12-12-claude-docker-mcp-setup.md` - MCP setup in Docker

**Related Artifacts:**
- **Model:** `.kb/models/model-access-spawn-paths.md` - Current dual spawn architecture
- **Model:** `.kb/models/escape-hatch-visibility-architecture.md` - Escape hatch principles
- **Guide:** `.kb/guides/dual-spawn-mode-implementation.md` - Implementation patterns
- **Decision:** `.kb/decisions/2026-01-09-dual-spawn-mode-architecture.md` - Dual mode decision

---

## Implementation History

**2026-01-20 10:45:** Investigation started
- Initial question: How to integrate Docker as third backend for orch spawn?
- Context: Docker provides Statsig fingerprint isolation for rate limit escape hatch

**2026-01-20 11:30:** Key findings synthesized
- Existing Docker workaround is working foundation
- Host tmux + Docker container is simplest approach
- Dashboard integration infeasible and unnecessary (escape hatch philosophy)

**2026-01-20 11:45:** Investigation completed
- Status: Complete
- Key outcome: Docker backend should be minimal third escape hatch with host tmux pattern
