# API Development Guide

**Purpose:** Single authoritative reference for orch-go API endpoint development. Covers patterns, performance, and best practices learned from 11+ investigations (Dec 2025 - Jan 2026).

**Last verified:** Jan 6, 2026

---

## Architecture Overview

The orch-go API (`orch serve`) provides a REST API consumed by the web dashboard at `http://localhost:5188`.

```
cmd/orch/serve.go           │ Main server setup, route registration, CORS
cmd/orch/serve_agents.go    │ /api/agents, /api/events, /api/agentlog
cmd/orch/serve_beads.go     │ /api/beads, /api/beads/ready, /api/issues
cmd/orch/serve_reviews.go   │ /api/pending-reviews, /api/dismiss-review
cmd/orch/serve_system.go    │ /api/usage, /api/focus, /api/servers, /api/daemon, /api/config
cmd/orch/serve_learn.go     │ /api/gaps, /api/reflect
cmd/orch/serve_errors.go    │ /api/errors
cmd/orch/serve_changelog.go │ /api/changelog
```

---

## Core Patterns

### 1. Handler Structure

Every endpoint follows this structure:

```go
func handleEndpoint(w http.ResponseWriter, r *http.Request) {
    // 1. Method check
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // 2. Query parameters
    param := r.URL.Query().Get("param")
    
    // 3. Business logic
    result, err := getResult(param)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // 4. Response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

**Source:** All serve_*.go files follow this pattern

### 2. CORS Middleware

All handlers are wrapped with `corsHandler()`:

```go
mux.HandleFunc("/api/endpoint", corsHandler(handleEndpoint))
```

**Why:** The dashboard runs on port 5188, API on port 3348. CORS allows cross-origin requests.

**Source:** `cmd/orch/serve.go:180-202`

### 3. Response Types

Define JSON-tagged structs for all responses:

```go
type EndpointAPIResponse struct {
    Field1 string `json:"field_1"`
    Field2 int    `json:"field_2"`
}
```

**Convention:** Keep response types with their handlers for cohesion (not in a separate types file).

---

## Performance Patterns

### Critical: Avoid N+1 Query Pattern

**The Problem:** Sequential API calls scale terribly with data size.

**Example from `/api/agents` fix:**
- Before: 564 sequential beads API calls → 26 seconds
- After: 1 batch call + parallel processing → 0.35 seconds

**Root cause in code:**
```go
// BAD: N+1 pattern - each iteration makes API call
for _, workspace := range workspaces {
    comments, _ := verify.GetComments(beadsID)  // Sequential!
    if hasPhaseComplete(comments) { ... }
}

// GOOD: Batch first, then process
beadsIDs := collectBeadsIDs(workspaces)
allComments := verify.GetCommentsBatch(beadsIDs)  // Parallel with semaphore
for _, workspace := range workspaces {
    comments := allComments[beadsID]
    if hasPhaseComplete(comments) { ... }
}
```

**Related investigations:**
- `2025-12-27-inv-api-agents-endpoint-takes-19s.md` - O(N) sequential calls
- `2026-01-05-inv-pending-reviews-api-times-out.md` - N+1 beads calls

### HTTP Client Timeouts

**The Problem:** `http.DefaultClient` has no timeout, causing indefinite hangs.

**Solution:** Configure HTTP clients with timeouts:

```go
const DefaultHTTPTimeout = 10 * time.Second

client := &http.Client{
    Timeout: DefaultHTTPTimeout,
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        if len(via) >= 10 {
            return fmt.Errorf("too many redirects")
        }
        return nil
    },
}
```

**Exception:** SSE connections need redirect limits but NO timeout (long-running streams).

**Source:** `2025-12-26-inv-api-endpoint-api-agents-hangs.md`

### Parallel Processing with Semaphore

For batch operations, use bounded concurrency:

```go
const maxConcurrent = 20
sem := make(chan struct{}, maxConcurrent)
var wg sync.WaitGroup

for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        sem <- struct{}{}        // Acquire
        defer func() { <-sem }() // Release
        processItem(item)
    }(item)
}
wg.Wait()
```

**Why 20:** Prevents overwhelming backend services while maintaining good throughput.

---

## SSE (Server-Sent Events)

### Pattern for Event Streaming

```go
func handleEventsSSE(w http.ResponseWriter, r *http.Request) {
    // 1. Check flusher support
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }
    
    // 2. Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    // 3. Stream events
    for {
        select {
        case event := <-eventChan:
            fmt.Fprintf(w, "data: %s\n\n", event)
            flusher.Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

### File Polling for Events

When streaming file changes (e.g., events.jsonl), use polling:

```go
const pollInterval = 500 * time.Millisecond

ticker := time.NewTicker(pollInterval)
defer ticker.Stop()

lastOffset := fileSize()
for {
    select {
    case <-ticker.C:
        newOffset := fileSize()
        if newOffset > lastOffset {
            // Read and send new lines
            newLines := readFrom(lastOffset, newOffset)
            sendSSE(w, newLines)
            flusher.Flush()
            lastOffset = newOffset
        }
    case <-r.Context().Done():
        return
    }
}
```

**Trade-off:** Up to 500ms latency for new events, but simple and reliable.

**Source:** `2025-12-20-inv-add-api-agentlog-endpoint-serve.md`

---

## Endpoint Categories

### Data Source Endpoints

| Endpoint | Data Source | pkg/ Import |
|----------|-------------|-------------|
| `/api/agents` | OpenCode API + workspaces | `opencode`, `spawn`, `verify` |
| `/api/beads` | beads CLI | `beads` |
| `/api/usage` | Anthropic API | `usage` |
| `/api/errors` | events.jsonl | `events` |
| `/api/changelog` | git repos | (local git) |

### Computed Endpoints

| Endpoint | Computation | Cost |
|----------|-------------|------|
| `/api/pending-reviews` | Workspace scan + beads batch | Medium |
| `/api/gaps` | Event analysis | Low |
| `/api/reflect` | Read JSON file | Low |

---

## Testing Patterns

### Required Tests Per Endpoint

1. **Method validation:** Only allowed methods succeed
2. **JSON format:** Response matches expected structure
3. **Error handling:** Missing data handled gracefully

```go
func TestHandleEndpointMethodNotAllowed(t *testing.T) {
    req := httptest.NewRequest("POST", "/api/endpoint", nil)
    rec := httptest.NewRecorder()
    handleEndpoint(rec, req)
    if rec.Code != http.StatusMethodNotAllowed {
        t.Errorf("expected 405, got %d", rec.Code)
    }
}

func TestEndpointAPIResponseJSONFormat(t *testing.T) {
    resp := EndpointAPIResponse{Field1: "test"}
    data, _ := json.Marshal(resp)
    if !strings.Contains(string(data), "field_1") {
        t.Error("expected snake_case JSON")
    }
}
```

**Source:** `cmd/orch/serve_test.go` - 1016 lines of tests

---

## Adding a New Endpoint

### Checklist

1. **Choose appropriate file** (serve_agents.go, serve_beads.go, etc.)
2. **Define response type** with JSON tags
3. **Implement handler** following standard pattern
4. **Register route** in serve.go with corsHandler
5. **Add tests** (method, JSON format, error handling)
6. **Update documentation**:
   - `serveCmd.Long` help text
   - `runServeStatus()` endpoint list
   - Console output in `runServe()`

### Example: Adding /api/example

```go
// In serve_system.go (or appropriate file)

type ExampleAPIResponse struct {
    Value string `json:"value"`
    Count int    `json:"count"`
}

func handleExample(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Query params
    filter := r.URL.Query().Get("filter")
    
    // Business logic
    result := getExampleData(filter)
    
    // Response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

// In serve.go runServe()
mux.HandleFunc("/api/example", corsHandler(handleExample))
```

---

## Refactoring Strategy

When serve.go grows too large (>1000 lines), split by domain:

**Phase approach (500-800 lines per phase):**
1. Extract largest/most complex handler group first
2. Move tests to corresponding test file
3. Verify build + tests after each extraction
4. Keep shared utilities (corsHandler, beadsClient) in serve.go

**File naming:** `serve_{domain}.go`, `serve_{domain}_test.go`

**Source:** `2026-01-03-inv-map-serve-go-api-handler.md` - Detailed split plan

---

## Known Constraints

### HTTP/1.1 Connection Limits

Browsers limit connections per origin (typically 6 for HTTP/1.1). SSE connections count against this limit, potentially blocking regular API calls.

**Mitigation:** 
- Keep SSE connections minimal
- Consider HTTP/2 for production deployment

### OpenCode API Stability

The OpenCode HTTP API can become unresponsive (redirect loops, hangs). Always:
- Use timeouts on HTTP clients (10s default)
- Limit redirects (max 10)
- Handle errors gracefully (return empty/default values)

**Source:** `2025-12-26-inv-api-endpoint-api-agents-hangs.md`

### Beads RPC Availability

Beads CLI uses RPC to communicate. If beads daemon is down:
- `bd` commands fail
- Batch operations return empty results
- Dashboard shows stale data

**Mitigation:** Use existing error handling patterns; don't crash on beads unavailability.

---

## Source Investigations

This guide synthesizes findings from:

| Investigation | Key Pattern |
|--------------|-------------|
| `2025-12-20-inv-add-api-agentlog-endpoint-serve.md` | SSE file polling |
| `2025-12-20-inv-poc-port-python-standalone-api.md` | TUI ready detection |
| `2025-12-24-inv-add-api-usage-endpoint-serve.md` | Reusing pkg/ functions |
| `2025-12-26-inv-add-api-errors-endpoint-error.md` | Error pattern analysis |
| `2025-12-26-inv-api-endpoint-api-agents-hangs.md` | HTTP timeouts |
| `2025-12-26-inv-evaluate-building-api-proxy-layer.md` | ToS constraints |
| `2025-12-27-inv-api-agents-endpoint-takes-19s.md` | N+1 elimination |
| `2026-01-03-inv-add-api-changelog-endpoint-orch.md` | Extracting shared logic |
| `2026-01-03-inv-map-serve-go-api-handler.md` | File split strategy |
| `2026-01-05-inv-pending-reviews-api-times-out.md` | Batch fetching |
| `2025-12-26-add-api-reflect-endpoint-expose.md` | Simple JSON endpoints |

---

## Quick Reference

**Performance:** Batch → Parallelize → Cache

**HTTP clients:** Always set timeouts (except SSE)

**Response format:** JSON with snake_case fields

**Testing:** Method + JSON format + error handling

**Refactoring:** Domain-based file split, 500-800 lines per phase
