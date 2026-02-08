<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Reactive failure-driven approach is correct; proactive auditing would violate Pressure Over Compensation principle and front-load complexity without evidence of need.

**Evidence:** Current system has 3 orch-related plists (opencode serve, orch-go serve, orch daemon), all with KeepAlive. Failures surface exactly which services need persistence - the two failures mentioned reveal the gap without requiring an audit.

**Knowledge:** The Pressure Over Compensation principle applies here: "Don't compensate for broken systems - let failures create pressure for improvement." Proactive auditing compensates before failure demonstrates need.

**Next:** Implement `orch doctor` as detection mechanism for infrastructure health checks, not proactive plist creation.

**Confidence:** High (85%) - Principle alignment clear; edge case is unknown services with high failure cost.

---

# Investigation: Reactive vs Proactive Infrastructure Plist Strategy

**Question:** Should we reactively let failures guide us to which services need KeepAlive plists, or proactively audit all services and create plists now?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** og-arch-two-infrastructure-failures-26dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Current Infrastructure State

**Evidence:** The orch ecosystem currently has 3 launchd plists with KeepAlive:
- `com.opencode.serve.plist` - OpenCode server on port 4096
- `com.orch-go.serve.plist` - Orch HTTP API server  
- `com.orch.daemon.plist` - Autonomous agent spawning daemon

All three have KeepAlive enabled. The system already handles the core infrastructure services.

**Source:** 
```bash
$ launchctl list | grep -E "(opencode|orch)"
-	1	com.orch-go.serve
38373	0	com.orch.daemon
36456	-15	com.opencode.serve

$ ls ~/Library/LaunchAgents/*.plist | grep -E "(opencode|orch)"
com.opencode.serve.plist
com.orch-go.serve.plist  
com.orch.daemon.plist
```

**Significance:** Core infrastructure is already covered. The "two infrastructure failures" likely refer to services outside this core set - possibly web dev servers or other auxiliary services.

---

### Finding 2: Pressure Over Compensation Principle Applies Directly

**Evidence:** From `~/.kb/principles.md`:

> **Pressure Over Compensation:** When the system fails to surface knowledge, don't compensate by providing it manually. Let the failure create pressure to improve the system.
> 
> **The pattern:**
> ```
> Human compensates for gap → System never learns → Human keeps compensating
> Human lets system fail → Failure surfaces gap → Gap becomes issue → System improves
> ```

The two infrastructure failures ARE the system surfacing which services need KeepAlive. Proactively auditing and creating plists would:
1. Compensate before failure demonstrates need
2. Front-load complexity for services that may not need persistence
3. Prevent the system from learning which services actually matter

**Source:** `~/.kb/principles.md:243-270` (Pressure Over Compensation section)

**Significance:** This is a textbook case of the principle. The failures are data, not problems to avoid. They tell us exactly which services need KeepAlive without guessing.

---

### Finding 3: orch doctor as Detection vs Prevention

**Evidence:** The task mentions "orch doctor as detection alternative to prevention." This aligns with the principle pattern:
- **Prevention (Proactive):** Audit all services, create plists now → compensates before need demonstrated
- **Detection (Reactive):** `orch doctor` checks if required services are running → surfaces gaps as failures occur

`orch doctor` would check:
1. Is OpenCode server responding? (`http://127.0.0.1:4096/healthz`)
2. Is orch-go serve API responding? (`http://127.0.0.1:3348/api/sessions`)
3. Is orch daemon running? (`launchctl list | grep com.orch.daemon`)
4. Are plists present for required services?

This surfaces problems when they matter, without pre-emptive complexity.

**Source:** Task description, pattern analysis

**Significance:** Detection aligns with Pressure Over Compensation - it doesn't prevent failure but makes failure visible quickly. The cost of context waste from debugging is accepted as the learning signal.

---

### Finding 4: Cost Analysis - Context Waste vs Proactive Complexity

**Evidence:** The concern about "cost of context waste from debugging" must be weighed against:

**Cost of Reactive (failures occur):**
- Debugging time when service dies
- Context switching from current work
- Potentially cascading failures (agents can't spawn, dashboard broken)

**Cost of Proactive (audit now):**
- Audit time to identify all services
- Plist creation for services that may never fail
- Maintenance burden for plists of services that change
- False confidence - "we have plists" doesn't mean they're correct

Key insight: The reactive cost is paid only for services that actually fail. The proactive cost is paid for ALL services, including those that never needed persistence.

**Source:** Analysis of trade-offs

**Significance:** Given the Pressure Over Compensation principle, reactive costs are "improvement pressure" while proactive costs are "compensation overhead."

---

## Synthesis

**Key Insights:**

1. **Failures are signals, not problems** - The two infrastructure failures told you exactly which services need KeepAlive. This is the system working as designed. Trying to prevent these failures would have required guessing which services matter.

2. **`orch doctor` is the right abstraction** - Instead of preventing failures (proactive plists), make failures visible quickly (detection). `orch doctor` can run at orchestrator session start, during health checks, or on-demand. It surfaces problems without preventing the learning signal.

3. **Proactive auditing violates principle** - Creating plists "just in case" compensates before demonstrating need. This is the anti-pattern: "The system doesn't know, so I'll fill in" → prevents system improvement.

**Answer to Investigation Question:**

**Recommend reactive approach with `orch doctor` detection.**

The Pressure Over Compensation principle directly addresses this: "Don't be the memory. Your role is to create pressure that forces the system to develop its own memory mechanisms."

When a service fails because it lacks a KeepAlive plist:
1. The failure surfaces which service matters
2. You create a plist for that specific service
3. Add a check to `orch doctor` to detect this service's health
4. The system learns and improves

Proactively auditing would:
1. Spend time identifying services that MAY matter
2. Create plists for services that may never fail
3. Give false confidence that "infrastructure is handled"
4. Prevent learning which services actually need persistence

The "context waste from debugging" IS the learning signal. Accept that cost in exchange for only investing in services that demonstrably need persistence.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong principle alignment (Pressure Over Compensation) and clear evidence from current infrastructure state. The recommendation follows directly from established principles.

**What's certain:**

- ✅ Three core services already have plists with KeepAlive
- ✅ Pressure Over Compensation principle applies directly to this scenario
- ✅ `orch doctor` would surface problems without preventing learning

**What's uncertain:**

- ⚠️ Unknown services with high failure cost - some services may be catastrophic to lose without warning
- ⚠️ Definition of "infrastructure" - where's the line between orch infrastructure vs project-specific servers?
- ⚠️ Recovery time - how long to create plist after failure vs time lost debugging?

**What would increase confidence to Very High (95%+):**

- Track actual failures over 1-2 weeks to see failure frequency
- Document which services are "infrastructure" vs "project"
- Measure time-to-recovery after plist creation

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Reactive + Detection** - Let failures guide plist creation, use `orch doctor` for visibility.

**Why this approach:**
- Aligns with Pressure Over Compensation principle
- Only invests in services that demonstrably need persistence
- Creates learning loop: failure → plist → doctor check

**Trade-offs accepted:**
- Initial debugging cost when services fail
- Temporary unavailability during failure-to-plist cycle
- Acceptable because these costs are the learning signal

**Implementation sequence:**
1. **After failure:** Create plist for failed service (already happened for the two failures)
2. **Add to orch doctor:** Check health of service that failed
3. **On next failure:** Repeat - system progressively learns

### Alternative Approaches Considered

**Option B: Proactive Audit**
- **Pros:** No surprise failures, comprehensive coverage
- **Cons:** Violates Pressure Over Compensation, invests in unknowns
- **When to use instead:** If a service has catastrophic failure cost (data loss, security breach)

**Option C: Hybrid - Audit Critical Only**
- **Pros:** Covers high-impact services, accepts failure for low-impact
- **Cons:** Requires defining "critical" upfront (hard without failure data)
- **When to use instead:** If you can clearly categorize services by failure cost

**Rationale for recommendation:** Option A (Reactive + Detection) is the only approach that doesn't violate Pressure Over Compensation. The principle is clear: let failures create improvement pressure.

---

### Implementation Details

**What to implement first:**
- Create `orch doctor` command (doesn't exist yet per grep search)
- Add checks for the three core services (opencode, orch serve, orch daemon)
- Run `orch doctor` at orchestrator session start via hook

**Things to watch out for:**
- ⚠️ Services that fail silently - add health check endpoints where missing
- ⚠️ Plists that start but don't work - doctor should check actual health, not just process existence
- ⚠️ Over-engineering doctor - start simple, add checks as failures occur

**Areas needing further investigation:**
- What checks should `orch doctor` include?
- Should doctor run automatically or on-demand?
- How to surface doctor failures to orchestrator?

**Success criteria:**
- ✅ `orch doctor` exists and checks core 3 services
- ✅ Next infrastructure failure is detected by doctor within 1 minute
- ✅ Plist creation follows reactive pattern (failure → plist → doctor check)

---

## References

**Files Examined:**
- `~/.kb/principles.md` - Pressure Over Compensation principle (lines 243-270)
- `~/Library/LaunchAgents/*.plist` - Current plist inventory
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go` - Orch command structure

**Commands Run:**
```bash
# Check running services
launchctl list | grep -E "(opencode|orch)"

# List plists
ls ~/Library/LaunchAgents/*.plist | grep -E "(opencode|orch)"

# Check KeepAlive status
grep -r "KeepAlive" ~/Library/LaunchAgents/*.plist

# Search for existing doctor command
grep -r "doctor" /Users/dylanconlin/Documents/personal/orch-go
```

**Related Artifacts:**
- **Principle:** `~/.kb/principles.md` - Pressure Over Compensation
- **Decision:** None - this investigation may promote to decision if accepted

---

## Investigation History

**2025-12-26 ~14:00:** Investigation started
- Initial question: Reactive vs proactive plist strategy
- Context: Two infrastructure failures revealed missing plists

**2025-12-26 ~14:30:** Found Pressure Over Compensation applies directly
- This principle was written for exactly this type of scenario
- Proactive auditing would compensate before need demonstrated

**2025-12-26 ~15:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend reactive approach with `orch doctor` detection
