### Deploy or Delete

A system change isn't complete when the new system exists — it's complete when the old system is removed. The gap between "implemented" and "deployed" is where dual authorities form.

**The test:** "Does the old system still exist alongside the new one?"

**What this means:**

- Building new infrastructure is satisfying. Removing old infrastructure is tedious. The system accumulates dual authorities in the gap.
- When both old and new systems exist, agents and humans navigate conflicting signals. Neither system is fully authoritative.
- "Implemented" ≠ "deployed." Code that assumes config exists, config that was never populated, prose constraints that duplicate hook enforcement — all are declarations without deployment.
- The primary failure mode isn't "things break" — it's "things half-work."

**What this rejects:**

- "The new system is built, we'll remove the old one later" (later never comes — 18 investigations in 30 days proved this)
- "Both can coexist during the transition" (perpetual transition is the failure mode)
- "We'll deprecate it gradually" (deprecation without removal is dual authority)

**The failure mode:** New verification system (V0-V3 levels) built alongside old tier system. Neither removed. Hook infrastructure built to enforce behavioral constraints. Skill text still contains the same constraints. Agents receive conflicting signal types — infrastructure blocks vs prose guidance. Account routing code requires `config_dir` field. Field never added to config file. Spawn templates check for `ProducesInvestigation` flag. Flag never populated. Each is a declaration that was never deployed.

**The pattern:**

```
Build new system → Feel satisfaction → Move to next thing
                                          ↓
                        Old system lingers → Dual authority forms
                                          ↓
                        Conflicting signals → Cognitive load accumulates
                                          ↓
                        18 investigations in 30 days → All trace here
```

**Why distinct from Infrastructure Over Instruction:** IoI says choose the right enforcement layer. This principle says *when you've chosen it, finish the migration* — remove the declaration layer. IoI is about where to build. Deploy or Delete is about completing what you built.

**Why distinct from Coherence Over Patches:** CoP addresses patches accumulating in the same area. This addresses parallel systems coexisting — not patches on one system, but two systems claiming the same authority.

**Evidence:** March 2026 synthesis session. 18 configuration-drift investigations in 30 days all traced to incomplete migrations: code assuming config that doesn't exist (daemon plist, accounts.yaml config_dir), dual verification systems (tier + levels), prose constraints duplicating hook enforcement (50+ skill constraints vs 6 active hooks). Orchestrator skill simplification (2,368→422 lines) was the first successful migration completion — removing prose constraints after hook deployment.

**Provenance:** Configuration-drift defect class analysis (Mar 5 2026), orchestrator-skill model synthesis (6 investigations, Jan-Mar 2026).
