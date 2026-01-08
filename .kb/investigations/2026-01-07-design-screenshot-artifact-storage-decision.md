<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Screenshots should be stored in `.orch/workspace/{agent}/screenshots/` with metadata JSON, not as loose files - this enables verification at `orch complete` time.

**Evidence:** Verify package checks for MENTIONS of screenshots but cannot verify files exist; Glass CLI defaults to cwd; Playwright MCP returns base64 (ephemeral); no central registry exists.

**Knowledge:** Without canonical storage, "visual verification evidence" is just pattern matching in comments - agents can claim screenshots exist without proof. File-based evidence enables verification.

**Next:** Implement workspace-scoped screenshot storage with metadata; add verification to check files exist at `orch complete`.

**Promote to Decision:** recommend-yes - Establishes artifact storage pattern applicable beyond screenshots.

---

# Investigation: Screenshot Artifact Storage Decision

**Question:** Where should screenshots be stored, how should they be referenced, what's the lifecycle/cleanup, and who owns them?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete
**Confidence:** High (85%)

---

## Problem Framing

### Design Question

Screenshots are produced by 3 disconnected systems with no canonical storage, referencing, or lifecycle:
1. **Playwright MCP** - Returns base64 image data in tool response (ephemeral)
2. **Glass CLI** - Writes to specified path or `screenshot.png` in cwd
3. **Glass MCP** - Returns base64 image data in tool response (ephemeral)
4. **User/Manual** - Any arbitrary location

The verify package (`pkg/verify/visual.go`) checks for screenshot MENTIONS in beads comments but can't verify actual files exist. This enables agents to claim "screenshot captured" without proof.

### Success Criteria

1. Screenshots have a canonical storage location discoverable by `orch complete`
2. Verification can confirm screenshot files actually exist (not just mentions)
3. Lifecycle is clear - screenshots cleaned up when workspace archives
4. Ownership is agent-scoped (not project-wide pollution)
5. Works with existing Glass CLI and can extend to Playwright

### Constraints

- Must not break existing Glass CLI usage (glass screenshot -o path)
- Must not require modifying how Playwright MCP works (returns base64)
- Storage must be per-agent (not global) to avoid conflicts
- Should integrate with existing workspace lifecycle

### Scope

**In scope:**
- Storage location design
- Referencing mechanism (how to find screenshots)
- Lifecycle (creation, discovery, cleanup)
- Integration with `orch complete` verification

**Out of scope:**
- Screenshot comparison/diff tooling
- Automated visual regression testing
- Modifying Playwright MCP internals

---

## Findings

### Finding 1: Verify Package Checks Mentions, Not Files

**Evidence:** `pkg/verify/visual.go:82-107` defines `visualEvidencePatterns` that match strings like "screenshot", "browser_take_screenshot", "visual verification" in beads comments. The function `HasVisualVerificationEvidence` returns true if ANY pattern matches ANY comment text.

```go
var visualEvidencePatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)screenshot`),
    regexp.MustCompile(`(?i)browser_take_screenshot`),
    // ... more patterns
}
```

An agent can write `bd comment <id> "Visual verification: screenshot captured"` without actually capturing anything and pass verification.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/visual.go:82-107, 228-246`

**Significance:** Current verification is theater - it checks for words, not artifacts. File-based verification would provide actual proof.

---

### Finding 2: Glass CLI Writes to Arbitrary Paths

**Evidence:** Glass CLI screenshot command accepts `-o` flag for output path, defaulting to `screenshot.{format}` in current working directory:

```go
// From main.go:362-378
screenshotFlags := flag.NewFlagSet("screenshot", flag.ExitOnError)
outputPath := screenshotFlags.String("o", "", "Output file path (default: screenshot.png)")
// ...
outPath := *outputPath
if outPath == "" {
    outPath = "screenshot." + *format
}
```

This means screenshots land wherever the agent happens to be, with no central tracking.

**Source:** `/Users/dylanconlin/Documents/personal/glass/main.go:361-416`

**Significance:** Without a convention for where screenshots go, discovery is impossible. Glass CLI is the easiest integration point - just need to establish a convention.

---

### Finding 3: Glass/Playwright MCP Return Base64 (Ephemeral)

**Evidence:** Both Glass MCP and Playwright MCP return screenshots as base64-encoded image data in the tool response:

```go
// From glass/pkg/mcp/server.go:685-692
return &mcp.CallToolResult{
    Content: []mcp.Content{
        mcp.ImageContent{
            Type:     "image",
            Data:     base64Data,  // base64 encoded
            MIMEType: mimeType,
        },
    },
}, nil
```

This data exists only in the Claude conversation context - it's never persisted to disk unless the agent explicitly saves it.

**Source:** `/Users/dylanconlin/Documents/personal/glass/pkg/mcp/server.go:630-694`

**Significance:** MCP tool screenshots are ephemeral by design. To persist them, agents must explicitly decode and write. This is a feature (Claude can view inline) but means no automatic persistence.

---

### Finding 4: Workspaces Have Defined Structure

**Evidence:** Each agent has a workspace at `.orch/workspace/{name}/` containing:
- `SPAWN_CONTEXT.md` - Spawn configuration
- `SYNTHESIS.md` - Work summary (full tier only)
- `.spawn_time` - Timestamp for scoping git operations

Workspaces are archived to `.orch/workspace/archived/` when cleaned up.

**Source:** `ls -la /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/`

**Significance:** The workspace already serves as the canonical location for agent artifacts. Screenshots are a natural fit as another artifact type.

---

### Finding 5: No Glass Patterns in Visual Verification

**Evidence:** `visualEvidencePatterns` includes playwright patterns (`browser_take_screenshot`, `browser_navigate`) but not glass patterns. The prior investigation (2025-12-27-inv-glass-integration-status-orch-ecosystem.md) noted this gap.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/visual.go:95-98`

**Significance:** Even if we fix storage, verification still won't recognize Glass tool usage. This is a separate but related fix needed.

---

## Synthesis

**Key Insights:**

1. **Verification requires artifacts, not just claims** - The current pattern-matching approach allows agents to claim verification without proof. Moving to file-based verification provides actual evidence.

2. **Workspace is the natural home** - Agent workspaces already contain SPAWN_CONTEXT.md and SYNTHESIS.md. Adding `screenshots/` follows the same pattern and inherits workspace lifecycle (archived together).

3. **CLI integration is simpler than MCP** - Glass CLI already writes files; just need convention. MCP returns base64 requiring agent action to save - harder to enforce.

4. **Metadata enables verification** - A metadata file alongside screenshots can record when/why captured, enabling `orch complete` to verify specific screenshots exist.

**Answer to Investigation Question:**

Screenshots should be stored in `.orch/workspace/{agent}/screenshots/` with an accompanying `screenshot-manifest.json` metadata file. This provides:

| Question | Answer |
|----------|--------|
| **Where to store?** | `.orch/workspace/{agent}/screenshots/` |
| **How to reference?** | `screenshot-manifest.json` lists all screenshots with timestamps/context |
| **Lifecycle/cleanup?** | Archived with workspace; deleted when `orch clean` runs |
| **Ownership?** | Agent that created the workspace owns all screenshots in it |

---

## Structured Uncertainty

**What's tested:**

- ✅ Visual verification uses pattern matching on comments (verified: read pkg/verify/visual.go)
- ✅ Glass CLI writes to specified path or cwd (verified: read glass/main.go)
- ✅ Glass MCP returns base64 (verified: read glass/pkg/mcp/server.go)
- ✅ Workspaces have consistent structure (verified: ls workspace directories)

**What's untested:**

- ⚠️ Agent compliance with screenshot convention (not enforced yet)
- ⚠️ Performance impact of file-based verification (not benchmarked)
- ⚠️ Whether base64→file saving adds significant friction for agents (not user-tested)

**What would change this:**

- If MCP protocol evolves to support file saving natively
- If Glass tools get workspace-awareness built in
- If a central artifact store (outside workspace) proves better for cross-agent references

---

## Implementation Recommendations

### Recommended Approach ⭐

**Workspace-Scoped Screenshot Storage with Manifest** - Store screenshots in `.orch/workspace/{agent}/screenshots/` with a `screenshot-manifest.json` tracking all captures.

**Why this approach:**
- Follows existing workspace pattern (SPAWN_CONTEXT.md, SYNTHESIS.md)
- Enables file-based verification at `orch complete`
- Auto-cleanup via workspace archival
- Agent-scoped to prevent conflicts

**Trade-offs accepted:**
- Agents must explicitly save to workspace path (not automatic)
- MCP screenshots require extra decode+write step
- No cross-agent screenshot sharing (acceptable - each agent verifies their own work)

**Implementation sequence:**
1. Add `screenshots/` directory support to workspace structure
2. Add Glass CLI `--workspace` flag or environment variable for auto-pathing
3. Update `pkg/verify/visual.go` to check for screenshot files
4. Add Glass patterns to `visualEvidencePatterns`
5. Update feature-impl skill to guide screenshot storage

### Alternative Approaches Considered

**Option B: Central Screenshot Store**
- **Pros:** Single location, easier to browse all screenshots
- **Cons:** No ownership, harder cleanup, conflicts if same agent name reused
- **When to use instead:** If screenshots need to be shared across agents or projects

**Option C: Inline-Only (Keep Status Quo)**
- **Pros:** No storage overhead, screenshots in Claude context
- **Cons:** Can't verify at completion time, ephemeral evidence
- **When to use instead:** If verification isn't actually needed (but it is)

**Rationale for recommendation:** Workspace-scoping follows existing patterns, provides clear ownership, and enables the verification we need without overcomplicating the system.

---

### Implementation Details

**What to implement first:**
1. Add `glass_*` patterns to `visualEvidencePatterns` (quick fix, no dependencies)
2. Create workspace screenshots directory structure support
3. Add screenshot file verification to `VerifyVisualVerification`

**File targets:**
- `pkg/verify/visual.go` - Add glass patterns, add file-based verification
- `pkg/spawn/workspace.go` - Add screenshots directory creation
- Feature-impl skill docs - Add screenshot storage guidance

**Screenshot manifest format:**
```json
{
  "screenshots": [
    {
      "filename": "dashboard-stats-bar.png",
      "captured_at": "2026-01-07T21:15:00Z",
      "context": "Visual verification of stats bar after implementation",
      "tool": "glass_screenshot",
      "url": "http://localhost:5188/"
    }
  ]
}
```

**Things to watch out for:**
- ⚠️ Glass CLI may be run outside workspace context - need fallback behavior
- ⚠️ Base64 screenshots from MCP need agent instruction to save
- ⚠️ Don't break existing pattern-based verification (additive change)

**Areas needing further investigation:**
- Whether to add `glass screenshot --workspace <path>` flag
- Whether manifest should be auto-generated or agent-maintained
- Screenshot naming conventions (timestamp vs descriptive)

**Success criteria:**
- ✅ `orch complete` can verify screenshot files exist in workspace
- ✅ Glass tool usage detected by visual verification patterns
- ✅ Screenshots cleaned up when workspace archives
- ✅ Feature-impl agents have clear guidance on where to save screenshots

---

## Feature List Review

**Validation of existing items:** N/A (this investigation doesn't modify features.json directly)

**New items to add (for orchestrator):**
1. `feature-impl: Add glass_* patterns to visual verification` - Ready for implementation
2. `feature-impl: Add screenshot file verification to VerifyVisualVerification` - Depends on storage design
3. `feature-impl: Add workspace screenshots/ directory support` - Foundational

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/visual.go` - Visual verification logic
- `/Users/dylanconlin/Documents/personal/glass/main.go` - Glass CLI screenshot handling
- `/Users/dylanconlin/Documents/personal/glass/pkg/mcp/server.go` - Glass MCP screenshot tool
- `.kb/investigations/2025-12-26-inv-evaluate-snap-cli-integration-visual.md` - Prior snap/playwright comparison
- `.kb/investigations/2025-12-27-inv-design-ui-validation-gate-system.md` - Prior UI validation design
- `.kb/investigations/2025-12-27-inv-glass-integration-status-orch-ecosystem.md` - Glass integration status

**Commands Run:**
```bash
# Check existing screenshots
find /Users/dylanconlin/Documents/personal/orch-go -name "*.png" | head -30

# List workspace structure
ls -la /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/

# Search for screenshot patterns
grep -n "screenshot\|Screenshot" pkg/verify/visual.go
```

**Related Artifacts:**
- **Decision:** kn-cc1c45 - "MCP for agent-internal use, CLI for orchestrator/scripts/humans"
- **Investigation:** 2025-12-27-inv-design-ui-validation-gate-system.md - UI validation gate design

---

## Investigation History

**2026-01-07 21:11:** Investigation started
- Initial question: Where to store screenshots, how to reference, lifecycle, ownership?
- Context: Screenshots produced by 3 disconnected systems with no canonical storage

**2026-01-07 21:20:** Problem framing complete
- Identified 3 producers: Playwright MCP, Glass CLI, Glass MCP
- Identified verification gap: pattern matching on comments, not files

**2026-01-07 21:35:** Exploration complete
- Found verify checks mentions not files
- Found Glass CLI writes to arbitrary paths
- Found MCP returns base64 (ephemeral)
- Found workspace structure is natural home

**2026-01-07 21:45:** Investigation completed
- Status: Complete
- Key outcome: Workspace-scoped storage with manifest enables file-based verification
