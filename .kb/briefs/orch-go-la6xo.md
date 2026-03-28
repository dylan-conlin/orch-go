# Brief: orch-go-la6xo

## Frame

You promoted a thread called "Generative systems are organized around named incompleteness" and the model directory came out as `generative-systems-are-organized-around` — the concept itself got sliced off. The slug truncation was doing exactly what it was designed to do (first 5 words), but thread titles are sentences, and the meaning of a sentence isn't usually in the first five words.

## Resolution

The fix is a `--name` flag on `orch thread promote`. When you know what the model should be called — and with promote, you always do — you pass `--name named-incompleteness` and the directory gets that name instead of the slug. No changes to Slugify itself, because Slugify works fine for what it was built for (thread filenames where truncation is cosmetic). The mismatch was using it for a purpose where the name carries meaning.

## Tension

This makes promote require one more flag for long titles. The alternative would have been making Slugify "smarter" — maybe taking the last N words, or some noun-phrase extraction — but that's a heuristic fighting against the fact that the caller already knows the answer. Worth asking: should `--name` be required (not optional) for promote, since model/decision names always matter?
