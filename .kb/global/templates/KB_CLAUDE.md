# Knowledge Base

This project uses a knowledge base (`.kb/`) to compound understanding across sessions. AI agents and humans contribute investigations, probes, and models that build on each other instead of starting from scratch.

## The Cycle

```
Question → Investigation → Findings → Model
                ↑                        │
                └── Probe ←──────────────┘
```

1. **Investigation:** Observe and test to answer a question. Evidence comes from running code, not reading artifacts.
2. **Model:** Synthesized understanding from 3+ investigations on the same topic. Models describe how things work and why they fail.
3. **Probe:** Test a specific claim in an existing model. Confirms, contradicts, or extends the model.

## Before You Investigate

Check what's already known:

```bash
kb context "your topic"
```

This returns existing models, investigations, decisions, and constraints. Read any relevant model files — they contain synthesized understanding you should build on, not repeat.

If a model exists for your topic, you're in **probe mode**: test its claims rather than starting a fresh investigation.

## Directory Structure

```
.kb/
├── investigations/          # Observations and findings
│   └── YYYY-MM-DD-inv-*.md
├── models/                  # Synthesized understanding (the important stuff)
│   └── {topic}/
│       ├── model.md         # The model itself
│       └── probes/          # Evidence gathered against this model
│           └── YYYY-MM-DD-probe-*.md
├── decisions/               # Choices made and why
├── guides/                  # How-to references
└── quick/                   # Quick entries (constraints, attempts, questions)
    └── entries.jsonl
```

## Creating Artifacts

### Investigations

```bash
kb create investigation "descriptive-slug" --model model-name
# or if no model applies:
kb create investigation "descriptive-slug" --orphan
```

**Required sections:**
- **Question** — What you're testing (specific, not vague)
- **What I Tested** — Actual commands, code runs, observations
- **What I Observed** — Concrete results (not interpretation)
- **Conclusion** — What the evidence shows

**The rule:** Only observed behavior is evidence. What code does is not what comments or docs say it does.

### Models

Create when 3+ investigations converge on a coherent mechanism:

```bash
mkdir -p .kb/models/{topic}/probes
# Copy template or create model.md with these sections:
```

**Required sections:**
- **Summary (30 seconds)** — One paragraph explaining the mechanism
- **Core Mechanism** — How it works, key components, critical invariants
- **Why This Fails** — Failure modes and root causes
- **References** — Links to investigations that built this understanding

### Probes

Test a claim in an existing model:

```bash
# Create in the model's probes directory:
# .kb/models/{model-name}/probes/YYYY-MM-DD-probe-{slug}.md
```

**Required sections:**
- **Question** — Which model claim are you testing?
- **What I Tested** — Commands and experiments run
- **What I Observed** — Concrete results
- **Model Impact** — Does it confirm, contradict, or extend the model?

**After writing a probe, update the parent model.md** with your findings. Probes that sit unmerged break the knowledge loop.

## Quick Entries

For things too small for a full investigation:

```bash
kb quick decide "We'll use X because Y"
kb quick constrain "System can't do X because Y"
kb quick tried "Attempted X, failed because Y"
kb quick question "Should we X or Y?"
```

## Finding Patterns

```bash
kb reflect              # Surface synthesis opportunities, stale decisions, patterns
kb search "topic"       # Search across all artifact types
kb context "topic"      # Get relevant context for a question
```

## Principles

- **Evidence hierarchy:** Primary (code behavior, test output) > Secondary (artifacts, docs). Trust code over models. When they conflict, code wins — update the model.
- **Models are living documents.** Update them when understanding changes.
- **Probes structurally couple findings to models.** By living in `.kb/models/{name}/probes/`, they create a directory-level connection that prevents findings from becoming orphans.
- **Every investigation should check prior work.** Run `kb context` before starting. Don't re-investigate what's already known.
