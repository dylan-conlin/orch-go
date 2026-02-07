<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Automated Frame Gate requires two-signal detection (orchestrator session + code file edit) with tiered injection (warning → strong warning) using existing plugin infrastructure.

**Evidence:** Analyzed coaching.ts detectWorkerSession pattern (lines 894-935), injection pattern (lines 567-614), and frame collapse investigation (`.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md`).

**Knowledge:** Orchestrator sessions are detected by inverting worker detection logic; code files are distinguished from orchestration artifacts by extension and path patterns; existing injection infrastructure supports real-time coaching.

**Next:** Implement the frame gate detection in coaching.ts using the design below, test with debug session.

**Promote to Decision:** Superseded - coaching plugin disabled (2026-01-28)

---

# Investigation: Design Implement Automated Frame Gate

**Question:** How should we design and implement automated detection and intervention when orchestrators do code file edits (frame collapse)?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** og-arch-design-implement-automated-16jan-43d5
**Phase:** Complete
**Next Step:** Implementation
**Status:** Complete

---

## Findings

### Finding 1: Existing Worker Detection Pattern

**Evidence:** The coaching plugin already has robust worker session detection at lines 894-935:
```typescript
function detectWorkerSession(sessionId: string, tool: string, args: any): boolean {
  // Signal 1: bash tool with workdir in .orch/workspace/
  // Signal 2: read tool accessing SPAWN_CONTEXT.md
  // Signal 3: any tool with filePath in .orch/workspace/
}
```

Worker sessions are cached in `workerSessions` Map for performance.

**Source:** `plugins/coaching.ts:894-935`

**Significance:** Orchestrator session detection is simply the inverse - if a session is NOT detected as a worker, it's an orchestrator. This pattern is already proven and can be reused.

---

### Finding 2: Existing Injection Infrastructure

**Evidence:** The coaching plugin already injects coaching messages via:
```typescript
async function injectCoachingMessage(
  client: any,
  sessionId: string,
  patternType: "action_ratio" | "analysis_paralysis",
  details: any
): Promise<void> {
  await client.session.prompt({
    path: { id: sessionId },
    body: {
      noReply: true,
      parts: [{ type: "text", text: message }],
    },
  })
}
```

The `noReply: true` pattern prevents blocking the orchestrator workflow.

**Source:** `plugins/coaching.ts:567-614`

**Significance:** We can extend this function to support a new pattern type `frame_collapse` with appropriate messaging. Infrastructure is production-ready.

---

### Finding 3: Frame Collapse Signal - Code File Edits

**Evidence:** From the frame collapse investigation (`.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md`):

> "Reading code files is the first signal - If orchestrator opens `.rb`, `.ts`, `.css` (not `.md` in `.kb/` or `.orch/`), frame collapse has begun"

> "Track `Edit` tool usage. If edits target code files (`.go`, `.ts`, `.css`, `.py`) rather than orchestration artifacts (`.md` in `.orch/`, `.kb/`), flag after threshold (e.g., 5+ edits, >50 lines changed)."

Real incident from price-watch (2026-01-13): Orchestrator made 4 separate `Update()` calls editing controller and view files before intervention.

**Source:** `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md:326-333`

**Significance:** Edit tool on code files is the definitive signal. First edit should trigger warning, subsequent edits escalate severity.

---

### Finding 4: File Classification Logic

**Evidence:** From analyzing the incident and investigation:

**Code Files (Frame Collapse indicators):**
- `.go`, `.ts`, `.tsx`, `.js`, `.jsx` (backend/frontend)
- `.css`, `.scss`, `.less`, `.sass` (styling)
- `.py`, `.rb` (scripting)
- `.java`, `.rs`, `.c`, `.cpp`, `.h` (systems)
- `.html`, `.vue`, `.svelte` (templates)

**Orchestration Artifacts (Allowed):**
- `.md` files in `.kb/`, `.orch/`, `.beads/` directories
- `CLAUDE.md`, `SKILL.md`, `README.md` anywhere
- `SPAWN_CONTEXT.md`, `SYNTHESIS.md`, `SESSION_HANDOFF.md`
- Files in `.orch/workspace/` (worker artifacts)
- Configuration files: `.yaml`, `.json`, `.toml` in orchestration dirs

**Source:** Analysis of incident transcript + investigation recommendations

**Significance:** File classification must use both extension AND path context. A `.ts` file in `.orch/workspace/` is a worker artifact, but `.ts` in `src/` is code.

---

## Synthesis

**Key Insights:**

1. **Two-signal detection is required** - Both conditions must be true: (a) Session is orchestrator (NOT worker), AND (b) Edit tool targets a code file. Either signal alone would produce false positives.

2. **Tiered response prevents alarm fatigue** - First code edit gets warning, 3+ edits gets strong warning. This acknowledges occasional legitimate edge cases while escalating for sustained violations.

3. **Path context matters as much as extension** - A `.ts` file in `.orch/plugins/` is orchestration infrastructure, while `.ts` in `src/` is application code. Detection must consider both.

**Answer to Investigation Question:**

The Automated Frame Gate should be implemented as an extension to the existing coaching plugin with:
1. **Detection**: Track Edit tool calls in `tool.execute.after` hook, check for orchestrator session (NOT worker) + code file path
2. **Classification**: Use extension whitelist + path blacklist (orchestration directories) to identify code files
3. **Injection**: Use existing `injectCoachingMessage()` pattern with new `frame_collapse` type
4. **Thresholds**: Warning on first code edit, strong warning on 3+ edits, reset on session change

---

## Structured Uncertainty

**What's tested:**

- ✅ Worker detection pattern works reliably (coaching.ts deployed and logging correctly)
- ✅ Injection infrastructure works (action_ratio and analysis_paralysis messages fire)
- ✅ `noReply: true` pattern doesn't block orchestrator (verified in production)

**What's untested:**

- ⚠️ Edit tool args structure for file path extraction (need to verify input.args.filePath exists)
- ⚠️ Edge case: orchestrator editing plugin files (should be allowed)
- ⚠️ Edge case: orchestrator editing CLAUDE.md (should be allowed)

**What would change this:**

- Finding would be wrong if Edit tool doesn't pass filePath in args
- Finding would be wrong if code file edits are sometimes legitimate orchestrator behavior
- Finding would be wrong if injection causes loops (orchestrator reads warning, triggers more actions)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Tiered Frame Gate with Extension + Path Classification** - Add frame collapse detection to coaching.ts `tool.execute.after` hook using two-signal detection (orchestrator session + code file edit) with tiered injection response.

**Why this approach:**
- Builds on proven infrastructure (detectWorkerSession, injectCoachingMessage)
- Two-signal detection minimizes false positives
- Tiered response respects occasional legitimate edge cases
- Real-time intervention catches frame collapse as it happens (not post-hoc)

**Trade-offs accepted:**
- May occasionally warn on legitimate orchestrator edits (e.g., hotfixing a typo in skill file)
- Adds complexity to coaching plugin (already 1300 lines)
- Requires careful path classification to avoid false positives

**Implementation sequence:**
1. Add `isCodeFile(filePath: string)` classifier function
2. Add frame collapse state tracking to SessionState interface
3. Extend `tool.execute.after` hook for Edit tool detection
4. Add `frame_collapse` pattern to `injectCoachingMessage()`
5. Test with orchestrator session editing a code file

### Alternative Approaches Considered

**Option B: Post-hoc detection at session completion**
- **Pros:** Simpler, less intrusive, no false positive risk during work
- **Cons:** Detection happens after damage done, doesn't prevent frame collapse
- **When to use instead:** If real-time injection proves too disruptive

**Option C: Read tool tracking (not just Edit)**
- **Pros:** Catches frame collapse earlier (at investigation stage)
- **Cons:** Many legitimate reasons for orchestrator to read code files (context gathering)
- **When to use instead:** If Edit detection alone misses too many cases

**Rationale for recommendation:** Real-time Edit detection is the sweet spot - it's definitive (editing code = doing implementation work) while not over-triggering on legitimate context gathering via Read.

---

### Implementation Details

**What to implement first:**
1. `isCodeFile()` function with extension + path classification
2. Frame collapse state in SessionState
3. Detection in `tool.execute.after` for Edit tool

**File classification function:**
```typescript
/**
 * Determine if a file path represents code (vs orchestration artifact).
 * Returns true if editing this file indicates frame collapse.
 */
function isCodeFile(filePath: string): boolean {
  if (!filePath) return false

  const lowerPath = filePath.toLowerCase()

  // Orchestration directories - ALLOWED for orchestrators
  const orchestrationPaths = [
    '/.orch/',
    '/.kb/',
    '/.beads/',
    '/skills/',
    'claude.md',
    'skill.md',
    'readme.md',
    'spawn_context.md',
    'synthesis.md',
    'session_handoff.md',
  ]

  for (const orchPath of orchestrationPaths) {
    if (lowerPath.includes(orchPath)) {
      return false // Orchestration artifact, not code
    }
  }

  // Code file extensions - frame collapse indicators
  const codeExtensions = [
    '.go', '.ts', '.tsx', '.js', '.jsx',
    '.css', '.scss', '.less', '.sass',
    '.py', '.rb', '.java', '.rs', '.c', '.cpp', '.h',
    '.html', '.vue', '.svelte',
  ]

  for (const ext of codeExtensions) {
    if (lowerPath.endsWith(ext)) {
      return true // Code file
    }
  }

  return false // Unknown file type, not flagged
}
```

**Session state extension:**
```typescript
interface FrameCollapseState {
  codeEditCount: number          // Cumulative code file edits
  lastCodeEditPath: string | null  // Most recent code file edited
  warningInjected: boolean       // Have we warned yet?
  strongWarningInjected: boolean // Have we strongly warned yet?
}
```

**Things to watch out for:**
- ⚠️ Ensure Edit tool args have `filePath` or `file_path` (check actual schema)
- ⚠️ Plugin plugins directory (e.g., `plugins/coaching.ts`) should be in orchestration path list
- ⚠️ Worker sessions should NOT trigger frame collapse (use existing detectWorkerSession)
- ⚠️ Reset frame collapse state on new session

**Areas needing further investigation:**
- Exact Edit tool args structure (verify with debug logging)
- Whether Write tool should also trigger detection (likely yes, same logic)
- Threshold tuning (1 for warning, 3 for strong - may need adjustment)

**Success criteria:**
- ✅ Orchestrator editing a `.go` file receives frame collapse warning
- ✅ Orchestrator editing `.kb/investigations/*.md` does NOT receive warning
- ✅ Worker sessions never receive frame collapse warnings
- ✅ 3+ code edits triggers strong warning with escalation language

---

## Injection Message Design

**First Warning (1 code edit):**
```markdown
## ⚠️ Frame Collapse Warning

You've edited a code file: `{filePath}`

**Observation:** Orchestrators delegate implementation to workers. Editing code files directly indicates potential frame collapse.

**Consider:**
1. Is this work you should have spawned to a worker?
2. If an agent already failed, try different parameters (skill, model, --mcp)
```

**Strong Warning (3+ code edits):**
```markdown
## 🚨 Frame Collapse - Multiple Code Edits

You've now made **{count} code file edits** in this session.

**Last edited:** `{filePath}`

**This is a clear frame collapse pattern.** Orchestrators should delegate, not implement.

**Required Action:**
1. **STOP** editing code files
2. Spawn a worker with `orch spawn feature-impl "your task" --issue BEADS-ID`
3. If struggling with spawn strategy, consider `--mcp playwright` for UI work

**Why this matters:** Frame collapse wastes orchestrator capacity and bypasses quality gates (worker verification, beads tracking).
```

---

## References

**Files Examined:**
- `plugins/coaching.ts` - Current coaching plugin implementation
- `plugins/orchestrator-session.ts` - Orchestrator session detection patterns
- `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md` - Frame collapse investigation
- `.kb/investigations/2026-01-11-inv-pivot-coaching-plugin-two-frame.md` - Injection pattern reference

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-detect-orchestrator-frame-collapse-doing.md` - Original frame collapse analysis
- **Design:** `docs/designs/2026-01-10-orchestrator-coaching-plugin.md` - Coaching plugin architecture

---

## Investigation History

**2026-01-16 18:30:** Investigation started
- Initial question: How to design automated frame gate for orchestrators
- Context: Prior investigation identified frame collapse as key problem, this designs the automated detection

**2026-01-16 18:45:** Key findings documented
- Identified existing infrastructure (detectWorkerSession, injectCoachingMessage)
- Designed file classification logic
- Defined tiered response strategy

**2026-01-16 19:00:** Investigation completed
- Status: Complete
- Key outcome: Two-signal detection (orchestrator + code file) with tiered injection using existing plugin infrastructure
