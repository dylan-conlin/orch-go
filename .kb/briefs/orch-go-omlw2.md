# Brief: orch-go-omlw2

## Frame

The daemon's completion pipeline has three paths: auto-complete-light (effort:small), auto-complete (scan/auto tier), and label-ready-review (everything else). Headless brief generation — the thing that pre-writes `.kb/briefs/` so Dylan arrives to readable summaries instead of raw completions — was only wired into the label-ready-review path. Investigations and small fixes, which are the bulk of daemon-completed work, silently skipped brief generation.

## Resolution

Two lines added: `d.fireHeadlessCompletion()` in the auto-complete and auto-complete-light branches of `ExecuteCompletionRoute`. The method already existed and worked correctly — it just wasn't called from those paths. The TDD cycle was clean: wrote two tests that timed out waiting for `CompleteHeadless` to be called, added the two calls, both tests passed, full suite green.

## Tension

This bug existed since the auto-complete paths were introduced but was invisible because the comprehension queue (`comprehension:unread` label) still worked — Dylan could still find and review completed work. The briefs just weren't there waiting for him. Worth asking: are there other fire-and-forget side effects in `labelReadyReview` that the auto-complete paths should also be running? I checked and `recordUnverifiedCompletion` is correctly *not* called from auto-complete (those don't need human verification), but the pattern of "add a new thing to one path, forget the others" seems like a recurring risk in this switch statement.
