# Session Handoff - 2026-01-07 (Markdown Issues Spike)

**Session:** Strategic discussion on tracking layer
**Focus:** Should we replace beads with markdown-based issues?

---

## What Happened

Resumed from prior session about the Strategic Orchestrator Model. Dylan raised friction with beads:
- "Issues aren't artifacts"
- "Opacity" 
- "Shoehorned"
- "Not our project"

Through discussion, we refined the problem:

**The tracking layer needs to be as legible and moldable as the knowledge layer.**

Currently:
- Knowledge layer (kb): legible, moldable, direct access, Dylan owns it
- Tracking layer (beads): opaque, indirect access, someone else's decisions

This matters because Dylan needs visibility into system state without going through the AI. He built 2 UIs (beads-ui-svelte, orch-go dashboard) to recover observability that markdown would provide natively.

---

## Key Insight

The real issue isn't "beads vs markdown" for agents. It's about **who has direct access to system state**.

| Audience | Beads | Markdown |
|----------|-------|----------|
| Agents (via CLI) | ✅ Works | ✅ Would work |
| Orchestrator (via CLI) | ✅ Works | ✅ Would work |
| Dylan (directly) | ❌ Needs UI | ✅ Native access |

---

## Evidence Gathered

### What beads provides (and whether it matters):

| Feature | Importance | Can replicate? |
|---------|-----------|----------------|
| Atomic operations | Low (agents work serially) | Yes |
| Dependency queries | Low (<1% usage) | Yes (JSONL scan) |
| Performance | Low at current scale | Yes (kb-cli is fine) |
| Schema enforcement | Medium | Yes (templates + validation) |
| Built CLI | Known cost | Yes (~1000 lines) |
| Sync mechanism | Low | Yes (same git workflow) |

### kb-cli patterns already solve the hard problems:
- JSONL for structured metadata (`.kb/quick/entries.jsonl`)
- Markdown for rich content (investigations, decisions)
- Templates with structure (D.E.K.N.)
- Cross-project search
- Proven at scale

### Beads DB corruption revealed tracking/knowledge disconnect:
- Dashboard epic showed "open" but was complete
- 3 synthesis issues showed "ready" but work was done Jan 6
- Issues were created *after* work completed (daemon auto-creation)
- No reconciliation between artifacts and tracking

---

## Decisions Made

None formally. This is still exploratory.

**Direction emerging:** Markdown-based issues using kb-cli patterns. The losses are acceptable, the cost is known (~2-3 days), the patterns are proven.

---

## Spike Created

Created `.issues/orch-go-akhff.md` as a test of markdown-based epic format.

**Irony:** The epic is about making agent state visible in the dashboard (tabs). We're testing whether markdown makes issue state visible to Dylan. Same problem, two levels.

**Outcome:** The epic was already complete but beads showed it as open. We verified code, closed it. The markdown file now shows full history directly.

---

## Issues Closed This Session

| Issue | Reason |
|-------|--------|
| orch-go-akhff | Epic complete - all tabs implemented, panel refactored |
| orch-go-8qg67 | Synthesis already done Jan 6 |
| orch-go-1lrzg | Synthesis already done Jan 6 |
| orch-go-5kjlw | Synthesis already done Jan 6 |

---

## Beads DB Rebuilt

The DB was corrupted (FK violations, `no such column: repro`). Rebuilt via:
```bash
rm .beads/beads.db && bd init
```

Imported 1528 issues. Some cleanup needed (`bd doctor --fix`).

---

## Open Questions

1. **Should we run a proper spike?** We created `.issues/orch-go-akhff.md` but the epic was already done. Need an actually-in-progress issue to test whether direct access changes how Dylan works.

2. **What format for markdown issues?** Draft exists in `.issues/orch-go-akhff.md`:
   - Understanding section (problem, constraints, risks, done criteria)
   - Children table (linked issues)
   - Execution log (key events with dates)
   - Evidence chain (linked artifacts)

3. **Build vs continue spiking?** Evidence is strong. Could proceed to build `issue` CLI using kb-cli patterns. Or continue spiking with real in-progress work.

---

## Files Changed

- `.issues/orch-go-akhff.md` - Created (spike markdown epic)
- `.beads/beads.db` - Rebuilt from JSONL
- `.beads/issues.jsonl` - Updated (closed 4 issues)

---

## Next Session

Options:

1. **Continue spike** - Pick an actually in-progress issue, maintain it as markdown for a week, observe whether Dylan uses it

2. **Build the thing** - Start `issue` CLI using kb-cli patterns. The evidence is probably sufficient.

3. **Different direction** - Dylan may have other priorities

**Recommended:** Ask Dylan. The analysis is done; this is now a prioritization decision.

---

## Resume Commands

```bash
# Check current state
orch status
bd ready

# See the spike file
cat .issues/orch-go-akhff.md

# See what needs cleanup
bd doctor
```
