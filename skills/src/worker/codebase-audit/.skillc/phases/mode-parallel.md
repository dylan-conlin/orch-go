# Parallel Execution Mode

**TLDR:** Use 5 parallel Haiku agents (one per dimension) for 3x faster comprehensive audits. Each agent runs pattern searches and returns JSON findings, which a synthesis agent combines into a prioritized report.

**When to use:** Comprehensive audit needed across multiple dimensions, time-constrained review, full codebase health check before major work.

**Output:** Single investigation file with prioritized findings from all dimensions.

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

---

## Key Design Decisions

1. **Haiku for dimension agents** - Pattern searches are IO-bound (grep/glob), not reasoning-heavy. Haiku is 3x faster and cheaper than Sonnet for this workload.

2. **JSON output from dimension agents** - Structured data enables consistent synthesis across agents.

3. **Separate synthesis step** - Keeps dimension agents focused on discovery; synthesis agent handles prioritization logic.

4. **No confidence scoring** - Unlike code-review (which filters false positives), codebase-audit produces objective pattern matches (file exists at N lines = fact, not opinion).

---

## Workflow

### Step 1: Spawn 5 Parallel Dimension Agents

Use a single message with 5 Task tool invocations to spawn all dimension agents concurrently:

```markdown
**For orchestrators:** Spawn parallel audit agents using:

1. Security Agent (Haiku) - Returns JSON with secrets, injection, auth findings
2. Performance Agent (Haiku) - Returns JSON with large files, complexity, N+1 findings
3. Architecture Agent (Haiku) - Returns JSON with god objects, coupling findings
4. Tests Agent (Haiku) - Returns JSON with coverage gaps, flaky test indicators
5. Organizational Agent (Haiku) - Returns JSON with drift patterns, doc sync findings

Each agent prompt should specify:
- Dimension to audit
- Project directory
- JSON output format requirement
- Pattern search commands to run
```

**Example Task tool invocation (5 in one message):**

```
Task 1: "Audit security dimension of PROJECT_DIR. Run pattern searches for secrets, injection, auth issues. Return JSON: {potential_secrets: N, injection_risks: N, auth_issues: N, findings: [...]}"

Task 2: "Audit performance dimension of PROJECT_DIR. Run pattern searches for large files, complexity, N+1. Return JSON: {large_files: [...], complexity_issues: N, findings: [...]}"

Task 3: "Audit architecture dimension of PROJECT_DIR. Run pattern searches for god objects, coupling. Return JSON: {god_objects: [...], coupling_issues: N, findings: [...]}"

Task 4: "Audit tests dimension of PROJECT_DIR. Run pattern searches for coverage gaps, flaky indicators. Return JSON: {coverage_gaps: N, flaky_tests: N, findings: [...]}"

Task 5: "Audit organizational dimension of PROJECT_DIR. Run pattern searches for drift, doc sync. Return JSON: {roadmap_drift: N, template_drift: N, findings: [...]}"
```

### Step 2: Wait for All Agents to Complete

All 5 agents run concurrently. Wait for all Task results to return.

**Expected latency:** ~5-10 seconds (parallel) vs ~15-30 seconds (sequential)

### Step 3: Spawn Synthesis Agent

Once all dimension agent results are available, spawn a synthesis agent:

```markdown
Task: "Synthesize codebase audit findings from 5 dimension agents.

Security findings: {JSON from agent 1}
Performance findings: {JSON from agent 2}
Architecture findings: {JSON from agent 3}
Tests findings: {JSON from agent 4}
Organizational findings: {JSON from agent 5}

Produce prioritized findings:
1. Combine all findings
2. Assign severity (Critical/High/Medium/Low)
3. Sort by ROI (impact vs effort)
4. Return top 20 findings with recommendations"
```

### Step 4: Write Investigation File

Write synthesis output to investigation file:

```bash
# Investigation file location
.kb/investigations/YYYY-MM-DD-audit-comprehensive-parallel.md
```

---

## Expected Benefits

| Metric | Sequential | Parallel | Improvement |
|--------|------------|----------|-------------|
| Wall-clock time | ~15-30 min | ~5-10 min | **3x faster** |
| Token cost | 1x Sonnet | 5x Haiku + 1x Haiku | ~Equal or cheaper |
| Coverage | Single dimension | All dimensions | **Comprehensive** |

---

## Agent Output Format

Each dimension agent returns structured JSON for synthesis:

**Security Agent:**
```json
{
  "dimension": "security",
  "potential_secrets": 20,
  "injection_risks": 3,
  "auth_issues": 0,
  "findings": [
    {"type": "secret", "file": "config.py", "line": 45, "severity": "high", "description": "Hardcoded API key"},
    {"type": "injection", "file": "api.py", "line": 123, "severity": "critical", "description": "SQL injection risk"}
  ]
}
```

**Architecture Agent:**
```json
{
  "dimension": "architecture",
  "god_objects": [
    {"file": "cli.py", "lines": 4031, "methods": 85},
    {"file": "spawn.py", "lines": 2110, "methods": 42}
  ],
  "coupling_issues": 52,
  "findings": [
    {"type": "god_object", "file": "cli.py", "severity": "high", "description": "4031 lines exceeds 300-line threshold"}
  ]
}
```

---

## Synthesis Output Format

The synthesis agent produces a prioritized report:

```markdown
# Comprehensive Audit: [Project Name]

**Date:** YYYY-MM-DD
**Mode:** Parallel (5 dimension agents + synthesis)
**Duration:** X minutes

## Executive Summary

- **Critical findings:** N
- **High priority:** N
- **Medium priority:** N
- **Total findings:** N

## Prioritized Findings (by ROI)

### 1. [CRITICAL] Security: SQL injection in api.py:123
**Dimension:** Security
**Impact:** High (data breach risk)
**Effort:** Low (parameterized queries)
**Recommendation:** Use parameterized queries instead of string formatting

### 2. [HIGH] Architecture: cli.py at 4031 lines
**Dimension:** Architecture
**Impact:** High (maintainability, testing difficulty)
**Effort:** Medium (extract modules)
**Recommendation:** Extract command handlers to separate modules

### 3-20. [Additional findings...]

## Metrics Baseline

| Dimension | Key Metric | Value |
|-----------|------------|-------|
| Security | Potential secrets | 20 |
| Architecture | Files >300 lines | 3 |
| Tests | Coverage gaps | 15 |
| Performance | N+1 queries | 5 |
| Organizational | ROADMAP drift | 8 |

## Next Steps

1. Address critical findings immediately
2. Schedule high-priority fixes this sprint
3. Add medium-priority to backlog
4. Re-audit in 30 days to measure improvement
```

---

## When NOT to Use Parallel Mode

- **Single dimension focus** - If you already know the problem area, use focused audit instead
- **Quick health check** - Use `dimension: quick` for rapid triage without parallel overhead
- **Limited context** - Parallel spawns 6 agents; if context window is constrained, use sequential

---

## Comparison with Sequential Audit

| Aspect | Sequential | Parallel |
|--------|------------|----------|
| **Speed** | 15-30 min | 5-10 min |
| **Token cost** | Lower | Similar (Haiku is cheap) |
| **Depth** | Single dimension deep dive | All dimensions breadth |
| **Use case** | Known problem area | Comprehensive health check |
| **Coordination** | Simple | Requires synthesis step |

---

## Reference

- **Investigation:** `.kb/investigations/simple/2025-11-29-explore-multi-agent-parallel-review.md`
- **Pattern source:** Code-review plugin parallel agent architecture
