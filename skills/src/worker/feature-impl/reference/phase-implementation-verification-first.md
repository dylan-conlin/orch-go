# Implementation Phase: Verification-First Mode (Detailed Reference)

**Purpose:** Implement feature by specifying expected behavior, instrumenting verification, implementing, then verifying behavior matches spec.

**When to use:**
- Verification spec exists (from architect, design-session, or spec-kit workflow)
- Multi-agent work where interface contracts define boundaries
- High-risk features where "working" must be explicitly defined upfront
- When TDD's discovery-driven approach isn't appropriate

---

## Overview: Verification-First vs TDD

| Aspect | Verification-First | TDD |
|--------|-------------------|-----|
| **Starting point** | Spec defines what "working" means | Tests discover what "working" means |
| **Test origin** | Derived from acceptance criteria (AC-xxx) | Written to express desired behavior |
| **Failure modes** | Specified upfront in spec | Discovered through implementation |
| **Evidence** | Required per spec | Optional (tests are evidence) |
| **Best for** | Contract-driven, multi-agent, high-risk | Discovery, rapid iteration, simple features |

**Key insight:** Verification-first is NOT a replacement for TDD - it's TDD with an explicit contract phase. You still write tests first; you just derive them from a spec rather than discovering them.

---

## The Verification-First Workflow

```
SPECIFY → INSTRUMENT → IMPLEMENT → VERIFY
   ↓          ↓           ↓          ↓
  Spec     Tests        Code      Evidence
```

### Phase 1: SPECIFY (Consume Verification Spec)

**Goal:** Understand and enumerate what "working" looks like.

**Locate the spec:**
1. Spawn context (attached verification-spec.md)
2. Beads issue (linked verification spec)
3. Project docs (`docs/verification/` or `specs/`)
4. Create minimum viable spec if none exists

**Extract from spec:**

| Element | What to Extract | Example |
|---------|-----------------|---------|
| Observable Behaviors | What can be seen when working | "User sees 'Logged in as [name]' after auth" |
| Acceptance Criteria | Pass/fail conditions with IDs | AC-001: Login with valid creds shows username in <2s |
| Failure Modes | Symptom → root cause → fix | FM-001: Spinner hangs → timeout → check API handler |
| Evidence Requirements | What artifacts prove it works | Screenshot + test output |

**Create traceability matrix:**

```markdown
| Behavior | Criterion | Test File | Evidence |
|----------|-----------|-----------|----------|
| Login displays username | AC-001 | auth.test.ts | Screenshot + test output |
| Invalid creds show error | AC-002 | auth.test.ts | Test output |
```

**Report enumeration:**
```bash
bd comment <beads-id> "Spec consumed: 3 behaviors, 5 acceptance criteria, 2 failure modes. Traceability matrix created."
```

---

### Phase 2: INSTRUMENT (Write Tests from Spec)

**Goal:** Write tests that directly prove acceptance criteria - BEFORE implementation.

**Test naming convention:**
```typescript
// AC-001: [Criterion description from spec]
test('AC-001: displays username after successful login', () => {
  // Test body
});
```

**Test structure:**
```typescript
// Reference the spec criterion
// AC-001: Login with valid credentials displays username in header within 2 seconds

describe('Authentication (Verification Spec)', () => {
  // AC-001
  test('AC-001: displays username after successful login', async () => {
    // Arrange - setup from spec context
    const validCredentials = { email: 'test@example.com', password: 'valid' };

    // Act - trigger the behavior
    await login(validCredentials);

    // Assert - DIRECTLY from acceptance criterion
    expect(getHeader()).toContain('Logged in as');
  });

  // AC-002
  test('AC-002: shows error for invalid credentials', async () => {
    const invalidCredentials = { email: 'test@example.com', password: 'wrong' };
    await login(invalidCredentials);
    expect(getError()).toBe('Invalid credentials');
  });

  // FM-001: Failure mode test (optional but recommended)
  test('FM-001: timeout triggers error state on slow response', async () => {
    mockApiDelay(5000);
    await login(validCredentials);
    expect(getErrorState()).toBe('timeout');
  });
});
```

**Run tests - verify FAILURE:**
```bash
npm test -- --grep "AC-001"
# Expected: FAIL (because implementation doesn't exist yet)
```

**Commit failing tests:**
```bash
git commit -m "test: add verification tests for auth per AC-001, AC-002"
```

---

### Phase 3: IMPLEMENT (Minimal Code to Pass)

**Goal:** Implement just enough to make tests pass.

**Constraints:**
- Implement ONLY what's needed to pass tests
- Tests are tied to acceptance criteria - passing tests = criteria met
- If tests feel incomplete → add more criteria to spec first, not ad-hoc tests

**Workflow:**
1. Take first failing test
2. Write minimal code to pass it
3. Run tests - verify green
4. Commit: `git commit -m "feat: implement [behavior] per AC-001"`
5. Repeat for remaining criteria

**Refactor while green:**
- Clean up code (extract, rename, simplify)
- Run tests after each change
- Tests must stay green

---

### Phase 4: VERIFY (Behavior Matches Spec)

**Goal:** Confirm actual behavior matches spec - tests passing is necessary but not sufficient.

**Cross-reference check:**

For each acceptance criterion, verify three things:

| Check | Question | Evidence |
|-------|----------|----------|
| Test passes | Does the test pass? | Test output |
| Behavior observable | Can you see the behavior in running system? | Screenshot, log, manual check |
| Evidence captured | Is the evidence artifact available? | File exists, documented |

**Verification matrix:**

```markdown
| Criterion | Test Status | Behavior Observed | Evidence Captured |
|-----------|-------------|-------------------|-------------------|
| AC-001 | ✅ Pass | ✅ Username shows in header | auth-success.png |
| AC-002 | ✅ Pass | ✅ Error message visible | test-output.txt |
```

**Report verification:**
```bash
bd comment <beads-id> "Verification complete: AC-001 ✅ (test + behavior + evidence), AC-002 ✅ (test + behavior + evidence). All behaviors match spec."
```

---

## Minimum Viable Verification Spec

When no formal spec exists and creating one would be overhead, use this inline format:

```markdown
## Verification Spec (Minimum Viable)

**Observable Behavior:** [What can be seen when working - one sentence]

**Acceptance Criterion:** [Testable pass/fail condition - one criterion]

**Failure Mode:**
- **Symptom:** [What you see when broken]
- **Fix:** [How to resolve]

**Evidence:** [What artifact proves it works]
```

**Example:**

```markdown
## Verification Spec (Minimum Viable)

**Observable Behavior:** User sees "Logged in as [name]" after successful authentication.

**Acceptance Criterion:** AC-001: Login with valid credentials displays username in header within 2 seconds.

**Failure Mode:**
- **Symptom:** Spinner continues indefinitely after login button click
- **Fix:** Check API response handling in auth.ts:45 - likely missing await or timeout

**Evidence:** Screenshot showing header with username + test output from auth.test.ts
```

**When minimum viable is sufficient:**
- Single-behavior features
- Low-risk changes
- Clear scope with no ambiguity
- No multi-agent coordination needed

**When full spec is needed:**
- Multiple behaviors with interactions
- Interface contracts for multi-agent work
- High-risk features (security, data integrity)
- Unclear scope needing disambiguation

---

## UI Feature Requirements (web/ Changes)

**⚠️ Tests passing ≠ feature working (especially for UI)**

If you modified ANY file in `web/`:

1. **Check:** `git diff --name-only | grep "^web/"`

2. **If files returned → visual verification is MANDATORY**

3. **Capture evidence per spec:**
   - Use Glass MCP `glass_screenshot` (if spawned with --mcp glass)
   - Or Playwright MCP `browser_take_screenshot`

4. **Document:**
   ```bash
   bd comment <beads-id> "Visual verification: [AC-001: username visible in header, AC-002: error shows on invalid login]. Screenshots: auth-success.png, auth-error.png"
   ```

---

## Red Flags - STOP and Reassess

| Red Flag | Why It's Bad | What to Do |
|----------|--------------|------------|
| Writing code before tests | Loses contract-first discipline | Delete code, write test first |
| Tests don't reference AC-xxx | Ad-hoc testing, not spec-driven | Rewrite tests to trace to criteria |
| Implementing features not in spec | Scope creep | Stop, escalate if needed |
| Behavior passes tests but doesn't match spec | Bad tests | Fix tests to actually verify spec |
| Skipping evidence capture | "Tests pass" isn't enough | Capture per spec requirements |

---

## Completion Criteria

Before moving to Validation phase:

- [ ] Verification spec consumed and enumerated
- [ ] Traceability matrix created (Behavior → Criterion → Test → Evidence)
- [ ] All tests reference acceptance criteria (AC-xxx naming)
- [ ] All tests pass (green)
- [ ] Behavior cross-referenced against spec (manual check)
- [ ] Evidence captured per spec requirements
- [ ] UI smoke test complete (if web/ changes)
- [ ] Reported via beads with traceability

**Report format:**
```bash
bd comment <beads-id> "Phase: Validation - Tests: [command] - [output]. Spec criteria: AC-001 ✅, AC-002 ✅, AC-003 ✅"
```

---

## Related Resources

- **Verification Spec Template:** `.kb/templates/verification-spec.md`
- **Investigation (format design):** `.kb/investigations/2026-02-03-inv-design-verification-specification-format-verifiability.md`
- **Decision (spec-kit integration):** `.kb/decisions/2026-02-03-standalone-verification-spec-complements-spec-kit.md`
- **TDD Reference:** `~/.claude/skills/worker/feature-impl/reference/phase-implementation-tdd.md`
