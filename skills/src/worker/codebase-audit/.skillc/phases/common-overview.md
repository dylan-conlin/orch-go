## When to Use This Skill

**Use when the user says:**
- "Audit [focus area]" (security, performance, tests, architecture, organizational)
- "Run codebase health check"
- "Find [category] issues in the codebase"
- "Quick scan the codebase"

**Auto-detect dimension from context:**
- "Security vulnerabilities" → security dimension
- "Performance bottlenecks" → performance dimension
- "Test coverage" → tests dimension
- "God objects" / "tight coupling" → architecture dimension
- "ROADMAP drift" / "template drift" → organizational dimension
- "Quick health check" → quick dimension

---

## Skill Overview

This skill performs systematic codebase audits with configurable dimensions. Each dimension focuses on a specific area and produces an investigation file with findings, evidence, and actionable recommendations.

**Core workflow:**
1. **Pattern Search** - Automated searches for known issues
2. **Evidence Collection** - Concrete examples with file paths/line numbers
3. **Analysis** - Root cause identification and severity assessment
4. **Documentation** - Investigation file with prioritized recommendations

**Key deliverables:**
- Investigation file at `.kb/investigations/YYYY-MM-DD-audit-{dimension}.md`
- Progress tracked via `bd comment <beads-id> "Phase: [current phase] - [progress details]"`

---

## Evidence Hierarchy

**Artifacts are claims, not evidence.**

| Source Type | Examples | Treatment |
|-------------|----------|-----------|
| **Primary** (authoritative) | Actual code, test output, observed behavior | This IS the evidence |
| **Secondary** (claims to verify) | Workspaces, investigations, decisions | Hypotheses to test |

When an artifact says "X is not implemented," that's a hypothesis—not a finding to report. Search the codebase before concluding.

**The failure mode:** An audit reads a workspace claiming "feature X NOT DONE" and reports that as a finding without checking if feature X actually exists in the code. Always verify artifact claims against primary sources.

---

## Model Awareness (Probe vs Investigation Routing)

**Before creating any artifact, check SPAWN_CONTEXT.md for model-claim markers.**

### Detection

Find the `### Models (synthesized understanding)` section in SPAWN_CONTEXT.md. Look for injected model-claim markers in model entries:
- `- Summary:`
- `- Critical Invariants:` or `- Constraints:`
- `- Why This Fails:` or `- Failure Modes:`

### If markers are present → Probe Mode

Your audit findings likely confirm, contradict, or extend an existing model's claims about the system. Route findings to a probe instead of a standalone investigation.

- Pick the most relevant model from the injected models section
- Create: `.kb/models/{model-name}/probes/{date}-{slug}.md`
- Use template: `.orch/templates/PROBE.md`
- Required sections: `Question`, `What I Tested`, `What I Observed`, `Model Impact`
- Focus on how audit findings confirm, contradict, or extend the model's invariants

**Example:** Auditing architecture when a "completion pipeline" model exists → create a probe documenting how the audit's coupling/complexity findings confirm or contradict the model's architectural claims.

### If markers are absent → Investigation Mode

Follow standard investigation file setup below.

---

## Investigation File Setup

**CRITICAL:** Before starting the audit, create investigation file from template. This ensures all findings are documented progressively with proper metadata (including Resolution-Status field for synthesis workflow).

### Create Investigation Template

```bash
# Create investigation using kb CLI command
# Update SLUG based on your audit dimension and topic
# Use audit/ prefix for audit investigations
kb create investigation "audit/dimension-audit-description"
```

**After creating the template:**
1. Fill Question field with specific audit focus from SPAWN_CONTEXT
2. Update metadata (Started date set automatically, verify Status)
3. Document findings progressively during audit (don't wait until end)
4. Update Confidence and Resolution-Status when completing audit

**Important:**
- The `kb create investigation` command auto-detects project directory and creates the investigation in the appropriate subdirectory.
- The investigation file includes Resolution-Status field (Unresolved/Resolved/Recurring/Synthesized/Mitigated) which is critical for the synthesis workflow. Always fill this field when completing the investigation.

**Now proceed with dimension-specific audit guidance below.**

---

## Available Dimensions

### Focused Audits (30-90 min)

**security** - Security vulnerabilities, unsafe patterns, secrets exposure, OWASP compliance
- When: Investigating security risks, penetration test prep, compliance audit
- Output: Security findings with severity ratings (Critical/High/Medium/Low)

**performance** - Performance bottlenecks, N+1 queries, algorithmic complexity, slow operations
- When: App is slow, high resource usage, scaling issues
- Output: Performance findings with profiling data and optimization recommendations

**tests** - Test coverage gaps, flaky tests, missing test types, test quality
- When: Flaky builds, low confidence in tests, missing edge case coverage
- Output: Testing gaps with risk assessment and coverage metrics

**architecture** - Coupling, god objects, missing abstractions, modularity issues
- When: Hard to add features, tight coupling, unclear boundaries
- Output: Architectural issues with refactoring effort estimates

**organizational** - ROADMAP drift, template drift, documentation sync, process violations
- When: Docs out of date, ROADMAP showing completed work as TODO, templates inconsistent
- Output: Organizational drift findings with system amnesia analysis

### Quick Scan (1 hour)

**quick** - Automated pattern search across all focus areas, high-priority issues only
- When: Need rapid health check before major work, onboarding to new codebase
- Output: Top 10 findings across all categories with quick-win recommendations

---

## Common Patterns

**Full audit workflow (2-4 hours):**
1. Run `quick` dimension to identify top issues
2. Run focused dimension for high-priority areas
3. Synthesize findings into single investigation file
4. Prioritize using ROI framework (impact vs effort)

**Targeted audit workflow (30-90 min):**
1. Run single focused dimension (user knows the problem area)
2. Investigation file documents findings
3. Add high-priority items to ROADMAP
