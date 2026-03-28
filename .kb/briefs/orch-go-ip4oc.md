# Brief: orch-go-ip4oc

## Frame

Nine out of nine architect completions closed without creating implementation issues. The designs were written, committed, and marked complete — then nothing happened. Twenty design documents sat in .kb/investigations/ with no downstream work. The architect skill's Phase 5d only told agents to create issues for multi-component designs, so single-component designs just got a bullet point. The skill text was fixed (Phase 6: Handoff added), but advisory-only gates have a 100% bypass rate. The question was: where's the code-level gap?

## Resolution

The gate already existed — `VerifyArchitectHandoff` — but it had a blind spot I didn't expect. When SYNTHESIS.md was missing, the gate returned "passing" with a comment: "synthesis gate handles that separately." That was wrong. Architects run at verification level V1, and the synthesis gate only fires at V2+. So the gate was deferring to a check that never ran. The result: an architect without SYNTHESIS.md sailed through the entire completion pipeline without anyone noticing.

The fix was surgical: make the architect_handoff gate self-sufficient instead of relying on a higher-level gate that may not exist for this skill. Missing SYNTHESIS.md now fails the gate directly. I also wired in the comment-based signals from the new Phase 6 skill text — if an architect manually creates issues and reports them in a Phase: Handoff comment, the gate recognizes that. And if the architect explicitly opts out ("No implementation issues: [reason]"), the gate passes with a warning instead of blocking.

The defect class is Filter Amnesia (Class 1 in the taxonomy): a validation existed in path A (V2 synthesis gate) but was absent from path B (V1 architect_handoff gate). The assumption that "someone else checks this" was never verified against the actual execution path.

## Tension

The architect_handoff gate and the synthesis gate now both check SYNTHESIS.md for architect skill — there's redundancy if architect is ever bumped to V2. More importantly: the comment-based opt-out ("No implementation issues: [reason]") is a string match on free-text beads comments. It works, but it's a softer signal than the structured title-pattern match from auto-create. If agents learn to write "No implementation issues: it's fine" without genuine reasoning, the gate becomes decorative again. The structural question is whether comment-based gates can stay honest without something verifying the quality of the reason.
