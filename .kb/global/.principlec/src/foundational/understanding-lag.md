### Understanding Lag

Observability can improve faster than humans can understand what new visibility means. When new visibility reveals problems, pause and ask: "Are these new problems, or newly-visible old problems?"

**The test:** "Am I interpreting new visibility as system degradation, or as seeing what was always there?"

**What this means:**

- New monitoring, dashboards, or alerts reveal things that were previously invisible
- The revealed problems often existed before - they're newly-visible, not newly-created
- The urge to "roll back the monitoring" is a sign of understanding lag
- Correct response: fix the underlying issue, not hide the visibility

**What this rejects:**

- "The system is worse since we added monitoring" (you're seeing what was hidden)
- "These alerts are too noisy" (maybe they're showing real issues)
- "Let's roll back the dashboard changes" (hides the problem again)

**The failure mode:** During Dec 27-Jan 2, agents added dead session detection. Dashboard started showing "dead" and "stalled" agents. We interpreted this as system degradation and rolled back 347 commits. Investigation revealed: those agents had ALWAYS been dead - we just couldn't see them before.

**Why distinct from Verification Bottleneck:** Verification Bottleneck is about code changes outpacing behavioral verification. Understanding Lag is about observability changes outpacing meaning interpretation.

**When adding new observability:**

1. **Baseline first** - What does this show for known-good state?
2. **Known-bad test** - What does this show for known-bad state?
3. **Gradual rollout** - Time to understand before full visibility
4. **Frame correctly** - "This metric shows X, which existed before we could see it"
