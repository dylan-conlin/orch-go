### Escape Hatches

Critical paths need independent secondary paths. When infrastructure can fail, build escape hatches that don't depend on what failed.

**The test:** "If the primary path fails, can I complete this work through another path?"

**What this means:**

- Critical infrastructure needs redundant execution paths
- Escape hatches must be independent of primary infrastructure
- "Independent" means: doesn't share failure modes, provides visibility, can complete work

**What this rejects:**

- Single-path critical infrastructure (all eggs in one basket)
- "We'll fix it when it breaks" (can't fix from inside broken infrastructure)
- Backup paths that share dependencies with primary (correlated failure)

**The pattern:** Primary path (daemon + OpenCode API) + Escape hatch (manual claude CLI spawning)

**Why this is distinct from Graceful Degradation:** GD says core works without optional layers. Escape Hatches says critical paths need *independent alternative paths* for when core itself fails.
