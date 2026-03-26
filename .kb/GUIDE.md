# How Work Becomes Understanding

Most systems track what happened. This one tracks what you learned.

The difference sounds small until you've run fifty agents and can't remember what any of them concluded. You have logs, patches, transcripts. What you don't have is a clear answer to: *what does the project believe now, and why?*

That's the problem this method solves. Not by adding process — by giving knowledge somewhere to go.

---

## The Cycle

Here's how a question becomes durable understanding:

```
                    ┌─────────┐
           ┌───────→│ Thread  │←──────────────┐
           │        └────┬────┘               │
           │             │                    │
           │        asks a question           │
           │             │                    │
           │             v                    │
           │     ┌───────────────┐            │
           │     │ Investigation │     feeds back into
           │     └───────┬───────┘            │
           │             │                    │
           │        produces evidence         │
           │             │                    │
           │             v                    │
           │        ┌─────────┐               │
           │        │  Probe  │               │
           │        └────┬────┘               │
           │             │                    │
           │        tests a claim             │
           │             │                    │
           │             v                    │
           │        ┌─────────┐               │
           │        │  Model  │───────────────┘
           │        └────┬────┘
           │             │
           │     when a model matures
           │             │
           │             v
           │       ┌──────────┐     ┌──────────┐
           └───────│  Brief   │     │ Decision │
                   └──────────┘     └──────────┘
                   (for humans)     (for the system)
```

This isn't a pipeline you march through in order. It's a cycle. Models feed questions back into threads. Decisions create new threads. Briefs surface tensions that need investigation. The artifacts compose because each one has a job.

---

## What Each Artifact Does

### Thread — The Spine

A thread is a line of thinking that persists across sessions, agents, and days. Not a task. Not a ticket. A question you're following.

A thread called *"Do stronger models accrete code faster?"* might live for two weeks. It accumulates dated entries as you learn. It links to the investigations it spawned. When the question resolves — or transforms into a better question — the thread records that.

**When to create one:** When you notice you're thinking about the same question across multiple work sessions. When you want to remember not just what happened, but what you were trying to understand.

### Investigation — The Evidence Gathering

An investigation answers a specific question with evidence. Where a thread tracks ongoing thinking, an investigation has a start, findings, and an end.

Each investigation carries a D.E.K.N. header — Delta (what changed), Evidence (what supports it), Knowledge (what we learned), Next (what follows). This isn't ceremony. It's the minimum information someone needs to trust or challenge your conclusion without reading the whole thing.

**When to create one:** When a thread raises a question specific enough to investigate. *"How does X work?"* is an investigation. *"What should we think about X?"* stays in the thread.

### Probe — The Test

A probe tests a specific claim against evidence. Models accumulate claims over time — a probe asks: *is this one still true?*

A model might claim "pre-commit gates reduce accretion." A probe measures the actual accretion rate over two weeks and reports back: confirmed, contradicted, or extended. The probe doesn't live alone — its findings merge back into the parent model.

**When to create one:** When a model makes a claim that matters enough to verify. When you suspect something the system "knows" might be wrong.

### Model — The Synthesis

A model is the system's current best understanding of a domain. Not a spec. Not documentation. A living synthesis that gets stronger as probes and investigations feed into it.

The knowledge-accretion model, for example, tracks how this system accumulates understanding — including honest measurements of where it fails (85% of investigations go orphaned, never linked back). Models carry validation status: working hypothesis, validated, inert. They're meant to be updated, not preserved.

**When to create one:** When you notice repeated investigations in the same area producing findings that should be synthesized. When the same question keeps getting re-investigated because nobody wrote down the answer.

### Brief — The Translation

A brief translates work into comprehension for humans. Three sections: Frame (what was confusing), Resolution (what we figured out), Tension (what's still unresolved).

Briefs exist because raw investigation output isn't readable at scale. If you've run six investigations across three days, nobody is going to read all of them. The brief is what you'd tell someone in two minutes.

**When to create one:** When work completes and the result matters beyond this session. When the orchestrator or a human needs to understand what happened without reading every artifact.

### Decision — The Commitment

A decision records a choice the system has made and why. Standard ADR pattern: context, decision, tradeoffs. But with an enforcement field — is this a product principle? An architectural constraint? A team convention?

Decisions are the most standalone artifact. Someone can read a decision file and understand what was chosen without any other context. That's by design — decisions need to outlast the investigations that produced them.

**When to create one:** When investigation or modeling produces a fork in the road and you pick a direction. When you need to record *why* so future-you doesn't re-litigate it.

---

## The Part Nobody Documents

The cycle above looks clean. Here's what actually happens:

You start working on something. Three sessions in, you realize you keep asking the same question. That's a thread. The thread spawns an investigation. The investigation produces findings that contradict what you assumed. Now you have a probe — *was the old assumption ever true?* The probe says no. The model updates. The updated model reveals a tension you hadn't seen. That becomes a new thread.

**The artifacts don't compose because someone enforces a process.** They compose because each one answers a different question:

| Artifact | Question It Answers |
|---|---|
| Thread | What are we trying to understand? |
| Investigation | What does the evidence say? |
| Probe | Is this specific claim still true? |
| Model | What do we believe, and how confident are we? |
| Brief | What happened, in two minutes? |
| Decision | What did we choose, and why? |

If you're not sure which artifact to create, ask which question you're answering. The answer picks the format.

---

## The Turn

The insight behind this system isn't the artifact formats. Markdown files with headers aren't novel.

The insight is that **execution output and durable knowledge are different things**, and most systems conflate them. An agent runs, produces a patch and a transcript. The patch is execution output — it happened. But the *understanding* of why that approach worked, what constraint it revealed, what the system should believe differently — that's knowledge, and it has nowhere to go unless you give it a place.

That's what the cycle does. Not more process. A place for each kind of understanding to land, so it compounds instead of scattering.

Threads give learning a spine.
Investigations give claims evidence.
Probes keep models honest.
Models give the system memory.
Briefs give humans orientation.
Decisions give the future context.

The system works when each artifact feeds the next. It fails when artifacts accumulate without connecting — orphaned investigations, stale models, briefs nobody reads. The discipline isn't in creating the artifacts. It's in closing the loop.
