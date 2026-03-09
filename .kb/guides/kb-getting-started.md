# Getting Started: The Investigation/Probe/Model Cycle

A guide for solo technical researchers who use AI agents and want their knowledge to compound across sessions.

---

## The Problem

You use AI agents (Claude Code, Cursor, etc.) to investigate complex questions. Each session produces insights — but the next session starts from zero. You've solved the same problem three times because no session remembers what the others learned.

The investigation/probe/model cycle fixes this. It gives your AI agents a shared knowledge substrate that accumulates understanding over time.

---

## What You'll Build

By the end of this guide, you'll have:
1. A `.kb/` directory tracking your project's knowledge
2. Your first **investigation** — a structured answer to a real question
3. Your first **model** — synthesized understanding from multiple investigations
4. Your first **probe** — a targeted test of a model's claims

The cycle: **investigate** (ask questions) → **synthesize** (build models) → **probe** (test claims) → **update** (refine models). Each turn makes the system smarter.

---

## Prerequisites

- **Git** — your project should be in a git repo
- **Claude Code** (`claude` CLI) — or any AI agent with file system access
- **kb CLI** (`kb`) — install from the kb-cli repo

---

## Step 1: Initialize Your Knowledge Base

```bash
cd your-project
kb init
```

This creates a `.kb/` directory with subdirectories for investigations, decisions, and guides. You'll add the `models/` directory yourself shortly.

Commit the skeleton:

```bash
git add .kb/
git commit -m "Initialize knowledge base"
```

---

## Step 2: Your First Investigation

Pick a real question about your project — something you'd normally ask an AI agent and forget the answer to.

```bash
kb create investigation "how-auth-tokens-refresh"
```

This creates `.kb/investigations/YYYY-MM-DD-inv-how-auth-tokens-refresh.md` with a template. Open it and fill in:

### Required Sections

```markdown
# Investigation: How Auth Tokens Refresh

**Date:** 2026-03-09
**Status:** Active

## Question

How does the auth token refresh flow work? When does it fail?

## Findings

### Finding 1: Refresh happens on 401 response
[What you discovered, with evidence]

### Finding 2: Concurrent refreshes race
[Another finding]

## Conclusion

### D.E.K.N. Summary
- **Delta:** Token refresh uses a single-retry pattern with no mutex
- **Evidence:** `pkg/auth/client.go:142` — refresh called inline on 401
- **Knowledge:** Concurrent requests can trigger multiple refreshes simultaneously
- **Next:** Add mutex around refresh, or investigate token cache
```

**D.E.K.N.** (Delta, Evidence, Knowledge, Next) ensures every investigation produces actionable output, not just notes.

Run your investigation using Claude Code — ask it to explore the question, test behavior, and document findings in the investigation file. When done, set `Status: Complete` and commit.

```bash
git add .kb/investigations/
git commit -m "investigation: how auth tokens refresh"
```

---

## Step 3: Build Understanding (More Investigations)

Run 2-3 more investigations on related topics. For the auth example:

```bash
kb create investigation "token-storage-locations"
kb create investigation "auth-failure-modes"
```

Each investigation stands alone. But after three investigations on auth, you'll notice patterns — recurring findings, shared failure modes, consistent behavior. That's the signal to create a model.

### Finding Prior Work

Before each investigation, check what's already known:

```bash
kb context "auth token refresh"
```

This returns relevant existing artifacts — investigations, decisions, models. Read them first. Don't re-investigate solved questions.

---

## Step 4: Your First Model

When 3+ investigations converge on the same topic, synthesize them into a model.

Create the model directory and file:

```bash
mkdir -p .kb/models/auth-system
mkdir -p .kb/models/auth-system/probes
```

Create `.kb/models/auth-system/model.md`:

```markdown
# Model: Auth System

**Domain:** Authentication and token management
**Last Updated:** 2026-03-09
**Synthesized From:**
- `.kb/investigations/2026-03-09-inv-how-auth-tokens-refresh.md`
- `.kb/investigations/2026-03-10-inv-token-storage-locations.md`
- `.kb/investigations/2026-03-11-inv-auth-failure-modes.md`

---

## Summary (30 seconds)

Auth uses OAuth2 with refresh tokens stored in the system keychain.
Token refresh triggers on 401 responses with a single-retry pattern.
The primary failure mode is concurrent refresh races when multiple
requests hit 401 simultaneously.

---

## Core Mechanism

### Key Components
- Access token (short-lived, in memory)
- Refresh token (long-lived, in keychain)
- Token refresh interceptor (triggers on 401)

### Critical Invariants

1. **Refresh tokens are single-use.** Using a refresh token invalidates it
   and returns a new one. If two requests refresh concurrently, one gets
   an invalid token.

2. **Token storage is the source of truth.** The keychain value is
   authoritative. In-memory tokens are caches.

3. **401 means expired, not unauthorized.** The system assumes 401
   always means "refresh needed," which breaks for genuinely
   unauthorized requests.

---

## Why This Fails

### Concurrent refresh race
Two requests hit 401 → both call refresh → first succeeds, second
gets "invalid_grant" → second request fails permanently.

### Keychain read latency
macOS keychain reads take 50-200ms. Under load, this adds visible
latency to every token refresh.

---

## Constraints

### Why not store tokens in a file?
**Constraint:** macOS keychain provides OS-level encryption at rest.
**Implication:** File-based storage would require implementing encryption.

---

## Evolution

**2026-03-09:** Initial model from three auth investigations.

---

## References

**Investigations:**
- `.kb/investigations/2026-03-09-inv-how-auth-tokens-refresh.md` - Refresh mechanism
- `.kb/investigations/2026-03-10-inv-token-storage-locations.md` - Storage analysis
- `.kb/investigations/2026-03-11-inv-auth-failure-modes.md` - Failure modes

**Primary Evidence:**
- `pkg/auth/client.go:142` - Refresh interceptor
- `pkg/auth/keychain.go:28` - Keychain read/write
```

**The Critical Invariants section is the most important part.** These are numbered, testable claims. They become the targets for probes.

Commit:

```bash
git add .kb/models/auth-system/
git commit -m "model: auth system — synthesized from 3 investigations"
```

---

## Step 5: Your First Probe

A probe tests a specific claim in a model. Pick one of your Critical Invariants and test it empirically.

Create `.kb/models/auth-system/probes/2026-03-12-probe-concurrent-refresh-race.md`:

```markdown
# Probe: Concurrent Refresh Race Condition

**Model:** auth-system
**Date:** 2026-03-12
**Status:** Active

---

## Question

Critical Invariant #1 claims refresh tokens are single-use and concurrent
refreshes cause races. Does this actually happen under normal load, or is
it theoretical?

---

## What I Tested

Simulated concurrent 401 responses by adding a sleep to the refresh
endpoint and firing 5 parallel requests.

```bash
# Test script that sends 5 concurrent requests with expired token
for i in {1..5}; do curl -H "Authorization: Bearer expired" http://localhost:8080/api/data & done
wait
```

---

## What I Observed

3 of 5 requests failed with "invalid_grant" errors. The refresh
interceptor has no deduplication — each goroutine independently calls
the refresh endpoint.

Server logs show 5 refresh attempts within 12ms. First succeeds,
remaining 4 use the now-invalidated refresh token.

---

## Model Impact

- [x] **Confirms** invariant: #1 (refresh tokens are single-use)
- [x] **Extends** model with: The race isn't theoretical — it triggers
  under modest concurrency (5 requests). Fix: add a mutex or
  singleflight around the refresh call.

---

## Notes

The fix is straightforward: `sync.Once` or `golang.org/x/sync/singleflight`
around the refresh call. Estimated 10-line change.
```

### The Merge Step (Required)

After completing a probe, **merge its findings into the parent model**. This is what makes the cycle compound.

Edit `.kb/models/auth-system/model.md`:
- Update the "Concurrent refresh race" failure mode with the empirical evidence
- Add a note that it triggers under modest concurrency (5 requests)
- Reference the probe in the Evolution section

```markdown
## Evolution

**2026-03-09:** Initial model from three auth investigations.
**2026-03-12:** Probe confirmed concurrent refresh race is not theoretical —
triggers at 5 concurrent requests. singleflight recommended.
```

Commit both together:

```bash
git add .kb/models/auth-system/
git commit -m "probe: concurrent refresh race — confirmed under 5 concurrent requests"
```

---

## The Cycle in Practice

Once you have a model, the cycle becomes natural:

1. **New question arises** → check `kb context "topic"` for existing models
2. **Model exists** → create a probe testing a specific claim
3. **No model exists** → create an investigation
4. **3+ investigations converge** → synthesize into a model
5. **Probe finds something** → merge findings into the model
6. **Repeat**

Each turn makes the knowledge base more accurate. AI agents that read the model before working get pre-loaded with your project's hard-won understanding — they don't start from zero.

---

## Using It With AI Agents

### Manual (Any AI Agent)

Tell your agent at the start of a session:

> Before investigating, run `kb context "your topic"` to find existing models.
> If a model exists, create a probe in `.kb/models/{name}/probes/` testing
> a specific claim. If no model exists, create an investigation with
> `kb create investigation "slug"`. Always merge probe findings back into
> the parent model before finishing.

### With Claude Code

Add to your project's `CLAUDE.md`:

```markdown
## Knowledge Base

Before investigating any topic, run `kb context "topic"` to check for
existing models in `.kb/models/`. If a relevant model exists, create a
probe testing a specific claim rather than a fresh investigation.

Probe files go in `.kb/models/{model-name}/probes/YYYY-MM-DD-probe-{slug}.md`.
After completing a probe, merge findings into the parent model.md.
```

### With the Investigation Skill (Advanced)

If you have the investigation skill installed (`~/.claude/skills/worker/investigation/SKILL.md`), it automatically detects whether to run in investigation mode or probe mode based on whether model claims are present in the context.

---

## Reflection: Finding Synthesis Opportunities

Periodically check for knowledge that should be synthesized:

```bash
kb reflect
```

This surfaces:
- **Synthesis opportunities** — clusters of 3+ investigations on the same topic (model candidates)
- **Stale models** — models that haven't been probed recently
- **Recurring patterns** — themes across investigations

---

## Measuring Health

As your knowledge base grows, track these metrics:

```bash
# How many investigations connect to models?
total=$(find .kb/investigations -name "*.md" 2>/dev/null | wc -l)
echo "Total investigations: $total"

# How many models exist?
find .kb/models -name "model.md" 2>/dev/null | wc -l

# How many probes per model?
for model in .kb/models/*/; do
  name=$(basename "$model")
  count=$(find "$model/probes" -name "*.md" 2>/dev/null | wc -l)
  echo "$name: $count probes"
done

# Unmerged contradictions (knowledge debt)
grep -rl "Contradicts" .kb/models/*/probes/*.md 2>/dev/null
```

A healthy knowledge base has:
- Models for your most-investigated topics
- Regular probes testing model claims
- Zero unmerged contradiction verdicts
- Investigations citing prior work via `kb context`

---

## Quick Reference

| Artifact | Location | When to Create |
|----------|----------|---------------|
| Investigation | `.kb/investigations/YYYY-MM-DD-inv-{slug}.md` | New question, no relevant model |
| Model | `.kb/models/{name}/model.md` | 3+ investigations converge |
| Probe | `.kb/models/{name}/probes/YYYY-MM-DD-probe-{slug}.md` | Testing a model's claim |
| Decision | `.kb/decisions/YYYY-MM-DD-{slug}.md` | Recording a choice and rationale |

| Command | Purpose |
|---------|---------|
| `kb init` | Initialize `.kb/` in current project |
| `kb create investigation "slug"` | Create investigation file |
| `kb context "query"` | Find relevant existing knowledge |
| `kb reflect` | Surface synthesis opportunities |
| `kb list` | List artifacts by type |
| `kb search "term"` | Search knowledge artifacts |

---

## What's Next

- **Scale:** When you're running multiple AI agents on investigations simultaneously, look into orchestration tools (orch, beads) for tracking and quality gates
- **Theory:** The investigation/probe/model cycle follows predictable dynamics described in the knowledge-physics model — accretion, attractors, gates, and entropy are substrate-independent
- **Gates:** As your knowledge base grows, consider adding pre-commit hooks or tooling checks to enforce conventions (like requiring probes to reference their parent model)
