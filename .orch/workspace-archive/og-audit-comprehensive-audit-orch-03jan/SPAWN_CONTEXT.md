TASK: Comprehensive audit of orch-go: bugs/reliability concerns, architectural gaps, refactoring needs, test coverage gaps, code quality issues. Focus on actionable findings - things we should fix or track. Be thorough but prioritize findings by impact.

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "comprehensive"

### Prior Decisions
- Dashboard beads stats use bd stats --json API call
  - Reason: Provides comprehensive issue statistics with ready/blocked/open counts in single call
- Single-Agent Review Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2025-12-21-single-agent-review-command.md

### Related Investigations
- CLI orch complete Command Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-complete-command.md
- CLI Orch Status Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-cli-orch-status-command.md
- OpenCode Client Package Session Management
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-opencode-session-management.md
- SSE Event Monitoring Client
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-19-inv-client-sse-event-monitoring.md
- Add Wait Command to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-add-wait-command-orch.md
- Enhance orch review to parse and display SYNTHESIS.md
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-enhance-orch-review-parse-display.md
- Enhance status command with swarm progress
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-enhance-status-command-swarm-progress.md
- Expose Strategic Alignment Commands
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-expose-strategic-alignment-commands-focus.md
- Finalize Native Implementation for orch send
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-finalize-native-implementation-orch-send.md
- Agent Registry for Persistent Tracking
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-agent-registry-persistent.md
- orch-go Add Question Command
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-orch-add-question-command.md
- Final Sanity Check of orch-go Commands
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-perform-final-sanity-check-orch.md
- Refactoring pkg/registry as Beads Issue State Cache
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-plan-refactoring-pkg-registry-act.md
- Deep Pattern Analysis Across Orchestration Artifacts
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-design-deep-pattern-analysis-orchestration-artifacts.md
- Design: Minimal Artifact Set Specification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-design-minimal-artifact-taxonomy.md
- Add tmux fallback for orch status and tail
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-add-tmux-fallback-orch-status.md
- Registry Usage Audit in orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md
- Beads ↔ KB ↔ Workspace Relationship Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-beads-kb-workspace-relationships-how.md
- Beads OSS Relationship - Fork vs Contribute vs Local Patches
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-beads-oss-relationship-fork-vs.md
- Deep Dive into Inter-Agent Communication Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-deep-dive-inter-agent-communication.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.




📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `/exit` to close the agent session



CONTEXT: [See task description]

PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go

SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])
- Default estimation
- Recommend checkpoint after Phase 1 if session exceeds 2 hours


AUTHORITY:
**You have authority to decide:**
- Implementation details (how to structure code, naming, file organization)
- Testing strategies (which tests to write, test frameworks to use)
- Refactoring within scope (improving code quality without changing behavior)
- Tool/library selection within established patterns (using tools already in project)
- Documentation structure and wording

**You must escalate to orchestrator when:**
- Architectural decisions needed (changing system structure, adding new patterns)
- Scope boundaries unclear (unsure if something is IN vs OUT scope)
- Requirements ambiguous (multiple valid interpretations exist)
- Blocked by external dependencies (missing access, broken tools, unclear context)
- Major trade-offs discovered (performance vs maintainability, security vs usability)
- Task estimation significantly wrong (2h task is actually 8h)

**When uncertain:** Err on side of escalation. Document question in workspace, set Status: QUESTION, and wait for orchestrator response. Better to ask than guess wrong.

**Surface Before Circumvent:**
Before working around ANY constraint (technical, architectural, or process):
1. Document it in your investigation file: "CONSTRAINT: [what constraint] - [why considering workaround]"
2. Include the constraint and your reasoning in SYNTHESIS.md
3. The accountability is a feature, not a cost

This applies to:
- System constraints discovered during work (e.g., API limits, tool limitations)
- Architectural patterns that seem inconvenient for your task
- Process requirements that feel like overhead
- Prior decisions (from `kb context`) that conflict with your approach

**Why:** Working around constraints without surfacing them:
- Prevents the system from learning about recurring friction
- Bypasses stakeholders who should know about the limitation
- Creates hidden technical debt

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go)
2. **SET UP investigation file:** Run `kb create investigation comprehensive-audit-orch-go-bugs` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-comprehensive-audit-orch-go-bugs.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-audit-comprehensive-audit-orch-03jan/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input



## SKILL GUIDANCE (codebase-audit)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: codebase-audit
skill-type: procedure
description: Systematic codebase audit with configurable dimension (security/performance/tests/architecture/organizational/quick)
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 3e2577f6c227 -->
<!-- Source: worker/codebase-audit/.skillc -->
<!-- To modify: edit files in worker/codebase-audit/.skillc, then run: skillc build -->
<!-- Last compiled: 2025-12-24 07:51:19 -->

## Summary

**Use when the user says:**
- "Audit [focus area]" (security, performance, tests, architecture, organizational)
- "Run codebase health check"
- "Find [category] issues in the codebase"
- "Quick scan the codebase"

---

# Codebase Audit

<!-- SKILL-TEMPLATE: common-overview -->
<!-- Auto-generated from phases/common-overview.md -->

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

<!-- /SKILL-TEMPLATE -->

---

<!-- MODE-SPECIFIC CONTENT -->
<!-- Use --parallel flag for comprehensive multi-agent audits -->

<!-- SKILL-TEMPLATE: mode-parallel -->
<!-- Auto-generated from phases/mode-parallel.md -->

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

<!-- /SKILL-TEMPLATE -->

---

<!-- DIMENSION-SPECIFIC CONTENT -->
<!-- The build system will inject the appropriate dimension module here based on spawn configuration -->

<!-- For backward compatibility with old skill names, detect dimension from SPAWN_CONTEXT -->
<!-- If spawned as codebase-audit-security, auto-set dimension=security -->
<!-- If spawned as codebase-audit --dimension performance, use that -->

**Dimension-specific guidance below:**

---

<!-- SKILL-TEMPLATE: dimension-security -->
<!-- Auto-generated from phases/dimension-security.md -->

# Codebase Audit: Security

**TLDR:** Security-focused audit identifying vulnerabilities, unsafe patterns, secrets exposure, and OWASP compliance gaps.

**Status:** STUB - To be fleshed out when needed

**When to use:** Security review needed, penetration test prep, compliance audit, incident investigation

**Output:** Investigation file with security findings rated by severity (Critical/High/Medium/Low) with remediation steps

---

## Focus Areas (To be expanded)

1. **Secrets Exposure** - API keys, passwords, tokens in code/git history
2. **Injection Vulnerabilities** - SQL injection, command injection, XSS
3. **Authentication/Authorization** - Weak auth, missing access controls
4. **Cryptography** - Weak encryption, insecure random, poor key management
5. **Dependencies** - Known vulnerabilities in packages
6. **Input Validation** - Unsafe user input handling
7. **OWASP Top 10** - Compliance with OWASP security standards

---

## Pattern Search Commands (To be expanded)

```bash
# Secrets exposure
rg "password|secret|api_key|token|private_key" --type py --type js -i

# SQL injection
rg "execute\(.*%|\.format\(|f\".*FROM|f\".*WHERE" --type py

# Command injection
rg "subprocess\.call|os\.system|eval\(|exec\(" --type py

# XSS vulnerabilities
rg "innerHTML|dangerouslySetInnerHTML|\.html\(" --type js --type jsx

# Hardcoded credentials
rg "password\s*=\s*['\"]|api_key\s*=\s*['\"]" --type py --type js
```

---

*This skill stub establishes security audit structure. Expand with detailed workflow, severity ratings, and remediation patterns when security audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-performance -->
<!-- Auto-generated from phases/dimension-performance.md -->

# Codebase Audit: Performance

**TLDR:** Performance-focused audit identifying bottlenecks, algorithmic issues, inefficient queries, and optimization opportunities.

**Status:** STUB - To be fleshed out when needed

**When to use:** App is slow, high CPU/memory usage, scaling problems, response time issues

**Output:** Investigation file with performance findings, profiling data, and optimization recommendations with effort estimates

---

## Focus Areas (To be expanded)

1. **Algorithmic Complexity** - O(n²) loops, inefficient algorithms
2. **Database Queries** - N+1 queries, missing indexes, slow queries
3. **Resource Usage** - Memory leaks, excessive allocations
4. **I/O Operations** - Blocking I/O, unnecessary file reads
5. **Caching** - Missing caches, cache invalidation issues
6. **Concurrency** - Poor parallelization, lock contention

---

## Pattern Search Commands (To be expanded)

```bash
# Nested loops (potential O(n²))
rg "for.*:\s*\n.*for.*:" --type py -U

# N+1 query patterns
rg "\.all\(\)|\.filter\(" --type py -C 3 | rg "for.*in"

# Large files (potential complexity issues)
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -20

# TODO/FIXME about performance
rg "TODO.*performance|FIXME.*slow|HACK.*optimize" -i

# Blocking I/O in loops
rg "for.*:\s*\n.*open\(|for.*:\s*\n.*requests\." --type py -U
```

---

*This skill stub establishes performance audit structure. Expand with profiling methodology, optimization patterns, and benchmarking when performance audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-tests -->
<!-- Auto-generated from phases/dimension-tests.md -->

# Codebase Audit: Tests

**TLDR:** Testing-focused audit identifying coverage gaps, flaky tests, missing test types, and test quality issues.

**Status:** STUB - To be fleshed out when needed

**When to use:** Flaky CI builds, low confidence in tests, missing edge case coverage, test suite maintenance needed

**Output:** Investigation file with testing gaps, risk assessment, coverage metrics, and test improvement roadmap

---

## Focus Areas (To be expanded)

1. **Coverage Gaps** - Modules without tests, uncovered edge cases
2. **Flaky Tests** - Time-dependent, random, inconsistent results
3. **Missing Test Types** - Unit/integration/e2e gaps
4. **Test Quality** - No assertions, over-mocking, brittle tests
5. **Test Organization** - Poor structure, hard to maintain
6. **Test Performance** - Slow tests, inefficient setup/teardown

---

## Pattern Search Commands (To be expanded)

```bash
# Modules without test files
comm -23 <(find . -name "*.py" | grep -v test | sort) \
         <(find . -name "test_*.py" | sed 's/test_//' | sort)

# Flaky test indicators (sleep, random, time-based)
rg "sleep|time\.sleep|random\.|datetime\.now" tests/

# Tests without assertions
rg "def test_" tests/ -l | xargs rg "assert" -L

# Large test files (potential god test class)
find tests/ -name "*.py" | xargs wc -l | sort -rn | head -10

# Over-mocking indicators
rg "Mock|patch|MagicMock" tests/ -c | sort -rn | head -10
```

---

*This skill stub establishes testing audit structure. Expand with coverage analysis, flaky test patterns, and test quality metrics when testing audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-architecture -->
<!-- Auto-generated from phases/dimension-architecture.md -->

# Codebase Audit: Architecture

**TLDR:** Architecture-focused audit identifying coupling issues, god objects, missing abstractions, and modularity problems.

**Status:** STUB - To be fleshed out when needed

**When to use:** Hard to add features, tight coupling between modules, unclear boundaries, refactoring needed

**Output:** Investigation file with architectural issues, dependency analysis, and refactoring effort estimates

---

## Focus Areas (To be expanded)

1. **God Objects** - Classes/modules doing too much
2. **Tight Coupling** - Modules depending on too many others
3. **Missing Abstractions** - Repeated patterns not extracted
4. **Circular Dependencies** - Modules importing each other
5. **Poor Modularity** - Unclear boundaries, leaky abstractions
6. **Violation of SOLID Principles** - SRP, OCP, LSP, ISP, DIP violations

---

## Pattern Search Commands (To be expanded)

```bash
# God classes (many methods)
rg "^\s+def \w+\(self" --type py | uniq -c | sort -rn | head -10

# Tight coupling (many imports from one module)
rg "^from (\w+) import" --type py | cut -d' ' -f2 | sort | uniq -c | sort -rn

# Large files (potential god objects)
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -20

# Missing abstractions (switch/if-elif chains on type)
rg "if.*isinstance|if.*type\(.*\) ==" --type py -C 3

# Circular dependencies (imports at bottom of file)
rg "^from .* import" --type py | tail -20

# Deep nesting (complexity indicator)
rg "^\s{16,}(if|for|while|def)" --type py
```

---

*This skill stub establishes architecture audit structure. Expand with dependency analysis, refactoring patterns, and SOLID principles when architecture audit is needed.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-organizational -->
<!-- Auto-generated from phases/dimension-organizational.md -->

# Codebase Audit: Organizational Drift

**TLDR:** Systematic investigation of organizational drift - ROADMAP hygiene, artifact coherence, template consistency, process adherence. Produces prioritized recommendations with system amnesia root cause analysis.

**When to use:** Dylan says "audit organizational drift", "check ROADMAP hygiene", "find documentation drift", or when you suspect accumulated organizational debt.

**Output:** Investigation file with drift patterns, evidence, system amnesia analysis, and actionable fixes.

---

## Quick Reference

### Focus Areas

1. **ROADMAP Drift** - Completed work marked TODO, missing tasks, stale priorities
2. **Documentation Drift** - Reference docs vs operational templates out of sync
3. **Template Drift** - Workspace templates vs actual workspaces inconsistent
4. **State Duplication** - Same info in multiple places falling out of sync
5. **Context Boundary Leaks** - Manual sync points across contexts (code ↔ docs ↔ tracking)

### Process (4 Phases)

1. **Pattern Search** (15-30 min) - Use automated tools to find drift candidates
2. **Evidence Collection** (30-60 min) - Validate patterns, gather concrete examples
3. **System Amnesia Analysis** (15-30 min) - Identify which coherence principles violated
4. **Documentation** (30 min) - Write investigation with recommendations and fixes

### Key Deliverable

Investigation file at `.kb/investigations/YYYY-MM-DD-audit-organizational-drift.md` with:
- **Status:** Complete
- **Root Cause:** Drift patterns with system amnesia analysis
- **Recommendations:** Prioritized fixes (forcing functions, automation, validation)

---

## Detailed Workflow

### Phase 1: Pattern Search (15-30 minutes)

**Use automated tools to find drift candidates:**

#### ROADMAP Drift Patterns

```bash
# Compare ROADMAP entries against recent git commits
cd ~/meta-orchestration
git log --oneline --since="30 days ago" | rg "feat:|fix:" | head -20
# Manually compare against docs/ROADMAP.org TODO items

# Find DONE items without completion metadata
rg "^\*\* DONE" docs/ROADMAP.org -A 5 | rg -v "CLOSED:|:Completed:"

# Find completed agents not in ROADMAP
orch history | rg "Completed" | head -10
# Check if these appear in ROADMAP
```

#### Template Drift Patterns

```bash
# Find workspaces missing new template fields
rg "^# Workspace:" .orch/workspace/ -l | while read ws; do
  grep -q "Session Scope" "$ws" || echo "MISSING SESSION SCOPE: $ws"
  grep -q "Checkpoint Strategy" "$ws" || echo "MISSING CHECKPOINT STRATEGY: $ws"
done

# Compare workspace template against actual workspaces
diff -u ~/.orch/templates/workspace/WORKSPACE.md \
        .orch/workspace/latest-workspace/WORKSPACE.md | head -50
```

#### Documentation Drift Patterns

```bash
# Find orch commands in code but not in operational templates
rg "def (spawn|check|status|complete|resume|send)" tools/orch/cli.py -o | \
  cut -d' ' -f2 | while read cmd; do
    grep -q "$cmd" ~/.orch/templates/orchestrator/orch-commands.md || \
      echo "MISSING IN TEMPLATE: $cmd"
  done

# Find features documented but not in reference docs
rg "orch \w+" ~/.orch/templates/orchestrator/ -o | sort -u > /tmp/template_cmds
rg "^###? orch" tools/README.md -o | sort -u > /tmp/readme_cmds
comm -23 /tmp/template_cmds /tmp/readme_cmds
```

#### Manual Sync Points (Fragile Patterns)

```bash
# Find "remember to" or "don't forget" instructions
rg "remember to|don't forget|make sure to update" docs/ --type md -i

# Find TODO comments about updating related files
rg "TODO.*update|FIXME.*sync" --type py --type md -C 2
```

#### State Duplication

```bash
# Find status/phase duplicated across systems
rg "status.*=.*(active|completed|paused)" --type py -l | \
  xargs rg "Phase.*=.*(Active|Complete|Paused)" -l

# Find completion timestamps in multiple places
rg "completed_at|completion_time|finished_at" --type py --type json
```

**Document all search commands in investigation file** (reproducibility)

---

### Phase 2: Evidence Collection (30-60 minutes)

**For each pattern found, gather concrete evidence:**

#### Evidence Standards

**For ROADMAP drift:**
- Specific ROADMAP entry + corresponding git commit showing drift
- Date completed vs date still showing as TODO
- Count of drift instances (how pervasive?)
- User impact (does this affect planning/prioritization?)

**For documentation drift:**
- Specific inaccuracy (what docs say vs what code does)
- File paths showing divergence
- When drift introduced (git blame to find when docs last updated)
- Impact (who's affected by stale docs - orchestrators, developers, both?)

**For template drift:**
- Specific workspace missing field + template showing field should exist
- Date workspace created vs date template updated
- Migration effort (how many workspaces need updating?)
- Graceful degradation check (does code handle missing fields?)

**For state duplication:**
- Concrete example showing same state in multiple files
- Which is source of truth? (or neither?)
- Instances where states diverged
- Proposed fix (derive, don't duplicate)

**For manual sync points:**
- Specific "remember to" instruction in docs
- Evidence of sync failures (times this was forgotten)
- Automation opportunity (can this be enforced?)

#### Investigation File Structure

```markdown
# Investigation: Organizational Drift Audit

**Date:** YYYY-MM-DD
**Status:** Complete
**Investigator:** Claude (codebase-audit-organizational skill)
**Trigger:** [Dylan's request or suspected drift]

---

## TLDR

**Key findings:** [2-3 sentence summary of major drift patterns]
**Highest priority:** [Top recommendation with ROI]
**Total drift instances:** [Count across all categories]

---

## Scope

**Focus areas:** Organizational drift (ROADMAP, docs, templates, state duplication)
**Boundaries:** [Project-specific or global artifacts?]
**Time spent:** [Actual time for audit]

---

## Findings by Category

### ROADMAP Drift (Priority: High/Medium/Low)

**Pattern:** [Name of drift pattern found]

**Evidence:**
- Instance 1: ROADMAP entry "Task X" marked TODO, git commit abc123 completed 2025-11-10
- Instance 2: [...]
- Total instances: [count]

**Metrics:**
- Tasks completed but not marked DONE: [count]
- Tasks missing completion metadata: [count]
- Average drift age: [days between completion and discovery]

**Impact:** [How this affects planning/orchestration]

**Recommendation:** [Specific fix with automation approach]

**ROI:** [Value gained / time invested]

---

### [Other categories following same structure]

---

## System Amnesia Analysis

**See:** `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`

**Coherence principles violated:**
- [ ] Single Source of Truth - [Example showing duplication]
- [ ] Automatic Loop Closure - [Example showing manual step]
- [ ] Cross-Boundary Coherence - [Example showing context switch failure]
- [ ] Observable Drift - [Example showing silent drift]
- [ ] Forcing Functions at Creation - [Example showing optional field]

**Common failures observed:**
- [ ] ROADMAP Drift - [X instances, root cause: manual ROADMAP updates]
- [ ] Documentation Drift - [X instances, root cause: template not rebuilt]
- [ ] Template Drift - [X instances, root cause: no migration mechanism]
- [ ] State Duplication - [X instances, root cause: derived state manual]
- [ ] Context Boundary Leaks - [X instances, root cause: no cross-project search]

**Design pattern recommendations:**
- Use "Derive, Don't Duplicate" for [specific case - e.g., registry status from workspace Phase]
- Add "Validation at Boundaries" for [specific workflow - e.g., orch complete checks Phase]
- Implement "Build Systems for Consistency" for [specific docs - e.g., template rebuild automation]
- Add "Forcing Functions" for [specific creation - e.g., ROADMAP requires task-id]

---

## Prioritization

**High Priority (fix now):**
1. [Issue] - Blocking orchestration, high impact, low effort
2. [Issue] - Data loss risk, silent failures

**Medium Priority (schedule soon):**
1. [Issue] - Maintenance burden, moderate effort
2. [Issue] - Developer experience impact

**Low Priority (backlog):**
1. [Issue] - Minor improvement, can defer
2. [Issue] - Nice-to-have, low impact

---

## Recommendations

**Immediate actions (this week):**
- [ ] [Specific task with owner and approach]
  - **Fix:** [What to do]
  - **Automation:** [How to prevent recurrence]
  - **Effort:** [Hours estimated]

**Short-term (this month):**
- [ ] [Planned fix with scope]

**Long-term (next quarter):**
- [ ] [Strategic improvement with ROI]

---

## Reproducibility

**Commands used for pattern search:**
```bash
# Document all grep/rg/find/diff commands used
# This allows re-running audit in future to measure improvement
```

**Metrics baseline:**
- Total ROADMAP entries: [count]
- ROADMAP drift instances: [count]
- Template drift instances: [count]
- Documentation drift instances: [count]
- State duplication instances: [count]
- Manual sync points: [count]

**Re-audit schedule:** 3 months (measure drift reduction)

---

## Related Work

- Decision: `.kb/decisions/2025-11-15-system-amnesia-as-design-constraint.md`
- Checklist: `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`
- Investigation: [Link to related organizational investigations]

---

## Next Steps

1. **Discuss findings with Dylan** (present prioritization, get approval)
2. **Add high-priority items to ROADMAP** (with effort estimates)
3. **Spawn agents for fixes** (if Dylan approves immediate action)
4. **Schedule re-audit** (3 months to measure improvement)
```

---

### Phase 3: System Amnesia Analysis (15-30 minutes)

**Identify which coherence principles were violated for each drift pattern:**

**Checklist for each finding:**

1. **Single Source of Truth** - Is there duplicate state? Which is authoritative?
2. **Automatic Loop Closure** - Does workflow require manual step to complete?
3. **Cross-Boundary Coherence** - Do updates span contexts (code/docs/tracking)?
4. **Observable Drift** - Was drift silent until manual inspection?
5. **Forcing Functions at Creation** - Could invalid state be created?

**For each violation, propose design pattern:**

| Violation | Pattern | Example Fix |
|-----------|---------|-------------|
| Duplicate state | Derive, Don't Duplicate | Registry status derived from workspace Phase |
| Manual loop closure | Atomic Multi-Context Updates | `orch complete` updates all systems |
| Silent drift | Validation at Boundaries | `orch complete` checks workspace Phase |
| No forcing function | Build Systems for Consistency | Template rebuild on SessionStart hook |

**Root cause categories:**
- **Return trip tax** - Easy to create, hard to remember to update
- **Context switching** - Update happens in different session/context
- **No single source of truth** - Multiple systems maintain same state
- **Manual sync points** - "Remember to" instructions
- **No observability** - Drift accumulates silently

---

### Phase 4: Documentation (30 minutes)

**Write investigation file following template above**

**Critical sections:**
- ✅ TLDR with key findings and top priority
- ✅ Evidence section with concrete examples (file paths, commit shas, counts)
- ✅ System Amnesia Analysis (which principles violated, proposed fixes)
- ✅ Prioritization using ROI framework (impact vs effort)
- ✅ Recommendations with specific, actionable tasks
- ✅ Reproducibility section with commands and baseline metrics

**Present findings to Dylan:**
- "Organizational drift audit complete. Key findings: [TLDR]"
- "Highest priority: [Top item with ROI]"
- "System amnesia root causes: [Top 2-3 principles violated]"
- "Would you like me to add high-priority items to ROADMAP or spawn agents to address them?"

---

## Anti-Patterns to Avoid

**❌ Audit without concrete examples**
- "ROADMAP has drift issues" (vague, not actionable)
✅ **Fix:** "12 tasks completed but marked TODO: Task X (commit abc123, completed 2025-11-10), Task Y (commit def456, completed 2025-11-09), ..."

**❌ No system amnesia analysis**
- Lists drift but doesn't identify root cause or prevention
✅ **Fix:** Map each finding to violated coherence principle, propose forcing function

**❌ No reproducibility**
- Can't re-run audit to measure improvement
✅ **Fix:** Document all commands + baseline metrics

**❌ Recommendations too vague**
- "Fix ROADMAP drift" (what does that mean?)
✅ **Fix:** "Add `orch complete` auto-update: read workspace task-id field, mark ROADMAP entry DONE"

**❌ No prioritization**
- Dylan doesn't know what to fix first
✅ **Fix:** Use ROI framework (impact vs effort matrix)

---

## Related Documentation

- **System amnesia patterns:** `~/meta-orchestration/docs/amnesia-compensation-checklist.md#system-level-amnesia-resilience`
- **Investigation template:** `.orch/templates/INVESTIGATION.md`
- **ROADMAP management:** `docs/work-prioritization.md`
- **Template build system:** `.kb/decisions/2025-11-14-orchestrator-restructuring-template-build-system.md`

---

## Example Usage

**Dylan:** "audit organizational drift in meta-orchestration"

**You:**
1. Create investigation file: `.kb/investigations/2025-11-15-organizational-drift-audit.md`
2. Run pattern search commands (ROADMAP drift, template drift, docs drift)
3. Collect evidence (12 ROADMAP drift instances, 5 template drift instances, 3 doc drift instances)
4. System amnesia analysis (violated: Automatic Loop Closure, Observable Drift)
5. Prioritize using ROI framework
6. Write investigation file with recommendations
7. Present: "Audit complete. Found 20 drift instances across 3 categories. Highest priority: Fix `orch complete` to auto-update ROADMAP (violates Automatic Loop Closure - easy fix, high impact). Add to ROADMAP?"

---

*This skill enables systematic, evidence-based organizational drift assessment with system amnesia root cause analysis and actionable recommendations.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: dimension-quick -->
<!-- Auto-generated from phases/dimension-quick.md -->

# Codebase Audit: Quick Scan

**TLDR:** 1-hour automated health check across all audit areas. Returns top 10 high-priority findings with quick-win recommendations.

**When to use:** Need rapid health check before major work, onboarding to new codebase, monthly health monitoring, or before deciding which focused audit to run.

**Output:** Investigation file with top findings across all categories, sorted by ROI.

---

## Quick Reference

### Scan Areas (All Categories)

1. **Security** - Secrets, unsafe patterns, SQL injection, XSS
2. **Performance** - Large files, complex functions, N+1 queries
3. **Tests** - Missing tests, coverage gaps, flaky indicators
4. **Architecture** - God objects, tight coupling, missing abstractions
5. **Organizational** - ROADMAP drift, template drift, doc drift

### Process (30-60 minutes)

1. **Automated Scan** (30 min) - Run all pattern search commands
2. **Triage** (15 min) - Filter to top 10 by severity/effort
3. **Document** (15 min) - Write investigation with findings

### Deliverable

Investigation file: `.kb/investigations/YYYY-MM-DD-audit-quick-scan.md`
- Top 10 findings sorted by ROI
- Recommended next steps (which focused audit to run?)

---

## Workflow

### Step 1: Automated Scan (30 minutes)

**Run these commands and capture counts:**

```bash
# Security patterns
echo "=== SECURITY ===" >> /tmp/audit.txt
rg "password|secret|api_key|token" --type py --type js -i | wc -l >> /tmp/audit.txt
rg "eval\(|exec\(|__import__|subprocess\.call" --type py | wc -l >> /tmp/audit.txt

# Performance patterns
echo "=== PERFORMANCE ===" >> /tmp/audit.txt
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -10 >> /tmp/audit.txt
rg "TODO.*performance|FIXME.*slow" -i | wc -l >> /tmp/audit.txt

# Testing patterns
echo "=== TESTS ===" >> /tmp/audit.txt
comm -23 <(find . -name "*.py" | grep -v test | sort) \
         <(find . -name "test_*.py" | sed 's/test_//' | sort) | wc -l >> /tmp/audit.txt
rg "sleep|time\.sleep|random\." tests/ | wc -l >> /tmp/audit.txt

# Architecture patterns
echo "=== ARCHITECTURE ===" >> /tmp/audit.txt
rg "^\s+def \w+\(self" --type py | uniq -c | sort -rn | head -5 >> /tmp/audit.txt
rg "^from|^import" --type py | cut -d' ' -f2 | sort | uniq -c | sort -rn | head -5 >> /tmp/audit.txt

# Organizational patterns
echo "=== ORGANIZATIONAL ===" >> /tmp/audit.txt
git log --since="30 days ago" --oneline | grep -E "feat:|fix:" | wc -l >> /tmp/audit.txt
rg "remember to|don't forget" docs/ -i | wc -l >> /tmp/audit.txt
```

**Review `/tmp/audit.txt` for high counts indicating issues**

---

### Step 2: Triage (15 minutes)

**From scan results, identify top 10 by severity:**

**Severity matrix:**
- **Critical** - Security vulnerabilities, data loss risk, production blockers
- **High** - Blocking development, significant performance impact, major tech debt
- **Medium** - Maintenance burden, developer experience, moderate risk
- **Low** - Minor improvement, cosmetic, low risk

**Effort estimation:**
- **Quick win** (<4h) - Rename, add docs, simple refactor
- **Medium** (4-16h) - Extract classes, add tests, fix duplication
- **Large** (>16h) - Architectural changes, large-scale refactoring

**Top 10 = Highest severity + Lowest effort (ROI = Severity / Effort)**

---

### Step 3: Document (15 minutes)

**Investigation file structure:**

```markdown
# Investigation: Quick Audit Scan

**Date:** YYYY-MM-DD
**Status:** Complete
**Investigator:** Claude (codebase-audit-quick skill)
**Scan Duration:** [X minutes]

---

## TLDR

**Top 10 findings identified** across security, performance, tests, architecture, organizational

**Recommended next step:** Run focused audit for [category with most high-severity findings]

**Quick wins available:** [Count of findings with <4h effort]

---

## Top 10 Findings (Sorted by ROI)

### 1. [Finding Name] (Severity: Critical/High/Medium, Effort: <4h/4-16h/>16h)

**Category:** Security/Performance/Tests/Architecture/Organizational

**Issue:** [One sentence describing the problem]

**Evidence:** [Quick pointer - file path, line count, or command showing issue]

**Impact:** [Why this matters]

**Quick fix:** [What to do - 1-2 sentences]

**ROI:** High/Medium/Low

---

### 2-10. [Following same structure]

---

## Scan Summary

**Total patterns scanned:** 15+ automated searches

**Findings by category:**
- Security: [count] potential issues
- Performance: [count] potential issues
- Tests: [count] potential issues
- Architecture: [count] potential issues
- Organizational: [count] potential issues

**Baseline metrics:**
- Total files: [count]
- Total lines: [count]
- Largest file: [path] ([lines] lines)
- Test coverage: [X modules without tests]
- ROADMAP drift: [X completed but marked TODO]

---

## Recommended Next Steps

**Immediate actions (quick wins <4h):**
- [ ] [Finding #X] - [Quick fix]

**Focused audits needed:**
- [ ] Run `codebase-audit-[category]` for [specific area with most critical findings]
- [ ] Run `codebase-audit-[category]` for [second priority area]

**Schedule:**
- This week: Address quick wins
- Next week: Run focused audit for [highest priority category]
- Next month: Re-run quick scan to measure improvement

---

## Reproducibility

**Commands to re-run scan:**
See Step 1 automated scan commands above.

**Re-scan schedule:** Monthly (track trend over time)
```

---

## Usage Notes

**When to use quick scan:**
- ✅ Monthly health monitoring
- ✅ Before starting major work (identify risks)
- ✅ Onboarding to unfamiliar codebase
- ✅ Deciding which focused audit to run

**When NOT to use quick scan:**
- ❌ You know the problem area (use focused audit instead)
- ❌ Need deep analysis (quick scan is surface-level)
- ❌ Investigation requires manual code reading

**Follow-up workflow:**
1. Run quick scan
2. Identify category with most critical findings
3. Run focused audit: `codebase-audit-[category]`
4. Address high-priority findings
5. Re-run quick scan in 1 month to measure improvement

---

## Anti-Patterns

**❌ Treating quick scan as comprehensive**
- Quick scan is triage, not deep analysis
✅ **Fix:** Use focused audits for thorough investigation

**❌ No follow-up action**
- Running scan without addressing findings
✅ **Fix:** Always identify at least one quick win to fix immediately

**❌ No baseline tracking**
- Can't measure improvement over time
✅ **Fix:** Re-run monthly, track metrics trend

---

*This skill provides rapid health check across all audit areas, enabling quick triage and informed decision on which focused audit to run next.*

<!-- /SKILL-TEMPLATE -->

---

<!-- SKILL-TEMPLATE: self-review -->
<!-- Auto-generated from phases/self-review.md -->

# Self-Review (Mandatory)

Before completing the audit, verify quality of findings and recommendations.

---

## Audit-Specific Checks

| Check | Verification | If Failed |
|-------|--------------|-----------|
| **Evidence concrete** | Each finding has file:line reference | Add specific locations |
| **Reproducible** | Pattern searches documented | Add grep/glob commands used |
| **Prioritized** | Recommendations ranked by ROI | Add severity/effort matrix |
| **Actionable** | Each recommendation has clear next step | Make specific |
| **Baseline captured** | Metrics for re-audit comparison | Add counts/percentages |

---

## Self-Review Checklist

### 1. Findings Quality

- [ ] **Each finding has evidence** - Concrete file:line references, not "there are issues"
- [ ] **Pattern searches documented** - grep/glob commands that found issues
- [ ] **False positives filtered** - Reviewed results, removed non-issues
- [ ] **Severity assessed** - Each finding has impact level (critical/high/medium/low)

### 2. Recommendations Quality

- [ ] **Prioritized by ROI** - High impact, low effort items first
- [ ] **Actionable** - Each recommendation specifies what to do
- [ ] **Scoped** - Recommendations are achievable (not "rewrite everything")
- [ ] **Linked to findings** - Each recommendation traces to specific findings

### 3. Documentation Quality

- [ ] **Investigation file complete** - All sections filled
- [ ] **Baseline metrics** - Numbers for future comparison
- [ ] **Reproduction commands** - Someone can re-run the audit
- [ ] **NOT DONE claims verified** - For each 'NOT DONE' or 'NOT IMPLEMENTED' finding, confirmed with file/code search (not just artifact reading)

### 4. Commit Hygiene

- [ ] Conventional format (`audit:` or `chore:`)
- [ ] Investigation file committed

### 5. Discovered Work Check

*Audits typically discover actionable work. Track it in beads so it doesn't get lost.*

| Type | Examples | Action |
|------|----------|--------|
| **Security bugs** | Vulnerabilities, injection risks | `bd create "SECURITY: description" --type bug` |
| **Architecture issues** | God objects, tight coupling, tech debt | `bd create "ARCHITECTURE: description" --type task` |
| **Performance issues** | N+1 queries, missing indexes | `bd create "PERFORMANCE: description" --type bug` |
| **Missing tests** | Coverage gaps, critical paths untested | `bd create "TESTING: description" --type task` |

**Triage labeling for daemon processing:**

After creating issues, apply triage labels based on finding severity:

| Severity | Label | When to use |
|----------|-------|-------------|
| Critical/High | `triage:ready` | Clear problem, known fix approach, well-scoped |
| Medium/Low | `triage:review` | Needs orchestrator review before work starts |

Example:
```bash
bd create "SECURITY: SQL injection in api.py:123" --type bug
bd label <issue-id> triage:ready  # Critical severity, clear fix
kn decide "Use optimistic locking for updates" --reason "Prevents lost updates without blocking reads"
kn tried "Pessimistic locking" --failed "Caused deadlocks under high concurrency"
```

**Good externalization after debugging:**
```bash
kn constrain "Cache invalidation requires explicit call" --reason "TTL alone causes stale reads"
```

**Good externalization after investigation:**
```bash
kn question "Is the legacy API still used? Found no callers but unclear if external consumers exist"
```

---

## Completion Criteria (Leave it Better)

- [ ] Reflected on what was learned during the session
- [ ] Ran at least one `kn` command OR documented why nothing to externalize
- [ ] Included "Leave it Better" status in completion comment

**Only proceed to final completion after Leave it Better is done.**

<!-- /SKILL-TEMPLATE -->



---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `/exit`


⚠️ Your work is NOT complete until you run these commands.
