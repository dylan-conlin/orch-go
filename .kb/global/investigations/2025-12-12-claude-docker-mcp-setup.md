# Investigation: Claude Docker MCP Setup

**Date:** 2025-12-12
**Status:** Action Required
**Related:** `.kb/investigations/2025-11-30-claude-code-cross-account-rate-limit-bug.md`

## Context

Following the cross-account rate limit bug workaround (Docker provides fresh Statsig fingerprint), we need to configure the Docker environment to support MCP servers and match native Claude capabilities.

## Findings

### MCP Server Dependencies

The `web-to-markdown` MCP server requires system dependencies not in the base image:

- **shot-scraper** - Python CLI using Playwright
- **markitdown** - HTML to Markdown converter
- **Playwright browser deps** - System libraries for Chromium

Current `.mcp.json` config:
```json
{
  "mcpServers": {
    "web-to-markdown": {
      "command": "node",
      "args": ["/Users/dylanconlin/Documents/personal/mcp/web-to-markdown/index.js"],
      "type": "stdio"
    }
  }
}
```

### Hooks Status

Hooks in `~/.config/claude-code/settings.json` are Node.js scripts - all Linux compatible:
- ✅ `pattern-recognizer.js`
- ✅ `complexity-guard.js`
- ✅ `pattern-learner.js`
- ✅ `consciousness-bootstrap.js`
- ❌ `session-start-hook` (missing - pre-existing issue)

### Strategy Decision

**Native-first, Docker as fallback** when rate-limited.

### Orch Spawn Compatibility

**Problem:** `orch spawn` relies on tmux to create windows and send commands. Without tmux in the container, spawn fails.

**Solution:** Install tmux, tmuxinator, and orch-cli in the Docker image. Run spawn from INSIDE the container.

**Updated Dockerfile additions:**
- `tmux` - Terminal multiplexer for spawn windows
- `ruby` + `tmuxinator` - Session management for orch
- `orch-cli` - Orchestration tooling

**Usage modes:**
```bash
# Direct Claude (no spawn)
claude-docker ~/project

# Tmux mode (for orch spawn)
claude-docker --tmux ~/project
# Then inside: orch spawn investigation "task" --project PROJECT

# Shell mode (debugging)
claude-docker --shell ~/project
```

## Action Items

### 1. Copy updated Docker files

Files already created at `~/claude-docker-updated/`:

```bash
cp ~/claude-docker-updated/Dockerfile ~/.claude/docker-workaround/
cp ~/claude-docker-updated/run.sh ~/.claude/docker-workaround/
chmod +x ~/.claude/docker-workaround/run.sh
```

### 2. Rebuild Docker image

```bash
cd ~/.claude/docker-workaround
docker build -t claude-clean .
```

This will take a few minutes - downloads Playwright browsers (~280MB).

### 3. (Optional) Create alias

```bash
echo 'alias claude-docker="~/.claude/docker-workaround/run.sh"' >> ~/.zshrc
source ~/.zshrc
```

### 4. Test MCP server works

```bash
claude-docker ~/orch-knowledge
# Then inside Claude: use web-to-markdown on any URL
```

## Updated Dockerfile

Location: `~/claude-docker-updated/Dockerfile`

```dockerfile
# Claude Code Docker with MCP Dependencies + orch spawn support
FROM node:22-slim

RUN apt-get update && apt-get install -y \
    git curl jq python3 python3-pip python3-venv openssh-client \
    tmux ruby \
    # Playwright browser dependencies
    libglib2.0-0 libnspr4 libnss3 libdbus-1-3 libatk1.0-0 \
    libatk-bridge2.0-0 libcups2 libxkbcommon0 libatspi2.0-0 \
    libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libgbm1 \
    libcairo2 libpango-1.0-0 libasound2 \
    && rm -rf /var/lib/apt/lists/*

RUN gem install tmuxinator
RUN npm install -g @anthropic-ai/claude-code
RUN pip3 install --break-system-packages shot-scraper markitdown orch-cli
RUN shot-scraper install

WORKDIR /workspace
CMD ["bash"]
```

## Updated Run Script

Location: `~/claude-docker-updated/run.sh`

Commands:
- `claude-docker [dir]` - Start Claude directly
- `claude-docker --tmux [dir]` - Start tmux session (for orch spawn)
- `claude-docker --shell [dir]` - Start bash shell
- `claude-docker --rebuild` - Rebuild image (cached)
- `claude-docker --update` - Rebuild with latest Claude Code (no cache)

## What's Persisted

| Data | Location | Persisted |
|------|----------|-----------|
| Auth | `~/.claude-docker/.credentials.json` | ✅ |
| History | `~/.claude-docker/history.jsonl` | ✅ |
| Statsig | `~/.claude-docker/statsig/` | ✅ |
| Playwright browsers | `~/.cache/ms-playwright/` | ✅ |
| pip packages | `~/.local/` | ✅ |
| orch workspaces | `~/PROJECT/.orch/workspace/` | ✅ (home mounted) |
| tmuxinator configs | `~/.config/tmuxinator/` | ✅ (home mounted) |

## Known Limitations

| Issue | Impact | Workaround |
|-------|--------|------------|
| No browser opening | Can't auto-open URLs | Copy from terminal |
| No clipboard | No pbcopy/pbpaste | Manual copy |
| Slower file I/O | ~10-20% overhead | Acceptable |
| Separate tmux from host | Can't see Docker tmux from host | Use --tmux mode, stay in container |

## Verification

After setup, verify:

1. `claude-docker --rebuild` completes without errors
2. Inside Docker Claude, `web-to-markdown` tool is available and functional
3. Hooks execute (check for pattern detection output)
4. `claude-docker --tmux` starts tmux session
5. Inside container: `orch spawn investigation "test" --project orch-knowledge` works
