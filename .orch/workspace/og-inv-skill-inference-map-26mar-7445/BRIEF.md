# Brief: orch-go-hv9lc

## Frame

This looked like a question about "NLP," but the real concern was more practical: when someone writes a task in plain English, what exactly in that wording changes the skill the daemon picks? That matters because a lot of work enters the system as a generic `task`, and if the routing rules are more brittle than they appear, the daemon quietly pushes that work toward `feature-impl`.

## Resolution

The turn was realizing there is almost no NLP here in the modern sense. The daemon does something much more legible: it checks for an explicit `skill:*` label, then looks at title patterns, then scans the description for a short list of hard-coded substrings, and only after all of that falls back to issue type. So the description is not the primary steering wheel; it is the third backup path.

That makes the system easier to reason about than the word "inference" suggests, but it also exposes the cost of vague issue writing. A description like "fix the auth issue" does not trigger the debugging route unless it includes concrete signals such as an error string, reproduction clue, or expected-vs-actual detail. The daemon is choosing predictability over aggressive guessing, which is probably right, but it means better routing comes more from issue enrichment than from smarter free-text interpretation.

## Tension

The open question is whether the next improvement should be stronger issue-authoring discipline or better measurement of routing correctness. Right now the code can tell you which tier won, but not whether the chosen skill was actually the right one.
