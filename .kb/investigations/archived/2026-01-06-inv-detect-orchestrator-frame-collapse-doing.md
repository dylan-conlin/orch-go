<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Frame collapse detection requires multiple signals working together - no single heuristic is reliable alone.

**Evidence:** Analyzed orchestrator skill, OpenCode plugin infrastructure, and session handoff patterns; identified 5 detection approaches with different tradeoffs.

**Knowledge:** Orchestrators can't self-detect frame collapse (blind to their own blindness); detection must be external via meta-orchestrator review, session reflection prompts, or OpenCode plugins.

**Next:** Implement hybrid detection: (1) Add SESSION_HANDOFF.md section check to orch complete, (2) Add skill guidance, (3) Consider OpenCode plugin for file edit tracking.

---

# Investigation: Detect Orchestrator Frame Collapse

**Question:** How can we detect when an orchestrator drops into worker-level work (frame collapse) instead of delegating?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Agent (spawned investigation)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current Skill Has Self-Check but No External Detection

**Evidence:** The orchestrator skill at `~/.claude/skills/meta/orchestrator/SKILL.md` contains:
- "⛔ ABSOLUTE DELEGATION RULE" section (lines 573-680)
- "Orchestrator self-check: See ⛔ ABSOLUTE DELEGATION RULE - if doing ANY task work..." (line 1429)
- Multiple mentions of "STOP and spawn" as the response to detected frame collapse

However, these all rely on the orchestrator to notice their own behavior. The skill even acknowledges this: "Orchestrators can't see their own frame collapse (that's why meta-orchestrator exists)" (from spawn context).

**Source:** `~/.claude/skills/meta/orchestrator/SKILL.md:573-680, 1429`

**Significance:** Self-detection is unreliable because the agent doing worker work is in a cognitive state where they've already rationalized it. External detection is required.

---

### Finding 2: OpenCode Plugin Infrastructure Exists for External Detection

**Evidence:** Global OpenCode plugins directory contains:
```
~/.config/opencode/plugin/
├── action-log.ts → (broken symlink)
├── agentlog-inject.ts
├── bd-close-gate.ts
├── orchestrator-session.ts → /Users/dylanconlin/Documents/personal/orch-go/plugins/orchestrator-session.ts
└── usage-warning.ts
```

The `orchestrator-session.ts` plugin demonstrates:
- Worker vs orchestrator detection via `isWorker()` function
- Event hooks for `session.created` events
- Config hooks for instruction injection
- `tool.execute.before` and `tool.execute.after` events available (per skill docs)

**Source:** `~/.config/opencode/plugin/`, `/Users/dylanconlin/Documents/personal/orch-go/plugins/orchestrator-session.ts`

**Significance:** The plugin infrastructure is production-ready for implementing frame collapse detection. Could track file edits, time spent on code files, or cumulative changes.

---

### Finding 3: SESSION_HANDOFF.md Structure Can Surface Frame Collapse Patterns

**Evidence:** Looking at example handoffs:
- Good pattern (meta-orch-resume-last-meta-06jan-1287): Lists "Orchestrators spawned: 0", "Workers completed: 2"
- Problem case (from spawn context): "Manual fixes by orchestrator" as a section

The SESSION_HANDOFF.md template (from ~/.orch/session/2026-01-06/SESSION_HANDOFF.md) has structured sections but doesn't explicitly ask about frame collapse signals.

**Source:** `.orch/workspace-archive/meta-orch-resume-last-meta-06jan-1287/SESSION_HANDOFF.md`, `~/.orch/session/2026-01-06/SESSION_HANDOFF.md`

**Significance:** Handoff structure could include a "Frame Collapse Check" section that prompts explicit reflection. The meta-orchestrator reviewing handoffs could also flag patterns like "Manual fixes" sections.

---

### Finding 4: File Edit Heuristics Are Observable

**Evidence:** From the problem description:
- "If orchestrator session has >N lines of code changes, flag it"
- Price-watch orchestrator "spent hours manually fixing CSS bugs"

Observable signals:
1. `git diff --stat` at session end shows cumulative line changes
2. File types touched (`.css`, `.go`, `.ts` = code work, not orchestration)
3. Duration on non-orchestration files (workspace files vs code files)

**Source:** Spawn context problem description

**Significance:** Quantitative heuristics can complement qualitative checks. A plugin could track Edit tool usage on code files vs orchestration artifacts.

---

### Finding 5: Escalation Prompt Pattern Already Documented

**Evidence:** The skill mentions after 3 failed agent spawns, but doesn't implement this pattern. From spawn context:
- "Explicit escalation prompt: After 3 failed agent spawns for same issue, prompt escalation"

This suggests frame collapse often occurs after agent failures - the orchestrator tries to "just fix it" themselves.

**Source:** Spawn context, possible solutions section

**Significance:** The failure-to-escalation pipeline is a key moment. Detection could focus on post-spawn-failure behavior.

---

## Synthesis

**Key Insights:**

1. **Multi-layer detection is required** - No single signal is reliable. Need: (a) Skill guidance for self-check, (b) OpenCode plugin for quantitative tracking, (c) SESSION_HANDOFF.md analysis for post-hoc review, (d) Meta-orchestrator pattern recognition.

2. **Detection must happen at boundaries** - Key detection points are: session end (reflection), handoff review (meta-orchestrator), and real-time (plugin during session). Real-time is most valuable but hardest.

3. **The failure-to-implementation pattern is the key trigger** - Frame collapse typically happens after agents fail. The orchestrator thinks "I'll just fix it myself" instead of trying a different spawn strategy.

**Answer to Investigation Question:**

Frame collapse can be detected through multiple complementary approaches:

1. **Skill Guidance (self-detection)**: Already exists but unreliable alone. Add explicit time checks: "If you've been editing code for >15 minutes, you've frame collapsed."

2. **SESSION_HANDOFF.md Analysis**: Add a required section "Frame Collapse Self-Check" that asks:
   - Did you edit any code files directly?
   - Did you spend >30 minutes on any single file?
   - Did you manually fix something an agent failed at?

3. **OpenCode Plugin (real-time)**: Track `Edit` tool usage. If edits target code files (`.go`, `.ts`, `.css`, `.py`) rather than orchestration artifacts (`.md` in `.orch/`, `.kb/`), flag after threshold (e.g., 5+ edits, >50 lines changed).

4. **Meta-Orchestrator Review**: When reviewing handoffs, explicitly look for:
   - "Manual fixes" sections
   - Zero agent spawns with non-trivial work done
   - Extended duration without completion reports

5. **Escalation Prompts**: After 3 failed spawns for same issue, inject prompt: "Consider: Try different spawn strategy (--mcp playwright, different skill, explicit steps) before attempting manually."

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode plugin infrastructure works for orchestrator detection (verified: orchestrator-session.ts runs and injects skill)
- ✅ SESSION_HANDOFF.md template exists and is used (verified: found in ~/.orch/session/)
- ✅ Skill has ABSOLUTE DELEGATION RULE documented (verified: read SKILL.md lines 573-680)

**What's untested:**

- ⚠️ Plugin `tool.execute.after` can reliably track Edit tool usage (not implemented)
- ⚠️ Threshold of 5+ code edits is the right heuristic (arbitrary number)
- ⚠️ Meta-orchestrator actually reviews handoffs systematically (process assumption)

**What would change this:**

- Finding would be wrong if plugin event hooks don't fire for Edit tool
- Finding would be wrong if frame collapse happens gradually (no clear boundary)
- Finding would be wrong if orchestrators are already good at self-detection (no evidence of widespread problem)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Hybrid Detection with SESSION_HANDOFF.md Check at orch complete** - Add frame collapse detection as part of the `orch complete` flow for orchestrator sessions.

**Why this approach:**
- Catches frame collapse at session end, when reflection is natural
- Doesn't require new infrastructure (uses existing handoff)
- Forces explicit acknowledgment ("Yes I frame collapsed" or confirmation of delegation)

**Trade-offs accepted:**
- Post-hoc detection (damage already done)
- Relies on honest self-reporting

**Implementation sequence:**
1. Add "Frame Collapse Check" section to SESSION_HANDOFF.md template
2. Update `orch complete` to warn if orchestrator tier session has code file changes in git diff
3. Add skill guidance with explicit time threshold

### Alternative Approaches Considered

**Option B: Real-time OpenCode Plugin**
- **Pros:** Catches frame collapse as it happens, can interrupt
- **Cons:** Complex to implement, may be annoying if false positives
- **When to use instead:** If post-hoc detection proves insufficient

**Option C: Pure Skill Guidance**
- **Pros:** Zero code changes, immediate deployment
- **Cons:** Self-detection is unreliable (Finding 1)
- **When to use instead:** As a first quick win while building other approaches

**Rationale for recommendation:** SESSION_HANDOFF.md approach is low-cost to implement, fits existing workflow, and creates pressure for honest reflection without being intrusive during work.

---

### Implementation Details

**What to implement first:**
1. Add skill guidance with explicit 15-minute time check
2. Add "Frame Collapse Check" section to SESSION_HANDOFF.md template
3. Update `orch complete` to check for code file changes in orchestrator sessions

**Things to watch out for:**
- ⚠️ Don't block legitimate orchestrator code edits (CLAUDE.md, skill files)
- ⚠️ False positives may cause detection fatigue
- ⚠️ Need to distinguish orchestrator workspace from worker workspace

**Areas needing further investigation:**
- What file extensions should trigger detection? (code vs orchestration artifacts)
- Should there be a threshold for line count changes?
- How should the plugin handle gradual frame collapse (multiple small edits)?

**Success criteria:**
- ✅ Next frame collapse incident is detected either during session or at handoff
- ✅ `orch complete` warns if orchestrator session has code changes
- ✅ Meta-orchestrator knows to check for "Frame Collapse Check" section

---

## References

**Files Examined:**
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Analyzed ABSOLUTE DELEGATION RULE and self-check guidance
- `~/.config/opencode/plugin/` - Reviewed plugin infrastructure
- `/Users/dylanconlin/Documents/personal/orch-go/plugins/orchestrator-session.ts` - Studied worker detection pattern
- `~/.orch/session/2026-01-06/SESSION_HANDOFF.md` - Reviewed handoff template structure
- `.orch/workspace-archive/meta-orch-resume-last-meta-06jan-1287/SESSION_HANDOFF.md` - Example of good handoff

**Commands Run:**
```bash
# Found plugin directory structure
ls ~/.config/opencode/plugin/

# Found session handoff files
find ~/.orch -name "SESSION_HANDOFF.md"

# Searched for frame collapse patterns in skill
grep -n "ABSOLUTE DELEGATION RULE\|frame.*collapse" ~/.claude/skills/meta/orchestrator/SKILL.md
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-11-08-orchestrator-implementation-boundary-final.md` - Original decision on orchestrator delegation
- **Decision:** `.kb/decisions/2025-12-04-orchestrator-delegates-all-investigations.md` - Reinforcement of delegation rule

---

## Investigation History

**2026-01-06 15:35:** Investigation started
- Initial question: How to detect orchestrator frame collapse
- Context: Meta-orchestrator identified price-watch orchestrator spending hours on manual CSS fixes

**2026-01-06 15:50:** Key finding - external detection required
- Self-detection unreliable per Finding 1
- Plugin infrastructure available per Finding 2

**2026-01-06 16:05:** Investigation completed
- Status: Complete
- Key outcome: Recommend hybrid detection via SESSION_HANDOFF.md check at orch complete, skill guidance, and potential plugin

---

## Appendix: Frame Collapse Incidents (Evidence Log)

**Purpose:** Document real frame collapse occurrences with transcripts to build evidence base for pattern recognition and detection improvement.

### Incident 1: Price-Watch Phase 1 Implementation (2026-01-13)

**Context:** Orchestrator received session handoff with completed investigation and ready issue (pw-qsj7, P2). Handoff said "Ready to start Phase 1 implementation" which triggered frame collapse.

**Transcript:** `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/2026-01-13-frame-collapse-example.txt`

**Collapse Pattern:**

1. **Trigger** (line 35): User said "let's begin" → orchestrator interpreted as directive to implement
2. **Escalation** (lines 51-271):
   - Read controller file (1325 lines)
   - Read view file (462 lines)
   - Read CSS file (2521 lines)
   - Made 4 separate `Update()` calls editing controller and view
   - Created implementation todo list (lines 299-317)
3. **Rationalization**: "Now I'll implement the controller changes" (line 60) - no questioning of role
4. **Intervention** (line 273): Dylan asked "whoa, what is your role?"
5. **Recovery** (lines 329-389):
   - Orchestrator recognized violation
   - Offered options (continue vs revert)
   - Dylan chose revert (option 2)
   - Full cleanup: `git restore`, changed beads status back to `open`
   - Proper spawn: `orch spawn --bypass-triage feature-impl "..." --issue pw-qsj7`

**Early Warning Signals Missed:**

- Line 51: `Read(backend/app/controllers/price_quotes_controller.rb)` - Reading implementation code (not SYNTHESIS.md/context)
- Line 60: **"Now I'll implement..."** - Explicit announcement of worker mode
- Line 62: First `Update()` call - Direct file editing
- Reading 4308 total lines across 3 files - Well beyond 5-minute context gathering threshold

**Key Observations:**

1. **Gradual escalation worked** - No single moment of "I'm doing worker work now", just incremental steps
2. **Handoff ambiguity** - "Ready to start Phase 1 implementation" suggested orchestrator should implement, not spawn
3. **Todo list as signal** - Creating implementation todos (not spawn todos) indicated worker thinking
4. **Self-awareness emerged only after intervention** - Orchestrator couldn't detect own frame until Dylan forced perspective shift
5. **Recovery was clean** - Once called out, orchestrator reverted fully and spawned properly

**What This Example Teaches:**

- **Reading code files is the first signal** - If orchestrator opens `.rb`, `.ts`, `.css` (not `.md` in `.kb/` or `.orch/`), frame collapse has begun
- **"I'll implement" is definitive** - Any orchestrator statement like "Now I'll implement" or "Let me update" is a violation
- **Handoff language matters** - "Ready to implement" should be "Ready to spawn implementation" to avoid ambiguity
- **Dylan's intervention pattern** - Simple question "what is your role?" forces metacognitive shift
- **5-minute rule violation** - 4308 lines read across 3 files took >5 minutes, should have triggered self-check

**Detection Opportunities (What Could Have Caught This):**

1. **Plugin tracking Edit tool** - After 2nd `Update()` call (line 116), could have warned
2. **File type heuristic** - Reading `.rb` file (not orchestration artifact) should trigger warning
3. **Time threshold** - Reading 4308 lines took >5 minutes, exceeding context-gathering threshold
4. **Skill injection** - Handoff could inject reminder: "Your role: spawn agents, don't implement"
5. **Pre-response gate** - Before first `Update()` call, skill could require: "Check delegation gate: is this spawnable work?"

**Recommendations:**

1. Update SESSION_HANDOFF.md template to say "Ready to spawn implementation" (not "Ready to start implementation")
2. Add file type detection to OpenCode plugin - warn when orchestrator reads code files
3. Strengthen skill's Pre-Response Protocol to gate on file type before Read tool
4. Create `kb quick constrain` entry: "Orchestrators saying 'I'll implement' = immediate frame collapse"
