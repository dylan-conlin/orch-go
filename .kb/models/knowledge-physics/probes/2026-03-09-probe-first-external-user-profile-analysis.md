# Probe: First External User Profile Analysis

**Model:** knowledge-physics
**Date:** 2026-03-09
**Status:** Complete

---

## Question

The knowledge-physics model claims the ideal domain is "any organization where institutional amnesia is expensive — regulated R&D, defense, finance, high-turnover teams." The harness-publication plan asks: who runs the kb system first? R&D lab, solo researcher, startup with turnover?

Testing: Which user profile best matches (a) the system's current capabilities and dependencies, (b) the four conditions for substrate physics to emerge (multiple agents, amnesiac, locally correct, no structural coordination), and (c) the lowest adoption friction for a v1 standalone system?

---

## What I Tested

### Test 1: Dependency audit — what does kb-cli require to run?

```bash
kb --help  # Confirmed: standalone binary, no server dependency
kb create --help  # Subcommands: investigation, decision, guide, plan, research, specification
kb reflect --help  # 11 reflection types, all operate on .kb/ directory structure
```

**kb-cli hard dependencies:**
- Go binary (compiles to standalone)
- Git repo (`.kb/` lives inside a git repo for version control)
- File system (all artifacts are markdown files)
- No database, no server, no API keys required for core operations

**kb-cli soft dependencies (enhance but not required):**
- `bd` (beads) — issue tracking, used by reflect `--create-issue` and linking
- `kn` — quick entries (decisions, constraints, attempts)
- `orch` — agent spawning, daemon, orchestration
- `skillc` — skill compilation and deployment
- AI API access — only for `kb ask` (LLM synthesis), not core create/reflect/context

### Test 2: What does the investigation/probe/model cycle actually require?

Walked through the workflow:

1. **Create investigation:** `kb create investigation "my-question"` → creates `.kb/investigations/YYYY-MM-DD-inv-my-question.md`
2. **Do the work:** Fill in findings manually or via agent
3. **Create model:** Manual creation in `.kb/models/{name}/model.md` (no `kb create model` command exists)
4. **Create probe:** Manual creation in `.kb/models/{name}/probes/YYYY-MM-DD-probe-*.md` (no `kb create probe` command exists)
5. **Reflect:** `kb reflect` detects synthesis opportunities (3+ investigations on same topic)
6. **Context injection:** `kb context "query"` returns relevant artifacts for agent/human priming

**Critical gap:** No `kb create model` or `kb create probe` commands. These are the most structurally important artifacts (models are "the fundamental unit" per the knowledge-physics model) yet they have no tooling support. Users must manually create directories and files.

### Test 3: Onboarding experience audit — what would a new user's first hour look like?

```bash
kb init  # Creates .kb/ directory structure
# Then what?
```

Examined kb init behavior:
```bash
kb init --help
```

No guided onboarding. After `kb init`, user has an empty .kb/ directory and no guidance on what to create first. No tutorial, no example model, no "your first investigation" workflow.

**Additional finding:** `kb init` creates only 3 directories: `investigations/`, `decisions/`, `guides/`. It does NOT create `models/` — the artifact the knowledge-physics model calls "the fundamental unit of knowledge organization." Tested:

```bash
cd /tmp && mkdir kb-test && cd kb-test && git init -q && kb init
# Output: Created .kb/investigations/, .kb/decisions/, .kb/guides/
# Missing: .kb/models/, .kb/plans/, .kb/threads/, .kb/quick/
```

### Test 4: Analogous tool adoption patterns (web research)

Examined adoption patterns of four comparable structured knowledge tools:

**ADRs (Architectural Decision Records):**
- Nygard introduced ADRs in 2011 on his consulting projects. 6-10 developers rotating through those projects "appreciated the context."
- 7 years later ThoughtWorks placed them in "Adopt" ring (strongest endorsement).
- By 2025, UK Government mandated ADR framework across public sector after pilot across DSIT and other departments.
- Pattern: single architect → project team → industry standard. Classic bottom-up.

**Zettelkasten / Obsidian:**
- Obsidian: 1.5M+ monthly active users (2026), 22% YoY growth. User base: 64.3% male, 25-34 age group, skewing developer/programmer.
- Fundamentally individual-first. Team/enterprise features were added later (10K+ orgs now using it).
- Roam Research drove initial PKM interest in 2020 via "#roamcult" community of early adopters, but Obsidian overtook it with free tier and local-first data ownership.

**Electronic Lab Notebooks (Benchling, eLabFTW):**
- Benchling gave product free to academia → 200K+ scientists at 7K+ institutions → scientists carried it to industry. 1 in 4 biotech IPOs (2020-22) built on Benchling.
- Indiana University School of Medicine: 15 PIs volunteered as early adopters (Sep 2018), expanded to 829 users across 22/25 departments by Aug 2021. 70% said lab leadership encouraged use.
- University of Southampton: senior staff adopted more successfully than junior. "Researchers whose supervisors supported ELN adoption were far more positive themselves."
- eLabFTW: created 2012 by a research engineer who "realized there was a tool missing from his daily workflow." Practitioner solving own problem.

**Pattern across all four:** Successful knowledge tools follow a consistent 4-stage adoption:
1. Individual practitioner solves their own problem
2. Nearby collaborators see value through exposure
3. Champions carry practice to new contexts
4. Institutional/industry endorsement codifies the practice

Bottom-up adoption is stickier. Top-down support is necessary but not sufficient (Southampton ELN data). Only 45% of employees use KM tools even after institutional adoption (IDC).

### Test 5: Four-condition analysis per user profile

The knowledge-physics model requires four conditions for substrate dynamics to emerge:
1. Multiple agents write to the substrate
2. Agents are amnesiac
3. Contributions are locally correct
4. No structural coordination mechanism exists

| Profile | Multiple writers? | Amnesiac? | Locally correct? | No coordination? |
|---------|------------------|-----------|-----------------|-----------------|
| **Solo researcher + AI agents** | Yes (human + 1-3 AI agents) | AI agents: yes. Human: partially (forgets across months) | Yes (each investigation is self-contained) | Yes (no shared context between sessions) |
| **R&D lab (5-15 researchers)** | Yes (many humans, possibly AI) | Yes (researchers come and go, forget prior work) | Yes (each experiment documented independently) | Yes (lab knowledge is scattered across notebooks/drives) |
| **Startup with turnover** | Yes (rotating team) | Yes (departing employees = total amnesia) | Varies (startup quality varies) | Yes (institutional knowledge often undocumented) |

All three profiles meet the four conditions. The differentiator is not whether the physics apply, but **adoption friction** and **time to first value**.

---

## What I Observed

### Finding 1: The solo researcher with AI agents is Dylan's profile replicated

The current system was built by a solo developer using AI agents. The first external user who replicates this profile has the lowest adoption friction because:
- They already use AI coding assistants (Claude Code, Cursor, Copilot)
- They understand the "amnesiac agent" problem from personal experience
- They can start with a single project (no organizational buy-in needed)
- The investigation/probe/model cycle maps directly to how they already work (try things, record findings, build mental models)

**But:** The system's most impressive evidence (1,166 investigations, 85.5% orphan rate, coordination failure across agents) comes from *high volume* — many agents over months. A solo researcher might not generate enough volume for the physics to manifest visibly in weeks. They'd need patience.

### Finding 2: The R&D lab has the strongest pain but highest adoption friction

R&D labs lose institutional knowledge when:
- Postdocs leave (2-3 year cycles)
- Failed experiments aren't documented ("negative results")
- Protocols evolve without recording why
- New lab members repeat old mistakes

This is exactly the problem the kb system solves. But adoption friction is high:
- Lab IT policies may restrict tooling
- Researchers have established workflows (lab notebooks, Confluence, shared drives)
- PI buy-in required
- Training needed for non-developer users

**Critical observation:** R&D labs don't use Git as their primary workflow. The kb system requires a Git repo. This is a blocking mismatch for most wet-lab researchers.

### Finding 3: The startup with turnover has acute pain but wrong incentive timing

Startups feel knowledge loss most acutely *after* someone leaves — but by then it's too late. The kb system needs to be running *before* turnover happens to capture knowledge. This creates a temporal incentive mismatch: you need to adopt the system when the pain is low (team is stable) for it to pay off when the pain is high (team turns over).

Startups also tend to optimize for velocity over documentation. "We'll document later" = never. The kb system's ongoing ceremony (creating investigations, probes, models) competes with shipping features.

### Finding 4: Missing tooling for the core workflow

The kb system's core value proposition is the investigation/probe/model cycle. But:
- `kb create model` doesn't exist
- `kb create probe` doesn't exist
- No template for models (TEMPLATE.md exists in .kb/models/ but isn't wired to a command)
- No guided onboarding workflow
- No visualization of the model graph (which investigations connect to which models)
- `kb reflect` is the only automated feedback — but it requires 3+ investigations to trigger, meaning the first hour produces no automated value

For a first external user, the "magic moment" (first time they see the system's value) requires:
1. Create 3+ investigations on a topic
2. Run `kb reflect` to see synthesis opportunity
3. Manually create a model
4. Create a probe against the model
5. See the model evolve based on probe findings

Steps 3-5 have no tooling support. The magic moment requires too many manual steps.

### Finding 5: The viable first user profile

**Profile: Solo technical researcher/developer who already uses AI agents for knowledge work.**

Characteristics:
- Uses Claude Code, Cursor, or similar AI coding assistant daily
- Works on a complex project where they forget their own prior decisions
- Comfortable with Git and CLI tools
- Has 1-3 ongoing projects where knowledge compounds
- Frustrated by "I solved this before but can't remember how"
- Not working in a team (no coordination overhead to justify initially)

Why this profile:
- **Lowest adoption friction:** Already in Git, already uses CLI, already has AI agents
- **Fastest time to first value:** Personal pain (forgetting own decisions) is felt immediately, not after team turnover
- **Self-service:** No organizational buy-in, no IT approval, no team training
- **The "Dylan archetype":** They're the type of person who would build this if they had time — we're giving them the system pre-built

What they need (minimum viable standalone):
1. `kb init` with guided onboarding ("your first investigation")
2. `kb create model` command
3. `kb create probe` command
4. A 15-minute tutorial: "From first question to first model"
5. `kb reflect` working out of the box on a small corpus
6. Claude Code / AI agent integration: CLAUDE.md template that teaches agents to use the investigation/probe/model cycle

What they DON'T need:
- orch (agent orchestration)
- beads (issue tracking)
- skillc (skill compilation)
- daemon (autonomous processing)
- Multiple accounts, tmux management, SSE monitoring
- Multi-model routing, OpenCode server

### Finding 6: The R&D lab is the second user, not the first

The R&D lab adoption path:
1. A single researcher in the lab adopts kb (they ARE the solo researcher profile)
2. They produce visible value (models that prevent re-investigation of known failures)
3. Other lab members see the value and start contributing
4. The lab adopts kb as a shared knowledge system
5. The physics emerge naturally as multiple contributors add to the substrate

This is the classic bottom-up adoption pattern. The first external user is NOT the lab — it's the one person in the lab who cares about knowledge persistence. R&D lab is Phase 3-4 adoption, not Phase 3 launch.

---

## Model Impact

- [x] **Extends** model with: First external user profile analysis — the knowledge-physics model identifies the ideal domain (institutional amnesia) but doesn't specify adoption sequencing. The finding is that the ideal *first* user is not the ideal *domain* — the solo technical researcher with AI agents has the lowest adoption friction and fastest time to value, even though R&D labs have the strongest pain. The four conditions for substrate physics apply to all three profiles, so the differentiator is adoption friction, not physics applicability. Bottom-up adoption (solo → team → org) matches proven knowledge-tool adoption patterns (Zettelkasten, ADRs, lab notebooks). The model should note that identifying the ideal domain for the *physics* is different from identifying the ideal first *user* of the system.

- [x] **Extends** model with: Tooling gap discovery — the model's critical invariant #2 ("models are the fundamental unit of knowledge organization") is contradicted by the tooling: no `kb create model` or `kb create probe` commands exist. The most important artifacts have the least tooling support. For external adoption, this gap must be closed — the first user can't be expected to manually create directory structures and templates for the artifacts the system considers fundamental.

---

## Notes

### Recommended user profile one-pager

**Name:** Solo Technical Researcher (STR)

**Archetype:** Developer or researcher working alone on a complex long-running project, using AI agents (Claude Code, Cursor) for investigation and coding work.

**Pain:** "I keep re-investigating things I already figured out. My AI agents have no memory across sessions. I have investigations and notes scattered across files with no structure."

**Existing tools:** Git, CLI, AI coding assistant, maybe Notion/Obsidian for notes.

**What they need from kb:**
| Need | Priority | Current status |
|------|----------|----------------|
| `kb init` with guided start | P0 | Exists but bare |
| `kb create investigation` | P0 | Exists |
| `kb create model` | P0 | **MISSING** |
| `kb create probe` | P0 | **MISSING** |
| `kb reflect` | P1 | Exists |
| `kb context` for AI priming | P1 | Exists |
| 15-minute tutorial | P0 | **MISSING** |
| CLAUDE.md template | P1 | **MISSING** for external use |

**What they DON'T need:** orch, beads, skillc, daemon, multi-agent orchestration.

**Time to first value:** ~1 hour (with tooling fixes), currently ~1 day (manual setup).

**Adoption path:** Solo use → shared repo → team → organization (if applicable).

### Relationship to sibling investigations

- `orch-go-hrgor` (minimal kb substrate): The STR profile defines what "minimal" means — kb-cli + git + AI agent. No orch, no beads, no skillc.
- `orch-go-5j2cq` (human probes): The STR uses AI agents for investigations but may write probes manually (shorter, more structured). Both paths needed.
