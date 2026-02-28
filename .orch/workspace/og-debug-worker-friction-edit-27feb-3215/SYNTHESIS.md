# Session Synthesis

**Agent:** og-debug-worker-friction-edit-27feb-3215
**Issue:** orch-go-6al9
**Outcome:** success

---

## Plain-Language Summary

Claude Code's Edit tool fails on tab-indented files (Svelte, Go) because the Read tool's line-number prefix uses a tab delimiter that visually collides with content tabs. Workers can't distinguish where the prefix ends and content begins, leading to wrong tab counts in `old_string`. The fix adds guidance to orch-go's CLAUDE.md teaching workers to use `cat -vet` to see exact tab characters before editing, plus fallback strategies (Write tool, sed, multi-line context). A cross-repo issue was created for the worker-base skill template to give this guidance to all workers across projects.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for reproduction evidence and verification.

---

## TLDR

Added "Tab-Indented File Editing" section to CLAUDE.md with workaround guidance for the Claude Code Edit tool / Read tool tab collision on Svelte and Go files. This is a Claude Code limitation we mitigate via documentation — workers are told to verify whitespace with `cat -vet` before editing tab-indented files, and given fallback strategies when Edit fails.

---

## Delta (What Changed)

### Files Modified
- `CLAUDE.md` - Added "Edit tool + tab indentation" gotcha entry and new "Tab-Indented File Editing" section with problem description, affected files, ranked workarounds, and prevention guidance

### Files Created
- `.orch/workspace/og-debug-worker-friction-edit-27feb-3215/SYNTHESIS.md` - This file
- `.orch/workspace/og-debug-worker-friction-edit-27feb-3215/VERIFICATION_SPEC.yaml` - Verification evidence

---

## Evidence (What Was Observed)

- Read tool output for `web/src/lib/components/ui/badge/badge.svelte` line 2: `     2→	import { cn }...` — tab delimiter (`→`) immediately followed by content tab, visually indistinguishable
- `cat -vet` on same file shows `^Iimport { cn }...` — exactly ONE tab, countable and unambiguous
- Line 7 (double-indented): Read output `     7→		variant?: Variant;` vs `cat -vet` shows `^I^Ivariant?: Variant;` — two tabs clearly visible
- All 55 `.svelte` files in `web/src/` use tab indentation (confirmed via `cat -vet` on `+page.svelte`)
- Go files also use tabs per `gofmt` convention

### Root Cause
The Read tool format is: `[spaces][line_number][TAB_DELIMITER][actual_file_content]`. When file content starts with tab indentation, the delimiter tab and content tabs are adjacent. LLMs process this as text and cannot reliably count adjacent identical characters, leading to `old_string` with wrong leading tab count.

### Why Edit Fails
1. Agent reads file, sees `     2→	import...`
2. Agent constructs `old_string` — must include exactly the content AFTER the delimiter tab
3. Content starts with a tab (indentation), which is adjacent to the delimiter tab
4. Agent miscounts: either omits the content tab or includes the delimiter tab
5. Edit tool: "String to replace not found"
6. Agent retries 2-3x with same wrong approach, wasting tokens
7. Falls back to Write (small files) or `cat -vet` + Python one-liners

---

## Architectural Choices

### Documentation mitigation vs code fix
- **What I chose:** CLAUDE.md guidance (documentation approach)
- **What I rejected:** Modifying spawn context template to inject guidance conditionally
- **Why:** This is a Claude Code platform limitation — we can't fix the Read/Edit tool behavior. CLAUDE.md is loaded for every agent session in this project, making it the most reliable injection point. Spawn context conditional injection would add complexity for marginal benefit over CLAUDE.md.
- **Risk accepted:** Workers may not read the guidance section before encountering the problem. However, after the first failure, searching for "tab" or "Edit tool" in CLAUDE.md will surface it.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Claude Code Read tool tab delimiter collision with tab-indented content is a platform limitation — cannot be fixed from orch-go side
- LLMs fundamentally struggle with counting adjacent identical whitespace characters in text format
- The Edit tool's instruction ("preserve exact indentation as it appears AFTER the line number prefix") is correct but insufficient — LLMs can't reliably parse the boundary

---

## Next (What Should Happen)

**Recommendation:** close

### Cross-Repo Follow-up

CROSS_REPO_ISSUE:
  repo: ~/orch-knowledge
  title: "Add tab-indented file Edit guidance to worker-base skill"
  type: task
  priority: 3
  description: "Worker-base skill template should include a section about Claude Code Edit tool failures on tab-indented files. When the Read tool outputs line content, its tab delimiter collides with content tabs in Svelte/Go/Makefile files, causing 'String to replace not found' errors. Workers need guidance: (1) use cat -vet to verify tab count, (2) use multi-line old_string for unambiguous matches, (3) fallback to Write for small files. See orch-go CLAUDE.md 'Tab-Indented File Editing' section for reference text."

### If Close
- [x] All deliverables complete
- [x] CLAUDE.md guidance added
- [x] Reproduction demonstrated and documented
- [x] Ready for `orch complete orch-go-6al9`

---

## Unexplored Questions

- Could a Claude Code hook or plugin intercept Read output and expand tabs to spaces for display? This would eliminate the collision entirely but requires opencode fork changes.
- Are there other tool output formats that cause similar LLM parsing ambiguity? (e.g., grep output with colons in filenames)
- Would an `.editorconfig` enforcing spaces in Svelte files eliminate the problem entirely? This would be a project-level fix but changes coding convention.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-worker-friction-edit-27feb-3215/`
**Beads:** `bd show orch-go-6al9`
