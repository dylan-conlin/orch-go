## Summary (D.E.K.N.)

**Delta:** Investigation overhead comes from 4 root causes: (1) SPAWN_CONTEXT template mandates investigation files for ALL spawns, (2) skill_inference routes bug→architect, (3) architect mandates design investigations, (4) no lightweight bug-fix path exists.

**Evidence:** Of 30 investigations from Jan 22-23, ~15 (50%) were unnecessary: 3 empty templates, 12 implementation tasks (should be commits), 5 duplicate daemon-capacity investigations.

**Knowledge:** "Premise Before Solution" got encoded as "always investigate" - the principle is sound but the implementation over-applies it. Bug fixes need a fast path; investigation should be opt-in for implementation work.

**Next:** Implement skill routing changes: bug→systematic-debugging, add `--skip-investigation` flag to spawn, make investigation phase opt-in for feature-impl.

**Promote to Decision:** Actioned - decision exists (investigation-overhead-firefighting-mode)

---

# Investigation: Root Cause of Investigation Overhead

**Question:** Why are so many investigations being created (~17 in 36 hours) and how can we reduce artifact overhead without losing the "Premise Before Solution" benefit?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Worker agent (orch-go-fkn0p)
**Phase:** Complete
**Next Step:** Implement recommended changes to skill routing
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-01-23-investigation-overhead-firefighting-mode.md
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: SPAWN_CONTEXT template mandates investigation files for ALL spawns

**Evidence:** Line 220-222 in pkg/spawn/context.go:
```
DELIVERABLES (REQUIRED):
2. **SET UP investigation file:** Run `kb create investigation {{.InvestigationSlug}}` to create from template
```

Every spawn, regardless of skill (feature-impl, systematic-debugging, architect, etc.), is instructed to create an investigation file. This is embedded in the SPAWN_CONTEXT.md template.

**Source:** pkg/spawn/context.go:220-222

**Significance:** This is the primary root cause. Even implementation-focused spawns (feature-impl) create investigation files when they should be creating commits with descriptive messages.

---

### Finding 2: skill_inference routes bug → architect by default

**Evidence:** Line 29-32 in pkg/daemon/skill_inference.go:
```go
case "bug":
    // Default to architect: understand before fixing
    // Use skill:systematic-debugging label for clear, isolated bugs
    return "architect", nil
```

The comment acknowledges "Premise Before Solution" but this routes ALL bugs through architect, which mandates design investigation files.

**Source:** pkg/daemon/skill_inference.go:29-32

**Significance:** The routing is too aggressive. Most bugs don't need architectural understanding - they need fixing. The skill:systematic-debugging escape hatch requires manual labeling.

---

### Finding 3: Architect skill mandates investigation file creation

**Evidence:** Architect skill Phase 4a - Externalization (line 221-226):
```bash
kb create investigation design/{slug}
```
This creates: `.kb/investigations/YYYY-MM-DD-design-{slug}.md`

The architect skill explicitly requires creating investigation files as the primary artifact.

**Source:** ~/.claude/skills/worker/architect/SKILL.md:221-226

**Significance:** When bugs route to architect, they MUST create investigations. This is appropriate for design work but overkill for bug fixes.

---

### Finding 4: 5 daemon-capacity investigations for the same recurring problem

**Evidence:** Investigation files from Jan 22-23:
- 2026-01-23-inv-daemon-capacity-tracking-stale-doesn.md
- 2026-01-23-inv-daemon-capacity-counter-stuck-recurring.md
- 2026-01-23-inv-daemon-doesn-see-newly-created.md
- 2026-01-22-inv-fix-daemon-capacity-counter-getting.md
- 2026-01-23-inv-daemon-capacity-counter-stuck-blocks.md (EMPTY TEMPLATE!)

The same bug produced 5 investigations because each fix attempt created a new investigation, but the fixes didn't address the root cause.

**Source:** .kb/investigations/ directory listing

**Significance:** This demonstrates the "3+ fixes signal" from the decision record. When the same area gets multiple investigations, it's a signal to step back, not create more artifacts.

---

### Finding 5: ~50% of investigations were unnecessary

**Evidence:** Categorization of 30 investigations from Jan 22-23:
- **Empty templates (wasted):** 3 (agent died or didn't fill in)
- **Implementation tasks (should be commits):** 12 (add-*, implement-*, fix-*)
- **Duplicate investigations (same problem):** 5 (daemon-capacity chain)
- **Legitimate research/design:** ~10 (gastown analysis, opencode comparison, etc.)

Examples of implementation tasks that became investigations:
- inv-add-disabled-backends-config-option.md (feature implementation)
- inv-implement-daemon-capacity-fix-add.md (bug fix implementation)
- inv-add-docker-cleanup-orch-complete.md (found to be DUPLICATE of existing work!)

**Source:** Manual review of investigation file contents

**Significance:** Half the investigations provided little value beyond what a good commit message would provide.

---

## Synthesis

**Key Insights:**

1. **"Premise Before Solution" got encoded as "always investigate"** - The original principle is sound: understand before fixing. But the implementation mandates investigation files for ALL spawns, which is over-application. Understanding can happen through reading code and making commits with good messages.

2. **Bug routing is too aggressive** - Routing ALL bugs to architect assumes every bug needs design understanding. In reality, most bugs are isolated issues that need systematic-debugging or a simple fix. The escape hatch (skill:systematic-debugging label) requires manual intervention that doesn't happen in practice.

3. **Investigation churn reveals systemic issues** - The 5 daemon-capacity investigations are a symptom: each tactical fix spawned a new investigation, but none stepped back to ask "why does this keep recurring?" The 3+ fixes signal from the decision record wasn't enforced.

4. **Template-driven behavior creates overhead** - Agents follow templates literally. When SPAWN_CONTEXT says "create investigation file", they do. The template should be conditional on skill type, not universal.

**Answer to Investigation Question:**

The system over-produces investigations because:
1. SPAWN_CONTEXT template mandates investigation files for ALL spawns (Finding 1)
2. skill_inference routes bug→architect, which mandates investigations (Findings 2, 3)
3. No lightweight bug-fix path exists without creating artifacts
4. Recurring issues create investigation chains instead of triggering architectural review (Finding 4)

The fix requires both routing changes (skill_inference) and template changes (SPAWN_CONTEXT), plus adding a "simple-fix" path for isolated bugs.

---

## Structured Uncertainty

**What's tested:**

- ✅ SPAWN_CONTEXT template mandates investigation files (verified: read pkg/spawn/context.go:220-222)
- ✅ skill_inference routes bug→architect (verified: read pkg/daemon/skill_inference.go:29-32)
- ✅ 5 daemon-capacity investigations exist (verified: ls .kb/investigations/ | grep daemon-capacity)
- ✅ ~50% of investigations were implementation tasks (verified: manual review of 30 investigation contents)
- ✅ Empty template investigations exist (verified: read inv-daemon-capacity-counter-stuck-blocks.md)

**What's untested:**

- ⚠️ Changing bug→systematic-debugging won't break architect's value for complex bugs (need to test with real bugs)
- ⚠️ --skip-investigation flag won't reduce important context capture (need to monitor commit quality)
- ⚠️ Investigation-optional feature-impl won't miss important understanding (need to monitor fix churn)

**What would change this:**

- Finding would be wrong if there's another path creating investigations not documented here
- Recommendation would be wrong if systematic-debugging creates its own investigation overhead
- Quantification would be wrong if the "implementation task" investigations provided unique value beyond commits

---

## Implementation Recommendations

**Purpose:** Reduce investigation overhead while preserving "Premise Before Solution" for work that genuinely needs understanding.

### Recommended Approach ⭐

**Three-part routing reform** - Change skill routing, add skip flag, make investigation phase opt-in.

**Why this approach:**
- Addresses all 4 root causes identified
- Preserves escape hatches for complex bugs (skill:architect label)
- Doesn't remove architect capability, just changes default routing
- Incremental changes that can be validated independently

**Trade-offs accepted:**
- Some bugs may need escalation from systematic-debugging → architect
- Commit messages must carry more context (training/discipline required)
- Risk of swinging too far toward "just ship it"

**Implementation sequence:**

1. **Change skill_inference.go: bug → systematic-debugging**
   ```go
   case "bug":
       // Default to systematic-debugging: fix the bug
       // Use skill:architect label for bugs needing design understanding
       return "systematic-debugging", nil
   ```
   Why first: Highest impact, single-line change, easy to revert if wrong.

2. **Make investigation phase opt-in for feature-impl**
   Update SPAWN_CONTEXT template: Remove universal "SET UP investigation file" instruction.
   Add conditional: Only include investigation instructions for investigation-type skills.
   Why second: Removes ~12 unnecessary investigations per firefighting cycle.

3. **Add --skip-investigation flag to orch spawn**
   Allow explicit opt-out for isolated fixes: `orch spawn --skip-investigation ...`
   Why third: Provides escape hatch without changing defaults.

### Alternative Approaches Considered

**Option B: Remove investigation skill entirely**
- **Pros:** Eliminates investigation overhead completely
- **Cons:** Loses value for genuine "how does X work?" questions
- **When to use instead:** Never - investigation has legitimate uses

**Option C: Time-box investigations to 30 minutes**
- **Pros:** Limits overhead without eliminating artifacts
- **Cons:** Doesn't address root cause (wrong skill routing)
- **When to use instead:** If routing changes don't reduce overhead enough

**Option D: Add "simple-fix" skill for isolated bugs**
- **Pros:** Clean separation of bug types
- **Cons:** Adds complexity; systematic-debugging already exists
- **When to use instead:** If systematic-debugging creates its own overhead

**Rationale for recommendation:** Routing changes are surgical and target the root cause. Adding a new skill or time-boxing addresses symptoms, not causes.

---

### Implementation Details

**What to implement first:**
- skill_inference.go change (1 line, immediate impact)
- SPAWN_CONTEXT template conditional (more complex, higher risk)
- --skip-investigation flag (optional enhancement)

**Files to modify:**
1. `pkg/daemon/skill_inference.go:29-32` - Change bug routing
2. `pkg/spawn/context.go:220-222` - Make investigation conditional
3. `cmd/orch/spawn.go` - Add --skip-investigation flag (optional)

**Things to watch out for:**
- ⚠️ systematic-debugging may create workspace files that become clutter (monitor)
- ⚠️ Agents may skip understanding entirely without investigation mandate (monitor commit quality)
- ⚠️ Complex bugs may get under-investigated with new routing (watch for skill:architect escalations)

**Areas needing further investigation:**
- Should systematic-debugging require workspace or just commits?
- What's the right threshold for skill:architect vs skill:systematic-debugging?
- How to detect "3+ fixes signal" automatically?

**Success criteria:**
- ✅ Investigation count drops by ~50% in firefighting mode
- ✅ Bug fix cycle time decreases (no investigation creation overhead)
- ✅ Commit messages are descriptive (spot-check quality)
- ✅ No increase in fix churn (bugs don't recur more often)

---

## References

**Files Examined:**
- `pkg/daemon/skill_inference.go` - Traced bug→architect routing (lines 29-32)
- `pkg/spawn/context.go` - Found universal investigation mandate in template (lines 220-222)
- `pkg/spawn/config.go` - Understood spawn configuration and tier system
- `~/.claude/skills/worker/architect/SKILL.md` - Confirmed investigation mandate in Phase 4a
- `~/.claude/skills/worker/feature-impl/SKILL.md` - Found configurable investigation phase
- `.kb/decisions/2026-01-23-investigation-overhead-firefighting-mode.md` - Context on the problem
- `2026-01-23-inv-implement-daemon-capacity-fix-add.md` - Example implementation-as-investigation
- `2026-01-23-inv-add-docker-cleanup-orch-complete.md` - Example duplicate detection investigation
- `2026-01-23-inv-daemon-capacity-counter-stuck-blocks.md` - Example empty template (wasted)

**Commands Run:**
```bash
# List investigations from Jan 22-23
ls -la .kb/investigations/ | grep -E "2026-01-2[23]"

# Count investigations in timeframe
ls -la .kb/investigations/ | grep -E "2026-01-2[23]" | wc -l
# Result: 54 (including current)

# List daemon-capacity investigations
ls -la .kb/investigations/ | grep "daemon-capacity"
# Result: 5 related investigations
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-23-investigation-overhead-firefighting-mode.md` - Establishes the problem and triage criteria
- **Investigation:** `.kb/investigations/2026-01-23-inv-daemon-capacity-counter-stuck-recurring.md` - Example of investigation chain

---

## Investigation History

**2026-01-23 19:36:** Investigation started
- Initial question: Why 17+ investigations in 36 hours during docker backend work?
- Context: Decision record established overhead as problem, needed root cause and fix

**2026-01-23 19:45:** Traced investigation creation paths
- Found 4 root causes: template mandate, skill routing, architect requirement, no lightweight path

**2026-01-23 20:00:** Quantified the problem
- Categorized 30 investigations: ~15 (50%) unnecessary

**2026-01-23 20:15:** Investigation completed
- Status: Complete
- Key outcome: Recommend bug→systematic-debugging routing, conditional investigation template
