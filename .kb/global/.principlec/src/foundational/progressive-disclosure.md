### Progressive Disclosure

TLDR first. Key sections next. Full details available.

**Why:** Context windows are finite. Attention is limited. Front-load the signal.

**Pattern:** Summary → Key Findings → Details → Appendix

**Applied examples:**

- **Skill bloat reduction:** 89% of feature-impl spawns use 2-3 phases. Extract detailed phase guidance to reference docs, keep core workflow inline. Result: 77% reduction (1757→400 lines) without quality loss.
- **Dashboard sections:** Active/Recent/Archive with temporal thresholds — operational visibility (active always visible), historical debugging (expand as needed), UI clarity (collapsed sections reduce clutter).
- **Agent card summaries:** Show full TLDR on completed cards — users want the complete summary at a glance, not click-to-expand.

**The test:** Is the reader forced to process everything to find what matters? If yes, restructure for progressive disclosure.
