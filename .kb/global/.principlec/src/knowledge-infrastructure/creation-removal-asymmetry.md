### Creation/Removal Asymmetry

Adding is local. Removing is global. This asymmetry is substrate-independent — it applies to code, knowledge, dependencies, configuration, and organizational structure.

**The test:** "Does removing this require understanding everything that depends on it?"

**What this means:**

- Adding something requires only local knowledge: what I'm building, where it goes, does it work
- Removing something requires global knowledge: what depends on it, what breaks, what assumes it exists
- This asymmetry is structural, not motivational — it's not laziness that prevents removal, it's that removal is genuinely harder
- Systems accrete because the cost of addition is always lower than the cost of removal
- The asymmetry compounds: each addition increases the global knowledge required for future removals

**What this rejects:**

- "We'll clean it up later" (later requires more global knowledge than now — it only gets harder)
- "Adding and removing are symmetric operations" (they require fundamentally different scopes of understanding)
- "The problem is discipline" (the problem is information asymmetry, not willpower)
- "Just delete it and see what breaks" (treats global dependencies as discoverable through failure — sometimes they are, often they aren't)

**The failure mode:** A system accumulates 1,200 investigations. Each was locally justified when created. Removing any one requires understanding whether anything references it, whether its findings were incorporated elsewhere, whether it established constraints still in effect. The 91% orphan rate isn't negligence — it's the asymmetry at work. Nobody added an investigation thinking "this will be hard to remove." But removal requires global knowledge that no single session possesses.

**Why this is substrate-independent:**

| Domain | Addition (local) | Removal (global) |
|--------|-----------------|------------------|
| Code | Add function to file | What calls it? What imports it? What tests it? |
| Knowledge | Write investigation | What references it? What constraints did it establish? |
| Config | Add config field | What reads it? What breaks without it? |
| Dependencies | Add package | What uses it transitively? What conflicts emerge? |
| Organization | Add team/role | What workflows assume it? What communication paths depend on it? |

**The relationship to accretion:** Accretion Gravity describes the code-level manifestation (agents add to existing files). Creation/Removal Asymmetry explains the mechanism: adding is local work that any task-scoped agent can do; removing requires global understanding that no task-scoped agent has. The asymmetry is why accretion is the default across all substrates, not just code.

**The relationship to Deploy or Delete:** Deploy or Delete says "finish migrations — remove the old system." This principle explains why that's hard: the old system has global dependencies that the new system's creator may not see. Building the replacement was local; removing the predecessor is global.

**Evidence:** Knowledge accretion model (Mar 2026). Observed across code (spawn_cmd.go 200→2000 lines), knowledge (1,200 investigations, 91% orphan rate), configuration (dual verification systems coexisting), and organizational structure (skill system accumulating compliance layers). The pattern is identical in each domain — the only difference is the substrate.

**Provenance:** Knowledge accretion model synthesis (Mar 2026). Thread: "Creation/removal asymmetry — adding is local, removing is global, and this is substrate-independent" (Mar 12 2026).
