**TLDR:** Question: What does an AI agent observe about the TUI interface during a session? Answer: The TUI is a terminal-based interface with markdown rendering, tool call visualization, and scrollable output - but as an AI agent, I cannot directly "see" the visual rendering, only infer from context and output. High confidence (90%) - direct observation limited by AI perception constraints.

---

# Investigation: TUI Rendering Observation

**Question:** What can an AI agent observe and describe about the OpenCode TUI interface during a session?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** og-inv-test-tui-rendering-20dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: AI agents cannot directly "see" the TUI

**Evidence:** As an AI language model, I process text input and generate text output. I don't have visual perception - I cannot see pixels, colors, or layout. What I observe is the text that flows through the conversation.

**Source:** Direct observation of my own capabilities during this session

**Significance:** This is a fundamental limitation. When asked to "describe what you see," I can only describe what I know from context (system prompts, documentation) and what I can infer from tool outputs.

---

### Finding 2: TUI characteristics inferred from context

**Evidence:** From the system context I received, I know:
- Output displayed on CLI with monospace font
- GitHub-flavored markdown rendering (CommonMark specification)
- Tool results and user messages visible
- Commands like `ctrl+p` available for actions

**Source:** System prompt at session start - explicitly states "Your output will be displayed on a command line interface"

**Significance:** The TUI is designed for CLI/terminal rendering with markdown support, not graphical visualization.

---

### Finding 3: Tool execution is observable through output

**Evidence:** When I invoke tools like `bash`, `read`, `write`, I receive structured results:
- Command outputs
- File contents
- Success/error messages

I can "see" these results as text returned to me.

**Source:** Tool invocation results during this session (e.g., `bd comment`, `kb create`, `pwd`)

**Significance:** This is the primary way agents interact with and observe the system - through tool inputs and outputs, not visual rendering.

---

## Synthesis

**Key Insights:**

1. **Perception asymmetry** - Users see a rich TUI with colors, layout, spinners. Agents see text streams and tool results.

2. **Context provides metadata** - System prompts tell agents about the TUI environment without requiring visual access.

3. **Tool output is the interface** - For agents, the "TUI" is effectively the structured tool call/response interface, not the visual presentation.

**Answer to Investigation Question:**

An AI agent can observe the TUI only indirectly - through:
1. System context describing the interface (markdown rendering, monospace font, CLI display)
2. Tool outputs that return structured data
3. User messages that may reference visual elements

The agent cannot "see" visual elements like:
- Colors or syntax highlighting
- Cursor position or focus
- Layout/spacing
- Loading spinners or animations
- Window dimensions

This is not a bug but a fundamental characteristic of how AI agents interact with systems.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Direct observation of my own capabilities combined with clear documentation about TUI characteristics.

**What's certain:**

- ✅ I cannot visually perceive the TUI rendering
- ✅ Tool outputs are my primary observation mechanism
- ✅ System context describes the TUI environment accurately

**What's uncertain:**

- ⚠️ Specific TUI features may vary by OpenCode version
- ⚠️ Some TUI elements may have accessible descriptions I haven't discovered
- ⚠️ Future AI capabilities might include visual perception

**What would increase confidence to Very High (95%+):**

- Documentation review of OpenCode TUI internals
- Testing with accessibility features that might expose visual state
- Confirmation from a human observer about what they see

---

## Implementation Recommendations

**Purpose:** N/A - This was an observational investigation, not an implementation task.

### Recommended Approach ⭐

**Accept the perception constraint** - AI agents work with text/tool interfaces, not visual rendering

**Why this approach:**
- Matches the actual architecture of AI agent interaction
- System context already provides necessary TUI metadata
- Tool outputs give agents what they need to function

**Trade-offs accepted:**
- Agents cannot debug visual rendering issues
- "Describe what you see" tasks have inherent limitations

---

## References

**Files Examined:**
- `.orch/workspace/og-inv-test-tui-rendering-20dec/SPAWN_CONTEXT.md` - Task definition and context

**Commands Run:**
```bash
# Verify working directory
pwd
# Result: /Users/dylanconlin/Documents/personal/orch-go

# Report phase to beads
bd comment orch-go-e0u "Phase: Planning - Testing TUI rendering, will describe interface observations"
# Result: Comment added

# Create investigation file
kb create investigation test-tui-rendering-say-hello
# Result: Created investigation file
```

**External Documentation:**
- None required

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-inv-test-tui-rendering-20dec/`

---

## Test Performed

**Test:** Attempted to describe the TUI interface while running as an AI agent in OpenCode

**Result:** Confirmed that AI agents cannot visually perceive the TUI. All observations come from:
1. Text in system prompts describing the environment
2. Structured tool call outputs
3. User messages

This is expected behavior - AI agents are text-based and don't have visual input channels.

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-20:** Investigation started
- Initial question: What does an AI agent observe about the TUI interface?
- Context: Spawned as test of TUI rendering observation capabilities

**2025-12-20:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: AI agents cannot visually perceive TUI - they work with text/tool interfaces
