# Synthesis Workflow Guide

**Purpose:** Document the established process for synthesizing multiple investigations into authoritative guides.

**Last verified:** 2026-02-14  
**Synthesized from:** 10 synthesis investigations (Jan-Feb 2026)

---

## When to Synthesize

Synthesis should happen when:
- **kb reflect shows 10+ investigations** in a topic cluster
- **Investigations accumulate over 1-2 weeks** (natural rhythm)
- **Same patterns appear across 3+ investigations** (redundancy signal)
- **Guide becomes stale** (multiple investigations fix/extend same guide)

**Investigation velocity as health metric:**
- High velocity (10+ in a week): Active development OR system friction
- Medium velocity (3-5 in a week): Normal evolution
- Low velocity (1-2 in a week): System stabilization

---

## The 5-Step Synthesis Workflow

All 10 observed syntheses (status, daemon, dashboard, orchestrator, spawn, verification, completion, extract, serve) followed this identical pattern:

### Step 1: Chronicle

```bash
kb chronicle "topic"
```

**Purpose:** Understand investigation evolution and timeline  
**What to look for:**
- Date clusters (when did most investigations happen?)
- Evolution patterns (early explorations → fixes → refinements)
- Contradictions (did later investigations contradict earlier ones?)

### Step 2: Read All Investigations

**Pattern:** Read every investigation file in the cluster, not just summaries.

**What to capture:**
- Findings and evidence (what was discovered?)
- Solutions implemented (what was fixed?)
- Patterns across investigations (what recurs?)
- Contradictions (where do findings conflict?)

**Tip:** Use a structured note-taking approach - one "finding" per investigation pattern.

### Step 3: Identify Themes

**Pattern:** Group findings into 3-9 major themes.

Examples from observed syntheses:
- **Status (5 themes):** Stale sessions, performance, liveness detection, title format, cross-project
- **Dashboard (4 themes):** Performance optimization, cross-project visibility, data pipeline integrity, activity feed
- **Verification (4 layers):** Meta-principle, visual verification, declarative constraints, completion gates
- **Completion (4 causes):** Agent-scoping ambiguity, gate proliferation, evidence vs claim, legacy compatibility

**Anti-pattern:** Creating 20+ micro-themes. Consolidate related findings.

### Step 4: Update or Create Guide

**Decision tree:**

```
Does authoritative guide exist for this topic?
├─ YES → Update existing guide
│  ├─ Add new sections for new themes
│  ├─ Update "Last verified" date
│  └─ Add investigations to References section
└─ NO → Create new guide
   ├─ Use existing guide structure (see status.md, daemon.md)
   ├─ Include: Architecture, Common Problems, Key Concepts, Decision History
   └─ Cross-reference related guides
```

**Guide-first pattern:** 8 of 10 syntheses updated existing guides. Only create new guides when no authoritative reference exists.

**Guide structure template:**
1. **Purpose** - What this guide covers
2. **Architecture** - How the system works
3. **Key Concepts** - Terms and patterns to understand
4. **Common Problems** - Troubleshooting checklist
5. **Decision History** - Why things are this way
6. **References** - Investigations synthesized

### Step 5: Archive Investigations

**Pattern:** After synthesis, investigations can be:
- **Archived** - Move to `.kb/investigations/archived/` if obsolete
- **Synthesized** - Move to `.kb/investigations/synthesized/{topic}/` if consolidated into guide
- **Kept** - Keep as reference if foundational, recent, or still relevant

**Example archival:**
```bash
mkdir -p .kb/investigations/synthesized/code-extraction-patterns
mv .kb/investigations/*extract*.md .kb/investigations/synthesized/code-extraction-patterns/
```

---

## kb reflect Integration

### Known Issues

**Issue 1: Scans archived/synthesized directories**
- **Symptom:** kb reflect still reports investigations after archival
- **Root cause:** Scanning logic includes archived/ and synthesized/ directories
- **Workaround:** Manually verify guide completeness, ignore kb reflect for synthesized clusters
- **Fix needed:** kb-cli should exclude these directories from synthesis detection

**Issue 2: Lexical clustering creates false positives**
- **Symptom:** Unrelated investigations grouped by keyword (e.g., "extract" matches code extraction, knowledge extraction, constraint extraction)
- **Root cause:** Clustering uses filename keywords, not semantic analysis
- **Workaround:** Human triage required - verify investigations are actually related
- **Fix needed:** Semantic clustering or topic tagging

**Issue 3: Time-drifted conclusions**
- **Symptom:** Investigation findings become stale after code changes
- **Root cause:** No mechanism to detect when code invalidates investigation claims
- **Workaround:** Re-validate findings during synthesis, test claims against current code
- **Fix needed:** Investigation revision tracking or staleness detection

### How to Use kb reflect for Synthesis

```bash
# Find synthesis opportunities
kb reflect --type synthesis --format json

# Review specific cluster
kb chronicle "topic"

# After synthesis, update model with findings
# (kb reflect model is at .kb/models/kb-reflect-cluster-hygiene.md)
```

**Pattern:** kb reflect is a **discovery tool, not a decision tool**. It surfaces signals; humans must triage.

---

## Quality Gates for Synthesis

Before marking synthesis complete:

- [ ] All investigations in cluster read and analyzed
- [ ] Themes identified (3-9 major patterns)
- [ ] Guide updated or created
- [ ] "Last verified" date updated
- [ ] Investigation count updated in guide header
- [ ] Cross-references to related guides added
- [ ] Archival decision made (archive, synthesize directory, or keep)
- [ ] No contradictions left unresolved in guide

---

## Synthesis Rhythm

**Observed pattern (Jan 2026):**
- Jan 6: 3 syntheses
- Jan 7: 5 syntheses
- Jan 8: 3 syntheses
- Jan 14: 1 synthesis
- Jan 17: 4 syntheses

**Natural rhythm:** Accumulate investigations over 1-2 weeks → synthesis session → repeat.

**Trigger:** When `kb reflect --type synthesis` shows 10+ investigation clusters.

---

## Examples from Practice

### Example 1: Status Synthesis (Jan 6)

**Input:** 10 status investigations (Dec 20 - Jan 5)  
**Action:** Updated `.kb/guides/status.md`  
**Themes identified:** 5 (stale sessions, performance, liveness, title format, cross-project)  
**Result:** Single authoritative reference for status command debugging

**Key insight:** Four-layer architecture (OpenCode in-memory, OpenCode disk, orch registry, tmux windows) is fundamental to understanding status issues.

### Example 2: Verification Synthesis (Jan 14)

**Input:** 25 verification investigations (Dec 21 - Jan 14)  
**Action:** Created NEW `.kb/guides/verification.md`  
**Themes identified:** 4 layers (meta-principle, visual, declarative, completion)  
**Result:** Comprehensive verification reference complementing existing completion guide

**Key insight:** Verification Bottleneck principle ("system cannot change faster than humans verify") emerged from 462 lost commits across two rollbacks.

### Example 3: Extract Synthesis (Jan 17)

**Input:** 13 extraction investigations  
**Action:** Verified existing guide complete, archived investigations  
**Outcome:** Discovered kb reflect bug (scans synthesized/ directories)  

**Key insight:** Sometimes synthesis reveals the guide is already complete - verification is the deliverable, not guide creation.

---

## Anti-Patterns to Avoid

### 1. Creating New Guides When Existing Ones Exist

**Symptom:** Multiple guides covering overlapping topics  
**Fix:** Update existing guide rather than creating parallel documentation  
**Observed:** 8 of 10 syntheses updated existing guides

### 2. Leaving Investigations Scattered

**Symptom:** 20+ investigations on same topic, no authoritative reference  
**Fix:** Follow synthesis workflow - consolidate into guide  
**Observed:** All 10 syntheses created or updated guides

### 3. Trusting kb reflect Without Validation

**Symptom:** Synthesizing unrelated investigations because keyword matched  
**Fix:** Human triage required - verify investigations are semantically related  
**Observed:** Extract synthesis found false positives (code extraction ≠ knowledge extraction)

### 4. Synthesis Without Re-validation

**Symptom:** Guide contains stale findings contradicted by current code  
**Fix:** Test key claims during synthesis, update or remove stale conclusions  
**Observed:** Verification synthesis found time-drifted conclusions

---

## Decision History

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-01-06 | Synthesis uses guide-first approach | Prevents knowledge fragmentation, maintains single source of truth |
| 2026-01-14 | kb reflect is discovery tool, not decision tool | Lexical clustering creates false positives requiring human triage |
| 2026-01-17 | Investigation velocity is system health metric | High counts indicate active development or friction points |
| 2026-02-14 | Synthesis workflow documented as guide | 10 successful syntheses prove pattern works, future agents benefit |

---

## References

**Investigations Synthesized:**
1. `2026-01-06-inv-synthesize-status-investigations.md` - Status synthesis (10 investigations)
2. `2026-01-07-inv-synthesize-daemon-investigations.md` - Daemon synthesis (33 investigations)
3. `2026-01-07-inv-synthesize-dashboard-investigations.md` - Dashboard synthesis (58 investigations)
4. `2026-01-07-inv-synthesize-orchestrator-investigations.md` - Orchestrator synthesis (47 investigations)
5. `2026-01-07-inv-synthesize-spawn-investigations.md` - Spawn synthesis (60 investigations)
6. `2026-01-14-inv-synthesize-verification-investigations-consolidate-findings.md` - Verification synthesis (25 investigations)
7. `2026-01-17-inv-design-synthesize-26-completion-investigations.md` - Completion architectural analysis (26 investigations)
8. `2026-01-17-inv-synthesize-28-completed-investigations-complete.md` - Completion synthesis (28 investigations)
9. `2026-01-17-inv-synthesize-extract-investigation-cluster-13.md` - Extract synthesis (13 investigations)
10. `2026-01-17-inv-synthesize-serve-investigation-cluster-investigations.md` - Serve synthesis (9 investigations)

**Related Guides:**
- `.kb/guides/status.md` - Status command debugging
- `.kb/guides/daemon.md` - Daemon operation and troubleshooting
- `.kb/guides/dashboard.md` - Dashboard architecture and performance
- `.kb/guides/orchestrator-session-management.md` - Orchestrator session lifecycle
- `.kb/guides/spawn.md` - Spawn command and agent creation
- `.kb/guides/verification.md` - Verification patterns and gates
- `.kb/guides/completion.md` - Completion verification workflow
- `.kb/guides/background-services-performance.md` - Serve performance patterns
- `.kb/guides/code-extraction-patterns.md` - Code extraction best practices

**Related Models:**
- `.kb/models/kb-reflect-cluster-hygiene.md` - kb reflect cluster triage patterns
