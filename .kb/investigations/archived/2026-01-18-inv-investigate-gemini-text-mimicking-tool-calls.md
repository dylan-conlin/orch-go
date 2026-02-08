<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Gemini models output text mimicking tool calls because the `gemini.txt` system prompt contains 14+ examples with `[tool_call: ...]` text patterns that the model learns to mimic instead of using the actual function calling API.

**Evidence:** Found 14 instances of `[tool_call: ...]` in `gemini.txt` (lines 77-146); Anthropic prompt has zero such patterns; bug report shows `[tool_call: edit{...}]` with `<ctrl46>` escape characters confirming text output not API call.

**Knowledge:** LLMs follow prompt examples literally; showing tool calls as text teaches models to output text instead of using API-level function calling; other providers' prompts avoid this pattern.

**Next:** Remove or replace `[tool_call: ...]` examples in `gemini.txt` with prose descriptions that don't create a mimickable format.

**Promote to Decision:** recommend-no (tactical fix in OpenCode fork, not architectural)

---

# Investigation: Gemini Models Output Text Mimicking Tool Calls Instead of Invoking Tools

**Question:** Why do Gemini models sometimes output TEXT mimicking tool call format (e.g., `[tool_call: edit{...}]`) instead of actually invoking tools via the API?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** Worker agent (orch-go-nsy7w)
**Phase:** Complete
**Next Step:** None - escalate fix to OpenCode maintainers or implement in fork
**Status:** Complete

<!-- Lineage -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Gemini prompt contains 14+ `[tool_call: ...]` text examples

**Evidence:** The `gemini.txt` system prompt contains examples showing tool calls as TEXT:
```
model: [tool_call: ls for path '/path/to/project']
model: [tool_call: bash for 'node server.js &' because it must run in the background]
[tool_call: glob for path 'tests/test_auth.py']
[tool_call: read for absolute_path '/path/to/tests/test_auth.py']
[tool_call: write or edit to apply the refactoring to 'src/auth.py']
[tool_call: bash for 'ruff check src/auth.py && pytest']
[tool_call: grep for pattern 'UserProfile|updateProfile|editUser']
```

14 total instances at lines 77, 82, 89, 90, 94, 107, 109, 122, 124, 126, 128, 135, 138, 146.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt/gemini.txt` lines 77-146

**Significance:** These examples teach the model "when I want to call a tool, I output `[tool_call: ...]` as text". The model mimics this pattern instead of using the actual Gemini function calling API. The `<ctrl46>` characters in the bug report (`<ctrl46>/path/to/file<ctrl46>`) show the model trying to escape/quote strings within its text-based output.

---

### Finding 2: Anthropic prompt does NOT have this pattern

**Evidence:** The `anthropic.txt` system prompt refers to tool calls abstractly without showing text representations:
- "You can call multiple tools in a single response" (line 83)
- References "tool use content blocks" (line 84)
- No `[tool_call: ...]` examples anywhere

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt/anthropic.txt`

**Significance:** Anthropic models don't exhibit this bug because they're not trained by the prompt to output tool calls as text. The difference in prompt style is the root cause.

---

### Finding 3: System prompt selection is model-based

**Evidence:** In `system.ts` line 30:
```typescript
if (model.api.id.includes("gemini-")) return [PROMPT_GEMINI]
```

Different models get different system prompts:
- `gemini-*` → `gemini.txt` (problematic)
- `claude*` → `anthropic.txt` (good)
- `gpt-5*` → `codex.txt` (no text tool patterns)

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/system.ts` lines 26-33

**Significance:** The bug is Gemini-specific because only the Gemini prompt contains the problematic `[tool_call: ...]` patterns.

---

### Finding 4: Gemini API supports proper function calling

**Evidence:** Web search confirms Gemini API returns structured `functionCall` objects:
- Function calling returns `{"name": "get_order_status", "args": {"order_id": "123"}}`
- This is proper JSON, not text output
- The AI SDK handles this via `@ai-sdk/google`

**Source:** [Google AI Gemini Function Calling Docs](https://ai.google.dev/gemini-api/docs/function-calling)

**Significance:** The API itself is capable of proper function calling. The bug is purely a prompt engineering issue where the model is trained by examples to output text instead of using the API.

---

## Synthesis

**Key Insights:**

1. **Prompt examples are treated as training data** - LLMs learn from in-context examples. Showing `[tool_call: ls for path '...']` teaches the model to output that format as text.

2. **Text representation competes with API invocation** - When a model sees both "use the function calling API" (from AI SDK) and "output `[tool_call: ...]`" (from system prompt), it can choose either. The abundant prompt examples bias it toward text output.

3. **Provider-specific prompts require provider-specific testing** - The Gemini prompt was likely created without testing whether its example format would be mimicked. Other prompts avoided this by not showing a mimickable format.

**Answer to Investigation Question:**

Gemini models output text mimicking tool calls because the `gemini.txt` system prompt contains 14 examples showing tool invocations as `[tool_call: tool_name for args]` text format. The model learns this pattern from the examples and outputs it as text instead of using the actual Gemini function calling API. This is NOT:
- A rate limit fallback (the model is not being throttled)
- An API limitation (Gemini API supports proper function calling)
- A model confusion issue (the model is correctly following its training)

It IS a prompt engineering bug where examples inadvertently train the model to output text.

---

## Structured Uncertainty

**What's tested:**

- ✅ `gemini.txt` contains 14 `[tool_call: ...]` patterns (verified: `grep "tool_call:" gemini.txt | wc -l`)
- ✅ `anthropic.txt` contains zero `[tool_call: ...]` patterns (verified: same grep returns empty)
- ✅ System prompt selection routes Gemini models to `gemini.txt` (verified: read `system.ts` line 30)

**What's untested:**

- ⚠️ Removing the patterns fixes the bug (requires prompt change and re-test)
- ⚠️ Frequency of bug occurrence (no data on how often this happens)
- ⚠️ Specific Gemini model versions affected (gemini-3-flash, gemini-2.5-pro, etc.)

**What would change this:**

- Finding would be wrong if Gemini exhibits this bug with a prompt that has no `[tool_call: ...]` examples
- Finding would be incomplete if other prompt elements also contribute to the behavior

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Remove `[tool_call: ...]` text patterns from gemini.txt** - Replace with prose descriptions that don't create a mimickable format.

**Why this approach:**
- Directly addresses root cause (examples that model mimics)
- Low risk (only changes system prompt, not code)
- Consistent with how other provider prompts handle examples

**Trade-offs accepted:**
- Gemini prompt may be less instructive without concrete examples
- Acceptable because proper tool calling is more important than instruction clarity

**Implementation sequence:**
1. Replace all 14 `[tool_call: ...]` instances with prose like "Use the read tool to examine the file"
2. Test with Gemini models to verify proper function calling
3. Monitor for regression in tool usage understanding

### Alternative Approaches Considered

**Option B: Add explicit "never output tool calls as text" instruction**
- **Pros:** Keeps examples, adds guardrail
- **Cons:** Instructions can be ignored; examples are stronger signal than instructions
- **When to use instead:** If removing examples degrades tool usage understanding

**Option C: Use different example format like `[Assistant uses read tool]`**
- **Pros:** Still shows tool usage, less mimickable
- **Cons:** May still create a format model outputs
- **When to use instead:** If completely removing examples is too radical

**Rationale for recommendation:** Removing the examples is cleanest. The Anthropic prompt works without them, proving they're not necessary.

---

### Implementation Details

**What to implement first:**
- Edit `gemini.txt` to remove/replace `[tool_call: ...]` patterns
- Quick fix, single file change

**Things to watch out for:**
- ⚠️ Test that Gemini models still understand how to use tools after prompt change
- ⚠️ Monitor for increased "I don't know how to use this tool" responses
- ⚠️ The `[tool_call: ...]` format may be documented elsewhere

**Areas needing further investigation:**
- Frequency of this bug in production (metrics)
- Whether specific Gemini model versions are more susceptible
- Whether thinking-mode models (gemini-3) behave differently

**Success criteria:**
- ✅ Gemini models no longer output `[tool_call: ...]` as text
- ✅ `git diff` shows actual file changes after edit commands
- ✅ No regression in tool usage comprehension

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt/gemini.txt` - Gemini system prompt with problematic examples
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt/anthropic.txt` - Anthropic prompt for comparison
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/system.ts` - Prompt selection logic
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/provider/provider.ts` - Provider configuration
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/provider/transform.ts` - Tool schema handling

**Commands Run:**
```bash
# Find tool_call patterns in Gemini prompt
grep "tool_call:" /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt/gemini.txt

# Verify Anthropic prompt lacks pattern
grep "tool_call:" /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt/anthropic.txt
```

**External Documentation:**
- [Google AI Gemini Function Calling](https://ai.google.dev/gemini-api/docs/function-calling) - Confirms API supports structured function calls

**Related Artifacts:**
- **Investigation:** orch-go-f5c6b (display duplication - cosmetic manifestation of same root cause)
- **Beads Issue:** orch-go-nsy7w (this investigation's parent issue)

---

## Investigation History

**2026-01-18 ~14:00:** Investigation started
- Initial question: Why do Gemini models output text mimicking tool calls?
- Context: Bug report showed `[tool_call: edit{...}]` with `<ctrl46>` escape characters, file not modified

**2026-01-18 ~14:15:** Found `[tool_call: ...]` patterns in gemini.txt
- Counted 14 instances in example blocks
- Realized these examples train model to output text format

**2026-01-18 ~14:20:** Confirmed Anthropic prompt lacks this pattern
- Compared prompts, found clear difference in how examples are shown
- Validated hypothesis that prompt is root cause

**2026-01-18 ~14:30:** Investigation completed
- Status: Complete
- Key outcome: Bug caused by prompt examples teaching model to output tool calls as text
