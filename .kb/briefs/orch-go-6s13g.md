# Brief: orch-go-6s13g

## Frame

The named-incompleteness model makes a spatial claim: open questions are "coordinates in possibility space" that converge, while conclusions scatter. The previous probe tested this with TF-IDF on orch-go's own briefs and found the opposite — findings clustered tighter. But TF-IDF measures word overlap, not meaning. The cross-domain probe identified an untested experiment: nobody has measured whether academic papers' open questions cluster tighter than their findings in semantic embedding space. This investigation built and ran that experiment.

## Resolution

I fetched 50 RAG papers from arXiv, extracted questions and findings from abstracts, and embedded both sets with sentence-transformers. The result reversed the TF-IDF finding: questions cluster ~15% tighter (d=0.27), consistent across two embedding models. The moment the direction flipped from TF-IDF was the click — the model's prediction about "reference vs resemblance" played out exactly. Gaps compose through meaning, not through shared vocabulary.

But the honest statistical test (permutation, not t-test) says this isn't significant at N=50. The parametric tests said p < 0.001, which felt like a triumph — until I realized they treated 1225 pairwise similarities as independent when they come from only 50 papers. The permutation test, which correctly shuffles at the paper level, gives p=0.196. Power analysis says I need 240 papers for 80% power. The protocol works; it just needs more data.

## Tension

The trend is there but the evidence isn't. This is exactly the situation the model predicts should be generative: a named gap (is the effect real at N=240?) specific enough to converge on, with a validated protocol ready to test it. But running the full study is a resource decision — 240 papers × Claude extraction × full-text processing. Is this worth pursuing as a publication, or does it serve better as a calibration probe that already told us what we needed (sentence embeddings detect what TF-IDF can't)?
