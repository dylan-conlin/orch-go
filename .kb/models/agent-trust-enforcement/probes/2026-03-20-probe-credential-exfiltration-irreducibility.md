# Probe: Credential Exfiltration Irreducibility (ATE-08)

**Claim:** Credential exfiltration (bypass class 4) is irreducible — no structural enforcement prevents an agent with file read access from using credentials
**Falsification condition:** A mechanism that prevents credential use without removing credential access from the agent's filesystem
**Date:** 2026-03-20
**Status:** Partially confirmed — claim too strong as stated, holds for a narrower reading

---

## Analysis

The claim conflates two distinct threats that require separate analysis:

1. **Direct credential use** — agent reads credentials and connects to the target service
2. **Credential exfiltration** — agent reads credentials and sends them to an external destination

### Mechanism 1: Network Isolation (srt allowedDomains) — Partially Falsifies

srt's `allowedDomains` operates at L1 (OS-level sandbox). If a service's host is NOT in the allowlist, the agent:
- CAN read credentials from `.env`, `auth.json`, keychain, etc. (file access preserved)
- CANNOT connect to the service those credentials authenticate against (network blocked at OS level)

**This IS a structural mechanism that prevents credential use without removing file access.** The credentials are readable but useless because the network path is severed.

**Evidence from codebase:**
- `pkg/spawn/claude.go:146` — All Claude CLI agents spawned with `--dangerously-skip-permissions` (full filesystem access)
- Model C2 (ATE-02) confirmed: srt blocks curl, node, python, wget — 11 behavioral tests
- No filesystem restrictions exist on credential files — agents CAN read `.env`, `~/.orch/accounts.yaml`, `~/.local/share/opencode/auth.json`

**Limitation:** This only works for services NOT in the allowlist. For services the agent legitimately needs access to (which ARE in allowedDomains), network isolation provides no protection.

### Mechanism 2: Credential Scoping — Mitigates, Doesn't Prevent

Giving the agent scoped credentials (read-only DB user with 7 tables instead of full admin with 88 tables) doesn't prevent USE but bounds DAMAGE. This is mitigation, not prevention, so it does not falsify the claim.

**Evidence:** C11 (ATE-11 analog) confirms the slave DB user has 12.6x over-privilege. Scoping would reduce blast radius without removing access.

### Mechanism 3: Short-lived Tokens — Reduces Window, Doesn't Prevent

Vault-style tokens that expire quickly reduce the exploitation window but don't prevent use during the valid window. The agent operates within a session where the token is active.

### Why the Claim Holds for Allowed Services

For any service in `allowedDomains` that the agent legitimately needs:
- Agent has network access to the service (by necessity)
- Agent has filesystem access to read credentials (by necessity for `--dangerously-skip-permissions`)
- No mechanism prevents the agent from using one against the other

**Covert channel problem for exfiltration:** Even with strict allowlist, the agent can embed credentials in:
- Git commit messages (git is always allowed)
- API calls to allowed services (e.g., embed in a GitHub issue body)
- File writes that get synced externally

This means exfiltration to an attacker is possible through any allowed communication channel, making true exfiltration prevention irreducible absent removing ALL external communication.

### What Would Actually Falsify the Claim

A mechanism like:
- **Hardware Security Module (HSM)** — credential stored in hardware, agent gets a session handle but never sees the raw credential. Agent can USE the service but cannot EXTRACT the credential.
- **Credential proxy** — agent authenticates through a proxy that injects credentials server-side. Agent never has the raw credential on its filesystem. (This IS "removing credential access from the filesystem," so it doesn't technically falsify.)
- **Capability-based tokens** — agent gets a token scoped to specific operations, not the underlying credential. The token is useless outside the agent's session context (bound to IP, short-lived, operation-scoped).

These exist in theory. None are implemented in this codebase or in Claude Code's sandbox.

---

## Verdict

**REFINE** — The claim as stated is too strong. It should be narrowed:

**Current:** "no structural enforcement prevents an agent with file read access from using credentials"

**Refined:** "no structural enforcement prevents an agent from using credentials against services it has network access to; network isolation (srt) can prevent use against non-allowed services without removing file access, but exfiltration via allowed channels remains irreducible"

The key distinction:
| Scenario | Preventable? | Mechanism |
|----------|-------------|-----------|
| Use credentials against non-allowed service | **Yes** | srt allowedDomains (L1) |
| Use credentials against allowed service | **No** | Convention only (L4) |
| Exfiltrate credentials via allowed channel | **No** | Convention only (L4) |
| Exfiltrate credentials via non-allowed channel | **Yes** | srt allowedDomains (L1) |

**Net assessment:** 2 of 4 scenarios are structurally preventable. The claim's "irreducible" framing is too absolute — the reality is that credential risk is PARTIALLY reducible via network isolation, with an irreducible residual for services the agent must access.

---

## Recommendations

1. **Update ATE-08 claim text** to the refined version above
2. **Update model.md "irreducible gap" section** (line 105) to acknowledge partial reducibility via network isolation
3. **Status: partially confirmed** — the irreducible core holds (credentials usable against allowed services), but the absolute framing is falsified by srt
