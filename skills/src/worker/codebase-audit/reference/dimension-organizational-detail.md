# Organizational Drift Audit (Detailed Reference)

## Detailed Workflow

### Phase 1: Pattern Search (15-30 minutes)

#### ROADMAP Drift Patterns

```bash
git log --oneline --since="30 days ago" | rg "feat:|fix:" | head -20
orch history | rg "Completed" | head -10
bd list --status=closed | head -10
```

#### Template Drift Patterns

```bash
rg "^# Workspace:" .orch/workspace/ -l | while read ws; do
  grep -q "Session Scope" "$ws" || echo "MISSING SESSION SCOPE: $ws"
  grep -q "Checkpoint Strategy" "$ws" || echo "MISSING CHECKPOINT STRATEGY: $ws"
done
```

#### Documentation Drift Patterns

```bash
orch --help 2>&1 | rg "^\s+\w+" | awk '{print $1}' | while read cmd; do
    grep -rq "$cmd" ~/.claude/skills/meta/orchestrator/ || \
      echo "MISSING IN SKILL: $cmd"
  done
```

#### Manual Sync Points

```bash
rg "remember to|don't forget|make sure to update" docs/ --type md -i
rg "TODO.*update|FIXME.*sync" --type py --type md -C 2
```

### Phase 2: Evidence Collection (30-60 minutes)

**For ROADMAP drift:** Specific entry + git commit showing drift, date, count, user impact
**For documentation drift:** What docs say vs what code does, file paths, git blame, impact
**For template drift:** Workspace missing field + template showing field, date, migration effort
**For state duplication:** Same state in multiple files, which is source of truth
**For manual sync points:** "Remember to" instruction + evidence of sync failures

### Phase 3: System Amnesia Analysis (15-30 minutes)

Checklist for each finding:
1. **Single Source of Truth** - Is there duplicate state?
2. **Automatic Loop Closure** - Does workflow require manual step?
3. **Cross-Boundary Coherence** - Do updates span contexts?
4. **Observable Drift** - Was drift silent until manual inspection?
5. **Forcing Functions at Creation** - Could invalid state be created?

| Violation | Pattern | Example Fix |
|-----------|---------|-------------|
| Duplicate state | Derive, Don't Duplicate | Registry status derived from workspace Phase |
| Manual loop closure | Atomic Multi-Context Updates | `orch complete` updates all systems |
| Silent drift | Validation at Boundaries | `orch complete` checks workspace Phase |
| No forcing function | Build Systems for Consistency | Template rebuild on SessionStart hook |

### Phase 4: Documentation (30 minutes)

Write investigation file with TLDR, evidence, system amnesia analysis, prioritization, recommendations, reproducibility metrics.

## Investigation File Template

```markdown
# Investigation: Organizational Drift Audit

**Date:** YYYY-MM-DD
**Status:** Complete

## TLDR
**Key findings:** [2-3 sentence summary]
**Highest priority:** [Top recommendation with ROI]
**Total drift instances:** [Count]

## Findings by Category

### ROADMAP Drift (Priority: High/Medium/Low)
**Pattern:** [Name]
**Evidence:** [Concrete instances with git commits]
**Metrics:** [Counts, averages]
**Impact:** [How this affects planning]
**Recommendation:** [Specific fix]

## System Amnesia Analysis
- [ ] Single Source of Truth
- [ ] Automatic Loop Closure
- [ ] Cross-Boundary Coherence
- [ ] Observable Drift
- [ ] Forcing Functions at Creation

## Prioritization
**High Priority:** [Blocking, high impact, low effort]
**Medium Priority:** [Moderate effort]
**Low Priority:** [Minor, can defer]

## Reproducibility
[Commands used + baseline metrics]
```

## Related

- `.kb/guides/resilient-infrastructure-patterns.md`
