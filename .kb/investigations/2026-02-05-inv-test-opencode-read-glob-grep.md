## Summary (D.E.K.N.)

**Delta:** All OpenCode MCP file tools (Read, Glob, Grep, Write) are broken - they all return "RangeError: Maximum call stack size exceeded."

**Evidence:** Tested each tool directly; all failed with identical stack overflow error. Bash equivalents (head, find, grep, cat) work correctly.

**Knowledge:** This is a critical bug in OpenCode's MCP tool implementation - likely an infinite recursion in the tool handler.

**Next:** File bug report for OpenCode; workaround by using bash commands until fixed.

**Authority:** implementation - Bug report, no architectural impact

---

# Investigation: Test OpenCode Read/Glob/Grep Tools

**Question:** Are the OpenCode Read, Glob, and Grep MCP tools working correctly?

**Started:** 2026-02-05
**Updated:** 2026-02-05
**Owner:** Dylan (via worker agent)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Read Tool Fails with Stack Overflow

**Evidence:** 
```
mcp_read("/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md", limit=20)
→ Error: RangeError: Maximum call stack size exceeded.
```

**Source:** Direct tool invocation during this session

**Significance:** The Read tool cannot read any files. This is a fundamental failure of the MCP filesystem tool.

---

### Finding 2: Glob Tool Fails with Stack Overflow

**Evidence:**
```
mcp_glob(pattern="*.go", path="/Users/dylanconlin/Documents/personal/orch-go/cmd/orch")
→ Error: RangeError: Maximum call stack size exceeded.
```

**Source:** Direct tool invocation during this session

**Significance:** The Glob tool cannot find files by pattern. Same error as Read tool suggests shared root cause.

---

### Finding 3: Grep Tool Fails with Stack Overflow

**Evidence:**
```
mcp_grep(pattern="func main", path="/Users/dylanconlin/Documents/personal/orch-go/cmd/orch")
→ Error: RangeError: Maximum call stack size exceeded.
```

**Source:** Direct tool invocation during this session

**Significance:** The Grep tool cannot search file contents. Same error pattern confirms systemic issue.

---

### Finding 4: Write Tool Also Fails

**Evidence:**
```
mcp_write(filePath="...", content="...")
→ Error: RangeError: Maximum call stack size exceeded.
```

**Source:** Attempted to write investigation file, same error

**Significance:** The failure extends beyond read operations to write operations as well.

---

### Finding 5: Bash Equivalents Work Correctly

**Evidence:**
- `head -20 CLAUDE.md` → Returns file contents correctly
- `find cmd/orch -name "*.go"` → Returns 113 .go files
- `grep -rn "func main" cmd/orch` → Returns `cmd/orch/main.go:27:func main() {`
- `cat > file << EOF` → Successfully writes files

**Source:** Bash tool invocations during this session

**Significance:** The underlying filesystem is accessible; the problem is specifically in OpenCode's MCP tool implementation.

---

## Synthesis

**Key Insights:**

1. **All MCP file tools share the same failure mode** - The identical "Maximum call stack size exceeded" error across Read, Glob, Grep, and Write suggests a common bug in the MCP server's tool dispatch or handling code.

2. **Bash tool remains functional** - The workaround is to use bash commands (head, find, grep, cat) instead of the specialized MCP tools.

3. **This is likely an infinite recursion bug** - A JavaScript/TypeScript "Maximum call stack size exceeded" error typically indicates unbounded recursion in the code path.

**Answer to Investigation Question:**

No, the OpenCode Read, Glob, and Grep MCP tools are NOT working correctly. All tested MCP file tools fail with a stack overflow error. The workaround is to use bash equivalents.

---

## Structured Uncertainty

**What's tested:**

- ✅ Read tool fails with stack overflow (verified: called mcp_read twice, same error)
- ✅ Glob tool fails with stack overflow (verified: called mcp_glob)
- ✅ Grep tool fails with stack overflow (verified: called mcp_grep)
- ✅ Write tool fails with stack overflow (verified: called mcp_write)
- ✅ Bash alternatives work correctly (verified: head, find, grep, cat all work)

**What's untested:**

- ⚠️ Root cause of the stack overflow (requires OpenCode codebase investigation)
- ⚠️ Whether Edit tool is affected
- ⚠️ Whether this is a recent regression or longstanding bug

**What would change this:**

- Finding would be wrong if these tools work for other agents (would suggest session-specific issue)

---

## References

**Commands Run:**
```bash
# Test bash alternatives (all worked)
head -20 /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md
find /Users/dylanconlin/Documents/personal/orch-go/cmd/orch -name "*.go"
grep -rn "func main" /Users/dylanconlin/Documents/personal/orch-go/cmd/orch
```

**Expected Results (from bash):**
- CLAUDE.md: 20 lines of project documentation starting with "# orch-go"
- Glob: 113 .go files in cmd/orch/
- Grep: `cmd/orch/main.go:27:func main() {`

---

## Investigation History

**2026-02-05:** Investigation completed
- All four tested MCP tools failed with stack overflow
- Bash alternatives verified working
- Status: Complete
