<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added screenshot file verification to `VerifyVisualVerification` - now checks for actual image files in `{workspace}/screenshots/` directory, not just comment mentions.

**Evidence:** All 47 tests pass including 9 new tests for `HasScreenshotFilesInWorkspace`; function correctly detects PNG, JPG, JPEG, WEBP, and GIF files.

**Knowledge:** Screenshot directories are created by `spawn.CreateScreenshotsDir()` at `{workspace}/screenshots/`. Visual verification now has three evidence sources: beads comments, SYNTHESIS.md, and actual screenshot files.

**Next:** Close issue - implementation complete and tested.

**Promote to Decision:** recommend-no (tactical feature addition, follows existing pattern)

---

# Investigation: Feature Add Screenshot File Verification

**Question:** How to verify actual screenshot files exist instead of just checking for "screenshot" keyword mentions?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent og-feat-feature-add-screenshot-08jan-dcbf
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Screenshots directory structure already exists

**Evidence:** `CreateScreenshotsDir()` in `pkg/spawn/context.go:28-36` creates a `screenshots/` subdirectory in each workspace.

**Source:** 
- `pkg/spawn/context.go:28-36`
- Tests in `pkg/spawn/context_test.go:465-500`

**Significance:** The infrastructure for storing screenshots already exists. Agents that capture screenshots via Playwright MCP should save them here.

---

### Finding 2: Visual verification only checked keyword mentions

**Evidence:** `VerifyVisualVerificationWithComments` checked three sources:
1. Beads comments (for keywords like "screenshot", "visual verification")
2. SYNTHESIS.md Evidence section (same keywords)
3. SYNTHESIS.md TLDR section (same keywords)

But no check for actual screenshot files on disk.

**Source:** `pkg/verify/visual.go:402-520`

**Significance:** An agent could claim "screenshot captured" without actually capturing one, or the screenshot could fail to save. Verifying actual files provides stronger evidence.

---

### Finding 3: Standard image extensions for screenshots

**Evidence:** Common screenshot formats are PNG (most common), JPG/JPEG, WEBP, and GIF. These are the formats output by:
- Playwright MCP `browser_take_screenshot`
- Browser DevTools
- macOS screenshot utility
- Glass CLI (`glass screenshot`)

**Source:** Industry standard, verified against Playwright MCP documentation

**Significance:** Supporting these five extensions covers essentially all screenshot capture tools.

---

## Synthesis

**Key Insights:**

1. **Evidence hierarchy strengthened** - Actual files are stronger evidence than keyword mentions. An agent saying "screenshot captured" might have failed to save; a file proves it exists.

2. **Non-breaking addition** - This adds a new evidence source without removing existing ones. If comments mention screenshots OR files exist, evidence is found.

3. **Seamless integration** - The implementation slots into existing verification flow at the same point where SYNTHESIS.md is checked.

**Answer to Investigation Question:**

Implemented `HasScreenshotFilesInWorkspace()` which scans the workspace's `screenshots/` directory for image files (.png, .jpg, .jpeg, .webp, .gif). When screenshot files are found, they're added to the Evidence field with "Screenshot file: {filename}" entries. This provides concrete file-based verification alongside existing keyword-based checks.

---

## Structured Uncertainty

**What's tested:**

- ✅ Empty workspace path returns false (verified: unit test passes)
- ✅ Non-existent workspace returns false (verified: unit test passes)
- ✅ Empty screenshots directory returns false (verified: unit test passes)
- ✅ Finds PNG, JPG, JPEG, WEBP, GIF files (verified: unit tests pass)
- ✅ Ignores non-image files (verified: unit test passes)
- ✅ Ignores subdirectories (verified: unit test passes)
- ✅ Case-insensitive extension matching (verified: unit test passes)
- ✅ Integration with VerifyVisualVerification flow (verified: go build succeeds, existing tests pass)

**What's untested:**

- ⚠️ End-to-end flow where agent saves screenshot via Playwright MCP (requires real agent spawn)
- ⚠️ Performance with many screenshot files (should be negligible - just directory listing)

**What would change this:**

- Finding would be wrong if Playwright MCP saves screenshots in a different location than `{workspace}/screenshots/`
- Finding would be wrong if screenshot extensions vary (but core formats are covered)

---

## Implementation Recommendations

**Purpose:** This section documents the completed implementation.

### Recommended Approach ⭐

**Add file-based screenshot verification** - Implemented `HasScreenshotFilesInWorkspace()` function that scans workspace screenshots directory for image files.

**Why this approach:**
- Provides concrete evidence (file exists) vs soft evidence (keyword mentioned)
- Non-breaking: adds to existing evidence sources, doesn't replace them
- Simple implementation: just directory listing with extension filtering

**Trade-offs accepted:**
- Doesn't verify image content (could be 0-byte or corrupted) - acceptable because presence is the gate
- Doesn't track which screenshot is for which verification - acceptable because any screenshot is evidence of visual work

**Implementation sequence:**
1. Added `HasScreenshotFilesInWorkspace()` function - core detection logic
2. Integrated into `VerifyVisualVerificationWithComments()` - adds to evidence sources
3. Added comprehensive tests - 9 new test cases plus 2 integration tests

---

## References

**Files Examined:**
- `pkg/verify/visual.go` - Core visual verification logic
- `pkg/verify/visual_test.go` - Existing tests for visual verification
- `pkg/spawn/context.go` - CreateScreenshotsDir function

**Commands Run:**
```bash
# Find screenshot directories
find .orch/workspace -name "screenshots" -type d

# Run tests
go test ./pkg/verify/... -v
go build ./...
```

**Related Artifacts:**
- **Prior constraint:** "Agents modifying web/ files MUST capture visual verification via Playwright MCP before completing"

---

## Investigation History

**2026-01-08 18:10:** Investigation started
- Initial question: How to verify actual screenshot files exist instead of just comment mentions?
- Context: Spawned from beads issue orch-go-bdtyl

**2026-01-08 18:15:** Found existing infrastructure
- Screenshots directory created by spawn system
- Visual verification only checked keyword mentions

**2026-01-08 18:25:** Implementation complete
- Added HasScreenshotFilesInWorkspace function
- Integrated into VerifyVisualVerificationWithComments
- All 47 tests pass

**2026-01-08 18:30:** Investigation completed
- Status: Complete
- Key outcome: Screenshot file verification now integrated into visual verification flow
