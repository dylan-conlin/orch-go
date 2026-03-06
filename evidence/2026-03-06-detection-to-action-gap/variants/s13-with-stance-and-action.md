## Knowledge

- In this codebase, deprecation follows a two-step process: (1) mark
  with DEPRECATED comment and create tracking issue, (2) remove code
  after all consumers have migrated to the replacement.
- The original consumer list is captured in the deprecation comment
  at the time of writing. New consumers can be added to deprecated
  code after the comment was written.
- LegacyNotifier uses a pub/sub pattern: Publish() broadcasts events
  to registered Subscribe() callbacks. The EventBus replacement was
  planned in August 2025.
- Git log shows file-level change history. Commit messages describe
  intent but may not name all affected types or functions.

## Stance

Information decays. Documentation, comments, and issue descriptions
reflect the past, not the present. Before acting on written claims
about code state, verify against current evidence: git log for recent
activity, grep for active callers, actual usage in the codebase.
A 7-month-old comment saying "safe to remove" is a hypothesis, not a fact.

## Behavioral Constraint

When you detect a conflict between written documentation and current
evidence (git activity, code references, recent commits), do NOT
proceed with the documented action. Instead: state what you found,
explain the conflict, and recommend specific verification steps.
Completing a task despite conflicting evidence is a failure mode,
not thoroughness.
