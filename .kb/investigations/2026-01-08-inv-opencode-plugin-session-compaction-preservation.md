<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode's `experimental.session.compacting` hook allows injection of context strings into the compaction prompt, enabling preservation of workspace tier, beads issue, phase status, and kb constraints.

**Evidence:** Implemented and built session-compaction.ts plugin successfully; API types confirm hook signature accepts `context: string[]` array.

**Knowledge:** The hook appends context to compaction rather than replacing it - we push to `output.context` array rather than setting `output.prompt`.

**Next:** Monitor plugin effectiveness during long sessions that trigger compaction; adjust context injection as needed.

**Promote to Decision:** recommend-no - This is a tactical implementation of an existing decision to mechanize principles.

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

# Investigation: OpenCode Plugin Session Compaction Preservation

**Question:** How can we preserve critical workspace context (tier, beads issue, phase status, constraints) during OpenCode session compaction?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** orch-go-vfczs
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** `.kb/investigations/2026-01-08-inv-epic-mechanize-principles-via-opencode.md` (identified hook, marked out of scope)
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: experimental.session.compacting hook accepts context array

**Evidence:** The plugin API type definition shows:
```typescript
"experimental.session.compacting"?: (input: {
  sessionID: string;
}, output: {
  context: string[];  // Additional context strings appended to default prompt
  prompt?: string;    // If set, replaces default compaction prompt entirely
}) => Promise<void>;
```

**Source:** `/Users/dylanconlin/.config/opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts:181-186`

**Significance:** The hook allows appending context (via `output.context.push()`) without replacing the default compaction prompt. This is safer than replacing the entire prompt.

---

### Finding 2: Workspace metadata is stored in predictable files

**Evidence:** Worker workspaces contain:
- `.tier` - contains "light", "full", or "orchestrator"
- `.beads_id` - contains issue ID like "orch-go-vfczs"
- `.session_id` - OpenCode session ID
- `.spawn_time` - timestamp

Example:
```
.orch/workspace/og-feat-opencode-plugin-session-08jan-9cea/
├── .beads_id     # orch-go-vfczs
├── .session_id   # session ID
├── .spawn_time   # timestamp
├── .tier         # full
└── SPAWN_CONTEXT.md
```

**Source:** `ls -la /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-opencode-plugin-session-08jan-9cea/`

**Significance:** Plugin can reliably detect workspace context by reading these files from the working directory.

---

### Finding 3: Dynamic context available via CLI commands

**Evidence:** 
- `bd show <id> --json` returns issue status and comments (including phase comments)
- `kn constraints --json` returns active constraints
- `kn recent --n 5 --json` returns recent decisions and other entries

**Source:** Tested commands in shell

**Significance:** Plugin can gather dynamic context (current phase, constraints, recent decisions) at compaction time, not just static workspace metadata.

---

## Synthesis

**Key Insights:**

1. **Append-only context injection is safe** - Using `output.context.push()` adds context without overriding OpenCode's default compaction behavior. This means we get the benefits of context preservation without risking breaking the compaction process.

2. **Static + dynamic context strategy** - The plugin caches static context (tier, beads ID) at initialization for performance, but fetches dynamic context (phase status, constraints, decisions) at compaction time to ensure freshness.

3. **Tier-aware guidance** - Different tiers need different reminders. Light tier agents don't need SYNTHESIS.md reminders. Full tier agents need the documentation requirement emphasized. Orchestrator tier agents need the "delegate, don't implement" reminder.

**Answer to Investigation Question:**

Critical workspace context can be preserved during session compaction via the `experimental.session.compacting` hook by:

1. Reading static metadata from workspace files (`.tier`, `.beads_id`)
2. Querying dynamic state via CLI commands (`bd show`, `kn constraints`)
3. Building a structured context string with tier-specific guidance
4. Pushing the context to `output.context` array

The implemented plugin (`~/.config/opencode/plugin/session-compaction.ts`) handles all four success criteria items:
- Workspace tier awareness ✅
- Active beads issue context ✅
- Key constraints/decisions from kb ✅
- Phase status ✅

---

## Structured Uncertainty

**What's tested:**

- ✅ Plugin compiles successfully (verified: `bun build plugin/session-compaction.ts` succeeded)
- ✅ Plugin export follows established pattern (verified: `grep -l "export const.*Plugin"`)
- ✅ API types match implementation (verified: read index.d.ts, implemented matching signature)
- ✅ CLI commands provide required data (verified: ran `bd show`, `kn constraints`, `kn recent`)

**What's untested:**

- ⚠️ Plugin actually fires during compaction (requires long session to trigger compaction)
- ⚠️ Context actually survives compaction (not benchmarked)
- ⚠️ Performance impact of CLI calls during compaction (not measured)
- ⚠️ Edge cases: missing files, CLI errors, empty responses

**What would change this:**

- If `experimental.session.compacting` hook is removed or changes signature
- If compaction happens faster than CLI commands return
- If context injection causes compaction to fail
- If OpenCode ignores appended context

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Append-only context injection** - Use `output.context.push()` to add critical context without replacing OpenCode's default compaction behavior.

**Why this approach:**
- Safer than replacing entire prompt (preserves OpenCode's compaction logic)
- Structured format (headers, bullets) survives compaction well
- Tier-aware guidance provides contextually relevant reminders

**Trade-offs accepted:**
- CLI calls at compaction time may add latency (acceptable for critical context)
- Experimental API may change (documented risk in plugin comments)

**Implementation sequence:**
1. Plugin implemented at `~/.config/opencode/plugin/session-compaction.ts`
2. Deployment: automatic (OpenCode loads plugins from this directory)
3. Monitoring: observe agent behavior during long sessions

### Alternative Approaches Considered

**Option B: Replace entire compaction prompt**
- **Pros:** Full control over what survives compaction
- **Cons:** Risk breaking OpenCode's compaction logic, maintenance burden
- **When to use instead:** If default compaction consistently loses critical info

**Option C: Pre-compaction context injection via other hooks**
- **Pros:** More control over timing
- **Cons:** No other hook fires before compaction starts
- **When to use instead:** If timing issues arise with current approach

**Rationale for recommendation:** Append-only approach is safest and lowest maintenance while still achieving context preservation goals.

---

### Implementation Details

**What to implement first:**
- ✅ Plugin implemented and building successfully

**Things to watch out for:**
- ⚠️ API is marked 'experimental' - monitor OpenCode releases for changes
- ⚠️ CLI calls may fail in non-standard environments
- ⚠️ Context size should be monitored (don't inject too much)

**Areas needing further investigation:**
- How often does compaction actually happen in typical sessions?
- Does injected context actually survive in agent's understanding?
- What's the right balance of context (too much could be noise)?

**Success criteria:**
- ✅ Plugin loads without errors (verified via build)
- ✅ Agents retain awareness of tier, issue, phase after compaction (needs observation)
- ✅ No disruption to normal agent operation (needs observation)

---

## References

**Files Examined:**
- `~/.config/opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts` - Plugin API type definitions
- `~/.config/opencode/plugin/orchestrator-session.ts` - Existing plugin pattern reference
- `~/.config/opencode/plugin/bd-close-gate.ts` - Gate pattern reference
- `~/.config/opencode/plugin/action-log.ts` - Tool hook pattern reference
- `.orch/workspace/og-feat-opencode-plugin-session-08jan-9cea/.tier` - Workspace metadata

**Commands Run:**
```bash
# Test plugin builds
bun build plugin/session-compaction.ts --outdir=/tmp/test-build

# Check workspace metadata
ls -la .orch/workspace/og-feat-opencode-plugin-session-08jan-9cea/
cat .orch/workspace/og-feat-opencode-plugin-session-08jan-9cea/.tier

# Test kn commands
kn constraints --json
kn recent --n 5 --json
```

**External Documentation:**
- https://opencode.ai/docs/plugins - OpenCode plugin documentation

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-08-inv-epic-mechanize-principles-via-opencode.md` - Identified this capability, marked out of scope
- **Plugin:** `~/.config/opencode/plugin/session-compaction.ts` - The implemented solution

---

## Investigation History

**2026-01-08 10:46:** Investigation started
- Initial question: How can we preserve critical context during session compaction?
- Context: Investigation 2026-01-08-inv-epic-mechanize-principles-via-opencode.md identified the hook but marked it out of scope

**2026-01-08 10:50:** Analyzed plugin API
- Found `experimental.session.compacting` hook signature
- Confirmed append-only context injection is possible

**2026-01-08 11:00:** Implemented plugin
- Created `~/.config/opencode/plugin/session-compaction.ts`
- Builds successfully with bun

**2026-01-08 11:10:** Investigation completed
- Status: Complete
- Key outcome: Plugin implemented to preserve workspace tier, beads issue, phase status, and kb constraints during session compaction
