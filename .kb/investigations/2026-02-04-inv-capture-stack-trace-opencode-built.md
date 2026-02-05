<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Stack trace is lost at processor.ts:204 which uses `.toString()` on errors instead of preserving `.stack` property.

**Evidence:** Code review confirmed: `error: (value.error as any).toString()` loses stack; `FormatUnknownError` in error.ts correctly uses `.stack` but processor bypasses it.

**Knowledge:** Error handling path has inconsistency - CLI error formatting preserves stacks, but tool error storage strips them.

**Next:** Fix processor.ts:204 to preserve stack trace, enable `--log-level DEBUG` to capture errors during reproduction.

**Authority:** implementation - Single file fix within OpenCode, no architectural changes needed.

---

# Investigation: Stack Trace Capture for OpenCode Built-in Tools Stack Overflow

**Question:** How can we capture the full stack trace for the "Maximum call stack size exceeded" error affecting read/glob/grep tools?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Investigation worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-02-04-inv-root-cause-opencode-mcp-tools.md` | extends | yes | Confirmed tools are built-in not MCP; verified schema simplicity |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** None found - prior investigation accurately identified candidate locations

---

## Findings

### Finding 1: Stack trace is lost at processor.ts:204

**Evidence:** In `session/processor.ts` line 204, tool errors are stored with:
```typescript
error: (value.error as any).toString(),
```
This converts the Error to just its message string, losing the `.stack` property that contains the full call trace.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/processor.ts:204`

**Significance:** This is the PRIMARY cause of stack trace loss. The error object has a `.stack` property but `.toString()` only returns `[name]: [message]` without the trace.

---

### Finding 2: CLI error handling DOES preserve stack traces

**Evidence:** In `cli/error.ts` lines 43-57:
```typescript
export function FormatUnknownError(input: unknown): string {
  if (input instanceof Error) {
    return input.stack ?? `${input.name}: ${input.message}`
  }
  ...
}
```
This function correctly uses `.stack` when available. But the processor doesn't use this function.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/cli/error.ts:43-57`

**Significance:** The correct pattern exists in the codebase - processor.ts just doesn't follow it.

---

### Finding 3: OpenCode supports DEBUG log level via --log-level flag

**Evidence:** From index.ts:55-67, OpenCode accepts `--log-level DEBUG` flag:
```typescript
.option("logLevel", {
  describe: "log level",
  type: "string",
  choices: ["DEBUG", "INFO", "WARN", "ERROR"],
})
```
When running locally (Installation.isLocal()), DEBUG is the default.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/index.ts:55-67`

**Significance:** We can enable verbose logging to capture errors at their origin, before they reach the processor error handler.

---

### Finding 4: Tool schemas are simple - overflow likely in JSON schema conversion

**Evidence:** Compared all affected tools:
- `read.ts`: `z.object({ filePath: z.string(), offset: z.number().optional(), limit: z.number().optional() })`
- `glob.ts`: `z.object({ pattern: z.string(), path: z.string().optional() })`
- `grep.ts`: `z.object({ pattern: z.string(), path: z.string().optional(), include: z.string().optional() })`

These are flat, non-recursive schemas. Bash tool has similar complexity but uses async init form.

**Source:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/read.ts:19-23`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/glob.ts:11-19`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/grep.ts:14-18`

**Significance:** The overflow is NOT in the schema definition itself. It must be in either:
1. `z.toJSONSchema()` conversion (prompt.ts:701)
2. Something else in the tool execution path

---

## Synthesis

**Key Insights:**

1. **Error handling inconsistency** - The codebase has correct stack-preserving error formatting in `cli/error.ts` but the processor uses a simpler `.toString()` that loses the trace.

2. **Two-pronged capture needed** - To get the full stack trace, we need both:
   - Fix the processor to preserve stack (for errors that reach the client)
   - Use DEBUG logging to capture errors at their origin point

3. **Schema complexity ruled out** - All affected tools have simple flat schemas, confirming the prior investigation's hypothesis that the issue is in the conversion/wrapper layer, not the schema definitions.

**Answer to Investigation Question:**

To capture the full stack trace:
1. **Immediate fix**: Modify `processor.ts:204` to preserve `.stack` property instead of using `.toString()`
2. **Debug reproduction**: Run OpenCode with `--log-level DEBUG` and reproduce the error to see the full trace in logs
3. **Alternative**: Add try/catch around `z.toJSONSchema()` at prompt.ts:701 with explicit stack logging

---

## Structured Uncertainty

**What's tested:**

- ✅ processor.ts:204 uses `.toString()` (verified: read source code)
- ✅ cli/error.ts uses `.stack` (verified: read source code)
- ✅ Tool schemas are simple flat objects (verified: read all tool definitions)
- ✅ DEBUG log level available via flag (verified: read index.ts)

**What's untested:**

- ⚠️ Whether the fix actually exposes the stack trace (need to rebuild and test)
- ⚠️ Whether the error originates at z.toJSONSchema or elsewhere (need stack trace to confirm)
- ⚠️ Whether bash tool's async init form is the reason it works (correlational, not causal)

**What would change this:**

- If stack trace shows error NOT in toJSONSchema, then prior investigation's hypothesis is wrong
- If error occurs BEFORE processor handles it, we'd need additional logging points

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Fix processor.ts:204 to preserve stack | implementation | Single file change, clear pattern exists in codebase |
| Add debug logging at z.toJSONSchema call | implementation | Non-invasive diagnostic addition |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)

### Recommended Approach ⭐

**Preserve stack trace in processor.ts** - Change line 204 to use `.stack` instead of `.toString()`.

**Why this approach:**
- Direct fix for the immediate problem (stack trace loss)
- Follows existing pattern in cli/error.ts
- Minimal code change, easy to review

**Trade-offs accepted:**
- Stack traces may be verbose in error output
- Acceptable because debugging is the goal

**Implementation sequence:**
1. Modify processor.ts:204 to use error.stack ?? error.toString()
2. Rebuild OpenCode: `cd ~/Documents/personal/opencode/packages/opencode && bun run build`
3. Restart server and reproduce to capture stack trace

### Alternative Approaches Considered

**Option B: Add try/catch wrapper at toJSONSchema**
- **Pros:** Captures error at exact origin point
- **Cons:** More invasive, adds try/catch blocks to hot path
- **When to use instead:** If processor fix doesn't give clear enough trace

**Option C: Use Node.js --stack-trace-limit flag**
- **Pros:** Captures longer stack traces
- **Cons:** Requires process-level configuration
- **When to use instead:** If standard stack trace is too short to identify root cause

**Rationale for recommendation:** Processor fix is simplest and matches existing codebase patterns.

---

### Implementation Details

**What to implement first:**
- Fix processor.ts:204 (one-line change)
- Rebuild OpenCode

**Things to watch out for:**
- ⚠️ OpenCode server needs restart after rebuild
- ⚠️ May need to clear browser cache if using web UI
- ⚠️ Error may be intermittent - have reproduction steps ready

**Areas needing further investigation:**
- After getting stack trace, need to identify exact recursion location
- May need to investigate zod v4 upgrade path

**Success criteria:**
- ✅ Stack trace shows full call chain when error occurs
- ✅ Can identify which function is causing the recursion
- ✅ Have clear path to root cause fix

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/processor.ts` - Tool error handling
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/cli/error.ts` - CLI error formatting with stack preservation
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt.ts` - Tool setup with z.toJSONSchema call
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/read.ts` - Read tool definition
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/glob.ts` - Glob tool definition
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/grep.ts` - Grep tool definition
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/bash.ts` - Bash tool (works, uses async init)
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/util/log.ts` - Logging infrastructure
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/index.ts` - CLI options including log level

**Commands Run:**
```bash
# Check server status
curl -s http://localhost:4096/health

# Search for error handling patterns
grep "\.stack" packages/opencode/src/**/*.ts
grep "\.toString()" packages/opencode/src/**/*.ts
```

**External Documentation:**
- Zod v4 stack overflow issue: https://github.com/colinhacks/zod/issues/4994

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-04-inv-root-cause-opencode-mcp-tools.md` - Prior root cause analysis

---

## Investigation History

**2026-02-04:** Investigation started
- Initial question: How to capture stack trace for stack overflow error
- Context: Extends prior investigation that identified likely locations but couldn't get trace

**2026-02-04:** Found stack trace loss location
- processor.ts:204 uses .toString() which loses .stack
- cli/error.ts has correct pattern that processor should follow

**2026-02-04:** Investigation completed
- Status: Complete
- Key outcome: Identified exact line causing stack trace loss; proposed one-line fix
