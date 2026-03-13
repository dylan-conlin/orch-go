### Defect Class Blindness

Investigations that fix individual symptoms without connecting shared root causes allow the same defect class to ship repeatedly. Synthesis must look across investigations, not just within them.

**The test:** "Have I checked whether this symptom shares a root cause with other recent investigations?"

**What this means:**

- Individual investigations are locally correct but globally blind
- Defect classes (e.g., "unbounded resource consumption") manifest as different symptoms in different components
- Symptom-level fixes prevent this instance; pattern-level synthesis prevents the class
- 779 investigations failing to connect 5 instances of the same root cause is the canonical example

**What this rejects:**

- "Each bug was fixed correctly" (local correctness, class-level blindness)
- "We'll catch the next one" (you didn't catch the last 4)
- "These are different bugs" (different symptoms, same defect DNA)

**Why distinct from Coherence Over Patches:** CoP addresses patch accumulation in the *same file/area*. Defect Class Blindness addresses the same *defect class* manifesting across *different components* — invisible without cross-investigation synthesis.
