# Self-Review Phase

**Purpose:** Quality gate before completion. Catch anti-patterns, verify commit hygiene, ensure deliverables are complete.

**When you're in this phase:** Implementation and validation are done. Before marking complete, review your own work against quality standards.

---

## Why Self-Review Matters

Agents often mark work "complete" with:
- God objects introduced (files doing too much)
- Incomplete implementations (TODOs, placeholders)
- Poor commit hygiene (WIP commits, wrong types)
- Missing test coverage for edge cases
- Security issues (hardcoded secrets, injection vulnerabilities)
- **Orphaned code (components/functions that exist but aren't wired in)**
- **Demo/placeholder data that should be real data**

Self-review catches these before orchestrator sees them.

---

## Self-Review Checklist

**Perform each check. Document significant findings via `bd comment`.**

### 0. Original Symptom Validation (For Bug Fixes)

**If your work is fixing a bug reported in the original issue:**

| Check | How | If Failed |
|-------|-----|-----------|
| **Original symptom identified** | Find the exact command/scenario that demonstrates the bug in the issue | Document what was originally failing |
| **Original symptom re-tested** | Run the EXACT same command/scenario (same flags, same mode) | STOP - fix is incomplete |
| **Result documented** | Report actual result with timing/output via `bd comment` | Document evidence before claiming complete |

**⚠️ CRITICAL: Scope Redefinition Warning**

Agents can claim "fix complete" by silently testing a different scenario than the original issue:
- Original issue: `time orch status` (1m25s) → Agent tests `orch status --json` (1s) → Claims "65x faster"
- Original issue: Error on invalid input → Agent tests valid input → Claims "works now"

**Before claiming complete, verify:**
- [ ] I tested the EXACT command/scenario from the original issue
- [ ] If I tested a different mode/flag, I explicitly documented why and ALSO tested the original
- [ ] My "after" measurement uses the same method as the "before" (no ~estimates, actual measurements)

**Examples:**
```bash
# Original issue shows: "time orch status # 1:25.67 total"
# WRONG: time orch status --json  # Testing different mode!
# RIGHT: time orch status          # Same as original

# Original issue shows: "API returns 500 on POST /users with empty name"
# WRONG: Test POST /users with valid name  # Different scenario!
# RIGHT: Test POST /users with empty name  # Same as original
```

**Why this matters:** Testing a different scenario than the original issue enables agents to rationalize partial fixes as complete. The fix is only verified when the original failing scenario passes.

**Skip this section if:** Your work is a new feature (not fixing existing behavior).

---

### 1. Scope Verification (For Refactoring/Migration Work)

**If your work involved renaming, refactoring, migrating, or changing patterns across the codebase:**

| Check | How | If Failed |
|-------|-----|-----------|
| **Scope was determined** | Ran `rg "old_pattern"` before starting to count all occurrences | Document scope retroactively, verify you found them all |
| **All instances updated** | Run `rg "old_pattern"` now - should return 0 matches | Find and fix remaining instances |
| **New pattern consistent** | Run `rg "new_pattern"` - count matches expected scope | Investigate mismatches |

**Examples:**
```bash
# Renaming a function
rg "oldFunctionName" --type py  # Should be 0
rg "newFunctionName" --type py  # Should match expected count

# Migrating path from .orch/ to .kb/
rg "\.orch/investigations" --type py  # Should be 0
rg "\.kb/investigations" --type py    # Should show new paths

# Updating config pattern
rg "old_config_key" --type yaml  # Should be 0
```

**Why this matters:** Partial migrations are worse than no migration - they create inconsistent state that's hard to debug later.

**Skip this section if:** Your work was purely additive (new files, new functions) with no changes to existing patterns.

---

### 2. Anti-Pattern Detection

Review your changes for common anti-patterns:

| Anti-Pattern | How to Check | If Found |
|--------------|--------------|----------|
| **God objects** | Any file >300 lines or doing multiple concerns? | Extract responsibilities |
| **Tight coupling** | Components directly instantiating dependencies? | Use dependency injection |
| **Magic values** | Hardcoded numbers/strings without explanation? | Extract to named constants |
| **Deep nesting** | Logic nested >3 levels? | Extract to helper functions |
| **Incomplete work** | Any TODO, FIXME, placeholder comments? | Complete or document as known limitation |

### 3. Security Review

Check for common security issues:

- [ ] No hardcoded secrets (API keys, passwords, tokens)
- [ ] No SQL injection vulnerabilities (use parameterized queries)
- [ ] No XSS vulnerabilities (escape user input in output)
- [ ] No path traversal (validate file paths)
- [ ] No command injection (avoid shell commands with user input)

**If security issue found:** Fix immediately. Do not proceed.

### 4. Commit Hygiene

Review your commits:

```bash
git log --oneline -10
```

| Check | Standard | If Violated |
|-------|----------|-------------|
| **Conventional format** | `type: description` (feat, fix, refactor, test, docs, chore) | Amend or squash |
| **Atomic commits** | Each commit is one logical change | Squash related commits |
| **No WIP commits** | No "WIP", "temp", "fix typo" commits in history | Squash into meaningful commits |
| **Test/impl separation** | Test commits separate from implementation (TDD mode) | OK to have together if not TDD |

### 5. Test Coverage

Review test adequacy:

- [ ] Happy path tested (main functionality works)
- [ ] Edge cases covered (empty input, boundaries, nulls)
- [ ] Error paths tested (what happens when things fail)
- [ ] No test gaps for new code (every new function has test)

**For TDD mode:** You should already have this covered. Verify.

**For direct mode:** Verify existing tests still pass, no new behavioral code added without tests.

### 6. Documentation Check

- [ ] Public APIs have clear signatures (types, return values)
- [ ] Complex logic has inline comments explaining "why"
- [ ] No commented-out code left behind
- [ ] No debug statements (console.log, print, debugger)

### 7. Deliverables Verification

Cross-check against SPAWN_CONTEXT requirements:

- [ ] All required deliverables exist (investigation, design, tests, implementation as applicable)
- [ ] Deliverables are complete (not stubs or placeholders)
- [ ] Deliverables reported via `bd comment` with paths/summary

### 8. Integration Wiring Check (CRITICAL)

**New code MUST be wired into the system, not just exist in isolation.**

Components that exist but aren't connected are worse than no implementation - they create false confidence that work is done.

| Check | How | If Failed |
|-------|-----|-----------|
| **New modules imported** | Search for imports of your new files (`rg "import.*new-file"` or `rg "require.*new-file"`) | Wire into consuming code or delete orphaned file |
| **New functions called** | Search for calls to new functions (`rg "newFunctionName\("`) | Add calls or delete unused functions |
| **New exports used** | Check that exported symbols are imported elsewhere | Remove unused exports or wire them in |
| **New routes registered** | If adding endpoints, verify they appear in route registration | Register routes in app/router |
| **New components rendered** | If adding UI components, verify they're rendered somewhere | Add to parent component or page |
| **New config referenced** | If adding config options, verify they're read somewhere | Wire config into code that uses it |

**Examples:**
```bash
# New React component - verify it's rendered
rg "import.*NewComponent" --type tsx  # Should find at least one import
rg "<NewComponent" --type tsx          # Should find at least one render

# New API endpoint - verify it's registered
rg "router\.(get|post|put|delete).*\/new-endpoint"  # Should find registration

# New utility function - verify it's called
rg "newUtilFunction\(" --type ts  # Should find at least one call
```

**Why this matters:** "Code exists" ≠ "Feature works". A component that isn't wired in does nothing. Tests may even pass because the dead code path is never executed.

**Red flags (STOP and fix):**
- New file with 0 imports elsewhere
- New export with 0 consumers
- New route handler not in route registry
- New component not rendered anywhere
- New hook not called anywhere

### 9. Demo/Placeholder Data Ban (CRITICAL)

**Work is NOT complete if it contains demo, placeholder, or mock data that should be real.**

This is different from intentional test fixtures or development seeds. The check is: "Would this data cause problems in production?"

| Pattern | Examples | Action |
|---------|----------|--------|
| **Fake identities** | "John Doe", "Jane Smith", "Test User", "Admin User" | Replace with actual data source or configurable value |
| **Placeholder domains** | example.com, test.com, foo.bar, localhost hardcoded | Use environment variables or config |
| **Lorem ipsum** | "Lorem ipsum dolor sit amet...", placeholder text | Replace with real content or clear indication it's a template |
| **Magic numbers as data** | `price: 9.99`, `quantity: 100`, `id: 12345` hardcoded | Use actual data source or named constants with clear purpose |
| **Fake contact info** | "555-1234", "test@example.com", "123 Main St" | Use real data source or redact |
| **Hardcoded credentials** | Any username/password even for "testing" | Use environment variables |
| **Mock responses inline** | JSON blobs hardcoded instead of from API/DB | Wire to actual data source |

**How to check:**
```bash
# Common placeholder patterns
rg -i "john doe|jane smith|test user|admin user" --type-add 'code:*.{ts,tsx,js,jsx,py,rb,go}'
rg -i "example\.com|test\.com|localhost" --type-add 'code:*.{ts,tsx,js,jsx,py,rb,go}'
rg -i "lorem ipsum" --type-add 'code:*.{ts,tsx,js,jsx,py,rb,go}'
rg "555-|123-456|test@|placeholder" --type-add 'code:*.{ts,tsx,js,jsx,py,rb,go}'
```

**Exceptions (these are OK):**
- Test fixtures in `/test/`, `/tests/`, `/__tests__/`, `*.test.*`, `*.spec.*` directories
- Seed data explicitly marked as development-only
- Storybook stories or component demos
- Documentation examples

**If found in production code:** STOP. Replace with:
- Environment variable: `process.env.API_URL`
- Config file reference: `config.defaultEmail`
- Dynamic data: data from API/database
- Clear template marker: `"{{USER_NAME}}"` that gets replaced

**Work containing demo data in production paths is NOT complete.**

### 10. Discovered Work Check

*During this implementation, did you discover any of the following?*

| Type | Examples | Action |
|------|----------|--------|
| **Bugs** | Broken functionality, edge cases that fail | `bd create "description" --type bug` |
| **Technical debt** | Workarounds, code that needs refactoring | `bd create "description" --type task` |
| **Enhancement ideas** | Better approaches, missing features | `bd create "description" --type feature` |
| **Documentation gaps** | Missing/outdated docs | Note in completion summary |

**Triage labeling for daemon processing:**

When creating issues for discovered work, apply triage labels so the daemon can process them:

| Confidence | Label | When to use |
|------------|-------|-------------|
| High | `triage:ready` | Clear problem, known fix approach, well-scoped |
| Lower | `triage:review` | Uncertain scope, needs orchestrator input |

```bash
# Example: Creating and labeling a discovered bug
bd create "Edge case fails when input empty" --type bug
bd label <issue-id> triage:ready  # High confidence - daemon can auto-spawn

# Example: Uncertain discovery needs review
bd create "Potential performance issue in query" --type task
bd label <issue-id> triage:review  # Lower confidence - human reviews first
```

**Why triage labels matter:** Issues with `triage:ready` are automatically picked up by the daemon for autonomous processing. Without this label, discovered work requires manual intervention.

**Checklist:**
- [ ] **Reviewed for discoveries** - Checked work for patterns, bugs, or ideas beyond original scope
- [ ] **Tracked if applicable** - Created beads issues for actionable items (or noted "No discoveries")
- [ ] **Labeled for triage** - Applied `triage:ready` or `triage:review` based on confidence
- [ ] **Included in summary** - Completion comment mentions discovered items (if any)

**If no discoveries:** Note "No discovered work items" in completion comment. This is common and acceptable.

**Why this matters:** Implementation work often reveals issues beyond the original scope. Beads issues with triage labels ensure these discoveries surface and get processed autonomously rather than getting lost.

---

## Document Findings

**If self-review finds issues:**
1. Fix them before proceeding
2. Report via `bd comment <beads-id> "Self-review: Fixed [issue summary]"`

**If self-review passes:**
- Report via `bd comment <beads-id> "Self-review passed - ready for completion"`

**Checklist summary (verify mentally, report issues only):**
- **Original symptom validation (bug fixes): Re-ran original failing command/scenario, documented result**
- Anti-patterns: No god objects, tight coupling, magic values, deep nesting, incomplete work
- Security: No hardcoded secrets, no injection vulnerabilities, no XSS
- Commit hygiene: Conventional format, atomic commits, no WIP commits
- Test coverage: Happy path, edge cases, error paths
- Documentation: APIs documented, no debug statements, no commented-out code
- Deliverables: All required deliverables exist and complete
- **Integration wiring: New code imported/called/rendered somewhere (not orphaned)**
- **Demo data ban: No placeholder data in production code paths**
- Discovered work: Reviewed for discoveries, tracked or noted "No discoveries"

---

## If Issues Found

1. **Fix immediately** - Don't proceed with issues
2. **Commit fixes** - Use appropriate commit type (`fix:`, `refactor:`, `chore:`)
3. **Update checklist** - Mark items as resolved
4. **Re-run affected checks** - Verify fix didn't introduce new issues

---

## Completion Criteria

Before proceeding to mark work complete:

- [ ] **For bug fixes:** Original symptom re-tested with exact command/scenario (not a different mode/flag)
- [ ] All anti-pattern checks passed
- [ ] Security review passed (no vulnerabilities)
- [ ] Commit hygiene verified
- [ ] Test coverage adequate
- [ ] Documentation complete
- [ ] All deliverables verified
- [ ] **Integration wiring verified (new code connected to system, not orphaned)**
- [ ] **No demo/placeholder data in production code paths**
- [ ] Discovered work reviewed and tracked (or noted "No discoveries")
- [ ] Self-review passed and reported via bd comment

**If ANY box unchecked, self-review is NOT complete.**

---

## After Self-Review Passes

1. Report self-review status:
   ```bash
   bd comment <beads-id> "Self-review passed - ready for completion"
   ```

2. Proceed to mark complete:
   - Report: `bd comment <beads-id> "Phase: Complete "[deliverables summary]"`
   - Output: "✅ Self-review passed, work complete"
   - Call /exit to close agent session
