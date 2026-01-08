## Summary (D.E.K.N.)

**Delta:** Store load-bearing guidance metadata in skill.yaml as `load_bearing[]` array, verified by skillc during check/deploy.

**Evidence:** Investigation of 3 options showed skill.yaml already supports structured build-time constraints (outputs, phases); kb tracks knowledge atoms not enforcement locations; inline SKILL.md comments would be swept away during refactors.

**Knowledge:** Load-bearing guidance is a build-time constraint, not runtime knowledge. Guards must be external to what they protect.

**Next:** Implement in skillc: add LoadBearingEntry struct, add verification, add `skillc protected` command.

---

# Decision: Load-Bearing Guidance Data Model

**Date:** 2026-01-08
**Status:** Accepted

**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A
**Related Epic:** orch-go-lv3yx (Protect Load-Bearing Guidance from Refactor Erosion)
**Investigation:** .kb/investigations/2026-01-08-inv-design-data-model-load-bearing.md

---

## Context

Hard-won behavioral guidance (from real friction) has no protection against refactoring. When token budgets pressure skills, guidance looks like cruft and gets cut. Friction recurs.

The system has:
- Token budgets (gates on size)
- Checksums (gates on unexpected changes)

The system lacks:
- Mechanism to distinguish "essential insight" from "verbose fluff"
- Protection for guidance that came from friction

Trigger: 2026-01-08 orchestrator session adding "Filter Before Presenting" and "Surface Decision Prerequisites" to orchestrator skill. Question: "How do we make sure these aren't swept away in refactors?"

---

## Options Considered

### Option A: skill.yaml load_bearing array ⭐ SELECTED
Store structured metadata in skill.yaml that skillc verifies during check/deploy.

```yaml
load_bearing:
  - pattern: "ABSOLUTE DELEGATION RULE"
    provenance: "2025-11 orchestrator doing investigations led to 3-day derailment"
    evidence: ".kb/investigations/2025-11-xx-orchestrator-investigation-derailment.md"
    severity: error  # error = block deploy, warn = advisory
    
  - pattern: "Filter Before Presenting"
    provenance: "2026-01-08 Dylan observed option theater pattern"
    evidence: "orch-go-lv3yx epic description"
    severity: warn
```

- **Pros:** Follows established skill.yaml pattern; external to SKILL.md (survives refactors); skillc already parses/verifies; provenance captures friction story
- **Cons:** Requires manual registration; patterns are strings not semantic understanding; no runtime query via kb context

### Option B: kb friction command
New kb command to record friction events with skill-location links.

- **Pros:** Centralizes knowledge in kb; queryable via kb context
- **Cons:** kb tracks knowledge atoms, not enforcement locations; conflates two concerns; kb is per-project but skills are cross-project

### Option C: Inline HTML comments in SKILL.md
Embed provenance as HTML comments next to protected guidance.

- **Pros:** Self-documenting; pattern and provenance in same place
- **Cons:** SKILL.md is compiled output, not source; comments can be swept away during refactors (exactly what we're trying to prevent); harder to aggregate across skills

---

## Decision

**Chosen:** Option A - skill.yaml load_bearing array

**Rationale:** Load-bearing guidance is a *build-time constraint* (verify during compilation), not *runtime knowledge* (query during work). skill.yaml is where build-time constraints live. This follows the established pattern of outputs, phases, and deliverables - structured data that skillc parses and verifies.

**Trade-offs accepted:**
- Manual registration required (vs auto-detection from kn entries) - acceptable because we want explicit, intentional protection
- Patterns are string substrings, not semantic understanding - acceptable for MVP, can enhance later
- No runtime query via kb context - acceptable because build-time verification is the goal

---

## Data Model

```go
// LoadBearingEntry represents a load-bearing guidance pattern that must exist in compiled output
type LoadBearingEntry struct {
    Pattern    string `yaml:"pattern"`    // Substring to search for in compiled output
    Provenance string `yaml:"provenance"` // Friction story: what happened without this guidance
    Evidence   string `yaml:"evidence"`   // Path to investigation/decision/kn entry
    Severity   string `yaml:"severity"`   // "error" (block deploy) or "warn" (advisory)
}
```

**Fields:**
- `pattern`: Case-insensitive substring to search for in compiled SKILL.md
- `provenance`: Human-readable story of the friction that produced this guidance (1-2 sentences)
- `evidence`: Path to artifact documenting the friction (optional, for deep context)
- `severity`: `error` blocks deploy, `warn` is advisory (default: `error`)

---

## Tooling

**skillc check** (existing): Add load-bearing pattern verification
- Search compiled output for each pattern
- Report missing patterns with provenance
- Respect severity for error vs warning

**skillc protected** (new): List all protected patterns
- Aggregate across skills (or filter by path)
- Show pattern, provenance, evidence
- Use before refactoring to see what's protected

**skillc deploy** (existing): Fail if any severity=error patterns missing

---

## Structured Uncertainty

**What's tested:**
- ✅ skill.yaml already supports structured arrays (read manifest.go)
- ✅ skillc verify already validates patterns (ran skillc verify --help)
- ✅ kn entries don't have skill-location fields (read entries.jsonl)

**What's untested:**
- ⚠️ Pattern matching performance with many entries (not benchmarked)
- ⚠️ User experience of registration workflow (no prototype)
- ⚠️ Whether severity distinction is needed in practice

**What would change this:**
- If kb needed "which skills use this constraint?" → would need bidirectional links
- If load-bearing metadata needs runtime query → would need different storage
- If string matching proves too fragile → might need semantic markers

---

## Consequences

**Positive:**
- Explicit protection for hard-won guidance
- Refactoring agents can't silently remove protected patterns
- Provenance preserved - future agents know WHY guidance exists
- Follows established skillc patterns - minimal new infrastructure

**Risks:**
- Pattern drift: guidance might be reworded in ways that don't match pattern
- Over-protection: everything marked as load-bearing defeats the purpose
- Maintenance burden: patterns need updating when guidance evolves

**Mitigations:**
- Use distinctive phrases unlikely to change ("ABSOLUTE DELEGATION RULE")
- Reserve for genuinely friction-derived guidance, not all guidance
- `skillc protected` provides visibility into what's protected

---

## Implementation Plan

1. **orch-go-lv3yx.4** - Register friction-to-guidance links
   - Add LoadBearingEntry struct to manifest.go
   - Add YAML parsing for load_bearing array
   - Add verification to checker.go

2. **orch-go-lv3yx.5** - skillc warns when load-bearing patterns missing
   - Integrate with skillc check
   - Integrate with skillc deploy
   - Add severity-based error/warning

3. **orch-go-lv3yx.6** - Refactor review gate for significant reductions
   - Use token budget delta + load-bearing check
   - Warn when significant reduction might have removed protected content

4. **orch-go-lv3yx.7** - Migration: Tag existing hard-won patterns
   - Add load_bearing entries to orchestrator skill.yaml
   - Document provenance for ABSOLUTE DELEGATION RULE, etc.
