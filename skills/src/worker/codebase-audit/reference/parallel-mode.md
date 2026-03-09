# Parallel Execution Mode (Reference)

**TLDR:** Use 5 parallel Haiku agents (one per dimension) for 3x faster comprehensive audits. Each agent runs pattern searches and returns JSON findings, which a synthesis agent combines into a prioritized report.

**When to use:** Comprehensive audit needed across multiple dimensions, time-constrained review, full codebase health check before major work.

---

## Architecture

```
┌─────────────────┐
│  Orchestrator   │ (spawns all agents in single message)
└────────┬────────┘
         │
    ┌────┴────┬────────┬────────┬────────┐
    ▼         ▼        ▼        ▼        ▼
┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐
│Security│ │Perf   │ │Arch   │ │Tests  │ │Org    │
│ Agent │ │ Agent │ │ Agent │ │ Agent │ │ Agent │
│(Haiku)│ │(Haiku)│ │(Haiku)│ │(Haiku)│ │(Haiku)│
└───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘
    │         │        │        │        │
    └────┬────┴────────┴────────┴────────┘
         │ (JSON findings)
         ▼
┌─────────────────┐
│  Synthesis      │ (Haiku - prioritizes & formats)
│  Agent          │
└────────┬────────┘
         │ (Prioritized report)
         ▼
┌─────────────────┐
│  Final Output   │
└─────────────────┘
```

## Workflow

### Step 1: Spawn 5 Parallel Dimension Agents

Use a single message with 5 Task tool invocations:

```
Task 1: "Audit security dimension of PROJECT_DIR. Return JSON: {potential_secrets: N, injection_risks: N, findings: [...]}"
Task 2: "Audit performance dimension. Return JSON: {large_files: [...], complexity_issues: N, findings: [...]}"
Task 3: "Audit architecture dimension. Return JSON: {god_objects: [...], coupling_issues: N, findings: [...]}"
Task 4: "Audit tests dimension. Return JSON: {coverage_gaps: N, flaky_tests: N, findings: [...]}"
Task 5: "Audit organizational dimension. Return JSON: {roadmap_drift: N, template_drift: N, findings: [...]}"
```

### Step 2: Synthesis Agent

Combine all dimension results:
```
Task: "Synthesize findings from 5 dimension agents. Combine, assign severity, sort by ROI, return top 20."
```

### Step 3: Write Investigation File

`.kb/investigations/YYYY-MM-DD-audit-comprehensive-parallel.md`

## Agent Output Format

Each dimension agent returns structured JSON:

```json
{
  "dimension": "security",
  "potential_secrets": 20,
  "injection_risks": 3,
  "findings": [
    {"type": "secret", "file": "config.py", "line": 45, "severity": "high", "description": "Hardcoded API key"}
  ]
}
```

## Expected Benefits

| Metric | Sequential | Parallel | Improvement |
|--------|------------|----------|-------------|
| Wall-clock time | ~15-30 min | ~5-10 min | 3x faster |
| Token cost | 1x Sonnet | 5x Haiku + 1x Haiku | ~Equal or cheaper |
| Coverage | Single dimension | All dimensions | Comprehensive |

## When NOT to Use

- Single dimension focus → use focused audit instead
- Quick health check → use `dimension: quick`
- Limited context → parallel spawns 6 agents
