<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Designed a layered provenance system for synthesized artifacts. Three tiers: (1) Session Metadata (always present), (2) Source References (file reads tracked by OpenCode), (3) Thread Lineage (conversation evolution).

**Evidence:** OpenCode API already provides `GetMessages()` returning full conversation including tool invocations. MessagePart.Type includes "tool-invocation" which can track file reads. Session metadata (ID, title, directory, timestamps) available via `/session/{id}` endpoint.

**Knowledge:** Provenance is the natural extension of Self-Describing Artifacts principle. Key insight: OpenCode already captures the raw data (messages, tool invocations), we just need to structure it. Most value comes from Tier 1 (minimal overhead), with Tier 2-3 available for high-stakes synthesis.

**Next:** Create beads issue for implementation. Start with Tier 1 (session metadata in artifact headers) since it's zero-cost and immediately useful.

---

# Investigation: Post Provenance System Trace Artifacts

**Question:** How can synthesized artifacts (posts, decisions, investigations) include their provenance chain - sources consulted, prompts that shaped output, thread evolution?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** Create follow-up issue for implementation
**Status:** Complete

---

## Findings

### Finding 1: OpenCode API Already Provides Raw Provenance Data

**Evidence:** The OpenCode client (`pkg/opencode/client.go:474-495`) provides `GetMessages(sessionID)` which returns all messages including:

```go
type Message struct {
    Info  MessageInfo   `json:"info"`
    Parts []MessagePart `json:"parts"`
}

type MessagePart struct {
    ID        string `json:"id"`
    Type      string `json:"type"` // "text", "reasoning", "step-start", "step-finish", "tool-invocation"
    Text      string `json:"text,omitempty"`
}
```

Tool invocations (file reads, searches, writes) are captured as message parts. This means the full provenance chain already exists - we just need to structure it.

**Source:** `pkg/opencode/types.go:77-125`, `pkg/opencode/client.go:474-495`

**Significance:** No new data capture infrastructure needed. The design question becomes: how do we extract and present provenance from existing data?

---

### Finding 2: Provenance Aligns with Self-Describing Artifacts Principle

**Evidence:** From `~/.kb/principles.md`:

> "Generated files and agent-edited artifacts must contain their own operating instructions."
> 
> Key questions artifacts should answer:
> 1. What is this
> 2. What NOT to do
> 3. Where is source
> 4. How to modify
> 5. When generated

Provenance extends this to: **How was this synthesized?** The principle already requires "where is source" - provenance is the rigorous version of that.

**Source:** `~/.kb/principles.md:67-101`

**Significance:** Provenance isn't a new concept - it's the natural extension of existing principles. The `Provenance` principle itself states: "Every conclusion must trace to something outside the conversation."

---

### Finding 3: Three Tiers of Provenance Complexity

**Evidence:** Analyzing the SPAWN_CONTEXT example from the task description and current templates:

**Tier 1: Session Metadata (Minimal)**
```markdown
<!-- PROVENANCE -->
<!-- Session: og-feat-post-provenance-27dec -->
<!-- Project: /Users/dylanconlin/Documents/personal/orch-go -->
<!-- Timestamp: 2025-12-27 14:30:00 -->
<!-- Skill: feature-impl -->
<!-- /PROVENANCE -->
```
- Zero runtime cost
- Captures WHERE this came from
- Enables linking back to full session via OpenCode API

**Tier 2: Source References (Medium)**
```markdown
<!-- PROVENANCE -->
...
<!-- Sources consulted:
<!--   - ~/.kb/principles.md (read 2x)
<!--   - pkg/opencode/types.go (read 1x)
<!--   - pkg/spawn/context.go (read 1x)
<!-- /PROVENANCE -->
```
- Requires parsing tool invocations from messages
- Shows WHAT informed the output
- Most useful for blog posts, decisions, significant artifacts

**Tier 3: Thread Lineage (Full)**
```markdown
<!-- PROVENANCE -->
...
<!-- Thread evolution:
<!--   1. Initial prompt: "Design post provenance system"
<!--   2. Investigation: Explored OpenCode API
<!--   3. Finding: API already captures tool invocations
<!--   4. Synthesis: Designed 3-tier system
<!-- Conversation: <opencode-session-id>
<!-- /PROVENANCE -->
```
- Full prompt lineage
- Shows HOW the thread evolved
- Most useful for auditing complex synthesis

**Source:** Task description example + current SYNTHESIS.md template analysis

**Significance:** Tiered approach matches existing patterns (light vs full tier spawns). Start minimal, add richness when needed.

---

### Finding 4: Implementation Hooks Already Exist

**Evidence:** The spawn system already writes metadata files:

```go
// pkg/spawn/context.go:457-468
func WriteContext(cfg *Config) error {
    // ...
    // Write tier metadata file for orch complete to read
    if err := WriteTier(workspacePath, cfg.Tier); err != nil { ... }
    // Write spawn time for constraint verification scoping
    if err := WriteSpawnTime(workspacePath, time.Now()); err != nil { ... }
}
```

Pattern: `.tier`, `.spawn_time`, `.session_id` files in workspace. Provenance could follow same pattern with a `.provenance` file or embed directly in artifacts.

**Source:** `pkg/spawn/context.go:438-468`, `.orch/workspace/*/` directory listing

**Significance:** Infrastructure for per-session metadata capture exists. Adding provenance follows established patterns.

---

### Finding 5: Artifact Templates Need Provenance Slot

**Evidence:** Current templates don't have a provenance section:

- SYNTHESIS.md has Session Metadata but no provenance block
- Investigation template has no provenance block
- Blog posts (external) would need provenance for validation

The templates should include an optional provenance block that agents can fill.

**Source:** `.orch/templates/SYNTHESIS.md:146-153`

**Significance:** Template update is low-effort, high-value. Add optional `## Provenance` or comment block to templates.

---

## Synthesis

**Key Insights:**

1. **Data exists, structure doesn't** - OpenCode captures everything (messages, tool invocations, timestamps). We need extraction and formatting, not new capture infrastructure.

2. **Tier-based approach matches system design** - Just like light/full spawns, artifacts can have minimal/rich provenance based on importance.

3. **Session ID is the minimal provenance** - If an artifact includes its session ID, the full provenance chain is recoverable via `client.GetMessages(sessionID)`.

4. **Provenance completes the Self-Describing Artifacts principle** - Artifacts should answer "where is source" - provenance is the rigorous answer for AI-synthesized content.

**Answer to Investigation Question:**

Posts and synthesized artifacts can include provenance through:

1. **Minimal (always):** Session ID in artifact header - enables recovery of full chain
2. **Standard (for decisions/investigations):** Session ID + source file list
3. **Rich (for blog posts/external artifacts):** Full thread evolution + sources + prompts

The implementation path is:
1. Add `## Provenance` block to artifact templates
2. Create `orch provenance <session-id>` command that extracts provenance from OpenCode messages
3. Agents fill provenance section before completing (gated for full-tier spawns)

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode API provides GetMessages with tool invocations (verified: read types.go and client.go)
- ✅ Session metadata (ID, title, directory) available via API (verified: read client.go:303-325)
- ✅ Workspace already captures .session_id files (verified: listed workspace directory)

**What's untested:**

- ⚠️ Performance of parsing tool invocations from large message histories
- ⚠️ Whether agents will consistently fill provenance sections
- ⚠️ Whether session IDs remain valid/accessible after OpenCode restarts

**What would change this:**

- If OpenCode doesn't persist sessions, provenance via session ID fails → need to extract at completion time
- If tool invocations aren't parsed reliably, source tracking is unreliable → need to log file reads separately

---

## Implementation Recommendations

**Purpose:** Bridge from investigation to actionable implementation.

### Recommended Approach ⭐

**Tier-based provenance with session ID as foundation**

Start with Tier 1 (session metadata), make Tier 2-3 opt-in for high-stakes artifacts.

**Why this approach:**
- Minimal overhead for most work
- Session ID enables full provenance recovery when needed
- Follows established tier pattern (light/full spawns)
- Aligns with Provenance principle without adding friction

**Trade-offs accepted:**
- Full provenance requires separate extraction step
- Agents must fill provenance consciously (not automatic)

**Implementation sequence:**

1. **Add provenance block to templates** - `## Provenance` or HTML comment block
2. **Create `orch provenance <session-id>` command** - Extracts sources consulted, thread evolution from messages
3. **Update feature-impl skill** - Gate full-tier spawns on provenance completion
4. **Document for external artifacts** - Blog posts should run `orch provenance` and paste result

### Alternative Approaches Considered

**Option B: Automatic provenance capture at completion**
- **Pros:** Zero agent effort, always complete
- **Cons:** Requires completion hook, adds latency, may capture irrelevant sources
- **When to use instead:** If agents consistently fail to fill provenance manually

**Option C: Full provenance embedded in every artifact**
- **Pros:** Maximum transparency
- **Cons:** Bloated artifacts, noise in most cases
- **When to use instead:** Never - always use tiered approach

**Rationale for recommendation:** Tiered approach matches existing system patterns and balances transparency with overhead.

---

### Implementation Details

**What to implement first:**
- Template update (5 min)
- `orch provenance` command basic version (session metadata + source list)

**Things to watch out for:**
- ⚠️ Session persistence - verify sessions survive OpenCode restart
- ⚠️ Tool invocation parsing - may need to handle different part types
- ⚠️ Private data in provenance - may need filtering

**Areas needing further investigation:**
- How long do OpenCode sessions persist?
- Should provenance be extracted at completion or on-demand?

**Success criteria:**
- ✅ Blog post can include `<!-- PROVENANCE -->` block with session + sources
- ✅ `orch provenance <id>` returns formatted provenance for any session
- ✅ Full-tier spawns gated on provenance section filled

---

## References

**Files Examined:**
- `~/.kb/principles.md` - Provenance and Self-Describing Artifacts principles
- `pkg/opencode/types.go` - Message and MessagePart structures
- `pkg/opencode/client.go` - GetMessages, GetSession API
- `pkg/spawn/context.go` - Workspace metadata patterns
- `.orch/templates/SYNTHESIS.md` - Current template structure

**Commands Run:**
```bash
# List workspace contents
ls .orch/workspace/

# Search for message handling
rg "GetMessages|Message" pkg/ --include "*.go"
```

**Related Artifacts:**
- **Principle:** `~/.kb/principles.md` - Provenance (foundational) and Self-Describing Artifacts
- **Investigation:** `.kb/investigations/2025-12-21-inv-citation-mechanisms-how-artifacts-track.md` - Content-based citation discovery patterns

---

## Investigation History

**2025-12-27 ~14:00:** Investigation started
- Initial question: How can synthesized artifacts include provenance chain?
- Context: Surfaced during blog post outlining, deferred to ship post first

**2025-12-27 ~14:45:** Core findings complete
- Found OpenCode API already captures raw provenance data
- Designed 3-tier provenance system
- Identified template update as first implementation step

**2025-12-27 ~15:00:** Investigation completed
- Status: Complete
- Key outcome: Tiered provenance system with session ID as foundation
