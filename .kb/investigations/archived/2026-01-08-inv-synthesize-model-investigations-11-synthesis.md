## Summary (D.E.K.N.)

**Delta:** The "11 model investigations" synthesis task is a false positive - only 1 new investigation exists since Jan 6 synthesis, and it's about skillc data models, NOT AI model selection.

**Evidence:** Examined all 11 files - `2026-01-08-inv-design-data-model-load-bearing.md` is about skill.yaml schema; `2025-12-24-inv-test-gemini-flash-model-resolution.md` doesn't exist; `2026-01-04-inv-implement-priority-cascade-model-dashboard.md` is about dashboard status logic.

**Knowledge:** The word "model" triggers false synthesis matches - kb reflect needs semantic awareness or topic tagging to distinguish AI model selection from data models, status models, etc.

**Next:** Close as "no action needed" - prior synthesis at `.kb/guides/model-selection.md` is complete and current; create issue for improving kb reflect deduplication.

**Promote to Decision:** recommend-yes - Need decision on how to handle multi-meaning keywords like "model" in kb reflect synthesis detection.

---

# Investigation: Synthesize Model Investigations 11 Synthesis

**Question:** Are there new AI model selection investigations since the Jan 6 synthesis that need consolidation into `.kb/guides/model-selection.md`?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** kb-reflect synthesis agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Prior synthesis already complete and comprehensive

**Evidence:** The investigation `2026-01-06-inv-synthesize-model-investigations-10-synthesis.md` already synthesized 10 AI model selection investigations into `.kb/guides/model-selection.md`. The guide covers:
- Model aliases (opus, sonnet, haiku, flash, pro)
- Architecture (pkg/model, pkg/account, OpenCode auth.json)
- Spawn mode model passing (fixed Dec 2025)
- Cost/pricing analysis
- Multi-provider patterns

**Source:** `.kb/guides/model-selection.md` (326 lines), `.kb/investigations/2026-01-06-inv-synthesize-model-investigations-10-synthesis.md`

**Significance:** No need for a new synthesis - the authoritative guide already exists and is current.

---

### Finding 2: The "11th investigation" is a false positive (wrong topic)

**Evidence:** `2026-01-08-inv-design-data-model-load-bearing.md` contains:
- D.E.K.N.: "Load-bearing links belong in skill.yaml with a `load_bearing` array"
- Question: "What's the data model for linking friction → guidance?"
- Topic: skillc build-time verification, NOT AI model selection

The word "model" in "data model" triggered the synthesis match, but the investigation is about skill.yaml schema design.

**Source:** Read first 100 lines of `2026-01-08-inv-design-data-model-load-bearing.md`

**Significance:** kb reflect's synthesis detection matched on the keyword "model" without semantic understanding of context.

---

### Finding 3: Other "model" investigations are also false positives

**Evidence:** Files matching `*model*.md` in `.kb/investigations/`:

| File | Actual Topic | AI Model Selection? |
|------|--------------|---------------------|
| `2026-01-04-inv-implement-priority-cascade-model-dashboard.md` | Dashboard status determination | ❌ |
| `2026-01-04-design-dashboard-agent-status-model.md` | Agent status state machine | ❌ |
| `2026-01-04-inv-phase-consolidate-agent-status-model.md` | UI display state consolidation | ❌ |
| `2025-12-27-inv-completion-escalation-model-completed-agents.md` | Completion workflow escalation | ❌ |
| `2026-01-08-inv-design-data-model-load-bearing.md` | skillc schema design | ❌ |

**Source:** Read D.E.K.N. summaries of each file

**Significance:** The word "model" has multiple meanings in this codebase - AI model, data model, status model, escalation model. Keyword matching alone is insufficient.

---

### Finding 4: One listed investigation doesn't exist

**Evidence:** `2025-12-24-inv-test-gemini-flash-model-resolution.md` is listed in spawn context but file does not exist:
```bash
$ [ -f ".kb/investigations/2025-12-24-inv-test-gemini-flash-model-resolution.md" ] && echo EXISTS || echo MISSING
MISSING
```

**Source:** Direct file existence check

**Significance:** The spawn context included a phantom file, possibly from cached/stale kb reflect output.

---

## Synthesis

**Key Insights:**

1. **Prior synthesis is complete** - The Jan 6 synthesis already produced `.kb/guides/model-selection.md` covering all 10 AI model selection investigations. No new AI model investigations have been created since then.

2. **"Model" is a polysemous keyword** - The word "model" appears in at least 5 different contexts in this codebase: AI model selection, data model (schema), status model (state machine), escalation model (workflow), display model (UI state). Simple keyword matching conflates these.

3. **Deduplication check failed** - This synthesis task shouldn't have been created. The `2026-01-06-inv-synthesize-model-investigations-10-synthesis.md` exists and is complete. kb reflect's `synthesisIssueExists()` either didn't run or failed to detect the prior synthesis.

**Answer to Investigation Question:**

**No**, there are no new AI model selection investigations since the Jan 6 synthesis that need consolidation. The `.kb/guides/model-selection.md` guide is current and complete.

The "11 investigations" count is inflated by:
- False positive matches on "model" meaning different things
- One non-existent file in the list
- Failure to recognize the prior synthesis was complete

---

## Structured Uncertainty

**What's tested:**

- ✅ Prior synthesis exists (verified: read `.kb/guides/model-selection.md` - 326 lines)
- ✅ `2026-01-08-inv-design-data-model-load-bearing.md` is about skillc, not AI models (verified: read D.E.K.N.)
- ✅ `2025-12-24-inv-test-gemini-flash-model-resolution.md` does not exist (verified: file existence check)
- ✅ Other "model" investigations are about status/escalation/data models (verified: read D.E.K.N. summaries)

**What's untested:**

- ⚠️ Root cause of why synthesis task was created despite prior synthesis existing (hypothesis: dedup check failed on JSON parse)
- ⚠️ Whether kb reflect should use topic tags instead of keyword matching (not designed)

**What would change this:**

- Finding would be wrong if a new AI model selection investigation was created on Jan 7-8 (verified: none created)
- Finding would be wrong if the guide is stale (verified: last updated Jan 6, no new AI model changes since)

---

## Proposed Actions

### Archive Actions
| ID | Target | Reason | Approved |
|----|--------|--------|----------|
| (none) | | Prior synthesis investigations not ready for archival - guide references them | |

### Create Actions
| ID | Type | Title | Description | Approved |
|----|------|-------|-------------|----------|
| C1 | issue | "kb reflect false positive on polysemous keywords" | "model" matches 5+ meanings; need semantic awareness or topic tags | [ ] |
| C2 | issue | "kb reflect synthesis dedup check failed to detect prior synthesis" | Created orch-go-fx0pg despite 2026-01-06 synthesis being complete | [ ] |

### Update Actions
| ID | Target | Change | Reason | Approved |
|----|--------|--------|--------|----------|
| U1 | `.kb/guides/model-selection.md` | Add "Last verified: Jan 8, 2026" | Confirms guide is still current after review | [ ] |

**Summary:** 3 proposals (0 archive, 2 create, 1 update)
**High priority:** C2 (dedup failure), C1 (false positives)

---

## References

**Files Examined:**
- `.kb/guides/model-selection.md` - The authoritative guide from Jan 6 synthesis
- `.kb/investigations/2026-01-06-inv-synthesize-model-investigations-10-synthesis.md` - Prior synthesis investigation
- `.kb/investigations/2026-01-08-inv-design-data-model-load-bearing.md` - False positive (skillc data model)
- `.kb/investigations/2026-01-04-inv-implement-priority-cascade-model-dashboard.md` - False positive (dashboard status)
- `.kb/investigations/2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md` - Related dedup bug investigation

**Commands Run:**
```bash
# Check file existence of claimed investigations
for f in "file1.md" "file2.md" ...; do [ -f "$f" ] && echo EXISTS || echo MISSING; done

# List all investigations matching "model"
ls -la .kb/investigations/ | grep model

# Glob for model-related investigations
glob ".kb/investigations/*model*.md"
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Guide:** `.kb/guides/model-selection.md` - The already-complete synthesis output
- **Investigation:** `.kb/investigations/2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md` - Documents JSON parse failure causing duplicate issues
- **Prior Synthesis:** `.kb/investigations/2026-01-06-inv-synthesize-model-investigations-10-synthesis.md` - The complete synthesis this task duplicates

---

## Investigation History

**2026-01-08 ~09:00:** Investigation started
- Initial question: Are there 11 model investigations needing synthesis?
- Context: kb reflect spawned synthesis task via daemon

**2026-01-08 ~09:15:** Discovered prior synthesis is complete
- Read `.kb/guides/model-selection.md` - full 326-line guide exists
- Read prior synthesis investigation - marked Complete on Jan 6

**2026-01-08 ~09:30:** Identified false positives
- `2026-01-08-inv-design-data-model-load-bearing.md` is about skillc schema
- Multiple "model" investigations are about status/escalation models, not AI models
- One file in list doesn't exist

**2026-01-08 ~09:45:** Investigation completed
- Status: Complete
- Key outcome: No synthesis needed - prior guide is current; task was false positive due to keyword matching and dedup failure
