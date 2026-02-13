<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added push batching guidance to orchestrator skill and cross-repo issue handoff protocol to worker-base skill.

**Evidence:** Orchestrator was pushing after every `orch complete` (observed in price-watch: OauthToken pushed before ScsOauthClient). Workers failed to create cross-repo issues due to shell sandbox restriction on `cd`.

**Knowledge:** Push decisions belong to orchestrator (not per-completion). Cross-repo issue creation requires structured handoff blocks since workers can't escape their sandbox.

**Next:** Close — guidance is in place in both deployed and source skill files.

**Authority:** implementation - Tactical guidance additions within existing skill framework, no architectural changes

---

# Investigation: Bundle Fix Push Batching Guidance

**Question:** How should orchestrator push strategy and cross-repo issue handoff be documented in skills?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** bundle-fix agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Extracted-From:** orch-go-ga6 (push batching P1), orch-go-4mn (cross-repo issue creation P2)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-01-16-inv-workers-pushing-remote-no-push | extends | Yes - confirmed no-push rule exists in spawn context | None |

---

## Findings

### Finding 1: Orchestrator push-per-completion causes partial deploys

**Evidence:** In price-watch, orchestrator pushed after completing OauthToken model agent before ScsOauthClient (which uses it) was built. Remote received non-functional partial infrastructure.

**Source:** orch-go-ga6 issue description, AGENTS.md line 24 ("Work is NOT complete until git push succeeds" — session-end guidance misapplied per-completion)

**Significance:** Need explicit push batching guidance distinguishing session-end push from mid-chain commits.

---

### Finding 2: Workers cannot create cross-repo beads issues

**Evidence:** `bd create` fails with ENOENT when workers try to `cd` to another project directory. Shell sandbox restriction prevents directory changes outside project root.

**Source:** orch-go-4mn issue description, prior decision on cross-project epics (Option A: epic in primary repo, ad-hoc spawns with --no-track)

**Significance:** Need structured handoff protocol — workers output blocks, orchestrator picks up during completion review.

---

### Finding 3: Existing no-push guidance was already in spawn context

**Evidence:** Prior investigation (2026-01-16) added "NEVER run git push" to SPAWN_CONTEXT template. But orchestrator skill lacked push *batching* strategy (when to push vs accumulate).

**Source:** /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-16-inv-workers-pushing-remote-no-push.md

**Significance:** The worker-side was fixed (don't push). The orchestrator-side needed fixing (when to push).

---

## Synthesis

**Key Insights:**

1. **Push timing is orchestrator responsibility** - Workers commit locally. Orchestrator decides when accumulated commits form a coherent, verifiable changeset worth pushing.

2. **Cross-repo issues need structured handoff** - Since workers are sandboxed, the CROSS_REPO_ISSUE block format gives orchestrator actionable structured data during completion review.

3. **Both deployed and source files need editing** - Deployed skill files take immediate effect but are overwritten by `skillc deploy`. Source files in orch-knowledge ensure durability.

**Answer to Investigation Question:**

Push batching guidance was added as "Push Strategy (Batch Coherent Changesets)" subsection in the orchestrator skill's SYNTHESIZE section. Cross-repo issue handoff was added as "Cross-Repo Issue Handoff" subsection in worker-base's Discovered Work section, with corresponding "Cross-Repo Issue Pickup" guidance in the orchestrator skill. Both deployed and source files were edited.

---

## Structured Uncertainty

**What's tested:**

- ✅ Deployed skill files updated with new sections (verified: edited and read back)
- ✅ Source skill files updated for durability (verified: edited orch-knowledge source files)
- ✅ Prior no-push investigation findings confirmed (verified: read investigation, findings align)

**What's untested:**

- ⚠️ Orchestrator will actually batch pushes (behavioral change, needs observation)
- ⚠️ Workers will output CROSS_REPO_ISSUE blocks (need real cross-repo scenario)
- ⚠️ Orchestrator will pick up CROSS_REPO_ISSUE blocks during completion (need real scenario)

**What would change this:**

- Finding would be wrong if orchestrator ignores push strategy guidance under pressure
- Finding would be wrong if CROSS_REPO_ISSUE block format is too verbose/complex for workers to use

---

## References

**Files Examined:**
- /Users/dylanconlin/.opencode/skill/meta/orchestrator/SKILL.md - Orchestrator skill (deployed)
- /Users/dylanconlin/.opencode/skill/shared/worker-base/SKILL.md - Worker-base skill (deployed)
- /Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md - Orchestrator source
- /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc/discovered-work.md - Worker-base source (discovered work part)
- /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-16-inv-workers-pushing-remote-no-push.md - Prior investigation

**Related Artifacts:**
- **Issue:** orch-go-ga6 - Push batching (P1)
- **Issue:** orch-go-4mn - Cross-repo issue creation (P2)
- **Investigation:** 2026-01-16-inv-workers-pushing-remote-no-push - Prior no-push work

---

## Investigation History

**2026-02-13:** Investigation started
- Initial question: How to add push batching and cross-repo issue handoff guidance to skills
- Context: Bundle fix for two issues (orch-go-ga6 P1, orch-go-4mn P2)

**2026-02-13:** Implementation complete
- Added 3 new sections across orchestrator and worker-base skills
- Edited both deployed and source files
- Status: Complete
