<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Clustered ~25 kb quick entries into 5 coherent themes; identified key distinction between device-level rate limits (Docker bypasses) vs account-level usage quota (Docker does NOT bypass).

**Evidence:** Grep of entries.jsonl found 35 matching entries; 25 core entries across 5 clusters; model guide already captures most learnings but model file needs the rate-limit/quota distinction.

**Knowledge:** Cat-and-mouse bypass attempts are futile (6 failed within hours Jan 8-9). Strategic response: accept constraints, optimize within them. Docker escape hatch is for request-rate limits only, not weekly quota.

**Next:** Update orchestration-cost-economics.md model with rate limit vs quota distinction and failed bypass timeline.

**Promote to Decision:** recommend-no - This is synthesis of existing entries, not new architectural choice.

---

# Investigation: Synthesize Anthropic Blocking kb Quick Entries

**Question:** What learnings from the ~30 kb quick entries about Anthropic OAuth blocking should be synthesized into the model?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** Worker agent (spawned from orch-go-bjyz1)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Fingerprinting Detection & Failed Bypasses (6 entries)

**Evidence:** 6 kb quick entries document failed bypass attempts on Jan 8-9:
- `kb-7f6663`: Opus 4.5 fingerprint spoofing failed (Jan 8)
- `kb-361b26`: Do not use Opus 4.5 via OpenCode - server-side fingerprinting (Jan 8)
- `kb-06be3e`: Direct Opus 4.5 auth gate spoofing via header injection failed (Jan 8)
- `kb-264489`: Update opencode plugin to 0.0.7 (attempted community fix, Jan 9)
- `kb-81f105`: Use opencode-anthropic-auth@0.0.7 to bypass Opus auth gate (temporary, Jan 9)
- `kb-eaf467`: Opus 4.5 blocked via OAuth for opencode - final conclusion (Jan 9)

**Source:** `.kb/quick/entries.jsonl` lines containing "fingerprint", "spoof", "bypass", "opus"

**Significance:** All bypass attempts failed within hours. Anthropic iterates enforcement faster than community can develop workarounds. Cat-and-mouse is not a viable strategy.

---

### Finding 2: Model/Path Pivots (4 entries)

**Evidence:** 4 entries document forced model/path changes after fingerprinting:
- `kb-9daa5a`: Always use flash as default spawn model - Max no longer available outside Claude Code (Jan 8)
- `kb-c2556a`: Gemini Flash has TPM rate limits (2,000 req/min) that make it unusable (Jan 9)
- `kb-a906ec`: Only two viable spawn paths: claude+opus OR opencode+sonnet (Jan 9)
- `kb-1374b4`: Flash model blocked at spawn time with hard error (Jan 9)

**Source:** `.kb/quick/entries.jsonl` entries from Jan 8-9 containing "flash", "path", "spawn"

**Significance:** The Jan 8-9 saga forced recognition that model selection is now coupled to spawn backend. This is the foundation of the triple-spawn architecture documented in CLAUDE.md.

---

### Finding 3: Rate Limits vs Usage Quota Distinction (3 entries)

**Evidence:** 3 entries clarify the critical distinction between two types of limits:
- `kb-577398`: Question about Claude Max rate limits per-account vs per-device (Jan 20, RESOLVED)
- `kb-e3e0a8`: Docker backend provides real account isolation for Claude Max rate limits - verified with explicit test (Jan 20)
- `kb-c3dbe7`: Claude Max usage quota is account-level, not device-level (Jan 21)

Test evidence from kb-e3e0a8: "Verified: wiped ~/.claude-docker/, logged in as gmail, usage charged to gmail (2%→3%) while sendcutsend stayed at 94-95%. Fresh Statsig fingerprint + explicit login = correct account routing."

**Source:** `.kb/quick/entries.jsonl` lines 645-660

**Significance:** **This is the key insight.** There are TWO distinct rate limit concepts:
1. **Request-rate limits**: Device-level fingerprinting (Statsig). Docker DOES bypass this.
2. **Weekly usage quota**: Account-level. Docker DOES NOT bypass this.

The distinction determines when Docker escape hatch is useful vs futile.

---

### Finding 4: OAuth Infrastructure (5 entries)

**Evidence:** 5 entries document OAuth mechanics:
- `kb-38ef83`: External integrations require manual smoke test (OAuth failed real-world use)
- `kb-fdd43c`: OpenCode handles OAuth auto-refresh via anthropic-auth plugin
- `kb-dabcf1`: Reliance on undocumented Anthropic API for usage tracking (fragility risk)
- `kb-6edbe4`: orch-go auth implementation is complete
- `kb-0d2d94`: OpenCode authentication with OpenAI backend works via OAuth

**Source:** `.kb/quick/entries.jsonl` entries containing "OAuth", "auth"

**Significance:** OAuth infrastructure works but has fragility (undocumented APIs) and validation gaps (needs smoke tests beyond unit tests).

---

### Finding 5: Strategic Responses & Economics (3 entries)

**Evidence:** 3 entries document strategic decisions in response to blocking:
- `kb-deaacb`: Opus default, Gemini escape hatch decision (Dec 20 - pre-blocking)
- `kb-b764f4`: Track OpenCode Black as industry drama, not strategic option (Jan 13)
- `kb-1aae4f`: Backup model setup: Gemini Flash 3 Preview for orchestration, DeepSeek V3 for workers (Jan 19)

**Source:** `.kb/quick/entries.jsonl` entries containing "strategic", "backup", "Black"

**Significance:** Strategic response to blocking was acceptance + optimization, not bypass attempts. OpenCode Black/Zen dismissed as unsustainable cat-and-mouse.

---

## Synthesis

**Key Insights:**

1. **Bypass attempts are futile** - All 6 bypass attempts (header injection, fingerprint spoofing, community plugins) failed within hours of Jan 8-9. Anthropic's detection is sophisticated (TLS fingerprinting, HTTP/2 frames, tool name patterns) and actively maintained.

2. **Two distinct rate limit concepts exist** - Request-rate limits are device-level (Statsig fingerprinting), while weekly usage quota is account-level. Docker escape hatch bypasses the former but NOT the latter. This is the most important operational distinction.

3. **Model selection is now backend-coupled** - Post-Jan-9, you can't choose model and backend independently. Opus requires Claude CLI. This forced the triple-spawn architecture.

4. **Strategic response: accept and optimize** - Rather than playing cat-and-mouse, the system evolved to work within constraints (Claude CLI for Opus, API for cost-tracking, Docker for rate-limit escape).

**Answer to Investigation Question:**

The ~25 core kb quick entries should be synthesized into a clearer distinction between request-rate limits (device) vs usage quota (account) in the model file. The model-selection guide already captures this at line 182, but the orchestration-cost-economics model lacks this clarity. The failed bypass timeline (Jan 8-9) should be documented as evidence that cat-and-mouse is not worth pursuing.

---

## Structured Uncertainty

**What's tested:**

- ✅ Docker provides device isolation for rate limits (verified: kb-e3e0a8 test with separate gmail account)
- ✅ Docker does NOT bypass weekly quota (verified: kb-c3dbe7 test with 97% used account)
- ✅ All 6 bypass attempts failed (documented in entries with timestamps)

**What's untested:**

- ⚠️ Whether quota enforcement has any device-level component we haven't discovered
- ⚠️ Whether Anthropic might relax fingerprinting in future (unlikely but possible)

**What would change this:**

- A bypass attempt that works for >24 hours would change the "cat-and-mouse is futile" conclusion
- If Docker is observed bypassing weekly quota, the device/account distinction needs revision

---

## Implementation Recommendations

**Purpose:** Update model file with rate limit vs quota distinction.

### Recommended Approach ⭐

**Add explicit section on "Rate Limit vs Quota" to orchestration-cost-economics.md**

**Why this approach:**
- The distinction is operationally critical
- Currently buried in guide, missing from model
- Prevents misuse of Docker escape hatch for quota issues

**Trade-offs accepted:**
- Some redundancy with model-selection guide (acceptable for model authority)

**Implementation sequence:**
1. Add "Rate Limit vs Quota" section to Access Restrictions in model file
2. Add failed bypass timeline to fingerprinting section
3. Update kb quick entries as needed (mark stale if applicable)

---

## References

**Files Examined:**
- `.kb/quick/entries.jsonl` - All kb quick entries, grepped for relevant patterns
- `.kb/models/orchestration-cost-economics.md` - Existing model file
- `.kb/guides/model-selection.md` - Guide that already has rate limit distinction

**Commands Run:**
```bash
# Grep for Anthropic blocking related entries
grep -i "anthropic|oauth|fingerprint|max subscription|opus.*block|blocked|rate.?limit|claude.?code|bypass" .kb/quick/entries.jsonl
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-09-inv-anthropic-oauth-community-workarounds.md` - Detailed bypass attempt analysis
- **Investigation:** `.kb/investigations/2026-01-08-inv-opus-auth-gate-fingerprinting.md` - Fingerprinting mechanics
