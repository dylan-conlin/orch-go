# Meta-Orchestrator Session Completion

Like orchestrators complete workers, you complete orchestrator sessions. And like orchestrators have session end rituals, you have meta-session patterns.

---

## Completing an Orchestrator Session

```bash
orch complete --orchestrator <session-id>
```

**What this checks:**
- SESSION_HANDOFF.md exists and has content
- Git pushed (work not stranded locally)
- Follow-up issues created for discovered work
- Friction captured (or explicitly none)

---

## Synthesis Across Sessions

When multiple orchestrator sessions complete, synthesize:

| Question | Action |
|----------|--------|
| What patterns emerge? | Note in meta-session handoff |
| What friction recurs? | Create system improvement issue |
| What decisions were made? | Ensure captured in kb |
| What's the overall progress? | Update epic/milestone status |

---

## Meta-Session Reflection

Before ending your meta-orchestrator session:

### 1. Knowledge Check
- What strategic insights emerged?
- Should any become decisions or principles?
- What did orchestrators learn that should be shared?

### 2. System Reaction Check
- Does the orchestrator skill need updating?
- Does the meta-orchestrator skill need updating?
- Are there recurring gaps that need new mechanisms?

### 3. Friction Check
- What was harder than it should have been at the meta level?
- Are you compensating for missing infrastructure?
- What would make the next meta-session easier?

### 4. Next Session Setup
- What's the strategic priority?
- What orchestrator sessions should be spawned?
- What context should the next meta-orchestrator have?

---

## Success Criteria

A successful meta-orchestrator session:

- [ ] Set clear strategic focus for orchestrator sessions
- [ ] Spawned orchestrator sessions (not workers directly)
- [ ] Reviewed orchestrator handoffs
- [ ] Captured strategic decisions
- [ ] Noted system improvement opportunities
- [ ] Produced clear handoff for next meta-session

---

## The Meta-Handoff

Like orchestrators produce SESSION_HANDOFF.md, you produce meta-level handoffs:

```markdown
# Meta-Session Handoff - [Date]

## Strategic Focus This Session
[What direction was set]

## Orchestrator Sessions Spawned
[List with outcomes]

## Decisions Made
[Strategic choices, with rationale]

## System Evolution
[Improvements identified, created, or made]

## Friction Encountered
[What was hard at the meta level]

## Next Meta-Session
[Strategic priority, first actions]
```

---

## The Frame Shift Discipline

Remember: You exist because incremental optimization from within the orchestrator frame isn't enough. Your value is seeing patterns orchestrators can't see.

If your session produced only tactical optimizations, you may have dropped into the orchestrator frame. The test: Did you make WHICH decisions, or just better HOW decisions?
