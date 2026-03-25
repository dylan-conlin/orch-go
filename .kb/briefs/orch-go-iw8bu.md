# Brief: orch-go-iw8bu

## Frame

Marking briefs as read in the dashboard worked once, then silently stopped persisting across server restarts. The persistence code was correct — the regression came from somewhere else entirely.

## Resolution

The real `~/.orch/briefs-read-state.json` was being overwritten by test runs. Three integration tests added with the original orch-go-1vut9 fix called `saveBriefReadState()` without isolating their HOME directory, so each `go test` run replaced the real state file with temp-directory-keyed test data. On next server restart, `loadBriefReadState()` loaded garbage keys that matched nothing. The fix was three lines: `t.Setenv("HOME", t.TempDir())` in each test. The pattern was already established in the unit tests — the integration tests just missed it.

## Tension

This is a class of bug where tests silently corrupt real user state. The only reason it was caught is that the state loss was visible in the UI. If other tests write to paths under `os.UserHomeDir()` without HOME isolation, the same pattern could be corrupting state elsewhere without anyone noticing.
