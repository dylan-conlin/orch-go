## Summary (D.E.K.N.)

**Delta:** The current url-to-markdown pipeline uses shot-scraper (Playwright-based JS rendering) + markitdown (HTML-to-markdown conversion), and Go can eliminate both Python dependencies using chromedp + html-to-markdown libraries.

**Evidence:** Examined shot-scraper docs (uses Playwright for JS rendering + HTML extraction), markitdown docs (Microsoft's Python HTML-to-Markdown tool), and found mature Go alternatives: chromedp (12.6k stars, Chrome DevTools Protocol) and html-to-markdown (3.3k stars, full Markdown support).

**Knowledge:** The core workflow is simple: fetch rendered HTML → convert to markdown. Go native libraries can replicate this without Python/Node dependencies, enabling a single binary deployment.

**Next:** Implement Go version using chromedp for HTML fetching and html-to-markdown for conversion. Keep current shell script approach as fallback option.

**Confidence:** High (85%) - Go libraries are mature and well-documented, but untested on actual URL edge cases.

---

# Investigation: URL-to-Markdown Architecture Review Before Go Rewrite

**Question:** What does the current url-to-markdown implementation do, can we eliminate Python dependencies, and what's the minimal viable Go implementation?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Current Implementation Uses Two Tools in Pipeline

**Evidence:** The shell script at `/Users/dylanconlin/Documents/personal/mcp/web-to-markdown/url-to-markdown.sh` shows:
```bash
shot-scraper html "$URL" --wait "$WAIT_TIME" | markitdown -m text/html -o "$TEMP_FILE"
```

This is a simple two-stage pipeline:
1. `shot-scraper html` - Fetches URL, renders JavaScript, outputs final HTML
2. `markitdown` - Converts HTML to Markdown

**Source:** `/Users/dylanconlin/Documents/personal/mcp/web-to-markdown/url-to-markdown.sh:43`

**Significance:** The architecture is very simple. Each tool has a single responsibility. This makes it straightforward to replace with Go equivalents.

---

### Finding 2: shot-scraper Uses Playwright for JavaScript Rendering

**Evidence:** From shot-scraper documentation (https://shot-scraper.datasette.io):
- Built on Playwright (Python bindings for Chrome/Firefox/WebKit)
- `shot-scraper html` command fetches page, waits for JavaScript to execute, outputs final DOM
- Supports `--wait` for explicit delay, `--selector` for targeting specific elements
- Can execute custom JavaScript before capture

Key flags from `shot-scraper html --help`:
- `--wait INTEGER` - Wait milliseconds before taking snapshot
- `-s, --selector TEXT` - Return outerHTML of first element matching CSS selector
- `-j, --javascript TEXT` - Execute JS prior to saving HTML
- `-b, --browser [chromium|firefox|webkit|chrome|chrome-beta]` - Browser choice

**Source:** https://shot-scraper.datasette.io/en/stable/html.html

**Significance:** The core functionality is "render JavaScript, get HTML". This is exactly what chromedp does, and chromedp is a pure Go library that doesn't require Python or Playwright installation.

---

### Finding 3: markitdown Is a General-Purpose Converter (Overkill for HTML)

**Evidence:** From markitdown documentation (https://github.com/microsoft/markitdown):
- Converts PDF, PowerPoint, Word, Excel, Images, Audio, HTML, CSV, JSON, XML, ZIP, YouTube URLs, EPubs
- 84.7k GitHub stars (Microsoft project)
- Has MCP server integration
- Designed for "LLM-ready Markdown"

For HTML specifically, it:
- Preserves headings, lists, tables, links
- Handles complex formatting
- Focuses on structural preservation

**Source:** https://github.com/microsoft/markitdown

**Significance:** markitdown is designed for many file formats. For pure HTML→Markdown, we only need the HTML converter. Go has excellent alternatives:

- **html-to-markdown** (https://github.com/JohannesKaufmann/html-to-markdown) - 3.3k stars
  - Pure Go, no dependencies
  - Supports bold, italic, lists, blockquotes, code, links, images, tables
  - Smart escaping
  - Plugin system for extensions
  - CLI tool included

---

### Finding 4: MCP Server Has Additional Features Worth Noting

**Evidence:** From `/Users/dylanconlin/Documents/personal/mcp/web-to-markdown/index.js`:

1. **web_to_markdown** - Basic conversion with options:
   - `url`, `wait`, `selector`, `includeImages`, `openaiApiKey`, `outputDir`, `fileName`
   - Adds YAML frontmatter with source URL and date

2. **web_to_markdown_advanced** - Extended options:
   - `javascript` - Custom JS to execute before capture
   - `viewport` - Browser viewport size
   - `browser` - chromium/firefox/webkit choice
   - `markdownOptions` - Pass-through options to markitdown

3. **youtube_to_markdown** - YouTube transcript extraction:
   - Uses `youtube-transcript-api` Python library
   - Falls back to `yt-dlp` if needed
   - Timestamps in MM:SS format

4. **web_metadata** - Metadata extraction:
   - Title, description, canonical URL
   - Open Graph tags
   - Twitter cards
   - JSON-LD structured data

**Source:** `/Users/dylanconlin/Documents/personal/mcp/web-to-markdown/index.js:44-674`

**Significance:** For a minimal viable Go implementation, only `web_to_markdown` is essential. YouTube and metadata extraction are nice-to-have features that can be added later.

---

### Finding 5: Go Has Excellent Alternatives for Both Components

**Evidence:** Research on Go libraries:

**For JavaScript Rendering (replacing shot-scraper):**
- **chromedp** (https://github.com/chromedp/chromedp) - 12.6k stars
  - Pure Go, Chrome DevTools Protocol
  - Faster, simpler than Selenium/Playwright
  - No external dependencies (uses system Chrome or headless-shell)
  - Supports all needed features: navigation, wait, DOM extraction, screenshot

**For HTML-to-Markdown (replacing markitdown):**
- **html-to-markdown** (https://github.com/JohannesKaufmann/html-to-markdown) - 3.3k stars
  - Pure Go library with CLI
  - Excellent Markdown output quality
  - Supports: bold/italic, lists, blockquotes, code, links, images, tables
  - Smart escaping, plugin system
  - `WithDomain()` option to convert relative URLs to absolute
  - v2 is actively maintained

**Source:** GitHub repositories and documentation

**Significance:** Both tools are mature, well-maintained, and provide equivalent or better functionality than the Python alternatives. A Go implementation would be:
- Single binary deployment (no Python/Node runtime)
- Faster startup (no interpreter overhead)
- Easier distribution (just the binary)
- Better integration with orch-go

---

## Synthesis

**Key Insights:**

1. **Simple Pipeline** - The current implementation is just: fetch rendered HTML → convert to markdown. There's no complex logic, just tool composition. This makes a Go rewrite straightforward.

2. **Go Alternatives Are Mature** - chromedp has 12.6k stars and is the standard for Go browser automation. html-to-markdown has 3.3k stars and excellent markdown output. Both are production-ready.

3. **Python Dependencies Are Eliminable** - Neither shot-scraper nor markitdown provides functionality that Go can't replicate. The main blocker was JavaScript rendering, which chromedp handles natively.

4. **MCP Features Are Layered** - The MCP server adds features beyond the core script (YouTube, metadata, advanced options). A minimal implementation can skip these and add them incrementally.

**Answer to Investigation Question:**

**1. What does each tool actually do?**
- `shot-scraper`: Uses Playwright to load URL in browser, wait for JavaScript, extract final HTML
- `markitdown`: Converts HTML (and other formats) to clean Markdown suitable for LLMs

**2. Can we eliminate Python dependencies?**
YES. Go has equivalent tools:
- `chromedp` replaces shot-scraper (Chrome DevTools Protocol, native Go)
- `html-to-markdown` replaces markitdown (pure Go, excellent output)

**3. What's the minimal viable implementation?**
```go
// Pseudocode for minimal implementation
func URLToMarkdown(url string, waitMs int) (string, error) {
    // 1. Use chromedp to fetch rendered HTML
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()
    
    var html string
    err := chromedp.Run(ctx,
        chromedp.Navigate(url),
        chromedp.Sleep(time.Duration(waitMs) * time.Millisecond),
        chromedp.OuterHTML("html", &html),
    )
    
    // 2. Use html-to-markdown to convert
    conv := htmltomarkdown.NewConverter()
    markdown, err := conv.ConvertString(html)
    
    return markdown, err
}
```

**4. Edge cases/features from MCP server worth keeping:**
- **Essential:** URL, wait time, selector targeting
- **Nice-to-have:** YAML frontmatter with source URL, custom JavaScript execution
- **Defer:** YouTube transcripts, metadata extraction, image OCR

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The tools (chromedp, html-to-markdown) are well-established with large user bases. The architecture is straightforward. However, edge cases (specific website rendering quirks, JavaScript-heavy SPAs) haven't been tested.

**What's certain:**

- ✅ shot-scraper uses Playwright for JavaScript rendering (documented, verified)
- ✅ markitdown converts HTML to Markdown (documented, verified)
- ✅ chromedp is the Go standard for browser automation (12.6k stars, active)
- ✅ html-to-markdown is mature and feature-complete (3.3k stars, v2 active)
- ✅ The pipeline architecture is simple and replicable

**What's uncertain:**

- ⚠️ Specific website edge cases (JavaScript frameworks, lazy loading, SPAs)
- ⚠️ Exact parity with current markitdown output quality
- ⚠️ Performance comparison (Go should be faster, but untested)
- ⚠️ Headless Chrome availability on target systems

**What would increase confidence to Very High (95%+):**

- Test on 10+ real URLs comparing Python vs Go output
- Benchmark performance comparison
- Test in CI environment (headless-shell Docker image)
- Verify output quality matches LLM consumption needs

---

## Implementation Recommendations

### Recommended Approach ⭐

**Pure Go Implementation** - Build a Go package using chromedp + html-to-markdown

**Why this approach:**
- Single binary deployment (no runtime dependencies)
- Native integration with orch-go codebase
- Faster startup and execution
- Easier distribution and installation
- Well-tested, mature Go libraries

**Trade-offs accepted:**
- Requires Chrome/Chromium installed (or use chromedp/headless-shell Docker image)
- Initial development effort higher than shell script
- Some edge cases may need tuning

**Implementation sequence:**
1. Create `pkg/urltomd/` package with basic URL→Markdown function
2. Add configuration options (wait time, selector, domain for absolute URLs)
3. Add CLI command (`orch fetch-markdown` or similar)
4. Test against real URLs, compare with Python output
5. Add advanced features as needed (frontmatter, metadata)

### Alternative Approaches Considered

**Option B: Shell out to existing Python tools**
- **Pros:** Zero development effort, already working
- **Cons:** Requires Python + pip + shot-scraper + markitdown installed; slower; distribution complexity
- **When to use instead:** If Go implementation proves too complex or edge cases are insurmountable

**Option C: Use HTTP fetch + no JS rendering**
- **Pros:** Simpler, no browser needed
- **Cons:** Many modern sites require JavaScript; won't work for SPAs
- **When to use instead:** For static HTML sites only; not suitable for general use

**Rationale for recommendation:** Go native approach provides the best long-term maintainability, performance, and distribution story. The Python dependency chain (shot-scraper → Playwright → Chrome + markitdown) is complex and fragile.

---

### Implementation Details

**What to implement first:**
1. Basic `URLToMarkdown(url string) (string, error)` function
2. `WithWait(duration)` option for JavaScript rendering delay
3. `WithSelector(selector)` option for targeting specific content
4. `WithDomain(domain)` option for absolute URL conversion

**Things to watch out for:**
- ⚠️ chromedp needs Chrome/Chromium binary available (use headless-shell in Docker/CI)
- ⚠️ Some sites detect and block headless browsers (may need user-agent spoofing)
- ⚠️ Memory usage with many concurrent fetches (reuse browser context)
- ⚠️ html-to-markdown v2 has breaking changes from v1 (use v2 patterns)

**Areas needing further investigation:**
- YouTube transcript extraction (separate library needed, consider `github.com/kkdai/youtube`)
- Metadata extraction (can use Go HTML parser, `golang.org/x/net/html`)
- Rate limiting and respectful crawling

**Success criteria:**
- ✅ Can fetch and convert any URL that current Python pipeline handles
- ✅ Output quality comparable to markitdown (check headings, links, lists, code blocks)
- ✅ Works in CI/Docker environment with headless-shell
- ✅ Performance equal or better than Python version

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/mcp/web-to-markdown/url-to-markdown.sh` - Current shell script implementation
- `/Users/dylanconlin/Documents/personal/mcp/web-to-markdown/index.js` - MCP server with all tools

**External Documentation:**
- https://shot-scraper.datasette.io/en/stable/html.html - shot-scraper HTML extraction docs
- https://github.com/microsoft/markitdown - markitdown repository
- https://github.com/chromedp/chromedp - Go Chrome DevTools Protocol library
- https://github.com/JohannesKaufmann/html-to-markdown - Go HTML-to-Markdown converter

**Related Artifacts:**
- None directly related in this repository

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: Review url-to-markdown architecture before Go rewrite
- Context: Planning to add URL-to-Markdown functionality to orch-go

**2025-12-26:** Tool analysis complete
- Examined shot-scraper (Playwright-based JS rendering)
- Examined markitdown (Microsoft's HTML-to-Markdown)
- Identified Go alternatives (chromedp, html-to-markdown)

**2025-12-26:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Go can fully replace Python dependencies with chromedp + html-to-markdown
