## Summary (D.E.K.N.)

**Delta:** Upgraded orch serve from HTTP/1.1 to HTTP/2 with TLS to permanently fix browser connection pool exhaustion (6 connections per origin limit).

**Evidence:** Build succeeds (`make build`), go vet passes, all frontend stores updated to https://localhost:3348.

**Knowledge:** Go 1.24's http.ListenAndServeTLS auto-enables HTTP/2; browsers only support HTTP/2 over TLS; self-signed certs work for localhost.

**Next:** Install binary, verify HTTP/2 in browser network panel, verify SSE streams connect simultaneously.

---

# Investigation: HTTP/2 TLS Daemon Server Upgrade

**Question:** How to upgrade orch serve from HTTP/1.1 to HTTP/2 with TLS to fix connection pool exhaustion?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** Orchestrator verification of HTTP/2 in browser
**Status:** Complete

---

## Findings

### Finding 1: Self-signed TLS certificate generated with openssl

**Evidence:** 
- Created `pkg/certs/cert.pem` and `pkg/certs/key.pem`
- Used 4096-bit RSA key with 10-year validity
- Includes both localhost DNS and 127.0.0.1 IP in Subject Alt Names

**Source:** 
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 3650 -nodes -subj "/CN=localhost" -addext "subjectAltName = DNS:localhost, IP:127.0.0.1"
```

**Significance:** Self-signed cert is acceptable for localhost development tooling. Browser will show a warning once, then work normally after exception is added.

---

### Finding 2: Server code updated to use TLS

**Evidence:**
- Changed `cmd/orch/serve.go:303` from `http.ListenAndServe` to `http.ListenAndServeTLS`
- Added `crypto/tls` import and `tlsConfigSkipVerify()` helper
- Updated CORS to accept https origins
- Updated status check to use https with InsecureSkipVerify for self-signed cert

**Source:** `cmd/orch/serve.go:281-307`

**Significance:** Go's HTTP/2 is automatically enabled when using TLS. No application code changes needed for SSE handlers.

---

### Finding 3: All frontend stores updated to HTTPS

**Evidence:** Updated API_BASE in 11 store files:
- agents.ts, agentlog.ts, beads.ts, config.ts, daemon.ts
- focus.ts, hotspot.ts, orchestrator-sessions.ts, pending-reviews.ts
- servers.ts, usage.ts

**Source:** `web/src/lib/stores/*.ts`

**Significance:** Frontend will now connect over HTTPS, which browsers auto-negotiate to HTTP/2.

---

## Synthesis

**Key Insights:**

1. **HTTP/2 requires TLS for browsers** - All modern browsers only support HTTP/2 over TLS (h2), not cleartext (h2c).

2. **Go makes HTTP/2 transparent** - Simply switching from ListenAndServe to ListenAndServeTLS enables HTTP/2 automatically.

3. **Self-signed certs are fine for localhost** - This is development tooling, not user-facing, so a one-time browser warning is acceptable.

**Answer to Investigation Question:**

Successfully upgraded by: (1) generating self-signed TLS cert with openssl, (2) changing server to use ListenAndServeTLS, (3) updating all frontend API_BASE to https. HTTP/2 is now automatic.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build succeeds (`make build` completes without errors)
- ✅ Go vet passes (`go vet ./cmd/orch/` has no issues)
- ✅ Binary runs (`./build/orch version` works)

**What's untested:**

- ⚠️ HTTP/2 protocol verification in browser network panel (needs manual test)
- ⚠️ Both SSE streams connecting simultaneously (needs runtime test)
- ⚠️ Browser self-signed cert acceptance flow (needs manual verification)

**What would change this:**

- If Go's HTTP/2 implementation has issues with SSE streaming
- If TLS handshake adds unacceptable latency
- If self-signed cert causes problems with Playwright tests

---

## Implementation Recommendations

### Recommended Approach ⭐

**HTTP/2 with TLS** - Implemented as specified in architect investigation.

**Why this approach:**
- Eliminates connection limit constraint at protocol level
- No SSE handler changes needed
- Solves the problem class, not just current symptom

**Trade-offs accepted:**
- Browser shows self-signed cert warning (one-time)
- Tests may need certificate handling

### Implementation Details

**What was implemented:**
1. TLS certificate in `pkg/certs/` directory
2. Server code change in `cmd/orch/serve.go`
3. Frontend updates in `web/src/lib/stores/*.ts`

**Things to watch out for:**
- ⚠️ Browser must accept self-signed cert to see dashboard
- ⚠️ Load tests (web/tests/load-test.spec.ts) updated to use https

**Success criteria:**
- ✅ Dashboard loads at https://localhost:5188
- ✅ Network panel shows "h2" protocol
- ✅ Both SSE streams connect without pending requests
- ✅ No more connection pool exhaustion

---

## References

**Files Modified:**
- `cmd/orch/serve.go` - Server TLS configuration
- `web/src/lib/stores/*.ts` - All frontend API stores
- `web/src/lib/services/sse-connection.ts` - Example URL in docs
- `web/tests/load-test.spec.ts` - Test API URL
- `pkg/certs/cert.pem` - TLS certificate (new)
- `pkg/certs/key.pem` - TLS private key (new)

**Commands Run:**
```bash
# Generate TLS cert
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -sha256 -days 3650 -nodes -subj "/CN=localhost" -addext "subjectAltName = DNS:localhost, IP:127.0.0.1"

# Build binary
make build

# Verify build
go vet ./cmd/orch/
./build/orch version
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-05-design-permanent-fix-http-connection-pool.md` - Architect design
- **Constraint:** Dashboard SSE connections can exhaust HTTP/1.1 browser connection pool

---

## Investigation History

**2026-01-06 07:30:** Investigation started
- Task: Upgrade orch serve to HTTP/2 with TLS per architect investigation

**2026-01-06 07:38:** TLS certificate generated
- Created pkg/certs/ with cert.pem and key.pem

**2026-01-06 07:45:** Server code updated
- Changed ListenAndServe to ListenAndServeTLS
- Added TLS imports and config

**2026-01-06 07:50:** Frontend stores updated
- Changed all API_BASE from http to https

**2026-01-06 07:55:** Investigation completed
- Status: Complete
- Key outcome: HTTP/2 with TLS implemented, ready for verification
