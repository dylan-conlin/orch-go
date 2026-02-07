# Resilient Infrastructure Patterns

**Pattern:** Critical Paths Need Escape Hatches

**Created:** 2026-01-10
**Origin:** Dashboard reliability crisis - OpenCode server crashes killed agents building observability fixes

---

## The Problem: Self-Modifying Systems

When building or fixing infrastructure, you often depend on that infrastructure:
- Deploying new code requires the deployment system
- Monitoring failures requires the monitoring system
- Fixing crash recovery requires the system not to crash

**Failure mode:** Infrastructure fails → can't fix infrastructure → death spiral

---

## The Pattern: Primary Path + Escape Hatch

### Architecture

```
┌─────────────────────────────────────┐
│     Primary Path (Optimized)        │
│  - High performance/concurrency     │
│  - Depends on infrastructure        │
│  - Normal operations                │
└─────────────────────────────────────┘
              ↓
         Infrastructure
              ↓
┌─────────────────────────────────────┐
│    Escape Hatch (Independent)       │
│  - Degraded performance OK          │
│  - Independent of infrastructure    │
│  - Recovery/critical work           │
└─────────────────────────────────────┘
```

### Escape Hatch Criteria

An effective escape hatch must be:

1. **Independent:** Doesn't depend on what can fail
   - Different execution path
   - Different dependencies
   - Different failure modes

2. **Visible:** Provides observability
   - Can monitor progress
   - Can intervene if stuck
   - Clear failure indicators

3. **Capable:** Can complete the work
   - Sufficient resources/quality
   - Access to required tools
   - Authority to make changes

---

## Implementation Examples

### Example 1: orch Spawn Modes

**Context:** Building observability infrastructure while OpenCode server crashes

**Primary path (daemon):**
- Execution: OpenCode HTTP API → headless agents
- Benefits: High concurrency, automated, batch processing
- Dependency: OpenCode server must be running
- Failure: Server crashes → all agents die → can't build fixes

**Escape hatch (manual):**
- Execution: Claude CLI → tmux windows → direct process
- Benefits: Crash-resistant, visible, can intervene
- Independence: Doesn't use OpenCode server
- Cost: Manual management, lower concurrency

**Usage:**
```bash
# Primary (normal workflow)
bd create "task" -l triage:ready
orch daemon run

# Escape hatch (critical infrastructure work)
orch spawn --bypass-triage --mode claude --model opus --tmux \
  feature-impl "fix crash recovery" --issue ID
```

**Outcome:** Agents survived 3 server crashes, completed observability fixes

### Example 2: Overmind Supervision

**Context:** Dashboard services crash frequently, need supervision

**Primary path (services):**
- Execution: overmind → api/web/opencode services
- Benefits: Unified management, atomic restart, clean logs
- Dependency: overmind process must be running
- Failure: overmind crashes → all services down → no recovery

**Escape hatch (supervision):**
- Execution: launchd → overmind
- Benefits: OS-level recovery, automatic restart
- Independence: launchd is separate from overmind
- Cost: Platform-specific (macOS/Linux differ)

**Architecture:**
```
launchd (OS supervisor)
  ↓ supervises + auto-restarts
overmind (process manager)
  ↓ manages
services (api, web, opencode)
  ↓ monitored by
service monitor (in orch serve)
```

**Key insight:** Each layer has a supervisor at the layer above, no circular dependencies

### Example 3: Kubernetes (Industry Standard)

**Primary path:**
- API server handles requests
- Single control plane node

**Escape hatch:**
- Multiple API server replicas
- etcd cluster (3-5 nodes)
- Can lose N-1 nodes and continue

**Pattern:** Same as ours, applied at scale

---

## When to Apply This Pattern

### Apply when:

✅ **Building self-modifying systems**
- Infrastructure that deploys itself
- Monitoring that monitors monitoring
- Recovery systems that need recovery

✅ **Critical path with single point of failure**
- Deployment pipeline
- Authentication system
- Monitoring infrastructure

✅ **High cost of downtime**
- Revenue-generating systems
- Development productivity tools
- Customer-facing services

### Don't apply when:

❌ **Low criticality**
- Internal tools with manual workarounds
- Non-essential features
- Rarely-used functionality

❌ **Simple systems**
- No circular dependencies
- Minimal failure modes
- Easy manual recovery

❌ **Over-engineering**
- Adding complexity without evidence of need
- "Just in case" redundancy
- Premature optimization

---

## Anti-Patterns

### ❌ Single Path Dependency

**Problem:** Only one way to do critical work
```
Deploy system requires deployment system (no escape)
```

**Fix:** Add independent path (manual deploy, blue/green, canary)

### ❌ Circular Dependencies

**Problem:** A supervises B, B supervises A
```
Service monitor polls overmind
  ↓
overmind crashes
  ↓
Service monitor can't detect (depends on overmind)
```

**Fix:** Layer supervision (launchd → overmind → services → monitor)

### ❌ Invisible Escape Hatch

**Problem:** Escape hatch exists but no visibility
```
Fallback system runs but can't tell if it's working
```

**Fix:** Add logging, metrics, or visual indicators (like tmux windows)

---

## Design Questions

When designing critical infrastructure, ask:

1. **What happens if this system fails while being fixed?**
   - Can recovery proceed?
   - Is there an independent path?

2. **Does the deployment/monitoring path depend on what it deploys/monitors?**
   - Circular dependency?
   - Can the supervisor supervise itself?

3. **Is there visibility into the recovery path?**
   - Can I tell if it's working?
   - Can I intervene if stuck?

4. **What's the cost of this redundancy?**
   - Worth it for criticality?
   - Simpler alternatives exist?

---

## Related Patterns

- **Layered Supervision:** Each layer supervised by layer above (systemd → docker → containers)
- **Blue/Green Deployment:** Old system stays up while deploying new
- **Circuit Breaker:** Automatic fallback when primary fails
- **Graceful Degradation:** Reduced functionality better than no functionality

---

## Success Criteria

An escape hatch is working when:

✅ **Used during crisis**
- Actually used when primary failed
- Successfully completed critical work
- Prevented death spiral

✅ **Discoverable**
- Team knows it exists
- Documentation clear
- Obvious when to use

✅ **Maintainable**
- Tested regularly
- Doesn't bit-rot
- Updated alongside primary path

---

## Real-World Validation (Jan 10, 2026)

**Scenario:** Building observability infrastructure (orch doctor, overmind supervision, Phase 2 dashboard)

**Primary path failure:**
- OpenCode server crashed 3 times in 1 hour
- Each crash killed all active agents
- Agents were building the fixes for crash recovery
- Death spiral: can't fix crashes because crashes kill the fixes

**Escape hatch activation:**
```bash
# Abandoned crashed opencode agents
orch abandon orch-go-uyveu orch-go-vtf1s orch-go-b6hwn

# Respawned with escape hatch
orch spawn --bypass-triage --mode claude --model opus --tmux \
  feature-impl "P0: Implement orch doctor" --issue orch-go-uyveu

# Result: Agents survived subsequent crashes, continued working
```

**Outcome:**
- 3 agents running crash-resistant in tmux windows
- Visible progress (can attach and watch)
- Independent of OpenCode server
- Building the supervision infrastructure that will prevent future crashes

**Lesson:** Escape hatch broke the death spiral. The "old spawn path" wasn't legacy - it was disaster recovery.

---

## References

- **kb-bf4f55:** Constraint "Critical paths need independent escape hatches"
- **kb-d562c9:** Decision to use claude+opus+tmux for critical infrastructure
- **CLAUDE.md:** "Dual Spawn Modes: Resilience by Design" section
- **Origin issue:** orch-go-95vz4 (Dashboard Reliability Infrastructure epic)
