# Investigation: Cross-Project Interface Agreement Coverage Gaps

**Date:** 2026-02-28
**Status:** Complete
**Beads:** orch-go-8t6x
**References:** `.kb/models/drift-taxonomy/model.md`

## Summary

Inventoried ~25 cross-project interfaces across the orch ecosystem. Only 5 have executable agreements, leaving 20 interfaces vulnerable to silent-drop or stale-shadow drift. The highest-risk gaps are: beads JSON output parsing (3 duplicate Issue structs, no contract), OpenCode API type alignment (14+ struct types with no version pinning), and spawn context → agent consumption (complex template with no consumer-side validation).

---

## Existing Agreements (5)

| ID | Interface | Failure Mode | Severity |
|----|-----------|-------------|----------|
| `kb-reflect-output-passthrough` | kb reflect JSON → orch-go daemon ReflectSuggestions struct | silent-drop | error |
| `spawn-template-config-wiring` | Spawn templates → no hardcoded paths | unchecked-assumption | error |
| `skill-action-compliance` | Skill instructions → available agent tools | stale-shadow | warning |
| `probe-routing-wiring` | kb reflect probe data → spawn routing | unchecked-assumption | warning |
| `verification-model-accuracy` | pkg/verify/ code → verification model docs | stale-shadow | warning |

---

## Complete Interface Inventory

### Category 1: orch-go ← beads (bd CLI / RPC)

| # | Interface | Producer | Consumer | Format | Failure Mode | Has Agreement |
|---|-----------|----------|----------|--------|-------------|---------------|
| 1 | `bd ready --json` output | beads CLI | `pkg/daemon/issue_adapter.go`, `pkg/focus/guidance.go` | JSON array of Issue objects | silent-drop | **NO** |
| 2 | `bd list --json` output | beads CLI | `pkg/daemon/issue_adapter.go:143` | JSON array of Issue objects | silent-drop | **NO** |
| 3 | `bd show <id> --json` output | beads CLI | `pkg/daemon/issue_adapter.go:249,508` | JSON Issue object (with dependencies) | silent-drop | **NO** |
| 4 | `bd comments <id> --json` output | beads CLI | `pkg/verify/beads_api.go:89` | JSON array of Comment objects | silent-drop | **NO** |
| 5 | `bd config get issue_prefix` output | beads CLI | `cmd/orch/status_cmd.go:1136` | Plain text | silent change | **NO** |
| 6 | Beads RPC protocol (socket) | beads daemon | `pkg/beads/client.go` (Request/Response/Issue types) | JSON-RPC over Unix socket | silent-drop | **NO** |

**Risk:** HIGH. Three separate `Issue` structs exist (`pkg/beads/types.go:141`, `pkg/daemon/issue_queue.go:9`, `pkg/verify/beads_api.go:31`) that each parse beads output with different field subsets. If beads adds/renames a field (e.g., `assignee`, `epic_id`), some consumers get it and others silently drop it. The RPC protocol (`Request`/`Response` types in `pkg/beads/types.go`) is an even tighter coupling — version mismatches between bd daemon and orch-go client can cause silent field drops. The `ClientVersion` field exists but `Compatible` is not enforced as a hard gate.

### Category 2: orch-go ← kb-cli

| # | Interface | Producer | Consumer | Format | Failure Mode | Has Agreement |
|---|-----------|----------|----------|--------|-------------|---------------|
| 7 | `kb reflect --format json` output | kb-cli | `pkg/daemon/reflect.go` ReflectSuggestions struct | JSON object with typed arrays | silent-drop | **YES** (kb-reflect-output-passthrough) |
| 8 | `kb projects list --json` output | kb-cli | `pkg/spawn/kbcontext.go:478`, `pkg/daemon/project_resolution.go:43`, `cmd/orch/serve_agents_cache.go:440` | JSON array of {name, path} | silent-drop | **NO** |
| 9 | `kb quick list --json` output | kb-cli | `pkg/daemon/knowledge_health.go:63`, `cmd/orch/knowledge_maintenance.go:161` | JSON array of QuickEntry objects | silent-drop | **NO** |
| 10 | `kb agreements check --json` output | kb-cli | `cmd/orch/kb.go:874` | JSON | silent-drop | **NO** |
| 11 | `kb promote`, `kb quick obsolete` exit codes | kb-cli | `cmd/orch/knowledge_maintenance.go:185,193` | Exit code + stderr | silent failure | **NO** |

**Risk:** MEDIUM-HIGH. The `kb projects list --json` interface is consumed in 3+ places, each with its own `kbProjectEntry` struct (duplicated in `pkg/spawn/kbcontext.go:468` and `pkg/daemon/project_resolution.go:29`). The `kb quick list --json` interface has two different `quickEntry`/`QuickEntry` structs with different field sets (`pkg/daemon/knowledge_health.go:55` has 3 fields, `cmd/orch/knowledge_maintenance.go:23` has 5 fields, `pkg/tree/parser.go:584` has 8 fields). New fields from kb-cli will be silently dropped by the minimal structs.

### Category 3: orch-go ← OpenCode (HTTP API)

| # | Interface | Producer | Consumer | Format | Failure Mode | Has Agreement |
|---|-----------|----------|----------|--------|-------------|---------------|
| 12 | Session list/get API (`/session`) | OpenCode server | `pkg/opencode/client.go` Session struct | JSON | silent-drop | **NO** |
| 13 | Message list API (`/session/{id}/message`) | OpenCode server | `pkg/opencode/client.go` Message/MessagePart/ToolState structs | JSON | silent-drop | **NO** |
| 14 | SSE event stream (`/event`) | OpenCode server | `pkg/opencode/sse.go` (3 different parse attempts) | SSE + JSON | silent-drop | **NO** |
| 15 | Session status (`/session/status`) | OpenCode server | `pkg/opencode/types.go` SessionStatusInfo struct | JSON | silent-drop | **NO** |

**Risk:** HIGH. OpenCode is a maintained fork, so the producer and consumer are both under Dylan's control — but they're separate codebases with separate type systems. The SSE parser in `pkg/opencode/sse.go:109-143` tries 3 different JSON parse formats sequentially (old format, new format, generic map), which means format changes are absorbed silently rather than failing visibly. The `Message`/`MessagePart`/`ToolState` types mirror OpenCode's Zod schemas but have no compile-time or runtime validation that they match. Schema additions in OpenCode (e.g., new message part types, new tool state fields) are silently dropped.

### Category 4: orch-go → spawned agents (produced interfaces)

| # | Interface | Producer | Consumer | Format | Failure Mode | Has Agreement |
|---|-----------|----------|----------|--------|-------------|---------------|
| 16 | SPAWN_CONTEXT.md template | `pkg/spawn/context.go` | Spawned agent (Claude/OpenCode) | Markdown with structured sections | stale-shadow | **PARTIAL** (spawn-template-config-wiring covers hardcoded paths only) |
| 17 | AGENT_MANIFEST.json | `pkg/spawn/session.go` AgentManifest struct | `orch complete`, `orch status`, `orch clean` | JSON | internal (same binary) | **NO** |
| 18 | Workspace dotfiles (`.beads_id`, `.session_id`, `.spawn_mode`, `.spawn_time`, `.tier`) | `pkg/spawn/atomic.go` | `pkg/spawn/session.go`, `cmd/orch/shared.go`, `cmd/orch/lifecycle_adapters.go` | Plain text files | internal (same binary) | N/A |
| 19 | Skill SKILL.md frontmatter | orch-knowledge (`skills/src/`) | `pkg/skills/loader.go` SkillMetadata struct | YAML frontmatter in Markdown | stale-shadow | **PARTIAL** (skill-action-compliance covers command references only) |

**Risk:** MEDIUM. SPAWN_CONTEXT.md is the most complex produced interface — a ~2000-line template with structured sections (CONFIG RESOLUTION, PRIOR KNOWLEDGE, HOTSPOT AREA WARNING, SKILL GUIDANCE, etc.). Agents parse these sections heuristically. If section headers change, agents may miss critical context. Skill frontmatter has 9 fields (`name`, `skill-type`, `audience`, `spawnable`, `composable`, `category`, `description`, `dependencies`) — if orch-knowledge adds a new field, `pkg/skills/loader.go` silently ignores it.

### Category 5: orch-go ← filesystem conventions

| # | Interface | Producer | Consumer | Format | Failure Mode | Has Agreement |
|---|-----------|----------|----------|--------|-------------|---------------|
| 20 | `.orch/config.yaml` schema | User / `orch init` | `pkg/config/config.go` Config struct | YAML | unchecked-assumption | **NO** |
| 21 | `~/.orch/accounts.yaml` schema | User / `orch account` | `pkg/account/account.go` | YAML | unchecked-assumption | **NO** |
| 22 | `~/.orch/events.jsonl` format | `pkg/events/logger.go` | `cmd/orch/serve_agents_events.go`, dashboard, `scripts/analyze_*` | JSONL | internal (same binary) | N/A |
| 23 | `~/.orch/focus.json` schema | `pkg/focus/focus.go` | `cmd/orch/focus.go`, dashboard | JSON | internal (same binary) | N/A |
| 24 | `~/.claude/settings.json` | Claude Code | `pkg/spawn/opencode_mcp.go` (reads MCP config) | JSON | silent-drop | **NO** |
| 25 | OpenCode auth file (`~/.local/share/opencode/auth.json`) | OpenCode | `pkg/account/account.go:223` | JSON | silent-drop | **NO** |

**Risk:** MEDIUM. `.orch/config.yaml` and `accounts.yaml` are user-facing and relatively stable. `settings.json` and `auth.json` are more concerning — they're produced by external tools (Claude Code, OpenCode) whose formats orch-go parses without any schema validation.

### Category 6: orch serve → web dashboard

| # | Interface | Producer | Consumer | Format | Failure Mode | Has Agreement |
|---|-----------|----------|----------|--------|-------------|---------------|
| 26 | REST API (45+ endpoints) | `cmd/orch/serve*.go` | `web/src/` (Svelte dashboard) | JSON over HTTP | stale-shadow | **NO** |

**Risk:** LOW-MEDIUM. Both sides are in the same repo, so `go build` + `bun build` catch most issues. However, the API shapes are implicitly defined — there are no shared type definitions or OpenAPI specs. Adding a field server-side is invisible to the frontend unless someone updates the TypeScript types.

### Category 7: coaching/monitoring

| # | Interface | Producer | Consumer | Format | Failure Mode | Has Agreement |
|---|-----------|----------|----------|--------|-------------|---------------|
| 27 | Coaching metrics JSONL | OpenCode coaching plugin | `pkg/coaching/metrics.go` Metric struct | JSONL | silent-drop | **NO** |

**Risk:** MEDIUM. The coaching plugin is in the OpenCode fork. If it adds new `metric_type` values or changes the `details` schema, orch-go's `Metric` struct (which uses `map[string]interface{}` for `Details`) will absorb changes silently — but any code that inspects specific detail keys will fail silently.

---

## Highest-Risk Uncovered Interfaces (Prioritized)

### Priority 1: Beads JSON Output Contract (Interfaces 1-6)

**Why highest risk:**
- 3 duplicate `Issue` structs parsing the same source with different field subsets
- RPC protocol has version field but no enforcement gate
- beads is actively developed — field additions are ongoing
- Failure mode: silent-drop (new fields lost without warning)

**Recommended agreement:** `beads-issue-json-passthrough`
- Check: Compare `bd show --json` output keys against all 3 Issue struct json tags
- Severity: error
- Could also create `beads-rpc-version-compat` for the RPC client

### Priority 2: OpenCode API Type Alignment (Interfaces 12-15)

**Why high risk:**
- OpenCode fork is actively modified (schema changes in SQL migrations)
- 14+ Go structs mirror Zod schemas with no automated validation
- SSE parser silently absorbs format changes (3 fallback parse attempts)
- Failure mode: silent-drop

**Recommended agreement:** `opencode-session-schema-passthrough`
- Check: Compare OpenCode's Zod type definitions against orch-go's Go struct json tags
- Severity: error

### Priority 3: kb quick/projects JSON Passthrough (Interfaces 8-9)

**Why medium-high risk:**
- 2-3 duplicate structs per interface with different field coverage
- `quickEntry` in `knowledge_health.go` has only 3 of 8+ fields
- Failure mode: silent-drop

**Recommended agreement:** `kb-quick-list-passthrough`
- Check: Compare `kb quick list --json` output keys against consumer structs
- Severity: warning

### Priority 4: Spawn Context Section Contract (Interface 16)

**Why medium risk:**
- ~2000-line template with structured sections consumed by AI agents
- Agents parse sections heuristically — no formal contract
- Failure mode: stale-shadow (section changes → agents miss context)

**Recommended agreement:** `spawn-context-section-headers`
- Check: Verify expected section headers exist in generated context
- Severity: warning

### Priority 5: Skill Frontmatter Schema (Interface 19)

**Why medium risk:**
- 9 YAML fields parsed by `pkg/skills/loader.go`
- orch-knowledge may add fields (e.g., `trigger-conditions`, `timeout`)
- Failure mode: silent-drop (new fields ignored)

**Recommended agreement:** `skill-frontmatter-schema-passthrough`
- Check: Compare skill frontmatter keys in orch-knowledge against SkillMetadata struct yaml tags
- Severity: warning

---

## Structural Observations

### The Duplicate Struct Pattern

The most dangerous pattern found is **multiple Go structs parsing the same external JSON**. This occurs for:
- `Issue`: 3 structs (`pkg/beads/types.go`, `pkg/daemon/issue_queue.go`, `pkg/verify/beads_api.go`)
- `QuickEntry`: 3 structs (`pkg/tree/parser.go`, `pkg/daemon/knowledge_health.go`, `cmd/orch/knowledge_maintenance.go`)
- `kbProjectEntry`: 2 structs (`pkg/spawn/kbcontext.go`, `pkg/daemon/project_resolution.go`)

Each struct independently decides which fields to include. When the source adds a field, some consumers get it and others don't — with no error. This is the classic **silent-drop** pattern from the drift taxonomy.

**Recommendation:** Consolidate to single canonical structs in shared packages (e.g., all beads Issue parsing should use `pkg/beads/types.Issue`). This converts cross-project drift into intra-project drift, which the compiler can catch.

### SSE Triple-Parse Anti-Pattern

`pkg/opencode/sse.go:109-143` attempts 3 different JSON parse formats for SSE events. While resilient, this means format changes are silently absorbed — the old format attempt succeeds, and new fields in the new format are lost. This is drift-by-design.

**Recommendation:** Log when old-format parsing succeeds (indicates potential schema migration needed).

---

## Recommendations Summary

| Priority | Agreement | Interfaces | Risk Mitigated |
|----------|-----------|------------|----------------|
| P1 | `beads-issue-json-passthrough` | 1-6 | 3 duplicate structs, RPC protocol |
| P2 | `opencode-session-schema-passthrough` | 12-15 | 14+ structs mirroring Zod schemas |
| P3 | `kb-quick-list-passthrough` | 8-9 | 2-3 duplicate structs per interface |
| P4 | `spawn-context-section-headers` | 16 | Template section changes |
| P5 | `skill-frontmatter-schema-passthrough` | 19 | Skill metadata additions |

Additionally, **consolidating duplicate structs** (P1 architectural fix) would reduce the interface surface from ~25 to ~18 and convert cross-project drift into intra-project drift.
