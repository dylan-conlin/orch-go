### Accretion Gravity

Agents add code where similar code already exists. Without structural constraints, the largest file always gets larger.

**The test:** "Does the agent have a reason to create a new file, or will it add to the existing one?"

**What this means:**

- Agents are task-scoped — they optimize for completing their task, not system health
- "Add feature X to spawn" → agent opens spawn_cmd.go, adds X there (because that's where spawn logic is)
- No agent will spontaneously create pkg/spawn/gates/hotspot.go — that's not in the task
- Over N agents, the file with the most logic attracts the most new logic
- This is gravitational: the more mass (code) a file has, the more it attracts

**What this rejects:**

- "Agents understand the architecture" (they understand their task)
- "Good code organization will emerge" (it won't — accretion is the default)
- "We'll refactor later" (later never comes; the file is now 2,000 lines)
- "Each feature was added correctly" (locally correct, structurally wrong)

**The failure mode:** 25 agents each add one feature to spawn_cmd.go. Each addition is correct. The file grows from 200 to 2,000 lines. It now does backend selection, concurrency checks, worktree management, context generation, hotspot analysis, triage bypass, gap analysis, beads tracking, and the actual spawn — all in one function. 495 bugs follow. The agents that fix the bugs add more code to the same file.

**The fix is structural constraints, not better agents:**

- Explicit boundaries in CLAUDE.md: "new logic goes in pkg/, not cmd/"
- File-level rules: "if adding >10 lines to X, create a new package"
- Architectural review at completion, not just functional review

**Why distinct from Coherence Over Patches:** CoP addresses *fix accumulation* (patches making code incoherent). Accretion Gravity addresses *feature accumulation* (new capabilities gravitating to existing files). CoP fires when bugs cluster. Accretion Gravity fires when features cluster. Both produce god functions, but through different mechanisms.
