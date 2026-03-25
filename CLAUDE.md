# orch-go

Go rewrite of orch-cli - AI agent orchestration via OpenCode API.

## Architecture & Packages

See `.kb/guides/architecture-overview.md` for system diagram, directory structure, spawn backends, and architectural principles.

See `.kb/guides/key-packages.md` for detailed package descriptions.

## Commands

See `.kb/guides/cli.md` for the full CLI reference.

**Common commands:**

```bash
orch spawn feature-impl "implement X" --issue proj-123   # Spawn from issue
orch spawn --dry-run feature-impl "implement X"           # Dry run
orch spawn --explore investigation "how does X work?"     # Parallel decomposition
orch spawn --thread daemon-capacity feature-impl "fix Y"  # Spawn linked to thread
orch work proj-123 --inline                               # Blocking TUI (--inline on work, not spawn)
orch account switch work                                  # Switch accounts
orch wait proj-123 --timeout 30m                          # Wait for completion
orch complete proj-123                                    # Verify and close
orch complete proj-123 --headless                         # Non-interactive (daemon-triggered)
orch clean                                                # Clean up finished agents
orch thread new "How X relates to Y"                      # Create living thread
orch thread list                                          # List threads with status
orch opsec status                                         # Check OPSEC proxy health
orch opsec install                                        # Install OPSEC as environmental enforcement
```

## Development

```bash
make build      # Build
make test       # Test
make install    # Install to ~/bin/orch
orch version    # Verify version
```

### Adding New Commands

1. Add command to `cmd/orch/main.go` (inline with Cobra)
2. Or create `cmd/orch/{name}.go` for complex commands
3. Add to `rootCmd.AddCommand()` in init()

### Adding New Packages

1. Create `pkg/{name}/{name}.go`
2. Create `pkg/{name}/{name}_test.go`
3. Import in cmd/orch as needed

## Accretion Boundaries

**Rule:** Files >1,500 lines require extraction before feature additions. Run `orch hotspot` to check current bloated files. If modifying large files, see `.kb/guides/code-extraction-patterns.md` for extraction workflow.

**Enforcement (advisory):**
- **Spawn context advisory:** Hotspot info injected into SPAWN_CONTEXT.md for agent awareness.
- **Daemon escalation:** Daemon routes feature-impl/systematic-debugging to architect when issue targets hotspot files.
- **Completion gates (warning):** Warn on additions >50 lines to files >800 lines.
- Decision: `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md` (converted from blocking to advisory after 100% bypass rate over 2-week measurement)

## Architectural Constraints

### No Local Agent State

orch-go must not maintain local agent state (registries, projection DBs, SSE materializers, caches for agent discovery).
Query beads and OpenCode directly. If queries are slow, fix the authoritative source; do not build a projection.

## Knowledge Base Structure

This project has two knowledge directories:

- **`.kb/`** — Project-level knowledge (models, guides, decisions, investigations specific to orch-go)
- **`.kb/global/`** — Cross-project knowledge (models, guides, decisions shared across all projects)

`~/.kb` is a symlink to `.kb/global/`. The `kb context` CLI searches both automatically.

**When searching for models, guides, or investigations, always check BOTH paths:**
- `.kb/models/` — project models (e.g., spawn-architecture, daemon-autonomous-operation)
- `.kb/global/models/` — cross-project models (e.g., behavioral-grammars, skillc-testing)
- `.kb/guides/` — project guides
- `.kb/global/guides/` — cross-project guides (e.g., meta-orchestrator-mental-models)
- `.kb/decisions/` + `.kb/global/decisions/` — same pattern

**When creating probes for global models:** write to `.kb/global/models/{name}/probes/`, not `.kb/models/`.

## Key References

**Before debugging, check the relevant guide in `.kb/guides/`:**

| Topic                    | Guide                                  | When to Read                                            |
| ------------------------ | -------------------------------------- | ------------------------------------------------------- |
| Agent lifecycle          | `agent-lifecycle.md`                   | Agents not completing, dashboard wrong                  |
| Spawn                    | `spawn.md`                             | Spawn failures, wrong context, flags                    |
| Status/Dashboard         | `status-dashboard.md`                  | Wrong status, dashboard issues                          |
| Beads integration        | `beads-integration.md`                 | bd commands failing, issue tracking                     |
| Skill system             | `skill-system.md`                      | Skill not loading, wrong behavior                       |
| Daemon                   | `daemon.md`                            | Auto-spawn issues, triage workflow                      |
| Resilient infrastructure | `resilient-infrastructure-patterns.md` | Building/fixing critical infrastructure, backend independence |
| Architecture overview    | `architecture-overview.md`             | System diagram, directory structure, spawn backends     |
| Key packages             | `key-packages.md`                      | Package responsibilities and APIs                       |
| Event tracking           | `event-tracking.md`                    | Event types, enrichment fields, beads close hook        |

These guides synthesize 280+ investigations into authoritative references. Created Jan 4, 2026 after repeatedly re-investigating documented problems.

## Dashboard Server Management

**Always use `orch-dashboard` script** - handles orphan cleanup, stale sockets, and proper startup:

```bash
orch-dashboard start    # Start all services (kills orphans first)
orch-dashboard stop     # Stop all services
orch-dashboard restart  # Full restart with cleanup
orch-dashboard status   # Check service status
orch-dashboard logs     # View service logs (overmind echo)
```

**Service ports:** OpenCode (4096), orch serve (3348), Web UI (5188)

**Dashboard URL:** http://localhost:5188

**Why not raw overmind?** Direct `overmind start` can fail silently when orphan processes hold ports or stale sockets exist. The `orch-dashboard` script handles these edge cases.

**Production:** Future VPS deployment will use systemd. See `.kb/decisions/2026-01-10-dev-vs-prod-architecture.md`.

## Event Tracking

See `.kb/guides/event-tracking.md` for the full event type table and beads close hook reference.

## Gotchas

- **Window targeting**: Use workspace name, not window index
- **Model default**: Opus (Max subscription), not Gemini (pay-per-token)
- **SSE parsing**: Event type is inside JSON data, not `event:` prefix
- **Beads integration**: Shells out to `bd` CLI, doesn't use API directly
- **OpenCode auth**: Reads from `~/.local/share/opencode/auth.json`
- **Edit tool + tab indentation**: Svelte files in `web/src/` and Go files use tab indentation. The Read tool's line-number prefix uses a tab delimiter that collides with content tabs, causing Edit tool "String to replace not found" errors. See "Tab-Indented File Editing" section below.
- **OAuth tokens**: Never share refresh tokens between orch (`accounts.yaml`) and Claude CLI (keychain) — rotation invalidates the copy in the other system immediately
- **Account routing**: `accounts.yaml` `config_dir` field is REQUIRED for account routing to work — without it, `CLAUDE_CONFIG_DIR` is never injected
- **Non-Anthropic models**: GPT-4o/GPT-5.2-codex have 67-87% stall rates on protocol-heavy skills (architect, investigation). Use Anthropic models for these.
- **BEADS_DIR**: `BEADS_DIR=~/path/.beads bd close/update/list` enables cross-project beads operations from any directory
- **Skill sources**: Live in `orch-go/skills/src/`, deployed via `skillc deploy` to `~/.claude/skills/`
- **URL-to-markdown**: Use `scrape <url>` CLI, NOT the web-to-markdown MCP tools (`mcp__web-to-markdown__*`). `scrape` auto-selects the best extraction strategy (API, Playwright, screenshot+vision, HTML, PDF, YouTube) per URL type. The MCP is a dumb HTML fetcher with no strategy selection.
- **Playwright CLI**: Default for visual verification (1 bash call, ~1s). MCP only for interactive page exploration. On SSE-heavy pages, use `domcontentloaded` + `waitForSelector`, not `networkidle`.
- **OPSEC proxy**: When `opsec.sandbox: true` in project config, spawns require the local proxy on port 8199. Run `orch opsec start` or `orch opsec install` for persistent enforcement via LaunchAgent.

## Tab-Indented File Editing

**Problem:** The Read tool outputs `line_number→[TAB][content]`. When file content also starts with tabs (Svelte, Go, Makefile), adjacent tabs create ambiguity. Agents construct `old_string` with the wrong number of leading tabs, and Edit fails.

**Files affected in this project:** All `.svelte` files in `web/src/` use tab indentation. Go files use tabs per `gofmt` convention.

**Workarounds (in order of preference):**

1. **Include more context lines** in `old_string` — multi-line matches are less ambiguous than single-line matches with leading tabs
2. **Check exact whitespace first:** `head -20 file.svelte | cat -vet` — tabs display as `^I`, making them countable
3. **Use Write tool** for small files (<100 lines) — rewrite the entire file to avoid tab-matching issues
4. **Use sed via Bash** for surgical line edits: `sed -i '' '10s/old/new/' file.svelte`

**Prevention:** Before editing any tab-indented file, verify the indentation with `cat -vet` on the relevant lines. Do not rely solely on the Read tool output to count leading tabs.

## OpenCode Fork (We Own It)

**OpenCode is NOT a third-party dependency.** Dylan maintains a fork at `~/Documents/personal/opencode` (upstream: `sst/opencode`). This means:

- **Bugs in OpenCode → fix them in the fork**, not "report upstream"
- **Schema changes** in `*.sql.ts` require running `cd packages/opencode && bun drizzle-kit generate` and committing the migration
- **After code changes:** rebuild with `cd ~/Documents/personal/opencode/packages/opencode && bun run build`, then restart via `orch-dashboard restart`
- **Never install opencode-ai from npm** — it shadows the fork

The fork uses SQLite + Drizzle ORM (migrated from JSON file storage). Database at `~/.local/share/opencode/opencode.db`.

## Related

- **OpenCode fork:** `~/Documents/personal/opencode` (we maintain this)
- **Python orch-cli:** `~/Documents/personal/orch-cli` (fallback: `orch-py`)
- **Beads:** Issue tracking via `bd` CLI
- **Orchestrator skill:** `~/.claude/skills/meta/orchestrator/SKILL.md`
- **orch-knowledge:** *(merged into orch-go — skills in `skills/src/`, knowledge in `.kb/`)*
