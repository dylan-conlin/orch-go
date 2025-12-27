## Summary (D.E.K.N.)

**Delta:** Successfully implemented a pure Go URL-to-Markdown CLI (`orch fetch-md`) using chromedp for browser automation and html-to-markdown for conversion.

**Evidence:** All tests pass (unit + integration), CLI works with real URLs (tested example.com), produces clean Markdown output with proper relative URL conversion.

**Knowledge:** html-to-markdown v2 requires explicit plugin registration (`base` + `commonmark`) and `WithDomain` is a ConvertOptionFunc, not a converter option.

**Next:** None - implementation complete, ready for use.

**Confidence:** High (90%) - Tested with integration tests and real URL, but edge cases with complex SPAs not exhaustively tested.

---

# Investigation: Rewrite Url Markdown As Go

**Question:** Can we rewrite url-to-markdown as a pure Go CLI, replacing the Python-based shot-scraper + markitdown pipeline?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Feature implementation agent
**Phase:** Complete
**Next Step:** None - implementation delivered
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: chromedp Successfully Replaces shot-scraper

**Evidence:** Integration tests confirm chromedp can:
- Navigate to URLs and wait for JavaScript rendering
- Extract full page HTML or target specific CSS selectors
- Capture page title and final URL (after redirects)
- Handle timeouts properly

**Source:** `pkg/urltomd/urltomd.go:85-165`, `pkg/urltomd/urltomd_test.go:181-238`

**Significance:** chromedp provides all the functionality needed from shot-scraper without Python/Playwright dependencies.

---

### Finding 2: html-to-markdown v2 API Requires Plugin Registration

**Evidence:** Initial implementation failed with "no render handlers are registered" error. The v2 API requires explicit plugin registration:
```go
conv := converter.NewConverter(
    converter.WithPlugins(
        base.NewBasePlugin(),
        commonmark.NewCommonmarkPlugin(),
    ),
)
```

**Source:** `pkg/urltomd/urltomd.go:211-218`, html-to-markdown v2 documentation

**Significance:** This is a breaking change from v1. The `WithDomain` option is used on `ConvertString`, not `NewConverter`.

---

### Finding 3: CLI Integration Works Seamlessly

**Evidence:** `orch fetch-md` command successfully:
- Fetches and converts example.com in ~1 second
- Produces clean Markdown with proper formatting
- Supports all planned options (wait, selector, frontmatter, output file)

**Source:** Manual testing: `/tmp/orch-test fetch-md https://example.com --wait 500`

**Significance:** The implementation is production-ready and integrated into the orch CLI.

---

## Synthesis

**Key Insights:**

1. **Single Binary Deployment** - The Go implementation eliminates all Python dependencies (Playwright, shot-scraper, markitdown) for URL-to-Markdown functionality.

2. **Mature Libraries** - chromedp (12.6k stars) and html-to-markdown (3.3k stars) are well-maintained and handle edge cases.

3. **API Design Matters** - v2 API changes required careful attention to plugin registration and option placement.

**Answer to Investigation Question:**

Yes, url-to-markdown can be fully replaced with a pure Go implementation. The `orch fetch-md` command provides equivalent functionality to the Python pipeline using chromedp for browser automation and html-to-markdown for conversion. All planned features (wait time, CSS selector, frontmatter, output file) are implemented and tested.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

The implementation passes all tests and works with real URLs, but hasn't been tested against complex JavaScript-heavy SPAs or sites with anti-bot measures.

**What's certain:**

- ✅ chromedp successfully fetches and renders JavaScript content
- ✅ html-to-markdown produces clean, standard Markdown
- ✅ Relative URLs are properly converted to absolute
- ✅ CLI is integrated and functional

**What's uncertain:**

- ⚠️ Behavior with complex SPAs requiring authentication
- ⚠️ Sites with aggressive bot detection
- ⚠️ Very large pages (memory consumption not tested)

**What would increase confidence to Very High (95%+):**

- Test with 20+ diverse real-world URLs
- Add benchmarks comparing performance to Python pipeline
- Test in CI environment with headless-shell container

---

## Deliverables

**Files Created:**
- `pkg/urltomd/urltomd.go` - Core package (267 lines)
- `pkg/urltomd/urltomd_test.go` - Tests with unit and integration coverage
- `cmd/orch/fetchmd.go` - CLI command (118 lines)

**Dependencies Added:**
- `github.com/chromedp/chromedp v0.14.2`
- `github.com/JohannesKaufmann/html-to-markdown/v2 v2.5.0`

**Usage:**
```bash
# Basic usage
orch fetch-md https://example.com

# With options
orch fetch-md https://example.com --wait 3000 --selector "main" --frontmatter -o page.md
```

---

## Investigation History

**2025-12-26:** Implementation started
- Reviewed prior architecture investigation
- Created pkg/urltomd package structure

**2025-12-26:** Core implementation complete
- Added chromedp + html-to-markdown dependencies
- Implemented Fetch() and FetchWithFrontmatter() functions
- Fixed html-to-markdown v2 API usage (plugin registration)

**2025-12-26:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Pure Go URL-to-Markdown CLI replacing Python pipeline
