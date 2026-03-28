# Brief: orch-go-4k8z9

## Frame

The named-incompleteness model makes a bold spatial claim: gaps are "specific coordinates in possibility space" while conclusions are "generic." When Dylan noticed this language and asked whether it's metaphor or something quantifiable, the question became: do independent traditions — ones that have never heard of each other's work — converge on the same structural asymmetry between questions and statements?

## Resolution

I expected to find the spatial language was a useful analogy that mapped loosely to one or two traditions. What I found was convergence across four, and the convergence corrected the model's language in a way I didn't anticipate.

The surprise was information geometry. The Fisher metric does formalize curvature differences — but in the *opposite* direction from what you'd naively expect. Conclusions (near-certainty distributions) have *tighter* curvature than gaps (high-entropy distributions). I almost called this a contradiction. Then I read deeper into e-projections and m-projections (Amari's dual geometry), and the real structure emerged: a question doesn't define a *point* that's more specific — it defines a *constraint surface* (a submanifold) that specifies which dimensions matter. A conclusion is a point that could have arrived from many different surfaces. The gap carries more structural information because it encodes the *relationship between dimensions*, not just a location.

This maps precisely to a formal proof I found in a completely different tradition. Inquisitive semantics (Ciardelli, Groenendijk, Roelofsen 2018) proves — actually proves, in a formal logical framework — that questions carry strictly more structure than assertions. Assertions eliminate possible worlds. Questions do the same elimination *plus* partition what remains into alternatives. The !-operator that strips a question down to its assertive content literally "cancels the issues raised while leaving its informational content untouched." Questions are assertions plus structure.

Then cognitive science: tip-of-the-tongue states carry the target word's first letter, syllable count, and semantic neighborhood before retrieval succeeds. The gap has a precise, structured address in memory. Loewenstein's curiosity research shows an inverted U — maximum curiosity at intermediate knowledge, where the gap is specific enough to point somewhere but incomplete enough to pull you forward. And bibliometrics: de Solla Price measured that research fronts (active gaps) cluster at 6x the citation density of established knowledge.

Four traditions, four different methods, same conclusion: gaps define constraint surfaces; conclusions are underdetermined points on those surfaces. The model's spatial language isn't metaphor — it maps to real mathematical, logical, cognitive, and bibliometric structure. But the language needed sharpening: "specific coordinates" understates it. Gaps define the *submanifold geometry* that makes convergence possible.

## Tension

One tradition didn't deliver what I expected: NLP. The architectural evidence is strong — HyDE gets +38% retrieval by converting questions to documents before embedding, asymmetric encoders produce disjoint Q/A clusters, PromptBERT shows 34-point shifts from syntactic templates. But nobody has run the controlled experiment: take N topics, generate matched question/statement pairs, embed them, compare cluster properties. The specific measurement that would make the NI claim empirically testable in the most accessible substrate is an identified gap. Is this worth running as an experiment? It could be a small but genuine novel contribution — or it could be obvious enough that the result would be unsurprising.
