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
