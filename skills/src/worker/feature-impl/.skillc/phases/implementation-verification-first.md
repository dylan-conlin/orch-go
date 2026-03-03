# Implementation Phase (Verification-First Mode)

**Purpose:** Implement feature by first specifying expected behavior, instrumenting verification, implementing, then verifying behavior matches spec.

**When to use:** Feature work where verifiability is critical - the spec defines what "working" means BEFORE implementation starts.

**Core principle:** Specify what "working" looks like before writing code. The spec is the contract; tests prove the contract is met.

---

## When to Use Verification-First vs TDD

| Criterion              | Verification-First                                      | TDD                                  |
| ---------------------- | ------------------------------------------------------- | ------------------------------------ |
| **Spec exists**        | Yes - consume existing verification spec                | No - discover behavior through tests |
| **Multi-agent work**   | Required - interface contracts define boundaries        | Optional                             |
| **High-risk features** | Preferred - explicit failure modes guide implementation | Acceptable                           |
| **Simple changes**     | Minimum viable spec suffices                            | May be lighter weight                |

**Default recommendation:** Use verification-first when spec exists or for non-trivial features. Use TDD when rapidly iterating without upfront spec.

---

## Step 0.5: Consume Verification Specification (ADVISORY)

> **Note:** This is an **advisory checkpoint** for verification-first mode. The code-enforced gates will verify deliverables exist, but consuming the spec upfront is suggested best practice.

**Before any implementation, consume and understand the verification spec.**

### Locate the Spec

Check these locations in order:

1. Spawn context (attached verification-spec.md)
2. Beads issue (linked verification spec)
3. Project docs (`docs/verification/` or `specs/`)

**If no spec found:**

- For simple work: Create minimum viable spec (see template below)
- For complex work: BLOCKED - request spec from orchestrator

### Parse the Spec

Extract and enumerate:

1. **Observable Behaviors:** What can be seen when working?

   ```
   bd comment <beads-id> "Behaviors to implement: 1. [primary] 2. [secondary]..."
   ```

2. **Acceptance Criteria:** What proves each behavior works?

   ```
   bd comment <beads-id> "Acceptance criteria: AC-001: [condition], AC-002: [condition]..."
   ```

3. **Failure Modes:** What breaks it and how to diagnose?

   ```
   bd comment <beads-id> "Failure modes to handle: FM-001: [symptom → fix]..."
   ```

4. **Evidence Requirements:** What artifacts prove verification?
   ```
   bd comment <beads-id> "Evidence required: [test output / screenshot / log / etc.]"
   ```

### Create Traceability Matrix

Map behaviors → criteria → tests → evidence:

```markdown
| Behavior | Criterion | Test     | Evidence   |
| -------- | --------- | -------- | ---------- |
| [B1]     | AC-001    | test_xxx | [artifact] |
| [B2]     | AC-002    | test_yyy | [artifact] |
```

**Report readiness:**

```bash
bd comment <beads-id> "Spec consumed: [N] behaviors, [M] acceptance criteria, [K] failure modes. Ready to instrument verification."
```

---

## Minimum Viable Verification Spec

For simple work when no formal spec exists, create inline:

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
- **Fix:** Check API response handling in auth.ts:45

**Evidence:** Screenshot showing header with username + test output from auth.test.ts
```

---

## Phase 1: Instrument Verification (Tests from Spec)

**Purpose:** Write tests that directly prove acceptance criteria - BEFORE implementation.

### Key Difference from TDD

| TDD                                | Verification-First                      |
| ---------------------------------- | --------------------------------------- |
| Write test for behavior you want   | Write test that proves AC-xxx from spec |
| Test describes what code should do | Test proves contract is met             |
| Discovery-driven                   | Contract-driven                         |

### Write Tests from Acceptance Criteria

For each acceptance criterion:

1. **Reference the criterion in test:**

   ```typescript
   // AC-001: Login with valid credentials displays username in header
   test('AC-001: displays username after login', async () => {
     // Arrange
     const validCredentials = { email: 'test@example.com', password: 'valid' }

     // Act
     await login(validCredentials)

     // Assert - directly from AC-001 condition
     expect(getHeader()).toContain('Logged in as')
   })
   ```

2. **Run test - verify it FAILS:**
   - Test must fail before implementation
   - Failure reason must match expected gap (not a syntax error)

3. **Commit failing test:**
   ```bash
   git commit -m "test: add AC-001 verification test for [behavior]"
   ```

### Handle Failure Modes

For each failure mode in spec:

1. **Add diagnostic test (optional but recommended):**

   ```typescript
   // FM-001: Spinner continues indefinitely
   test('FM-001: timeout triggers error state on slow response', async () => {
     mockApiDelay(5000) // Simulate slow response
     await login(validCredentials)
     expect(getErrorState()).toBe('timeout')
   })
   ```

2. **These may pass or fail initially** - they define expected error handling

---

## Phase 2: Implement to Pass Tests

**Now implement - minimal code to make tests pass.**

### Key Constraints

- ⚠️ Implement ONLY what's needed to pass tests
- Tests are tied to acceptance criteria - passing tests = criteria met
- If tests feel incomplete → add more criteria to spec, not ad-hoc tests

### Implementation Workflow

1. Implement smallest unit to pass first test
2. Run tests - verify green
3. Commit: `git commit -m "feat: implement [behavior] per AC-001"`
4. Repeat for remaining criteria

### Refactor While Green

After tests pass:

- Clean up implementation (extract, rename, simplify)
- Run tests after each refactor
- Tests must stay green

---

## Phase 3: Verify Behavior Matches Spec

**Tests passing is necessary but not sufficient.** Verify actual behavior matches spec.

### Cross-Reference Check

For each acceptance criterion:

| Criterion | Test Status | Behavior Observed       | Evidence Captured |
| --------- | ----------- | ----------------------- | ----------------- |
| AC-001    | ✅ Pass     | [Yes/No - manual check] | [artifact path]   |
| AC-002    | ✅ Pass     | [Yes/No - manual check] | [artifact path]   |

### Capture Evidence per Spec

The spec defines what evidence is required. Capture it:

| Evidence Type | How to Capture                        |
| ------------- | ------------------------------------- |
| Test output   | Copy terminal output or `tee` to file |
| Screenshot    | Playwright MCP or Glass MCP           |
| Log entry     | Extract from application logs         |
| Metric        | Query from monitoring/DB              |

### Report Verification

```bash
bd comment <beads-id> "Verification complete: AC-001 ✅ (test + behavior + evidence), AC-002 ✅ (test + behavior + evidence). All behaviors match spec."
```

---

## UI Feature Requirements (web/ Changes)

**⚠️ CRITICAL: Tests passing ≠ feature working (especially for UI)**

If you modified ANY file in `web/` directory:

1. **Check for web/ changes:**

   ```bash
   git diff --name-only | grep "^web/"
   ```

2. **If files returned → visual verification is MANDATORY**

3. **Capture screenshot evidence** (per spec evidence requirements)

4. **Document via beads:**
   ```bash
   bd comment <beads-id> "Visual verification: [behavior observed matches spec]"
   ```

---

## Red Flags - STOP and Restart

**If doing any of these, stop and reassess:**

- Writing code before tests exist
- Tests don't reference acceptance criteria (ad-hoc tests)
- Implementing features not in spec (scope creep)
- Skipping evidence capture ("tests pass is enough")
- Behavior doesn't match spec but tests pass (bad tests)

---

## Completion Criteria

Before moving to Validation phase:

- [ ] Verification spec consumed and enumerated
- [ ] Tests reference acceptance criteria (AC-xxx)
- [ ] All tests pass (green)
- [ ] Behavior cross-referenced against spec
- [ ] Evidence captured per spec requirements
- [ ] UI smoke test complete (if web/ changes)
- [ ] Reported via beads with traceability: `bd comment <beads-id> "Phase: Validation "All [N] acceptance criteria verified with evidence"`

---

## When to Move to Validation Phase

Once completion criteria met → Report via `bd comment <beads-id> "Phase: Validation "Tests: <command> - <output>. Spec criteria: AC-001 ✅, AC-002 ✅"` → Proceed to Validation phase
