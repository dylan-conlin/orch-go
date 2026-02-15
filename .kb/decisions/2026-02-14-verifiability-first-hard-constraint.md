# Decision: Verifiability-First as Hard Operating Constraint

**Date:** 2026-02-14
**Status:** Proposed
**Context:** The entropy spiral history (1,625 commits lost across three rollbacks) proves that orch-go has never operated in verified mode. From day 1 (Dec 19), velocity was 40-130 commits/day. This decision establishes the founding operating constraint: the system cannot proceed to the next deliverable until Dylan has verified the current one through a two-gate process.

## The Problem

**Historical evidence:** Three entropy spirals with escalating damage:
- First (Dec 21): 115 commits lost
- Second (Dec 27 - Jan 2): 347 commits lost  
- Third (Jan 18 - Feb 12): 1,163 commits lost, 5.4M LOC churn, 26 days of autonomous operation with zero human commits

**Root cause across all three:** Velocity exceeded verification bandwidth. Agents produced locally-correct work that composed into globally-incoherent systems. Each "fix" altered ground truth for the next agent. Investigations replaced actual testing. The system accelerated while reporting success.

**The devastating finding:** Every sampled "fix:" commit was a real fix. The code did what it said. But the system spiraled into a state where nothing worked. Local correctness ≠ global correctness when composition happens faster than verification.

**Key trajectory data:**
- Fix:feat ratio in third spiral: 0.96:1 (each feature produced nearly one bug)
- Unverified velocity has negative value after accounting for recovery effort
- Post-mortems after spirals 1 and 2 identified correct mitigations — none were implemented
- Documentation without enforcement = data-plane artifact agents can ignore

## The Decision

### Constraint 1: Two-Gate Verification (Mandatory)

The system **CANNOT** proceed to the next deliverable until Dylan has completed both gates:

**Gate 1: Comprehension (Explain-Back)**
- Dylan must explain what the deliverable does in his own words
- "Looks good" or "approved" is insufficient
- Must demonstrate understanding of: behavior, architecture, trade-offs
- Failure mode: signing off on work you haven't comprehended

**Gate 2: Behavioral Verification** 
- Dylan must observe the behavior, not just read about it
- Tests pass, commands run, UI changes visible, metrics move
- Artifact-based evidence (screenshots, logs, diffs) required
- Failure mode: trusting self-reported success

**Both gates required.** Comprehension without verification = theoretical approval. Verification without comprehension = blind testing.

### Constraint 2: No Autonomous Progression

The following transitions are **BLOCKED** until Dylan completes both gates:

- Agent A completes work → Agent B starts dependent work ❌
- Investigation complete → Decision made ❌  
- Design approved → Implementation started ❌
- Implementation complete → Deployment/merge ❌
- Feature works → Next feature begins ❌

**Exception:** Independent parallel work is allowed (Agent A on feature X, Agent B on unrelated feature Y) as long as each individual deliverable gets two-gate verification before its next step.

### Constraint 3: Mechanical Enforcement

This is **NOT** a recommendation in a skill document. This is an architectural constraint enforced through:

**Control Plane Mechanisms (Immutable):**
1. **Verification checkpoint file** (`~/.orch/verification-checkpoints.jsonl`)
   - Each deliverable gets entry: `{beads_id, deliverable, gate1_complete, gate2_complete, timestamp}`
   - `orch complete` reads this before closing issues
   - Missing checkpoint = cannot close

2. **Heartbeat file with verification timestamp** (`~/.orch/control-heartbeat`)
   - Written by Dylan (not agents) when verification happens
   - Daemon reads this before spawning next agent
   - Stale heartbeat (>24h) = halt autonomous spawning

3. **Session continuity gate**
   - Before spawning Agent N, check: has Agent N-1's work been verified?
   - Read verification checkpoint file for completion
   - Block spawn if unverified work exists

**Data Plane Mechanisms (Agent-visible):**
- `orch complete` requires `--explain "..."` flag with non-empty explanation
- SPAWN_CONTEXT includes verification requirements
- Phase: Complete requires Dylan's explicit confirmation

**Why both layers:** Control plane ensures agents cannot bypass verification even if they modify orch-go code. Data plane provides ergonomic interface and clear expectations.

### Constraint 4: Violation Handling

**If constraint violated:**
1. **Halt autonomous operation** - daemon stops spawning
2. **Surface violation** - `orch control status` shows what's blocked
3. **Require explicit override** - `orch control resume --verified "explanation"`
4. **Log violation** - `~/.orch/metrics/verification-violations.log`

**Override path exists** - for emergencies or false positives. But override requires explicit action and explanation, not passive continuation.

## Rationale

### Why This Has Never Existed

Analysis of orch-go commit history reveals:
- Day 1 (Dec 19): 40 commits (velocity already above verification bandwidth)
- Normal operation: 3-4 agents producing 20-60 commits/day
- "Recovery periods" after rollbacks immediately resumed spiral velocity
- The three spirals weren't discrete events — they were one continuous acceleration with two rollbacks

**This decision doesn't prevent regression. It establishes something that has never been true.**

### Why Mechanical Enforcement

From entropy spiral analysis: "Documentation doesn't prevent recurrence; immutable infrastructure does."

Three post-mortems. Fifteen specific mitigations identified. All documented. None implemented.

The system learned intellectually (investigations, principles) but not structurally (circuit breakers, gates). Mitigations that live as documentation are data-plane artifacts — agents can read them, ignore them, or modify them.

**Hard constraint enforced mechanically** = control-plane artifact. Agents operate within it whether or not they've read the post-mortem.

### Why Explain-Back Specifically

From the AI Deference Pattern (global CLAUDE.md): Following AI guidance without checking whether you have relevant experience the AI doesn't know about.

During third spiral: commit messages said "fix:", synthesis files said "success", daemon continued spawning. All signals were self-reported by the system. Dylan trusted them without independent verification.

**Explain-back forces comprehension.** If you can't explain what it does, you haven't verified you understand it. Understanding lag (mistaking new visibility for new problems) wouldn't survive explain-back.

### Why Two Gates

Gate 1 alone (comprehension): Can explain a design that doesn't actually work.

Gate 2 alone (behavioral): Can verify tests pass without understanding what they test.

**Both required:** Comprehension ensures you know what should happen. Behavioral verification ensures it actually happens.

## Implementation Phases

### Phase 1: Checkpoint Infrastructure (Control Plane)
- [ ] Create `~/.orch/verification-checkpoints.jsonl` file
- [ ] Add verification checkpoint tracking to `pkg/control/`
- [ ] Wire `orch complete` to read checkpoints before closing
- [ ] Add `--explain` flag validation to `orch complete`
- [ ] Add checkpoint creation command: `orch verify complete <beads-id> --explain "..."`

### Phase 2: Heartbeat Integration
- [ ] Extend `~/.orch/control-heartbeat` with verification timestamp
- [ ] Wire daemon spawn logic to check heartbeat age
- [ ] Add `orch verify heartbeat` command to update timestamp
- [ ] Halt spawning if heartbeat >24h old

### Phase 3: Session Continuity Gate  
- [ ] Add dependency tracking to spawn context
- [ ] Before spawning Agent N, check if Agent N-1 verified
- [ ] Block spawn if unverified work exists
- [ ] Add override: `orch spawn --bypass-verification "reason"`

### Phase 4: Violation Handling
- [ ] Create `~/.orch/metrics/verification-violations.log`
- [ ] Wire `orch control status` to show verification state
- [ ] Add `orch control resume --verified "explanation"` command
- [ ] Log all violations with timestamp, beads_id, reason

### Phase 5: Ergonomic Integration
- [ ] Add verification checklist to `orch complete` output
- [ ] Update SPAWN_CONTEXT template with verification requirements
- [ ] Add verification status to dashboard
- [ ] Document verification workflow in skills

## Consequences

### Positive

**Structural impossibility of entropy spiral:**
- Unverified velocity cannot accumulate
- 24h heartbeat requirement means max 1 day of autonomous drift
- Verification checkpoint blocks issue closure until comprehension + behavior confirmed
- Control plane enforcement means agents cannot erode defenses

**Human bandwidth acknowledgment:**
- System explicitly paces itself to verification capacity
- No pretense that documentation replaces enforcement
- Verification bottleneck becomes feature, not bug

**Forcing function for understanding:**
- Explain-back requirement prevents passive approval
- Can't sign off on work you don't comprehend
- Understanding lag gets caught (if you can't explain it, you haven't understood it)

### Negative

**Velocity reduction:**
- Daemon cannot spawn continuously without human presence
- Batch work requires batch verification
- High-value urgent work may hit verification backlog

**Human becomes bottleneck:**
- Dylan's verification bandwidth is the rate-limiting step
- Cannot delegate verification to agents (defeats purpose)
- Vacations/illness halt autonomous operation

**Process overhead:**
- Every deliverable needs two-gate verification
- `orch verify complete` command adds step to workflow
- False positives possible (heartbeat expired during valid work)

### Risks

**Dylan changes without adapting:**
- If "this means I need to change, then so be it, I'll change" doesn't materialize
- Verification queue grows, becomes bottleneck, gets bypassed
- Override becomes norm, constraint erodes

**Mitigation:** Verification violations logged. Monthly review of override frequency. If >50% of completions use override, constraint is broken — escalate to architecture change.

**Implementation half-done:**
- Control plane mechanisms (checkpoints, heartbeat) built but not wired
- Data plane mechanisms (--explain flag) exist but bypass via --force still works
- Gates present but bypassable = security theater

**Mitigation:** All five implementation phases required before declaring decision "Accepted". Partial implementation = Proposed status until complete.

**Agent modification of control plane:**
- Control plane files (`~/.orch/verification-checkpoints.jsonl`) are outside repo
- But agents could theoretically modify `pkg/control/` code that reads them
- If agents modify checkpoint-reading code, enforcement erodes

**Mitigation:** Protected path monitoring for `pkg/control/` (from immutable control plane decision). Changes to control plane code require human review.

## Evidence

**Entropy Spiral Analysis:**
- `.kb/investigations/2026-02-14-inv-entropy-spiral-deep-analysis.md` — comprehensive post-mortem proving unverified velocity has negative value
- Commit statistics: 1,625 commits lost, fix:feat ratio 0.96:1, 26 days with zero human commits
- Root cause: locally-correct changes composing incorrectly when velocity exceeds verification

**Verifiability-First Model:**
- `~/orch-knowledge/kb/models/verifiability-first-development.md` — mental model for development when human cannot directly verify code through comprehension
- Core insight: human role shifts from "write code" to "specify behavior and verify outcomes"
- Verification bottleneck principle: system cannot change faster than human can verify behavior

**Historical Pattern:**
- Three spirals with identical root cause
- Mitigations documented after spirals 1 and 2, none implemented before spiral 3
- "Documentation without enforcement" proven insufficient

**Design Session Notes (Feb 14):**
- orch-go has NEVER operated in non-spiral mode
- From day 1 (Dec 19), velocity was 40-130 commits/day
- The three spirals weren't discrete events — one continuous acceleration
- This decision establishes founding constraint, not regression fix

## Open Questions

1. **What counts as "deliverable" for verification purposes?**
   - Investigation file? Feature implementation? Bug fix? All of the above?
   - Granularity matters: too fine = verification bottleneck, too coarse = batched risk

2. **How to handle multi-agent parallel work?**
   - Agent A on feature X, Agent B on feature Y — both need verification before merge
   - Does verification queue become linear bottleneck?
   - Can features be verified independently if truly isolated?

3. **What's the override threshold before constraint is failing?**
   - If 10% of completions use `--bypass-verification`, is that acceptable friction?
   - If 50%? At what point is the constraint broken vs working-as-designed?

4. **How to verify the verification system?**
   - Who verifies that checkpoints are being written correctly?
   - How to detect if control plane code gets modified to bypass gates?
   - Infinite regress problem: at some point human must trust something

## Related Decisions

- **Immutable Control Plane** (if/when created) — this decision depends on control plane files being agent-unreachable
- **Verification Bottleneck Principle** (`.kb/principles.md`) — foundational constraint this decision operationalizes
- **Strategic-First Orchestration** (`.kb/decisions/2026-01-11-strategic-first-orchestration.md`) — Dylan as strategic comprehender, verification is comprehension activity

## Next Steps

1. **Accept or reject this decision** - Dylan must confirm willingness to change workflow
2. **Phase 1 implementation** - checkpoint infrastructure and `--explain` flag
3. **Calibration period** - run for 2 weeks, measure override frequency
4. **Iterate on deliverable granularity** - adjust what requires verification based on friction
5. **Monitor violation patterns** - if same constraint repeatedly overridden, constraint is wrong
