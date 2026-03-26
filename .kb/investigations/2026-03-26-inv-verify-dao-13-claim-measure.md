## Summary (D.E.K.N.)

**Delta:** Current `SPAWN_CONTEXT.md` files in active workspaces are much smaller than DAO-13's cited 63-76 KB band, and the repo's own estimator puts them at roughly 8-17K tokens rather than 40-50K.

**Evidence:** Measured all `.orch/workspace/**/SPAWN_CONTEXT.md` files with a Python script and validated the estimator with `go test ./pkg/spawn -run 'TestEstimateTokens|TestEstimateContentTokens|TestEstimateContextTokens'`.

**Knowledge:** DAO-13's size/token framing reflects older GPT-5.2-era prompt conditions, not the current prompt distribution or the current `chars/4` estimator used by orch-go.

**Next:** Close this investigation and update DAO-13/model text separately so historical GPT-5.2 observations are not presented as current GPT-5.4 prompt sizing.

**Authority:** architectural - this changes how a shared model claim is framed for future routing and benchmarking decisions.

## TLDR

I measured the actual `SPAWN_CONTEXT.md` files in this repo and checked the token estimator the codebase currently uses. Today's active spawn contexts are about 32-42 KB for the March 26 sessions and 9.5-10.5K estimated tokens, so the DAO-13 wording about 63-76 KB / 40-50K GPT tokens looks historical and stale rather than representative of current prompts.

---

# Investigation: Verify DAO-13 Claim Measure

**Question:** What are the actual `SPAWN_CONTEXT.md` file sizes in the current orch-go workspace set, and what GPT-5.4 token count should we estimate from those files using the repo's current token estimation logic?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** OpenCode (`og-inv-verify-dao-13-26mar-04eb`)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** daemon-autonomous-operation

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-23-inv-investigate-revisit-opencode-model-routing.md` | extends | yes | That investigation repeated DAO-13's older `63-76KB` / `40-50K` framing; current measurements are materially smaller. |
| `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` | confirms | yes | No conflict; its `~8K tokens` comment is directionally consistent with the current estimator and current-day prompts. |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## What I tried

1. Read DAO-13 and the surrounding model/investigation artifacts to locate the exact historical claim.
2. Read the current token estimator in `pkg/spawn/tokens.go` and `pkg/spawn/kbcontext.go`.
3. Measured every `.orch/workspace/**/SPAWN_CONTEXT.md` file in this repo with a Python script.
4. Ran the spawn token-estimation tests in `pkg/spawn` to verify the estimator behavior.

## What I observed

- The current estimator is explicitly `chars / 4`, not a GPT-specific tokenizer.
- The current workspace for this task is `37,930` bytes, which estimates to `9,482` tokens.
- The other March 26 active workspaces are `38,920` bytes (`9,730` tokens) and `41,980` bytes (`10,495` tokens).
- Across all 133 active workspaces, the median `SPAWN_CONTEXT.md` size is `47,404` bytes (`11,851` tokens), the 90th percentile is `54,725` bytes (`13,681` tokens), and the maximum is `69,518` bytes (`17,379` tokens).
- Only 1 active workspace falls inside DAO-13's old `63-76 KB` size band.
- Across all 1,442 archived + active workspaces, the maximum observed file is `145,931` bytes (`36,482` tokens), still below DAO-13's quoted `40-50K` token range when using the current estimator.

## Test performed

```bash
# Validate estimator behavior in code
go test ./pkg/spawn -run 'TestEstimateTokens|TestEstimateContentTokens|TestEstimateContextTokens'

# Measure current workspace and historical SPAWN_CONTEXT.md files
python3 - <<'PY'
from pathlib import Path
root = Path('/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace')
files = [p for p in root.glob('**/SPAWN_CONTEXT.md') if p.is_file()]
rows = []
for p in files:
    size = p.stat().st_size
    rows.append((size, size // 4, '/archived/' not in str(p), p))
PY
```

---

## Findings

### Finding 1: The repo's only token estimator is `chars / 4`

**Evidence:** `EstimateTokens` returns `charCount / CharsPerToken`, and `CharsPerToken` is defined as `4` with the comment "Claude typically uses ~4 chars per token for English text." The unit tests assert `4000 chars -> 1000 tokens` and `400000 chars -> 100000 tokens`.

**Source:** `pkg/spawn/tokens.go:80`, `pkg/spawn/kbcontext.go:31`, `pkg/spawn/tokens_test.go:8`, `go test ./pkg/spawn -run 'TestEstimateTokens|TestEstimateContentTokens|TestEstimateContextTokens'`

**Significance:** The current codebase does not have a GPT-5.4-specific tokenizer estimate. The most defensible estimate available inside the repo is the shared `chars / 4` heuristic.

---

### Finding 2: Current March 26 spawn contexts are roughly 38-42 KB, not 63-76 KB

**Evidence:** The three active March 26 workspaces measure `37,930`, `38,920`, and `41,980` bytes, corresponding to `9,482`, `9,730`, and `10,495` estimated tokens.

**Source:** Python measurement over `.orch/workspace/*/SPAWN_CONTEXT.md`; `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-verify-dao-13-26mar-04eb/SPAWN_CONTEXT.md`

**Significance:** The current GPT-5.4 benchmark/verification spawns are nowhere near the file-size band DAO-13 cites.

---

### Finding 3: DAO-13's size/token framing is historical upper-band data, not current typical prompt sizing

**Evidence:** Active workspaces have median `47,404` bytes (`11,851` tokens) and max `69,518` bytes (`17,379` tokens); only 1 active file lands inside `63-76 KB`. Across all archived + active workspaces, the max file is `145,931` bytes (`36,482` tokens), which still stays below DAO-13's `40-50K` token estimate under the repo's current estimator.

**Source:** Python measurement across all `.orch/workspace/**/SPAWN_CONTEXT.md`; `.kb/models/daemon-autonomous-operation/claims.yaml:230`; `.kb/models/daemon-autonomous-operation/model.md:371`; `.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md:126`

**Significance:** DAO-13 is still useful as a historical GPT-5.2 stall claim, but its prompt-size wording should be reframed as historical evidence rather than as the current GPT-5.4 prompt reality.

---

## Synthesis

**Key Insights:**

1. **Current prompts are smaller than the claim remembers** - the current spawn-context population is mostly in the low-teens kilotoken range under the repo's estimator, not 40-50K tokens.

2. **The estimator and the claim speak different languages** - orch-go currently estimates with a universal `chars / 4` heuristic, while DAO-13's wording assumes a much denser GPT tokenization ratio that is not implemented or re-measured anywhere in the repo.

3. **The risk moved from context-window exhaustion to protocol compliance** - with GPT-5.4's 1.05M context window and current ~9-17K prompt estimates, prompt size alone no longer looks like the primary blocker.

**Answer to Investigation Question:**

Using orch-go's current estimator, today's active `SPAWN_CONTEXT.md` files are about `8-17K` tokens, with this workspace at `9,482` tokens and the other March 26 active workspaces at `9,730` and `10,495`. The old DAO-13 wording about `63-76 KB` files consuming `40-50K` GPT tokens does not match the current active prompt distribution and is not supported by the repo's present `chars / 4` estimation logic. I did not measure with a model-native GPT-5.4 tokenizer because no such tokenizer exists in this repo, so the conclusion is: current orch-go evidence supports a much smaller prompt estimate, and DAO-13 should be treated as historical GPT-5.2-era evidence unless re-measured with a GPT tokenizer.

---

## Structured Uncertainty

**What's tested:**

- ✅ Current token estimator behavior (`chars / 4`) is verified by code and unit tests.
- ✅ Current March 26 workspace file sizes are directly measured from disk.
- ✅ Active and archived workspace size distributions are directly measured from disk.

**What's untested:**

- ⚠️ GPT-5.4's model-native tokenizer count for these exact files is not measured.
- ⚠️ The old GPT-5.2 `40-50K` estimate was not reproduced against the original tokenizer or original session payload.
- ⚠️ Full initial prompt size including tool schema and system instructions was not recomputed here.

**What would change this:**

- A GPT-5.4 tokenizer run over these same files producing materially higher counts than `chars / 4`.
- Discovery that the historical `40-50K` figure referred to full prompt payload, not just `SPAWN_CONTEXT.md`.
- A new spawn template expansion that pushes active prompts back into the `63-76 KB` band.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Update DAO-13/model wording to distinguish historical GPT-5.2 prompt sizing from current spawn-context measurements | architectural | It changes a shared model used for routing and benchmark interpretation across future sessions. |

### Recommended Approach ⭐

**Historical framing update** - Keep DAO-13 as a historical non-Anthropic reliability claim, but rewrite the prompt-size sentence so it clearly refers to the older GPT-5.2 observations and cites the current `chars / 4` estimator separately.

**Why this approach:**
- It preserves the valuable stall-rate history without presenting stale prompt-size assumptions as current fact.
- It aligns the model with what the codebase actually estimates today.
- It reduces future benchmark confusion when GPT-5.4 reliability is evaluated.

**Trade-offs accepted:**
- The claim becomes more qualified and less rhetorically sharp.
- Exact GPT-5.4 tokenizer counts remain unknown until someone measures them outside the current repo.

**Implementation sequence:**
1. Update DAO-13 evidence text to mark `63-76 KB / 40-50K` as historical GPT-5.2-era evidence.
2. Add a note citing the current `chars / 4` estimator and the measured current workspace distribution.
3. Re-run any benchmark/design docs that quote DAO-13 so they inherit the corrected framing.

### Alternative Approaches Considered

**Option B: Leave DAO-13 unchanged**
- **Pros:** No artifact churn.
- **Cons:** Future GPT-5.4 benchmark work will keep inheriting stale context-size assumptions.
- **When to use instead:** If DAO-13 is intentionally preserved as a frozen historical snapshot.

**Option C: Add a model-native GPT tokenizer tool first**
- **Pros:** Gives more faithful OpenAI-specific prompt counts.
- **Cons:** More work than this investigation required and not needed to establish that current prompt sizes are well below the old claim band.
- **When to use instead:** If routing decisions require exact OpenAI token accounting rather than current repo-level estimates.

**Rationale for recommendation:** Reframing the claim is the smallest change that makes the knowledge base honest again without requiring new infrastructure.

---

### Implementation Details

**What to implement first:**
- DAO-13 wording refresh in the daemon-autonomous-operation model/claim artifacts.
- A one-line note in benchmark docs that current prompts estimate to roughly `9-17K` tokens in active workspaces.

**Things to watch out for:**
- ⚠️ Do not erase the original GPT-5.2 stall evidence; label it as historical instead.
- ⚠️ Keep repo-estimator numbers and model-native-tokenizer numbers distinct if both get cited later.

**Areas needing further investigation:**
- Whether the historical `40-50K` number came from full prompt payload rather than file-only size.
- Whether GPT-5.4/O-series tokenization materially differs from the repo's `chars / 4` heuristic on these markdown-heavy prompts.

**Success criteria:**
- ✅ DAO-13 no longer implies today's active prompts are still `63-76 KB / 40-50K` tokens.
- ✅ Future benchmark docs cite current measurements when discussing GPT-5.4 context usage.

---

## References

**Files Examined:**
- `.kb/models/daemon-autonomous-operation/claims.yaml` - DAO-13 claim text and falsification criteria
- `.kb/models/daemon-autonomous-operation/model.md` - Narrative version of DAO-13
- `.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md` - Historical source of the prompt-size wording
- `.kb/investigations/2026-03-23-inv-investigate-revisit-opencode-model-routing.md` - Recent investigation that repeated the older framing
- `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` - Current benchmark investigation with `~8K tokens` note
- `pkg/spawn/tokens.go` - Token estimation implementation
- `pkg/spawn/kbcontext.go` - `CharsPerToken = 4`
- `pkg/spawn/tokens_test.go` - Tests proving estimator behavior
- `.orch/workspace/**/SPAWN_CONTEXT.md` - Measured files

**Commands Run:**
```bash
# Verify project directory and inspect issue
pwd && bd show orch-go-ff21f

# Create investigation file
kb create investigation verify-dao-13-claim-measure --model daemon-autonomous-operation

# Search prior DAO-13 references
grep DAO-13 across .kb and .beads via repo search tools

# Measure SPAWN_CONTEXT sizes and convert to chars/4 token estimate
python3 <measurement scripts>

# Validate estimator behavior in tests
go test ./pkg/spawn -run 'TestEstimateTokens|TestEstimateContentTokens|TestEstimateContextTokens'
```

**External Documentation:**
- None

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-23-inv-investigate-revisit-opencode-model-routing.md` - Carries the older DAO-13 framing forward
- **Investigation:** `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` - Already points toward much smaller current prompt estimates
- **Workspace:** `.orch/workspace/og-inv-verify-dao-13-26mar-04eb/` - Session workspace and deliverables

---

## Investigation History

**[2026-03-26 08:41]:** Investigation started
- Initial question: Are current `SPAWN_CONTEXT.md` files still large enough to justify DAO-13's `63-76 KB / 40-50K tokens` wording for GPT-5.4?
- Context: DAO-13 is being used to reason about GPT-5.4 viability, so stale prompt-size assumptions would distort the benchmark frame.

**[2026-03-26 08:50]:** Current estimator and current files checked
- Verified that orch-go estimates tokens with `chars / 4` and measured current active workspaces directly from disk.

**[2026-03-26 08:58]:** Investigation completed
- Status: Complete
- Key outcome: Current active spawn contexts estimate to roughly `9-17K` tokens, so DAO-13's prompt-size wording is historical/stale rather than current.
