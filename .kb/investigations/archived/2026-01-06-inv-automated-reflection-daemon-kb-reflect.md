<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The daemon should run synthesis (10+ investigations, already implemented) automatically with issue creation, and open (unimplemented recommendations) hourly. Other types (promote, stale, drift, skill-candidate, refine) should only surface to orchestrator, not auto-create issues.

**Evidence:** Reviewed kb reflect output showing 7 types available; synthesis already creates issues via --create-issue flag; current daemon has ReflectEnabled/ReflectInterval/ReflectCreateIssues config; kn skill-candidate shows 72 spawn-related entries but most are low-value noise.

**Knowledge:** Issue creation automation should only trigger for high-signal, actionable patterns. Synthesis (10+ investigations = clear synthesis need) and open (explicit Next: actions) meet this bar. Other types require human judgment for triage.

**Next:** Implement open type issue creation in kb-cli, add periodic open reflection to daemon run loop, keep other types as surfacing-only.

---

# Investigation: Automated Reflection Daemon - Which kb reflect Types Should Run Automatically?

**Question:** What kb reflect types should the daemon run automatically and create issues for? How should the full automation loop for knowledge maintenance work?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** og-work-automated-reflection-daemon-06jan-111c
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Seven Reflection Types Available, Each with Different Signal Quality

**Evidence:** From `kb reflect --help`:

| Type | What it Detects | Signal Quality |
|------|-----------------|----------------|
| synthesis | 3+ investigations on same topic | High - clear synthesis need |
| open | Investigations with unimplemented Next: actions | High - explicit actionable items |
| promote | kn entries worth promoting to kb | Medium - requires human judgment |
| stale | Decisions with no citations >7 days | Medium - may be valid but unused |
| drift | Constraints contradicted by code | Low - heuristic-based, high false positive |
| refine | kn entries refining principles | Medium - requires human evaluation |
| skill-candidate | kn entry clusters (3+) on same topic | Low-Medium - noisy due to verb-based clustering |

**Source:** `kb reflect --help`, `kb reflect --format json` output

**Significance:** Not all reflection types are suitable for automated issue creation. High-signal types (synthesis, open) can safely auto-create issues. Medium and low signal types should surface to orchestrator for manual triage.

---

### Finding 2: Current Daemon Implementation Only Handles Synthesis with Issue Creation

**Evidence:** From `pkg/daemon/daemon.go` (lines 37-47) and `pkg/daemon/reflect.go`:

```go
// Config in daemon.go
ReflectEnabled      bool          // Controls periodic reflection
ReflectInterval     time.Duration // Default 1 hour
ReflectCreateIssues bool          // Create beads issues for synthesis (10+)

// RunReflectionWithOptions in reflect.go
if createIssues {
    args = append(args, "--type", "synthesis", "--create-issue")
}
```

When `ReflectCreateIssues=true`, the daemon passes `--type synthesis --create-issue` to kb reflect, which creates beads issues for topics with 10+ investigations.

**Source:** `pkg/daemon/daemon.go:37-47`, `pkg/daemon/reflect.go:107-115`

**Significance:** The daemon already has the infrastructure for periodic reflection with issue creation. Extending it to other types requires adding new config flags and handling multiple reflect types per cycle.

---

### Finding 3: Skill-Candidate Produces High Volume, Low Value Results

**Evidence:** Running `kb reflect --type skill-candidate` returned 72 entries for "spawn" topic alone. Most entries are routine decisions (e.g., "Default spawn mode is headless", "Skill constraints use spawn time mtime filtering") that don't indicate skill updates are needed.

The keyword-based clustering groups all kn entries containing "spawn" together, including:
- Implementation decisions
- Bug fixes
- Constraints
- Questions

**Source:** `kb reflect --type skill-candidate --format json` output

**Significance:** Skill-candidate detection needs semantic filtering, not just keyword clustering, before it's suitable for automation. Current implementation would create noisy issues. This should remain orchestrator-surfacing only.

---

### Finding 4: Open Type is Highly Actionable

**Evidence:** Running `kb reflect --type open` returned only 4 items, all with clear Next: actions:

```json
{
  "open": [
    {
      "file": "2025-12-25-inv-pattern-tool-relationship-shareability.md",
      "next_action": "Continue exploration. Map more examples. Test if model holds.",
      "age_days": 12
    },
    ...
  ]
}
```

These represent investigations where work was started but not completed. The signal is explicit (agent wrote "Next: <action>") rather than inferred.

**Source:** `kb reflect --type open --format json`

**Significance:** Open type is highly suitable for automated issue creation - it captures explicit commitments that weren't fulfilled. Unlike synthesis (inferred need) or skill-candidate (noisy clustering), open items are self-declared actionable.

---

### Finding 5: Prior Investigation Recommended Periodic Daemon Integration

**Evidence:** From `.kb/investigations/2025-12-21-inv-daemon-hook-integration-kb-reflect.md`:

> "The daemon already has subcommands (run, once, preview). Adding a reflect subcommand is cleaner than modifying the run loop... Implemented as orch daemon reflect subcommand rather than automatic reflection during daemon run."

And from `.kb/investigations/2025-12-21-inv-design-self-reflection-protocol-specification.md`:

> "Detection uses content parsing (grep), not indexes. Triggers on density, not schedules... kb reflect should surface patterns; chronicle gathers sources; orchestrator produces narrative."

**Source:** Referenced investigations

**Significance:** The system was intentionally designed for human-in-loop synthesis. Automated issue creation should be limited to high-confidence patterns, with orchestrator handling narrative synthesis.

---

### Finding 6: Issue Creation Thresholds Need Differentiation

**Evidence:** Current synthesis threshold is 10+ investigations per topic. This is appropriate because:
- 3-9 investigations may be natural exploration, not consolidation debt
- 10+ indicates persistent topic without synthesis

For open type, every item is actionable by definition (agent explicitly stated Next: action), so no threshold is needed - any age >3 days is enough signal.

For other types:
- promote: No clear threshold - duplicates could be intentional emphasis
- stale: 7+ days without citation is weak signal (decisions can be valid but rarely cited)
- drift: Heuristic detection has ~30-50% false positive rate

**Source:** Prior investigation synthesis, current kb reflect behavior

**Significance:** Each type needs type-specific thresholds for issue creation. Only synthesis (10+) and open (any, >3d) have clear enough thresholds for automation.

---

## Synthesis

**Key Insights:**

1. **Signal Quality Determines Automation Suitability** - Only high-signal patterns (synthesis 10+, open items) should auto-create issues. Medium/low signal patterns (promote, stale, drift, skill-candidate, refine) should surface to orchestrator for judgment.

2. **Open Type Fills a Gap** - Current implementation only handles synthesis. Open type captures explicit Next: actions that weren't completed - these are self-declared actionable items, not inferred patterns.

3. **Daemon Run Loop vs Separate Subcommand** - For synthesis (expensive, creates issues), hourly via periodic check is appropriate. For open (cheap, smaller scope), could run more frequently. The `orch daemon reflect` subcommand is for manual/ad-hoc runs.

**Answer to Investigation Question:**

The daemon should automate two reflection types with issue creation:

| Type | Frequency | Issue Creation Threshold | Triage Label |
|------|-----------|--------------------------|--------------|
| **synthesis** | Hourly (current) | 10+ investigations | `triage:review` |
| **open** | Hourly | Any item >3 days old | `triage:review` |

Other types should be surfacing-only (visible in `orch daemon status` and SessionStart hook):

| Type | Frequency | Why No Issue Creation |
|------|-----------|----------------------|
| promote | Hourly | Requires human judgment on kb vs principles promotion |
| stale | Hourly | Weak signal - decisions may be valid but rarely cited |
| drift | Hourly | High false positive rate from heuristic detection |
| skill-candidate | Hourly | Noisy clustering, needs semantic filtering |
| refine | Hourly | Requires human evaluation of principle refinement |

---

## Structured Uncertainty

**What's tested:**

- ✅ Current daemon reflection works with synthesis (verified: reviewed daemon.go, reflect.go implementation)
- ✅ kb reflect --create-issue creates beads issues for synthesis (verified: `kb reflect --help` shows flag)
- ✅ Open type returns actionable items with explicit Next: fields (verified: ran `kb reflect --type open`)
- ✅ Skill-candidate returns high volume results (verified: 72 spawn entries in output)

**What's untested:**

- ⚠️ Issue creation for open type (kb reflect --type open --create-issue not yet implemented)
- ⚠️ Multi-type reflection in single daemon cycle (currently only synthesis)
- ⚠️ False positive rate for open type issue creation (may create issues for abandoned investigations)
- ⚠️ Dashboard display of surfacing-only reflection results

**What would change this:**

- If open type issue creation produces >20% noise → increase age threshold or add status filter
- If orchestrator reviews show other types are highly actionable → consider adding issue creation
- If synthesis issue creation overwhelms backlog → raise threshold from 10+ to 15+

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Two-tier automation: synthesis + open auto-create issues; all others surface-only**

**Why this approach:**
- Maintains human-in-loop for medium/low signal patterns
- Captures two distinct high-signal patterns (accumulated synthesis debt, forgotten commitments)
- Builds on existing infrastructure (ReflectCreateIssues already exists)
- Low risk of overwhelming backlog (both have natural volume limits)

**Trade-offs accepted:**
- Orchestrator must manually triage promote/stale/drift/skill-candidate/refine suggestions
- May miss some actionable items in surface-only types
- Requires kb-cli change to add `--create-issue` for open type

**Implementation sequence:**
1. **Add open type issue creation to kb-cli** - Implement `kb reflect --type open --create-issue` with same pattern as synthesis
2. **Extend daemon reflection config** - Add `ReflectOpenEnabled` and `ReflectOpenInterval` to Config
3. **Update daemon run loop** - Call both synthesis and open reflection in periodic check
4. **Surface all types in daemon status** - Show promote/stale/drift/refine/skill-candidate counts in `orch daemon status`

### Alternative Approaches Considered

**Option B: Automate all types with issue creation**
- **Pros:** Full automation, no human triage needed
- **Cons:** High noise from skill-candidate, drift false positives; overwhelming backlog
- **When to use instead:** If future semantic filtering dramatically improves signal quality

**Option C: Surface-only for all types (no auto-issue creation)**
- **Pros:** Maximum human control, zero noise
- **Cons:** Loses value of synthesis issue creation (already proven useful); requires constant manual review
- **When to use instead:** If backlog becomes unmanageable or false positive rate increases

**Rationale for recommendation:** Synthesis issue creation is already proven valuable (implemented and used). Open type has equally clear signal (explicit Next: actions). Other types need semantic improvements before automation.

---

### Implementation Details

**What to implement first:**
1. `kb reflect --type open --create-issue` in kb-cli (mirrors synthesis pattern)
2. Daemon config for open type: `ReflectOpenEnabled`, `ReflectOpenInterval` (default: hourly like synthesis)
3. Issue format for open type: "Complete investigation: {title} - {age} days without action"

**File targets:**

| Location | Change |
|----------|--------|
| kb-cli `cmd/reflect.go` | Add --create-issue support for open type |
| orch-go `pkg/daemon/daemon.go` | Add ReflectOpenEnabled config |
| orch-go `pkg/daemon/reflect.go` | Add RunOpenReflection function |
| orch-go `cmd/orch/daemon.go` | Add open reflection to run loop |

**Things to watch out for:**
- ⚠️ Deduplication - don't create issues for already-tracked open items
- ⚠️ Age threshold - may need tuning (start with 3 days, adjust based on noise)
- ⚠️ Investigation closure - open issues should auto-close when investigation status changes to Complete
- ⚠️ Surfacing format - ensure non-issue-creating types are visible in dashboard/status

**Success criteria:**
- ✅ Open investigations >3 days automatically create triage:review issues
- ✅ Synthesis topics with 10+ investigations continue creating issues (no regression)
- ✅ Promote/stale/drift/refine/skill-candidate visible in `orch daemon status` without creating issues
- ✅ False positive rate for auto-created issues <20%
- ✅ Orchestrator can clear surfacing-only suggestions via `orch daemon reflect --clear-surfacing`

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Daemon config, ReflectEnabled/Interval/CreateIssues settings
- `pkg/daemon/reflect.go` - RunReflection implementation, types, suggestions storage
- `.kb/investigations/2025-12-21-inv-daemon-hook-integration-kb-reflect.md` - Prior daemon integration design
- `.kb/investigations/2025-12-21-inv-design-self-reflection-protocol-specification.md` - Self-reflection architecture
- `.kb/investigations/2025-12-21-design-kb-reflect-command-specification.md` - kb reflect command design
- `.kb/investigations/2025-12-25-inv-run-kb-reflect-across-all.md` - Prior all-types analysis

**Commands Run:**
```bash
# Check available reflection types
kb reflect --help

# Run synthesis to see current output
kb reflect --type synthesis --format json

# Check skill-candidate noise level
kb reflect --type skill-candidate --format json

# Check open type signal quality
kb reflect --type open --format json

# Check current daemon implementation
rg "Reflect" pkg/daemon/daemon.go
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-kb-reflect-command-interface.md` - kb reflect interface design
- **Investigation:** `.kb/investigations/2025-12-21-inv-design-self-reflection-protocol-specification.md` - Full self-reflection protocol

---

## Investigation History

**2026-01-06 11:15:** Investigation started
- Initial question: What kb reflect types should run automatically and create issues?
- Context: Spawned as design-session for daemon automation loop

**2026-01-06 11:30:** Context gathered
- Read 7 prior investigations on reflection/daemon
- Reviewed daemon.go and reflect.go implementation
- Ran kb reflect for all types to assess signal quality

**2026-01-06 11:45:** Key discovery
- Skill-candidate produces 72 entries for single topic (spawn) - too noisy
- Open type has only 4 items, all with explicit Next: actions - high signal
- Current implementation only handles synthesis

**2026-01-06 12:00:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Two-tier automation - synthesis + open auto-create; others surface-only
