<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Local PDF tools (poppler) can extract text at 16ms vs seconds for API, saving 6-12x tokens for text-heavy documents; hybrid workflow (local split → API analyze) handles 100+ page documents.

**Evidence:** Tested pdftotext (16ms for 4-page PDF), pdfseparate (creates valid per-page PDFs), pdfinfo (metadata query) - all work correctly on real documents.

**Knowledge:** Use triage-first pattern: check pages/content type with pdfinfo, extract text locally for text docs, split large docs before API, use API only for visual content or targeted pages.

**Next:** Document this pattern in CLAUDE.md or agent guidance; consider helper script for triage workflow.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Pre-Process Documents Locally vs API

**Question:** When should we pre-process documents (especially PDFs) locally vs relying on Anthropic's API capabilities? What tools exist, and what are the tradeoffs?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A (complements 2025-12-30-inv-research-anthropic-official-document-skills.md which covers API-side)
**Superseded-By:** N/A

---

## Findings

### Finding 1: Poppler (pdftotext) Already Installed

**Evidence:** Poppler 25.02.0 is installed at `/opt/homebrew/Cellar/poppler/25.02.0/` with these tools:
- `pdftotext` - Extract text from PDFs (supports page ranges via `-f` and `-l` flags)
- `pdfinfo` - Get PDF metadata (pages, size, encryption status)
- `pdfseparate` - Split PDF into individual pages
- `pdfunite` - Merge multiple PDFs
- `pdftoppm` - Convert pages to images
- `pdfimages` - Extract embedded images

**Source:** `ls /opt/homebrew/Cellar/poppler/25.02.0/bin/`

**Significance:** Core PDF manipulation already available. No installation required for basic workflows.

---

### Finding 2: Text Extraction is Fast (16ms for 4-page PDF)

**Evidence:** Tested pdftotext on a 577KB, 4-page PDF:
```bash
$ time pdftotext "/path/to/Firmware_Update_Instruction.pdf" -
# 0.016 total (16ms)
# Output: 2060 bytes of text
```

Page-specific extraction also works:
```bash
$ pdftotext -f 1 -l 1 doc.pdf -  # Extract only page 1
```

**Source:** Direct testing with Dell monitor firmware PDF

**Significance:** Local extraction is essentially instant compared to API round-trip. For a 100-page document:
- Local pdftotext: ~400ms estimated (16ms × 25 scaling)
- API call: Seconds + 150K-300K tokens (1,500-3,000 per page)

---

### Finding 3: PDF Page Splitting Works for Large Documents

**Evidence:** pdfseparate can split PDFs into individual pages:
```bash
$ pdfseparate -f 1 -l 2 input.pdf /tmp/page-%d.pdf
# Creates page-1.pdf (490KB), page-2.pdf (432KB)
```

**Source:** Direct testing

**Significance:** This enables the "local split → API analyze" hybrid workflow:
1. Split 200-page handbook into chunks of 50 pages
2. Send only the relevant chunk to API
3. Or: Extract text first, find relevant pages, send only those

---

### Finding 4: Homebrew Has 30+ PDF Tools Available

**Evidence:** `brew search pdf` returns 30+ options including:
- **pdfcpu** - Go-based PDF processor (validation, manipulation)
- **qpdf** - PDF transformation (decrypt, linearize, split)
- **mupdf-tools** - Lightweight viewer/extractor (mutool)
- **ocrmypdf** - Add OCR text layer to scanned PDFs
- **pdfgrep** - Search text in PDFs (grep-like)

**Source:** `brew search pdf`

**Significance:** Ecosystem covers most use cases. Key missing capability: OCR (ocrmypdf not installed but available).

---

### Finding 5: Token Cost Analysis - Local Saves Significantly

**Evidence:** Based on prior investigation (2025-12-30-inv-research-anthropic-official-document-skills.md):
- API PDF processing: 1,500-3,000 tokens/page
- For 100-page document: 150K-300K input tokens
- Claude API pricing (Opus): ~$15/M input tokens → $2.25-4.50 per 100-page doc

Local pdftotext:
- Extracts text in ~400ms for 100 pages
- Text output typically 1-2KB per page → 100-200KB total
- If fed as text instead of PDF: ~25K-50K tokens
- Cost reduction: 6-12x fewer tokens

**Source:** Token estimates from prior investigation, local testing

**Significance:** For text-heavy documents (handbooks, contracts), local extraction saves both money and context window.

---

### Finding 6: Visual Content Requires API

**Evidence:** pdftotext extracts text only. For documents with:
- Charts and graphs
- Diagrams and flowcharts
- Embedded images with important content
- Forms with visual layout

The API's PDF-as-images approach is required.

**Source:** pdftotext help (text-only output)

**Significance:** Use case determines approach:
- Text documents (handbooks, legal) → Local first
- Visual documents (presentations, reports with charts) → API required

---

## Synthesis

**Key Insights:**

1. **Local-first is fast and cheap for text documents** - pdftotext extracts text at 16ms per 4-page document, saves 6-12x on tokens compared to API PDF processing. For the employee handbook scenario that triggered this investigation, local extraction + text search could have found the holidays section in seconds.

2. **Hybrid workflows unlock large documents** - API has 100-page limit. Local pdfseparate can split 200-page documents into API-digestible chunks, or pdftotext can extract text to identify which pages are relevant before sending only those to API.

3. **Document type determines approach** - Text-heavy documents (handbooks, contracts, legal) benefit most from local-first. Visual documents (presentations, infographics, charts) require API's vision capabilities.

4. **Tools are already available** - Poppler is installed. No additional setup required for basic text extraction, page splitting, and metadata queries.

**Answer to Investigation Question:**

**When to pre-process locally:**
- Document > 100 pages (split before API)
- Document is primarily text (extract text, save tokens)
- Need to search/find specific sections (grep text locally first)
- Cost optimization is important (6-12x token savings)
- Low latency required (16ms vs seconds)

**When to use API directly:**
- Document has visual content (charts, diagrams, images)
- Need visual layout understanding (forms, tables)
- Document < 100 pages and moderately sized
- OCR required (scanned documents) - unless ocrmypdf installed

**Hybrid workflow for large text documents:**
1. `pdfinfo` → Check page count, size
2. `pdftotext` → Extract full text, search for relevant sections
3. `pdfseparate` → Extract only relevant pages
4. API → Analyze the targeted pages with full context

---

## Structured Uncertainty

**What's tested:**

- ✅ pdftotext extracts text at 16ms for 4-page PDF (tested directly)
- ✅ pdftotext -f -l flags work for page-specific extraction (tested: page 1 only)
- ✅ pdfseparate creates valid per-page PDFs (tested: 2 pages extracted)
- ✅ Poppler 25.02.0 installed at /opt/homebrew/Cellar/poppler/ (verified)

**What's untested:**

- ⚠️ Performance scaling to 100+ page documents (extrapolated from 4-page test)
- ⚠️ Token count reduction claim (6-12x) not measured with real API comparison
- ⚠️ OCR workflow with ocrmypdf (not installed, not tested)
- ⚠️ Actual hybrid workflow end-to-end (concept validated, not executed)

**What would change this:**

- If pdftotext produces garbled output for complex PDFs (encoding issues)
- If Claude's text analysis is significantly worse than PDF-as-image analysis
- If token counting for extracted text is higher than estimated

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Triage-First Pattern** - Check document characteristics before deciding approach.

**Decision tree for document handling:**
```
1. Get metadata: pdfinfo doc.pdf
2. Is pages > 100? → Split required (pdfseparate or pdftotext chunks)
3. Is content primarily text? → Extract text first (pdftotext)
4. Has visual content (charts, forms)? → API required for those pages
5. Otherwise → Direct API is fine
```

**Why this approach:**
- Avoids wasting API calls on simple text extraction
- Handles 100-page limit gracefully
- Uses best tool for each document type
- Already possible with installed tools

**Trade-offs accepted:**
- Additional step before API calls
- Need to assess document type (manual or heuristic)

### Alternative Approaches Considered

**Option B: Always use API**
- **Pros:** Simplest, no local tooling
- **Cons:** 100-page limit, higher token cost, slower
- **When to use instead:** Small documents (<50 pages) with visual content

**Option C: Always extract text locally**
- **Pros:** Fastest, cheapest
- **Cons:** Loses visual understanding (charts, forms, images)
- **When to use instead:** Pure text documents (code, articles, manuals)

**Rationale for recommendation:** Triage pattern is low-effort (one pdfinfo call) and enables optimal handling per document.

---

### Implementation Details

**Practical workflow for the "employee handbook holidays" use case:**
```bash
# 1. Check size
pdfinfo handbook.pdf  # Pages: 150

# 2. Extract text, find relevant section
pdftotext handbook.pdf - | grep -n -i "holiday"
# Output: Lines 2450-2520 contain holiday info (pages ~50-52)

# 3. Extract just those pages
pdfseparate -f 50 -l 52 handbook.pdf /tmp/holidays-%d.pdf

# 4. Send targeted pages to API
# → 3 pages instead of 150 = 50x fewer tokens
```

**What to implement first:**
- Document this pattern in CLAUDE.md or skill guidance
- Consider helper script: `pdf-prep <file>` that outputs triage recommendation

**Things to watch out for:**
- ⚠️ pdftotext can fail on encrypted/protected PDFs (check pdfinfo first)
- ⚠️ Some PDFs have text as images (scanned) - ocrmypdf needed
- ⚠️ Complex layouts may extract poorly (tables, columns)

**Areas needing further investigation:**
- ocrmypdf installation and workflow for scanned documents
- Integration with Files API (upload extracted pages, not full doc)
- Automation: detect document type automatically

**Success criteria:**
- ✅ Large documents (>100 pages) can be processed via hybrid workflow
- ✅ Token usage reduced for text-heavy documents
- ✅ Pattern documented for agent use

---

## References

**Files Examined:**
- `/opt/homebrew/Cellar/poppler/25.02.0/bin/` - Available poppler tools
- `.kb/investigations/2025-12-30-inv-research-anthropic-official-document-skills.md` - Prior API research

**Commands Run:**
```bash
# Check installed tools
ls /opt/homebrew/Cellar/poppler/25.02.0/bin/

# Test text extraction timing
time pdftotext doc.pdf -

# Test page-specific extraction
pdftotext -f 1 -l 1 doc.pdf -

# Test page splitting
pdfseparate -f 1 -l 2 input.pdf /tmp/page-%d.pdf

# Search available PDF tools
brew search pdf

# Get PDF metadata
pdfinfo doc.pdf
```

**External Documentation:**
- https://poppler.freedesktop.org/ - Poppler documentation
- Anthropic PDF docs (covered in prior investigation)

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-30-inv-research-anthropic-official-document-skills.md` - Covers API-side capabilities (100-page limit, token costs)

---

## Investigation History

**2025-12-30 10:20:** Investigation started
- Initial question: When should we pre-process PDFs locally vs use API?
- Context: Hit 100-page limit trying to read employee handbook; could have extracted holidays section locally first

**2025-12-30 10:30:** Local tools surveyed
- Poppler already installed with full suite (pdftotext, pdfseparate, pdfinfo)
- 30+ additional tools available via Homebrew

**2025-12-30 10:40:** Performance tested
- pdftotext: 16ms for 4-page document
- pdfseparate: Creates valid per-page PDFs

**2025-12-30 10:50:** Investigation completed
- Status: Complete
- Key outcome: Local-first triage recommended - use pdfinfo/pdftotext to assess, split large docs, send only relevant pages to API
