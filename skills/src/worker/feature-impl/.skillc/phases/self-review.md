# Self-Review Phase

**Purpose:** Quality gate before completion. Catch issues that require human judgment — automated checks (debug statements, commit format, placeholder data, orphaned files) run at `orch complete` time via the `self_review` gate.

---

## Self-Review Checklist

**Perform each check. Document significant findings via `bd comment`.**

### 0. Original Symptom Validation (For Bug Fixes)

**Skip if:** Not fixing a bug.

- [ ] Re-ran EXACT command/scenario from original issue (same flags, same mode)
- [ ] Documented actual result via `bd comment`
- [ ] If testing different mode/flags: justified why via `bd comment`

**⚠️ Scope Redefinition Warning:** Testing a different scenario than the original issue enables agents to rationalize partial fixes as complete.

---

### 1. Scope Verification (For Refactoring/Migration)

**Skip if:** Purely additive work (new files, new functions).

```bash
rg "old_pattern"  # Should return 0
rg "new_pattern"  # Should match expected count
```

---

### 2. Anti-Pattern Detection

- [ ] No god objects (files >300 lines or multiple concerns)
- [ ] No tight coupling (use dependency injection)
- [ ] No magic values (use named constants)
- [ ] No deep nesting (>3 levels → extract helpers)
- [ ] No incomplete work (TODOs, placeholders)

### 3. Security Review

- [ ] No hardcoded secrets (API keys, passwords, tokens)
- [ ] No injection vulnerabilities (SQL, XSS, command, path traversal)

**If security issue found:** Fix immediately.

### 4. Test Coverage

- [ ] Happy path tested
- [ ] Edge cases covered
- [ ] Error paths tested

### 5. Deliverables Verification

- [ ] All required deliverables exist and complete
- [ ] Deliverables reported via `bd comment`

### 6. Integration Wiring (CRITICAL)

**New code MUST be wired into the system, not just exist in isolation.**

| Check | How |
|-------|-----|
| New modules imported | `rg "import.*new-file"` |
| New functions called | `rg "newFunctionName\("` |
| New routes registered | Check route registration |
| New components rendered | `rg "<NewComponent"` |

**Red flags:** New file with 0 imports, new export with 0 consumers, new route not registered.

### 7. Accessibility (If UI Changed)

**Skip if:** No web/ changes.

- [ ] Semantic HTML, keyboard-accessible, form labels, alt text
- [ ] Color contrast (4.5:1 text, 3:1 UI components)

### 8. Performance Impact (If Applicable)

- [ ] No N+1 queries, no unbounded loops on user input
- [ ] New routes/components lazy-loaded where appropriate

### 9. Error Resilience (If UI/API Changed)

- [ ] API failures show helpful error states (not stack traces)
- [ ] Components handle missing/null data without crashing

### 10. Discovered Work

| Type | Action |
|------|--------|
| Bugs | `bd create "description" --type bug -l triage:review` |
| Tech debt | `bd create "description" --type task -l triage:review` |
| Enhancements | `bd create "description" --type feature -l triage:review` |
| Strategic unknowns | `bd create "description" --type question -l triage:review` |

High-confidence items: use `triage:ready` instead.

**If no discoveries:** Note "No discovered work" in completion comment.

---

## Automated Checks (run at `orch complete`)

The following are enforced by the `self_review` verification gate — you do NOT need to manually check these:

- **Debug statements** — `console.log`, `fmt.Print`, `debugger`, `pdb.set_trace` in production files
- **Commit format** — WIP/temp/fixup commits blocked
- **Placeholder data** — John Doe, lorem ipsum, test@example.com, 555-xxxx in production files
- **Orphaned Go files** — New `.go` files in packages not imported anywhere

---

## Document Findings

**If self-review finds issues:** Fix, then `bd comments add <beads-id> "Self-review: Fixed [issue]"`

**If passed:** `bd comments add <beads-id> "Self-review passed - ready for completion"`

---

## Completion Criteria

- [ ] Original symptom validated (bug fixes) or N/A
- [ ] Anti-patterns checked
- [ ] Security reviewed
- [ ] Test coverage adequate
- [ ] Deliverables verified
- [ ] Integration wiring verified
- [ ] Accessibility checked (if UI changed)
- [ ] Discovered work reviewed and tracked
- [ ] Self-review reported via bd comment
