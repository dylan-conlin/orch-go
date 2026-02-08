# Investigation: Pattern/Tool Relationship and Shareability

## Summary (D.E.K.N.)

**Delta:** Identified three-layer model (principle → structure → practice) and "teeth check" diagnostic. Skillc sits at structure layer, distinct from workflow tools.

**Evidence:** 2025-12-25 session exploring why skillc felt different from orch/kb. Built `skillc verify` to test whether structure without enforcement has value.

**Knowledge:** The shareability question depends on where in the stack something sits. Principles share as writing. Structures share as examples + tools. Practice doesn't share.

**Next:** Continue exploration. Map more examples. Test if model holds.

---

## Status

**Phase:** Complete (archived)  
**Resolution:** Unresolved - model emerging but not validated

---

## Question

What is the relationship between patterns and tools, and how does that affect what's shareable?

## Context

Session started with "share principles, keep tools private" as the accepted wisdom. But skillc felt different. Exploring why led to a taxonomy:

| Layer | What | Shareable as | Example |
|-------|------|--------------|---------|
| Principle | Conceptual insight | Writing | "Gate over remind" |
| Structure | Embodiment of principle | Example + tool | skillc, D.E.K.N. format |
| Practice | Validation through use | Not shareable | 779 investigations |

## Key Observations

### 1. Structure without teeth is a reminder

The investigation skill's `outputs.required` pattern existed in the manifest, but nothing enforced it. An agent could skip creating the artifact entirely.

The question "does it have teeth?" revealed this gap:
- What breaks when this is violated?
- If nothing breaks, it's a reminder, not a gate

This led to building `skillc verify` - adding enforcement to the structure layer.

### 2. Skillc is different from orch

| Property | orch/kb/kn | skillc |
|----------|------------|--------|
| Coupled to Dylan's workflow | High | Low |
| Standalone utility | No (needs ecosystem) | Yes (just builds markdown) |
| Value without ecosystem | Limited | Full |

Skillc embodies "Self-Describing Artifacts" principle in a self-contained way. You don't need the orchestration system to get value.

### 3. The blog post test

"English is the New Programming Language" claimed skillc was a type system. But without `verify`, the claim was hollow - structure without enforcement.

After adding `verify`, the claim has teeth. The tool delivers what the writing promises.

### 4. "Teeth check" as diagnostic

Emerged during session as a reusable question:
- When adding a principle → what breaks when violated?
- When claiming a tool enforces something → where's the gate?
- When a pattern feels important but vague → what's the observable failure?

Not sure yet if this is a principle, a diagnostic, or just a useful question.

## Open Questions

1. **Does the three-layer model hold for other examples?**
   - What about beads? (tool that embodies "surfacing over browsing")
   - What about the investigation skill itself? (structure that embodies "test before concluding")

2. **Is "teeth check" a principle or a diagnostic?**
   - Principles have criteria: tested, generative, not derivable, has teeth
   - "Teeth check" is meta - it's how you check if something has teeth
   - Maybe it's a diagnostic, not a principle

3. **When does a pattern require tooling to be useful?**
   - "Gate over remind" doesn't require tooling - you can implement gates however
   - "Compile skills with checksums" kind of does - the value is in automation
   - What's the distinguishing factor?

4. **What's the publishing threshold?**
   - Skillc is now private again
   - The question isn't "should we share?" but "when is it ready?"
   - What would make it ready? More validation? Better examples? Just deciding?

## Evidence Gathered

- Built `skillc verify` command (orch-go-mkie) ✅
- Created integration issue (orch-go-loh8)
- Created issue to add outputs.required to 5 more skills (orch-go-0iim)
- Made skillc repo private pending further exploration

## Next Steps

- [ ] Map more examples to the three-layer model
- [ ] Decide if "teeth check" is worth capturing (and where)
- [ ] Revisit publishing question after more practice with verify
