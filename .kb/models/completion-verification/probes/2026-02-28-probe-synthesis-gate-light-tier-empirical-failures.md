# Probe: SYNTHESIS.md Gate Fails 3/3 Light-Tier Feature-Impl Agents (Feb 28 Empirical)

**Model:** Completion Verification Architecture
**Date:** 2026-02-28
**Status:** Complete
**Beads:** orch-go-4czz
**Extends:** 2026-02-27-probe-light-tier-v2-verification-conflict.md

---

## Question

The Feb 27 probe identified that light tier + V2 verify_level creates a contradictory instruction path for feature-impl agents: SPAWN_CONTEXT says "SYNTHESIS.md NOT required" but `orch complete` runs synthesis gate at V1+. Has this unresolved conflict caused real agent failures in production?

---

## What I Tested

### 1. Examined all 3 failed agents from Feb 28

Checked workspace directories, AGENT_MANIFEST.json, beads comments, and FAILURE_REPORT.md for:
- orch-go-i9qr (price consolidation, feature-impl)
- orch-go-vem4 (daemon orphan detection, feature-impl)
- orch-go-amty (kb-cli minimum score, feature-impl)

```bash
# Checked tier files
cat .orch/workspace/archived/og-feat-consolidate-price-watch-28feb-df6a/.tier  # → light
cat .orch/workspace/archived/og-feat-add-daemon-orphan-28feb-79c8/.tier         # → light
cat .orch/workspace/og-feat-kb-cli-add-27feb-201b/.tier                          # → light

# Checked manifests
cat .orch/workspace/archived/og-feat-consolidate-price-watch-28feb-df6a/AGENT_MANIFEST.json
# → tier: "light", verify_level: "V2"
cat .orch/workspace/archived/og-feat-add-daemon-orphan-28feb-79c8/AGENT_MANIFEST.json
# → tier: "light", verify_level: "V2"
cat .orch/workspace/og-feat-kb-cli-add-27feb-201b/AGENT_MANIFEST.json
# → tier: "light", verify_level: "V2"

# Checked SPAWN_CONTEXT for SYNTHESIS mentions
grep -c "SYNTHESIS" .orch/workspace/archived/og-feat-consolidate-price-watch-28feb-df6a/SPAWN_CONTEXT.md
# → 5 mentions, all saying "NOT required"
```

### 2. Examined agent Phase: Complete comments

All three reported detailed, legitimate Phase: Complete comments:
- i9qr (10:51): "Consolidated 25 kb quick entries... Both models updated..."
- vem4 (10:52): "Added daemon orphan recovery... 4 new files... 17 new tests, all passing"
- amty (10:47): "Added StemmedOnly field... Tests: go test... 9 new tests + all existing pass"

None mentioned SYNTHESIS.md — they were told it wasn't required.

### 3. Verified synthesis gate code path

```go
// pkg/verify/check.go:549-576
// V1 gate: Synthesis
if workspacePath != "" && ShouldRunGate(verifyLevel, GateSynthesis) {
    // ShouldRunGate("V2", "synthesis") → true (V2 includes all V1 gates)
    // IsKnowledgeProducingSkill("feature-impl") → false
    // VerifySynthesis(workspacePath) → false (file doesn't exist)
    // → result.Passed = false, "SYNTHESIS.md is missing or empty"
}
```

### 4. Checked for fixes after Feb 27 probe

```bash
git log --oneline --after="2026-02-27" -- pkg/verify/check.go pkg/spawn/verify_level.go
```
No fixes were applied. The Feb 27 probe documented the conflict but no code change was made.

---

## What I Observed

**The Feb 27 probe's prediction was exactly realized on Feb 28.** All three feature-impl agents:

1. Received SPAWN_CONTEXT saying "SYNTHESIS.md is NOT required" (5 mentions per file, all negative)
2. Correctly followed instructions — did their work, reported Phase: Complete with detailed summaries
3. Were rejected by `orch complete` because verify_level=V2 includes GateSynthesis
4. Were abandoned with template-only FAILURE_REPORT.md files (auto-generated, not filled)
5. Were manually closed ~30 minutes later with EXPLAIN-BACK comments

**Timeline for all three:**

| Agent | Phase Complete | Abandoned | Closed | Gap |
|-------|---------------|-----------|--------|-----|
| orch-go-amty | 10:47 | 11:00 | 11:28 | 13 min → abandon |
| orch-go-i9qr | 10:51 | 10:57 | 11:28 | 6 min → abandon |
| orch-go-vem4 | 10:52 | 11:00 | 11:28 | 8 min → abandon |

**Impact:** 3/3 feature-impl completions on Feb 28 failed the synthesis gate. 100% failure rate for the most common spawn type. Work was completed successfully but could not be verified through the standard pipeline.

**The agents were NOT at fault.** They followed their instructions precisely. The system gave them contradictory signals:
- Spawn-time: "SYNTHESIS.md is NOT required" (tier=light)
- Complete-time: "SYNTHESIS.md is missing or empty" (verify_level=V2 → synthesis gate fires)

---

## Model Impact

- [x] **Confirms** invariant violation: The Feb 27 probe's prediction that "every LIGHT feature-impl completion requires `--skip-synthesis`" is confirmed with 3/3 empirical failures.
- [x] **Extends** model with: The conflict has measurable operational cost:
  1. **100% failure rate** for feature-impl completions (most common type)
  2. **Orphaned work**: Agents do real work that can't pass verification
  3. **Manual overhead**: Orchestrator must abandon + manually close
  4. **Lost knowledge**: FAILURE_REPORT.md files are template-only (no human fills them during batch abandonment)

**Root cause chain (confirmed):**
1. Verify-level system (V0-V3) was designed to REPLACE tier system
2. Migration was incomplete: SPAWN_CONTEXT template still uses tier for SYNTHESIS.md instructions
3. `pkg/verify/check.go:550` comment says "replaces tier-based 'light' skip" — but the tier-based agent instructions were NOT replaced
4. `pkg/spawn/config.go` still maps `feature-impl → TierLight`
5. `pkg/spawn/verify_level.go` maps `feature-impl → VerifyV2`
6. These two mappings disagree on synthesis requirements

**Two possible fixes (architect should decide):**
1. **Add tier-based skip to synthesis gate** (like line 388's `&& tier != "light"` for architectural choices) — quick fix, preserves both systems
2. **Align tier defaults with verify levels** — feature-impl → TierFull since V2 requires synthesis — removes contradiction, forces agents to write SYNTHESIS.md

---

## Notes

- The Feb 27 probe (orch-go-i9qi) identified this exact issue. No code fix was applied in the 24 hours between probe and these failures.
- The `GateArchitecturalChoices` gate at check.go:388 already has `&& tier != "light"` — someone previously hit this class of problem for a different gate but didn't generalize the fix.
- The issue affects ALL feature-impl agents, not just these three. Every feature-impl completion since V2 was introduced either required `--skip-synthesis` or failed.
