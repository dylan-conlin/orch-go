## Summary (D.E.K.N.)

**Delta:** Implemented `kb index` command that outputs a concise markdown index of knowledge artifacts for session context injection.

**Evidence:** Created `index.go` (300+ lines) with tests in `index_test.go` (250+ lines) in kb-cli repo.

**Knowledge:** Models use `## Summary (30 seconds)` section; guides use `**Purpose:**` line; decisions are filtered by date (--recent flag, default 30 days).

**Next:** Build and test on host machine (Go not available in sandbox), then integrate with session start hook.

**Promote to Decision:** recommend-no - tactical implementation of existing design

---

# Investigation: Kb Index Surface Knowledge Artifacts

**Question:** How should `kb index` scan and format knowledge artifacts for session context injection?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** Build and test on host machine
**Status:** Complete

---

## Findings

### Finding 1: Knowledge artifact formats differ by type

**Evidence:**
- Models use `# Model: Title` and `## Summary (30 seconds)` section
- Guides use `# Title` and `**Purpose:**` line
- Decisions use `# Decision: Title` and `**Purpose:**` line with YYYY-MM-DD date prefix in filename

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/models/decidability-graph.md`
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/guides/spawn.md`

**Significance:** Implementation must handle each format differently when extracting summaries.

---

### Finding 2: kb-cli source is at ~/Documents/personal/kb-cli

**Evidence:** Task description mentioned "orch-knowledge repo" but actual kb CLI source is at `~/Documents/personal/kb-cli/cmd/kb/`.

**Source:** `find ~/Documents -name "main.go" -path "*kb*"` returned `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/main.go`

**Significance:** Implementation is in kb-cli repo but issue is tracked in orch-go for orchestration workflow visibility.

---

### Finding 3: Existing patterns in kb-cli for file scanning

**Evidence:**
- `list.go` has `ListArtifacts()`, `extractArtifactMetadata()` functions
- `guides.go` has `extractGuideMetadata()` with title/purpose extraction
- `context.go` has comprehensive artifact search with scoring

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/`

**Significance:** Used similar patterns for consistency - cobra command structure, metadata extraction from first 50 lines, etc.

---

## Implementation

Created two files in kb-cli:

### `cmd/kb/index.go` (~300 lines)

**Command:** `kb index [--recent N] [--path]`

**Features:**
- Scans `.kb/models/`, `.kb/guides/`, `.kb/decisions/`
- Extracts title from H1 heading (removes `Model:`, `Decision:` prefixes)
- Extracts summary from `**Purpose:**` line or `## Summary` section
- Filters decisions to last N days (default 30, configurable with `--recent`)
- Skips template/meta files (README.md, TEMPLATE.md, PHASE*.md)
- Optional `--path` flag to include file paths

**Output format:**
```
## Knowledge Index (query with `kb context "<topic>"`)

**Models (synthesized understanding):**
- decidability-graph - A decidability graph encodes decision dependencies.
- daemon-autonomous-operation - How daemon traverses work

**Guides (procedural knowledge):**
- spawn - Single authoritative reference for how orch spawn creates agents
- agent-lifecycle - States, completion, abandonment

**Recent Decisions:**
- use-go-for-cli - Go provides better CLI tooling and cross-compilation.
```

### `cmd/kb/index_test.go` (~250 lines)

**Tests:**
- `TestBuildIndex_Models` - verifies model scanning and summary extraction
- `TestBuildIndex_Guides` - verifies guide scanning and purpose extraction
- `TestBuildIndex_RecentDecisions` - verifies date filtering
- `TestBuildIndex_SkipsTemplates` - verifies template/meta file exclusion
- `TestBuildIndex_WithPath` - verifies --path flag behavior
- `TestBuildIndex_EmptyKB` - verifies empty state handling
- `TestBuildIndex_NoKBDir` - verifies error when .kb missing
- `TestExtractIndexMetadata_*` - verifies metadata extraction
- `TestTruncateSummary` - verifies summary truncation
- `TestFilterRecentItems` - verifies date filtering

---

## Structured Uncertainty

**What's tested:**
- ⚠️ Code compiles and test coverage exists (not verified - Go not available in sandbox)

**What's untested:**
- ⚠️ Actual test execution on host machine
- ⚠️ Integration with session start hook

**What would change this:**
- Tests fail on host machine → fix code
- Output format not suitable for LLM injection → adjust formatting

---

## Implementation Recommendations

### Recommended Approach ⭐

**Build and test on host, then integrate with session hook**

**Implementation sequence:**
1. Build kb-cli on host: `cd ~/Documents/personal/kb-cli && make install`
2. Run tests: `make test`
3. Test manually: `kb index` in orch-go directory
4. Add to session start hook in `~/.claude/hooks/` to call `kb index` at session start

### Session Hook Integration

Add to session start hook (similar to `bd prime`):

```bash
# In session start hook
kb index 2>/dev/null || true  # Output knowledge index, ignore if kb not found
```

---

## References

**Files Examined:**
- `~/Documents/personal/kb-cli/cmd/kb/main.go` - Entry point structure
- `~/Documents/personal/kb-cli/cmd/kb/list.go` - Artifact listing patterns
- `~/Documents/personal/kb-cli/cmd/kb/guides.go` - Guide metadata extraction
- `~/Documents/personal/kb-cli/cmd/kb/context.go` - Comprehensive context search

**Files Created:**
- `~/Documents/personal/kb-cli/cmd/kb/index.go` - Main implementation
- `~/Documents/personal/kb-cli/cmd/kb/index_test.go` - Test coverage

**Related Artifacts:**
- **Task:** orch-go-b111k - Original issue for this implementation

---

## Investigation History

**2026-01-23:** Investigation started
- Initial question: How to implement kb index for session context injection

**2026-01-23:** Implementation complete
- Created index.go and index_test.go in kb-cli
- Go compiler not available in sandbox - tests pending host execution
- Status: Complete (code written, tests pending)
