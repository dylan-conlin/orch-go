# Brief: orch-go-f8y50

## Frame

The thread-first home surface shipped — threads above operations, nav reordered, status-bucket sorting. But when you open localhost:5188, you still see a dashboard. Thread titles with entry counts. "7 unread" in a badge. "3 blocking" in another badge. The comprehension layer is *referenced* everywhere and *readable* nowhere. The question I needed to answer: what's actually missing between "useful dashboard organized around threads" and "this is the thing"?

## Resolution

I expected the answer to be a missing feature — thread graph visualization, or a digest product, or a ranking system. The turn was: every element that matters already exists. Thread entries exist in the API. Brief content exists in .kb/briefs/ (49 of them, some genuinely compelling — "The Timeout That Looked Like Absence" reads like a story, not a status update). Question text exists in the questions store. The `orch orient` CLI already produces more comprehension than the web surface does.

The gap is rendering mode, not feature inventory. The surface shows *that* things exist (counts, badges, collapsed titles) when it should show *what* they say (prose, synthesis, questions). Through elimination testing — systematically removing each candidate element and asking "does this still feel like the product?" — three elements emerged as jointly necessary: thread entry prose (not titles), brief content inline (not counts), and tension text (not badges). Remove any one and the surface collapses into a recognizable tool category: inbox, notebook, or status page. Add any other element — review queue, agents, knowledge tree — and the identity doesn't change.

The useful byproduct was a boundary rule: a surface change is *product clarification* if it moves a comprehension element from metadata to content rendering. It's *dashboard improvement* if it adds or polishes metadata. This should prevent future work from confusing the two.

## Tension

The whole thesis rests on a claim I can't verify without building it: that content-first rendering actually *feels* different to the person using it. The elimination analysis is logical, not experiential. It's possible that thread entry prose on a 666px half-screen feels like a wall of text, not an orientation surface. It's possible that one inline brief creates reading fatigue instead of comprehension. The only test is a week of use. The question: is the analysis convincing enough to warrant the implementation, or do you want to see a mockup first?
