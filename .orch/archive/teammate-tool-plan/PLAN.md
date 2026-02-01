# TeammateTool Integration Plan for orch-go

## Overview
Add TeammateTool integration to enable orch-go agents to participate in Claude Code native swarms.

## Steps

### 1. Create pkg/teammate package
- Define Go structs for teammate messages (write, broadcast, spawnTeam, etc.)
- Implement message serialization for the teammate protocol

### 2. Add teammate discovery
- Read team config from `~/.claude/teams/{team-name}/config.json`
- Parse member list (name, agentId, agentType)
- Implement `discoverTeams` operation

### 3. Implement core operations
- `write`: Send message to specific teammate
- `broadcast`: Send message to all teammates (with cost awareness)
- `spawnTeam`/`cleanup`: Team lifecycle management

### 4. Integrate with spawn flow
- Pass `CLAUDE_CODE_TEAM_NAME` and `CLAUDE_CODE_AGENT_ID` env vars to spawned agents
- Enable spawned agents to join existing teams

### 5. Add CLI commands
- `orch team list` - Show active teams
- `orch team send <agent> "message"` - Send teammate message
- `orch team join <team>` - Request to join a team

## Deliverables
- `pkg/teammate/` package with core types and operations
- Updated spawn flow with team context
- New `orch team` subcommand group
