# Investigation: Trust Calibration Meta-Pattern

**Date:** 2026-01-09
**Status:** Active
**Question:** Why does Dylan defer to AI recommendations even when he has relevant experience the AI doesn't know about?

---

## Summary

**Pattern:** Dylan has tool/domain knowledge (foreman, Docker) but doesn't assert it because he "assumes AI knew best." System doesn't surface Dylan's past experience, so AI makes elaborate recommendations without that context. Dylan defers instead of correcting.

**Recent examples:**
1. **Launchd recommendation** - Dylan used foreman in the past, but system didn't know. AI recommended launchd + tmuxinator. Dylan deferred. 2 weeks of problems.
2. **Price-watch button** - 510-line investigation with 5 hypotheses instead of 30-second DevTools check. Dylan waited instead of asserting "just check the browser."

---

## The Loop

```
1. Dylan has relevant experience (foreman, browser debugging)
       ↓
2. System doesn't surface this context
       ↓
3. AI makes elaborate recommendation/investigation without that knowledge
       ↓
4. Dylan assumes "AI knows best" and defers
       ↓
5. Problem persists or escalates
       ↓
6. Dylan compensates by providing context manually ("embarrassing thing: I've used foreman!")
       ↓
7. System still doesn't capture this for next time
       ↓
LOOP REPEATS
```

---

## Evidence

### Example 1: Launchd Recommendation

**What Dylan knew:**
- Used foreman in past
- Used Docker before
- Has opinions on process management

**What system surfaced:** Nothing about Dylan's tool experience

**What AI recommended:** launchd + tmuxinator (elaborate 3-layer architecture)

**What Dylan did:** Deferred to recommendation, "assumed AI knew best"

**Outcome:** 2 weeks of reliability issues, 186 investigations mentioning "restart"

**Post-mortem:** `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md`

---

### Example 2: SvelteKit Button Investigation

**What Dylan knew:**
- How to use browser DevTools
- Simple debugging approach: check console, inspect DOM

**What system surfaced:** Nothing about debugging workflow

**What AI did:** 510-line investigation with 5 elaborate hypotheses, marked "Blocked - Awaiting Browser DevTools Inspection"

**What Dylan did:** Waited for agent to ask for DevTools check instead of saying "just open DevTools"

**Outcome:** Investigation theater instead of 30-second test

**Investigation:** `price-watch/.kb/investigations/2026-01-09-inv-sveltekit-percentage-toggle-not-rendering.md`

---

## Why This Happens

### Dylan's Side: Trust Miscalibration

**The assumption:** "AI has access to all knowledge, probably knows better than me about [foreman/Docker/debugging]"

**The reality:** AI only knows what's in context. Dylan's past tool usage, preferences, debugging workflows are NOT surfaced.

**The dynamic:** Dylan has relevant knowledge but doesn't assert it because he's compensating for perceived AI superiority.

**Quote:** "this was an example of where i just assumed that the ai knew best"

---

### System Side: No Context Surfacing

**What's missing:**
- Dylan's tool experience (foreman, Docker, specific libraries)
- Dylan's debugging preferences (DevTools first, not elaborate hypotheses)
- Dylan's past project contexts (what tools were used where)
- Dylan's implicit knowledge (industry standard tools vs custom solutions)

**Current state:** kb/investigations/decisions exist but don't capture:
- "Dylan used foreman at [company] for [use case]"
- "Dylan's debugging protocol: DevTools → console → DOM → hypothesis"
- "Dylan prefers industry tools (foreman) over custom (launchd)"

**Result:** Every session starts cold on Dylan's context.

---

### The Compensation Pattern

**What Dylan does:** Provides context manually after problem emerges
- "embarrassing thing: I've used foreman before!"
- "I've also used Docker!"

**What this reveals:** Dylan knows he has relevant knowledge but didn't assert it upfront because system didn't prompt for it.

**Violates:** Pressure Over Compensation principle - Dylan compensating for system's missing context surfacing instead of letting it fail and creating pressure to improve.

---

## The Investigation Theater Sub-Pattern

**Price-watch example shows another failure mode:**

**Agent's approach:**
1. Write 510-line investigation
2. Generate 5 elaborate hypotheses
3. Create detailed test plans
4. Mark as "Blocked" waiting for Dylan

**What should have happened:**
1. Use Glass (browser automation we have)
2. Check DevTools console (30 seconds)
3. Inspect DOM (30 seconds)
4. Report findings
5. THEN hypothesize if needed

**Why this happened:**
- Agent doesn't have Glass context loaded
- Agent doesn't know Dylan's debugging workflow
- Agent optimizes for comprehensive investigation (looks thorough) vs quick answer (looks hasty)
- Blocking on Dylan creates appearance of thoroughness

**The irony:** More investigation ≠ better understanding. Quick test > elaborate hypothesis.

---

## Principles Violated

| Principle | How Violated |
|-----------|--------------|
| **Pressure Over Compensation** | Dylan compensating by providing context manually instead of letting system fail |
| **Evidence Hierarchy** | Investigation theater (reasoning) over testing (evidence) |
| **Surfacing Over Browsing** | System doesn't surface Dylan's tool experience, requires manual provision |
| **Provenance** | Recommendations based on reasoning ("doesn't fit polyrepo") not evidence (no foreman prototype) |

---

## What This Means

**Trust calibration is bidirectional:**

**Dylan → AI:**
- Currently: "AI probably knows best, I'll defer"
- Should be: "AI only knows what's in context, I need to assert my knowledge"

**AI → Dylan:**
- Currently: "I'll make comprehensive recommendations based on available context"
- Should be: "Do you have experience with [tool]? Have you used [approach] before?"

**The gap:** No mechanism to:
1. Surface Dylan's past tool usage
2. Prompt Dylan to assert relevant knowledge
3. Calibrate AI confidence ("I don't know if Dylan has tried foreman")
4. Prefer quick tests over elaborate investigation

---

## Proposed Solutions

### 1. Tool Experience Surface (Immediate)

**Mechanism:** When considering tools (foreman, overmind, Docker), prompt:
- "Have you used [tool] before?"
- "What's your experience with [approach]?"
- "Do you have preferences here?"

**Implementation:** LLM-detect tool selection questions, inject prompt

---

### 2. Debugging Protocol Surface (Immediate)

**Mechanism:** When investigation starts, check:
- "What's your debugging workflow for [domain]?"
- "Should I test first or investigate first?"
- "Do you want quick answer or comprehensive analysis?"

**Implementation:** Investigation skill update

---

### 3. Dylan Context Database (Medium-term)

**Mechanism:** Persistent store of Dylan's:
- Tool experience: "Used foreman at [company], preferred over systemd"
- Debugging workflows: "Browser issues: DevTools first, then hypothesize"
- Preferences: "Prefer industry standard tools over custom solutions"
- Past projects: "price-watch uses Docker, orch-go uses Go, etc."

**Implementation:** New kb category or CLAUDE.md section

---

### 4. Investigation Test-First Gate (Immediate)

**Mechanism:** Before writing elaborate hypotheses, require:
- "What's the simplest test I can run right now?"
- "Can I test this in 60 seconds?"
- "Why am I blocking on Dylan instead of testing?"

**Implementation:** Investigation skill update, Glass integration

---

### 5. Confidence Calibration (Medium-term)

**Mechanism:** AI explicitly states what it doesn't know:
- "I don't have context on your tool preferences"
- "I'm recommending launchd, but I don't know if you've used foreman"
- "I'm creating elaborate investigation, but I don't know if you prefer quick tests"

**Implementation:** Recommendation/investigation templates

---

## Next Steps

1. **Capture Dylan's tool experience** - Create section in global CLAUDE.md or kb for:
   - Tools Dylan has used (foreman, Docker, etc.)
   - Debugging workflows by domain
   - Preferences (industry tools vs custom)

2. **Update investigation skill** - Add "test-first" gate:
   - What's the 60-second test?
   - Why am I blocking instead of testing?
   - Do I have automation (Glass) available?

3. **Update orchestrator skill** - Add tool experience prompts:
   - "Have you used [tool] before?"
   - "What's your experience with [approach]?"

4. **Glass integration** - Make browser automation default for frontend investigations:
   - DevTools check (console, network, DOM)
   - Visual verification
   - Screenshot capture

---

## Key Insight

**The embarrassment isn't that Dylan used foreman before.**

**The embarrassment is that the system didn't ask.**

Dylan has relevant knowledge. System doesn't surface it. Dylan defers to AI. Problem persists. Dylan compensates manually. System still doesn't capture it.

The loop continues until someone (Dylan or orchestrator) recognizes the pattern and forces the system to change.

---

## References

**Post-mortems:**
- `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md` - Launchd recommendation failure

**Investigations:**
- `price-watch/.kb/investigations/2026-01-09-inv-sveltekit-percentage-toggle-not-rendering.md` - Investigation theater example

**Principles:**
- `~/.kb/principles.md` - Pressure Over Compensation, Evidence Hierarchy, Surfacing Over Browsing

**Decisions:**
- `.kb/decisions/2026-01-09-dashboard-reliability-architecture.md` - Overmind decision (correct, but late)
