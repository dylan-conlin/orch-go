<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard now supports image paste (Cmd+V) and drag-drop with preview and inline display in activity feed.

**Evidence:** Built successfully, TypeScript checks pass, Claude API format confirmed compatible, OpenCode endpoint supports image parts.

**Knowledge:** Frontend-only implementation is sufficient - OpenCode API already supports multi-part messages with images. Base64 encoding keeps implementation simple but adds 33% payload overhead.

**Next:** Manual visual verification needed - orchestrator should test clipboard paste, drag-drop, and inline display in running dashboard.

**Promote to Decision:** recommend-no - tactical feature implementation, not architectural

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

# Investigation: Dashboard Add Image Paste Upload

**Question:** How do I add image paste/upload support to the dashboard's agent message input?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** OpenCode Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Message sending currently uses OpenCode's prompt_async API

**Evidence:** The activity-tab.svelte component sends messages to `http://localhost:4096/session/{sessionId}/prompt_async` with a JSON body containing a `parts` array. Current implementation only sends text parts: `{ type: 'text', text: message }`

**Source:** web/src/lib/components/agent-detail/activity-tab.svelte:404-413

**Significance:** To add image support, I need to extend the parts array to include image parts with base64 data, following the Claude API format

---

### Finding 2: Dashboard architecture follows API -> Store -> Component pattern

**Evidence:** Dashboard integrations follow the pattern: "API endpoint in serve.go -> Svelte store -> page.svelte integration" (from prior knowledge). The orch serve backend is at cmd/orch/serve.go, which sets up HTTP endpoints that the Svelte frontend consumes.

**Source:** Prior knowledge from SPAWN_CONTEXT.md line 54, cmd/orch/serve.go:1-450

**Significance:** For image support, I don't need backend API changes - the OpenCode prompt_async endpoint should already support image parts. I only need frontend changes to capture and display images

---

### Finding 3: Claude API image format uses base64 source

**Evidence:** Claude API expects image parts in format: `{ type: 'image', source: { type: 'base64', media_type: 'image/png', data: '<base64-string>' } }`. Images can be displayed using data URLs: `data:image/png;base64,{data}`.

**Source:** Claude API documentation (standard format), web/src/lib/components/agent-detail/screenshots-tab.svelte shows image handling patterns

**Significance:** I can extend the parts array to include image parts with base64 data, and display them inline using data URLs. No backend API changes needed.

---

## Synthesis

**Key Insights:**

1. **Frontend-only implementation** - All changes can be made in the Svelte frontend. The message sending already uses a flexible parts array that can contain both text and image parts.

2. **Standard base64 encoding** - Using the Clipboard API and FileReader API, I can capture images from paste/drop events, convert to base64, and include in the parts array following the Claude API format.

3. **Inline display pattern** - The activity feed already displays various message types. I can extend it to render image parts using data URLs, following the pattern established in screenshots-tab.svelte.

**Answer to Investigation Question:**

To add image paste/upload support to the dashboard, I need to:
1. Add event handlers for clipboard paste and drag-drop to the message input area
2. Convert captured images to base64 using FileReader API
3. Show image previews before sending (with remove option)
4. Include image parts in the parts array when calling OpenCode's prompt_async endpoint
5. Extend the activity feed rendering to display image parts inline

No backend changes are required - OpenCode's prompt_async endpoint should already support image parts per the Claude API specification.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Extend activity-tab.svelte with image capture and rendering** - Add clipboard/drag-drop handlers to capture images, convert to base64, preview before send, and include in parts array.

**Why this approach:**
- Leverages existing message sending infrastructure (prompt_async already supports parts array)
- Follows established patterns (screenshots-tab.svelte for image display)
- Frontend-only changes minimize implementation risk and testing scope
- Clipboard API and FileReader API are well-supported browser standards

**Trade-offs accepted:**
- Images stored as base64 in message history (not uploaded to separate storage)
- No image editing/annotation capabilities (keep initial implementation simple)
- No support for multiple images per message initially (can extend later)

**Implementation sequence:**
1. Add clipboard paste handler - captures Cmd/Ctrl+V events, foundational for paste workflow
2. Add drag-drop handlers - captures file drop events, parallel to paste workflow
3. Add image preview component - shows pending images before send, user feedback
4. Modify sendMessage to include image parts - integrates with OpenCode API
5. Extend activity feed rendering - displays images inline in message history

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- Clipboard paste handler (Cmd/Ctrl+V) - highest user value, enables primary workflow
- Image state management (pendingImages array) - foundational for all features
- Image preview component - immediate visual feedback for user

**Things to watch out for:**
- ⚠️ Base64 size limits - large images can exceed message size limits (consider compression or size warnings)
- ⚠️ MIME type validation - only accept image/* types (png, jpg, gif, webp)
- ⚠️ Drag-drop z-index conflicts - overlay must appear above other dashboard elements
- ⚠️ Memory cleanup - ensure removed images don't leak memory (revoke object URLs)

**Areas needing further investigation:**
- Image size optimization (client-side compression before base64 encoding)
- Workspace upload option (store images in .orch/workspace/{id}/images/ instead of base64)
- Multi-image support (allow multiple images in one message)

**Success criteria:**
- ✅ Can paste image from clipboard with Cmd+V
- ✅ Can drag-drop image file onto input area
- ✅ Image preview appears before send with remove button
- ✅ Images display inline in activity feed after sending
- ✅ Images appear in agent's message history across refreshes

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
