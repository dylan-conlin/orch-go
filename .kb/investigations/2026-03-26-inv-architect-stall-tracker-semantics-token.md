## Summary (D.E.K.N.)

**Delta:** The timestamp-reset bug and token-type mismatch from the prior investigation are both already fixed; the remaining design issue is that `IsStalled` conflates 4 distinct failure signals (token stall, phase stall, never-started, stale spawn) and the CLI and dashboard set it from different subsets.

**Evidence:** Project compiles cleanly, all stall tracker tests pass (including `FrequentPollingDetectsStall`), commit `c4d4aa496` fixed timestamp reset, commit `5062bdf08` unified on `execution.TokenStats`. But `serve_agents_handlers.go` sets `IsStalled` from 4 code paths while `status_cmd.go` sets it from only 1 (token tracker).

**Knowledge:** This is a Defect Class 5 violation (Contradictory Authority Signals) — the same field carries different meanings across CLI and dashboard. Downstream consumers (StuckCollector, attention, status display) cannot distinguish which signal fired.

**Next:** Add `StallReason string` field to both `AgentAPIResponse` and `AgentInfo`, set it to the specific stall type at each code path, keep `IsStalled` as convenience boolean. Single implementation issue.

**Authority:** architectural — Crosses `cmd/orch` (CLI + dashboard), `pkg/attention` (consumer), and `pkg/daemon` (tracker). Multiple valid approaches exist.

---

# Investigation: Stall Tracker Semantics and Token Type Boundary

**Question:** Should `IsStalled` be redesigned to separate token-stall from phase-stall semantics, and what is the correct token type boundary for the stall tracker?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** Architect (orch-go-cig47)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-26-inv-stall-tracker-pkg-daemon-stall | extends | Yes — re-verified code and build state | Finding 2 (timestamp reset) was correct when written but has since been fixed by commit c4d4aa496 |
| 2026-02-28-audit-stalled-agent-failure-patterns | deepens | Yes — IsStalled ambiguity confirmed in audit findings | None |

---

## Findings

### Finding 1: Both cited bugs are already fixed

**Evidence:**
- Timestamp reset bug: Fixed in commit `c4d4aa496` ("fix: stall tracker timestamp reset on every poll masked truly stalled agents"). The `Update` method now only overwrites the snapshot timestamp when tokens increase, not on every poll. Test `TestStallTracker_FrequentPollingDetectsStall` validates this: 5 rapid polls at 50ms intervals with unchanged tokens followed by a 300ms sleep correctly triggers stall detection at 500ms threshold.
- Token type boundary: Unified in commit `5062bdf08` ("feat: migrate 54 files from pkg/opencode to pkg/execution SessionClient"). Both `stall_tracker.go`, its test file, and all callers (`status_cmd.go:354`, `serve_agents_handlers.go:423-429`) use `execution.TokenStats`. The adapter layer (`opencode_adapter.go:82-97`) converts `opencode.TokenStats` → `execution.TokenStats` at the boundary.
- `go build ./...` succeeds with no errors. `go test -run TestStall ./pkg/daemon/` passes all 7 tests.

**Source:** `pkg/daemon/stall_tracker.go:66-78`, `pkg/daemon/stall_tracker_test.go:117-150`, `pkg/execution/opencode_adapter.go:82-97`, `git log --oneline -10 -- pkg/daemon/stall_tracker.go`

**Significance:** The spawning investigation's two primary concerns (timestamp reset, type mismatch) are resolved. The remaining work is the semantic design issue the investigation flagged for architect review.

---

### Finding 2: `IsStalled` is set by 4 distinct code paths with different meanings

**Evidence:** In the dashboard handler (`serve_agents_handlers.go`), `IsStalled` is set `true` by:

| # | Code Path | Condition | Threshold | Location |
|---|-----------|-----------|-----------|----------|
| 1 | Token stall | `globalStallTracker.Update()` returns true | 3 min no token progress | `serve_agents_handlers.go:444` |
| 2 | Phase stall | `now.Sub(phaseReportedAt) > stalledThreshold` | 15 min no phase update | `serve_agents_handlers.go:187` |
| 3 | Never started | `agents[i].Reason == "never_started"` | Unconditional | `serve_agents_handlers.go:208` |
| 4 | Stale spawn | `elapsed > stalledThreshold` (no phase, spawned 15+ min ago) | 15 min since spawn | `serve_agents_handlers.go:213` |

In the CLI (`status_cmd.go`), `IsStalled` is set only by path #1 (token stall). The CLI handles phase-timeout separately as `isUnresponsive` inline logic (`status_cmd.go:287-291`).

The doc comments disagree: `serve_agents_types.go:25` says "same phase for 15+ minutes", `status_format.go:108` says "no token progress for 3+ minutes".

**Source:** `cmd/orch/serve_agents_handlers.go:187,208,213,444`, `cmd/orch/status_cmd.go:363`, `cmd/orch/serve_agents_types.go:25`, `cmd/orch/status_format.go:108`

**Significance:** This is a Defect Class 5 violation (Contradictory Authority Signals). The same field carries different meanings depending on whether you're reading the dashboard API or the CLI. Downstream consumers like `StuckCollector` (which reads `is_stalled` from JSON) have no way to know which signal triggered.

---

### Finding 3: The dashboard already has a cleaner signal (`IsUnresponsive`) that partially deconflates

**Evidence:** `serve_agents_handlers.go:193` sets `IsUnresponsive = true` when `elapsed > 30min && !IsProcessing`. The CLI also computes `isUnresponsive` at `status_cmd.go:288-291`. `serve_agents_types.go:26` documents it as "no phase update for 30+ minutes". This separates the "no phase for a long time" signal from the general "stalled" signal, but only at the 30-minute threshold — the 15-minute phase stall still routes to `IsStalled`.

The `StuckAgentItem` in `pkg/attention/stuck_collector.go:47` reads `IsStalled` but not `IsUnresponsive`, meaning the attention system sees the conflated signal.

**Source:** `cmd/orch/serve_agents_handlers.go:193-195`, `cmd/orch/status_cmd.go:287-291`, `pkg/attention/stuck_collector.go:47`

**Significance:** Partial deconflation exists but is incomplete. The 15-minute phase stall is the most common trigger for `IsStalled` in the dashboard path, and it's the one that collides with the 3-minute token stall.

---

### Finding 4: A third "stalled" concept in daemon self-health adds terminology confusion

**Evidence:** `pkg/daemon/status.go:213` computes `stalledThreshold := pollInterval * 2` and returns `"stalled"` when the daemon hasn't polled within 2x its expected interval. This is daemon liveness detection (is the polling loop itself hung?), not agent stall detection. Same word, completely different meaning.

**Source:** `pkg/daemon/status.go:211-216`

**Significance:** Minor terminology collision. Three distinct concepts share the "stalled" label: token stall (agent not generating), phase stall (agent not reporting), and poll stall (daemon not polling). The first two are conflated in code; the third is separate but uses the same word.

---

## Synthesis

**Key Insights:**

1. **The original bugs are resolved; the remaining issue is a semantic design problem** — The timestamp reset (Finding 1) was fixed in `c4d4aa496`, and the type boundary was cleaned up in `5062bdf08`. The compile failures and detection failures that motivated this architect review no longer exist. What remains is the `IsStalled` overloading (Finding 2).

2. **The field conflation creates a Defect Class 5 violation** — `IsStalled` means "token stall" in the CLI and "token stall OR phase stall OR never-started OR stale spawn" in the dashboard. This is exactly the "Multiple sources of truth disagree" pattern from the defect taxonomy. The StuckCollector and attention system consume the overloaded signal with no way to reason about which condition triggered.

3. **The fix is additive, not restructuring** — The existing `IsStalled` boolean remains as a backward-compatible convenience. Adding a `StallReason` string field at each code path gives consumers the distinction they need without changing wire format for existing consumers.

**Answer to Investigation Question:**

Yes, `IsStalled` should be deconflated — not by removing or replacing the boolean, but by adding a `StallReason` string that encodes which stall signal fired. The token type boundary is already correct (`execution.TokenStats` everywhere) and needs no changes. The recommended approach is a single implementation issue with ~4 touch points.

---

## Structured Uncertainty

**What's tested:**

- ✅ Project compiles cleanly (`go build ./...` — zero errors)
- ✅ Stall tracker tests pass including cumulative detection (`go test -run TestStall ./pkg/daemon/` — 7/7 pass, 6.7s)
- ✅ Token type is consistent across tracker, callers, and tests (verified by grep and reading all files)
- ✅ `IsStalled` is set from 4 code paths in dashboard handler (traced all 4 in `serve_agents_handlers.go`)
- ✅ CLI sets `IsStalled` from 1 code path only (traced in `status_cmd.go:363`)

**What's untested:**

- ⚠️ Production frequency of each stall type is unknown — which of the 4 paths fires most often in practice
- ⚠️ Whether any dashboard UI JS code branches on `is_stalled` specifically (could be affected by adding `stall_reason`)
- ⚠️ Whether the StuckCollector's behavior would materially change if it could distinguish stall types

**What would change this:**

- If production data showed token stalls dominate (>90% of `IsStalled` triggers), the deconflation would be lower priority
- If the dashboard web UI renders stall differently per type already via other fields, the JSON-level fix may not be needed
- If a consumer outside the traced paths relies on `IsStalled` meaning "any kind of stall", the additive approach stays safe but the StallReason values need documentation

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add `StallReason` field and set it at each code path | architectural | Crosses dashboard types, CLI types, and attention consumer; multiple valid approaches |
| Align CLI and dashboard IsStalled semantics | architectural | Same field, different behavior across two consumer surfaces |
| Rename daemon poll stall terminology | implementation | Internal to `pkg/daemon/status.go`, no cross-boundary impact |

### Recommended Approach ⭐

**Add `StallReason string` alongside `IsStalled bool`** — At each code path that sets `IsStalled = true`, also set `StallReason` to a typed string constant indicating which signal fired. Keep `IsStalled` as the backward-compatible convenience boolean.

**Why this approach:**
- Resolves Defect Class 5 (Contradictory Authority Signals) by giving consumers a way to distinguish stall types.
- Zero breaking change — all existing consumers of `IsStalled: true` continue working.
- Single field addition to two structs (`AgentAPIResponse`, `AgentInfo`) plus 4 setter sites in handlers.
- Enables future differentiation in the dashboard UI, attention routing, and CLI display without further schema changes.

**Trade-offs accepted:**
- Does not restructure the `IsStalled` boolean itself — it remains set from all 4 paths in the dashboard handler.
- Adds a field that is only meaningful when `IsStalled` is true (acceptable: standard "reason" pattern).

**Implementation sequence:**
1. Define `StallReason` constants: `"token_stall"`, `"phase_stall"`, `"never_started"`, `"spawn_stale"`.
2. Add `StallReason string` to `AgentAPIResponse` and `AgentInfo` (with JSON tag `"stall_reason,omitempty"`).
3. At each code path that sets `IsStalled = true`, also set `StallReason` to the appropriate constant.
4. Align CLI path: set `StallReason` in `status_cmd.go` when `globalStallTracker.Update` returns stalled.
5. Update `StuckAgentItem` in `pkg/attention/stuck_collector.go` to include `StallReason` for downstream use.
6. Update doc comments on both `IsStalled` fields to reflect the combined semantics accurately.

### Alternative Approaches Considered

**Option B: Replace `IsStalled bool` with `StallType string`**
- **Pros:** Cleaner — one field instead of two, no boolean/string redundancy.
- **Cons:** Breaking change for all JSON consumers. Dashboard JS, attention collector, and any external tooling reading `is_stalled` would need updates. Higher coordination cost for no material capability gain.
- **When to use instead:** If this were a fresh design with no existing consumers.

**Option C: Leave `IsStalled` as-is, document the ambiguity**
- **Pros:** Zero code change. Document that `IsStalled` means "any kind of health concern."
- **Cons:** Does not fix the Defect Class 5 violation. Consumers still cannot distinguish stall types. The CLI/dashboard semantic gap persists.
- **When to use instead:** If stall disambiguation has no operational value (but the investigation that triggered this review says it does).

**Rationale for recommendation:** Option A is the only approach that resolves the semantic ambiguity without breaking existing consumers. It's additive, low-risk, and creates the foundation for better attention routing.

---

### Implementation Details

**What to implement first:**
- Define constants and add the `StallReason` field to both response types.
- Update the 4 setter sites in `serve_agents_handlers.go` and 1 in `status_cmd.go`.
- Update `StuckAgentItem` in `pkg/attention/stuck_collector.go` to pass through `StallReason`.

**Things to watch out for:**
- ⚠️ The `IsStalled` setter sites in the dashboard handler are spread across enrichment (lines 187, 208, 213) and token-fetch result processing (line 444) — they're in different phases of the handler, so a stall can be set twice (phase stall at line 187, then overwritten conceptually by token stall at 444 or vice versa). The `StallReason` should capture the most significant stall, or the last one set. Consider: if both phase stall and token stall fire, `StallReason` should prefer `token_stall` (more specific signal).
- ⚠️ The CLI path (`status_cmd.go`) handles phase timeout as `isUnresponsive` inline, not as `IsStalled`. To align, the CLI should set `StallReason = "phase_stall"` where appropriate, OR accept the intentional divergence (CLI only cares about token stalls for `IsStalled`).
- ⚠️ `pkg/daemon/status.go:213` uses "stalled" for daemon poll health — renaming to `"polling_delayed"` is independent and low-risk but should be done in the same pass to avoid future confusion.

**Areas needing further investigation:**
- Dashboard web UI consumption of `is_stalled` — does the Svelte frontend branch on this field?
- Whether `StallReason` should be surfaced in the dashboard UI (e.g., different badge colors per stall type).

**Success criteria:**
- ✅ `StallReason` field populated with a typed value at every code path that sets `IsStalled = true`
- ✅ CLI and dashboard doc comments aligned on `IsStalled` meaning
- ✅ `go test ./pkg/daemon ./cmd/orch ./pkg/attention` compiles and passes
- ✅ `/api/agents` JSON response includes `stall_reason` when `is_stalled` is true

---

## References

**Files Examined:**
- `pkg/daemon/stall_tracker.go` — Core token-based stall detection logic, confirmed timestamp fix
- `pkg/daemon/stall_tracker_test.go` — 7 tests including FrequentPollingDetectsStall, all pass
- `pkg/daemon/status.go` — Third "stalled" concept (daemon poll health), line 213
- `cmd/orch/serve_agents_handlers.go` — Dashboard handler, 4 `IsStalled` setter sites (lines 187, 208, 213, 444)
- `cmd/orch/serve_agents_status.go` — Global stall tracker creation (line 225)
- `cmd/orch/serve_agents_types.go` — `AgentAPIResponse` struct, `IsStalled` doc comment (line 25)
- `cmd/orch/status_cmd.go` — CLI status, single `IsStalled` setter (line 363), separate `isUnresponsive` logic (line 288)
- `cmd/orch/status_format.go` — `AgentInfo` struct, `IsStalled` doc comment (line 108), rendering (line 350)
- `pkg/attention/stuck_collector.go` — `StuckAgentItem` consumer of `IsStalled` (line 47)
- `pkg/execution/types.go` — `execution.TokenStats` definition (line 87)
- `pkg/opencode/client_tokens.go` — `opencode.TokenStats` definition (line 4)
- `pkg/execution/opencode_adapter.go` — Type conversion boundary (lines 82-97)

**Commands Run:**
```bash
# Verify project compiles
go build ./...

# Run stall tracker tests
go test -v -count=1 -run TestStall ./pkg/daemon/

# Check stall tracker git history
git log --oneline -10 -- pkg/daemon/stall_tracker.go

# Check timestamp fix commit details
git show c4d4aa496 --stat
```

**External Documentation:**
- None

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-26-inv-stall-tracker-pkg-daemon-stall.md` — Prior investigation that identified the bugs and recommended architect review
- **Investigation:** `.kb/investigations/2026-02-28-audit-stalled-agent-failure-patterns.md` — Broader stall pattern audit
- **Model:** `.kb/models/defect-class-taxonomy/model.md` — Defect Class 5 (Contradictory Authority Signals) taxonomy

---

## Investigation History

**[2026-03-26]:** Investigation started
- Initial question: Should IsStalled be redesigned to separate token-stall from phase-stall semantics, and what is the correct token type boundary?
- Context: Prior investigation (2026-03-26-inv-stall-tracker-pkg-daemon-stall) found timestamp reset bug and type mismatch, recommended architect review.

**[2026-03-26]:** Verified prior investigation claims against current code
- Discovered both bugs (timestamp reset, type mismatch) were already fixed by commits c4d4aa496 and 5062bdf08.
- All tests pass, project compiles cleanly.

**[2026-03-26]:** Traced all IsStalled setter paths
- Found 4 distinct code paths in dashboard handler vs 1 in CLI — Defect Class 5 violation.
- Found third "stalled" concept in daemon self-health (pkg/daemon/status.go).

**[2026-03-26]:** Investigation completed
- Status: Complete
- Key outcome: Both prior bugs fixed. Remaining issue is IsStalled semantic overloading — recommend adding StallReason string field.
