# Brief: orch-go-im7tf

## Frame

Running `orch compose` against 77 briefs produced clusters named "checks / claude / dead" with one mega-cluster swallowing 24 briefs, and the same 2-3 threads matching everything with scores of 40+. The compose pipeline was generating output, but none of it was useful for actually discovering patterns.

## Resolution

Three independent root causes, three fixes. The naming was keyword soup because all shared keywords scored equally — high frequency within the cluster doesn't mean distinctive across clusters. Adding inverse cluster frequency scoring with within-cluster frequency as tiebreaker turned "checks / claude / dead" into "boundary / core / artifact." The mega-clusters formed because the document frequency filter (20%) was too permissive and 3-keyword overlap was too low a bar — tightening to 15% and requiring 4-keyword overlap shrank the max cluster from 24 to 8. Thread matching was the most revealing: the code unioned all keywords from all cluster members (creating bags of 100+ words) and matched against thread body content (500+ words per thread). Any large thread matched everything. Switching to shared-keywords-only vs title-keywords-only, with weighted scoring, dropped match scores from 47 to 3-6, and now different clusters match different threads.

The turn: I initially tried only the DF tightening (0.20→0.15) and it barely moved the needle — 24-brief cluster shrank to 19. The MinKeywordOverlap bump from 3 to 4 was what actually broke the mega-clusters. Both were needed because they target different failure modes: DF prevents common words from surviving; overlap prevents loose connections from clustering.

## Tension

The fix increased unclustered briefs from 7 to 16 (21% of corpus). That's more honest about genuine groupings, but it also means a fifth of work products get no structural home. Is there a value in a fallback pass — maybe cluster unclustered briefs at a lower threshold, explicitly labeled as "weak" clusters? Or is silence better than noise?
