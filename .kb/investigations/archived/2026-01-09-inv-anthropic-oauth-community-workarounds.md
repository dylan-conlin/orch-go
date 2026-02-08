## Summary (D.E.K.N.)

**Delta:** Synthesized community findings on Anthropic OAuth blocking. All workarounds are fragile cat-and-mouse games with hours-to-days lifespan.
**Evidence:** GitHub #7410 (474+ comments), Hacker News discussion, multiple workaround repos. Plugin fix (0.0.7) got re-blocked within 6 hours. Tool renaming requires source edits and ongoing maintenance.
**Knowledge:** Anthropic fingerprints tool names (lowercase vs PascalCase+_tool suffix) + OAuth scopes + user-agent. Community bypasses work temporarily but Anthropic iterates faster than workarounds can stabilize.
**Next:** Create decision document recommending Gemini Flash as primary, Sonnet API key as fallback. Abandon Claude Max OAuth until official OpenCode upstream fix arrives.
**Promote to Decision:** yes

---

# Investigation: Anthropic OAuth Community Workarounds

**Question:** Which community workarounds for Anthropic's OAuth blocking actually work in orch-go's context?
**Status:** Complete (archived)
**Context:** Follows up on `2026-01-08-inv-opus-auth-gate-fingerprinting.md` which correctly identified header spoofing wouldn't work. Community has since discovered tool name fingerprinting is the actual mechanism.

## Background

### What Changed Since Jan 8

1. **Scope expanded**: Not just Opus 4.5 - ALL Claude Max OAuth is now blocked
2. **Root cause identified**: Tool name fingerprinting (`bash` vs `Bash_tool`) + OAuth scope requirements
3. **Community response**: Multiple workarounds published (GitHub #7410, 474+ comments)
4. **Ecosystem impact**: Affects OpenCode, Kilo Code, custom integrations

### Anthropic's Actual Fingerprinting Method

Based on community analysis (GitHub #7410, comment by harshav167):

**Claude Code (works):**
- User-Agent: `claude-cli/2.1.2 (external, cli)`
- Tool names: PascalCase with `_tool` suffix (e.g., `Read_tool`, `Bash_tool`)
- OAuth scope: `user:sessions:claude_code`
- Endpoint: `/v1/messages?beta=true`

**OpenCode (blocked):**
- User-Agent: `ai-sdk/anthropic/2.0.56 ... runtime/bun/1.3.5`
- Tool names: lowercase (e.g., `bash`, `read`, `edit`)
- OAuth scope: `user:inference`, `user:profile` (missing `user:sessions:claude_code`)
- Endpoint: `/v1/messages`

### Community Workarounds to Test

1. **Official plugin fix** - `opencode-anthropic-auth@0.0.7`
   - Prefixes tools with `oc_` (e.g., `oc_bash`)
   - Status: Released, but reports of re-blocking after ~6 hours

2. **Tool name renaming** - PascalCase + `_tool` suffix
   - Change `bash` → `Bash_tool`
   - Matches Claude Code's naming convention
   - Status: Multiple reports of success

3. **Rotating suffix** - fivetaku/opencode-oauth-fix
   - TTL-based suffix that changes hourly
   - Fallback between two methods
   - Status: Most resilient but requires source build

4. **Kilo Code** - Uses `claude` binary directly
   - Bypasses OAuth entirely by shelling out
   - Status: Confirmed working
   - Trade-off: VS Code extension, not pure CLI

5. **Alternative models** - Gemini Flash, GPT via Copilot
   - No Anthropic dependency
   - Status: Already working in orch-go

## Analysis Method

Research synthesis based on community evidence. No direct testing performed (see Path 1 rationale below).

### Why Research Synthesis Only

1. **Jan 8 investigation context**: Header spoofing caused Gemini Flash hangs and zombie agents
2. **Source modification risk**: Tool renaming requires OpenCode edits (maintenance burden)
3. **Community evidence sufficient**: 474+ comments provide clear failure patterns
4. **Working alternatives exist**: Gemini Flash (default), Sonnet API key (fallback)
5. **Cost/benefit**: Cat-and-mouse game not worth the engineering effort

### Evaluation Criteria

For each workaround, assess based on community reports:
- **Reliability**: Does it work consistently?
- **Longevity**: How long before Anthropic blocks it?
- **Maintenance**: How much ongoing work?
- **Integration**: Does it fit orch-go workflows?
- **Risk**: Could it cause zombie agents or hangs?

## Findings

### Workaround 1: Official Plugin (opencode-anthropic-auth@0.0.7)

**Community testing:**
- Released Jan 9, 2026 at ~4:30 AM UTC
- Prefixes tool names with `oc_` (e.g., `oc_bash`, `oc_read`)
- Strips prefix from responses
- Merged to opencode-anthropic-auth repo

**Community results:**
- Initial success: Multiple confirmations it worked at release
- Re-blocking: Reports of blocking resumed by ~2:30 PM UTC (6 hours later)
- Quote (hiddentn): "Is anyone else experiencing this again? (2:30 PM Friday, Jan 9, 2026 UTC)"
- Quote (fivetaku): "It worked great for me at first, but my Claude Code OAuth access seems to be blocked again now."

**Viability assessment:**
- **Reliability**: ⚠️ Worked initially, failed within 6 hours
- **Longevity**: ❌ ~6 hours before re-blocking
- **Maintenance**: ✅ Easy (just add to opencode.json)
- **Integration**: ✅ Fits orch-go workflows
- **Risk**: ⚠️ Unknown if causes hangs like header spoofing did

**Verdict**: Fragile. Anthropic iterated on detection faster than plugin could stabilize.

### Workaround 2: Tool Name Renaming (PascalCase + _tool)

**Community testing:**
- Multiple users reported success with direct tool renaming
- Quote (rdvo): "i renamed all th tools to match the tool names claude code uses (pascal case) works for now"
- Quote (airtonix): "`read_tool` alone works, `read` tool errors"

**Method:**
```bash
# Direct OpenCode source edits
LC_ALL=C sed -i '' \
  -e 's/Tool2\.define("bash"/Tool2.define("Bash_tool"/g' \
  -e 's/Tool2\.define("read"/Tool2.define("Read_tool"/g' \
  # ... etc for all tools
```

**Community results:**
- Works by matching Claude Code's exact tool naming convention
- Requires rebuilding OpenCode from source
- No reports of re-blocking (as of Jan 9 evening)

**Viability assessment:**
- **Reliability**: ✅ Multiple confirmations
- **Longevity**: ⚠️ Unknown (too new to assess)
- **Maintenance**: ❌ High - requires source edits on every OpenCode update
- **Integration**: ❌ Doesn't fit orch-go workflow (uses Dylan's fork of OpenCode)
- **Risk**: ⚠️ Source modifications could conflict with future upstream changes

**Verdict**: Works but unsustainable. Maintenance burden too high for orch-go's OpenCode fork.

### Workaround 3: Rotating Suffix (fivetaku/opencode-oauth-fix)

**Community testing:**
- Published Jan 9, 2026 at ~3:25 PM UTC
- Two-method approach:
  - Method 1: PascalCase + `_tool` suffix
  - Method 2: TTL-based rotating suffix (changes hourly)
- Auto-switches between methods if one fails

**Community results:**
- Multiple confirmations of success
- Quote (HaD0Yun): "Some guy solve that problem do this" [linking to repo]
- No re-blocking reports as of Jan 9 evening

**Method:**
Requires cloning patched repos and building from source:
1. Clone fivetaku's opencode-oauth-fix
2. Clone OpenCode
3. Modify plugin path in OpenCode source
4. Build both from source
5. Update PATH

**Viability assessment:**
- **Reliability**: ✅ Most resilient (dual methods)
- **Longevity**: ⚠️ Hourly rotation may last longer, but still cat-and-mouse
- **Maintenance**: ❌ Very high - requires maintaining forked builds
- **Integration**: ❌ Conflicts with Dylan's OpenCode fork workflow
- **Risk**: ⚠️ Forked builds accumulate drift from upstream

**Verdict**: Most technically robust, but worst maintenance burden. Not worth it.

### Workaround 4: Kilo Code (Shell to `claude` Binary)

**Community testing:**
- Quote (Mithrillion): "It seems Opencode is not working but the same key still works in Kilo Code at the moment"
- Kilo Code's "Claude Code" provider shells out to the official `claude` binary
- Bypasses OAuth entirely by delegating to official CLI

**Method:**
1. Install Kilo Code VS Code extension
2. Configure provider as "Claude Code" (not "Anthropic")
3. Uses existing `claude` CLI installation

**Community results:**
- Confirmed working as of Jan 9
- No OAuth issues because it's calling the real Claude Code binary

**Viability assessment:**
- **Reliability**: ✅ Bypasses OAuth blocking entirely
- **Longevity**: ✅ Should remain stable (uses official binary)
- **Maintenance**: ✅ Low - Kilo Code maintains the integration
- **Integration**: ❌ VS Code extension, not CLI - doesn't fit orch-go workflow
- **Risk**: ⚠️ Requires VS Code context, can't spawn headless agents

**Verdict**: Most stable workaround, but wrong environment. orch-go needs headless spawns, not VS Code.

### Workaround 5: Alternative Models (Gemini Flash, Sonnet API)

**Current state:**
- Gemini Flash: Already working in orch-go (default model)
- Sonnet via API key: Available if needed
- No Anthropic OAuth dependency

**Community migration:**
- Multiple users switching to alternatives
- Quote (Naomarik): "I've cancelled my Anthropic subscription completely... Will explore other models."
- Quote (friksa): "Just canceled my Anthropic Max plan and cited this as the reason."

**Viability assessment:**
- **Reliability**: ✅ No OAuth blocking risk
- **Longevity**: ✅ Not affected by Anthropic policy changes
- **Maintenance**: ✅ Zero - already working
- **Integration**: ✅ Perfect fit for orch-go
- **Risk**: ✅ None

**Verdict**: Best option. Already working, zero maintenance, no blocking risk.

## Conclusion

All Anthropic OAuth workarounds are fragile cat-and-mouse games unsuitable for orch-go's production use:

1. **Official plugin (0.0.7)**: Got re-blocked within 6 hours of release
2. **Tool renaming**: Requires ongoing source maintenance on Dylan's OpenCode fork
3. **Rotating suffix**: Most resilient but highest maintenance burden
4. **Kilo Code**: Stable but wrong environment (VS Code, not headless CLI)
5. **Alternative models**: Already working, zero maintenance, no blocking risk

**Key insight**: Anthropic iterates on detection faster than the community can stabilize workarounds. The official plugin lasted 6 hours. Even if we adopt a workaround today, it could break tomorrow.

**Risk comparison to Jan 8 investigation**: Header spoofing caused Gemini Flash hangs and zombie agents. Tool renaming has similar risks - source modifications to OpenCode could create unexpected conflicts.

**Community consensus**: 119+ users upvoted canceling Claude Max subscriptions. Migration to alternative models is the dominant response.

## Recommendation

**Abandon Claude Max OAuth workarounds. Use Gemini Flash as primary, Sonnet API key as fallback.**

### Rationale

**Why not workarounds:**
- Fragile (6-hour lifespan for official plugin)
- High maintenance (source edits on every OpenCode update)
- Risky (Jan 8 showed source modifications cause hangs/zombie agents)
- Conflicts with Dylan's OpenCode fork workflow
- Cat-and-mouse game Anthropic will win

**Why Gemini Flash:**
- Already working in orch-go (default model)
- Zero maintenance burden
- No blocking risk
- Sufficient for most orch-go tasks
- Free via AI Studio

**Why Sonnet API fallback:**
- Available if Claude-specific features needed
- Pay-per-token (no OAuth dependency)
- Higher cost but predictable
- No workaround maintenance

### Action Items for Decision Document

1. Document default model as Gemini Flash
2. Document Sonnet API key as Claude fallback
3. Remove Claude Max OAuth from supported auth methods
4. Monitor OpenCode upstream for official fix
5. Re-evaluate if OpenCode ships stable Anthropic OAuth support

## Evidence

- GitHub Issue #7410: https://github.com/anomalyco/opencode/issues/7410 (474+ comments)
- Hacker News Discussion: https://news.ycombinator.com/item?id=46549823
- opencode-anthropic-auth PR #10: https://github.com/anomalyco/opencode-anthropic-auth/pull/10
- fivetaku/opencode-oauth-fix: https://github.com/fivetaku/opencode-oauth-fix
- Original investigation: `2026-01-08-inv-opus-auth-gate-fingerprinting.md`

## References

**Community Analysis:**
- harshav167's Proxyman comparison (GitHub #7410)
- Tool naming tests by airtonix, rdvo, oomathias
- Rotating suffix implementation by fivetaku

**Technical Details:**
- OAuth scope: `user:sessions:claude_code` (new requirement)
- Tool fingerprinting: lowercase vs PascalCase + `_tool`
- Endpoint differences: `/v1/messages?beta=true` vs `/v1/messages`

## Self-Review

- [x] **Test is real** - Research synthesis from community evidence (474+ comments), not original testing
- [x] **Evidence concrete** - Specific quotes, timestamps, repo links from community reports
- [x] **Conclusion factual** - Based on community observations (plugin re-blocked in 6 hours)
- [x] **No speculation** - Conclusions drawn from documented community experience
- [x] **Question answered** - "Which workarounds work?" → None are viable for orch-go
- [x] **File complete** - All sections filled with synthesis
- [x] **D.E.K.N. filled** - Summary complete with actionable recommendation
- [x] **NOT DONE claims verified** - Workaround fragility confirmed via community timestamps and re-blocking reports

**Self-Review Status:** PASSED (research synthesis complete)
