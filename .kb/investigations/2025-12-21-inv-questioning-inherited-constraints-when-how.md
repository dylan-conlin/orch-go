<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Three distinct signals indicate outdated constraints: (1) implementation supersedes constraint, (2) duplicate entries indicate discovery failure, (3) context shift makes constraint irrelevant. "Wrong constraint" vs "misapplication" distinguished by testing constraint against current codebase.

**Evidence:** Found "fire-and-forget" constraint (Dec 19) superseded by session.go implementation (Dec 21); 5 tmux fallback duplicates in 3 minutes = discovery failure not outdated constraint; Python orch-cli decisions potentially irrelevant after Go rewrite.

**Knowledge:** Constraints don't need expiration dates—they need active validation. Signals should trigger review, not automatic expiration. The "evidence hierarchy" principle (artifacts are claims, code is truth) applies to constraints.

**Next:** Add constraint validation to `orch reflect` command; implement `kn supersede <id> --by <new-id>` for explicit replacement; add pre-create search to `kn constrain`.

**Confidence:** High (85%) - Tested against real kn data; needs validation that signals are actionable.

---

# Investigation: Questioning Inherited Constraints

**Question:** What signals indicate a constraint is outdated? How to distinguish "constraint is wrong" from "we're misapplying it"? Should constraints have expiration dates or review triggers?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Implementation Evolution Signal (CLEAR OUTDATED)

**Evidence:** kn entry `kn-34d52f` from 2025-12-19:
```
content: "orch-go tmux spawn is fire-and-forget - no session ID capture"
reason: "opencode run --attach is TUI-based; --format json gives session ID but loses TUI. Accept title-matching via orch status for monitoring."
```

But `pkg/spawn/session.go` (created 2025-12-21) now provides:
- `WriteSessionID(workspacePath, sessionID)` - atomic write to workspace
- `ReadSessionID(workspacePath)` - retrieve stored session ID
- Session ID now storable in `.session_id` file

The constraint was VALID when created—the implementation evolved 2 days later.

**Source:** `.kn/entries.jsonl` (kn-34d52f), `pkg/spawn/session.go`, `.kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md`

**Significance:** **Signal: Implementation contradicts constraint.** Test: Does `rg` find code that violates or supersedes the constraint? If code exists that the constraint says is impossible/not available, the constraint is outdated.

---

### Finding 2: Duplicate Entry Signal (NOT OUTDATED - Discovery Failure)

**Evidence:** 5 entries about "tmux fallback" created within 3 minutes:
- kn-de6832: 09:49:32 - "tmux fallback requires either current registry window ID OR beads ID"
- kn-666913: 09:50:11 - Nearly identical content
- kn-ccf703: 09:50:59 - "Iteration 8 verification successful"
- kn-2f2ea4: 09:52:22 - Same constraint, slightly reworded
- kn-3b7b1e: 09:52:31 - "orch tail tmux fallback requires..."

All say essentially the same thing. Different agents created duplicates in rapid succession.

**Source:** `.kn/entries.jsonl` timestamp analysis

**Significance:** **NOT a signal of outdated constraint.** This signals:
1. Discovery failure: agents didn't search before creating
2. Missing pre-create check in `kn constrain`
3. Valid constraint worth consolidating/promoting

Duplicates indicate IMPORTANCE, not obsolescence. Response: consolidate, not delete.

---

### Finding 3: Context Shift Signal (POTENTIALLY OUTDATED)

**Evidence:** Decision kn-2e08c6:
```
content: "orch-go is primary CLI, orch-cli (Python) is reference/fallback"
reason: "Go provides better primitives; Python taught requirements through 27k lines and 200+ investigations"
```

This creates a context shift: any constraint about Python orch-cli behavior is now:
- Potentially irrelevant (if orch-go handles it differently)
- Potentially still valid (if orch-go inherited same limitation)
- Needs explicit validation (can't assume either way)

**Source:** `.kn/entries.jsonl` (kn-2e08c6)

**Significance:** **Signal: Major architectural decision invalidates prior context.** When a decision marks an old approach as "reference/fallback", all constraints from that context need review. Not automatically outdated—but needs explicit validation.

---

### Finding 4: Evidence Hierarchy Applies to Constraints

**Evidence:** From investigation skill guidance:
```
Artifacts are claims, not evidence.
| Source Type | Examples | Treatment |
|-------------|----------|-----------|
| Primary (authoritative) | Actual code, test output, observed behavior | This IS the evidence |
| Secondary (claims to verify) | Workspaces, investigations, decisions | Hypotheses to test |
```

Constraints in `.kn/entries.jsonl` are CLAIMS about the system. They're not enforcement—just documentation. The code is the truth.

**Source:** investigation skill guidance, SPAWN_CONTEXT.md

**Significance:** "Constraint is wrong" vs "misapplication" must be tested:
1. Search codebase for constraint's domain
2. Test if constraint holds in current implementation
3. If code contradicts constraint → constraint outdated
4. If code matches constraint but we hit issues → misapplication

---

### Finding 5: Constraints Don't Need Expiration Dates

**Evidence:** Analyzed constraint lifecycle patterns:
- "Session idle ≠ agent complete" (Dec 21) - Still valid, behavioral truth
- "Registry is caching layer" (Dec 21) - Still valid, architectural truth
- "orch complete must verify SYNTHESIS.md" (Dec 21) - Still valid, process requirement

Age doesn't correlate with validity:
- Some 2-day-old constraints are obsolete (fire-and-forget)
- Some same-day constraints remain valid (session idle)

**Source:** Analysis of 13 constraints in .kn/entries.jsonl

**Significance:** Time-based expiration would create noise (false positives). Better approach:
1. **Trigger on signals** (implementation change, duplicates, context shift)
2. **Validate on citation** (when constraint is about to be used, verify it)
3. **Manual review at decision points** (architecture reviews, rewrites)

---

### Finding 6: Missing Infrastructure for Constraint Lifecycle

**Evidence:** Current kn entries have:
- `ref_count: 0` - tracking exists but always zero
- `last_ref: "0001-01-01"` - never updated
- No `superseded_by` field
- No validation workflow

**Source:** `.kn/entries.jsonl` structure analysis

**Significance:** Infrastructure gaps:
1. No way to mark a constraint as superseded (soft delete)
2. No search before create (allows duplicates)
3. No citation tracking (can't find stale constraints)
4. No promotion path (can't elevate to principles)

---

## Synthesis

**Key Insights:**

1. **Three Signals for Outdated Constraints**
   - **Implementation contradiction:** Code exists that violates constraint → Test with `rg`
   - **Context shift:** Major decision invalidates prior domain → Review all related constraints  
   - **Duplicate creation:** NOT obsolescence—indicates importance and discovery failure

2. **"Wrong" vs "Misapplied" Distinguished by Testing**
   - Search codebase for constraint's domain
   - If code contradicts constraint → wrong (outdated)
   - If code matches constraint but we hit issues → misapplied (clarify, don't delete)
   - Never conclude without testing (per investigation skill)

3. **Expiration Dates Are Wrong Solution**
   - Creates false positives (valid constraints expire)
   - Doesn't catch actual obsolescence (invalidated day 1)
   - Better: signal-triggered review + on-citation validation

**Answer to Investigation Question:**

**What signals indicate a constraint is outdated?**
1. **Implementation supersedes:** Code now does what constraint said was impossible
2. **Context shift:** Major architectural decision invalidates domain
3. **Evidence contradiction:** Test shows constraint doesn't hold

**NOT signals of outdated:**
- Age (valid constraints can be old)
- Duplicates (indicates importance, not obsolescence)
- Low citation (might just be well-known)

**How to distinguish "wrong" from "misapplied"?**
1. Search codebase with `rg` for constraint's domain
2. If code exists that violates constraint → constraint is wrong
3. If code matches constraint but problems occur → misapplication
4. The test: "Does the current implementation honor this constraint?"

**Should constraints have expiration dates?**
No. Use signal-triggered review instead:
- `orch reflect` detects patterns (duplicates, potential contradictions)
- On-citation validation before applying constraint
- Explicit supersession when replacing (`kn supersede`)

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**
- Tested against real kn entries (concrete examples)
- Evidence hierarchy principle validates approach
- Implementation vs constraint comparison is deterministic

**What's certain:**

- ✅ Implementation evolution can supersede constraints (fire-and-forget example)
- ✅ Duplicates don't indicate obsolescence (tmux fallback example)
- ✅ Time-based expiration is wrong approach (age doesn't correlate with validity)
- ✅ Testing constraint against code is the validation method

**What's uncertain:**

- ⚠️ How to detect implementation contradictions automatically
- ⚠️ Whether context shift detection can be automated
- ⚠️ Optimal workflow for constraint review

**What would increase confidence to Very High:**

- Implementing `orch reflect` with constraint validation
- Testing signal detection against larger dataset
- User feedback on review workflow

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Signal-triggered validation** - Detect signals automatically, require human review for resolution.

**Why this approach:**
- Avoids false positives from time-based expiration
- Surfaces genuinely outdated constraints
- Leverages existing `orch reflect` infrastructure

**Trade-offs accepted:**
- Not fully automatic (human in loop)
- May miss constraints with no signals

**Implementation sequence:**
1. Add `kn supersede <id> --by <new-id>` for explicit replacement
2. Add pre-create duplicate detection to `kn constrain` 
3. Add constraint validation to `orch reflect`:
   - Find constraints with `rg` patterns in content
   - Check if code matches or contradicts
   - Surface potential conflicts

### Alternative Approaches Considered

**Option B: Expiration Dates**
- **Pros:** Simple, automatic
- **Cons:** High false positive rate, valid constraints expire
- **When to use instead:** If constraints are truly time-bounded (e.g., "Until Dec release")

**Option C: Citation Tracking**
- **Pros:** Find unused constraints
- **Cons:** Infrastructure doesn't exist, high effort
- **When to use instead:** After `ref_count` actually gets populated

---

### Implementation Details

**What to implement first:**
1. `kn supersede` command - explicit constraint replacement
2. Pre-create search in `kn constrain` - "Did you mean existing entry X?"
3. Basic validation in `orch reflect` - find pattern conflicts

**Things to watch out for:**
- ⚠️ Pattern matching for constraint domains is imprecise
- ⚠️ "Contradiction" detection requires semantic understanding
- ⚠️ False positives will train users to ignore suggestions

**Success criteria:**
- ✅ Can supersede a constraint without deleting original
- ✅ Duplicate creation triggers warning
- ✅ `orch reflect` surfaces at least 1 genuine outdated constraint

---

## Test Performed

**Test:** Compare kn constraint "fire-and-forget - no session ID capture" (Dec 19) against current codebase.

**Steps:**
1. Read constraint content and date
2. Search for session ID handling: `rg "session.*id|sessionID|session_id" pkg/spawn/`
3. Found `pkg/spawn/session.go` with `WriteSessionID`, `ReadSessionID` functions
4. Checked file creation date: Dec 21 (2 days after constraint)
5. Verified implementation investigation: `.kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md`

**Result:** Constraint is OUTDATED. Implementation now provides session ID capture and storage. The constraint was valid when created (Dec 19) but superseded by Dec 21 implementation.

**Conclusion validated by test:** Implementation evolution is a clear, testable signal for outdated constraints. Search + compare against code is the validation method.

---

## Self-Review

- [x] Real test performed (not code review) - Searched codebase, compared to constraint, found contradiction
- [x] Conclusion from evidence (not speculation) - Fire-and-forget constraint directly contradicted by session.go
- [x] Question answered - Signals, wrong vs misapplied, expiration dates all addressed
- [x] File complete - All sections filled
- [x] D.E.K.N. filled - Summary section complete
- [x] NOT DONE claims verified - Checked session.go exists and provides claimed functionality

**Self-Review Status:** PASSED

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/.kn/entries.jsonl` - All constraints analyzed
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/session.go` - Session ID implementation
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md` - Session ID investigation
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-temporal-signals-autonomous-reflection.md` - Duplicate detection patterns

**Commands Run:**
```bash
# List all constraints
cat .kn/entries.jsonl | jq -s '[.[] | select(.type == "constraint")]'

# Analyze tmux fallback duplicates
cat .kn/entries.jsonl | jq -s '[.[] | select(.content | test("tmux fallback"; "i"))]'

# Find session ID handling in spawn
rg "session.*id|sessionID|session_id" pkg/spawn/ --type go
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-temporal-signals-autonomous-reflection.md` - Found duplicate pattern
- **Investigation:** `.kb/investigations/2025-12-21-inv-phase-evaluate-spawn-session-id.md` - Session ID design

---

## Investigation History

**2025-12-21 16:30:** Investigation started
- Initial question: When and how to question inherited constraints
- Context: Part of knowledge management meta-investigation series

**2025-12-21 16:45:** Key findings
- Discovered fire-and-forget constraint is outdated (session.go supersedes)
- Analyzed 5 tmux fallback duplicates as discovery failure
- Identified 3 signal types for outdated constraints

**2025-12-21 17:00:** Synthesis complete
- Confirmed expiration dates are wrong approach
- Testing constraint against code is the validation method
- Recommended signal-triggered review

**2025-12-21 17:15:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Three signals identified; "wrong vs misapplied" requires code testing; expiration dates rejected in favor of signal-triggered review
