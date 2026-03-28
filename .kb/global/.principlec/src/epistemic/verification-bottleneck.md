### Verification Bottleneck

The system cannot change faster than a human can verify behavior. Velocity without verification is regression with extra steps.

**The test:** "Has a human observed this working, or just read that it works?"

**What this means:**

- Each change requires human behavioral verification before the next
- "Agent says it's fixed" is not verification
- Commit messages and synthesis files are claims, not evidence
- If verification takes 10 minutes, changes cannot happen faster than every 10 minutes

**What this rejects:**

- Spawning agents to investigate agents that are investigating agents
- System self-reporting health while deteriorating
- Trusting artifacts over observed behavior
- "The dashboard says healthy" when no human checked the dashboard

**The failure mode:** Each agent is locally correct. Each commit does what it says. But no one outside the system is checking if the *system* works. The dashboard says healthy. The agents say complete. The commits say fixed. And then you roll back 347 commits.

**Why distinct from Provenance:** Provenance asks "does this trace to evidence?" Verification Bottleneck asks "did a human *outside* the system observe the evidence?"

**The math:** If human verification takes V minutes, and the system produces changes at rate C per minute, then C × V must be ≤ 1. Otherwise, unverified changes accumulate faster than they can be checked.

**Verification scales with risk, not uniformly.** Binary verification (inspect everything equally or rubber-stamp everything) breaks flow. Calibrated verification matches effort to risk:

| Level | What it checks | When it applies |
|-------|---------------|-----------------|
| V0 (Acknowledge) | Did agent finish? | Trivial changes, config tweaks |
| V1 (Artifacts) | Are deliverables present? | Investigations, designs, knowledge work |
| V2 (Evidence) | Is there evidence of testing? | Features, bug fixes, implementation |
| V3 (Behavioral) | Did a human observe it working? | Architectural changes, UI work, trust-critical paths |

Most work should need light review. Architectural changes demand deep review. The system should surface the trust tier so the human can modulate attention — not force uniform depth on everything. Flow state is the scarce resource; verification that preserves flow sustains the system, verification that breaks flow gets skipped entirely.

**Why distinct from the core constraint:** The core ("can't change faster than you can verify") holds. The evolution is that verification has *resolution* — not all changes require the same depth. Uniform verification either wastes attention (everything deep) or wastes trust (everything rubber-stamped).
