<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Auto-switch silently invalidates active account tokens when refresh token comparison fails due to external rotation.

**Evidence:** Code trace shows `GetAccountCapacity` rotates tokens even when `isActiveAccount` check is false, leaving auth.json with stale tokens.

**Knowledge:** The `isActiveAccount` check at line 527 compares tokens that may have drifted if OpenCode rotated tokens externally; when they don't match, token rotation still happens but auth.json isn't updated.

**Next:** Fix by always updating auth.json when the account being checked is the active one (detect by matching after rotation, not before).

**Confidence:** High (85%) - Code analysis complete, but need runtime test to confirm the specific trigger scenario.

---

# Investigation: Auto-Switch Account Failing Silently

**Question:** Why does auto-switch account fail silently when refresh tokens have been rotated externally?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Agent (orch-go-bwrm)
**Phase:** Complete
**Next Step:** Implement fix in GetAccountCapacity
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Token comparison uses stale data

**Evidence:** In `GetAccountCapacity` at pkg/account/account.go:527:
```go
isActiveAccount := authErr == nil && currentAuth.Anthropic.Refresh == acc.RefreshToken
```
This compares `currentAuth.Anthropic.Refresh` (from auth.json) with `acc.RefreshToken` (from accounts.yaml). If OpenCode or another process rotated the token, these will be out of sync.

**Source:** pkg/account/account.go:526-527

**Significance:** When tokens don't match, `isActiveAccount` is false even for the actual active account. This causes the auth.json update (lines 545-551) to be skipped.

---

### Finding 2: Token rotation happens unconditionally

**Evidence:** At line 530, `RefreshOAuthToken(acc.RefreshToken)` is called regardless of `isActiveAccount` status:
```go
tokenInfo, err := RefreshOAuthToken(acc.RefreshToken)
```
This rotates the refresh token on Anthropic's servers, invalidating the old token.

**Source:** pkg/account/account.go:530

**Significance:** If the active account's token was out of sync AND `isActiveAccount` was false, rotating the token makes the token in auth.json completely invalid (since it's neither the old token nor the new token).

---

### Finding 3: Silent failure in ShouldAutoSwitch loop

**Evidence:** In `ShouldAutoSwitch` at line 862-863:
```go
capacity, err := GetAccountCapacity(name)
if err != nil {
    continue
}
```
Errors from `GetAccountCapacity` cause silent `continue` to next account.

**Source:** pkg/account/account.go:861-863

**Significance:** If token rotation fails (due to stale token), the error is silently swallowed and the account is skipped. Combined with Finding 2, this means the active account can have its token invalidated silently.

---

### Finding 4: currentName may be empty, causing active account to be checked

**Evidence:** When email matching fails (lines 816-824) and refresh token matching fails (lines 831-837) due to token drift, `currentName` remains empty. This causes the active account to NOT be skipped in the loop (line 857):
```go
if name == currentName {
    continue
}
```

**Source:** pkg/account/account.go:816-838, 857

**Significance:** The active account is supposed to be skipped in the alternate accounts loop, but when `currentName` is empty (due to token mismatch), it gets processed, triggering the token rotation bug from Finding 2.

---

## Synthesis

**Key Insights:**

1. **Token drift causes cascading failures** - When OpenCode rotates tokens externally, the mismatch between auth.json and accounts.yaml causes `isActiveAccount` to be false, which then causes auth.json to NOT be updated after rotation, leaving it with an invalid token.

2. **No defensive check after rotation** - The code checks if the account is active BEFORE rotation, but not AFTER. After rotation, we could re-verify and update auth.json if needed.

3. **The failure is silent** - Errors from `GetAccountCapacity` are swallowed with `continue`, and the invalid token in auth.json isn't discovered until the next API call fails.

**Answer to Investigation Question:**

Auto-switch fails silently because:
1. External token rotation (by OpenCode) causes mismatch between auth.json and accounts.yaml
2. This mismatch makes `isActiveAccount` check return false even for the active account
3. When `GetAccountCapacity` is called on the (unrecognized) active account, it rotates the token
4. Since `isActiveAccount` was false, auth.json is NOT updated with the new token
5. The old token in auth.json is now completely invalid (Anthropic invalidated it during rotation)
6. Future API calls with the stale auth.json token fail silently

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Code analysis clearly shows the token comparison logic and the conditional auth.json update. The failure path is clear from code inspection.

**What's certain:**

- ✅ The `isActiveAccount` check compares tokens that can drift (line 527)
- ✅ Token rotation happens unconditionally (line 530)
- ✅ auth.json is only updated when `isActiveAccount` is true (line 545-551)
- ✅ Errors from GetAccountCapacity are silently swallowed (line 862-863)

**What's uncertain:**

- ⚠️ Exact trigger conditions (when does OpenCode rotate tokens externally?)
- ⚠️ Whether email matching (line 819) provides a reliable fallback
- ⚠️ Real-world frequency of this failure mode

**What would increase confidence to Very High:**

- Runtime test that reproduces the exact failure scenario
- Log analysis showing token mismatch errors
- Confirmation that OpenCode rotates tokens independently of orch-go

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Update auth.json when account matches AFTER rotation** - After rotating tokens, re-check if the account is the active one by comparing the NEW refresh token with auth.json, and update if needed.

**Why this approach:**
- Directly addresses the root cause (stale `isActiveAccount` check)
- Minimal code change
- Works regardless of how tokens got out of sync

**Trade-offs accepted:**
- Slightly more complex logic in GetAccountCapacity
- Extra auth.json read after rotation (negligible cost)

**Implementation sequence:**
1. After `RefreshOAuthToken` returns successfully (line 530-533)
2. Re-load auth.json and check if `auth.Anthropic.Refresh` matches EITHER the old token (acc.RefreshToken from before rotation) OR was already the active account
3. If match, update auth.json with new tokens

### Alternative Approaches Considered

**Option B: Pre-rotation sync check**
- **Pros:** Prevents rotation of potentially conflicting tokens
- **Cons:** Doesn't fix existing out-of-sync state; adds delay before rotation
- **When to use instead:** If we want to be more conservative about token rotation

**Option C: Always update auth.json for all accounts**
- **Pros:** Simpler logic - always update
- **Cons:** Would overwrite active account's tokens with non-active account's tokens, breaking active sessions
- **When to use instead:** Never - this would cause immediate failures

**Rationale for recommendation:** Option A directly fixes the root cause with minimal risk. The post-rotation check is more reliable than the pre-rotation check because it catches the case where tokens drifted.

---

### Implementation Details

**What to implement first:**
- Add post-rotation check in `GetAccountCapacity` after line 538
- Check if the account WE just processed is the one currently in auth.json
- Use the OLD token (before rotation) for comparison since that's what auth.json has

**Things to watch out for:**
- ⚠️ The comparison should use the token BEFORE rotation (save it before line 530)
- ⚠️ Race condition: another process could update auth.json between check and write
- ⚠️ Test with multiple accounts to ensure we don't overwrite the wrong one

**Areas needing further investigation:**
- When does OpenCode rotate tokens on its own?
- Should we add logging for when token drift is detected?
- Should we add a "force sync" command?

**Success criteria:**
- ✅ Auto-switch works even after external token rotation
- ✅ Active sessions are not disrupted by capacity checks
- ✅ Add test case for this scenario

---

## References

**Files Examined:**
- pkg/account/account.go - Token rotation and auto-switch logic
- cmd/orch/main.go:961-1022 - checkAndAutoSwitchAccount caller

**Commands Run:**
```bash
# Get issue context
bd show orch-go-bwrm

# Search for AutoSwitch usage
grep -n "AutoSwitch" pkg/account/account.go
```

**Related Artifacts:**
- **Decision:** (pending) - Fix approach for token drift in auto-switch

---

## Investigation History

**2025-12-24 12:15:** Investigation started
- Initial question: Why does auto-switch fail silently?
- Context: Issue title mentions refresh token as cause

**2025-12-24 12:30:** Root cause identified
- Found token comparison at line 527 uses potentially stale data
- Traced through failure path when `isActiveAccount` is false

**2025-12-24 12:45:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Token drift causes `isActiveAccount` check to fail, leading to auth.json being left with stale tokens after rotation
