<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Anthropic provides comprehensive document handling capabilities including PDF support (100-page limit), vision (images), Files API (persistent storage), Agent Skills (PPTX, XLSX, DOCX, PDF creation), and web fetch tool (URL content + PDF retrieval).

**Evidence:** Reviewed official Anthropic documentation at docs.anthropic.com - PDF support page confirms 100-page limit and 32MB request size limit; Files API supports 500MB files and 100GB org storage; Agent Skills are pre-built for PowerPoint, Excel, Word, PDF creation.

**Knowledge:** The 100-page PDF limit is a hard API constraint. For documents exceeding this limit, options include: splitting into chunks, using web fetch for URL-based PDFs, or pre-processing with external tools. Agent Skills provide document creation capabilities that could be valuable for agents needing to produce reports.

**Next:** Create decision on which capabilities to integrate into orch ecosystem, prioritizing Files API for repeated document use and evaluating Agent Skills for report generation use cases.

---

# Investigation: Anthropic Official Document Skills Research

**Question:** What document processing capabilities does Anthropic officially support, and which would be valuable for the orch ecosystem's spawned agents?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Research Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: PDF Support with 100-Page Limit

**Evidence:** Anthropic's PDF support has explicit limits documented at docs.anthropic.com:

| Requirement | Limit |
|------------|--------|
| Maximum request size | 32MB |
| Maximum pages per request | 100 |
| Format | Standard PDF (no passwords/encryption) |

PDFs are processed by:
1. Converting each page to an image
2. Extracting text from each page alongside the image
3. Claude analyzes both text and images for visual content understanding

Token cost: 1,500-3,000 tokens per page depending on content density, plus image costs per page.

Three ways to provide PDFs:
1. URL reference (simplest)
2. Base64-encoded in document content blocks
3. Files API file_id (for repeated use)

**Source:** https://docs.anthropic.com/en/build-with-claude/pdf-support

**Significance:** The 100-page limit mentioned in the task is a hard API constraint. This impacts any workflow involving large documents (lengthy reports, books, legal documents). Workarounds include splitting documents or using alternative processing.

---

### Finding 2: Files API (Beta) - Persistent File Storage

**Evidence:** The Files API provides persistent file storage:

| Storage Limit | Value |
|--------------|-------|
| Maximum file size | 500 MB per file |
| Total storage | 100 GB per organization |

Supported content types:
- PDF (`application/pdf`) → `document` content block
- Plain text (`text/plain`) → `document` content block  
- Images (JPEG, PNG, GIF, WebP) → `image` content block
- Various data files → `container_upload` (for code execution)

Key features:
- Upload once, reference via `file_id` across multiple API calls
- No re-upload overhead for frequently used documents
- Files persist until deleted
- Workspace-scoped (shared across API keys in same workspace)
- FREE for upload/download/list/metadata operations
- Only charged for input tokens when file content used in Messages

**Source:** https://docs.anthropic.com/en/build-with-claude/files

**Significance:** For spawned agents working with the same documents repeatedly, the Files API eliminates redundant uploads and reduces latency. This is particularly valuable for investigations or multi-phase work on the same document set.

---

### Finding 3: Vision Capabilities for Images

**Evidence:** Claude's vision capabilities support:

| Limit | Value |
|-------|-------|
| Images per request | Up to 100 (API) / 20 (claude.ai) |
| Maximum image size | 8000x8000 px |
| Maximum image size (>20 images) | 2000x2000 px |
| Per-image file size | 5MB (API) / 10MB (claude.ai) |
| Request size limit | 32MB total |

Supported formats: JPEG, PNG, GIF, WebP

Token calculation: `tokens = (width px * height px) / 750`

Optimal size: No larger than 1.15 megapixels (1568px both dimensions) to avoid resizing overhead.

Three ways to provide images:
1. Base64-encoded in image content blocks
2. URL reference (images hosted online)
3. Files API file_id

**Source:** https://docs.anthropic.com/en/build-with-claude/vision

**Significance:** Agents can analyze screenshots, diagrams, charts, and visual documentation. Combined with PDF support (which renders pages as images), this enables comprehensive document analysis.

---

### Finding 4: Agent Skills for Document Creation (Beta)

**Evidence:** Anthropic provides pre-built Agent Skills for document creation:

| Skill ID | Purpose |
|----------|---------|
| `pptx` | Create PowerPoint presentations |
| `xlsx` | Create Excel spreadsheets with charts |
| `docx` | Create Word documents |
| `pdf` | Generate PDF documents |

Skills require:
- Code execution tool (`code_execution_20250825`)
- Skills beta header (`skills-2025-10-02`)
- Files API beta header (`files-api-2025-04-14`) for downloading outputs

Key characteristics:
- Skills run in a VM environment with filesystem access
- Maximum 8 Skills per request
- No network access within container
- Custom Skills can be uploaded (up to 8MB)
- Skills use progressive disclosure (metadata always loaded, instructions loaded on-demand)

**Source:** https://docs.anthropic.com/en/agents-and-tools/agent-skills/overview, https://docs.anthropic.com/en/build-with-claude/skills-guide

**Significance:** These Skills enable agents to CREATE documents, not just read them. An investigation agent could produce a formatted PDF report; a data analysis agent could create Excel visualizations.

---

### Finding 5: Web Fetch Tool (Beta) - URL Content Retrieval

**Evidence:** The web fetch tool can retrieve full content from URLs, including PDFs:

- Fetches full text content from web pages
- Automatic text extraction for PDFs
- Supports citations for fetched content
- Optional domain filtering (allowed/blocked domains)
- `max_content_tokens` parameter to limit context usage

Limits:
- URL max length: 250 characters
- Only text and PDF content types supported
- No JavaScript-rendered pages

Cost: No additional charges beyond standard token costs for fetched content.

**Source:** https://docs.anthropic.com/en/agents-and-tools/tool-use/web-fetch-tool

**Significance:** Agents can fetch PDFs from URLs on-demand without needing to pre-upload. Combined with web search, agents can find and analyze documents dynamically. This partially addresses the 100-page limit by enabling chunked fetching workflows.

---

### Finding 6: 1M Token Context Window

**Evidence:** Claude now supports an extended 1M token context window (beta), allowing processing of:
- Much larger documents
- Longer conversations
- More extensive codebases

Available on Claude API (beta), Bedrock (beta), Vertex AI (beta), Azure AI (beta).

**Source:** https://docs.anthropic.com/en/build-with-claude/overview (features table)

**Significance:** The 1M context window can accommodate significantly more content than the standard window, potentially allowing more document pages to be processed in a single request (though the 100-page limit may still apply as a separate constraint).

---

## Synthesis

**Key Insights:**

1. **100-Page Limit is Real but Workable** - The PDF limit is a hard constraint at the API level. For documents exceeding this, the strategy is: split into chunks, use web fetch for on-demand retrieval, or pre-process with external tools. The 32MB request limit is also relevant for image-heavy PDFs.

2. **Files API Enables Efficiency at Scale** - For orch workflows where agents repeatedly work with the same documents (e.g., investigation → implementation → validation phases), the Files API eliminates redundant uploads. Upload once via orchestrator, pass file_id to spawned agents.

3. **Agent Skills Enable Output, Not Just Input** - The pre-built Skills (PPTX, XLSX, DOCX, PDF) are for CREATING documents, not reading them. This opens use cases like: investigation agents producing formatted reports, data analysis agents creating visualizations, design agents generating presentations.

4. **Web Fetch Provides Dynamic Document Access** - Agents can retrieve PDFs from URLs on-demand, enabling research workflows that discover and analyze documents dynamically without pre-planning what to upload.

**Answer to Investigation Question:**

Anthropic officially supports:
- **PDF Support**: Reading PDFs up to 100 pages / 32MB, with visual understanding of charts and images
- **Vision**: Analyzing images up to 100 per request, 8000x8000px max
- **Files API**: Persistent storage up to 500MB/file, 100GB/org, for upload-once-use-many workflows
- **Agent Skills**: Document creation (PPTX, XLSX, DOCX, PDF) in code execution containers
- **Web Fetch**: On-demand URL content retrieval including PDFs

For the orch ecosystem, the most valuable capabilities are:
1. **Files API** - For multi-phase agent workflows on the same documents
2. **Agent Skills** - For agents that need to produce formatted output (reports, presentations)
3. **Web Fetch** - For dynamic research workflows

---

## Structured Uncertainty

**What's tested:**

- ✅ PDF 100-page limit confirmed (documented in official API docs)
- ✅ Files API 500MB limit confirmed (documented in Files API guide)
- ✅ Agent Skills require code execution tool (documented and demonstrated in Skills guide)
- ✅ Web fetch can retrieve PDFs from URLs (documented in web fetch tool guide)

**What's untested:**

- ⚠️ Behavior when exceeding 100-page limit (error message format, graceful degradation)
- ⚠️ Performance of 1M context window with large PDFs (beta feature availability)
- ⚠️ Custom Skills upload workflow for orch-specific document handling
- ⚠️ Interaction between Files API and prompt caching

**What would change this:**

- If Anthropic increases the 100-page limit, large document workflows become simpler
- If Agent Skills become GA (non-beta), they could be more reliably integrated
- If Files API adds cross-workspace sharing, document hand-off between orchestrator and agents would change

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Integrate Files API for Document-Heavy Workflows** - Modify spawn context to support passing file_ids to agents, enabling upload-once-reference-many patterns.

**Why this approach:**
- Eliminates redundant uploads when multiple agents work on same documents
- Reduces token overhead (no base64 encoding in spawn context)
- Aligns with Anthropic's recommended pattern for repeated document use

**Trade-offs accepted:**
- Adds complexity to spawn workflow (file upload step before spawn)
- Requires cleanup logic for uploaded files

**Implementation sequence:**
1. Add `--file` flag to `orch spawn` for pre-uploading documents
2. Include file_ids in SPAWN_CONTEXT.md as a structured field
3. Update agent skill guidance to reference documents via file_id

### Alternative Approaches Considered

**Option B: Embed Base64 documents in spawn context**
- **Pros:** Simpler, no API dependency
- **Cons:** Context bloat, redundant for multi-agent workflows, 32MB limit
- **When to use instead:** One-off document analysis, small documents

**Option C: Rely on agents to fetch documents via web fetch**
- **Pros:** Dynamic, agents discover what they need
- **Cons:** Requires URLs, doesn't work for local files, network dependency
- **When to use instead:** Research workflows where documents are online

**Rationale for recommendation:** Files API is the right abstraction for the orchestration model - orchestrator knows what documents are needed and can pre-stage them for agents.

---

### Implementation Details

**What to implement first:**
- Add Files API integration to orch-go for document staging
- Create spawn context field for file references
- Update skill guidance for file_id usage

**Things to watch out for:**
- ⚠️ Files API is beta - API may change
- ⚠️ 100GB org storage limit - need cleanup strategy
- ⚠️ File lifecycle management (delete after workflow complete)

**Areas needing further investigation:**
- Agent Skills integration for report generation use cases
- 1M context window availability and pricing
- Custom Skills for org-specific document handling

**Success criteria:**
- ✅ Agents can receive file_ids in spawn context
- ✅ Multi-phase workflows don't re-upload documents
- ✅ Cleanup happens after workflow completion

---

## References

**Files Examined:**
- N/A (web research only)

**Commands Run:**
```bash
# None - pure documentation research
```

**External Documentation:**
- https://docs.anthropic.com/en/build-with-claude/pdf-support - PDF limits and usage
- https://docs.anthropic.com/en/build-with-claude/files - Files API guide
- https://docs.anthropic.com/en/build-with-claude/vision - Image handling
- https://docs.anthropic.com/en/agents-and-tools/agent-skills/overview - Agent Skills overview
- https://docs.anthropic.com/en/build-with-claude/skills-guide - Skills API guide
- https://docs.anthropic.com/en/agents-and-tools/tool-use/web-fetch-tool - Web fetch tool
- https://docs.anthropic.com/en/build-with-claude/overview - Features overview table

**Related Artifacts:**
- None - first investigation on this topic

---

## Investigation History

**2025-12-30 10:00:** Investigation started
- Initial question: What document capabilities does Anthropic officially support?
- Context: Hit 100-page PDF limit, prompted exploration of alternatives

**2025-12-30 10:45:** Research completed
- Documented 6 major capabilities (PDF, Files API, Vision, Agent Skills, Web Fetch, 1M Context)
- Confirmed 100-page limit, identified workarounds
- Status: Complete
- Key outcome: Files API recommended for orch ecosystem integration
