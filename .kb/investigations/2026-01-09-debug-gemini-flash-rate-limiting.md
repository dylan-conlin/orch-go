# Investigation: Gemini Flash Rate Limiting

**Date:** 2026-01-09
**Type:** Debug
**Status:** In Progress
**Agent:** Orchestrator (Dylan)

## Problem Statement

When spawning agents with `--model flash` (Gemini Flash), seeing error in OpenCode:
```
"gemini is way too hot right now (click to expand) [retrying in 3s attempt #3]"
```

This occurs even with a **single agent spawn**, not multiple concurrent agents.

## Root Cause

### Rate Limit Discovery

Via `gcloud alpha services quota list`, found the bottleneck:

**Current limits (Paid Tier 2, gemini-3-flash):**
- **Requests per minute:** 2,000 ← **bottleneck**
- Input tokens per minute: 3,000,000 (not hitting this)
- Requests per day: 10,000 (sufficient)

### Why Single Agent Hits Limits

Tool-heavy agents (investigation, systematic-debugging) make rapid API calls:
- Each tool use (Read, Grep, Bash, etc.) = one API request
- Agent making 35+ tool calls/second = 2,000+ requests/minute
- Rate limit triggers even with one active agent

## Investigation Steps Taken

### 1. Initial Diagnosis
- User reported rate limiting with Gemini Flash
- Asked about applying for higher limits

### 2. CLI Setup
```bash
# Verified gcloud CLI installed
which gcloud  # /opt/homebrew/bin/gcloud

# Updated to latest version
gcloud components update  # 534.0.0 → 551.0.0

# Installed alpha commands
gcloud components install alpha --quiet

# Set correct project
gcloud config set project scs-dev-456116
gcloud auth application-default set-quota-project scs-dev-456116
```

### 3. Quota Discovery
```bash
# Listed all quotas for Generative Language API
gcloud alpha services quota list \
  --service=generativelanguage.googleapis.com \
  --consumer=projects/scs-dev-456116

# Found gemini-3-flash specific limits
gcloud alpha services quota list \
  --service=generativelanguage.googleapis.com \
  --consumer=projects/scs-dev-456116 \
  --format=json | jq '[.[] | select(.consumerQuotaLimits[0].quotaBuckets[]? | .dimensions.model == "gemini-3-flash")]'
```

**Key finding:** Paid Tier 2 limit is **2,000 requests/minute** per project per model.

### 4. Web UI Attempt
- Navigated to Google Cloud Console → IAM & Admin → Quotas & System Limits
- Filtered for `generate_content_paid_tier_2`
- Found quota row: "Request limit per model per minute for a project in the paid tier 2"
- Current value: 2,000
- Attempted to increase to 20,000
- **Result:** Interface restricts to max 2,000 (self-service limit reached)

### 5. Tier System Discovery
Google uses tiered quotas based on billing commitment:
- **Tier 2 (current):** 2,000 req/min
- **Tier 3 (target):** 20,000 req/min (10x improvement)

Tier upgrade requires either:
- Increased spending commitment (automatic)
- Google Cloud sales contact
- Support case requesting tier upgrade

## Solutions Identified

### Immediate Workarounds

1. **Use Sonnet/Opus (Anthropic API):**
   ```bash
   orch spawn --model sonnet investigation "task"
   ```
   - Different rate limits (more generous)
   - Requires Claude Max subscription

2. **Switch accounts:**
   ```bash
   orch account switch work
   ```
   - Separate quota pool
   - Requires second paid Google Cloud account

3. **Let retry logic handle it:**
   - OpenCode auto-retries with exponential backoff
   - Agent completes successfully, just slower

### Long-term Fix

**Upgrade to Paid Tier 3:**

Option A: Check if self-service upgrade available
1. Go to Quotas & System Limits page
2. Filter for `paid_tier_3`
3. If "Adjustable: Yes", request increase directly

Option B: Contact Google Support
1. Navigate to https://console.cloud.google.com/support
2. Create support case
3. Request: "Upgrade to Paid Tier 3 for Generative Language API"
4. Mention: Currently Tier 2 (2,000/min), need Tier 3 (20,000/min) for concurrent AI orchestration

## Current Status

**Blocked on:**
- User needs to check Tier 3 availability in web UI (filter for `paid_tier_3`)
- Determine if self-service upgrade possible or requires support case

**Workarounds in use:**
- Can use `--model sonnet` for urgent spawns
- Can tolerate retry delays for non-urgent work

## Recommendations

1. **Immediate:** Use Sonnet for tool-heavy skills (investigation, systematic-debugging)
2. **Short-term:** Submit support case for Tier 3 upgrade
3. **Long-term:** Monitor usage and consider multi-account strategy if Tier 3 insufficient

## Related

- **Project:** scs-dev-456116
- **Service:** generativelanguage.googleapis.com
- **Model:** gemini-3-flash (google/gemini-3-flash-preview)
- **gcloud version:** 551.0.0

## Artifacts

**gcloud quota query:**
```bash
gcloud alpha services quota list \
  --service=generativelanguage.googleapis.com \
  --consumer=projects/scs-dev-456116 \
  --format=json | jq '[.[] | select(.consumerQuotaLimits[0].quotaBuckets[]? | .dimensions.model == "gemini-3-flash") | {metric: .metric, unit: .unit, limit: .consumerQuotaLimits[0].quotaBuckets[] | select(.dimensions.model == "gemini-3-flash") | .effectiveLimit}]'
```

**Key metrics:**
```json
{
  "metric": "generativelanguage.googleapis.com/generate_content_paid_tier_2_requests",
  "unit": "1",
  "limit": "2000"  // per minute
}
```

## Next Actions

- [ ] Check Tier 3 quota availability in web UI
- [ ] Submit Google Cloud support case if needed
- [ ] Update orch default model if Tier 3 not available
- [ ] Document workaround in orch-go CLAUDE.md if persistent issue
