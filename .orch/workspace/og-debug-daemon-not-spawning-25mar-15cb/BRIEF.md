# Brief: orch-go-e4uiq

## Frame

The daemon showed 0 spawnable issues despite kxtrd sitting in the queue with triage:ready, open status, and type=investigation. It wasn't in the rejected list either — it had vanished from the preview entirely. The question was: where does an issue go when it's neither spawnable nor rejected?

## Resolution

kxtrd was being classified as a "test-writing task" and deferred behind implementation siblings. The heuristic (`isTestLikeIssue`) scans the full title + description for patterns like `"testing "`, and kxtrd's description says "property-based testing as agent verification layer." That single match — a research topic containing the word "testing" — caused the investigation to be treated as if it were a task to write unit tests. All three triage:ready orch-go issues had similar false positives, which is why the queue showed zero spawnable.

The fix exempts investigation and question types from test deferral entirely. These issue types produce knowledge artifacts, not code — deferring them behind implementation siblings is structurally meaningless. A feature issue titled "Add tests for auth" still gets deferred correctly. Separately, Preview now counts all compliant issues (was binary 0/1), so the display accurately reflects queue depth.

## Tension

The test-deferral heuristic searches full descriptions, which means any issue that *discusses* testing will match. The type-based exemption handles investigations and questions, but a feature or task that mentions testing in its context (not its goal) could still false-positive. The pattern `"testing "` with trailing space is quite broad — it's not clear whether tightening it would create worse false negatives than the false positives it prevents.
