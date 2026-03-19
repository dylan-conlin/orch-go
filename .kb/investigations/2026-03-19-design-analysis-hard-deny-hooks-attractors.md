# Design Analysis: Should Hard Deny Hooks Include Attractors?

**TLDR:** Hard deny hooks work perfectly as enforcement (zero bypasses) but create architectural displacement — agents put code in the wrong place because the hook says "don't write HERE" without saying "write THERE instead." Three attractor patterns analyzed: DENY+REDIRECT (best), DENY+SUGGEST (fragile), DENY+REPORT+QUEUE (indirect). Recommendation: add path-specific redirect hints to the denial message AND inject governance context into SPAWN_CONTEXT.md (pre-spawn prevention > runtime redirection).

**Status:** Complete

## D.E.K.N. Summary

- **Delta:** Identified a measurement blind spot and a design gap in governance hooks. The system tracks 63 hook denials but zero outcomes — what agents did INSTEAD. The scs-sp-8dm case proves displacement happens (concurrency gate logic landed in `pkg/orch/spawn_preflight.go` instead of `pkg/spawn/gates/`). Three attractor patterns analyzed with tradeoffs.
- **Evidence:** Hook denial message (gate-governance-file-protection.py:105-118), scs-sp-8dm commit (61e12c298), spawn preflight governance check (pkg/orch/governance.go), spawn gate canonical patterns (pkg/spawn/gates/*.go), gate passability decision (2026-01-04), agent-trust-enforcement model policy/enforcement distinction.
- **Knowledge:** (1) Gates without attractors create a hidden failure mode: correct enforcement + wrong architecture. (2) The governance warning at spawn time prints to stderr but is never injected into the agent's context — agents see the hook denial mid-work, not the governance warning. (3) Prevention (SPAWN_CONTEXT injection) is strictly better than correction (hook denial redirect) because it avoids wasted partial work. (4) Hard deny hooks are human checkpoints per the gate passability decision — but that's intentional. The attractor converts them from "blocked, figure it out" to "blocked, here's the right path."
- **Next:** Architect session to implement DENY+REDIRECT (modify hook + add SPAWN_CONTEXT governance injection). Implementation is straightforward but touches governance-protected files.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| `.kb/investigations/simple/2026-03-19-empirical-audit-displaced-code-governance-hooks.md` | Extends (this is the design analysis; that is the empirical audit) | pending | - |
| `.kb/threads/2026-03-19-architectural-displacement-hard-deny-hooks.md` | Deepens (thread raised the question; this investigates it) | yes | - |
| `.kb/models/architectural-enforcement/model.md` | Extends (Invariant 2: gates must be passable; this examines the tension with intentionally-impassable hooks) | yes | See Finding 4 |
| `.kb/models/agent-trust-enforcement/model.md` | Extends (policy vs enforcement framing; displacement is a policy gap, not an enforcement gap) | yes | - |
| `.kb/global/decisions/2026-01-04-gate-refinement-passable-by-gated.md` | Deepens (hard deny hooks ARE human checkpoints — but the decision doesn't address displacement) | yes | - |

## Question

When governance hooks block an agent from writing to a protected file, should the denial message include an "attractor" — guidance on where to put the code instead? What design patterns exist for this?

## Findings

### Finding 1: The Current Denial Message Has No Attractor

The governance hook (`~/.orch/hooks/gate-governance-file-protection.py:105-118`) outputs:

```
GOVERNANCE FILE PROTECTION: Workers cannot modify governance infrastructure.

Blocked file: {file_path}
Matched pattern: {matched_pattern}

Protected governance files include:
  - ~/.orch/hooks/*.py (deny hooks)
  - scripts/pre-commit* (pre-commit gates)
  - *_lint_test.go (structural tests)
  - pkg/spawn/gates/* (spawn gates)
  - pkg/verify/precommit.go, accretion.go (verification gates)

These files form the control plane and can only be modified by
the orchestrator (Dylan's direct session). If you need changes
to governance files, report via:
  bd comments add <id> "DISCOVERED: governance file <path> needs update - <reason>"
```

The message tells the agent:
1. WHAT is blocked (the specific file)
2. WHY it's blocked (governance infrastructure)
3. WHERE to report (bd comments)

But NOT:
4. WHERE to put the code instead

The agent is blocked with nowhere to go. It can either:
- Give up on that part of the task
- Put the code somewhere else (displacement)
- Report it and continue with other work

### Finding 2: Displacement Evidence — scs-sp-8dm Concurrency Gate

The scs-sp-8dm agent needed to add a concurrency gate. The canonical location is `pkg/spawn/gates/concurrency.go` — this matches the existing pattern (triage.go, hotspot.go, agreements.go, question.go all live in gates/).

The governance hook blocked `pkg/spawn/gates/` writes. The agent:
1. Reported via beads comment (correctly following the hook's instruction)
2. Added the concurrency check to `pkg/orch/spawn_preflight.go:30-41` instead

This is architecturally wrong. The concurrency check is a spawn gate — it belongs in `pkg/spawn/gates/`. But it works, the fix shipped (commit 61e12c298), and the oscillation bug was eliminated.

**The displacement was invisible.** The thread notes "zero tracking of what agents did INSTEAD" — we count 63 hook denials but don't know where the code landed.

### Finding 3: Spawn Gates vs Hard Deny Hooks — Information Asymmetry

Comparing what each enforcement mechanism tells the agent:

| Mechanism | Blocks? | Tells agent WHERE to go? | Agent can satisfy? |
|---|---|---|---|
| Triage gate (CheckTriageBypass) | Yes | Yes — "add --bypass-triage" + shows daemon workflow | Yes (add flag) |
| Hotspot gate (CheckHotspot) | No (advisory) | Yes — "daemon will schedule extraction" + architect recommendation | N/A (advisory) |
| Agreements gate (CheckAgreements) | No (advisory) | Yes — "Run 'kb agreements check' for details" | N/A (advisory) |
| Open questions gate (CheckOpenQuestions) | No (advisory) | Yes — "Work may need revision when questions are answered" | N/A (advisory) |
| Governance preflight (CheckGovernance) | No (warning) | Partial — "route this work to an orchestrator session" | N/A (warning) |
| **Governance deny hook** | **Yes** | **No** — just "report via bd comments" | **No (human checkpoint)** |

**Key observation:** Every spawn gate provides guidance on the right path forward. The governance deny hook is the ONLY enforcement mechanism that blocks without redirecting.

### Finding 4: The Gate Passability Tension

The Jan 4 gate passability decision established:
- Valid gate: agent can pass by doing work
- Human checkpoint: requires human action — "disguised as automation"
- Scheduling constraint: depends on external state

Hard deny hooks are intentionally human checkpoints. The decision says this is a smell ("when you need escape hatches for escape hatches, the gate is wrong"), but governance protection is different — we WANT this to be impassable by workers. The immutability IS the feature.

The tension: Invariant 2 says "gates must be passable by the gated party." Governance hooks violate this intentionally. The attractor doesn't make the gate passable — it makes the OUTCOME better by guiding the agent to an alternative.

This reframes the question: the issue isn't gate passability but **outcome guidance**. A human checkpoint can still guide the agent toward productive work instead of leaving it with nowhere to go.

### Finding 5: The Governance Warning at Spawn Time Is Invisible to Agents

`pkg/orch/governance.go:CheckGovernance()` detects at spawn time when a task references governance-protected paths and prints a warning to stderr:

```
⚠️  GOVERNANCE-PROTECTED FILES DETECTED
   Task references paths protected by governance hooks:
     • pkg/spawn/gates/ (spawn gate infrastructure)

   Workers editing these files will be BLOCKED by hooks at runtime.
   Consider: route this work to an orchestrator session instead.
```

But this warning:
1. Prints to stderr during spawn (before the agent starts)
2. Is NOT injected into SPAWN_CONTEXT.md
3. Is NOT visible to the agent

The agent discovers the governance restriction only when it tries to edit and gets denied. By then, it's already invested in an approach that includes modifying protected files.

**This is the key gap.** Prevention (telling the agent before it starts) is strictly better than correction (telling the agent after it's blocked).

### Finding 6: Three Attractor Patterns Analyzed

#### Pattern A: DENY+REDIRECT (Hook-Level Attractor)

Modify the deny message to include path-specific redirect hints:

```python
REDIRECTS = {
    r'pkg/spawn/gates/': (
        "If you need to add a new spawn gate, create it in pkg/orch/ "
        "and document the need for migration to pkg/spawn/gates/ in your "
        "beads comment. The orchestrator will move it to the protected "
        "location during review."
    ),
    r'pkg/verify/': (
        "If you need to add verification logic, add it to your workspace "
        "or pkg/orch/ and note the intended location in SYNTHESIS.md."
    ),
    r'\.orch/hooks/': (
        "Hooks can only be created by the orchestrator. Document the "
        "desired hook behavior in your investigation file and recommend "
        "it as a follow-up action."
    ),
}
```

**Tradeoffs:**
- (+) Immediate guidance at the moment of denial
- (+) Path-specific — different protected areas get different redirects
- (+) No infrastructure change needed — just a message update
- (-) Redirect destinations may be architecturally wrong (pkg/orch/ is not always the right place)
- (-) Maintenance: redirect hints must be updated when architecture changes
- (-) Still reactive — agent has already planned to write there

#### Pattern B: DENY+SUGGEST (Context-Aware Attractor)

The hook examines the blocked file path and suggests a specific alternative based on naming patterns:

```python
# If blocked from pkg/spawn/gates/concurrency.go,
# suggest: "Create pkg/orch/concurrency_gate.go instead"
suggested_path = file_path.replace("pkg/spawn/gates/", "pkg/orch/")
```

**Tradeoffs:**
- (+) More specific guidance — suggests actual file paths
- (-) Fragile — path transformation rules are simplistic
- (-) Can suggest wrong locations (not all gate logic belongs in pkg/orch/)
- (-) High maintenance burden — every new protected pattern needs a mapping
- (-) Still reactive

#### Pattern C: DENY+REPORT+QUEUE (Automatic Issue Creation)

The hook denial automatically creates a beads issue for orchestrator follow-up:

```python
# On denial, create a migration issue
os.system(f'bd create "Migrate {file_path} changes to governance location" '
          f'--type task -l triage:ready')
```

**Tradeoffs:**
- (+) Ensures follow-up happens — issue is tracked
- (+) Agent doesn't need to figure out where code goes — just put it somewhere reasonable
- (-) Creates issue noise (many denials are for files agents shouldn't be touching at all)
- (-) Side-effect in a hook (hooks should be pure deny/allow decisions)
- (-) Doesn't solve the displacement problem — agent still puts code in wrong place
- (-) The orchestrator already sees CONSTRAINT reports in beads comments

### Finding 7: The Real Solution Is Pre-Spawn Prevention

The most impactful change is NOT improving the deny message. It's preventing the agent from planning to modify governance files in the first place.

**The fix is two-part:**

1. **Inject governance context into SPAWN_CONTEXT.md** — The governance preflight check (`CheckGovernance`) already detects when tasks reference protected paths. If this result were injected into SPAWN_CONTEXT.md, the agent would know BEFORE IT STARTS PLANNING that certain paths are off-limits. This is analogous to how hotspot results are injected into spawn context.

2. **Add redirect hints to the deny message** (Pattern A, simplified) — As a fallback for agents that discover governance files organically (not from the task description), add a brief redirect hint. Not path-specific transformations (too fragile), but the general pattern: "Put the code in a non-protected location and document the intended destination in SYNTHESIS.md."

**Why prevention > correction:**
- Prevention avoids wasted partial work (agent doesn't plan around protected files)
- Prevention is proactive (agent plans around the constraint)
- Correction is reactive (agent has already invested in an approach)
- The architectural-enforcement model confirms: "Prevention > Detection > Rejection. Each layer further from the source has higher cost."

### Finding 8: Should There Be a Governance-Impl Skill?

The task asked: "Should there be a 'governance-impl' skill that CAN write to these files under controlled conditions?"

**Analysis:** This would mean setting `CLAUDE_CONTEXT` to something other than "worker" or adding the skill to an exemption list. This fundamentally undermines the governance boundary:

- The whole point of governance file protection is that workers cannot modify their own constraints
- A governance-impl skill IS a worker that modifies constraints
- The trust hierarchy (agent-trust-enforcement model) says enforcement must be structural, not behavioral — trusting a skill to "be careful" with governance files is convention-layer trust (L4)
- The escape hatch already exists: orchestrator sessions (Dylan's direct session) CAN modify these files

**Verdict: No.** The governance boundary should remain absolute for workers. If governance files need changes, the sequence is: worker reports → orchestrator reviews → orchestrator implements in direct session. This is the human checkpoint by design.

## Test Performed

1. **Read the current governance hook** (`~/.orch/hooks/gate-governance-file-protection.py`) — confirmed no attractor in denial message
2. **Read all 4 spawn gate implementations** (`pkg/spawn/gates/{triage,hotspot,agreements,question}.go`) — confirmed all provide guidance on the right path forward, unlike the governance hook
3. **Read the governance preflight check** (`pkg/orch/governance.go`) — confirmed it detects governance-protected paths at spawn time but does NOT inject into SPAWN_CONTEXT
4. **Read the gate passability decision** (2026-01-04) — confirmed hard deny hooks are intentional human checkpoints, not a bug
5. **Read the agent-trust-enforcement model** — confirmed policy (WHAT) vs enforcement (HOW) distinction; displacement is a policy gap
6. **Examined scs-sp-8dm commit** (61e12c298) — confirmed concurrency gate logic displaced from `pkg/spawn/gates/` to `pkg/orch/spawn_preflight.go`
7. **Cross-referenced hook audit probe** (2026-03-12) — confirmed 63 hook denials, zero outcome tracking

## Conclusion

**The current design has a gap: hard deny hooks block the wrong action but don't guide toward the right action.** This creates architectural displacement — code ends up in the wrong location because agents have nowhere else to go.

**Three attractor patterns compared:**

| Pattern | Prevents Displacement? | Maintenance | Risk | Recommendation |
|---|---|---|---|---|
| A: DENY+REDIRECT | Partially (reactive) | Low | Low (message only) | **Yes — implement as fallback** |
| B: DENY+SUGGEST | Partially (reactive, specific) | High (path mappings) | Medium (wrong suggestions) | No — too fragile |
| C: DENY+REPORT+QUEUE | No (tracks issue, not outcome) | Low | Medium (issue noise) | No — already covered by CONSTRAINT comments |

**The highest-impact change is NOT an attractor pattern.** It's pre-spawn prevention:

1. **Inject governance context into SPAWN_CONTEXT.md** so agents know about protected paths before they start planning. This is the same pattern as hotspot injection (which already works) and is strictly better than runtime denial.

2. **Add a brief redirect hint to the deny message** as a fallback: "Put the code in a non-protected location and document the intended destination in SYNTHESIS.md. The orchestrator will migrate it during review."

3. **Do NOT create a governance-impl skill.** The human checkpoint is intentional. Workers should not modify their own constraints.

4. **Track displacement outcomes.** The thread identified a measurement blind spot: 63 denials, zero outcome tracking. Add a `governance.displacement` event that captures what the agent did instead — this enables measuring whether attractors actually reduce architectural displacement.

**Does an attractor weaken the governance boundary?** No. The boundary is the hook enforcement (agent physically cannot write to the file). The attractor is guidance about what to do instead — it operates in the policy layer, not the enforcement layer. It's analogous to a "detour" sign on a road closure: the road is still closed, but drivers know where to go.
