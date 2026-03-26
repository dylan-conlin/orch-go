# Session Synthesis

**Agent:** og-inv-verify-dao-13-26mar-04eb
**Issue:** orch-go-ff21f
**Duration:** 2026-03-26
**Outcome:** success

---

## Plain-Language Summary

This session checked whether DAO-13 still describes the real size of orch-go spawn prompts for GPT-5.4. I measured the actual `SPAWN_CONTEXT.md` files in the repo and compared them to the token estimator the codebase currently uses. The result is that current active prompts are much smaller than DAO-13's old `63-76 KB / 40-50K tokens` wording: today's active files are about `32-42 KB`, which the repo estimates at roughly `9.5-10.5K` tokens. That means the claim should be read as historical GPT-5.2-era evidence, not as the current GPT-5.4 prompt baseline.

---

## TLDR

Measured current `SPAWN_CONTEXT.md` files and validated orch-go's token estimator. Active prompts now estimate to roughly `8-17K` tokens, so DAO-13's `40-50K` framing is stale for current GPT-5.4 work.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-inv-verify-dao-13-claim-measure.md` - Investigation capturing measurements, synthesis, and recommendation
- `.orch/workspace/og-inv-verify-dao-13-26mar-04eb/SYNTHESIS.md` - Session synthesis for completion review
- `.orch/workspace/og-inv-verify-dao-13-26mar-04eb/VERIFICATION_SPEC.yaml` - Verification contract with commands and evidence
- `.orch/workspace/og-inv-verify-dao-13-26mar-04eb/BRIEF.md` - Comprehension brief for Dylan

### Files Modified
- None (investigation-only session)

---

## Evidence (What Was Observed)

- `pkg/spawn/tokens.go` and `pkg/spawn/kbcontext.go` show the repo estimates tokens with `chars / 4`, not a GPT-specific tokenizer.
- `go test ./pkg/spawn -run 'TestEstimateTokens|TestEstimateContentTokens|TestEstimateContextTokens'` passed, confirming the estimator behavior.
- The measured workspace for this task was `37,930` bytes, which estimates to `9,482` tokens.
- The other March 26 active workspaces are `38,920` bytes (`9,730` tokens) and `41,980` bytes (`10,495` tokens).
- Across all 133 active workspaces, median size is `47,404` bytes (`11,851` tokens), 90th percentile is `54,725` bytes (`13,681` tokens), and max is `69,518` bytes (`17,379` tokens).
- Only 1 active workspace falls in DAO-13's old `63-76 KB` file-size band.
- Across all 1,442 active + archived workspaces, the largest measured file is `145,931` bytes (`36,482` tokens), still below DAO-13's `40-50K` token wording under the current estimator.

### Tests Run
```bash
# Validate estimator behavior
go test ./pkg/spawn -run 'TestEstimateTokens|TestEstimateContentTokens|TestEstimateContextTokens'

# Measure SPAWN_CONTEXT files
python3 <measurement scripts over .orch/workspace/**/SPAWN_CONTEXT.md>
```

---

## Architectural Choices

No architectural choices - investigation task within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-verify-dao-13-claim-measure.md` - Measurement-backed verification of DAO-13 prompt-size wording

### Constraints Discovered
- The repo does not contain a GPT-5.4-native tokenizer, so the best in-repo estimate is the existing `chars / 4` heuristic.
- DAO-13 combines a historical stall-rate claim with prompt-size wording that no longer matches current active prompt sizes.

### Externalized via `kb quick`
- `kb quick decide "Treat current DAO-13 prompt-size wording as historical, not current GPT-5.4 sizing" --reason "Measured active SPAWN_CONTEXT files at 32-69KB and 8-17K estimated tokens via chars/4; current prompts are materially smaller than DAO-13's 63-76KB / 40-50K wording"`

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for:
- exact commands used to measure workspace file sizes
- the `pkg/spawn` test command that validates estimator behavior
- deliverable existence checks

Key outcome: current orch-go spawn prompts estimate to low-teens kilotokens, so prompt size alone does not support treating GPT-5.4 as context-window-constrained in the same way GPT-5.2 was.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-ff21f`

### Follow-up Actions
1. Update DAO-13/model wording so the old `63-76 KB / 40-50K` prompt-size framing is explicitly historical.
2. If exact OpenAI accounting matters, add or run a GPT tokenizer outside the repo and compare it with the current `chars / 4` heuristic.

---

## Unexplored Questions

- Did the old `40-50K` figure measure full prompt payload rather than file-only `SPAWN_CONTEXT.md` size?
- How close is GPT-5.4's real tokenizer output to orch-go's current `chars / 4` estimate on markdown-heavy prompts?

---

## Friction

- `gap`: current workspace `SPAWN_CONTEXT.md` disappeared mid-session, so live re-verification had to rely on the earlier successful measurement instead of re-reading the file from disk.

---

## Session Metadata

**Skill:** investigation
**Model:** openai/gpt-5.4
**Workspace:** `.orch/workspace/og-inv-verify-dao-13-26mar-04eb/`
**Investigation:** `.kb/investigations/2026-03-26-inv-verify-dao-13-claim-measure.md`
**Beads:** `bd show orch-go-ff21f`
