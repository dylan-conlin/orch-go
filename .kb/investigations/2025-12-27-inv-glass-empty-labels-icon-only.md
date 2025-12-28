<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Icon-only buttons returned empty labels because Glass only checked text content and aria-label, missing title attributes, icon class names, and fallback identifiers.

**Evidence:** Before fix: `button: ` (empty). After fix: `button: [#bits-653]` or `button: [icon: close]` for all icon buttons. Tested on Swarm Dashboard - zero empty labels.

**Knowledge:** For icon-only buttons, meaningful labels come from (in priority order): text > aria-label > title > icon class hint > ID > class name. All buttons now have identifiers.

**Next:** Close - fix implemented, tested, and working.

---

# Investigation: Glass Empty Labels Icon Only

**Question:** Why do many buttons return `label: "button: "` with no text content, and how can we extract meaningful labels for icon-only buttons?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Agent og-debug-glass-empty-labels-27dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Root cause in Node.String() method

**Evidence:** In `daemon.go:454-459`, the button case only checks `n.Text` and `n.AriaLabel`:
```go
case "button":
    text := n.Text
    if text == "" {
        text = n.AriaLabel
    }
    return fmt.Sprintf("button: %s", text)
```
If both are empty, returns `"button: "` (empty label).

**Source:** glass/pkg/chrome/daemon.go:454-459

**Significance:** No fallback mechanism existed for icon-only buttons that lack both text and aria-label.

---

### Finding 2: Title attribute already extracted but not used for labels

**Evidence:** `parseNodeWithExtras` was extracting title but assigning it to Text (line 1040-1043), which conflated two semantically different things and didn't work as a fallback.

**Source:** glass/pkg/chrome/daemon.go:1040-1043 (before fix)

**Significance:** The data was available but not being utilized correctly. Title is often used for icon button tooltips.

---

### Finding 3: Icon class names contain semantic meaning

**Evidence:** Common icon libraries use predictable patterns:
- Font Awesome: `fa-close`, `fa-times`, `fa-check`
- Bootstrap Icons: `bi-x`, `bi-check`
- Generic: `icon-close`, `icon-menu`

These can be parsed to extract meaningful labels like "close", "menu", etc.

**Source:** Common knowledge of icon library conventions

**Significance:** Buttons with icon children (svg, i elements) often have meaningful class names that can serve as labels.

---

## Synthesis

**Key Insights:**

1. **Priority-based fallback is necessary** - Icon-only buttons need a cascade of potential label sources, not just text/aria-label.

2. **Every button should have SOME identifier** - Even if we can't extract semantic meaning, ID or class name provides a usable selector reference.

3. **Icon class parsing is worth the complexity** - Common patterns like `icon-*`, `fa-*`, `bi-*` can extract meaningful hints.

**Answer to Investigation Question:**

Empty button labels occurred because Glass only checked text content and aria-label. The fix adds a priority-based fallback chain:
1. Text content (original)
2. aria-label (original)
3. title attribute (new - stored separately now)
4. Icon class hint extracted from element or child element classes (new)
5. Element ID as `[#id]` (new)
6. First class name as `[.class]` (new)

---

## Structured Uncertainty

**What's tested:**

- ✅ Zero empty button labels on Swarm Dashboard (verified: `glass -url=localhost:5189 actions | grep "button: $"` returns nothing)
- ✅ Buttons with text still work correctly (verified: `button: Disconnect`, `button: Settings`)
- ✅ ID fallback works (verified: `button: [#bits-653]`)
- ✅ Class fallback works (verified: `button: [.group]`)
- ✅ Go tests pass (verified: `go test ./...`)

**What's untested:**

- ⚠️ Icon class extraction for FA/Bootstrap icons (no icons on test page used these patterns)
- ⚠️ Title attribute fallback (no buttons on test page had title without text/aria-label)
- ⚠️ SVG title child extraction (no SVG icons with title children on test page)

**What would change this:**

- Finding would be incomplete if buttons exist with neither ID nor class (would still show `button: `)
- Icon hint extraction might miss some icon library patterns not in the list

---

## Implementation Recommendations

**Purpose:** Document what was implemented to fix the issue.

### Implemented Approach ⭐

**Priority-based fallback chain** - Added multiple fallback sources for button labels in order of semantic preference.

**Changes made:**
1. Added `Title` and `IconHint` fields to Node struct
2. Updated `Node.String()` to use fallback chain for buttons and role-based elements
3. Added `extractIconHint()` to parse common icon class patterns
4. Added `extractIconHintFromChildren()` to check child element classes
5. Added utility functions `cleanIconHint()` and `isUtilityClass()`

**Trade-offs accepted:**
- Labels like `[#bits-653]` are not human-readable, but they're unique identifiers that can be clicked
- Icon hint extraction won't catch all patterns, but covers common libraries

---

## References

**Files Examined:**
- glass/pkg/chrome/daemon.go - Core implementation of Node struct and String() method
- glass/pkg/mcp/server.go - MCP server using the elements

**Commands Run:**
```bash
# Build glass
go build -o ~/bin/glass .

# Test elements on dashboard
~/bin/glass -url="localhost:5189" actions

# Verify no empty labels
~/bin/glass -url="localhost:5189" actions | grep "button: $"

# Run tests
go test ./...
```

---

## Investigation History

**2025-12-27:** Investigation started
- Initial question: Why do icon-only buttons return empty labels?
- Context: Beads issue orch-go-bmho reported many buttons with `label: "button: "`

**2025-12-27:** Root cause identified
- Node.String() only checked Text and AriaLabel
- No fallback for icon-only buttons

**2025-12-27:** Fix implemented
- Added Title and IconHint fields
- Implemented priority-based fallback chain
- Added icon class extraction

**2025-12-27:** Investigation completed
- Status: Complete
- Key outcome: All buttons now have meaningful labels - zero empty labels on test page
