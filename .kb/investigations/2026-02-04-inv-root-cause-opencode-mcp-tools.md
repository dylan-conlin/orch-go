<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Stack overflow affects read/glob/grep (NOT MCP tools - they're built-in), while bash works. Likely caused by Zod v4 or AI SDK tool wrapper, not the tool implementations themselves which are simple.

**Evidence:** Tool schemas are simple flat objects (verified). Zod v4.1.8 has documented stack overflow issues (GitHub #4994). Bash works in same session where read fails (session file evidence). OpenCode server not running to get stack trace.

**Knowledge:** Issue is structural (persists across restarts), NOT in tool code itself. Possible locations: toJSONSchema (prompt.ts:701), InstructionPrompt.resolve (read.ts:63), or AI SDK wrapper. Need stack trace to pinpoint.

**Next:** When OpenCode server available: enable DEBUG logging, reproduce error, capture full stack trace to identify exact recursion location.

**Authority:** architectural - Issue spans OpenCode fork, zod library, and AI SDK - needs Dylan to decide fix approach

---

# Investigation: Root Cause OpenCode MCP Tools Stack Overflow

**Question:** What causes OpenCode built-in tools (read, glob, grep) to fail with "Maximum call stack size exceeded" when server restart doesn't fix it?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Investigation worker
**Phase:** Synthesizing
**Next Step:** Enable DEBUG logging and capture stack trace when OpenCode server is running
**Status:** Paused - need running OpenCode server to reproduce

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| GitHub colinhacks/zod#4994 | related | yes | - |
| GitHub jlowin/fastmcp#93 | related | yes | Different cause (image encoding) |
| GitHub anomalyco/opencode#10202 | related | no | Different issue (web UI) |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Issue Title is Misleading - These Are Built-In Tools, Not MCP Tools

**Evidence:** Examined tool definitions in OpenCode source:
- read.ts: `Tool.define("read", {...})` - built-in tool
- glob.ts: `Tool.define("glob", {...})` - built-in tool
- grep.ts: `Tool.define("grep", {...})` - built-in tool
- MCP tools are loaded separately via `MCP.tools()` at prompt.ts:734

**Source:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/read.ts`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/glob.ts`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/grep.ts`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt.ts:696-732`

**Significance:** Clarifies that the issue is with OpenCode's core tool infrastructure, not the MCP integration. This narrows the investigation scope.

---

### Finding 2: OpenCode Uses Zod v4.1.8 Which Has Known Stack Overflow Issues

**Evidence:**
- `package.json:zod": "4.1.8"` in OpenCode root
- GitHub issue colinhacks/zod#4994 documents "Maximum call stack size exceeded" in zod v4
- Issue occurs during schema initialization with complex recursive structures
- Stack trace shows error in `escapeRegex` during `ZodLiteral` initialization

**Source:**
- `~/Documents/personal/opencode/package.json`
- https://github.com/colinhacks/zod/issues/4994

**Significance:** Confirms Zod v4 has documented issues that could cause this exact error. The fix mentioned in the issue (commit 9bdbc2f) may not cover all cases.

---

### Finding 3: Stack Overflow Occurs at z.toJSONSchema() During Tool Setup

**Evidence:** In prompt.ts line 701:
```typescript
const schema = ProviderTransform.schema(input.model, z.toJSONSchema(item.parameters))
```
This is called for EACH tool during resolveTools(). If any tool's parameter schema causes infinite recursion during JSON schema conversion, the error would occur.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt.ts:701`

**Significance:** This explains why server restart doesn't fix it - the code structure causes the recursion, not accumulated state. The error happens every time the tools are set up.

---

### Finding 4: Bash Tool Works Because It Has Simpler Schema Structure

**Evidence:** Compared tool definitions:
- bash.ts: Uses `Tool.define("bash", async () => {...})` - async function form
- read/glob/grep: Use `Tool.define("name", {...})` - direct object form

However, both forms should work per tool.ts implementation. The key difference is likely in the schema complexity:
- bash: Simple parameters (command, timeout, workdir, description)
- read: Has InstructionPrompt integration with ctx.messages
- glob/grep: Use Ripgrep module

**Source:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/bash.ts`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/tool.ts`

**Significance:** The tool definition form doesn't explain the difference. Need to investigate schema structure further.

---

## Synthesis

**Key Insights:**

1. **Misnomer: These are built-in tools, NOT MCP tools** - The issue title is misleading. read, glob, and grep are OpenCode's core built-in tools, not MCP (Model Context Protocol) tools. This changes where to look for fixes.

2. **Server restart not fixing points to code structure, not state** - Since the error persists across restarts, it's likely a structural issue in how tools are initialized or called, not accumulated runtime state.

3. **Bash works but read/glob/grep don't - the difference is subtle** - All tools use similar infrastructure (ctx.ask, permission system), so the difference must be in something specific to file-reading tools. Possible causes: InstructionPrompt.resolve iteration, assertExternalDirectory, or the file path handling.

4. **Zod v4 has documented stack overflow issues** - OpenCode uses zod 4.1.8, and GitHub issue colinhacks/zod#4994 documents similar "Maximum call stack size exceeded" errors in v4 with certain schema patterns.

**Answer to Investigation Question:**

The root cause is LIKELY in one of these areas (in order of probability):

1. **Zod v4's toJSONSchema conversion** (prompt.ts:701) - This happens for all tools during setup, but certain schemas might trigger the issue intermittently based on complex provider transformations.

2. **InstructionPrompt.resolve iteration** (read.ts:63) - Only read tool uses this, but it might indicate a pattern where ctx.messages contains circular references.

3. **AI SDK tool wrapper** - The vercel/ai SDK wraps tools and might have issues with certain parameter patterns.

**Limitation:** Could not reproduce live (OpenCode server not running). Need to test hypotheses when server is active.

---

## Structured Uncertainty

**What's tested:**

- ✅ Tool implementations are straightforward (verified: read source for read.ts, glob.ts, grep.ts, bash.ts)
- ✅ Zod schemas for these tools are simple flat objects (verified: grep for z.object patterns)
- ✅ OpenCode uses zod 4.1.8 (verified: cat package.json)
- ✅ GitHub has documented zod v4 stack overflow issues (verified: web search)
- ✅ Bash tool works in same session where read/glob/grep fail (verified: session file evidence)

**What's untested:**

- ⚠️ Whether toJSONSchema call actually causes the overflow (need stack trace)
- ⚠️ Whether messages array has circular references (need runtime debugging)
- ⚠️ Whether the issue is in AI SDK's tool wrapper vs OpenCode code
- ⚠️ Whether reverting zod version would fix the issue

**What would change this:**

- Full stack trace would pinpoint exact location of overflow
- If issue only occurs with specific model/provider, it's in ProviderTransform
- If issue occurs with fresh session (no messages), it's in tool setup not execution

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Get stack trace with DEBUG logging | implementation | Diagnostic step that doesn't change behavior |
| Check zod version/upgrade path | architectural | Affects OpenCode fork, needs cross-version testing |
| Consider AI SDK issue | architectural | May need upstream fix or workaround |

### Recommended Approach ⭐

**Get stack trace to pinpoint exact failure location** - Enable DEBUG logging in OpenCode and reproduce the error to see where in the call stack the overflow occurs.

**Why this approach:**
- Current investigation narrowed down to 3 possible areas but can't pinpoint without stack trace
- Stack trace will definitively show if it's zod, AI SDK, or OpenCode code
- Non-invasive diagnostic step before any code changes

**Trade-offs accepted:**
- Requires reproducing the issue (need working OpenCode server)
- Stack traces can be large and hard to parse

**Implementation sequence:**
1. Start OpenCode with DEBUG=* or equivalent logging
2. Call a failing tool (read, glob, or grep)
3. Capture the full stack trace from the error
4. Identify which function is recurring infinitely

### Alternative Approaches Considered

**Option B: Downgrade zod version**
- **Pros:** Quick fix if zod is the cause
- **Cons:** May break other functionality; zod 3 -> 4 had breaking changes
- **When to use instead:** If stack trace confirms zod as culprit and no other fix available

**Option C: Simplify tool schemas**
- **Pros:** Reduces complexity that could trigger overflow
- **Cons:** Loses helpful descriptions and constraints
- **When to use instead:** If specific schema patterns identified as causing issue

**Rationale for recommendation:** Need to confirm root cause before fixing. Debugging is lower risk than code changes.

---

### Next Steps

**Immediate (when OpenCode server running):**
1. Enable verbose logging: `DEBUG=* opencode serve`
2. Reproduce error with read tool
3. Capture stack trace
4. Report findings to update this investigation

**Things to watch out for:**
- ⚠️ Error may be intermittent (session file shows it happened but may not always)
- ⚠️ May need to test with specific provider/model combinations
- ⚠️ Stack trace may be truncated - need full trace

**Success criteria:**
- ✅ Full stack trace showing exact location of infinite recursion
- ✅ Confirmed whether zod, AI SDK, or OpenCode code is the source
- ✅ Clear path to fix identified

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/read.ts` - Built-in read tool implementation
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/glob.ts` - Built-in glob tool implementation
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/grep.ts` - Built-in grep tool implementation
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/bash.ts` - Built-in bash tool (works fine)
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/prompt.ts` - Tool setup and execution
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/tool.ts` - Tool.define framework
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/instruction.ts` - InstructionPrompt (used by read)
- `/Users/dylanconlin/Documents/personal/orch-go/session-ses_3d50.md` - Session file with error evidence

**Commands Run:**
```bash
# Check zod version
grep -r '"zod"' package.json packages/*/package.json

# Find tool source files
find ~/Documents/personal/opencode/packages/opencode/src/tool -name "*.ts"

# Check git history
cd ~/Documents/personal/opencode && git log --oneline -20
```

**External Documentation:**
- [Zod v4 Stack Overflow Issue #4994](https://github.com/colinhacks/zod/issues/4994) - Documents similar error in zod v4
- [FastMCP Image Error #93](https://github.com/jlowin/fastmcp/issues/93) - Related but different cause (image encoding)

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-03-inv-orch-go-investigation-mcp-vs.md` - MCP vs CLI investigation (clarifies what MCP actually is)

---

## Investigation History

**2026-02-04 15:52:** Investigation started
- Initial question: What causes read/glob/grep tools to fail with stack overflow?
- Context: Issue orch-go-21282 reported the problem

**2026-02-04 16:10:** Clarified terminology
- These are NOT MCP tools - they're OpenCode's built-in tools
- MCP tools are external, configured separately

**2026-02-04 16:30:** Found zod v4 connection
- Web search revealed zod v4 has documented stack overflow issues
- OpenCode uses zod 4.1.8

**2026-02-04 16:45:** Identified possible root causes
- toJSONSchema in prompt.ts:701
- InstructionPrompt.resolve for read tool
- AI SDK tool wrapper

**2026-02-04 17:00:** Investigation paused
- Status: Paused - need to reproduce with running server
- Key outcome: Narrowed to 3 possible areas; need stack trace to confirm
