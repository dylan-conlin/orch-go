# Decision: Five-Tier Completion Escalation Model

**Date:** 2026-01-17
**Status:** Accepted
**Context:** Synthesized from 28 investigations on agent completion (Dec 2025 - Jan 2026)

## Summary

Completion escalation uses a 5-tier model that enables daemon auto-completion for ~60% of routine work while preserving human review for high-judgment items.

## The Problem

When the daemon detects an agent has reported Phase: Complete, it must decide:
1. Auto-complete silently? (fast, but loses recommendations)
2. Surface for review? (preserves value, but creates backlog)
3. Block completion? (ensures human decision, but slows throughput)

Binary (auto vs manual) was too coarse - some completions need logging without blocking, others need mandatory review.

## The Decision

Five escalation levels with distinct responses:

| Level | Action | Trigger Examples |
|-------|--------|------------------|
| `None` | Auto-complete silently | Clean code-only work, all gates pass |
| `Info` | Auto-complete, log for optional review | Minor recommendations present |
| `Review` | Auto-complete, queue mandatory review | Knowledge-producing skills (investigation, architect) |
| `Block` | Do NOT auto-complete, surface immediately | Needs visual approval, human decision required |
| `Failed` | Do NOT auto-complete, failure state | Verification failed |

## The Decision Tree

```
1. VERIFICATION FAILED?
   └── YES → EscalationFailed

2. SKILL IS KNOWLEDGE-PRODUCING? (investigation, architect, research)
   └── YES → Has NextActions or Recommendation != "close"?
             └── YES → EscalationReview
             └── NO → EscalationInfo

3. VISUAL VERIFICATION NEEDS APPROVAL?
   └── YES → EscalationBlock

4. OUTCOME != "success"? (partial, blocked, failed)
   └── YES → EscalationReview

5. HAS RECOMMENDATIONS? (NextActions > 0)
   └── YES → file_count > 10?
             └── YES → EscalationReview
             └── NO → EscalationInfo

6. LARGE SCOPE? (file_count > 10)
   └── YES → EscalationInfo

7. OTHERWISE
   └── EscalationNone
```

## Expected Distribution

| Level | Percentage | Description |
|-------|------------|-------------|
| None + Info | ~60% | Auto-complete (routine work) |
| Review | ~30% | Auto-complete with mandatory review flag |
| Block + Failed | ~10% | Requires human decision |

## Why This Design

### Key insight: Knowledge work is different

Investigation, architect, and research skills produce recommendations and decisions that lose value if auto-completed without human absorption. These ALWAYS surface for review (EscalationReview or higher) regardless of verification status.

Code-only skills (feature-impl, systematic-debugging) can safely auto-complete when all gates pass.

### Key insight: Graduated response > binary

Binary "auto vs manual" forces a choice between:
- Auto-complete everything → lose recommendations, create drift
- Manual everything → orchestrator bottleneck, reduce throughput

Five tiers allow:
- Routine work: silent auto-complete
- Minor value: log for optional review
- Knowledge work: queue for mandatory review
- Human judgment: block until decided

### Trade-offs accepted

1. **Complexity** - 5 tiers instead of 2 adds decision surface
2. **Threshold tuning** - File count thresholds (10 files) are educated guesses
3. **Skill list maintenance** - Knowledge-producing skills list must stay current

Accepted because the alternative (binary) either loses value or creates bottlenecks.

## Implementation

```go
type EscalationLevel int

const (
    EscalationNone EscalationLevel = iota  // Auto-complete silently
    EscalationInfo                          // Auto-complete, log for review
    EscalationReview                        // Auto-complete, queue mandatory review
    EscalationBlock                         // Do NOT auto-complete
    EscalationFailed                        // Do NOT auto-complete, failure
)

func DetermineEscalation(skill string, verification *VerificationResult, synthesis *Synthesis, fileCount int) EscalationLevel
```

**File:** `pkg/verify/escalation.go`

## Evidence

- **Investigation:** `.kb/investigations/2025-12-27-inv-completion-escalation-model-completed-agents.md`
- **Guide:** `.kb/guides/completion.md` (Escalation Model section)
- **Implementation:** `pkg/verify/escalation.go`

## Related Decisions

- `.kb/decisions/2026-01-14-verification-bottleneck-principle.md` - Why verification needs gatekeeping
- `.kb/decisions/2026-01-16-single-daemon-orchestration-home.md` - Daemon architecture context
