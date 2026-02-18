# Probe: State.create rejected Promise caching

**Status:** Complete

## Question

Does `State.create` cache rejected Promise results, preventing retries for the same (directory, init) pair?

## What I Tested

- `bun -e "import { State } from './packages/opencode/src/project/state.ts'; let attempts = 0; const createState = State.create(() => 'root', () => { attempts += 1; return Promise.reject(new Error('fail')); }); const p1 = createState(); p1.catch(async () => { const p2 = createState(); p2.catch(() => { console.log(JSON.stringify({ attempts, samePromise: p1 === p2 })); }); });"` (from `~/Documents/personal/opencode`)
- Re-ran the same command after clearing cache-on-reject fix in `project/state.ts`

## What I Observed

- Output: `{ "attempts": 1, "samePromise": true }`
- The second call reused the same rejected Promise and did not re-run init.
- After fix: log warnings for rejection + output `{ "attempts": 2, "samePromise": false }`
- The second call re-ran init and returned a new Promise after rejection.

## Model Impact

- Confirms the original behavior (cached rejected Promise) and that clearing the cache on rejection enables retry.
