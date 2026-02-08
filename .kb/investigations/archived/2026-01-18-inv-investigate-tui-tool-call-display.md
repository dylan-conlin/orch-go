<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Gemini models output TEXT that mimics the tool_call prompt examples before making actual tool calls, causing duplicate display of tool metadata.

**Evidence:** Found gemini.txt prompt at `/packages/opencode/src/session/prompt/gemini.txt` contains `[tool_call: ...]` examples; Gemini outputs similar text before calling tools; TUI renders both text parts and tool calls.

**Knowledge:** This is a Gemini-specific behavior - the model interprets prompt examples as a format to follow when announcing tool calls, producing redundant text output that the TUI displays alongside the actual tool call rendering.

**Next:** Add text filtering via `experimental.text.complete` plugin hook to strip `tool_call:` prefixed text and JSON parameter blocks from Gemini model output.

**Promote to Decision:** recommend-no (provider-specific fix, not architectural)

---

# Investigation: TUI Tool Call Display Bug

**Question:** Why does the OpenCode TUI show redundant formatting for tool calls (tool_call: prefix, JSON params, markdown header)?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None - findings documented, fix approach identified
**Status:** Complete

---

## Findings

### Finding 1: Gemini prompt contains tool_call examples

**Evidence:** The file `/packages/opencode/src/session/prompt/gemini.txt` contains examples like:
```
model: [tool_call: ls for path '/path/to/project']
model: [tool_call: bash for 'node server.js &' because it must run in the background]
```

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt/gemini.txt:77-128`

**Significance:** Gemini models interpret these examples as a format to follow, outputting similar text before making actual tool calls. This explains the `tool_call:` prefix in the bug report.

---

### Finding 2: TUI renders both text parts and tool parts

**Evidence:** The Bash tool renderer in the TUI shows:
- When completed: `# description` header and `$ command`
- The text parts (model output) are rendered separately before tool parts

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/cli/cmd/tui/routes/session/index.tsx:1537-1556`

**Significance:** When Gemini outputs text like `tool_call: bash for '...'` AND makes the actual tool call, BOTH are rendered - first the text part, then the tool call with proper formatting.

---

### Finding 3: Plugin hook exists for text filtering

**Evidence:** The `experimental.text.complete` hook in the session processor allows plugins to modify text output before it's saved:
```typescript
const textOutput = await Plugin.trigger(
  "experimental.text.complete",
  { sessionID, messageID, partID },
  { text: currentText.text },
)
currentText.text = textOutput.text
```

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/processor.ts:308-317`

**Significance:** This hook provides a clean mechanism to filter out unwanted text patterns like `tool_call:` prefixes without modifying core rendering logic.

---

### Finding 4: JSON parameters shown due to title fallback

**Evidence:** The CLI `run.ts` shows JSON parameters when `part.state.title` is missing:
```typescript
const title =
  part.state.title ||
  (Object.keys(part.state.input).length > 0 ? JSON.stringify(part.state.input) : "Unknown")
```

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/cli/cmd/run.ts:166-168`

**Significance:** However, the bash tool DOES set a title (`params.description`), so this isn't the primary cause. The JSON in the bug report is likely from Gemini's text output, not the run.ts fallback.

---

## Synthesis

**Key Insights:**

1. **Gemini-specific behavior** - The prompt examples teach Gemini to announce tool calls with text, but the model outputs this text separately from the actual tool call, causing duplicate display.

2. **Text and tools render independently** - The TUI has separate rendering paths for text parts and tool parts. When both exist for the same "tool call", both are displayed.

3. **Plugin hook is the clean fix path** - The `experimental.text.complete` hook allows text modification before display without changing core logic.

**Answer to Investigation Question:**

The redundant formatting occurs because:
1. Gemini models output TEXT announcing tool calls (following prompt examples): `tool_call: bash for '...'`
2. This text may include JSON parameters as the model "explains" what it's doing
3. The actual tool call is made separately
4. The TUI renders BOTH the text parts (showing `tool_call:` and JSON) AND the tool parts (showing `# description` and `$ command`)

The fix is to filter out text matching `tool_call:` patterns via the `experimental.text.complete` plugin hook.

---

## Structured Uncertainty

**What's tested:**

- ✅ Gemini prompt contains tool_call examples (verified: read gemini.txt)
- ✅ TUI has separate text and tool rendering (verified: read index.tsx)
- ✅ Plugin hook exists for text filtering (verified: read processor.ts)
- ✅ Bash tool sets title correctly (verified: read bash.ts line 248)

**What's untested:**

- ⚠️ Actual Gemini model behavior (would need live testing with Gemini session)
- ⚠️ Whether other providers have similar issues (Claude, GPT models not checked)
- ⚠️ Plugin hook performance impact (not benchmarked)

**What would change this:**

- Finding would be wrong if Gemini doesn't output tool_call text (would need different root cause)
- Finding would need revision if text filtering breaks legitimate text content
- Fix approach would change if plugin hooks aren't called for streamed text

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Text Filtering Plugin** - Create a plugin using `experimental.text.complete` hook to strip `tool_call:` prefixed lines and JSON-like blocks from text output.

**Why this approach:**
- Uses existing plugin infrastructure - no core code changes needed
- Can be enabled/disabled per project or globally
- Easy to refine filtering rules without touching OpenCode core

**Trade-offs accepted:**
- May need careful regex to avoid filtering legitimate text
- Plugin must handle edge cases (partial matches, valid mentions of "tool_call")

**Implementation sequence:**
1. Create plugin at `~/.opencode/plugin/filter-tool-call-text.ts`
2. Register for `experimental.text.complete` hook
3. Strip lines matching `^tool_call:` and standalone JSON blocks
4. Test with Gemini model session

### Alternative Approaches Considered

**Option B: Modify Gemini prompt**
- **Pros:** Fixes at source, prevents model from outputting text
- **Cons:** May change model behavior in unintended ways; prompt is upstream in OpenCode
- **When to use instead:** If plugin filtering proves unreliable

**Option C: TUI text filtering**
- **Pros:** Applies only to display, doesn't modify stored data
- **Cons:** Requires TUI code changes; text still stored in session
- **When to use instead:** If plugin approach has performance issues with large text

**Rationale for recommendation:** Plugin approach is cleanest - isolated fix, uses existing hooks, no upstream changes needed.

---

### Implementation Details

**What to implement first:**
- Basic plugin that strips `tool_call:` prefixed lines
- Test with live Gemini session to verify fix

**Things to watch out for:**
- ⚠️ Regex must not match legitimate text (e.g., documentation mentioning tool_call)
- ⚠️ JSON block detection should be context-aware (only strip when following tool_call)
- ⚠️ Streaming text - ensure filtering works during incremental updates

**Areas needing further investigation:**
- Whether this affects Claude Code or other providers
- Whether gemini.txt prompt should be modified to discourage text announcements

**Success criteria:**
- ✅ Gemini tool calls show only the standard format (`$ command`, output) without preceding `tool_call:` text
- ✅ Legitimate text containing "tool_call" is not filtered
- ✅ No regression in other model behaviors

---

## References

**Files Examined:**
- `/packages/opencode/src/session/prompt/gemini.txt` - Gemini system prompt with tool_call examples
- `/packages/opencode/src/cli/cmd/tui/routes/session/index.tsx` - TUI session rendering
- `/packages/opencode/src/session/processor.ts` - Message processing with plugin hooks
- `/packages/opencode/src/tool/bash.ts` - Bash tool implementation
- `/packages/opencode/src/cli/cmd/run.ts` - CLI run command with JSON fallback

**Commands Run:**
```bash
# Search for tool_call patterns
rg "tool_call:" ~/Documents/personal/opencode

# Check plugin directory
ls -la ~/Documents/personal/opencode/.opencode/plugin/
```

**Related Artifacts:**
- **Guide:** `.kb/guides/opencode-plugins.md` - OpenCode plugin system documentation

---

## Investigation History

**2026-01-18 10:36:** Investigation started
- Initial question: Why does TUI show redundant tool call formatting?
- Context: Bug report showing `tool_call:` prefix, JSON params, and markdown header

**2026-01-18 10:50:** Root cause identified
- Gemini prompt examples cause model to output text before tool calls
- TUI renders both text and tool parts, causing duplication

**2026-01-18 11:00:** Investigation completed
- Status: Complete
- Key outcome: Gemini-specific text output issue; fixable via plugin filtering
