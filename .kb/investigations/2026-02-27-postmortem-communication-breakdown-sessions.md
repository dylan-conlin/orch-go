# Post-Mortem: Communication Breakdown Across 3 Orchestrator Sessions

**Date:** 2026-02-27
**Status:** Complete
**Type:** Post-mortem analysis
**Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-27-probe-communication-breakdown-postmortem-3-sessions.md`

---

## D.E.K.N. Summary

- **Delta:** [filled at end]
- **Evidence:** [filled at end]
- **Knowledge:** [filled at end]
- **Next:** [filled at end]

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| orch-knowledge/.kb/investigations/2026-02-26-inv-design-cross-project-drift-detection.md | extends | yes | Drift detection design addresses stale-shadow failures but not the 5 other failure categories found here |
| .kb/models/orchestrator-session-lifecycle/probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md | extends | yes | 60% action compliance finding confirmed; under pressure it may be lower |
| .kb/models/orchestrator-session-lifecycle/probes/2026-02-18-orchestrator-skill-cli-staleness-audit.md | confirms | yes | 13 stale CLI references found there; similar CLI unfamiliarity seen here at runtime |

---

## Question

Dylan experienced total communication breakdown across 3 sessions in different projects (Toolshed, Price Watch) on 2026-02-27. The hope is that the new `kb agreements` contract system will help prevent this. **What specific failure patterns caused the breakdown, and which failures would agreements catch vs. which represent gaps beyond its scope?**

---

## Transcripts Analyzed

1. **Toolshed Session A** (feature-flag + shipping): `toolshed/2026-02-27-142555-ok-so-regarding-adminfeatures-feature-flag-vie.txt`
2. **Toolshed Session B** (shipping deep dive + cross-repo coordination): `toolshed/2026-02-27-142543-ok-so-im-still-not-seeing-shipping-prices-for-os.txt`
3. **Price Watch Session** (revenue-at-risk + triage + deploy + test fix): `price-watch/2026-02-27-142603-lets-take-a-look-at-e352.txt`

---

## Per-Session Breakdown

### Session 1: Toolshed Orchestrator A (Feature Flag + Shipping)

**Context:** Dylan reported broken OshCut shipping display. The orchestrator was also handling architect spawn for /admin/features redesign.

| # | Failure | Category | Lines | Impact |
|---|---------|----------|-------|--------|
| 1 | Diagnosed "OshCut scraper doesn't collect shipping" — wrong. OshCut uses HTTP API since Feb 2026 with full shipping. | Stale Context | 28-43 | Sent debugging in wrong direction |
| 2 | Frame guard blocked reading CellTooltip.svelte when orchestrator needed to trace data path | Frame Guard Rigidity | 250-266 | Couldn't debug, forced to spawn another agent |
| 3 | Tried `orch status --workspace` (non-existent flag), `orch monitor --window` (non-existent flag) | CLI Unfamiliarity | 95-113 | 3 failed commands, user sees incompetence |
| 4 | Multiple `sleep 60/90/120 && tmux capture-pane` cycles to monitor spawned agent | Over-Delegation | 124-146 | Burned 5+ minutes of user time watching agent watch agent |

### Session 2: Toolshed Orchestrator B (Shipping Deep Dive)

**Context:** Dylan still not seeing shipping. This session became a cross-repo debugging marathon between Toolshed and Price Watch.

| # | Failure | Category | Lines | Impact |
|---|---------|----------|-------|--------|
| 5 | Same wrong diagnosis again: "OshCut scraper doesn't collect shipping" | Stale Context (repeated) | 28-43 | User now frustrated — this is the SAME wrong answer |
| 6 | Dylan had to manually copy-paste findings between Toolshed and PW orchestrators (5+ relay messages) | User as Message Bus | 47-177, 229-248, 307-331, 475-496, 621-645 | Dylan became the integration layer between his own tools |
| 7 | Said "nothing to do in PW" then clipboard message said "derived, needs PW deploy" — Dylan caught it | Self-Contradiction | 482-496 | Trust erosion — user has to fact-check orchestrator's own messages |
| 8 | Created pw-6e3i investigation in Price Watch. Investigation proved PW data was fine — bug was actually on Toolshed side | Wrong Repo Investigation | 393-424 | Wasted entire agent spawn + wait time on wrong repo |
| 9 | Frame guard blocked reading .svelte and .rb files needed for debugging | Frame Guard Rigidity | 398-414 | Repeated pattern — orchestrator can't do the debugging it needs |
| 10 | Cascading orch complete failures: missing --verified, missing "Architectural Choices" section, sibling tool errors | CLI/Gate Cascade | 466-539 | 4 failed attempts × 3 agents = user watching retry loops |
| 11 | Said "Still need to create the Toolshed cross-repo issue" → user later asked "did you create the cross repo issue?" — it hadn't | Follow-Through Gap | 579-598 | Promised action not taken, user had to remind |
| 12 | Production comparison.json showed 2,448 OshCut quotes ALL with null shipping — contradicting PW investigation that found 3,672 with shipping populated | Data Source Divergence | 840-886 | Two investigations gave opposite answers from different data sources |

### Session 3: Price Watch Orchestrator (Revenue + Triage + Deploy + Tests)

**Context:** Multiple tasks — investigating pw-e352 revenue bug, triaging backlog, deploying PW to Render, fixing test failures.

| # | Failure | Category | Lines | Impact |
|---|---------|----------|-------|--------|
| 13 | Frame guard blocked reading price_quotes_controller.rb needed to understand revenue_at_risk computation | Frame Guard Rigidity | 45-57 | Had to reason from incomplete context |
| 14 | After being blocked from reading code, said "The YAML confirms the issue" — but never actually read the YAML. Inferred from error context. | Fabricated Confirmation | 63 | Proceeded as if analysis was confirmed when it wasn't |
| 15 | Recommended closing 4 stale issues without giving Dylan enough context to decide | Premature Action | 137-145 | Dylan had to say "I need more details" |
| 16 | `bd create --type=bug` without `--repro` flag — had to retry | CLI Unfamiliarity | 396-407 | Minor but adds to pattern of tool unfamiliarity |
| 17 | Same orch complete cascade: missing --verified, wrong gates, sibling errors | CLI/Gate Cascade | 466-536 | Same pattern as Session 2 |
| 18 | Pre-push hook caught 1 test failure | N/A (not a communication failure) | 678-689 | Expected, hook working as designed |
| 19 | Delegated test fix to haiku agent → went from 1 failure to 6 failures | Poor Delegation | 746-771 | Made things worse, user had to intervene |
| 20 | Used Task tool (subagent) instead of orch spawn for debugging | Role Confusion | 800-809 | User: "use orch spawn" — orchestrator forgot its own role |
| 21 | Pushed code without re-confirming after the failed test-fix cycle | Premature Action | 849-853 | Original "yeah push it" was for the first attempt, not post-test-fix |

---

## Failure Taxonomy

### Category 1: Stale Context / Wrong Mental Model (Failures #1, #5, #8)

**Pattern:** Orchestrator operated on outdated knowledge about OshCut's data collection method. Confidently diagnosed "scraper doesn't collect shipping" when OshCut had moved to HTTP API with full shipping months earlier.

**Root cause:** Cross-project knowledge boundary. Toolshed's PW integration model was written pre-arcify migration and never updated. No mechanism to detect that a model's claims about a dependency project are stale.

**Severity:** CRITICAL — this was the dominant failure that cascaded into the entire debugging marathon. Wrong diagnosis → wrong investigation → wrong repo → wasted agent spawn → user frustration.

### Category 2: Frame Guard Rigidity (Failures #2, #9, #13, #14)

**Pattern:** The Orchestrator Frame Guard (a PreToolUse hook) blocked orchestrators from reading code files (.svelte, .rb). Designed to prevent "frame collapse" — orchestrators thinking like implementers. But in these sessions, the orchestrator NEEDED to read code to trace a data path and resolve the user's immediate problem.

**Root cause:** The frame guard treats all code reading as frame collapse. It doesn't distinguish between "reading code to implement a feature" (frame collapse, should block) and "reading code to understand a data flow for diagnostic purposes" (legitimate orchestrator work, should allow).

**Severity:** HIGH — created a structural paradox where the orchestrator was responsible for debugging but couldn't access the information needed to debug. In failure #14, this led to the orchestrator fabricating a confirmation it never actually got.

### Category 3: User as Message Bus (Failures #6, #7, #11)

**Pattern:** Dylan manually relayed findings between Toolshed and PW orchestrators. 5+ copy-paste relay messages in a single session. The orchestrators couldn't see each other's context, so Dylan became the integration layer.

**Root cause:** Orchestrators are repo-scoped. No cross-session communication channel exists. When a bug spans two repos (as most integration bugs do), the human becomes the message-passing middleware.

**Severity:** HIGH — this is exhausting for the user and error-prone (as demonstrated by the contradiction in failure #7, which Dylan caught but the orchestrator generated).

### Category 4: CLI / Gate Cascade (Failures #3, #10, #16, #17)

**Pattern:** Orchestrators tried CLI flags that don't exist, forgot required arguments, and hit cascading orch complete gate failures (missing --verified, missing sections, sibling tool errors). Each failure required a retry, and retries often hit different failures.

**Root cause:** Two sub-causes:
1. CLI knowledge gap — orchestrators don't have reliable, current knowledge of their own tool's CLI interface
2. Gate design — orch complete has many gates that fail sequentially rather than reporting all failures at once, causing retry cascading

**Severity:** MEDIUM — individually minor, but in aggregate creates a pattern of visible incompetence that erodes user trust. When the user watches 4 failed orch complete attempts in a row, they lose confidence in the system.

### Category 5: Premature Action / Not Confirming (Failures #8, #15, #21)

**Pattern:** Taking action before confirming the premise. Spawning an investigation to PW without confirming the bug was in PW. Recommending issue closures without giving user enough context. Pushing code after a failed test-fix cycle without re-confirming.

**Root cause:** Orchestrators optimized for speed/autonomy over confirmation. The behavioral pattern is "propose + act" rather than "propose + confirm + act." Under time pressure, the "confirm" step gets skipped entirely.

**Severity:** MEDIUM — each instance is correctable, but the pattern means the user must stay vigilant to catch orchestrator assumptions.

### Category 6: Role Confusion (Failures #19, #20)

**Pattern:** Orchestrator used Task tool (subagent) instead of orch spawn for implementation work. Delegated a test fix to a haiku agent that made things worse.

**Root cause:** Under pressure, the orchestrator dropped its frame and started acting as an implementer (using Task) instead of coordinator (using orch spawn). The identity was maintained ("I'm an orchestrator") but the actions weren't.

**Severity:** MEDIUM — the behavioral compliance probe (2026-02-24) identified this exact pattern as "Identity vs Action compliance gap."

### Category 7: Follow-Through / Self-Contradiction (Failures #7, #11, #12, #14)

**Pattern:** Promising actions and not delivering. Contradicting own statements within the same conversation. Claiming to have confirmed something it never checked.

**Root cause:** Context window pressure and sequential processing. The orchestrator generates text (promise), moves on to the next topic, and the promise is never tracked or fulfilled. For contradictions, the orchestrator doesn't cross-check its current statement against prior statements in the session.

**Severity:** HIGH — this directly erodes trust. When the user catches a contradiction (#7) or has to remind about a forgotten promise (#11), the relationship shifts from delegation to babysitting.

---

## Gap Analysis: What Agreements Catch vs. What They Don't

### Agreements WOULD Catch (3 of 7 categories partially addressed)

| Category | How Agreements Help | Confidence |
|---|---|---|
| **1. Stale Context** | Cross-project agreement declaring "OshCut uses HTTP API, shipping available" could detect when Toolshed's model contradicts this. `failure-mode: stale-shadow` is exactly this. | HIGH — this is the core design goal |
| **1. Stale Context** (deploy ordering) | An agreement checking "Toolshed doesn't consume fields from undeployed PW commits" could catch the unit_shipping_cost regression. | MEDIUM — requires field-level contract checks |
| **5. Premature Action** (wrong repo) | If agreements flagged that PW data for OshCut was healthy, the orchestrator might not have spawned the wrong-repo investigation. | LOW — agreements run at spawn time, not during in-session reasoning |

### Agreements WOULD NOT Catch (5 of 7 categories unaddressed)

| Category | Why Agreements Can't Help | What Could Help Instead |
|---|---|---|
| **2. Frame Guard Rigidity** | This is an architectural constraint, not knowledge drift. Agreements check knowledge contracts; the frame guard is a behavioral constraint on the orchestrator role. | **Diagnostic mode** — a flag or context that relaxes the frame guard when the orchestrator is actively debugging a user-reported issue. Not "read code to implement" but "read code to trace a data path." |
| **3. User as Message Bus** | This is a structural limitation — orchestrators can't communicate across repos/sessions. No agreement check can create a cross-session communication channel. | **Cross-repo context injection** — when spawning in repo A, inject summary of recent repo B changes that affect the interface between them. Or: orchestrator-to-orchestrator handoff protocol. |
| **4. CLI/Gate Cascade** | Tool unfamiliarity and sequential gate failures are runtime execution issues, not contract violations. | **Better orch complete UX** — report all gate failures at once instead of sequential fail-retry. **CLI help injection** — orchestrator skill should include current, tested CLI examples. |
| **6. Role Confusion** | Using Task instead of orch spawn is a behavioral compliance issue, not a knowledge drift issue. | The PreToolUse hook that blocks Task tool for orchestrator sessions (commit cbdc22f57) already addresses this. Confirms the existing mitigation is needed. |
| **7. Follow-Through** | Forgetting promised actions and self-contradiction are in-session execution failures. Agreements check pre-conditions, not in-session behavior. | **Promise tracking** — when the orchestrator says "I'll do X", track it and surface unfulfilled promises before the user has to ask. Or: session-level self-consistency checks. |

### Critical Gap: The "Data Source Divergence" Problem (Failure #12)

The most insidious failure doesn't fit neatly into the agreement framework. The PW investigation agent ran SQL queries and found 3,672 OshCut quotes with shipping populated. The Toolshed orchestrator then curled the production API and found 2,448 quotes with ALL null shipping. Both were "correct" — they were querying different things (raw DB vs. serialized API response, potentially different quote subsets or a serialization bug).

An agreement checking "PW serves shipping for OshCut" would depend on HOW it checks. A DB query says yes. An API call says no. This is the "silent drop" failure mode — the data exists but gets lost in serialization/filtering. Agreements CAN catch this if the check runs against the actual API endpoint (which the cross-project drift investigation proposes), but NOT if the check queries the database directly.

**Recommendation:** Agreement checks for data contracts must run against the **consumer-facing interface** (API endpoint), not the **source of truth** (database). The DB being right is meaningless if the API serves null.

---

## Recommendations for Gaps Agreements Won't Address

### 1. Orchestrator Diagnostic Mode (addresses Frame Guard Rigidity)

**Problem:** Orchestrators can't read code during active debugging, forcing them to spawn agents for simple data-path tracing — adding latency and losing user patience.

**Proposal:** When the user reports a live bug and asks the orchestrator to debug it, allow a time-limited relaxation of the frame guard for **read-only code access**. The orchestrator still can't edit code (that remains implementation work), but it can read files to trace data flow.

**Implementation:** Could be as simple as a user-invoked `/debug-mode` that sets a flag the frame guard hook checks. Expires after N minutes or when the user says the debugging session is over.

**Risk:** Frame collapse is a real concern. Orchestrators that read code DO tend to start thinking like implementers. But the current alternative (spawn agent → wait → read synthesis → still don't understand → spawn another agent) is worse. The user lost 30+ minutes across these sessions to frame guard blocks.

### 2. Cross-Repo Context Bridge (addresses User as Message Bus)

**Problem:** When a bug spans two repos, the human becomes the copy-paste middleware between orchestrators.

**Proposal:** When a session involves issues that reference another project (Toolshed consuming PW data), inject a summary of recent changes in the other project that affect the shared interface. This doesn't require real-time orchestrator-to-orchestrator communication — it's a spawn-time context injection.

**Implementation:** Could leverage `kb context` with cross-project filtering. At spawn time or session start, if the project has declared dependencies (like Toolshed depending on PW's serialize_quote), inject recent PW changes to that interface.

**Note:** The cross-project drift detection investigation (2026-02-26) already proposes "interface fingerprints" that could serve as the data source for this.

### 3. Orch Complete All-at-Once Gate Reporting (addresses CLI/Gate Cascade)

**Problem:** orch complete fails on the first unmet gate, requiring retry, which hits the next gate, requiring another retry. 4 failed attempts × 3 agents = user watching a retry loop.

**Proposal:** Report ALL unmet gates in a single failure message. Let the agent or orchestrator fix them all before retrying once.

**Implementation:** Change orch complete verification to collect all failures before returning, rather than short-circuiting on the first.

### 4. Promise Tracking (addresses Follow-Through)

**Problem:** Orchestrators promise actions ("I'll create the cross-repo issue") and forget.

**Proposal:** Lightweight session-level tracking of stated commitments. When the orchestrator says "I'll do X" or "let me do X next," log it. At session end or user prompt, surface any unfulfilled commitments.

**Implementation:** This is hard to implement reliably with current architecture. A simpler version: the orchestrator explicitly uses a todo/checklist when it identifies multiple actions to take, rather than listing them conversationally and hoping to remember.

### 5. Agreement Checks Must Use Consumer-Facing Interfaces (strengthens agreements)

**Problem:** The data divergence between DB queries (shipping populated) and API response (shipping null) means agreement checks against the database give false confidence.

**Proposal:** All cross-project data contract agreements must check against the **consumer's interface** (API endpoint, not DB). The check in the concrete example (curl comparison.json and verify fields) is the right pattern. Document this as a hard constraint on agreement check design.

---

## Findings

### Finding 1: Stale Context Was the Root Failure

The entire debugging marathon across 3 sessions traces back to one root cause: the Toolshed PW integration model was stale about OshCut's data collection method. This caused:
- Wrong diagnosis (Sessions 1 & 2)
- Wrong repo investigation (Session 2)
- Wrong "Not collected" UI behavior (Session 2)
- 30+ minutes of user time spent relaying correct information between orchestrators

Agreements directly address this with `stale-shadow` failure mode detection. If a cross-project agreement had checked "OshCut shipping is available via HTTP API" and the Toolshed model contradicted this, the stale context would have been flagged before the first agent spawned.

### Finding 2: Frame Guard Creates a Debugging Paradox

The frame guard is architecturally sound for routine orchestration — it prevents frame collapse. But during active debugging (user reports live bug, needs answer now), it creates a paradox: the entity responsible for coordinating the fix cannot access the information needed to understand the problem. This forced the orchestrator to spawn investigation agents for simple data-path tracing, adding 2-5 minutes per agent cycle while the user waited.

This is NOT a failure of the frame guard design — it's a missing mode. The guard needs a "diagnostic" mode for active debugging, separate from the "routine" mode for day-to-day orchestration.

### Finding 3: User as Message Bus Is the Hardest Gap

Of the 7 failure categories, "User as Message Bus" is the most structurally difficult to solve. It requires cross-session communication that doesn't exist in the current architecture. Dylan manually relayed 5+ messages between orchestrators in Session 2 alone. This is exhausting, error-prone (the self-contradiction in the clipboard message proves it), and fundamentally at odds with the orchestration promise of "coordinate so the human doesn't have to."

Agreements partially help (catching stale context before it causes disagreement), but can't replace a real cross-repo coordination mechanism.

### Finding 4: CLI Gate Cascade Is a UX Bug

The sequential gate failure pattern in orch complete (miss gate 1 → fix → miss gate 2 → fix → miss gate 3...) appeared in both Sessions 2 and 3. This is a straightforward UX fix (report all gates at once) that would meaningfully reduce user frustration. It's not a deep architectural problem.

### Finding 5: Agreement Checks Need Consumer-Side Verification

The most dangerous moment in Session 2 was when the PW investigation said "all quotes have shipping" (DB evidence) while the production API served null. If an agreement had checked the DB, it would have passed. Only checking the actual API endpoint (what the consumer sees) catches this class of failure. This should be a hard constraint on agreement check design.

---

## Conclusion

The communication breakdown across 3 sessions produced 21 individual failures across 7 categories. The `kb agreements` system would catch **1 of 7 categories well** (stale context / stale-shadow detection) and **partially help** with 1 more (premature action on wrong premises). The remaining **5 categories** (frame guard rigidity, user as message bus, CLI cascade, role confusion, follow-through gaps) represent failure modes that agreements cannot address because they are in-session behavioral issues, architectural constraints, or UX bugs rather than knowledge contract violations.

The good news: stale context was the ROOT cause that triggered the entire cascade. If agreements had caught the stale OshCut model claim, most of the downstream failures (wrong diagnosis, wrong repo investigation, user as message bus) would not have occurred. Agreements may address only 1-2 of 7 categories, but that 1 category was the domino that started the cascade.

The gaps that remain (frame guard diagnostic mode, cross-repo context bridge, gate cascade UX, promise tracking) are worth addressing independently, as they represent recurring friction that will compound across future sessions regardless of agreement coverage.

---

## D.E.K.N. Summary (filled)

- **Delta:** Identified 7 failure categories (stale context, frame guard rigidity, user as message bus, CLI cascade, premature action, role confusion, follow-through gaps) from 21 individual failures across 3 sessions. The agreement system addresses the root cause (stale context) but not 5 of 7 downstream categories.
- **Evidence:** Full transcripts of 3 sessions; concrete failure instances with line numbers; agreement system code review confirming 5 built-in + 5 custom agreements with non-blocking spawn gate architecture.
- **Knowledge:** Agreement checks must run against consumer-facing interfaces (API), not source of truth (DB). Frame guard needs a diagnostic mode. Orch complete should report all gates at once. Cross-repo coordination is the hardest structural gap.
- **Next:** (1) Create issues for diagnostic mode, gate cascade UX, and consumer-side agreement check constraint. (2) The stale context root cause validates the agreement system's core design — prioritize shipping cross-project agreements (stale-shadow) for Toolshed↔PW.
