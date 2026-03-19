# Model: Agent Trust Enforcement

**Domain:** Agent execution constraints / OPSEC / Environment control
**Last Updated:** 2026-03-19
**Validation Status:** WORKING HYPOTHESIS — synthesized from 9 investigations + 3 threads across scs-special-projects. Core patterns validated against two production incidents (OshCut detection Mar 6, custom OPSEC replacement Mar 19). 17 testable claims: 11 confirmed, 2 open, 1 untested, 1 partially confirmed.
**Synthesized From:**
- `scs-special-projects/.kb/investigations/2026-03-19-inv-audit-orch-go-against-claude.md` — Lifecycle-phase separation: orch-go adds pre-spawn + OS-level enforcement absent from Claude Code
- `scs-special-projects/.kb/investigations/2026-03-19-inv-evaluate-claude-code-native-sandbox.md` — srt replaces custom OPSEC; allowlist > blocklist
- `scs-special-projects/.kb/investigations/2026-03-19-inv-audit-orch-go-reinvented-wheels.md` — <5% OSS overlap; value is in cross-cutting glue, not reimplemented algorithms
- `scs-special-projects/.kb/investigations/2026-03-19-architect-layer1-local-opsec-enforcement.md` — Custom sandbox-exec + tinyproxy architecture (superseded by srt)
- `scs-special-projects/.kb/investigations/2026-03-18-inv-investigate-agent-execution-environment-bypass.md` — 6 bypass classes; probe pw-w8r0 caused OshCut detection via bare-IP curl
- `scs-special-projects/.kb/investigations/2026-03-17-inv-investigate-bypass-surface-toolshed-price.md` — CompetitorProxyEnforcement is Faraday-only; 6 bypass classes
- `scs-special-projects/.kb/investigations/2026-03-17-inv-investigate-toolshed-pii-exposure-ai.md` — Anonymization works server-side (controlled), fails on passthrough (uncontrolled)
- `scs-special-projects/.kb/investigations/2026-03-17-inv-investigate-toolshed-slave-db-credential.md` — DB credential 12.6x over-privileged; application-level vs structural write prevention
- `scs-special-projects/.kb/investigations/2026-03-18-redteam-toolshed-security-assessment.md` — Auth stack solid; AI integration layer has CORS, prompt injection, anonymization gaps
- `scs-special-projects/.kb/threads/2026-03-19-environment-control-as-agent-trust.md` — Trust is structural, not verbal; 4-layer enforcement model
- `scs-special-projects/.kb/threads/2026-03-19-policy-layer-vs-enforcement-layer.md` — Policy (WHAT) vs enforcement (HOW); orch-go owns policy, platforms own enforcement
- `scs-special-projects/.kb/threads/2026-03-19-risk-value-asymmetry-sp-role.md` — Infrastructure-backed answers vs assurance-backed answers

**Related Models:**
- `.kb/models/architectural-enforcement/model.md` — Four-layer gate mechanisms for code quality (complementary domain)
- `.kb/models/claude-code-agent-configuration/model.md` — Claude Code configuration layer

---

## Summary (30 seconds)

An agent is only as trustworthy as the environment it runs in. Instructions ("don't hit competitor APIs") are the weakest form of trust — a convention-layer constraint that decays under task pressure and can be bypassed by any agent with shell access. Trust that matters is structural: network isolation, OS-level file immutability, allowlist-based domain restriction. The key architectural distinction is **policy layer** (what constraints each agent needs — spawn gates, skill routing, completion checks) vs **enforcement layer** (how constraints are enforced — sandbox, hooks, permissions, OS primitives). Orch-go should own policy decisions; agent platforms (Claude Code, future alternatives) should own enforcement mechanisms.

The evidence comes from a production incident: a probe agent used curl to create 6 OshCut accounts from bare IP, contributing to detection — despite convention-layer instructions prohibiting competitor API access. The fix was not a stronger instruction but a structural change: Claude Code's native sandbox (`srt`) with allowlist-based domain restriction makes non-allowed domains unreachable at the OS level. This pattern generalizes: every enforcement domain (network, filesystem, credentials, data access) follows the same hierarchy from convention (weakest) to infrastructure (strongest).

---

## Core Mechanism

### The Trust Hierarchy (4 Layers)

Trust enforcement operates through four layers. Each subsequent layer is harder to bypass but more expensive to implement:

| Layer | Mechanism | What It Controls | Bypass Difficulty | Example |
|-------|-----------|-----------------|-------------------|---------|
| **L4: Convention** | CLAUDE.md instructions, skill guidance | Agent behavior via prompt | Trivial — agent with shell access ignores it | "Don't hit competitor APIs" in CLAUDE.md |
| **L3: Application** | Hooks, deny rules, middleware | Tool-level actions | Moderate — bypassed by using different tools | `permissions.deny`, CompetitorProxyEnforcement (Faraday-only) |
| **L2: Environment** | Proxy vars, session parameters | Process-level defaults | Moderate — bypassed by tools that ignore env vars | `HTTP_PROXY`, `default_transaction_read_only` |
| **L1: Infrastructure** | Sandbox, OS locks, network isolation | Process tree capabilities | Hard — OS-level, no application bypass | `chflags uchg`, `srt` allowedDomains, container network namespace |

**Critical property:** Higher layers are necessary but insufficient. L4 catches well-intentioned agents that forgot a rule. L1 catches everything, including agents that found creative workarounds. Defense-in-depth means using ALL layers, not choosing one.

**Note on L2/L3 convergence:** With the native sandbox (srt) replacing custom OPSEC, L2 (Environment) is increasingly absorbed into L3 (Application). The `--settings` flag carries sandbox config that controls both application-layer deny rules AND environment-level network restrictions. In practice, L2 and L3 collapse into a single "platform configuration" layer delivered via the settings.json adapter. The 4-layer model remains useful conceptually (different bypass characteristics) even though the implementation boundary has shifted.

### Policy Layer vs Enforcement Layer

The most important architectural distinction in this model:

| Aspect | Policy Layer | Enforcement Layer |
|--------|-------------|------------------|
| **Question** | WHAT constraints does this agent need? | HOW are constraints enforced? |
| **Owner** | Orch-go (orchestration system) | Agent platform (Claude Code, Codex, etc.) |
| **Examples** | Spawn gates, skill selection, hotspot routing, completion verification | Sandbox config, `--disallowedTools`, `--settings`, OS primitives |
| **Portability** | Platform-agnostic (survives platform migration) | Platform-specific (different for each platform) |
| **Test** | "Is this a WHAT or a HOW?" | — |

**Why this matters:** The OPSEC work proved this distinction. The architect designed the right policy (collection agents must route through anti-detection) but wrong mechanism (custom tinyproxy when srt already exists). Every orch-go component should be classified:

- **Keep (policy):** Spawn gates, skill selection, hotspot routing, coaching decisions, completion verification, beads tracking
- **Delegate (enforcement):** Sandbox, permissions, hooks config, file protection — via adapter to platform-native features

### Lifecycle-Phase Separation

Orch-go and Claude Code enforce at different lifecycle phases. This is complementary, not duplicative:

| Phase | When | Who Enforces | What It Catches |
|-------|------|-------------|----------------|
| **Pre-spawn** | Before agent starts | Orch-go (spawn gates) | Wasted sessions — cheaper to block than to let agent start and hit runtime enforcement |
| **During execution** | While agent works | Claude Code (hooks, permissions, sandbox) | Runtime violations — wrong tools, forbidden domains, file access |
| **Below execution** | Always (OS level) | Orch-go (chflags) + Claude Code (srt/Seatbelt) | Configuration tampering — agent cannot modify its own constraints |
| **Post-execution** | After work done | Orch-go (completion gates, verification) | Quality violations — accretion, missing deliverables, failed tests |

**Key insight:** Pre-spawn and below-execution have NO Claude Code equivalent. Claude Code has no concept of "should this agent start?" or "make this file OS-immutable." These are orchestration-level concerns that orch-go adds to the trust stack.

---

## The Bypass Taxonomy

Six classes of bypass exist for any application-level enforcement. These were empirically mapped across CompetitorProxyEnforcement, scsdb read-only, and custom OPSEC:

| # | Bypass Class | What It Bypasses | Which Layers Catch It |
|---|-------------|-----------------|----------------------|
| 1 | **Shell commands** (curl, wget, psql) | Any application-layer middleware | L1 (sandbox), L2 (proxy vars for curl/wget), L4 (CLAUDE.md) |
| 2 | **Non-standard HTTP clients** (Net::HTTP, node fetch) | Opt-in middleware (Faraday, specific clients) | L1 (sandbox catches all), L2 (node ignores proxy vars) |
| 3 | **Subprocess spawning** (node -e, python3 -c) | Parent process middleware | L1 (sandbox inherits to children), L2 (env vars inherit) |
| 4 | **Direct credential use** (read .env, construct connections) | All transport-layer controls | L4 only (convention — agent must choose not to) |
| 5 | **Platform tool bypass** (WebFetch, WebSearch) | Sandbox (runs inside Claude CLI process) | L3 (permissions.deny rules), L4 (CLAUDE.md) |
| 6 | **Configuration tampering** (modify settings.json, hook scripts) | Hook-based enforcement | L1 (chflags uchg makes files immutable at OS level) |

**Coverage by layer:**

| Layer | Bypass Classes Caught | Classes Missed |
|-------|----------------------|----------------|
| L1: Infrastructure | 1, 2, 3, 6 | 4 (credentials), 5 (platform tools) |
| L2: Environment | 1 (partial), 3 (partial) | 2 (node ignores proxy), 4, 5, 6 |
| L3: Application | 5 | 1, 2, 3, 4, 6 |
| L4: Convention | All (probabilistic) | None caught reliably |

**The irreducible gap:** Credential exfiltration (class 4) cannot be structurally prevented without removing credential access entirely. An agent that can read `.env` files can extract and use credentials outside any sandbox. This is mitigated by: (a) restricting which credentials are available in the agent's environment, (b) using short-lived tokens instead of long-lived credentials, (c) convention-layer instructions as the last line of defense.

---

## The Allowlist Principle

**Allowlist > Blocklist for agent network access.** This is the single most impactful finding.

| Property | Blocklist (custom OPSEC) | Allowlist (srt/native sandbox) |
|----------|------------------------|-------------------------------|
| New unknown domain | **PASSES** (not in blocklist) | **BLOCKED** (not in allowlist) |
| Configuration burden | Low (add known-bad) | Higher (must list all needed) |
| Security posture | Open by default | Closed by default |
| Maintenance | Must discover and add each bad domain | Only add when agent needs a new domain |
| Failure mode | Silent (new domain passes unnoticed) | Loud (agent fails when domain missing) |

**Why this matters for agents specifically:** Agents discover new execution paths that humans don't anticipate. A blocklist assumes you can enumerate all bad paths in advance. An allowlist assumes you can enumerate all good paths — which is easier because legitimate agent needs (GitHub, npm, API providers) are finite and known.

**Evidence:** Custom OPSEC used a 10-pattern blocklist. srt with `allowedDomains` uses an allowlist. When tested, srt blocked all 7 transport-layer bypass classes. The allowlist also rejects overly broad patterns (`"*"` returns an error), preventing accidental weakening.

---

## Why This Fails

### 1. Convention Decay Under Task Pressure (Observed: Mar 6, 2026)

**What happens:** Agent follows task instructions ("test OshCut account creation API") over convention-layer constraints ("don't hit competitor APIs directly").

**Root cause:** CLAUDE.md instructions compete with task description in the agent's context. When task says "test signup flow" and CLAUDE.md says "don't hit OshCut," the task wins because it's more specific and more recent in context.

**Evidence:** Probe pw-w8r0 used curl to create 6 OshCut accounts from bare IP on the same day as detection. The agent was following its task correctly. No structural constraint prevented it. CLAUDE.md hard gate was added *after* the incident.

**Lesson:** Convention constraints work for well-intentioned agents with aligned tasks. They fail when task description conflicts with constraint, or when the agent discovers more efficient paths (curl vs Faraday).

### 2. Application-Layer Enforcement Has Domain Boundaries (Observed: Mar 17, 2026)

**What happens:** Enforcement applies only to the specific abstraction it wraps. Code outside that abstraction bypasses it entirely.

**Root cause:** CompetitorProxyEnforcement is a Faraday::Middleware. It intercepts Faraday HTTP connections. Net::HTTP, curl, node fetch, Python requests — all bypass it because they're not Faraday connections.

**Evidence:** PW's `collection_run_notifier.rb:3` already uses Net::HTTP (non-Faraday) in the same codebase. Node.js scraper subprocess has its own HTTP stack. The middleware protects its own stack, not the process.

**Lesson:** Application-layer enforcement protects the happy path. It does NOT protect arbitrary code execution — which is exactly what agents have.

### 3. Development-Only Defense-in-Depth (Observed: Mar 17, 2026)

**What happens:** Docker proxy-gateway provides network isolation in development, but production deployments (Render, Fly.io) have zero egress control.

**Root cause:** Docker Compose has `internal: true` networks. Render and Fly.io managed services don't offer per-service egress firewalls. The defense-in-depth layer exists only where it's least needed.

**Lesson:** Verify enforcement in the deployment environment that matters (production/agent-execution), not just the environment that's convenient (local development).

### 4. Passthrough Endpoints Bypass Server-Side Controls (Observed: Mar 17, 2026)

**What happens:** Chat endpoints pass frontend-constructed context directly to LLM, bypassing server-side anonymization.

**Root cause:** Batch analysis builds prompts server-side (server controls content). Chat endpoints receive pre-built context from frontend (server is a passthrough). Different data flow patterns require different enforcement points.

**Evidence:** Expedite chat sends partial emails and full company names to LLM. The anonymize.Mapper exists and works — it's just not in the code path for chat.

**Lesson:** Enforcement must be at the boundary where data enters the untrusted system. If the server is a passthrough, it must still validate/transform at the passthrough point.

---

## Testable Claims

| # | Claim | Status | Evidence | Falsification Condition |
|---|-------|--------|----------|------------------------|
| C1 | Convention-layer constraints (CLAUDE.md, skill guidance) are insufficient to prevent agent bypass when the agent has shell access | **Confirmed** | Probe pw-w8r0 bypassed convention via curl (Mar 6 incident). CompetitorProxyEnforcement bypassed by non-Faraday clients. | A convention-layer constraint that reliably prevents bypass in 50+ agent sessions with shell access |
| C2 | OS-level enforcement (sandbox-exec/srt, chflags uchg) cannot be bypassed by application-layer agent actions | **Confirmed** | 11 behavioral tests: srt blocks curl, node, python, wget. chflags survives Bash writes, rm, sed. | An agent bypassing srt allowedDomains or chflags uchg without root escalation |
| C3 | Allowlist-based domain restriction is strictly more secure than blocklist for agent network access | **Confirmed** | srt: new domains blocked by default (Test 8c). Custom blocklist: new domains pass. srt rejects wildcards. | A scenario where blocklist catches something allowlist misses (impossible by construction — allowlist is superset of blocklist's blocking power) |
| C4 | Pre-spawn validation prevents wasted agent sessions and has no equivalent in Claude Code's native features | **Confirmed** | Spawn gates block before token spend. Claude Code hooks fire during execution only (no pre-spawn concept). | Claude Code adds a pre-spawn hook event type, or a way to prevent session creation conditionally |
| C5 | Orch-go's enforcement value is in policy decisions (WHAT), not enforcement mechanisms (HOW) | **Confirmed** | OPSEC work: right policy (route through proxy), wrong mechanism (custom tinyproxy vs srt). Reinvented wheels audit: <5% OSS overlap in mechanisms, 100% novel in cross-cutting policy integration. | A case where orch-go's enforcement mechanism provides unique value that no platform-native feature can replicate (beyond the policy/decision layer) |
| C6 | Policy layer and enforcement layer operate at independent lifecycle phases and are not duplicative | **Confirmed** | 6 orch-go features audited: 4 add value at different lifecycle phases, 1 properly delegates to native features, 1 was reimplemented (sandbox, since deprecated). | An orch-go enforcement feature that operates at the same lifecycle phase AND mechanism as a Claude Code native feature (other than the already-deprecated sandbox) |
| C7 | Application-layer middleware (Faraday, session params) protects only its own abstraction stack, not arbitrary code execution | **Confirmed** | CompetitorProxyEnforcement: Faraday-only, 6 bypass classes. scsdb read-only: session param, bypassable via SET command or new connection. PW SlaveDbConnection: zero guard. | An application-layer middleware that catches all execution paths without OS-level support |
| C8 | Credential exfiltration (bypass class 4) is irreducible — no structural enforcement prevents an agent with file read access from using credentials | **Untested** | Inferred from bypass taxonomy. No test has attempted credential isolation. | A mechanism that prevents credential use without removing credential access from the agent's filesystem |
| C9 | Defense-in-depth layers that exist only in development provide false confidence about production security posture | **Confirmed** | Docker proxy-gateway (dev only) vs Render/Fly.io (zero egress control). Design doc claimed multi-layer system but L2 was dev-only. | A deployment platform (Render, Fly.io) that provides per-service egress firewall equivalent to Docker's `internal: true` |
| C10 | The trust hierarchy (L1-L4) correctly predicts which enforcement mechanisms will be bypassed first under agent pressure | **Partially confirmed** | L4 (convention) bypassed first (pw-w8r0). L3 (application) bypassed by non-standard tools. L1 (infrastructure) not bypassed in any test. | An L1 mechanism bypassed before an L4 mechanism in a real agent session, or an L4 mechanism that holds while L1 fails |
| C11 | Over-privileged credentials amplify bypass impact — the slave DB user has 12.6x more access than needed (88 tables vs 7 used) | **Confirmed** | Direct psql: `information_schema.table_privileges` shows SELECT, INSERT, UPDATE, DELETE, TRUNCATE, TRIGGER, REFERENCES on all 88 public tables. Toolshed uses 7 tables. Creating a restricted role requires `postgres` superuser (Dylan lacks `rolcreaterole`). | A restricted `toolshed_readonly` role is provisioned, reducing access to 7 tables |
| C12 | PW SlaveDbConnection has zero write prevention — naming convention ("slave"/"read-only") is the only guard | **Confirmed** | Code review: `slave_db_connection.rb:97-104` uses raw `PG.connect`, no `default_transaction_read_only`. `conn.exec(sql)` accepts arbitrary SQL. Contrast with Toolshed `client.go:54` which at least sets RuntimeParam. | `SET default_transaction_read_only = on` added after PG.connect in SlaveDbConnection |
| C13 | Server-side prompt construction is structurally safer than frontend passthrough for PII enforcement | **Confirmed** | Batch analysis (server builds prompts from raw data, controls anonymization) = 3/3 paths anonymized. Chat (server passes frontend context unchanged) = 0/1 anonymized for PII-containing endpoint. The data flow pattern determines whether enforcement is structurally possible. | A passthrough endpoint that reliably enforces anonymization without server-side content inspection |
| C14 | WebFetch/WebSearch tools bypass both sandbox-exec wrapping and proxy env vars — they run inside the Claude CLI Bun process | **Open** | Architecture analysis identifies the gap in both custom and native sandbox. Both approaches require `permissions.deny` rules (application-layer, not transport-layer). Not behaviorally verified end-to-end with a live agent session. | End-to-end test of a sandboxed agent attempting WebFetch to a non-allowed domain |
| C15 | Production deployments (Render, Fly.io) have zero egress control — defense-in-depth collapses to single application layer | **Confirmed** | `render.yaml`: no HTTP_PROXY, no egress config. `fly.toml`: no egress restrictions. Docker `internal: true` network only in `docker-compose.yml` (dev). Production has single-layer defense. | Render or Fly.io provides per-service egress firewall capability |
| C16 | "It's your call" is risk transfer, not delegation of trust — organizational risk and technical risk are independent dimensions | **Open** | Thread analysis: Jim's pattern (slave DB removal, personal Claude accounts, scraping decision deferral) consistently transfers concentrated personal liability while capturing diffuse organizational value. OPSEC harness reduces technical risk but not organizational exposure. | Written authorization or equity/profit-sharing that aligns risk and value on the same person |
| C17 | Infrastructure-backed answers are defensible; assurance-backed answers are not | **Confirmed** | OPSEC harness (sandbox, allowlist, OS-level enforcement) creates a structurally defensible posture. Compare: "I told the agent not to" (assurance) vs "the agent physically cannot reach that domain" (infrastructure). The OshCut incident proved assurance failed; srt evaluation proved infrastructure works. | An assurance-backed control that proves more reliable than an infrastructure-backed one over 50+ agent sessions |

---

## Constraints

### Why All Four Layers?

**Constraint:** Each layer addresses bypass classes the others miss. L1 can't catch platform tools (WebFetch). L3 can't catch shell commands. L4 catches nothing reliably but provides the only defense for credential exfiltration.

**This enables:** No single bypass class goes completely unaddressed.
**This constrains:** Cannot simplify to fewer layers without creating coverage gaps.

### Why Policy and Enforcement Must Be Separated?

**Constraint:** Enforcement mechanisms are platform-specific (Claude Code today, Codex tomorrow). Policy decisions (what to constrain) are domain-specific and must survive platform migration.

**This enables:** Platform-agnostic orchestration that works across agent platforms.
**This constrains:** Orch-go cannot build enforcement mechanisms that depend on a specific platform's internals. Must use adapter patterns (e.g., generate settings.json for Claude Code, generate equivalent config for future platforms).

### Why Allowlist Over Blocklist?

**Constraint:** Agents discover novel execution paths faster than operators can enumerate them. Blocklists require operators to anticipate every bad path. Allowlists only require operators to know what the agent legitimately needs.

**This enables:** Default-deny security posture where new paths are blocked automatically.
**This constrains:** Higher initial configuration burden — every legitimate domain must be explicitly allowed. Agent failures at new legitimate domains require allowlist updates.

---

## Evolution

**Mar 17, 2026:** Bypass surface investigation mapped 6 bypass classes across CompetitorProxyEnforcement and scsdb read-only. Found both are convention-enforced, not structurally enforced. PII investigation found anonymization gap in chat endpoints (passthrough pattern).

**Mar 18, 2026:** Agent execution environment investigation traced OshCut detection to probe pw-w8r0. Confirmed 6 bypass classes for tmux-based agents. Established defense-in-depth ranking: CLAUDE.md (weakest) → HTTP_PROXY (partial) → Docker containers (structural).

**Mar 19, 2026:** Three investigations converged:
1. Native sandbox evaluation confirmed srt replaces custom OPSEC with superior security model (allowlist).
2. Orch-go audit against Claude Code confirmed lifecycle-phase separation (no reimplementation).
3. Reinvented wheels audit confirmed <5% OSS overlap; value is in policy integration.

Three threads synthesized the pattern:
- "Trust is structural, not verbal" — the core insight
- "Policy vs enforcement layer" — the architectural distinction
- "Risk-value asymmetry" — infrastructure-backed answers vs assurance-backed answers

**Mar 19, 2026:** Model created, synthesizing all sources into testable claims framework.

---

## References

**Investigations (primary evidence):**
- `scs-special-projects/.kb/investigations/2026-03-19-inv-audit-orch-go-against-claude.md`
- `scs-special-projects/.kb/investigations/2026-03-19-inv-evaluate-claude-code-native-sandbox.md`
- `scs-special-projects/.kb/investigations/2026-03-19-inv-audit-orch-go-reinvented-wheels.md`
- `scs-special-projects/.kb/investigations/2026-03-19-architect-layer1-local-opsec-enforcement.md`
- `scs-special-projects/.kb/investigations/2026-03-18-inv-investigate-agent-execution-environment-bypass.md`
- `scs-special-projects/.kb/investigations/2026-03-18-redteam-toolshed-security-assessment.md`
- `scs-special-projects/.kb/investigations/2026-03-17-inv-investigate-bypass-surface-toolshed-price.md`
- `scs-special-projects/.kb/investigations/2026-03-17-inv-investigate-toolshed-pii-exposure-ai.md`
- `scs-special-projects/.kb/investigations/2026-03-17-inv-investigate-toolshed-slave-db-credential.md`

**Threads (pattern synthesis):**
- `scs-special-projects/.kb/threads/2026-03-19-environment-control-as-agent-trust.md`
- `scs-special-projects/.kb/threads/2026-03-19-policy-layer-vs-enforcement-layer.md`
- `scs-special-projects/.kb/threads/2026-03-19-risk-value-asymmetry-sp-role.md`

**Related models in orch-go:**
- `.kb/models/architectural-enforcement/model.md` — Code quality gates (complementary enforcement domain)
- `.kb/models/claude-code-agent-configuration/model.md` — Claude Code configuration layer
- `.kb/models/harness-engineering/model.md` — Hard vs soft harness taxonomy

**Primary Evidence (Verify These):**
- `pkg/control/control.go:121-134` — chflags uchg OS-level immutability (L1 enforcement)
- `pkg/orch/spawn_preflight.go:11` — Pre-spawn gate pipeline (no Claude Code equivalent)
- `pkg/spawn/claude.go:70-142` — Launch command builder (policy→enforcement adapter)
- `pkg/spawn/gates/*.go` — Spawn gate implementations (policy decisions)
- `pkg/hook/schema.go` — Hook output validation (fills Claude Code CLI gap)
- `toolshed/pkg/scsdb/client.go:54` — RuntimeParams read-only guard (L2 enforcement, bypassable)
- `price-watch/backend/app/services/slave_db_connection.rb:97-104` — No read-only enforcement (zero guard)
- `price-watch/backend/app/middleware/competitor_proxy_enforcement.rb:24` — Faraday-only middleware (L3 enforcement)
- `toolshed/internal/expedite/chat_handler.go:75` — PII passthrough gap (no server-side anonymization)

### Merged Probes

| Probe | Date | Verdict | Key Finding |
|-------|------|---------|-------------|
| `probes/2026-03-19-probe-trust-hierarchy-codebase-validation.md` | 2026-03-19 | Confirms | All 4 trust layers have concrete orch-go implementations. Policy/enforcement separation is clean: spawn gates (policy), BuildClaudeLaunchCommand (adapter), Claude Code flags (enforcement). L2 converging into L3 via native sandbox settings. |
