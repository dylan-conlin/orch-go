<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** claude-sneakpeek v1.6.9 successfully enables native swarm mode (TeammateTool, Task delegate mode, teammate mailboxes) by patching the `tengu_brass_pebble` statsig gate in cli.js from `return!1` to `return!0`.

**Evidence:** Created `swarm-test` variant at `~/.claude-sneakpeek/swarm-test/` with `swarmModeEnabled: true`. Verified cli.js has 0 occurrences of `tengu_brass_pebble` (patched out), but retains TeammateTool (34 occurrences), launchSwarm (7), teammate_mailbox (1), and all operations (spawnTeam, approvePlan, rejectPlan, requestShutdown).

**Knowledge:** Native multi-agent is mature with 12+ operations (spawnTeam, write, broadcast, cleanup, requestShutdown, approveShutdown, rejectShutdown, approvePlan, rejectPlan, discoverTeams, requestJoin, approveJoin, rejectJoin). Two spawn backends: in-process (headless) and pane (tmux/iTerm2). Plan mode required for teammates unless approved by leader.

**Next:** To launch swarm variant: `~/.local/bin/swarm-test` (requires OAuth auth for mirror provider). Document launch instructions and test interactively.

**Promote to Decision:** recommend-no - Tactical setup, not architectural

---

# Investigation: Set Up Claude Sneakpeek for Native Swarm Mode

**Question:** How to set up claude-sneakpeek to enable native swarm mode (TeammateTool, Task primitives) and validate the multi-agent API?

**Started:** 2026-01-24
**Updated:** 2026-01-24
**Owner:** og-inv-set-up-claude-24jan-ed3a
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Swarm mode patch mechanism

**Evidence:** The file `src/core/variant-builder/swarm-mode-patch.ts` (lines 20-35, 79-107) implements the patch:
- Regex matches: `function\s+([a-zA-Z_$][\w$]*)\(\)\{if\([\w$]+\(process\.env\.CLAUDE_CODE_AGENT_SWARMS\)\)return!1;return\s*[\w$]+\("tengu_brass_pebble",!1\)\}`
- Replaces with: `function ${fnName}(){return!0}`

**Source:** `~/Documents/personal/claude-sneakpeek/src/core/variant-builder/swarm-mode-patch.ts`

**Significance:** One patch enables ALL native multi-agent features: TeammateTool, delegate mode, swarm spawning via ExitPlanMode, teammate mailbox/messaging, and task ownership.

---

### Finding 2: Swarm mode is enabled by default in v1.6.9

**Evidence:** From `src/core/constants.ts`:
```typescript
export const NATIVE_MULTIAGENT_MIN_VERSION = '2.1.16';
export const NATIVE_MULTIAGENT_SUPPORTED = true;
export const TEAM_MODE_SUPPORTED = false;  // Legacy team mode deprecated
```

SwarmModeStep is included in VariantBuilder when `NATIVE_MULTIAGENT_SUPPORTED` is true (line 64).

**Source:** `~/Documents/personal/claude-sneakpeek/src/core/constants.ts:11-19`, `src/core/variant-builder/VariantBuilder.ts:64`

**Significance:** No special flags needed - swarm mode is ON by default. Legacy team mode (Task* tools via cli.js patching) is deprecated.

---

### Finding 3: TeammateTool provides 12+ operations

**Evidence:** From cli.js extraction, the TeammateTool schema includes:

| Operation | Purpose |
|-----------|---------|
| `spawnTeam` | Create a new team |
| `write` | Send message to specific teammate |
| `broadcast` | Send message to ALL teammates |
| `cleanup` | Clean up team directories and worktrees |
| `requestShutdown` | Ask a teammate to shut down gracefully |
| `approveShutdown` | Accept shutdown request and exit |
| `rejectShutdown` | Decline shutdown request with reason |
| `approvePlan` | Approve a teammate's plan (leader only) |
| `rejectPlan` | Reject a teammate's plan with feedback |
| `discoverTeams` | List available teams to join |
| `requestJoin` | Request to join an existing team |
| `approveJoin` | Approve a join request (leader only) |
| `rejectJoin` | Reject a join request |

**Source:** cli.js grep extraction, `src/core/variant-builder/` TypeScript files

**Significance:** The API is mature with comprehensive team coordination primitives. More sophisticated than orch-go's external orchestration approach.

---

### Finding 4: Two spawn backends available

**Evidence:** From cli.js code:
- **In-process backend** (`in-process`) - Runs teammates in the same process, used for non-interactive sessions
- **Pane backend** (`tmux` | `iterm2`) - Spawns teammates in terminal panes, provides visibility

Configuration via `--teammate-mode` flag or `teammateMode` setting (options: `auto`, `tmux`, `in-process`).

**Source:** cli.js, `src/tui/state/types.ts`, research docs

**Significance:** Flexible deployment - headless for automation, pane-based for visibility during development.

---

### Finding 5: Variant successfully created and patch verified

**Evidence:**
```bash
npm run dev -- create --provider mirror --name swarm-test --yes --no-shell-env
# Output: ✓ Created: swarm-test
#         • Swarm mode enabled successfully
```

Verification:
```bash
grep -o "tengu_brass_pebble" ~/.claude-sneakpeek/swarm-test/npm/node_modules/@anthropic-ai/claude-code/cli.js
# Output: (empty - patched out)

grep -o "TeammateTool" ~/.claude-sneakpeek/swarm-test/npm/node_modules/@anthropic-ai/claude-code/cli.js | wc -l
# Output: 34
```

**Source:** CLI output, grep verification

**Significance:** Patch is applied correctly, all swarm features present in cli.js.

---

## Synthesis

**Key Insights:**

1. **Single gate, all features** - Patching one function (`tengu_brass_pebble` gate) enables the complete native multi-agent stack. This is elegant but fragile to Claude Code version changes.

2. **Mature coordination primitives** - The API includes join/leave workflow, plan approval, shutdown coordination, broadcast messaging. More than basic spawn/communicate.

3. **Flexible backends** - In-process for headless automation, tmux panes for visible development. The `auto` mode selects appropriately.

**Answer to Investigation Question:**

To set up claude-sneakpeek for native swarm mode:

1. **Clone and install:**
   ```bash
   git clone https://github.com/mikekelly/claude-sneakpeek ~/Documents/personal/claude-sneakpeek
   cd ~/Documents/personal/claude-sneakpeek
   npm install
   ```

2. **Create a variant** (swarm mode enabled by default):
   ```bash
   npm run dev -- create --provider mirror --name swarm-test --yes --no-shell-env
   # Or with API key for non-OAuth providers:
   npm run dev -- create --provider zai --name zai-swarm --api-key YOUR_KEY
   ```

3. **Launch:**
   ```bash
   ~/.local/bin/swarm-test
   # Or for Z.ai variant:
   ~/.local/bin/zai-swarm
   ```

4. **Test TeammateTool** (in session):
   - Create team: `{ "operation": "spawnTeam", "team_name": "test-team", "description": "Test swarm" }`
   - Spawn teammates via Task tool with `team_name` parameter
   - Use write/broadcast for messaging

---

## Structured Uncertainty

**What's tested:**

- ✅ npm install succeeded (verified: `npm install` output)
- ✅ Variant creation works (verified: `npm run dev -- create` succeeded)
- ✅ Patch applied correctly (verified: `tengu_brass_pebble` removed, TeammateTool present)
- ✅ Variant metadata correct (verified: `variant.json` shows `swarmModeEnabled: true`)

**What's untested:**

- ⚠️ Actually spawning a teammate via TeammateTool (requires interactive session with auth)
- ⚠️ Plan approval workflow (requires running team lead + teammate)
- ⚠️ In-process vs tmux backend behavior differences (not tested)
- ⚠️ Performance of native swarm vs orch-go's external orchestration (not benchmarked)

**What would change this:**

- Finding would change if Anthropic enables swarm mode publicly (patch becomes unnecessary)
- Finding would change if CLI function signatures change in future versions (patch breaks)
- Finding would change if native TeammateTool proves unstable in practice

---

## Implementation Recommendations

### Recommended Approach ⭐

**Use for experimental/development multi-agent work** - The native API is more integrated than external orchestration but requires patch maintenance.

**Why this approach:**
- Single binary with all coordination built-in
- No external state management needed
- Native tmux integration for visibility
- Mature API with 12+ operations

**Trade-offs accepted:**
- Fragile to Claude Code updates (function signatures may change)
- Requires OAuth auth for mirror variant
- Less control than orch-go's external orchestration

**Implementation sequence:**
1. Use `~/.local/bin/swarm-test` for interactive testing
2. For automated use, create API-key variant (zai, minimax, openrouter)
3. Monitor for Claude Code version updates that may break patch

### Alternative Approaches Considered

**Option B: Continue with orch-go external orchestration**
- **Pros:** Provider-agnostic, survives Claude Code updates, richer beads integration
- **Cons:** More complex, external state management, no native TeammateTool
- **When to use instead:** Production workloads, cross-project orchestration

---

## References

**Files Examined:**
- `~/Documents/personal/claude-sneakpeek/src/core/variant-builder/swarm-mode-patch.ts` - Patch implementation
- `~/Documents/personal/claude-sneakpeek/src/core/constants.ts` - Feature flags
- `~/Documents/personal/claude-sneakpeek/docs/research/native-multiagent-gates.md` - Research documentation
- `~/.claude-sneakpeek/swarm-test/variant.json` - Created variant metadata

**Commands Run:**
```bash
# Install dependencies
cd ~/Documents/personal/claude-sneakpeek && npm install

# Create variant
npm run dev -- create --provider mirror --name swarm-test --yes --no-shell-env

# Verify patch
grep -o "tengu_brass_pebble" ~/.claude-sneakpeek/swarm-test/npm/node_modules/@anthropic-ai/claude-code/cli.js | wc -l
grep -o "TeammateTool" ~/.claude-sneakpeek/swarm-test/npm/node_modules/@anthropic-ai/claude-code/cli.js | wc -l
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-24-inv-claude-sneakpeek-comparison.md` - Prior comparison of approaches

---

## Investigation History

**2026-01-24 16:41:** Investigation started
- Initial question: Set up claude-sneakpeek for native swarm mode validation
- Context: Need hands-on validation of Anthropic's native multi-agent API

**2026-01-24 16:43:** Explored source code
- Found swarm-mode-patch.ts, team-mode-patch.ts
- Understood NATIVE_MULTIAGENT_SUPPORTED = true, TEAM_MODE_SUPPORTED = false

**2026-01-24 16:46:** Created variant
- `npm run dev -- create --provider mirror --name swarm-test --yes --no-shell-env`
- Verified: swarmModeEnabled: true in variant.json

**2026-01-24 16:50:** Verified patch
- tengu_brass_pebble removed (0 occurrences)
- TeammateTool present (34 occurrences)
- All operations extracted from cli.js

**2026-01-24 16:55:** Investigation completed
- Status: Complete
- Key outcome: Native swarm mode successfully enabled via claude-sneakpeek v1.6.9
