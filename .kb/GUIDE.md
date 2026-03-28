# The Method

You arrive with intent — something to understand, build, or fix. Before reaching for a tool, notice the shape of the work.

## Shape

| Shape | You have... | First move |
|-------|-------------|------------|
| **Search** | Multiple unknowns, no convergence yet | `orch spawn --explore` — fan out, gather evidence |
| **Convergence** | Pieces exist, need sequencing | Plan — sequence what you already know |
| **Forming** | A question that isn't sharp yet | `orch thread new` — give the question a place to develop |
| **Execution** | Clear scope, known approach | `orch spawn` — implement directly |

Most intent starts as forming or search. If you're unsure, it's forming — start a thread.

---

## The Composition Cycle

Once you know the shape, this is how the pieces relate:

```
Thread (question)
    ↓
Investigation (evidence)
    ↓
Probe (test a claim)
    ↓
Model (update understanding)
    ↓
Brief (synthesize for human)
    ↓
Decision (resolve)  ──or──  New thread (next question)
```

This isn't a pipeline you march through in order. Models feed questions back into threads. Decisions create new threads. Briefs surface tensions that need investigation. Each artifact has a job:

**Thread** — A line of thinking that persists across sessions and days. Not a task. A question you're following. Investigations gather evidence for threads. Decisions resolve branches of threads. The thread is the spine; everything else hangs off it.

**Investigation** — Structured evidence gathering with a start, findings, and an end. Each one carries a D.E.K.N. header — Delta (what changed), Evidence (what supports it), Knowledge (what we learned), Next (what follows). The minimum someone needs to trust or challenge your conclusion.

**Probe** — Tests a specific claim in a model. *"The model says X — is X still true?"* Returns a verdict: confirm, contradict, or extend. Findings merge back into the parent model.

**Model** — The system's current best understanding of a domain. A living synthesis that gets stronger as probes feed it. Models carry validation status (working hypothesis, validated, inert) and explicit uncertainty. They're meant to be updated, not preserved.

**Brief** — Translates work into comprehension for a human. Three sections: Frame (why it matters), Resolution (what was found), Tension (what's unresolved). Here's a real one:

> **Frame.** You accepted that comprehension is the product and execution is substrate. But what does that look like in the actual codebase? I went through every package and counted, expecting rough parity. The number I found was 16/72.
>
> **Resolution.** Three packages alone contain 48% of all code, all substrate. The thread package — the conceptual spine — is seventeen times smaller than the daemon. But the ratio isn't wrong in itself. Some of that plumbing was the scaffolding that revealed the stronger thesis.
>
> **Tension.** The ratio tells you where the code mass is, not the value mass. Do you want to change the ratio, or change the front door? The front door change is a weekend. The ratio change is a quarter.

**Decision** — Records a choice and why. Context, decision, tradeoffs, enforcement level. Decisions are the most standalone artifact — readable without any other context.

---

## How Shape Enters the Cycle

- **Forming** → starts a thread → investigations follow when the question sharpens
- **Search** → spawns parallel investigations → findings converge into a thread
- **Convergence** → enters at the decision point → sequences existing evidence
- **Execution** → produces a brief when done → feeds back into active threads

---

## The Discipline

The artifacts don't compose because someone enforces a process. They compose because each one answers a different question:

| Artifact | Question |
|---|---|
| Thread | What are we trying to understand? |
| Investigation | What does the evidence say? |
| Probe | Is this claim still true? |
| Model | What do we believe, and how confident are we? |
| Brief | What happened, in two minutes? |
| Decision | What did we choose, and why? |

The system works when each artifact feeds the next. It fails when artifacts accumulate without connecting. The discipline isn't in creating them — it's in closing the loop.
