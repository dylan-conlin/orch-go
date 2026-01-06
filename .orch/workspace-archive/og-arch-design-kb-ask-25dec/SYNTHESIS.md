# Session Synthesis

**Agent:** og-arch-design-kb-ask-25dec
**Issue:** orch-go-do59
**Duration:** 2025-12-25 ~30 minutes
**Outcome:** success

---

## TLDR

Designed `kb ask` command for inline mini-investigations. It should be a kb-cli command (not orch) that composes `kb context` + LLM synthesis, with strict provenance enforcement (no context = no answer) and tiered save options (--kn, --save).

---

## Delta (What Changed)

### Files Created
- None (design investigation only)

### Files Modified
- `.kb/investigations/2025-12-25-inv-design-kb-ask-command-inline.md` - Completed investigation with full synthesis

### Commits
- (pending - will commit investigation file)

---

## Evidence (What Was Observed)

- `kb-cli/cmd/kb/context.go` (652 lines) already has complete retrieval infrastructure for kn entries and kb artifacts - synthesis layer is all that's needed
- `orch-go/cmd/orch/main.go:265` shows `orch ask` is alias for `orch send` (message to agent) - different semantics than proposed `kb ask` (knowledge synthesis)
- `~/.kb/principles.md` Provenance principle prohibits ungrounded LLM answers - "Every conclusion must trace to something outside the conversation"
- Gate Over Remind principle supports explicit save flags over post-hoc prompts

### Tests Run
```bash
# Verified no LLM integration in kb-cli currently
grep -rn "LLM\|claude\|anthropic" kb-cli/cmd/kb/
# No matches - confirms this is new capability

# Confirmed existing ask command semantics
grep -n "askCmd" orch-go/cmd/orch/main.go
# Line 265: "Send a message to an existing session (alias for send)"
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-inv-design-kb-ask-command-inline.md` - Full design investigation

### Decisions Made
- Decision 1: `kb ask` should be kb-cli command (kb owns knowledge, orch owns coordination)
- Decision 2: No `--force` flag - Provenance principle is non-negotiable
- Decision 3: Three tiers: ephemeral (default), --kn (quick), --save (full)
- Decision 4: Use OpenCode for LLM integration (consistent with ecosystem)

### Constraints Discovered
- Provenance principle absolutely prohibits LLM answers without kb context grounding
- kb-cli currently has no LLM integration - OpenCode integration pattern is the path

### Externalized via `kn`
- (None - insights captured in investigation file)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file)
- [x] Design recommendation clear and actionable
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-do59`

### Follow-up Work (via features.json)
**Feature:** `kb ask` command implementation
**Skill:** feature-impl
**Location:** kb-cli repo
**Context:** See investigation for interface design and implementation sequence

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to compute optimal N for context window dynamically
- Whether to cache context between rapid-fire questions (session mode?)
- Integration testing approach for OpenCode-dependent commands

**Areas worth exploring further:**
- Response latency benchmarks with different LLM integration approaches
- Token cost implications for frequent `kb ask` usage

**What remains unclear:**
- Exact OpenCode CLI invocation pattern for non-interactive synthesis
- How to handle OpenCode not running (graceful degradation)

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-kb-ask-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-design-kb-ask-command-inline.md`
**Beads:** `bd show orch-go-do59`
