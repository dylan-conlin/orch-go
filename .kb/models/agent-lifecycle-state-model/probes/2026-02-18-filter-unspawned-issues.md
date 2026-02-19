Question
Does the beads-first dashboard filter exclude unspawned issues so only issues with agent evidence appear as agents?

What I Tested

- `curl -sk https://localhost:3348/api/agents | python3 -c "import json,sys; data=json.load(sys.stdin); print('agents', len(data)); print('dead', sum(1 for a in data if a.get('status')=='dead'))"`
- `go run -ldflags "-X main.sourceDir=/Users/dylanconlin/Documents/personal/orch-go" ./cmd/orch serve --port 3349 >/tmp/orch-serve-1074.log 2>&1 & pid=$!; sleep 4; curl -sk https://localhost:3349/api/agents | python3 -c "import json,sys; data=json.load(sys.stdin); print('agents', len(data)); print('dead', sum(1 for a in data if a.get('status')=='dead'))"; kill $pid`

What I Observed

- Existing server on 3348 returned `agents 283` and `dead 240` before the fix binary was running.
- Fresh `go run` server on 3349 (with sourceDir set for TLS certs) returned `agents 25` and `dead 3` after applying the filter.

Model Impact

- Confirms that filtering beads-first issues by agent evidence (workspace/session/daemon labels) suppresses unspawned issues from the dashboard list, reducing dead-agent noise.

Status: Complete
