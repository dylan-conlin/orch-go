# Session Synthesis

**Agent:** og-arch-review-url-markdown-26dec
**Issue:** orch-go-untracked-1766786808
**Duration:** 2025-12-26
**Outcome:** success

---

## TLDR

Reviewed the url-to-markdown architecture for Go rewrite. The current Python-based pipeline (shot-scraper + markitdown) can be fully replaced with native Go libraries: chromedp (12.6k stars) for JavaScript rendering and html-to-markdown (3.3k stars) for Markdown conversion, enabling single-binary deployment without Python dependencies.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-design-review-url-markdown-architecture-before.md` - Full architecture review with recommendations

### Files Modified
- None

### Commits
- Pending (investigation file uncommitted)

---

## Evidence (What Was Observed)

- **shot-scraper** uses Playwright (Python) to render JavaScript and extract HTML. Source: https://shot-scraper.datasette.io
- **markitdown** is Microsoft's general-purpose converter (84.7k stars) but overkill for HTML-only use. Source: https://github.com/microsoft/markitdown
- Current implementation at `/Users/dylanconlin/Documents/personal/mcp/web-to-markdown/url-to-markdown.sh:43` is a simple pipeline:
  ```bash
  shot-scraper html "$URL" --wait "$WAIT_TIME" | markitdown -m text/html -o "$TEMP_FILE"
  ```
- **chromedp** (12.6k stars) provides equivalent functionality to shot-scraper in pure Go. Source: https://github.com/chromedp/chromedp
- **html-to-markdown** v2 (3.3k stars) provides equivalent functionality to markitdown's HTML conversion in pure Go. Source: https://github.com/JohannesKaufmann/html-to-markdown

### Tests Run
```bash
# No tests run - this was a design investigation
# Examined external documentation and existing code
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-design-review-url-markdown-architecture-before.md` - Complete architecture review

### Decisions Made
- **Decision: Use pure Go implementation** because it enables single-binary deployment, faster execution, and native orch-go integration
- **Decision: Use chromedp + html-to-markdown** because these are mature, well-maintained Go libraries that replicate Python functionality

### Constraints Discovered
- Requires Chrome/Chromium binary available (or chromedp/headless-shell Docker image)
- Some sites may block headless browsers (may need user-agent spoofing)
- html-to-markdown v2 has breaking changes from v1

### Externalized via `kn`
- N/A - No kn commands available in this environment

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file)
- [x] Tests passing (N/A for investigation)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete {issue-id}` (beads issue not found)

### Follow-up Work (for orchestrator to create issues)

**Issue 1: Implement Go URL-to-Markdown package**
- **Skill:** feature-impl
- **Context:** Create `pkg/urltomd/` using chromedp + html-to-markdown. See investigation for implementation details.

**Issue 2: Add CLI command for URL fetching**
- **Skill:** feature-impl  
- **Context:** Add `orch fetch-markdown <url>` command using the new package.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to handle YouTube transcript extraction in Go (need separate library)
- How to extract web metadata (OG tags, structured data) in Go
- Performance comparison between Python and Go implementations

**Areas worth exploring further:**
- Rate limiting and respectful crawling for batch URL fetching
- Caching strategies for frequently accessed URLs
- Integration with OpenAI for image description (markitdown feature)

**What remains unclear:**
- Exact output parity with markitdown (need real-world testing)
- Edge cases with JavaScript-heavy SPAs
- Headless browser detection by target sites

---

## Session Metadata

**Skill:** architect
**Model:** Claude
**Workspace:** `.orch/workspace/og-arch-review-url-markdown-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-design-review-url-markdown-architecture-before.md`
**Beads:** Issue not found in local beads database
