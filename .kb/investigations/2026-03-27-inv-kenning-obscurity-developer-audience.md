## Summary (D.E.K.N.)

**Delta:** "Kenning" sits in the same obscurity band as Ansible, Pulumi, and Istio — all successful devtools. No evidence that name obscurity kills developer tools; what kills adoption is being unpronounceable, unspellable, or unsearchable. Kenning scores well on all three. The real risk isn't obscurity — it's whether the Old Norse poetry connection reads as pretentious vs. intriguing to the specific audience (indie dev/AI infra practitioners).

**Evidence:** Analyzed 12 successful devtools with obscure names (Kafka, Kubernetes, Ansible, Pulumi, Istio, Bazel, etc.) plus naming critiques from larr.net and ntietz.com HN discussions. No tool was found where name obscurity was cited as a primary reason for failure.

**Knowledge:** The developer naming consensus is pragmatic: names need to be pronounceable, spellable, short, memorable, and searchable — NOT immediately understood. Obscure names actually outperform descriptive names for public-facing products because they're more ownable, more searchable, and create free word-of-mouth ("what does that mean?").

**Next:** Design a lightweight first-contact test (5-10 developers, async, <1 hour to run).

**Authority:** strategic — Product naming is Dylan's call. This provides evidence to inform it.

---

# Investigation: Is "Kenning" Too Obscure for the Target Developer Audience?

**Question:** Does the obscurity of "kenning" create adoption friction that outweighs its semantic and branding advantages?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** orch-go-k25gf
**Phase:** Complete
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-27-inv-explore-public-naming-candidates-naming.md` | extends | yes | - |
| `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` | extends | yes | - |

---

## Findings

### Finding 1: Name Obscurity Does Not Kill Developer Tools

12 successful devtools with non-obvious names, ranked by obscurity:

| Tool | Name Origin | Obscurity | Outcome |
|------|------------|-----------|---------|
| Pulumi | Hawaiian for "broom" (mentor tribute) | Very high | VC-funded, growing |
| Istio | Greek for "sail" | Very high | Service mesh standard |
| Bazel | Anagram of "Blaze" | High | Google ecosystem staple |
| Kubernetes | Greek for "helmsman" | High | Industry dominant (despite spawning k8s to cope with the name) |
| Kafka | Named after Franz Kafka | Medium-high | Industry standard for streaming |
| Ansible | Coined by Ursula K. Le Guin (1966 sci-fi) | Medium | Acquired by Red Hat |
| Nix | Dutch/German "nothing" + Unix echo | Medium | Devoted niche community |
| Terraform | Sci-fi term (1942) | Low-medium | IaC standard |
| Deno | Anagram of "Node" | Low | Growing |
| Svelte | English/French "slender" | Low | High and growing |
| Zig | Short, punchy, no etymology | Very low | Growing niche |
| Bun | Ordinary English word | Very low | Growing rapidly |

**No tool was found where name obscurity was the primary reason for failure.** Tools fail for DX, features, timing, and community — not names.

Kubernetes is the clearest case of name friction (spawned k8s, "how to pronounce" articles). It succeeded anyway because it had Google backing and no real competitor. The friction was real but not fatal.

**Where "kenning" sits:** Roughly Ansible-level obscurity. Less obscure than Pulumi, Istio, or Bazel. More obscure than Terraform or Svelte. Comfortably in the range where successful devtools live.

---

### Finding 2: What Actually Matters in a Developer Tool Name

The consensus from naming debates (larr.net, ntietz.com/HN, marketing literature):

| Criterion | Kenning Score | Notes |
|-----------|--------------|-------|
| Pronounceable | **Strong** | One obvious pronunciation: KEN-ing. Zero ambiguity. Better than Kubernetes, comparable to Ansible. |
| Spellable | **Strong** | 7 letters, phonetically regular. No "is it -etes or -etis?" problem. |
| Short | **Strong** | 2 syllables. Better than Kubernetes (5), Terraform (3), Ansible (3). |
| Memorable | **Strong** | Unusual enough to stick, not so alien it slides off. |
| Searchable | **Good** | "Kenning" currently returns literary/poetry results. A tool would quickly dominate (low competition). Much better than "Go" (unsearchable without "golang"). |
| Has a story | **Strong** | "It's a compound metaphor from Norse poetry — whale-road for sea, sky-candle for sun. We compose simpler concepts into richer understanding too." |

The key finding: **public-facing product names benefit from being non-descriptive.** Descriptive names are better for internal services (discoverability within orgs). For products, a distinctive name is more ownable, more searchable, and more brandable.

---

### Finding 3: The "Ken" Bridge Reduces Effective Obscurity

"Kenning" isn't floating in a vacuum. The root word "ken" is moderately well-known in English:
- "Beyond my ken" (range of knowledge/understanding) — common idiom
- "Ken" as Scottish/Northern English for "to know" — culturally familiar
- The "-ing" suffix makes it parse as an activity related to knowing

A developer encountering "Kenning" for the first time will likely think "something to do with knowing/understanding" before they learn the Old Norse poetry connection. This is a much better first-contact experience than Pulumi ("...broom?") or Bazel ("...anagram of what?").

The full story — compound metaphors, whale-road for sea — is the *second* layer that sticks once learned. The first layer already works.

---

### Finding 4: The Real Risk Is Tone, Not Recognition

The obscurity question is a proxy for the actual concern: **does "kenning" read as pretentious to the target audience?**

Two reception scenarios:

**Scenario A (curiosity):** "What's a kenning?" → reads the tagline → "Oh, compound metaphors from Norse poetry, that's cool" → the name becomes a conversation piece. This is the Kafka/Ansible pattern — the origin story becomes free marketing.

**Scenario B (eye-roll):** "What's a kenning?" → reads the tagline → "So they named their dev tool after an obscure literary device to seem smart" → mild negative first impression that the product has to overcome.

Which scenario plays out depends on:
- **Audience composition:** Indie devs and knowledge workers → more likely Scenario A. Enterprise platform engineers → more likely neutral (they tolerate Kubernetes, Istio, Pulumi without complaint).
- **How the story is told:** "We named it after Old Norse poetry" (pretentious). vs. "A kenning is a compound metaphor — whale-road for sea, sky-candle for sun. The tool composes simpler observations into understanding." (shows, doesn't tell).
- **Whether the product delivers:** A good product retroactively makes any name feel right. Nobody thinks Kafka is a bad name because the tool works.

**Assessment:** The pretension risk is real but manageable. It's mitigated by (a) leading with the concrete metaphor examples rather than "Old Norse," (b) the product genuinely doing what the name describes, and (c) the indie/knowledge-worker audience being more receptive to literary references than, say, enterprise IT buyers.

---

### Finding 5: A Lightweight First-Contact Test Design

The investigation that surfaced this question correctly identified that resolution requires user data. Here's a minimal test protocol:

**Method:** Show 5-10 developers the name + tagline cold. Measure three things.

**Script:**
> *You see a new developer tool called **Kenning**. Its tagline is "Durable understanding from AI agent work."*
>
> 1. What do you think this tool does? (open-ended, before any explanation)
> 2. Can you pronounce the name? Say it out loud.
> 3. [Show one-paragraph description.] Now that you know what it does — does the name fit?
> 4. Would you remember this name tomorrow? (1-5 scale)
> 5. Any reaction to the name — positive, negative, neutral?

**Criteria:**
- If >50% cannot pronounce it → name has a phonetics problem (unlikely given KEN-ing)
- If >50% guess the wrong category (e.g., "poetry tool," "NLP library") → name misdirects
- If >50% say they wouldn't remember it → name doesn't stick
- If >30% have a negative reaction ("pretentious," "trying too hard") → tone problem

**Who to ask:** The target audience is developers who use AI coding tools (Claude Code, Cursor, Copilot) and want to retain understanding from agent-generated work. Ideal test subjects: 3-5 indie devs who use AI tools daily, 2-3 senior engineers, 1-2 technical writers or DevRel people.

**Channel:** Twitter/X DM, Discord, or a brief async form. Can be completed in 15 minutes total across all respondents.

---

## Synthesis

**The short answer: "kenning" is not too obscure.**

It sits squarely in the obscurity range where successful devtools live (Ansible-level, well below Kubernetes/Pulumi/Istio). It passes every mechanical test: pronounceable, spellable, short, memorable, searchable. The "ken" root provides an intuitive bridge to "knowing/understanding" that most obscure names lack.

The deeper question — which this investigation reframes — isn't "is it too obscure?" but "does the Old Norse poetry angle land as curious or pretentious?" This is a tone question, not a recognition question, and it depends on audience and presentation. The lightweight first-contact test above can resolve it with 5-10 data points.

**Recommendation:** Proceed with Kenning as the working name. The obscurity concern does not warrant blocking or switching to a weaker candidate (Precis, Sinter). Run the first-contact test as a parallel activity — it validates reception but shouldn't gate adoption of the working name.

---

## Structured Uncertainty

**What's tested:**
- ✅ 12 successful devtools analyzed for name obscurity vs. adoption correlation
- ✅ Kenning evaluated against 6 naming criteria (pronounceable, spellable, short, memorable, searchable, story)
- ✅ "Ken" root familiarity assessed (common English idiom "beyond my ken")
- ✅ Naming consensus from developer marketing debates (larr.net, ntietz.com, HN)

**What's untested:**
- ⚠️ Actual first-contact reception from target developers (test protocol designed but not run)
- ⚠️ Whether the Old Norse poetry framing reads as curious vs. pretentious to the specific audience
- ⚠️ How the name lands in non-English-speaking developer communities
- ⚠️ Whether "kenning" evokes "Ken" (the name/doll) before "ken" (to know) for some audiences

**What would change this:**
- If first-contact test shows >30% negative tone reaction → consider Precis as fallback
- If pronunciation confusion emerges (unlikely) → the name has a phonetics problem
- If the audience is enterprise IT (not indie devs) → recalibrate tone assessment

---

## References

**Files Examined:**
- `.kb/investigations/2026-03-27-inv-explore-public-naming-candidates-naming.md` — Original naming investigation
- `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` — Product identity decision

**External Sources:**
- Wikipedia: Kenning, Merriam-Webster: kenning
- GeekWire: How did they come up with the Kubernetes name?
- LinkedIn: Why Kafka is named Kafka
- Pulumi community: Name origin
- larr.net: "Programmers lost the plot on naming tools"
- ntietz.com: "Names should be cute, not descriptive" (+ HN discussion)

---

## Investigation History

**2026-03-27:** Investigation started — assessing whether "kenning" obscurity is a blocking concern for developer adoption.
**2026-03-27:** Completed — obscurity is not blocking; reframed core risk as tone (curious vs. pretentious); designed lightweight first-contact test protocol.
