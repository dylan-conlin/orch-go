<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Disabled gates share a common failure pattern: friction-without-value (false positives blocking legitimate work) caused by unreliable detection mechanisms or over-broad applicability.

**Evidence:** Identified 9 disabled features: Repro Gate, Dependency Gate, Coaching Plugin + 6 role-dependent plugins. All share the pattern of creating friction that agents couldn't overcome without manual intervention.

**Knowledge:** Successful re-enablement requires three criteria: (1) upstream architectural fix, (2) single simple detection mechanism, (3) human-verified end-to-end test before declaring "fixed." The 55% --force bypass rate on completion gates shows the system is still fragile.

**Next:** Document re-enablement criteria clearly. Consider periodic audit to check if blockers have been resolved. Do not attempt re-enablement without all three criteria met.

**Authority:** architectural - Re-enablement affects cross-agent behavior and requires orchestrator judgment on timing.

---

# Investigation: Investigate Disabled Gate Failure Patterns

**Question:** What patterns led to gates being disabled, and what criteria would justify re-enabling them?

**Started:** 2026-01-31
**Updated:** 2026-01-31
**Owner:** og-inv-investigate-disabled-gate-31jan-23c1
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Nine Disabled Features Share Common Patterns

**Evidence:** Inventory of disabled features:

| Feature | Disabled Date | Reason | Location |
|---------|---------------|--------|----------|
| Repro Verification Gate | Jan 4, 2026 | "Too much friction - agents couldn't complete without manual intervention" | `cmd/orch/complete_cmd.go:781-797` (commented out) |
| Dependency Check Gate | Jan 4, 2026 | "Was blocking completions" | Mentioned in `.kb/guides/completion-gates.md:429` |
| Coaching Plugin | Jan 28, 2026 | 18+ investigations, never worked reliably | `plugins/coaching.ts.disabled` |
| orch-hud.ts | Jan 30, 2026 | Same broken detection pattern | `plugins/orch-hud.ts.disabled` |
| orchestrator-tool-gate.ts | Jan 30, 2026 | Unreliable role detection | `plugins/orchestrator-tool-gate.ts.disabled` |
| orchestrator-session.ts | Jan 30, 2026 | Wrong session injection | `plugins/orchestrator-session.ts.disabled` |
| task-tool-gate.ts | Jan 30, 2026 | Unreliable detection | `plugins/task-tool-gate.ts.disabled` |
| session-context.ts | Jan 30, 2026 | Server can't see agent env vars | `.opencode/plugin/session-context.ts.disabled` |

**Source:**
- `.kb/guides/completion-gates.md:295-310`
- `.kb/decisions/2026-01-28-coaching-plugin-disabled.md`
- `.kb/decisions/2026-01-30-role-dependent-plugins-disabled.md`
- `glob **/*.disabled` output

**Significance:** These aren't random failures - they represent systematic architectural gaps that prevent reliable detection mechanisms.

---

### Finding 2: The Verification Bottleneck Pattern

**Evidence:** The coaching plugin saga demonstrates the pattern clearly:
- 18+ investigations over 3 weeks
- 13+ fix commits to coaching.ts
- 4 different detection approaches (metadata.role, title patterns, tool paths, API lookup)
- On Jan 28 alone, 5 investigations produced contradictory conclusions

Each agent verified their slice looked correct, but none verified actual end-to-end experience. When Dylan tested manually, coaching alerts still fired on workers despite "fixed" claims.

The post-mortem from Dec 27-Jan 2 spiral showed the same pattern: 347 commits in 6 days, all individually correct, but compositionally broken. The system had no way to answer "is this actually working?" that wasn't self-reported.

**Source:**
- `.kb/decisions/2026-01-28-coaching-plugin-disabled.md:23-42`
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md:59-67`

**Significance:** Agents can grep logs, check code paths, and conclude "fixed" without the bug actually being fixed. This creates investigation churn until a human manually verifies end-to-end behavior.

---

### Finding 3: Re-enablement Criteria Are Consistently Three-Part

**Evidence:** Both coaching plugin and role-dependent plugins decisions specify the same re-enablement criteria:

**Coaching Plugin (Jan 28):**
1. OpenCode to expose `session.metadata.role` reliably (upstream fix)
2. A single, simple detection mechanism (not 4 layered heuristics)
3. Human-verified end-to-end test before claiming "fixed"

**Role-Dependent Plugins (Jan 30):**
1. OpenCode must expose `session.metadata.role` from the `x-opencode-env-ORCH_WORKER` header
2. Single detection mechanism - no more layered heuristics
3. Human-verified end-to-end test - agent must confirm they SEE the expected behavior

The repro verification gate's implicit criteria: "a mechanism that doesn't require manual intervention for every completion."

**Source:**
- `.kb/decisions/2026-01-28-coaching-plugin-disabled.md:56-60`
- `.kb/decisions/2026-01-30-role-dependent-plugins-disabled.md:56-63`
- `.kb/guides/completion-gates.md:302-308`

**Significance:** These criteria form a pattern: (1) fix the root cause architecturally, (2) simplify the mechanism, (3) verify end-to-end before declaring success.

---

### Finding 4: Completion Gate Bypass Rate Indicates Systemic Issues

**Evidence:** The Jan 17 synthesis of completion investigations found:
- 55% of completions used `--force` bypass
- 12 distinct gates with independent edge cases create multiplicative failure modes
- Agent-scoping ambiguity is the primary churn source (spawn time ≠ agent identity)

The architectural recommendation was Agent Manifest + Git-based scoping + Evidence Collection Phase, but this hasn't been implemented yet.

**Source:** `.kb/investigations/2026-01-17-inv-design-synthesize-26-completion-investigations.md:99-103`

**Significance:** Even the active gates are fragile. The high bypass rate indicates that gates are failing legitimate completions, not just blocking bad ones. This same pattern led to disabling the repro gate - friction without value.

---

## Synthesis

**Key Insights:**

1. **Friction-Without-Value is the Disabling Trigger** - Gates/plugins were disabled not because they were wrong in principle, but because they created friction (false positives, manual intervention required) that agents couldn't overcome. The benefit didn't justify the cost.

2. **Detection Mechanism Unreliability is the Common Root Cause** - Role-dependent plugins failed because of architectural gap: plugins run in OpenCode server process but role detection depends on agent process environment. The repro gate failed because it required human verification that couldn't scale.

3. **Investigation Churn Precedes Disabling** - Before a feature gets disabled, there's typically a pattern of repeated "fix" attempts. Coaching plugin had 18+ investigations; completion gates had 26+ investigations. The churn itself is a signal that the feature has fundamental problems.

4. **Re-enablement Requires Architectural Change** - All documented re-enablement criteria require upstream fixes or architectural changes, not just "better implementation." This suggests the features were disabled because the problems were unsolvable within current architecture.

**Answer to Investigation Question:**

**What patterns led to gates being disabled?**

The common pattern is: (1) feature introduced to solve real problem, (2) detection mechanism proves unreliable (false positives, can't distinguish cases), (3) creates friction that agents can't overcome, (4) investigation churn as repeated "fixes" fail, (5) eventually disabled to stop the bleeding.

The root causes are architectural:
- Plugins can't reliably detect worker vs orchestrator roles
- Time-based scoping fails for concurrent agents
- Claim-based verification is unreliable without end-to-end human validation

**What criteria would justify re-enabling them?**

Three-part criteria pattern from existing decisions:
1. **Upstream architectural fix** - The root cause must be solved (e.g., OpenCode exposing `session.metadata.role`)
2. **Single simple detection mechanism** - No layered heuristics; if it needs 4 detection methods, it won't work
3. **Human-verified end-to-end test** - Agent verification isn't sufficient; a human must observe the actual behavior before claiming "fixed"

Until these criteria are met, re-enablement will likely repeat the investigation churn cycle.

---

## Structured Uncertainty

**What's tested:**

- ✅ 9 disabled features exist (verified: `glob **/*.disabled` + code review of complete_cmd.go)
- ✅ Coaching plugin had 18+ investigations (verified: decision doc `.kb/decisions/2026-01-28-coaching-plugin-disabled.md:29`)
- ✅ 55% completion bypass rate (verified: synthesis investigation `.kb/investigations/2026-01-17-inv-design-synthesize-26-completion-investigations.md:99`)
- ✅ Re-enablement criteria are documented (verified: read both disabled decisions)

**What's untested:**

- ⚠️ Whether OpenCode will actually expose `session.metadata.role` (upstream dependency, no timeline)
- ⚠️ Whether Agent Manifest pattern would reduce gate churn (proposed but not implemented)
- ⚠️ Whether any of the disabled features should be re-enabled now (this is architectural judgment)

**What would change this:**

- Finding would be incomplete if there are additional disabled features I didn't discover
- Finding would be different if bypass rate has improved significantly since Jan 17
- Re-enablement criteria might be insufficient if upstream fixes don't actually solve the detection problem

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Leave features disabled; don't attempt re-enablement | architectural | Affects cross-agent behavior and requires synthesis across multiple systems |
| Create periodic audit issue | implementation | Simple tracking task within scope |
| Document re-enablement criteria in guide | implementation | Documentation update within existing patterns |

### Recommended Approach ⭐

**Do Nothing Now, Create Audit Mechanism** - Leave all features disabled and establish a quarterly check for whether blockers have been resolved.

**Why this approach:**
- The three-part criteria aren't met for any disabled feature (Finding 3)
- Attempting re-enablement without criteria met will repeat investigation churn (Finding 2)
- Human verification is required before re-enablement, not agent investigation

**Trade-offs accepted:**
- Orchestrators lose dynamic HUD, tool gating, coaching
- Agents must rely on CLAUDE.md instructions and self-discipline
- Some value from these features is lost until re-enablement

**Implementation sequence:**
1. Document clear re-enablement criteria in `.kb/guides/completion-gates.md` (extend existing history section)
2. Create recurring beads issue `bd create "Audit disabled feature re-enablement criteria" --type task` with quarterly recurrence
3. Each audit: check if OpenCode has exposed `session.metadata.role`, then evaluate other criteria

### Alternative Approaches Considered

**Option B: Re-enable with workarounds**
- **Pros:** Restores lost functionality
- **Cons:** Will likely repeat investigation churn; workarounds add complexity
- **When to use instead:** If upstream fix is blocked indefinitely and workaround is simple enough

**Option C: Remove disabled features entirely**
- **Pros:** Reduces codebase complexity, no maintenance burden
- **Cons:** Loses option value; harder to re-enable later
- **When to use instead:** If features have been disabled >1 year with no path to re-enablement

**Rationale for recommendation:** Option A preserves option value while avoiding investigation churn. Re-enablement should be driven by upstream changes, not effort.

---

### Implementation Details

**What to implement first:**
- Update `.kb/guides/completion-gates.md` history section with re-enablement criteria
- No code changes recommended

**Things to watch out for:**
- ⚠️ Don't conflate "disabled" with "should be deleted" - these features may be re-enabled when criteria met
- ⚠️ Agent investigation cannot substitute for human end-to-end verification
- ⚠️ The 55% bypass rate on active gates suggests more features may need disabling, not re-enabling

**Areas needing further investigation:**
- What's the timeline for OpenCode to expose `session.metadata.role`?
- Are there any disabled features whose criteria have already been met?
- Should the Agent Manifest proposal from Jan 17 synthesis be implemented to fix active gate fragility?

**Success criteria:**
- ✅ Re-enablement criteria are documented and discoverable
- ✅ No investigation churn on disabled features (churn = working around disablement)
- ✅ Quarterly audit surfaces any criteria that have been met

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go:781-797` - Disabled repro verification gate code (commented out)
- `.kb/guides/completion-gates.md` - Gate reference documentation including disabled gates
- `.kb/decisions/2026-01-28-coaching-plugin-disabled.md` - Coaching plugin disabling decision
- `.kb/decisions/2026-01-30-role-dependent-plugins-disabled.md` - Role-dependent plugins disabling decision
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` - Verification bottleneck pattern
- `.kb/investigations/2026-01-17-inv-design-synthesize-26-completion-investigations.md` - Gate churn analysis
- `pkg/verify/repro.go` - Repro verification logic

**Commands Run:**
```bash
# Find all disabled plugin files
glob **/*.disabled

# Search for disabled patterns in codebase
rg "DISABLED|disabled:|\.ts\.disabled" --files-with-matches

# Check git status
git status
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-28-coaching-plugin-disabled.md` - Primary coaching failure analysis
- **Decision:** `.kb/decisions/2026-01-30-role-dependent-plugins-disabled.md` - Plugin architecture audit
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-synthesize-26-completion-investigations.md` - Gate churn root cause analysis
- **Post-Mortem:** `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` - Verification bottleneck discovery
- **Guide:** `.kb/guides/completion-gates.md` - Should be updated with re-enablement criteria

---

## Investigation History

**2026-01-31 16:00:** Investigation started
- Initial question: What patterns led to gates being disabled, and what criteria would justify re-enabling them?
- Context: Orchestrator spawn to understand disabled gate failure patterns

**2026-01-31 16:15:** Initial context gathering
- Read completion-gates.md guide - found repro and dependency gates disabled
- Read coaching plugin decision - found 18+ investigation churn pattern
- Read role-dependent plugins decision - found 6 more disabled plugins

**2026-01-31 16:30:** Pattern analysis complete
- Identified 9 total disabled features
- Found common failure pattern: friction-without-value from unreliable detection
- Documented three-part re-enablement criteria from existing decisions

**2026-01-31 16:45:** Investigation completed
- Status: Complete
- Key outcome: Disabled gates share friction-without-value pattern; re-enablement requires (1) upstream fix, (2) simple mechanism, (3) human end-to-end verification
