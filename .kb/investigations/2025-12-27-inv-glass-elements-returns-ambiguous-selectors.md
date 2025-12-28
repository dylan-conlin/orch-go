## Summary (D.E.K.N.)

**Delta:** Glass `elements` command now generates unique selectors using priority order (ID > name > aria-label > text > class) and adds `[nth=N]` indices for duplicates.

**Evidence:** Built and tested glass with new selector generation - tests pass, duplicate selectors now get `[nth=1]`, `[nth=2]` suffixes.

**Knowledge:** Extended selectors (`:has-text()`, `[nth=N]`) require JavaScript-based element finding since they're not valid CSS; glass Click/Type functions now parse and handle these.

**Next:** Deploy glass binary and smoke-test with actual browser automation.

---

# Investigation: Glass Elements Returns Ambiguous Selectors

**Question:** Why do multiple buttons with the same class return identical selectors, and how should glass return more specific selectors?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: generateSelector used class-first priority without uniqueness check

**Evidence:** Original `generateSelector` function at `glass/pkg/chrome/daemon.go:661-699` built selectors using:
1. ID (if present)
2. name (for form elements)
3. tag + first class (e.g., `button.px-3`)
4. Additional attributes (role, aria-label, placeholder, href)

When multiple elements had the same class and no distinguishing attributes, they all got identical selectors.

**Source:** `/Users/dylanconlin/Documents/personal/glass/pkg/chrome/daemon.go:661-699`

**Significance:** This is the root cause - no uniqueness guarantee was provided for elements with same class.

---

### Finding 2: Click/Type functions used document.querySelector which only returns first match

**Evidence:** The `Click` and `Type` functions used `document.querySelector(selector)` which always returns the first matching element, making it impossible to target subsequent matches.

**Source:** `/Users/dylanconlin/Documents/personal/glass/pkg/chrome/daemon.go:1078-1150` (original)

**Significance:** Even with index hints, the old implementation couldn't use them - needed JavaScript-based element finding with `querySelectorAll`.

---

### Finding 3: Text content was available but not used for selectors

**Evidence:** The `Node` struct has a `Text` field populated from child text nodes, but `generateSelector` didn't use it. For buttons, text content like "Submit" or "Cancel" is often unique and descriptive.

**Source:** `/Users/dylanconlin/Documents/personal/glass/pkg/chrome/daemon.go:427-444`

**Significance:** Using text content for buttons provides more meaningful and often unique selectors.

---

## Synthesis

**Key Insights:**

1. **Selector priority matters** - More unique identifiers (ID, aria-label, text content) should take precedence over class names which are often reused.

2. **Post-processing for duplicates** - After generating initial selectors, a second pass to count and index duplicates ensures uniqueness.

3. **Extended selectors require JavaScript** - Pure CSS can't select by text content or by nth-match globally. JavaScript-based element finding is necessary.

**Answer to Investigation Question:**

The original implementation generated selectors in a fixed priority order that didn't prioritize uniqueness. Multiple elements with the same class got identical selectors like `button.px-3`. The fix implements:

1. **New priority order:** ID > name > aria-label > text content > placeholder > href > class
2. **Text-based selectors:** Buttons now use `:has-text("text")` syntax when text content is available
3. **Duplicate disambiguation:** After generating all selectors, duplicates get `[nth=1]`, `[nth=2]` suffixes
4. **Extended selector support:** Click/Type functions now parse and handle `:has-text()` and `[nth=N]` using JavaScript

---

## Structured Uncertainty

**What's tested:**

- Unit tests pass for generateSelector with new priority order
- Unit tests pass for disambiguateSelectors with duplicate detection
- Unit tests pass for escapeCSS and containsSpecialChars helpers
- Build succeeds with no errors

**What's untested:**

- End-to-end browser automation with actual duplicate buttons
- Performance impact of querySelectorAll vs querySelector
- Edge cases with very long text content or special characters in selectors

**What would change this:**

- If `:has-text()` doesn't match elements as expected in browser
- If performance degrades significantly with many elements

---

## Implementation Recommendations

### Recommended Approach (Implemented)

**Multi-strategy selector generation with duplicate disambiguation**

**Why this approach:**
- Provides meaningful selectors that describe elements (text content for buttons)
- Falls back gracefully when unique identifiers aren't available
- Guaranteed uniqueness via [nth=N] suffix for duplicates

**Trade-offs accepted:**
- Extended selectors (`:has-text()`, `[nth=N]`) aren't valid CSS - require JavaScript handling
- Slightly more complex Click/Type implementation

**Implementation sequence:**
1. Enhanced generateSelector with new priority order
2. Added disambiguateSelectors post-processing pass
3. Updated Click/Type to parse extended selectors and use querySelectorAll+filtering

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/glass/pkg/chrome/daemon.go` - Main implementation
- `/Users/dylanconlin/Documents/personal/glass/pkg/chrome/daemon_test.go` - Tests

**Commands Run:**
```bash
# Build check
/usr/local/go/bin/go build -o /dev/null .

# Run tests
/usr/local/go/bin/go test ./...
```

---

## Investigation History

**2025-12-27:** Investigation started
- Initial question: Why do multiple buttons with same class return identical selectors?
- Context: Spawned from beads issue orch-go-5fni

**2025-12-27:** Root cause identified
- generateSelector used class-first priority without uniqueness check
- No post-processing to detect and fix duplicates

**2025-12-27:** Fix implemented
- New selector priority: ID > name > aria-label > text > class
- Added disambiguateSelectors to add [nth=N] indices
- Updated Click/Type to handle extended selectors

**2025-12-27:** Investigation completed
- Status: Complete
- Key outcome: Glass now generates unique selectors via priority order and nth indices
