# Session Synthesis

**Agent:** og-feat-design-skill-execution-16jan-654b
**Issue:** orch-go-0vscq.6
**Duration:** 2026-01-16 (single session)
**Outcome:** success

---

## TLDR

Investigated how to cross-reference tool failures with SKILL.md guidance. Designed a 4-component Skill-Execution Bridge (parser, annotator, analyzer, reporter) to map tool failures to documentation gaps.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-16-inv-design-skill-execution-bridge-friction.md` - Investigation with 4 findings, synthesis, and implementation recommendations

### Files Modified
- None (investigation only, no implementation)

### Commits
- None yet (investigation artifact to be committed)

---

## Evidence (What Was Observed)

- Action logging exists in `pkg/action/action.go` but operates independently of skill context
- Skills are embedded in SPAWN_CONTEXT as full markdown content via `pkg/spawn/context.go` template
- OpenCode provides tool-invocation MessagePart events but tool results aren't captured for failure analysis
- Skills have parseable structure with markdown headers, phases, and tool mentions
- No existing code maps tool failures to skill guidance sections

### Key Code Locations
- Action logging: `/Users/dylanconlin/Documents/personal/orch-go/pkg/action/action.go:1-680`
- Skill loading: `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go`
- Spawn context: `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:238-249`
- OpenCode events: `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/types.go`

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-16-inv-design-skill-execution-bridge-friction.md` - Complete investigation with design

### Decisions Made
- **Architecture:** 4-component bridge design (parser, annotator, analyzer, reporter)
- **Approach:** Incremental implementation starting with skill parser, validate before building full system
- **Data flow:** Enrich action logs with skill/phase context at spawn time, analyze patterns against parsed skill structure
- **Output format:** Human-readable friction reports showing tool failure → skill guidance → gap analysis

### Constraints Discovered
- Phase detection relies on bd comment convention (not foolproof)
- Tool name normalization needed (case variations, aliases)
- Some failures are legitimate exploration (false positives need filtering)
- Privacy concern: action logs may contain sensitive paths/commands

### Design Components

**1. Skill Parser** (`pkg/skills/parser.go`)
- Extract markdown structure from SKILL.md
- Identify sections, phases, tool mentions
- Build queryable structure: FindSectionsForTool(tool, phase)

**2. Context Annotator** (extend `pkg/action`)
- Add Skill, Phase, BeadsID fields to ActionEvent
- Populate at spawn time from spawn command + SPAWN_CONTEXT

**3. Failure Analyzer** (extend `pkg/action`)
- Analyze patterns with skill context
- Classify friction type: missing-guidance, unclear-guidance, ignored-guidance
- Find relevant skill sections for failed tool + phase

**4. Friction Reporter** (`cmd/orch/friction_cmd.go`)
- CLI: `orch friction report [--session ID] [--skill name]`
- Generate markdown reports mapping failures to guidance
- Enable skill authors to identify documentation gaps

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with design)
- [x] Investigation file has `Phase: Complete`
- [x] Ready for orchestrator review

**Follow-up work:**
Create implementation issues for each component:
1. **Skill Parser** - Foundation for all other work
2. **Minimal Friction Report** - Validate approach with one known pattern
3. **Context Annotator** - Once value proven
4. **Full Failure Analyzer** - After real session testing

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to detect if guidance was "followed but insufficient" vs "not followed"?
- Should we track *successful* tool usage to validate guidance works?
- Can we use LLM to semantically analyze if tool usage matches guidance intent?
- Should friction analysis be integrated into `orch complete` as a quality gate?

**Areas worth exploring further:**
- Alternative phase detection mechanisms (workspace metadata, OpenCode events) if bd comments unreliable
- Skill structure variance across different skill types (meta vs worker, procedure vs framework)
- Privacy filtering strategy for action logs before export
- User study with skill authors: what format of friction reports is most actionable?

**What remains unclear:**
- False positive rate with simple heuristics (need testing)
- Whether skill authors will find this valuable (need user feedback)
- How to handle skill structure that doesn't follow conventions

---

## Session Metadata

**Skill:** feature-impl (design phase)
**Model:** claude-sonnet-3.5
**Workspace:** `.orch/workspace/og-feat-design-skill-execution-16jan-654b/`
**Investigation:** `.kb/investigations/2026-01-16-inv-design-skill-execution-bridge-friction.md`
**Beads:** `bd show orch-go-0vscq.6`
