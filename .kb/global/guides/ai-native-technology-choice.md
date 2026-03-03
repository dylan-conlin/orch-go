# AI-Native Technology Choice

Guidelines for choosing languages and frameworks when AI writes the code.

**Key insight:** When AI handles development, language choice shifts from "what's pleasant to work in" to "what artifact do I want to exist."

---

## The Paradigm Shift

**Old model:** Choose languages based on your ability to write, debug, and maintain code.

**New model:** Choose languages based on:
- Artifact quality (binary size, performance, dependencies)
- Deployment characteristics (single binary vs runtime deps)
- Compiler strictness (catches AI mistakes before runtime)
- Installation simplicity for end users

**What matters less:**
- Syntax pleasantness (AI doesn't care)
- Learning curve (AI already knows it)
- Verbose boilerplate (AI types for free)
- Ecosystem "vibes" (AI can navigate any ecosystem)

**The core principle:** Languages that were "too annoying" for humans (verbose, complex type systems) become viable. AI absorbs the annoyance; you get the benefits.

---

## Decision Framework by Project Type

### CLIs: Go or Rust

**Why:** The artifact is what users experience. Single binary, no runtime dependencies, fast startup.

| Dimension | Python | Go | Rust |
|-----------|--------|-----|------|
| Distribution | `pip install` + Python version hell | Single binary | Single binary |
| Binary size | N/A (needs runtime) | ~5-10MB | ~1-3MB |
| Startup time | 100-300ms | <10ms | <10ms |
| Cross-compilation | Painful | Easy | Easy |

**Recommendation:** Go for most CLIs (simpler, fast enough). Rust when binary size or peak performance matters.

---

### Frontends: Modern Stack (Bun/Svelte/shadcn)

**Why:** The artifact is the JS/CSS bundle. Smaller bundles, faster runtime, no virtual DOM overhead.

| Framework | Bundle Size | Runtime Perf | Human Learning Curve | AI Learning Curve |
|-----------|-------------|--------------|----------------------|-------------------|
| React | Larger | Virtual DOM overhead | Low | Zero |
| Svelte | Smaller | No runtime, compiled | Medium | Zero |

**The insight:** Svelte was "better artifact, more to learn." With AI, you just get "better artifact."

**Recommended stack:**
- **Bun:** Faster builds, faster runtime, npm-compatible
- **Svelte:** Smaller bundles, no virtual DOM, compiled
- **shadcn:** Copy-paste components you own (not a dependency)

---

### Web Backends: Python/Node (Usually)

**Why:** Ecosystem richness still matters. Auth, databases, deployment tooling are more mature.

| Dimension | Python/Node | Go | Rust |
|-----------|-------------|-----|------|
| Response latency | Fine (10-50ms) | Better (1-10ms) | Best (<5ms) |
| Memory usage | 50-200MB | 10-30MB | 5-20MB |
| Cold start (serverless) | 200-500ms | 50-100ms | 10-50ms |
| Ecosystem richness | ★★★★★ | ★★★ | ★★ |
| Deployment simplicity | ★★★★★ | ★★★★ | ★★★ |

**Default:** Python/Node. The ecosystem wins until proven otherwise.

**Consider Go/Rust when:** See "When Backend Performance Matters" below.

---

### Scripts and Glue: Python

**Why:** Artifact doesn't matter. Iteration speed does. AI can write Python quickly, you can read/modify it easily.

No change from the old model here.

---

## When Backend Performance Matters

Most projects don't need Go/Rust backends. Here's how to know when you do:

### Scale Thresholds

| Metric | Python/Node fine | Consider Go/Rust |
|--------|------------------|------------------|
| Requests/sec | < 1,000 | > 10,000 |
| Concurrent connections | < 10,000 | > 100,000 |
| Monthly hosting bill | < $100 | > $500 |
| Response time budget | > 100ms | < 20ms |

### Specific Scenarios

**Serverless cold starts matter when:**
- User-facing endpoints (not background jobs)
- Infrequent invocations (cold starts happen often)
- Latency-sensitive (checkout, auth, real-time)

**CPU-bound work:**
- Image/video processing
- Heavy computation
- Data transformation at scale
- Python is 10-100x slower for CPU work

**Memory-constrained:**
- Edge devices / Raspberry Pi
- Cheap VPS ($5/month tier)
- Tight container limits

### The Honest Heuristics

**Performance probably doesn't matter if:**
- You're pre-product-market-fit
- You have < 1,000 users
- Your bottleneck is the database, not the app
- You're optimizing before measuring

**Performance probably matters if:**
- You're paying > $500/month in compute
- Users complain about speed
- You've profiled and app code is the bottleneck
- You're building infrastructure others depend on

### The Real Answer

**Measure first.** Most performance intuitions are wrong.

```bash
# Is your app actually slow?
curl -w "%{time_total}\n" -o /dev/null -s https://your-api/endpoint

# Where is the time going?
# Usually: database queries, external APIs, not your code
```

---

## The Hybrid Pattern

**Recommended default stack for AI-native development:**

```
Frontend:  Bun + Svelte + shadcn
           (better artifact, AI handles complexity)

Backend:   Python/Node for most things
           (ecosystem matters more than raw performance)

CLI tools: Go or Rust
           (artifact matters, ecosystem doesn't)

Scripts:   Python
           (iteration matters, artifact doesn't)

Performance-critical services: Go/Rust
           (when measurement proves it matters)
```

---

## Summary

| Project Type | Old Default | AI-Native Choice | Why |
|--------------|-------------|------------------|-----|
| CLIs | Python | Go/Rust | Single binary distribution |
| Frontends | React | Svelte + modern tooling | Smaller bundles, no runtime |
| Backends | Python/Node | Python/Node (still) | Ecosystem > raw performance |
| Scripts | Python | Python | No change needed |
| Perf-critical | "Hire Rust dev" | Go/Rust | AI writes it, you get benefits |

**The principle:** Optimize for the artifact when the artifact matters. Optimize for ecosystem when the ecosystem matters. AI changes what's reachable, not what's valuable.

---

## Lineage

**Emerged from:**
- Investigation: `agentlog/.kb/investigations/2025-12-10-inv-native-language-choice-agentlog-interactive.md` - Language choice for agentlog CLI
- Practical experience building orch-cli (Go), beads (Go), kn (Go), kb-cli (Go)

**Related guides:**
- `.kb/guides/ai-first-cli-rules.md` - Interface design for AI-consumable CLIs

**Last updated:** 2025-12-10
