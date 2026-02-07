<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Four distinct failure modes identified in crash.log; AI_NoOutputGeneratedError is most frequent (43%), all stem from upstream library or null-check issues.

**Evidence:** Parsed 7 crash entries from Jan 24-27; three AI_NoOutputGeneratedError occurred in 30-second burst suggesting transient API issue; TypeError in summary.ts lacks null guard.

**Knowledge:** unhandledRejections don't crash server but corrupt sessions; Jan 26 investigation already analyzed these; crashes cluster temporally suggesting environmental triggers not code bugs.

**Next:** Fix null check in summary.ts:69; monitor for AI_NoOutputGeneratedError patterns correlating with API outages; consider retry logic for stream operations.

**Promote to Decision:** recommend-no (tactical fixes, Jan 26 investigation already provided actionable recommendations)

---

# Investigation: OpenCode Crash Log Failure Pattern Analysis

**Question:** What distinct failure modes exist in OpenCode crash.log, and what are their root causes, frequencies, and severity rankings?

**Started:** 2026-01-26
**Updated:** 2026-01-26
**Owner:** Worker agent (investigation)
**Phase:** Complete
**Next Step:** None - analysis complete, recommendations synthesized from prior investigations
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** None
**Extracted-From:** Issue orch-go-20944.1
**Supersedes:** None
**Superseded-By:** None

---

## Error Categorization

### Complete Error Inventory

| ID | Error Type | Count | Date Range | Stack Origin | Severity |
|----|-----------|-------|------------|--------------|----------|
| E1 | TypeError: msgWithParts.info | 2 | Jan 24 | summary.ts:69 | Medium |
| E2 | No user message in stream | 1 | Jan 24 | prompt.ts:293 | High |
| E3 | ProviderModelNotFoundError | 1 | Jan 26 | provider.ts:1082 | Low |
| E4 | AI_NoOutputGeneratedError | 3 | Jan 27 | ai@5.0.97 (external) | High |

**Total Entries:** 7 unhandledRejection events

---

## Findings

### Finding 1: AI_NoOutputGeneratedError is Most Frequent (43%)

**Evidence:** Three occurrences within 30 seconds (00:40:32, 00:40:47, 00:41:00 on Jan 27):
```
AI_NoOutputGeneratedError: No output generated. Check the stream for errors.
    at flush (../../node_modules/.bun/ai@5.0.97+d6123d32214422cb/node_modules/ai/dist/index.mjs:4786:31)
```

Memory at crash time: heapUsed 114-115MB, rss 336-340MB (stable, not exhaustion)

**Source:** `~/.local/share/opencode/crash.log:53-112`

**Significance:**
- **Temporal clustering** suggests external trigger (API timeout, rate limit, or connection drop)
- **Not a code bug** - error originates in Vercel's `ai` SDK npm package
- **Session impact:** HIGH - stream fails completely, no response generated

**Root Cause Hypothesis:** Upstream Claude/Gemini API returned empty stream or connection was interrupted. The `ai` SDK flush operation found no content to emit.

---

### Finding 2: TypeError in summarizeMessage Indicates Null Check Gap

**Evidence:** Two occurrences (Jan 24 02:01 and 07:15):
```
TypeError: undefined is not an object (evaluating 'msgWithParts.info')
    at summarizeMessage (src/session/summary.ts:69:21)
    at summarizeMessage (src/session/summary.ts:64:35)  // recursive call
    at <anonymous> (src/session/summary.ts:32:9)
```

**Source:** `~/.local/share/opencode/crash.log:1-28`

**Significance:**
- **Code bug** - missing null/undefined guard for `msgWithParts.info`
- **Recursive pattern** indicates the summarizer processes nested messages
- **Session impact:** MEDIUM - summarization fails but session may continue

**Root Cause Hypothesis:** Message without `info` property passed to summarizer (corrupted message, edge case in message format, or race condition during message processing).

---

### Finding 3: Stream Invariant Violation is Critical

**Evidence:** One occurrence (Jan 24 07:15:52.197):
```
Error: No user message found in stream. This should never happen.
    at <anonymous> (src/session/prompt.ts:293:32)
```

Occurred 2ms after a summarizeMessage error at same timestamp.

**Source:** `~/.local/share/opencode/crash.log:29-40`

**Significance:**
- **Invariant violation** - code asserts condition that was violated
- **Same-second correlation** with TypeError suggests cascade failure
- **Session impact:** HIGH - fundamental assumption about message stream broken

**Root Cause Hypothesis:** Cascade from summarizeMessage failure - if summarization corrupts or skips messages, downstream prompt processing may not find expected user message. The 2ms gap suggests direct causal relationship.

---

### Finding 4: ProviderModelNotFoundError is Configuration Issue

**Evidence:** One occurrence (Jan 26 21:51):
```
ProviderModelNotFoundError: ProviderModelNotFoundError
    at getModel (src/provider/provider.ts:1082:17)
```

**Source:** `~/.local/share/opencode/crash.log:41-52`

**Significance:**
- **Configuration error** - requested model not available
- **Session impact:** LOW - specific session fails, doesn't affect others
- **Likely cause:** Model alias resolution failed or API key lacks access to requested model

**Root Cause Hypothesis:** Session requested a model (via `--model` flag or config) that doesn't exist or isn't accessible with current credentials.

---

## Failure Mode Ranking

### By Frequency

| Rank | Failure Mode | Count | Percentage |
|------|-------------|-------|------------|
| 1 | AI_NoOutputGeneratedError | 3 | 42.9% |
| 2 | TypeError in summarizeMessage | 2 | 28.6% |
| 3 | No user message in stream | 1 | 14.3% |
| 4 | ProviderModelNotFoundError | 1 | 14.3% |

### By Severity (Session State Impact)

| Rank | Failure Mode | Severity | Impact |
|------|-------------|----------|--------|
| 1 | No user message in stream | **Critical** | Invariant violation, session fundamentally broken |
| 2 | AI_NoOutputGeneratedError | **High** | No response generated, stream failed |
| 3 | TypeError in summarizeMessage | **Medium** | Summarization broken, session may continue |
| 4 | ProviderModelNotFoundError | **Low** | Single session fails, config issue |

### Combined Priority Matrix

| Failure Mode | Freq | Severity | Priority | Fix Difficulty |
|-------------|------|----------|----------|----------------|
| AI_NoOutputGeneratedError | High | High | **P1** | Hard (external dep) |
| TypeError in summarizeMessage | Medium | Medium | **P2** | Easy (null check) |
| No user message in stream | Low | Critical | **P2** | Medium (investigate cascade) |
| ProviderModelNotFoundError | Low | Low | **P3** | Easy (better error message) |

---

## Cross-Reference with Prior Investigations

### Jan 26 Investigation (2026-01-26-inv-opencode-server-keeps-crashing-dying.md)

**Already Analyzed:**
- ✅ All three error types from Jan 24 (E1, E2, E3) covered
- ✅ Correctly identified that unhandledRejection doesn't crash server
- ✅ Recommended fixes for summary.ts:69, prompt.ts:293, provider.ts:1082

**New Information from This Analysis:**
- E4 (AI_NoOutputGeneratedError) not in prior analysis - occurred after that investigation
- Temporal clustering pattern (30-second burst) not previously noted
- Priority ranking not previously quantified

### Jan 23 Investigation (2026-01-23-inv-opencode-server-crashes-under-load.md)

**Context:**
- Documented 5+ server restarts with no crash logs
- Led to implementation of crash logging (which produced crash.log we're analyzing)
- Identified missing process.on('uncaughtException') handlers

**Relationship:**
- Jan 23 investigation enabled this analysis by recommending crash handlers
- The crash.log exists because of that recommendation
- However, actual server crashes (process exit) are still not captured in crash.log - only unhandledRejections

---

## Synthesis

**Key Insights:**

1. **Most Errors Are External or Edge Cases** - AI_NoOutputGeneratedError (43%) comes from upstream SDK, not OpenCode code. Only the TypeError (28%) is a straightforward code fix.

2. **Temporal Clustering Suggests Environmental Triggers** - The three AI errors in 30 seconds and the two TypeErrors 5 hours apart on same day suggest transient conditions (API issues, session state) rather than systematic bugs.

3. **Cascade Failures Are Possible** - The 2ms gap between TypeError and "No user message" at same timestamp suggests one error can trigger another. Defensive programming needed.

4. **unhandledRejection ≠ Server Crash** - As noted in Jan 26 investigation, these errors leave sessions in bad state but don't crash the server. Agent death comes from SSE stream breaking on actual server crash (which these don't capture).

**Answer to Investigation Question:**

Four distinct failure modes exist in crash.log:

1. **AI_NoOutputGeneratedError** (P1, High frequency, High severity) - External SDK failure when API returns empty stream. Root cause: upstream API issues. Fix: retry logic with exponential backoff.

2. **TypeError in summarizeMessage** (P2, Medium frequency, Medium severity) - Code bug with missing null check. Root cause: message without `info` property. Fix: add null guard at summary.ts:69.

3. **No user message in stream** (P2, Low frequency, Critical severity) - Invariant violation, possibly cascading from summarization failure. Root cause: message stream corruption. Fix: investigate cascade from summarizeMessage, add defensive checks.

4. **ProviderModelNotFoundError** (P3, Low frequency, Low severity) - Configuration/access issue. Root cause: invalid model requested. Fix: better error message with valid model list.

---

## Structured Uncertainty

**What's tested:**

- ✅ Error categorization complete (verified: parsed all 7 entries from crash.log)
- ✅ Frequency counts accurate (verified: counted occurrences)
- ✅ Stack traces point to specific code locations (verified: read stack traces)
- ✅ Temporal clustering pattern (verified: timestamps 00:40:32, 00:40:47, 00:41:00)
- ✅ Memory not exhausted at crash time (verified: heapUsed 114MB, rss 336MB - stable)

**What's untested:**

- ⚠️ Actual fix for summary.ts:69 (not implemented)
- ⚠️ AI_NoOutputGeneratedError correlation with API outages (not cross-referenced)
- ⚠️ Cascade hypothesis between TypeError and "No user message" (inferred from timing, not traced)
- ⚠️ Whether retry logic would help AI errors (not tested)

**What would change this:**

- If crash.log showed uncaughtException events, would indicate actual server crashes
- If API logs showed outages at Jan 27 00:40, would confirm external trigger hypothesis
- If summary.ts code review shows expected message format, would clarify TypeError root cause

---

## Implementation Recommendations

**Purpose:** Consolidate recommendations from this analysis and prior investigations.

### Recommended Approach ⭐

**Phased Fix Strategy** - Address easy wins first, then tackle external dependencies.

**Why this approach:**
- Quick wins (null check) reduce error noise immediately
- External dependency issues (AI SDK) need more investigation
- Matches priority ranking from analysis

**Trade-offs accepted:**
- AI_NoOutputGeneratedError (most frequent) fixed last because it's hardest
- Accepting some continued errors while external fix is designed

**Implementation sequence:**
1. Fix TypeError: Add null check at summary.ts:69 for `msgWithParts?.info`
2. Fix ProviderModelNotFoundError: Add helpful error message with valid models
3. Investigate cascade: Trace from summarizeMessage to prompt.ts to confirm cascade
4. Add retry logic: Wrap stream operations in retry with exponential backoff

### Alternative Approaches Considered

**Option B: Fix External Dependency First (AI SDK)**
- **Pros:** Addresses most frequent error
- **Cons:** External dependency, may require SDK upgrade or workaround
- **When to use instead:** If AI errors increase in frequency

**Option C: Defensive Wrapping of All Operations**
- **Pros:** Catches all unhandled rejections
- **Cons:** May mask underlying issues, harder to debug
- **When to use instead:** As last resort if specific fixes insufficient

**Rationale for recommendation:** Phased approach maximizes value delivery while managing complexity. Easy wins first builds confidence.

---

### Implementation Details

**What to implement first:**
- `summary.ts:69` - Change `msgWithParts.info` to `msgWithParts?.info`
- Rebuild OpenCode fork after change

**Things to watch out for:**
- ⚠️ AI SDK is external dependency - updates may fix or break things
- ⚠️ Cascade failures mean one fix may reduce multiple error types
- ⚠️ Temporal clustering suggests environmental triggers - monitor for patterns

**Areas needing further investigation:**
- What specific API conditions trigger AI_NoOutputGeneratedError
- Whether prompt.ts:293 error is always cascade or can occur independently
- Memory/connection limits under sustained load (Jan 23 concern still valid)

**Success criteria:**
- ✅ crash.log no longer shows TypeError at summary.ts:69
- ✅ AI_NoOutputGeneratedError has retry logic with success metrics
- ✅ Error frequency reduces by 50%+ over next week

---

## References

**Files Examined:**
- `~/.local/share/opencode/crash.log` - Primary crash telemetry (7 entries)
- `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md` - Prior analysis
- `.kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md` - Crash logging origin

**Commands Run:**
```bash
# Read crash log
cat ~/.local/share/opencode/crash.log

# Cross-reference with prior investigations
cat .kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md
cat .kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md
```

**External Documentation:**
- Vercel AI SDK: https://sdk.vercel.ai/ (source of AI_NoOutputGeneratedError)

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md` - Comprehensive analysis including fix recommendations
- **Investigation:** `.kb/investigations/2026-01-23-inv-opencode-server-crashes-under-load.md` - Origin of crash logging implementation

---

## Investigation History

**2026-01-26:** Investigation started
- Initial question: Analyze crash.log for failure patterns and rank by frequency/severity
- Context: Issue orch-go-20944.1 requested systematic analysis

**2026-01-26:** Parsed crash.log
- Identified 7 entries with 4 distinct error types
- Noted temporal clustering (3 AI errors in 30 seconds)
- Cross-referenced with prior investigations

**2026-01-26:** Investigation completed
- Status: Complete
- Key outcome: Four failure modes identified, ranked by frequency and severity, with root cause hypotheses and phased fix recommendation
