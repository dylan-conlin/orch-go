# Validation Phase

**Purpose:** Verify implementation works as intended.

**Validation level** determines workflow (read from SPAWN_CONTEXT configuration).

---

## ⚠️ STOP - Check for web/ Changes FIRST

**Before ANY validation workflow:**

```bash
git diff --name-only | grep "^web/"
```

**If this returns ANY files → Visual Verification is MANDATORY.**

You CANNOT mark Phase: Complete without:

1. Screenshot captured via `playwright-cli screenshot`
2. Visual evidence described in `bd comment`

**This is NOT optional. Tests passing does NOT verify UI renders correctly.**

---

## UI Visual Verification (CODE-ENFORCED GATE for web/ changes)

> **Note:** This is a **code-enforced gate** - `orch complete` will BLOCK completion if web/ files are modified without visual verification evidence. Use `--skip-visual --skip-reason` to bypass if needed.

**When:** Agent modifies any files in `web/` directory.

**Why:** UI changes cannot be validated through tests alone. Visual verification ensures the UI renders correctly and functions as expected.

**Workflow:**

1. **Rebuild server:**

   ```bash
   make install
   orch servers stop <project>
   orch servers start <project>
   ```

2. **Capture screenshot using playwright-cli:**

   ```bash
   playwright-cli open http://localhost:5188/your-page
   playwright-cli screenshot
   ```

3. **Include evidence in completion:**
   - Reference screenshot in `bd comment` completion message
   - Describe what the screenshot shows (page, state, key UI elements visible)

**Example completion with UI evidence:**

```bash
bd comment <beads-id> "Phase: Complete "Added stats bar component. Screenshot captured showing new stats bar with 3 agents active, 2 completed. UI renders correctly at localhost:5188."
```

**Critical:** If `playwright-cli` is not available, you MUST report this as a blocker:

```bash
bd comment <beads-id> "BLOCKED: UI changes require visual verification but playwright-cli is not available"
```

---

## Validation: none

**When to use:** Trivial changes where validation overhead exceeds value.

**Workflow:**

1. Confirm changes are complete
2. **Commit all changes** - `git add -A && git commit -m "[type]: [description]"`
3. **Verify commit** - `git status` shows "nothing to commit"
4. Report completion: `bd comment <beads-id> "Phase: Complete "[brief summary]"`
5. Call /exit to close agent session

**That's it - no validation required.**

---

## Validation: tests

**When to use:** Standard validation for features with test suites.

**Workflow:**

1. **Run test suite** - Use project-specific test command (see reference for examples by language)
2. **Verify all tests pass** - All green, no errors/warnings, adequate coverage
3. **Commit all changes** - `git add -A && git commit -m "[type]: [description]"`
4. **Verify commit** - `git status` shows "nothing to commit"
5. **Report completion with test evidence** - `bd comment <beads-id> "Phase: Complete "Tests: <command> - <actual output>"`
6. **Call /exit** - Close agent session

**Test Evidence Requirement:**
Your completion comment MUST include actual test output, not just "tests passing":

- Format: `Tests: <command> - <actual output summary>`
- Good: `Tests: go test ./... - 47 passed, 0 failed (2.3s)`
- Good: `Tests: npm test - 23 specs, 0 failures`
- Bad: `Tests passing` (no command, no numbers)
- Bad: `All tests pass` (no evidence)

**Why:** `orch complete` validates test evidence in comments. Vague claims trigger manual verification.

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/validation-examples.md` for:

- Test commands by language (JavaScript, Python, Ruby, Rust, Go)

---

## Validation: smoke-test

**When to use:** Features with UI components, user-facing functionality, or integration points where automated tests don't verify end-to-end behavior.

**Workflow:**

1. **Run test suite** - First verify automated tests pass (see "Validation: tests")
2. **Load feature** - Start dev server, open browser/API client/CLI (see reference for commands)
3. **Verify manually** - Use checklist for Web UI, API, or CLI verification (see reference)
4. **Capture evidence** - Screenshot for UI, request/response for API/CLI
5. **Commit all changes** - `git add -A && git commit -m "[type]: [description]"`
6. **Verify commit** - `git status` shows "nothing to commit"
7. **Report completion with evidence** - `bd comment <beads-id> "Phase: Complete "Tests: <command> - <output>. Smoke test: [verification summary]"`
8. **Call /exit** - Close agent session

**Critical:** Tests passing ≠ feature working. Always perform manual verification for user-facing features.

**Test Evidence Requirement:** Same as "Validation: tests" - include actual test command and output.

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/validation-examples.md` for:

- Commands to load feature (Web UI, API, CLI)
- Verification checklists (Web UI, API, CLI)

---

## Validation: multi-phase

**When to use:** Complex features with multiple phases where orchestrator needs to manually validate each phase before allowing next phase to proceed.

**Purpose:** Creates explicit checkpoint for orchestrator verification before next phase begins.

**Workflow:**

1. **Run test suite** - Verify automated tests pass
2. **Smoke test (if UI)** - Perform manual verification if feature includes UI
3. **Commit all changes** - `git add -A && git commit -m "[type]: [description]"`
4. **Verify commit** - `git status` shows "nothing to commit"
5. **Report awaiting validation** - `bd comment <beads-id> "AWAITING_VALIDATION - [phase details, evidence summary]"`
6. **STOP** - Wait for orchestrator approval. DO NOT proceed to next phase
7. **After approval** - Report: `bd comment <beads-id> "Phase: Complete "[summary]"`
8. **Call /exit** - Close agent session

**Critical:** STOP and wait for explicit orchestrator approval. Do not proceed or mark complete without approval.

---

## When Validation Fails

**If tests fail or smoke test reveals issues:**

1. **Check logs for runtime errors:**

   ```bash
   # Check test output logs
   make test 2>&1 | tail -50
   # Check project-specific logs
   tail -50 *.log 2>/dev/null
   ```

   This shows runtime errors from your implementation. Often reveals the root cause immediately (stack traces, assertion failures, uncaught exceptions).

2. **Analyze failure output** - Read test output carefully for specific assertion failures
3. **Return to Implementation** - Fix the issue, re-run tests
4. **Re-validate** - Repeat validation workflow after fix

---

## Common Issues

**See reference for detailed troubleshooting:**

- Tests pass but feature doesn't work (tests verify logic, not UI/integration)
- Smoke test reveals issues (return to Implementation, fix, re-validate)
- Multi-phase orchestrator finds issues (fix immediately, don't defend)

**Reference:** See `~/.claude/skills/worker/feature-impl/reference/validation-examples.md` for detailed common issues and solutions.

---

## Completion Criteria

**For validation: none:**

- [ ] Changes complete
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via `bd comment`: Phase: Complete

**For validation: tests:**

- [ ] Test suite passing
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via `bd comment`: Phase: Complete with **actual test output** (command + pass/fail count)

**For validation: smoke-test:**

- [ ] Test suite passing
- [ ] Manual verification complete
- [ ] Evidence captured (screenshot/output)
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via `bd comment`: Phase: Complete with **actual test output** AND verification summary

**For validation: multi-phase:**

- [ ] Test suite passing
- [ ] Smoke test complete (if UI)
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via `bd comment`: AWAITING_VALIDATION
- [ ] Orchestrator manually tested and approved
- [ ] Reported via `bd comment`: Phase: Complete (after approval)

**⚠️ If modified web/ files (ANY validation level - MANDATORY):**

- [ ] Ran `git diff --name-only | grep "^web/"` - confirmed web/ files modified
- [ ] Server rebuilt (`make install`)
- [ ] Server restarted (`orch servers stop/start`)
- [ ] Screenshot captured via `playwright-cli screenshot`
- [ ] Screenshot evidence described in completion comment
- [ ] `bd comment <beads-id> "Visual verification: [description of what screenshot shows]"`

**CRITICAL: If web/ files were modified and visual verification is missing, orchestrator WILL reject the completion.**

**If ANY box unchecked, validation is NOT complete.**
