**TLDR:** Question: Should spawned agents receive ecosystem context? Answer: YES - inject tooling knowledge (bd, kb, orch commands) universally, but inject project registry only for ecosystem repos or cross-project work. Use tiered approach: ECOSYSTEM.md exists at ~/.orch/ already - inject concise summary section into SPAWN_CONTEXT.md for ecosystem repos.

---

# Investigation: Scope Ecosystem Context Injection for Spawned Agents

**Question:** What ecosystem context should spawned agents receive, and under what conditions?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Worker agent (design-session)
**Phase:** Complete
**Next Step:** Spawn follow-up to implement EcosystemContext injection in pkg/spawn/context.go
**Status:** Complete
**Confidence:** High (85%) - Clear problem, existing infrastructure, straightforward solution

---

## Problem Statement

Spawned agents lack knowledge of Dylan's local projects and tools:
- **glass** - Browser automation MCP server
- **beads** - Issue tracking system (bd CLI)
- **kb-cli** - Knowledge base CLI
- **orch-go** - Orchestration system
- **kn** - Quick knowledge entries

When agents need to interact with these tools, they try GitHub searches instead of using local commands.

**Core Tension:** Should ALL spawns get ecosystem context (even in unrelated projects like price-watch)? Or only spawns in ecosystem repos?

---

## Findings

### Finding 1: ECOSYSTEM.md Already Exists

**Evidence:** `~/.orch/ECOSYSTEM.md` contains comprehensive documentation:
- Quick reference table of all repos with CLI names
- Per-repo documentation (path, purpose, key commands)
- Cross-repo data flows
- 150+ lines of structured ecosystem knowledge

**Source:** `cat ~/.orch/ECOSYSTEM.md`

**Significance:** The documentation already exists. Problem is it's not being injected into spawn contexts.

---

### Finding 2: Orchestrator Skill References ECOSYSTEM.md

**Evidence:** From `~/.claude/skills/meta/orchestrator/SKILL.md`:
```markdown
**Cross-repo architecture:** See `~/.orch/ECOSYSTEM.md` for comprehensive guide on all repos 
in Dylan's orchestration system, how they communicate, and data flows
```

**Significance:** Orchestrators know about ECOSYSTEM.md. Workers don't - they aren't given this context in SPAWN_CONTEXT.md.

---

### Finding 3: OrchEcosystemRepos Already Defines Scope

**Evidence:** `pkg/spawn/kbcontext.go:15-22`:
```go
var OrchEcosystemRepos = map[string]bool{
    "orch-go":        true,
    "orch-cli":       true,
    "kb-cli":         true,
    "orch-knowledge": true,
    "beads":          true,
    "kn":             true,
}
```

**Missing from list:** glass, beads-ui-svelte, skillc, agentlog (mentioned in ECOSYSTEM.md)

**Significance:** Already have infrastructure for "is this an ecosystem repo?" check. Just needs expansion and use in spawn context.

---

### Finding 4: Current Injection Points in SPAWN_CONTEXT.md

**Evidence:** From `pkg/spawn/context.go` template:
```
CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: {{.ProjectDir}}/CLAUDE.md
```

**What's missing:**
- No mention of ecosystem tools
- No `~/.orch/ECOSYSTEM.md` reference
- No tooling command quick reference

**Significance:** Clear injection point exists. Need to add ecosystem context section.

---

### Finding 5: Three Categories of Ecosystem Knowledge

**Analysis:** Based on ECOSYSTEM.md and skill content, there are three distinct types of knowledge:

| Category | Examples | Who Needs It | Injection Rule |
|----------|----------|--------------|----------------|
| **Core Tooling** | bd, kb, orch, kn commands | ALL spawns | Always inject (30 lines) |
| **Project Registry** | What repos exist, their paths | Cross-project work | Ecosystem repos only |
| **Full ECOSYSTEM.md** | Data flows, architecture | Orchestrators only | Reference only, don't embed |

**Significance:** Need tiered injection, not all-or-nothing.

---

## Synthesis

**Key Insights:**

1. **ECOSYSTEM.md is too large to embed** - 150+ lines, ~4k tokens. Would bloat every spawn context. Better to inject a concise summary.

2. **Tooling knowledge is universal** - Every spawned agent should know `bd`, `kb`, `orch`, `kn` exist. They're the orchestration primitives.

3. **Project registry is scoped** - Only ecosystem repo spawns need to know about sibling repos. Price-watch agents don't care about glass.

4. **Reference beats embedding** - Tell agents ECOSYSTEM.md exists, don't duplicate it.

**Answer to Investigation Question:**

Inject a ~30-40 line "Ecosystem Context" section into SPAWN_CONTEXT.md that:
1. Lists core tooling commands (always)
2. Lists ecosystem repos (for ecosystem projects only)
3. Points to ~/.orch/ECOSYSTEM.md for comprehensive details

---

## Recommended Implementation

### 1. Expand OrchEcosystemRepos

```go
// pkg/spawn/kbcontext.go
var OrchEcosystemRepos = map[string]bool{
    "orch-go":        true,
    "orch-cli":       true,
    "kb-cli":         true,
    "orch-knowledge": true,
    "beads":          true,
    "beads-ui-svelte": true,
    "kn":             true,
    "glass":          true,
    "skillc":         true,
    "agentlog":       true,
}
```

### 2. Add IsEcosystemRepo Helper

```go
// pkg/spawn/ecosystem.go (new file)
func IsEcosystemRepo(projectName string) bool {
    return OrchEcosystemRepos[projectName]
}

func GenerateEcosystemContext(projectName string) string {
    // Always include core tooling
    // Conditionally include project registry if ecosystem repo
}
```

### 3. Update SPAWN_CONTEXT.md Template

Add after "CONTEXT AVAILABLE" section:
```
{{if .EcosystemContext}}
## ECOSYSTEM TOOLING

{{.EcosystemContext}}
{{end}}
```

### 4. Concise Ecosystem Summary (~35 lines)

```markdown
## ECOSYSTEM TOOLING

**Core Commands (available in all orch-go managed projects):**
- `bd` - Beads issue tracking (create, show, close, ready)
- `kb` - Knowledge base (create investigation, context search)
- `kn` - Quick knowledge (decide, tried, constrain, question)
- `orch` - Agent orchestration (spawn, status, complete)

{{if .IsEcosystemRepo}}
**Ecosystem Projects (your current project is part of this system):**
| Repo | CLI | Purpose |
|------|-----|---------|
| orch-go | orch | Agent orchestration |
| beads | bd | Issue tracking |
| kb-cli | kb | Knowledge management |
| kn | kn | Quick knowledge capture |
| glass | glass | Browser automation |
| skillc | skillc | Skill compilation |

**For comprehensive documentation:** See `~/.orch/ECOSYSTEM.md`
{{end}}
```

---

## Answering Original Questions

### Q1: Tooling knowledge vs project registry - Different injection rules?

**YES.** 
- **Tooling (bd, kb, orch, kn):** Always inject. ~10 lines.
- **Project registry:** Only for ecosystem repos. ~20 lines.
- **Full ECOSYSTEM.md:** Reference only, never embed.

### Q2: Interaction with --workdir cross-project spawns?

When spawning in glass from orch-go:
- Check if TARGET project (glass) is in ecosystem repos
- If yes, include project registry section
- Always include core tooling section

### Q3: Format of ecosystem registry?

**Static in code** (OrchEcosystemRepos), not config file. Reasons:
- Rarely changes (new repos are infrequent)
- No additional dependencies
- Compile-time validation
- Easy to extend if needed

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**
- Clear problem (agents trying GitHub instead of local tools)
- Existing infrastructure (ECOSYSTEM.md, OrchEcosystemRepos)
- Straightforward solution (template injection)
- Minimal risk (additive change, doesn't break anything)

**What's certain:**
- ECOSYSTEM.md exists and is comprehensive
- Agents don't currently receive this context
- Tiered injection is the right approach (not all-or-nothing)

**What's uncertain:**
- Exact line count that's too much (30 vs 50 vs 100)
- Whether some non-ecosystem repos should get tooling context
- Whether this should be opt-in via flag or automatic

---

## Implementation Sequence

1. **Phase 1 (Minimal):** Add core tooling section to all spawns (~10 lines)
2. **Phase 2 (Full):** Add project registry for ecosystem repos (~25 lines)
3. **Phase 3 (Optional):** Add `--ecosystem` flag for forcing context on non-ecosystem projects

---

## References

**Files Examined:**
- `~/.orch/ECOSYSTEM.md` - Comprehensive ecosystem documentation
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator's ecosystem reference
- `pkg/spawn/kbcontext.go` - OrchEcosystemRepos definition
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template
- `pkg/spawn/config.go` - SpawnConfig structure

**Related Investigations:**
- `2025-12-25-inv-orchestrator-pre-spawn-context-gathering.md` - Context gathering scope

**Commands Run:**
```bash
cat ~/.orch/ECOSYSTEM.md | head -150
cat ~/.claude/skills/meta/orchestrator/SKILL.md | head -100
ls ~/Documents/personal/ | grep -E "^(orch|beads|glass|kb|kn)"
```

---

## Investigation History

**2025-12-28 ~11:00:** Investigation started
- Initial question from orch-go-tr0b: Scope ecosystem context injection

**2025-12-28 ~11:30:** Key findings complete
- Found ECOSYSTEM.md exists
- Identified tiered injection approach
- Proposed implementation

**2025-12-28 ~11:45:** Investigation completed
- Final confidence: High (85%)
- Recommendation: Tiered injection with core tooling always, project registry for ecosystem repos
