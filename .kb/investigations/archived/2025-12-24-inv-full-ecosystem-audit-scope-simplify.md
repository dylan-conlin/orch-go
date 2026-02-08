<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The orch ecosystem has 8 repos with significant overlap - orch-go (37k LoC) should be the primary, absorbing kb+kn functionality. skillc is well-integrated, beads is external risk, agentlog is orphaned value, Python orch-cli can be deprecated when orch-go reaches parity.

**Evidence:** orch-go has 334 skill references vs Python orch-cli's 27k LoC; kb (123 refs) and kn (108 refs) are used in parallel with significant conceptual overlap; agentlog has only 14 skill references suggesting low adoption.

**Knowledge:** The key architectural distinction is: beads = external dependency (external team ownership), everything else = internal. kb+kn merge is natural (kn entries promote to kb artifacts). skillc is essential build infrastructure.

**Next:** Create epic with prioritized consolidation: (1) kb absorbs kn, (2) orch-go reaches Python parity, (3) agentlog evaluated for archive vs integrate, (4) beads coupling reduced via interface abstraction.

**Confidence:** High (85%) - evidence from code metrics and skill references is solid; uncertainty in external beads roadmap.

---

# Investigation: Full Ecosystem Audit - Scope and Simplify

**Question:** How should the 8-repo orch ecosystem be consolidated for maintainability while preserving functionality?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Design Session Agent
**Phase:** Complete
**Next Step:** Create consolidation epic with beads issues
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Repo Size and Activity Analysis

**Evidence:**

| Repo | Language | LoC/Size | Last Activity | Primary Purpose |
|------|----------|----------|---------------|-----------------|
| orch-go | Go | 37,550 lines | Active (current) | Primary orchestration CLI |
| orch-cli | Python | 27,345 lines | Dec 17 | Legacy orchestration CLI |
| orch-knowledge | Mixed | ~2,600 lines CLAUDE.md | Active | Skills source + knowledge archive |
| skillc | Go | 3.9MB binary | Dec 23 | Skill compiler |
| kb-cli | Go | 4.4MB binary | Dec 23 | Long-form knowledge artifacts |
| kn | Go | 3.0MB binary | Dec 22 | Quick knowledge entries |
| beads | Go | 31MB binary (external) | Dec 21 | Issue tracking |
| agentlog | Go | 4.1MB binary | Dec 16 | Error logging for agents |

**Source:** File system analysis via `ls -la`, `wc -l`, binary sizes

**Significance:** orch-go is catching up to Python orch-cli. Two orchestration CLIs is unsustainable. beads is massive (external project with own trajectory).

---

### Finding 2: Tool Usage in Skills (Actual vs Declared)

**Evidence:**

| Tool | Skill References | Status |
|------|-----------------|--------|
| orch | 334 | Core, heavily used |
| bd (beads) | 307 | Core, heavily used |
| skillc | 153 | Essential build system |
| kb | 123 | Moderate use |
| kn | 108 | Moderate use |
| agentlog | 14 | Low adoption |

**Source:** `grep -r` across `~/.claude/skills/`

**Significance:** 
- orch + beads are the core workflow tools
- kb + kn together (231 refs) suggests merger value
- agentlog has failed to gain adoption despite being designed for agents

---

### Finding 3: Conceptual Overlap Between kb and kn

**Evidence:**

kb commands:
```
create, list, search, context, chronicle, reflect, promote, publish, link, migrate, guides
```

kn commands:
```
decide, tried, question, constrain, resolve, supersede, search, decisions, constraints, questions, attempts, recent, get, context
```

Overlapping concepts:
- Both have `context` command (unified knowledge query)
- Both have `search` capability
- `kn promote` → moves entries to kb artifacts
- `kn decide/constrain` → quick versions of `kb create decision`

**Source:** `kb --help`, `kn --help`

**Significance:** kn is essentially "quick kb entries" with a promotion path to full kb artifacts. Natural merge candidate.

---

### Finding 4: External vs Internal Ownership

**Evidence:**

| Repo | Ownership | Risk |
|------|-----------|------|
| beads | External (stevey) | HIGH - different trajectory, complex CLAUDE.md (28k chars!) |
| All others | Dylan | LOW - full control |

beads has:
- Own contributor guidelines
- Own release process
- Complex internal architecture (Gas Town, crew, polecats)
- 28k character CLAUDE.md (largest in ecosystem)

**Source:** beads/CLAUDE.md analysis, directory structure

**Significance:** beads is a dependency, not part of the ecosystem we control. Should treat it as an external interface with abstraction layer.

---

### Finding 5: Duplication Between Python and Go orch

**Evidence:**

Python orch-cli unique features (not yet in orch-go):
- `synthesis` - Synthesize recent activity
- `transcript` - Session transcript management
- `friction` - Analyze agent friction points
- `stale` - Show stale beads issues
- `doc-check/doc-gen` - Documentation management
- `lint` - CLAUDE.md size checking
- `logs` - Command logging viewer
- `history` - Agent history with analytics

Go orch-go unique features (not in Python):
- `swarm` - Batch spawn with concurrency control
- `port` - Port allocation management
- `servers` - Development server management
- `handoff` - Session handoff generation

**Source:** `orch --help` comparison between binaries

**Significance:** ~80% feature parity achieved. Go version has some unique features already. Migration path is clear.

---

### Finding 6: skillc Integration Pattern (Success Model)

**Evidence:**

skillc workflow:
```
orch-knowledge/skills/src/{category}/{skill}/.skillc/
    ↓ skillc deploy
~/.claude/skills/{category}/{skill}/SKILL.md
```

- skillc is standalone but tightly integrated
- Clear source/distribution split
- Self-describing output with checksums
- Used via `orch build skills` in Python, could be integrated

**Source:** skillc/README.md, orch-knowledge/CLAUDE.md

**Significance:** skillc demonstrates successful "build tool that stays separate" pattern. Could stay separate OR be absorbed into orch-go as subcommand.

---

### Finding 7: agentlog's Orphaned State

**Evidence:**

- Only 14 references in skills
- Last activity Dec 16 (8 days ago)
- No integration hooks with orch spawn
- Self-describes as "AI-first CLI for error visibility"
- No evidence of active use in spawn contexts

**Source:** Skill reference count, file timestamps, CLAUDE.md

**Significance:** agentlog was a good idea that hasn't been adopted. Either needs integration effort or should be archived.

---

## Synthesis

**Key Insights:**

1. **Two-tier consolidation** - The ecosystem naturally splits into:
   - **Core workflow:** orch-go + beads (issue tracking) + skillc (build)
   - **Knowledge management:** kb + kn (should merge)
   - **Legacy/Orphaned:** Python orch-cli (deprecate), agentlog (evaluate)

2. **Beads is special** - It's the only external dependency. Treating it as an internal repo will cause friction. Should wrap with abstraction layer in orch-go.

3. **Knowledge should be unified** - kb and kn serving slightly different UX for same conceptual purpose. `kn` is quick capture, `kb` is long-form. Merge under `kb` with subcommands or merge into orch-go.

4. **Go migration is nearly complete** - orch-go at 37k LoC has most features. Python deprecation is feasible in 1-2 months.

**Answer to Investigation Question:**

The ecosystem should consolidate from 8 repos to ~4 functional units:

1. **orch-go** (primary CLI) - absorbs Python orch-cli, potentially kb+kn
2. **orch-knowledge** (knowledge archive) - stays separate as content repo
3. **skillc** (build tool) - could merge into orch-go or stay separate
4. **beads** (external) - wrap with interface, reduce coupling

agentlog should be evaluated for archive vs active development.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong code metrics and usage data. Clear ownership boundaries. Uncertainty only in external beads roadmap and agentlog's future.

**What's certain:**

- ✅ Python orch-cli should be deprecated (Go version has caught up)
- ✅ kb and kn have significant conceptual overlap (merge beneficial)
- ✅ beads is external dependency requiring abstraction
- ✅ skillc is essential and well-integrated

**What's uncertain:**

- ⚠️ Whether agentlog should be archived or integrated
- ⚠️ beads API stability (external control)
- ⚠️ Timeline for Python deprecation (feature parity gaps)

**What would increase confidence to Very High:**

- User feedback on kb/kn usage patterns
- beads API stability commitment from stevey
- Test coverage analysis for orch-go

---

## Implementation Recommendations

### Recommended Approach ⭐

**Phased Consolidation over 6 months:**

**Why this approach:**
- Reduces maintenance burden progressively
- Maintains system stability during transition
- Allows user feedback at each phase

**Trade-offs accepted:**
- Slower than big-bang merge
- Temporary duplication during transition

**Implementation sequence:**

1. **Phase 1 (Month 1): kb absorbs kn** 
   - Add `kb quick` subcommand with kn semantics
   - Migrate .kn entries to .kb format
   - Deprecate kn binary

2. **Phase 2 (Month 2-3): orch-go reaches full parity**
   - Port remaining Python features
   - Add deprecation warnings to Python
   - Update skill references

3. **Phase 3 (Month 4): beads abstraction**
   - Create `pkg/beads/client.go` interface
   - Reduce direct `bd` CLI calls
   - Enable mock testing

4. **Phase 4 (Month 5-6): Final cleanup**
   - Archive Python orch-cli
   - Evaluate agentlog (archive or integrate)
   - Consider skillc integration

### Alternative Approaches Considered

**Option B: Merge everything into orch-go immediately**
- **Pros:** Single binary, simplest mental model
- **Cons:** Risky, large migration, beads integration complex
- **When to use instead:** If maintenance burden is critical

**Option C: Keep all repos, add shared library**
- **Pros:** Minimal change, preserve existing workflows
- **Cons:** Doesn't reduce complexity, maintenance burden continues
- **When to use instead:** If resources constrained

---

### Implementation Details

**What to implement first:**
- kb/kn merge (lowest risk, highest immediate value)
- Python feature parity analysis (unblocks deprecation)

**Things to watch out for:**
- ⚠️ beads API changes could break orch-go
- ⚠️ Skill references need updating after moves
- ⚠️ User workflows may depend on specific tool behaviors

**Success criteria:**
- ✅ Single knowledge CLI (kb) instead of two
- ✅ orch-go as only orchestration CLI
- ✅ All skill references point to valid tools
- ✅ beads interaction via abstraction layer

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md` - Project context
- `/Users/dylanconlin/orch-knowledge/CLAUDE.md` - Knowledge archive structure
- `/Users/dylanconlin/Documents/personal/*/CLAUDE.md` - All project contexts

**Commands Run:**
```bash
# Line counts
wc -l ~/Documents/personal/orch-go/**/*.go
wc -l ~/Documents/personal/orch-cli/src/orch/*.py

# Tool references in skills
grep -r "bd " ~/.claude/skills/ | wc -l
grep -r "kb " ~/.claude/skills/ | wc -l
grep -r "kn " ~/.claude/skills/ | wc -l

# Command comparison
orch --help
kb --help
kn --help
```

**Related Artifacts:**
- **Decision:** Pending - recommend creating consolidation decision
- **Investigation:** This document
- **Workspace:** `.orch/workspace/og-work-full-ecosystem-audit-24dec/`
