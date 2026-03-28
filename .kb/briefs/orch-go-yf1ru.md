# Brief: orch-go-yf1ru

## Frame

The pilot study tested whether the Named Incompleteness model's core claim — that questions have natural geometric structure while findings don't — could be measured in real academic papers. Fifty papers said "probably yes, but we can't be sure." The power analysis said we needed 240 to know. This session ran the full study.

## Resolution

373 RAG papers from arXiv. Extract their open questions and findings. Embed both sets with sentence-transformers. Shuffle the labels 5,000 times and ask: could this pattern happen by chance?

The answer is no. Permutation p=0.0086. Questions cluster significantly tighter than findings. The 95% confidence interval doesn't touch zero. A second, completely different embedding model reproduces the finding (p=0.035). Truncating findings to the same length as questions makes the effect *stronger*, not weaker — this isn't a text-length trick.

The surprise was that the pilot overestimated the effect. d=0.27 in the pilot became d=0.20 at scale — a 26% shrinkage, which turns out to be exactly what statisticians expect from small-sample pilots. This means the power analysis was slightly optimistic: N=240 would have given 68% power, not 80%. Good thing we got 373 papers instead of stopping at 240.

What this means for NI: the model has now produced and confirmed one quantitative prediction. Questions really do occupy a tighter region of meaning-space than findings. The information-theoretic argument (gaps define constraint surfaces, conclusions are underdetermined points) has measurable consequences in real text.

## Tension

The effect is real but small (d=0.20). We've only tested RAG papers — one subfield of computer science. If questions cluster tighter in RAG papers but not in, say, biomedical NLP or cognitive science, then the effect might be domain-specific rather than substrate-general, which would undercut NI-03's cross-domain claim. The pipeline is ready for cross-subfield replication, but whether that's worth the time depends on whether this is heading toward a publication or just model validation.
