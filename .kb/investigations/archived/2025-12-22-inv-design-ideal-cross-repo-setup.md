## Summary (D.E.K.N.)

**Delta:** Dylan's orchestration ecosystem should stay decentralized (per-repo beads, cross-repo kb search) with a new global `~/.orch/ECOSYSTEM.md` as the discoverable map.

**Evidence:** Tested kb context --global (works across 17 registered projects), examined 8 repos' .beads/.kb structure, found prior investigation confirming beads is fundamentally per-repo.

**Knowledge:** The existing architecture is mostly correct - kb already supports cross-repo search, beads per-repo isolation matches Yegge's design philosophy, skills already use category/skill naming convention.

**Next:** Create ~/.orch/ECOSYSTEM.md, document naming conventions in global CLAUDE.md, add cross-repo epic pattern to orchestrator skill.

**Confidence:** High (85%) - Most recommendations are incremental improvements, not architectural changes.

---

# Investigation: Ideal Cross-Repo Setup for Dylan's Orchestration Ecosystem

**Question:** How should Dylan's 8+ repos (orch-go, kb-cli, beads, beads-ui-svelte, skillc, agentlog, kn, orch-cli) be organized for cross-repo orchestration?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Architect agent (spawned from orch-go-d08v)
**Phase:** Complete
**Next Step:** None - ready for orchestrator review
**Status:** Complete
**Confidence:** High (85%)

---

## Design Questions Addressed

1. Should repos share one beads backlog or stay separate?
2. Should kb context search across repos?
3. How do cross-repo epics work?
4. Should there be a meta-orchestration repo?
5. How should skills/hooks/CLAUDE.md know about cross-repo config?
6. What's the naming convention for 'skills/hooks/claude artifacts' category?

---

## Findings

### Finding 1: Beads is designed to be per-repo (and should stay that way)

**Evidence:** 
- Each repo has its own `.beads/` directory with isolated SQLite database
- Prior investigation (2025-12-21-inv-cross-project-epic-orchestration-patterns.md) confirmed:
  - `bd show orch-go-ivtg` works in orch-go, fails in kb-cli
  - `bd repo` commands for multi-repo hydration are buggy (JSON parsing error)
  - Multi-repo hydration is read-only aggregation, not cross-repo relationships
- Decision 2025-12-21-beads-oss-relationship-clean-slate.md established: use upstream beads as-is

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-cross-project-epic-orchestration-patterns.md`
- Direct testing: `cd ~/Documents/personal/kb-cli && bd show orch-go-ivtg` → "no issue found"

**Significance:** Don't fight beads' design. Per-repo isolation is intentional. Cross-repo coordination should happen at the orchestration layer (orch), not the issue tracking layer (beads).

---

### Finding 2: kb already supports cross-repo search via --global flag

**Evidence:**
```bash
$ kb context "orchestration" --global
Context for "orchestration":

## CONSTRAINTS (from kn)
- [orch-knowledge] Orchestrators NEVER do spawnable work...

## DECISIONS (from kn)
- [orch-knowledge] bd work delegation pattern is compatible...
- [orch-go] Template ownership split by domain...

## DECISIONS (from kb)
- [orch-knowledge] Feature Coordination Skill Creation
  Path: /Users/dylanconlin/orch-knowledge/.kb/decisions/...
```

17 projects are registered with kb:
- kb-cli, orch-knowledge, beads, agentlog, orch-cli, beads-ui-svelte, skillc, kn
- Plus: dotfiles, snap, opencode, blog, and 5 work projects

**Source:** `kb projects list`, `kb context --global`

**Significance:** The cross-repo knowledge search capability already exists. It just needs better documentation and awareness in orchestration contexts.

---

### Finding 3: Skills use a consistent category/skill naming convention

**Evidence:**
```
~/.claude/skills/
├── meta/          # Skills about skills: analyze-skill-usage, audit-claude-md, writing-skills
├── policy/        # Always-loaded guidance: orchestrator
├── shared/        # Used by multiple contexts: code-review, issue-quality, session-transition
├── utilities/     # Reference docs: testing-anti-patterns, tmux-workspace-sync
└── worker/        # Spawnable task skills: investigation, feature-impl, architect, etc.

Symlinks provide flat access: architect -> worker/architect
```

**Source:** `ls -la ~/.claude/skills/`

**Significance:** The category/skill pattern is already established and working. Document it formally in global CLAUDE.md.

---

### Finding 4: No central ecosystem documentation exists

**Evidence:**
- Each repo has its own CLAUDE.md (good for project-specific context)
- Global CLAUDE.md at `~/.claude/CLAUDE.md` exists but only has knowledge placement guidance
- No artifact documents the relationships between repos
- New agents must discover the ecosystem through exploration

**Source:** 
- `~/.claude/CLAUDE.md` (52 lines, only knowledge placement)
- Individual repo CLAUDE.md files vary in quality and detail

**Significance:** Session amnesia means every new Claude instance has to rediscover the ecosystem. A single discoverable artifact would dramatically reduce onboarding friction.

---

### Finding 5: Cross-repo epics have a documented (if manual) pattern

**Evidence:**
From prior investigation:
```
For cross-project epics like orch-go-ivtg:
1. Epic lives in primary repo (orch-go)
2. For cross-repo children:
   - cd ~/Documents/personal/kb-cli
   - orch spawn feature-impl "Phase 2: kb reflect MVP" --no-track
   - After completion: cd ~/Documents/personal/orch-go && bd close orch-go-ivtg.2 --reason "..."
3. Epic tracks completion via child issue status in primary repo
```

**Source:** `.kb/investigations/2025-12-21-inv-cross-project-epic-orchestration-patterns.md`

**Significance:** The pattern exists but requires orchestrator discipline. Adding it to the orchestrator skill would make it discoverable.

---

### Finding 6: Template ownership is already split by domain

**Evidence:**
From kn entry:
> Template ownership split by domain: kb-cli owns knowledge artifacts (investigation, decision, guide, research); orch-go owns orchestration artifacts (SYNTHESIS, SPAWN_CONTEXT, FAILURE_REPORT, SESSION_HANDOFF)

This is already captured as a kn decision and working in practice.

**Source:** `kb context "template"` shows the kn entry

**Significance:** The architectural boundary is correct. No change needed.

---

## Synthesis

**Key Insights:**

1. **The architecture is mostly correct** - Per-repo beads, cross-repo kb search, category-based skills. The main gap is documentation and discoverability.

2. **Cross-repo coordination is an orchestrator responsibility** - Beads tracks per-repo work, kb provides cross-repo knowledge, orch coordinates. Don't push cross-repo logic into tools not designed for it.

3. **Ecosystem documentation would solve the discovery problem** - A single `~/.orch/ECOSYSTEM.md` file would let any agent understand the system quickly.

**Answer to Design Questions:**

| Question | Answer | Reasoning |
|----------|--------|-----------|
| **1. Shared beads backlog?** | **No - stay separate** | Per-repo is beads' design philosophy. Cross-repo coordination belongs in orch. |
| **2. Cross-repo kb search?** | **Yes - already exists** | `kb context --global` works across 17 registered projects. Just document it better. |
| **3. Cross-repo epics?** | **Ad-hoc spawns + manual close** | Pattern documented in prior investigation. Add to orchestrator skill. |
| **4. Meta-orchestration repo?** | **No - use ~/.orch/** | A new repo adds complexity without solving problems. Use global ~/.orch/ instead. |
| **5. Cross-repo config awareness?** | **Create ~/.orch/ECOSYSTEM.md** | Single discoverable artifact documents all repos, purposes, relationships. |
| **6. Naming convention?** | **{category}/{skill-name}** | Already established: meta/, policy/, shared/, utilities/, worker/. Document formally. |

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Most recommendations are incremental improvements on existing patterns, not architectural changes. The ecosystem already works - it just needs better documentation.

**What's certain:**

- ✅ Beads is per-repo by design (tested, documented in prior investigation)
- ✅ kb --global works across registered projects (tested directly)
- ✅ Skills use category/skill naming convention (observed in ~/.claude/skills/)
- ✅ Template ownership is split by domain (documented in kn)

**What's uncertain:**

- ⚠️ Whether ~/.orch/ECOSYSTEM.md is the right location vs ~/.claude/ECOSYSTEM.md
- ⚠️ How often agents actually use kb context --global (no usage data)
- ⚠️ Whether the manual cross-repo epic pattern will scale

**What would increase confidence to Very High:**

- Validate ECOSYSTEM.md location with Dylan
- Track kb --global usage to confirm agents discover it
- Try the cross-repo epic pattern on 2-3 more epics

---

## Implementation Recommendations

### Recommended Approach ⭐

**Incremental Documentation** - Create ecosystem map, document naming conventions, enhance orchestrator skill with cross-repo patterns.

**Why this approach:**
- Low risk - doesn't change working architecture
- High value - solves the discovery problem for new agents
- Builds on existing patterns - doesn't introduce new concepts

**Trade-offs accepted:**
- Cross-repo epics remain manual (vs automated beads enhancement)
- No unified cross-repo view of all work (vs hypothetical dashboard)

**Implementation sequence:**
1. Create `~/.orch/ECOSYSTEM.md` (ecosystem map - most impactful)
2. Update `~/.claude/CLAUDE.md` with naming conventions
3. Add cross-repo epic pattern to orchestrator skill
4. Document `kb context --global` in pre-spawn context

### Alternative Approaches Considered

**Option B: Unified Beads Database**
- **Pros:** Single view of all work across repos
- **Cons:** Fights beads' design, bd repo commands are buggy, maintenance burden
- **When to use instead:** If beads adds native multi-repo support upstream

**Option C: Meta-Orchestration Repo**
- **Pros:** Dedicated home for cross-repo artifacts
- **Cons:** Another repo to manage, doesn't solve cross-repo issue tracking
- **When to use instead:** Never - ~/.orch/ serves this purpose

**Rationale for recommendation:** The ecosystem mostly works. The problems are documentation and discoverability, not architecture. Fix with documentation.

---

### Implementation Details

**What to implement first:**
- Create `~/.orch/ECOSYSTEM.md` with full ecosystem map
- This unblocks all future cross-repo work

**Things to watch out for:**
- ⚠️ Keep ECOSYSTEM.md up to date when repos change (add to session-transition checklist?)
- ⚠️ Ensure kb projects registry stays in sync with ECOSYSTEM.md
- ⚠️ Document the manual nature of cross-repo epic coordination

**Areas needing further investigation:**
- Whether beads will add cross-repo support upstream
- How to automate ECOSYSTEM.md updates
- Whether a dashboard showing all repos' work would help

**Success criteria:**
- ✅ New agent sessions can discover ecosystem in <1 minute
- ✅ Cross-repo epics follow documented pattern
- ✅ `kb context --global` is used in pre-spawn contexts

---

## Deliverable: ECOSYSTEM.md Location

**Recommended:** `~/.orch/ECOSYSTEM.md`

**Reasoning:**
- `~/.orch/` is already the home for global orchestration state (accounts.yaml, focus.json, events.jsonl)
- Keeps ecosystem docs separate from Claude-specific context (~/.claude/)
- Natural discovery point for orchestration-related questions

**Alternative:** `~/.claude/ECOSYSTEM.md`
- Closer to where agents look for context
- But mixes ecosystem docs with Claude config

**Recommendation:** Use `~/.orch/ECOSYSTEM.md` and add a pointer from orchestrator skill.

---

## References

**Files Examined:**
- `~/.claude/CLAUDE.md` - Global Claude context (52 lines)
- `~/.claude/skills/` - Skill organization structure
- Individual repo CLAUDE.md files (orch-go, kb-cli, skillc)
- `.kb/investigations/2025-12-21-inv-cross-project-epic-orchestration-patterns.md`
- `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md`

**Commands Run:**
```bash
# Test cross-repo kb search
kb context "orchestration" --global

# List registered projects
kb projects list

# Test beads isolation
cd ~/Documents/personal/kb-cli && bd show orch-go-ivtg
# Result: Error - no issue found

# Explore skills structure
ls -la ~/.claude/skills/
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-cross-project-epic-orchestration-patterns.md` - Cross-repo epic patterns
- **Decision:** `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md` - Beads OSS relationship

---

## Investigation History

**2025-12-22 13:46:** Investigation started
- Initial question: How should 8+ repos be organized for cross-repo orchestration?
- Context: Orchestrator needs guidance on cross-repo patterns

**2025-12-22 14:00:** Key discovery - kb --global already works
- `kb context --global` searches 17 registered projects
- Cross-repo knowledge search is solved

**2025-12-22 14:15:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Ecosystem is mostly correct; main gap is documentation (ECOSYSTEM.md)
