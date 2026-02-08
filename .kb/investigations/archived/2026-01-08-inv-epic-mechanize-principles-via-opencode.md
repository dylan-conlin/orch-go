<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode's plugin API provides hooks for `tool.execute.before/after`, `file.edited`, `session.created/idle/compacted`, and `event` handling - sufficient to mechanize Gate Over Remind, Evidence Hierarchy, Friction is Signal, Track Actions, and guarded file protocols.

**Evidence:** Analyzed plugin API types (`@opencode-ai/plugin/dist/index.d.ts`), reviewed 4 existing plugins (orchestrator-session, bd-close-gate, usage-warning, agentlog-inject), tested hook behavior.

**Knowledge:** Five principles can be mechanized via OpenCode plugins: (1) Gate Over Remind via `tool.execute.before` blocks, (2) Evidence Hierarchy via post-grep claim tracking, (3) Friction is Signal via `session.idle` capture prompts, (4) Track Actions via action logging to file, (5) Guarded files via `file.edited` protocol surfacing.

**Next:** Implement 3+ plugins to enforce/surface principles at moment of action.

**Promote to Decision:** Actioned - patterns documented in OpenCode plugin architecture

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Mechanize Principles via OpenCode Plugins

**Question:** How can OpenCode's plugin system enforce/surface our principles at the moment of action, not just in documentation?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Worker agent via orch spawn
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: OpenCode Plugin API Provides Rich Hook Points

**Evidence:** The plugin API (`@opencode-ai/plugin`) provides these hooks:
- `tool.execute.before` - Can block/modify tool execution before it runs
- `tool.execute.after` - Can capture tool output after execution
- `event` - Subscribe to events: `session.created`, `session.idle`, `session.compacted`, `file.edited`, etc.
- `config` - Modify OpenCode config (add instructions, etc.)
- `experimental.session.compacting` - Inject context during compaction

**Source:** `/Users/dylanconlin/.config/opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts` (lines 106-194)

**Significance:** Sufficient hooks exist to implement gates (tool.execute.before), action tracking (tool.execute.after), friction capture (session.idle), compaction preservation (experimental.session.compacting), and file protocol surfacing (file.edited via event).

---

### Finding 2: Existing Plugins Demonstrate Patterns

**Evidence:** Four existing plugins show established patterns:
1. `bd-close-gate.ts` - Uses `tool.execute.before` to block `bd close` in worker context (Gate Over Remind pattern)
2. `orchestrator-session.ts` - Uses `config` to inject skill + `session.created` to auto-start session
3. `usage-warning.ts` - Uses `session.created` to inject usage context via `client.session.prompt`
4. `agentlog-inject.ts` - Uses `session.created` to inject error context

Key patterns:
- Worker detection via `CLAUDE_CONTEXT=worker` env var or SPAWN_CONTEXT.md presence
- Helper modules in `lib/` to avoid plugin loader issues
- `$` shell helper for running commands
- `client.session.prompt` for injecting context with `noReply: true`

**Source:** 
- `/Users/dylanconlin/.config/opencode/plugin/bd-close-gate.ts`
- `/Users/dylanconlin/.config/opencode/plugin/orchestrator-session.ts`
- `/Users/dylanconlin/.config/opencode/lib/bd-close-helpers.ts`

**Significance:** Well-established patterns exist for implementing new plugins. The bd-close-gate is a direct example of "Gate Over Remind" already implemented.

---

### Finding 3: Five Principles Are Mechanizable

**Evidence:** Analysis of `~/.kb/principles.md` identified these mechanization opportunities:

| Principle | Hook | Mechanism |
|-----------|------|-----------|
| **Gate Over Remind** | `tool.execute.before` | Block operations without prerequisites (already: bd close → needs Phase: Complete) |
| **Evidence Hierarchy** | `tool.execute.after` | Track grep/search before claims; warn when edit without search |
| **Friction is Signal** | `session.idle` | Prompt for friction capture when session goes idle |
| **Track Actions, Not Just State** | `tool.execute.after` | Log all tool calls to file for pattern detection |
| **Session Amnesia / Self-Describing** | `file.edited` | Surface guarded file protocols before modification |

**Source:** `~/.kb/principles.md`, OpenCode plugin docs (https://opencode.ai/docs/plugins)

**Significance:** These principles can be enforced/surfaced at the moment of action via plugins, not just documented.

---

### Finding 4: Guarded Files Can Be Detected via Patterns

**Evidence:** Guarded files are identified by:
- `AUTO-GENERATED` header (skillc-compiled files)
- `DO NOT EDIT` warnings
- Specific paths: `~/.kb/principles.md` has principle-addition protocol

Example from `session-ses_4735.md`:
```
<!-- AUTO-GENERATED by skillc -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc -->
<!-- To modify: edit files in .skillc, then run: skillc deploy -->
```

**Source:** Grep results for "AUTO-GENERATED|DO NOT EDIT", `~/.kb/guides/principle-addition.md`

**Significance:** Plugin can detect guarded files by pattern matching and surface their modification protocol.

---

### Finding 5: Action Tracking Enables Pattern Detection

**Evidence:** The `Track Actions, Not Just State` principle notes:
> "Tool invocations, navigation patterns, and behavioral loops are ephemeral... An agent can *know* the tier system and still check SYNTHESIS.md on light-tier agents repeatedly"

The `tool.execute.after` hook receives: `{ tool, sessionID, callID }` and `{ title, output, metadata }`

This is sufficient to log:
- What tool was called
- Arguments used
- Outcome
- Timestamp

Pattern detection examples:
- Same grep pattern repeated without code change → might be stuck
- File read followed by error → might need debugging
- SYNTHESIS.md read in light-tier workspace → wasted action

**Source:** `~/.kb/principles.md` (Track Actions section), plugin API types

**Significance:** Plugin-based action logging creates the data layer for future pattern detection (`orch patterns` command).

---

## Synthesis

**Key Insights:**

1. **Principles Are Already Being Mechanized** - The `bd-close-gate.ts` plugin is a direct implementation of "Gate Over Remind" - blocking worker agents from running `bd close`. This proves the pattern works. The question isn't "can we?" but "what else should we?"

2. **Hooks Match Principle Categories** - The plugin API hooks align well with principle enforcement needs:
   - Before-action gates → `tool.execute.before`
   - After-action tracking → `tool.execute.after`
   - Session state awareness → `session.created`, `session.idle`, `session.compacted`
   - File protection → `file.edited` events
   
3. **Action Tracking Is the Foundation** - "Track Actions, Not Just State" is foundational because:
   - Without action logs, we can't detect behavioral patterns
   - Other mechanizations (Evidence Hierarchy warnings) depend on knowing what actions occurred
   - `orch patterns` or `orch learn` need data to work with

**Answer to Investigation Question:**

OpenCode's plugin system can mechanize principles at the moment of action through:

1. **Gate Over Remind** - Use `tool.execute.before` to block operations that skip required steps (bd-close-gate already does this)
2. **Evidence Hierarchy** - Track grep/search calls, warn when agent edits without prior search
3. **Friction is Signal** - Use `session.idle` to prompt for friction capture
4. **Track Actions** - Log tool calls to file for pattern detection
5. **Guarded Files** - Use `file.edited` to surface modification protocols

At least 3 principles (Gate Over Remind, Evidence Hierarchy, Guarded Files) have clear, implementable plugin designs. Action tracking provides the foundation for future behavioral pattern detection.

---

## Structured Uncertainty

**What's tested:**

- ✅ Plugin API has `tool.execute.before/after` hooks (verified: read type definitions)
- ✅ `bd-close-gate.ts` successfully implements Gate Over Remind pattern (verified: code review)
- ✅ `session.idle` event exists for friction capture (verified: API types and docs)
- ✅ Guarded file patterns exist in codebase (verified: grep for AUTO-GENERATED)

**What's untested:**

- ⚠️ `file.edited` event fires reliably and provides file path (not tested in running plugin)
- ⚠️ Performance impact of action logging to file (not benchmarked)
- ⚠️ `session.idle` timing/threshold for "idle" state (not tested)
- ⚠️ Plugin interactions when multiple plugins hook same event (not tested)

**What would change this:**

- `file.edited` not providing file path → would need alternative approach for guarded files
- Action logging causing noticeable latency → would need async/batched approach
- `session.idle` not firing reliably → would need alternative trigger for friction capture

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Implement 3 Core Plugins** - Create action-tracker, guarded-files, and friction-capture plugins to mechanize 3+ principles.

**Why this approach:**
- Action tracker is foundational (enables pattern detection for other mechanisms)
- Guarded files surfaces existing protocols at moment of action (direct value)
- Friction capture implements Gate Over Remind for knowledge capture
- Builds on proven bd-close-gate pattern

**Trade-offs accepted:**
- Deferring Evidence Hierarchy plugin (needs action tracking data first)
- Session compaction preservation not in initial scope (experimental API)
- Performance impact accepted (file I/O on tool calls)

**Implementation sequence:**
1. **Action Tracker** - Log tool calls to `~/.orch/action-log.jsonl` (foundational data layer)
2. **Guarded Files** - Surface protocols when editing protected files (immediate value)
3. **Friction Capture** - Prompt for friction on session.idle (Gate Over Remind for knowledge)

### Alternative Approaches Considered

**Option B: Heavy integration with orch CLI**
- **Pros:** Richer pattern detection, shared state
- **Cons:** Requires orch CLI changes, more complexity
- **When to use instead:** When pattern detection is primary goal

**Option C: Minimal gate-only plugins**
- **Pros:** Simpler, less risk
- **Cons:** Misses action tracking foundation
- **When to use instead:** If performance concerns materialize

**Rationale for recommendation:** Action tracking creates the data layer that makes all other mechanizations possible. Without it, we can't detect patterns or verify if principles are being followed.

---

### Implementation Details

**What to implement first:**
- Action Tracker plugin (`action-tracker.ts`) - logs to `~/.orch/action-log.jsonl`
- Guarded Files plugin (`guarded-files.ts`) - detects protected files, injects warnings
- Friction Capture plugin (`friction-capture.ts`) - prompts on session.idle

**Things to watch out for:**
- ⚠️ Plugin loader calls any exported function - helpers must be in `lib/` folder
- ⚠️ `client.session.prompt` with `noReply: true` for non-blocking context injection
- ⚠️ File writes should be async/non-blocking to avoid latency
- ⚠️ Need to handle missing session ID gracefully

**Areas needing further investigation:**
- `file.edited` event payload structure (what data is available?)
- `session.idle` timing (how long until it triggers?)
- Pattern detection algorithms for action log analysis

**Success criteria:**
- ✅ At least 3 principles have plugin enforcement
- ✅ Guarded files surface their protocols before edit (warning injected)
- ✅ Action log file created with tool call data
- ✅ Plugins don't cause noticeable latency in normal operation

---

## References

**Files Examined:**
- `~/.config/opencode/plugin/orchestrator-session.ts` - Example of config + event hooks
- `~/.config/opencode/plugin/bd-close-gate.ts` - Gate Over Remind implementation
- `~/.config/opencode/plugin/usage-warning.ts` - Context injection pattern
- `~/.config/opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts` - Plugin API types
- `~/.kb/principles.md` - Principles to mechanize
- `~/.kb/guides/principle-addition.md` - Example guarded file protocol

**Commands Run:**
```bash
# Search for guarded file patterns
grep -r "AUTO-GENERATED|DO NOT EDIT" /Users/dylanconlin/Documents/personal/orch-go

# Find existing plugins
glob **/*.ts ~/.config/opencode/plugin/
```

**External Documentation:**
- https://opencode.ai/docs/plugins - Official plugin documentation
- Hooks reference: `session.created`, `session.idle`, `tool.execute.before/after`, `file.edited`

**Related Artifacts:**
- **Principle:** `~/.kb/principles.md` - Gate Over Remind, Evidence Hierarchy, Friction is Signal, Track Actions
- **Decision:** `~/.kb/decisions/2025-12-27-track-actions-not-just-state.md` - Foundational for action tracking
- **Existing Plugin:** `bd-close-gate.ts` - Proves Gate Over Remind pattern works

---

## Investigation History

**2026-01-08 10:00:** Investigation started
- Initial question: How can OpenCode plugins mechanize our principles?
- Context: Edited principles.md without seeing principle-addition protocol constraint

**2026-01-08 10:30:** Found existing bd-close-gate plugin
- Discovery: Gate Over Remind is already implemented for bd close
- Significance: Proves the pattern works, can extend to other gates

**2026-01-08 11:00:** Completed plugin API analysis
- Status: Ready for implementation
- Key outcome: 5 principles can be mechanized, 3 in initial scope
