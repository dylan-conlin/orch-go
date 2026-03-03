# TDD Best Practices and Red Flags

**Purpose:** Reference guide for test-driven development best practices, anti-patterns, and red flags.

---

## The Iron Law

```
NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST
```

Write code before the test? **Delete and start over** (ensures tests verify behavior).

**Why test-first matters:** If you didn't watch the test fail, you can't be sure it tests the right thing.

---

## Best Practices

**Test-first, always:**
- Write test before implementation (no exceptions)
- Watch test fail (verify failure reason)
- Write minimal code to pass

**Red-Green-Refactor:**
- RED: Fail → GREEN: Pass → REFACTOR: Clean

**Small cycles:**
- One behavior at a time
- Minimal test, minimal code
- Quick iterations

**Commit frequently:**
- Each TDD cycle gets commits (test commit → impl commit → refactor commit if needed)
- Separate test commits from implementation commits
- Use conventional commit format

---

## Good Tests

| Quality | Good | Bad |
|---------|------|-----|
| **Minimal** | One thing per test | `test('validates email and domain and whitespace')` |
| **Clear** | Name describes behavior | `test('test1')` or `test('works')` |
| **Shows intent** | Demonstrates desired API | Obscures what code should do |

**If test name has "and" → split it into multiple tests.**

---

## Red Flags - STOP and Start Over

**If you catch yourself doing any of these:**

- Writing code before test
- Writing test after implementation
- Test passes immediately (didn't see it fail)
- Can't explain why test failed
- Planning to add tests "later"
- Rationalizing "just this once"
- "I already manually tested it"
- "Tests after achieve the same purpose"
- "It's about spirit not ritual"
- "Already spent X hours, deleting is wasteful"
- "TDD is dogmatic, I'm being pragmatic"
- "This is different because..."

**All of these mean: Delete code. Start over with TDD.**

**The Iron Law is non-negotiable.**

---

## Code Examples

### RED Phase Example (TypeScript)

**Test (FAILING):**
```typescript
// tests/auth.test.ts
test('rejects empty email', async () => {
  const result = await submitForm({ email: '' });
  expect(result.error).toBe('Email required');
});
```

**Verify RED:**
```bash
$ npm test
FAIL: expected 'Email required', got undefined ✅
```

**Success criteria:** Test fails for expected reason (feature missing, not typo)

---

### GREEN Phase Example (TypeScript)

**Implementation (MINIMAL CODE):**
```typescript
// src/auth.ts
function submitForm(data: FormData) {
  if (!data.email?.trim()) {
    return { error: 'Email required' };
  }
  // ... rest of logic
}
```

**Verify GREEN:**
```bash
$ npm test
PASS ✅
```

**Success criteria:** Test passes, all green, no over-engineering

---

### REFACTOR Phase Example

**Steps:**
1. Remove duplication
2. Improve names
3. Extract helpers
4. Run tests again - verify still passing

**Commit (if refactored):**
```bash
git add src/
git commit -m "refactor: [what you improved]"
```

**Success criteria:** Tests still pass, code is cleaner

---

## UI Feature Requirements

**Pattern:** Tests passing ≠ feature working (especially for UI)

**Why This Matters:**

Unit/integration tests verify **logic**:
```ruby
test "action returns data" do
  assert_response :success
  assert_not_nil @data
end
# ✅ Tests pass
```

But they DON'T verify:
- ❌ Stylesheets actually load
- ❌ JavaScript runs without errors
- ❌ Data renders in HTML (not just exists in memory)
- ❌ Visual layout works
- ❌ User can actually use the feature

**Real example:**
- Agent: "240 tests passing" ✅
- Reality: No stylesheets loaded ❌
- Reality: All prices showing $0.00 ❌
- Reality: Interactive elements broken ❌

**DO NOT mark work complete without smoke test for UI features.**

---

## Commit Format Examples

**Test commit:**
```bash
git add tests/
git commit -m "test: add failing test for [behavior]"
```

**Implementation commit:**
```bash
git add src/
git commit -m "feat: implement [behavior]"
```

**Refactor commit:**
```bash
git add src/
git commit -m "refactor: [what you improved]"
```

---

## Completion Checklist

**TDD Cycle:**
- [ ] Every new function/method has a test
- [ ] Watched each test fail before implementing
- [ ] Each test failed for expected reason (feature missing, not typo)
- [ ] Wrote minimal code to pass each test
- [ ] All tests pass (green)

**Code Quality:**
- [ ] Output pristine (no errors, warnings)
- [ ] Tests use real code (mocks only if unavoidable)
- [ ] Edge cases and errors covered
- [ ] Refactored where needed (no duplication)

**UI Features (if applicable):**
- [ ] Smoke test complete (browser/actual verification)
- [ ] Smoke test documented in workspace
- [ ] No browser console errors / API errors
- [ ] Screenshot or detailed description of working feature

**Git:**
- [ ] Test commits separate from implementation commits
- [ ] Conventional commit format (test:/feat:/refactor:)
- [ ] Commit history shows red-green-refactor cycle

**Workspace:**
- [ ] Phase: Validation (ready for validation phase)
- [ ] TDD cycle documented (how many cycles, what was tested)
- [ ] Smoke test results (if UI feature)

**If ANY box unchecked, work is NOT complete. Do NOT move to Validation phase.**
