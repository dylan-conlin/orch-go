### Pain as Signal

Autonomous error correction requires that agents *feel* the friction of their own failure in real-time. Friction that is only observed by the human is wasted signal for the agent.

**The test:** "Does the agent *feel* the friction of its own failure in its active context?"

**What this means:**

- **Infrastructure over Reminder:** System-level sensors (coaching plugins) detect behavioral degradation (loops, thrashing, frame collapse).
- **Real-time Feedback:** Detections are injected directly into the agent's sensory stream as tool-layer messages (e.g., "⚠️ ANALYSIS PARALYSIS: You have called 'read' 10 times in a row without acting.")
- **Pressure Over Compensation:** Instead of the human manually breaking the loop, the system applies "pressure" by forcing the agent to confront its own thrashing.

**What this rejects:**

- **Silent Failure:** Agents continuing to stall or thrash without awareness.
- **Human Babysitting:** Relying on the human to notice, kill, and restart a stuck agent.
- **Passive Logging:** Capturing metrics for dashboards without feeding them back to the actor.

**Why this matters:** Knowing correct behavior (from skills) isn't enough to prevent incorrect behavior (Track Actions). When an agent's internal monologue enters a loop, it loses the external perspective needed to break it. By injecting "Pain" into the context, we restore that external perspective structurally.
