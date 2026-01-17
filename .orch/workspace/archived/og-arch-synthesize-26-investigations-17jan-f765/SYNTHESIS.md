# Session Synthesis

**Agent:** og-arch-synthesize-26-investigations-17jan-f765
**Issue:** orch-go-exx40
**Duration:** 2026-01-17 00:55 → 2026-01-17 01:50
**Outcome:** success

---

## TLDR

Synthesized 26+ completion/verification investigations, identified 4 root causes of churn (agent-scoping ambiguity, gate proliferation, evidence-vs-claim conflation, legacy compatibility), and proposed Agent Manifest architecture with git-based scoping to reduce false positives and eliminate concurrent agent pollution.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-design-synthesize-26-completion-investigations.md` - Comprehensive analysis of completion verification churn with architectural recommendations

### Files Modified
- None (this is an investigation/design session)

### Commits
- Pending: Investigation file with synthesis findings and recommendations

---

## Evidence (What Was Observed)

### Source Review
- Read `pkg/verify/check.go` - Found 12 distinct gate constants (lines 12-26)
- Read `.kb/models/completion-verification.md` - Found evolution through 6 phases (Dec 2025 - Jan 2026)
- Read `.kb/guides/completion.md` - Found 330-line guide documenting verification workflow

### Investigation Analysis
- Jan 8, 2026 Completion Synthesis: 10 investigations analyzed, 4 evolution phases identified
- Jan 14, 2026 Verification Synthesis: 25 investigations analyzed, 4 verification layers identified
- Jan 14 git_diff investigation: Missing spawn_time causes wrong git command → false positives
- Jan 8 test_evidence investigation: Concurrent agents pollute spawn-time-based checks
- Jan 14 targeted-skip-flags: 55% of completions used --force bypass

### Key Metrics
- 12 distinct verification gates in pkg/verify/check.go
- 55% of completions bypassed gates with --force (per Jan 14 investigation)
- 28 workspaces lack .spawn_time files (legacy compatibility burden)
- True completion rate ~89% vs reported 72% (metrics artifact)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-design-synthesize-26-completion-investigations.md` - Root cause analysis and architectural recommendations

### Root Causes Identified (4)

1. **Agent-Scoping Ambiguity** - No canonical agent identity; gates use imperfect proxies (spawn time, workspace path, beads ID) that fail for concurrent agents, cross-project work, or missing metadata

2. **Gate Proliferation** - 12 gates × N edge cases = multiplicative complexity; each gate independently implements agent-scoping, causing inconsistent behavior

3. **Evidence vs Claim Conflation** - Gates inconsistently check claims (keywords) vs evidence (actual files); claim-based gates unreliable, evidence-based gates hard to scope

4. **Legacy Workspace Compatibility** - "Skip if missing" patterns create two verification regimes; degraded verification quality for older workspaces

### Architectural Recommendation
**Agent Manifest + Git-Based Scoping + Evidence Collection Phase**
- Create canonical agent identity at spawn time (AGENT_MANIFEST.json)
- Use git commit SHA as baseline, not time-based scoping
- Collect evidence before running gates, gates operate on collected evidence
- Future: Gate consolidation and legacy cleanup

### Constraints Discovered
- Spawn time is insufficient for agent scoping (fails for concurrent agents)
- Claim-based verification (keywords in comments) produces false positives
- Legacy workspaces will behave differently until migrated

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with D.E.K.N. summary)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for orchestrator review

### Recommended Follow-up Actions
1. **Update Model**: Incorporate findings into `.kb/models/completion-verification.md` (add "Churn Causes" section)
2. **Create Decision**: If accepting Agent Manifest recommendation, create `.kb/decisions/verification-agent-manifest.md`
3. **Implement Phase 1**: Agent Manifest at spawn time (foundational for other improvements)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to handle cross-project spawns in Agent Manifest? (project_dir detection at spawn time vs verification time)
- Whether git author filtering works reliably across all spawn modes (Claude CLI vs OpenCode)
- Optimal gate count - is 12 right, or should some be consolidated?

**Areas worth exploring further:**
- Git commit-based scoping implementation (record SHA at spawn, diff at verify)
- Evidence collection data structures (what format enables reliable gate checking)
- Migration path for legacy workspaces (auto-generate manifests? or sunset?)

**What remains unclear:**
- Current force-bypass rate (55% mentioned in investigation, but is that current?)
- Whether author/committer metadata is reliable in current workflow

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-synthesize-26-investigations-17jan-f765/`
**Investigation:** `.kb/investigations/2026-01-17-inv-design-synthesize-26-completion-investigations.md`
**Beads:** `bd show orch-go-exx40`
