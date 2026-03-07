# Phase 1: Root Cause Investigation — Detailed Techniques

## Read Error Messages Carefully

- Don't skip past errors or warnings — they often contain the exact solution
- Read stack traces completely
- Note line numbers, file paths, error codes

## Reproduce Consistently

- Can you trigger it reliably? What are the exact steps?
- Does it happen every time?
- If not reproducible → gather more data, don't guess

## Whack-a-Mole Detection

**Pattern recognition check:**
- Search git history: `git log --all --grep="[issue-type]" --oneline`
- If 2+ previous fixes of same TYPE found → whack-a-mole detected

**Whack-a-mole indicators:**
- Same issue type fixed in different locations (proxy timeout, modal timeout, API timeout)
- Incrementally adjusting same variable (bumping timeouts, adding null checks, increasing retries)
- Each fix works temporarily but similar issues appear elsewhere

**If detected:**
1. STOP fixing symptoms
2. Investigate systemic cause (missing centralized config? validation layer? architectural issue?)
3. Design systematic solution BEFORE implementing fix
4. Escalate to orchestrator for systemic design if needed

**Example:** Modal timeout (2s → 10s fix needed). Git history: 4 previous timeout fixes. Pattern: hardcoded timeouts fail with residential proxies. Systemic solution: centralized timeout config with proxy multiplier.

## Multi-Component Diagnostics

**When system has multiple components (CI → build → signing, API → service → database):**

Add diagnostic instrumentation at each component boundary:
```
For EACH boundary:
  - Log what data enters component
  - Log what data exits component
  - Verify environment/config propagation
  - Check state at each layer

Run once → gather evidence showing WHERE it breaks
THEN investigate that specific component
```

**Example (multi-layer system):**
```bash
# Layer 1: Workflow
echo "=== Secrets available: ==="
echo "IDENTITY: ${IDENTITY:+SET}${IDENTITY:-UNSET}"

# Layer 2: Build script
echo "=== Env vars in build: ==="
env | grep IDENTITY || echo "IDENTITY not in environment"

# Layer 3: Signing
echo "=== Keychain state: ==="
security find-identity -v

# Layer 4: Actual operation
codesign --sign "$IDENTITY" --verbose=4 "$APP"
```

**This reveals:** Which layer fails (secrets → workflow ✓, workflow → build ✗)

## Layer Bias Anti-Pattern

**Where symptoms appear is often NOT where root cause lives.**

**Benchmark (Jan 2026):** Admin logout didn't work. 4/6 AI models created frontend fixes. Root cause was backend: missing `path="/"` in cookie operations. Frontend = symptom location; backend = fix location.

**Triggers:**
- UI shows wrong state → check if backend returns correct data FIRST
- Frontend behavior broken → check backend expected data FIRST
- Error visible at layer N → trace whether cause is at layer N-1

**Countermeasure:** Before implementing frontend fix, verify:
1. Is backend returning correct data?
2. Is state being set correctly at source?
3. Would fixing at source eliminate the need for frontend fix?

**Rule:** Fix at lowest layer that addresses root cause. UI fixes for backend bugs = symptom masking.

## Trace Data Flow

When error is deep in call stack:
- Where does the bad value originate?
- What called this with bad value?
- Keep tracing up until you find the source
- Fix at source, not at symptom

## Security Impact Assessment

Once you understand the bug, assess security implications:

| Question | If YES → |
|----------|----------|
| Could a malicious actor exploit this? | Flag as security issue, escalate urgency |
| Does this expose user data or credentials? | Flag, note data scope |
| Does this bypass authentication/authorization? | Flag, escalate immediately |
| Could this enable injection (SQL, XSS, command)? | Flag, document attack vector |

**If identified:** `bd comments add <beads-id> "SECURITY: [type] - [description]"`
