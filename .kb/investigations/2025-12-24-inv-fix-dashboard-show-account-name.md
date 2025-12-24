## Summary (D.E.K.N.)

**Delta:** Dashboard now displays account name (e.g., "personal", "work") from ~/.orch/accounts.yaml instead of email prefix.

**Evidence:** Go build passes, all tests pass, API endpoint updated, UI updated with graceful fallback.

**Knowledge:** Account lookup by email in accounts.yaml provides cleaner, more meaningful display. Fallback to email prefix ensures backward compatibility.

**Next:** Close - implementation complete, ready for server restart to verify.

**Confidence:** High (90%) - straightforward implementation, tests pass, fallback handles edge cases.

---

# Investigation: Fix Dashboard Show Account Name

**Question:** How to display account name (personal/work) instead of email prefix in the dashboard usage display?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: API returns email, not account name

**Evidence:** The `/api/usage` endpoint in `cmd/orch/serve.go:646` returns `info.Email` from the usage package. The UI displays `$usage.account.split('@')[0]` which gives ambiguous results like "dylan" instead of "personal" or "work".

**Source:** `cmd/orch/serve.go:659`, `web/src/routes/+page.svelte:383`

**Significance:** Need to lookup the account name from accounts.yaml by matching email to find the human-readable account name.

---

### Finding 2: accounts.yaml has the account name → email mapping

**Evidence:** The `~/.orch/accounts.yaml` file structure maps account names (like "personal", "work") to their email addresses.

**Source:** `~/.orch/accounts.yaml`, `pkg/account/account.go:60-64` (Config struct)

**Significance:** We can reverse-lookup the account name by iterating accounts and matching email.

---

### Finding 3: UI should gracefully fallback

**Evidence:** If account_name is not found (e.g., email not in accounts.yaml), the UI should still work by falling back to the email prefix.

**Source:** Design decision based on robustness requirements

**Significance:** Ensures backward compatibility and handles edge cases where account may not be configured.

---

## Implementation Done

### API Changes (cmd/orch/serve.go)

1. Added `AccountName` field to `UsageAPIResponse` struct
2. Created `lookupAccountName(email)` function that:
   - Loads config from `~/.orch/accounts.yaml`
   - Iterates accounts to find matching email
   - Returns account name if found, empty string otherwise
3. Modified `handleUsage()` to call `lookupAccountName(info.Email)`

### TypeScript Changes (web/src/lib/stores/usage.ts)

1. Added optional `account_name?: string` field to `UsageInfo` interface

### UI Changes (web/src/routes/+page.svelte)

1. Updated display to use `$usage.account_name || $usage.account.split('@')[0]`
2. Shows account_name if available, falls back to email prefix

---

## Verification

- Go build: PASS
- All Go tests: PASS
- TypeScript check: Pre-existing error in agent-detail-panel.svelte (unrelated to this change)
- Binary installed to ~/bin/orch

---

## References

**Files Modified:**
- `cmd/orch/serve.go` - Added account_name field and lookup function
- `web/src/lib/stores/usage.ts` - Added account_name to TypeScript interface
- `web/src/routes/+page.svelte` - Updated display logic

**Commands Run:**
```bash
go build ./cmd/orch/...
go test ./...
make install
curl http://127.0.0.1:3348/api/usage | jq .
```
