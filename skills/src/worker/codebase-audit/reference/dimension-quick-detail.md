# Quick Scan (Detailed Reference)

## Full Automated Scan Commands

```bash
# Security patterns
echo "=== SECURITY ==="
rg "password|secret|api_key|token" --type py --type js -i | wc -l
rg "eval\(|exec\(|__import__|subprocess\.call" --type py | wc -l

# Performance patterns
echo "=== PERFORMANCE ==="
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -10
rg "TODO.*performance|FIXME.*slow" -i | wc -l

# Testing patterns
echo "=== TESTS ==="
comm -23 <(find . -name "*.py" | grep -v test | sort) \
         <(find . -name "test_*.py" | sed 's/test_//' | sort) | wc -l
rg "sleep|time\.sleep|random\." tests/ | wc -l

# Architecture patterns
echo "=== ARCHITECTURE ==="
rg "^\s+def \w+\(self" --type py | uniq -c | sort -rn | head -5
rg "^from|^import" --type py | cut -d' ' -f2 | sort | uniq -c | sort -rn | head -5

# Organizational patterns
echo "=== ORGANIZATIONAL ==="
git log --since="30 days ago" --oneline | grep -E "feat:|fix:" | wc -l
rg "remember to|don't forget" docs/ -i | wc -l

# Accessibility patterns
echo "=== ACCESSIBILITY ==="
rg "<img " --glob "*.{svelte,jsx,tsx,html}" | rg -v "alt=" | wc -l
rg "onClick|on:click" --glob "*.{svelte,jsx,tsx}" -C 1 | rg "<(div|span)" | wc -l
rg "tabIndex=\"[1-9]|tabindex=\"[1-9]" --glob "*.{svelte,jsx,tsx,html}" | wc -l
```

## Triage Matrix

| Severity | Definition |
|----------|------------|
| Critical | Security vulnerabilities, data loss risk, production blockers |
| High | Blocking development, significant performance impact, major tech debt |
| Medium | Maintenance burden, developer experience, moderate risk |
| Low | Minor improvement, cosmetic, low risk |

| Effort | Definition |
|--------|------------|
| Quick win | <4h — rename, add docs, simple refactor |
| Medium | 4-16h — extract classes, add tests, fix duplication |
| Large | >16h — architectural changes, large-scale refactoring |

**Top 10 = Highest severity + Lowest effort (ROI = Severity / Effort)**

## Investigation File Template

```markdown
# Investigation: Quick Audit Scan

**Date:** YYYY-MM-DD
**Status:** Complete
**Scan Duration:** [X minutes]

## TLDR
**Top 10 findings identified** across all categories.
**Recommended next step:** Run focused audit for [category with most critical findings]
**Quick wins available:** [Count of <4h effort findings]

## Top 10 Findings (Sorted by ROI)

### 1. [Finding Name] (Severity: ..., Effort: ...)
**Category:** Security/Performance/Tests/Architecture/Organizational/Accessibility
**Issue:** [One sentence]
**Evidence:** [File path, count, or command]
**Quick fix:** [1-2 sentences]

## Scan Summary
- Security: [count] potential issues
- Performance: [count] potential issues
- Tests: [count] potential issues
- Architecture: [count] potential issues
- Organizational: [count] potential issues
- Accessibility: [count] potential issues

## Baseline Metrics
- Total files/lines, largest file, test coverage gaps, drift counts

## Recommended Next Steps
1. Address quick wins (<4h)
2. Run focused audit for highest-priority category
3. Re-scan in 1 month
```
