package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Enable pprof for CPU profiling
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/attention"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/certs"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/materializer"
	"github.com/dylan-conlin/orch-go/pkg/notify"
	"github.com/dylan-conlin/orch-go/pkg/service"
	"github.com/spf13/cobra"
)

// tlsConfigSkipVerify returns a TLS config that skips certificate verification.
// Used for connecting to the local server with self-signed certificates.
func tlsConfigSkipVerify() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // Self-signed localhost cert
	}
}

// DefaultServePort is the port orch serve listens on.
// This is infrastructure, not a project dev server.
const DefaultServePort = 3348

var (
	servePort int

	// serverStartTime tracks when the orch serve process started.
	// Used to distinguish agent death reasons (server restart vs other failures).
	serverStartTime time.Time

	// beadsClient is a persistent RPC client for beads operations.
	// Initialized at startup with auto-reconnect enabled.
	// Protected by beadsClientMu for thread-safe access across HTTP handlers.
	beadsClient   *beads.Client
	beadsClientMu sync.RWMutex

	// serviceMonitor is the global service monitor instance for accessing service state.
	// Initialized at startup and used by /api/services endpoint.
	// Protected by serviceMonitorMu for thread-safe access across HTTP handlers.
	serviceMonitor   *service.ServiceMonitor
	serviceMonitorMu sync.RWMutex

	// likelyDoneCache caches LIKELY_DONE attention signals.
	// Initialized at startup with 5-minute TTL to avoid slow git/workspace scans.
	globalLikelyDoneCache *attention.LikelyDoneCache

	// globalMaterializer is the SSE materializer that keeps state.db fresh
	// from OpenCode session events in real-time.
	globalMaterializer *materializer.Materializer
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server for the beads-ui dashboard",
	Long: `Start an HTTP API server that provides endpoints for the beads-ui dashboard.

This is orchestration infrastructure (persistent monitoring), NOT a project
dev server. Use 'orch serve status' to check if the API is running.

Endpoints:
  GET /api/agents    - Returns JSON list of active agents from OpenCode/tmux
  GET /api/events    - Proxies the OpenCode SSE stream for real-time updates
  GET /api/agentlog  - Agent lifecycle events
  GET /api/usage     - Claude Max usage stats
  GET /api/usage/cost - API cost tracking (Sonnet token usage)
  GET /api/focus     - Current focus and drift status
  GET /api/beads     - Beads stats (ready, blocked, open)
  GET /api/beads/ready - List of ready issues for queue visibility
  GET /api/beads/{id}/attempts - Attempt history for a beads issue
  GET /api/questions - Questions grouped by status (open, investigating, answered)
  GET /api/servers   - Servers status across projects
  GET /api/events/services - Service lifecycle events (supports ?follow=true for SSE)
  GET /api/daemon    - Daemon status (running, capacity, last poll)
  GET /api/gaps      - Gap tracker stats (total, recurring, by-skill)
  GET /api/reflect   - Reflect suggestions (synthesis, promote, stale)
  GET /api/kb-health - Knowledge hygiene signals (synthesis, promote, stale, investigation-promotion)
  GET /api/attention - Unified attention API (composes beads + git collectors, role-aware)
  GET /api/deliverables/{id} - Deliverables status for an issue
  POST /api/deliverables/override - Log override when closing with missing deliverables
  POST /api/attention/verify - Mark issue as verified or needs_fix (persisted to JSONL)
  GET /api/errors    - Error pattern analysis (recent errors, recurring patterns)
  GET /api/hotspot   - Hotspot analysis (fix density, investigation clusters)
  GET /api/frontier  - Decidability frontier (ready, blocked, active, stuck)
  GET /api/decisions - Decision center items grouped by action type
  GET/PUT /api/config - User configuration settings (~/.orch/config.yaml)
  GET /api/changelog - Aggregated changelog (?days=7&project=all)
  POST /api/approve  - Approve agent's work (creates beads comment + updates manifest)
  GET /health        - Health check

Examples:
  orch-go serve              # Start server on port 3348
  orch-go serve --port 8080  # Override with explicit port
  orch-go serve status       # Check if server is running`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServe(servePort)
	},
}

var serveStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the orch serve API is running",
	Long: `Check if the orch serve API server is running and accessible.

This command checks if the API server is listening on the expected port
and returns its status. The API server is orchestration infrastructure,
separate from project dev servers managed by 'orch servers'.

Examples:
  orch-go serve status           # Check status on default port
  orch-go serve status -p 8080   # Check status on custom port`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServeStatus(servePort)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", DefaultServePort, "Port to check/listen on")
	serveStatusCmd.Flags().IntVarP(&servePort, "port", "p", DefaultServePort, "Port to check")

	serveCmd.AddCommand(serveStatusCmd)
	rootCmd.AddCommand(serveCmd)
}

// runServeStatus checks if the orch serve API is running on the given port.
func runServeStatus(portNum int) error {
	addr := fmt.Sprintf("https://localhost:%d/health", portNum)

	// Skip TLS verification for self-signed localhost cert
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfigSkipVerify(),
		},
	}

	resp, err := client.Get(addr)
	if err != nil {
		fmt.Printf("❌ API server is NOT running on port %d\n", portNum)
		fmt.Println()
		fmt.Println("Start it with: orch serve")
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("⚠️  API server on port %d returned status %d\n", portNum, resp.StatusCode)
		return nil
	}

	// Parse the health response
	var health struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		fmt.Printf("✅ API server is running on port %d (health check responded)\n", portNum)
		return nil
	}

	fmt.Printf("✅ API server is running on port %d (HTTP/2 with TLS)\n", portNum)
	fmt.Printf("   Status: %s\n", health.Status)
	fmt.Printf("   URL:    https://localhost:%d\n", portNum)
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/agents    - Active agents")
	fmt.Println("  GET /api/events    - SSE event stream")
	fmt.Println("  GET /api/agentlog  - Agent lifecycle events")
	fmt.Println("  GET /api/usage     - Claude Max usage")
	fmt.Println("  GET /api/focus     - Focus and drift status")
	fmt.Println("  GET /api/beads     - Beads stats")
	fmt.Println("  GET /api/beads/ready - Ready issues list")
	fmt.Println("  GET /api/beads/graph - Full dependency graph (nodes + edges)")
	fmt.Println("  POST /api/beads/close - Close a beads issue")
	fmt.Println("  GET /api/beads/{id}/attempts - Attempt history for a beads issue")
	fmt.Println("  GET /api/questions   - Questions grouped by status")
	fmt.Println("  GET /api/servers   - Project servers status")
	fmt.Println("  GET /api/services  - Service health from overmind monitor")
	fmt.Println("  GET /api/events/services - Service lifecycle events (supports ?follow=true for SSE)")
	fmt.Println("  POST /api/issues   - Create new beads issue (for follow-ups)")
	fmt.Println("  POST /api/approve  - Approve agent's work")
	fmt.Println("  GET /api/daemon    - Daemon status (running, capacity, last poll)")
	fmt.Println("  GET /api/gaps      - Gap tracker stats")
	fmt.Println("  GET /api/reflect   - Reflect suggestions")
	fmt.Println("  GET /api/kb-health - Knowledge hygiene signals")
	fmt.Println("  GET /api/attention - Unified attention API")
	fmt.Println("  GET /api/deliverables/{id} - Deliverables status for an issue")
	fmt.Println("  POST /api/deliverables/override - Log override for missing deliverables")
	fmt.Println("  GET /api/errors    - Error pattern analysis")
	fmt.Println("  GET /api/frontier  - Decidability frontier")
	fmt.Println("  GET /api/decisions - Decision center items")
	fmt.Println("  GET /api/changelog - Aggregated changelog")
	fmt.Println("  GET /health        - Health check")

	return nil
}

func runServe(portNum int) error {
	// Record server start time for agent death diagnostics
	serverStartTime = time.Now()

	// Set default directory for beads socket discovery
	// This is needed because serve may run from any working directory
	if sourceDir != "" && sourceDir != "unknown" {
		beads.DefaultDir = sourceDir
	}

	// Resolve bd executable path at startup.
	// This is critical for launchd environments where PATH is minimal.
	// The resolved path is stored in beads.BdPath for use by Fallback* functions.
	if bdPath, err := beads.ResolveBdPath(); err != nil {
		fmt.Printf("Warning: could not resolve bd path (CLI fallback may fail): %v\n", err)
	} else {
		fmt.Printf("Resolved bd path: %s\n", bdPath)
	}

	// Initialize persistent beads client with auto-reconnect.
	// This avoids per-request connection overhead and handles daemon restarts.
	// Use 5s timeout (not 30s default) to fail fast when daemon dies.
	err := beads.Do(sourceDir, func(client *beads.Client) error {
		beadsClient = client
		return nil
	},
		beads.WithAutoReconnect(3),
		beads.WithTimeout(5*time.Second),
	)
	if err == nil {
		if connErr := beadsClient.Connect(); connErr != nil {
			// Non-fatal: handlers will fallback to CLI if client is nil
			fmt.Printf("Warning: beads daemon not available, using CLI fallback: %v\n", connErr)
			beadsClient = nil
		}
	}

	// Initialize bd subprocess limiter to prevent stampede under load.
	// Two-layer protection: singleflight deduplication + hard concurrency cap (max 5).
	// Without this, dashboard polling can spawn hundreds of concurrent bd subprocesses.
	globalBdLimiter = newBdLimiter()
	fmt.Printf("Initialized bd subprocess limiter (max %d concurrent)\n", maxBdConcurrent)

	// Start limiter stats logging in background (every 60s)
	limiterStop := make(chan struct{})
	go logLimiterStats(globalBdLimiter, 60*time.Second, limiterStop)
	defer close(limiterStop)

	// Initialize beads cache to prevent CPU spikes from excessive bd spawning.
	// Without caching, each /api/agents request spawns 20+ bd processes for 600+ workspaces.
	globalBeadsCache = newBeadsCache()

	// Initialize beads stats cache to prevent slow API responses.
	// Without caching, /api/beads spawns bd stats (~1.5s) on every request.
	globalBeadsStatsCache = newBeadsStatsCache()

	// Initialize kb health cache to prevent slow API responses.
	// kb reflect can be slow with many artifacts, so we cache with 5-minute TTL.
	globalKBHealthCache = newKBHealthCache()

	// Initialize likely done cache to prevent slow API responses.
	// Git log scanning and workspace checks can be slow, so we cache with 5-minute TTL.
	globalLikelyDoneCache = attention.NewLikelyDoneCache()

	// Start service monitoring daemon (Phase 1 MVP: crash detection + auto-restart)
	// Polls overmind status every 10s, tracks PIDs, emits crash notifications, auto-restarts services
	notifier := notify.Default()
	eventLogger := events.NewDefaultLogger()
	eventAdapter := service.NewEventLoggerAdapter(eventLogger)
	serviceMonitor = service.NewMonitor(sourceDir, notifier, eventAdapter, 10*time.Second, true)
	serviceMonitor.Start()
	fmt.Println("Started service monitor (polling every 10s, auto-restart enabled)")

	// Start SSE materializer to keep state.db fresh from OpenCode events.
	// Subscribes to SSE stream and writes is_processing, session_updated_at,
	// and token counts to state.db in real-time (~2s freshness target).
	globalMaterializer = materializer.New(materializer.Config{
		ServerURL: serverURL,
	})
	ctx := context.Background()
	if err := globalMaterializer.Start(ctx); err != nil {
		fmt.Printf("Warning: failed to start SSE materializer: %v\n", err)
		globalMaterializer = nil
	} else {
		fmt.Println("Started SSE materializer (state.db real-time sync)")
		defer globalMaterializer.Stop()
	}

	mux := http.NewServeMux()

	// CORS middleware wrapper
	corsHandler := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Allow requests from SvelteKit dev server and any localhost (http or https)
			origin := r.Header.Get("Origin")
			if origin == "" ||
				strings.HasPrefix(origin, "http://localhost") ||
				strings.HasPrefix(origin, "https://localhost") ||
				strings.HasPrefix(origin, "http://127.0.0.1") ||
				strings.HasPrefix(origin, "https://127.0.0.1") {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")

			// Cache invalidation headers (Phase 4: Dashboard Reliability)
			// Enable dashboard to detect stale data and prompt reload when binary updates
			w.Header().Set("X-Orch-Version", version)
			w.Header().Set("X-Cache-Time", time.Now().Format(time.RFC3339))

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			h(w, r)
		}
	}

	// GET /api/agents - returns JSON list of agents from OpenCode/tmux
	mux.HandleFunc("/api/agents", corsHandler(handleAgents))

	// GET /api/events - proxies OpenCode SSE stream
	mux.HandleFunc("/api/events", corsHandler(handleEvents))

	// GET /api/agentlog - returns agent lifecycle events from events.jsonl
	mux.HandleFunc("/api/agentlog", corsHandler(handleAgentlog))

	// GET /api/usage - returns Claude Max usage stats
	mux.HandleFunc("/api/usage", corsHandler(handleUsage))

	// GET /api/usage/cost - returns API cost tracking data
	mux.HandleFunc("/api/usage/cost", corsHandler(handleUsageCost))

	// GET /api/focus - returns current focus and drift status
	mux.HandleFunc("/api/focus", corsHandler(handleFocus))

	// GET /api/beads - returns beads stats (ready, blocked, open issues)
	mux.HandleFunc("/api/beads", corsHandler(handleBeads))

	// GET /api/beads/ready - returns list of ready issues for dashboard queue visibility
	mux.HandleFunc("/api/beads/ready", corsHandler(handleBeadsReady))

	// GET /api/beads/graph - returns full dependency graph (nodes + edges) for visualization
	mux.HandleFunc("/api/beads/graph", corsHandler(handleBeadsGraph))

	// GET /api/beads/{id}/attempts - returns attempt history for a specific beads issue
	// POST /api/beads/close - closes a beads issue
	mux.HandleFunc("/api/beads/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		// Route /api/beads/close to handleBeadsClose
		if r.URL.Path == "/api/beads/close" && r.Method == http.MethodPost {
			handleBeadsClose(w, r)
			return
		}
		// Route /api/beads/{id}/attempts to handleBeadsAttempts
		if strings.HasSuffix(r.URL.Path, "/attempts") {
			handleBeadsAttempts(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))

	// GET /api/questions - returns questions grouped by status for dashboard
	mux.HandleFunc("/api/questions", corsHandler(handleQuestions))

	// GET /api/servers - returns servers status across projects
	mux.HandleFunc("/api/servers", corsHandler(handleServers))

	// GET /api/services - returns service health from overmind monitor
	// GET /api/services/{name}/logs - returns logs for a specific service from overmind echo
	mux.HandleFunc("/api/services", corsHandler(handleServices))
	mux.HandleFunc("/api/services/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		// Route /api/services/{name}/logs to handleServiceLogs
		if strings.HasSuffix(r.URL.Path, "/logs") {
			handleServiceLogs(w, r)
		} else if r.URL.Path == "/api/services" || r.URL.Path == "/api/services/" {
			handleServices(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))

	// GET /api/events/services - returns service lifecycle events (supports ?follow=true for SSE)
	mux.HandleFunc("/api/events/services", corsHandler(handleServiceEvents))

	// POST /api/issues - creates a new beads issue (for follow-up from synthesis)
	mux.HandleFunc("/api/issues", corsHandler(handleIssues))

	// GET /api/daemon - returns daemon status (running, capacity, last poll)
	mux.HandleFunc("/api/daemon", corsHandler(handleDaemon))

	// GET /api/gaps - returns gap tracker statistics
	mux.HandleFunc("/api/gaps", corsHandler(handleGaps))

	// GET /api/reflect - returns reflect suggestions for kb reflect UI
	mux.HandleFunc("/api/reflect", corsHandler(handleReflect))

	// GET /api/kb-health - returns knowledge hygiene signals (synthesis, promote, stale, investigation-promotion)
	mux.HandleFunc("/api/kb-health", corsHandler(handleKBHealth))

	// GET /api/kb/artifacts - returns knowledge base artifacts for Work Graph Artifact Feed
	mux.HandleFunc("/api/kb/artifacts", corsHandler(handleKBArtifacts))

	// GET /api/attention - unified attention API composing multiple collectors
	mux.HandleFunc("/api/attention", corsHandler(handleAttention))

	// GET /api/attention/likely-done - returns issues with commits but no active workspace
	mux.HandleFunc("/api/attention/likely-done", corsHandler(handleLikelyDone))

	// POST /api/attention/verify - mark issue as verified or needs_fix
	mux.HandleFunc("/api/attention/verify", corsHandler(handleAttentionVerify))

	// POST /api/attention/verify-failed/clear - Clear a verification failure entry
	mux.HandleFunc("/api/attention/verify-failed/clear", corsHandler(handleVerifyFailedClear))

	// POST /api/attention/verify-failed/reset-status - Reset issue to open for re-spawn
	mux.HandleFunc("/api/attention/verify-failed/reset-status", corsHandler(handleVerifyFailedResetStatus))

	// GET /api/kb/artifact/content - returns full content of a specific artifact
	mux.HandleFunc("/api/kb/artifact/content", corsHandler(handleKBArtifactContent))

	// GET /api/errors - returns error pattern analysis
	mux.HandleFunc("/api/errors", corsHandler(handleErrors))

	// GET /api/pending-reviews - returns agents with unreviewed synthesis recommendations
	mux.HandleFunc("/api/pending-reviews", corsHandler(handlePendingReviews))

	// POST /api/dismiss-review - dismiss a specific recommendation
	mux.HandleFunc("/api/dismiss-review", corsHandler(handleDismissReview))

	// GET/PUT /api/config - user configuration settings
	mux.HandleFunc("/api/config", corsHandler(handleConfig))

	// GET/PUT /api/config/daemon - daemon-specific configuration
	mux.HandleFunc("/api/config/daemon", corsHandler(handleDaemonConfig))

	// GET /api/config/drift - check if plist matches config
	mux.HandleFunc("/api/config/drift", corsHandler(handleConfigDrift))

	// POST /api/config/regenerate - regenerate plist from config
	mux.HandleFunc("/api/config/regenerate", corsHandler(handleConfigRegenerate))

	// GET /api/changelog - aggregated changelog across ecosystem repos
	mux.HandleFunc("/api/changelog", corsHandler(handleChangelog))

	// POST /api/cache/invalidate - invalidate caches to force fresh data
	// Called by orch complete to ensure dashboard shows updated status
	mux.HandleFunc("/api/cache/invalidate", corsHandler(handleCacheInvalidate))

	// GET /api/hotspot - returns hotspot analysis (fix density, investigation clusters)
	mux.HandleFunc("/api/hotspot", corsHandler(handleHotspot))

	// GET /api/orchestrator-sessions - returns active orchestrator sessions from registry
	mux.HandleFunc("/api/orchestrator-sessions", corsHandler(handleOrchestratorSessions))

	// GET /api/file - returns file contents for investigation/workspace files
	mux.HandleFunc("/api/file", corsHandler(handleFile))

	// GET /api/screenshots - returns list of screenshots for an agent
	mux.HandleFunc("/api/screenshots", corsHandler(handleScreenshots))

	// GET /api/context - returns current tmux cwd and resolved projects for "follow orchestrator" filtering
	mux.HandleFunc("/api/context", corsHandler(handleContext))

	// GET /api/coaching - returns orchestrator behavioral coaching metrics
	mux.HandleFunc("/api/coaching", corsHandler(handleCoaching))

	// GET /api/frontier - returns decidability frontier (ready, blocked, active, stuck)
	mux.HandleFunc("/api/frontier", corsHandler(handleFrontier))

	// GET /api/decisions - returns decision center items grouped by action type
	mux.HandleFunc("/api/decisions", corsHandler(handleDecisions))

	// GET /api/session/{sessionID}/messages - proxies OpenCode session messages for activity feed history
	// Uses prefix matching to extract sessionID from path
	mux.HandleFunc("/api/session/", corsHandler(handleSessionMessages))

	// POST /api/approve - approve an agent's work (creates beads comment + updates workspace manifest)
	mux.HandleFunc("/api/approve", corsHandler(handleApprove))

	// GET /api/deliverables/{beads-id} - returns deliverables status for an issue
	// POST /api/deliverables/override - logs an override when closing with missing deliverables
	mux.HandleFunc("/api/deliverables/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/deliverables/")
		if path == "override" && r.Method == http.MethodPost {
			handleDeliverablesOverride(w, r)
		} else if path != "" && !strings.Contains(path, "/") {
			handleDeliverablesStatus(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))

	// Health check — includes bd limiter stats and materializer status for observability
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{"status": "ok"}
		if limiterStats := getLimiterStats(); limiterStats != nil {
			response["bd_limiter"] = limiterStats
		}
		if globalMaterializer != nil {
			response["materializer"] = globalMaterializer.Status()
		}
		json.NewEncoder(w).Encode(response)
	})

	// pprof handlers for CPU profiling (useful for debugging CPU runaway)
	// Access at: https://localhost:3348/debug/pprof/
	mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)

	// Load TLS certificate from embedded bytes
	cert, err := tls.X509KeyPair(certs.CertPEM, certs.KeyPEM)
	if err != nil {
		return fmt.Errorf("failed to load embedded TLS certificate: %w", err)
	}

	addr := fmt.Sprintf(":%d", portNum)
	fmt.Printf("Starting orch-go API server on https://localhost%s (HTTP/2 with TLS)\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/agents    - List of active agents from OpenCode/tmux")
	fmt.Println("  GET /api/events    - SSE proxy for OpenCode events")
	fmt.Println("  GET /api/agentlog  - Agent lifecycle events (supports ?follow=true for SSE)")
	fmt.Println("  GET /api/usage     - Claude Max usage stats")
	fmt.Println("  GET /api/usage/cost - API cost tracking (Sonnet token usage)")
	fmt.Println("  GET /api/focus     - Current focus and drift status")
	fmt.Println("  GET /api/beads     - Beads stats (ready, blocked, open)")
	fmt.Println("  GET /api/beads/ready - List of ready issues for queue visibility")
	fmt.Println("  GET /api/beads/graph - Full dependency graph (nodes + edges)")
	fmt.Println("  POST /api/beads/close - Close a beads issue")
	fmt.Println("  GET /api/beads/{id}/attempts - Attempt history for a beads issue")
	fmt.Println("  GET /api/questions - Questions grouped by status")
	fmt.Println("  GET /api/servers   - Servers status across projects")
	fmt.Println("  GET /api/services  - Service health from overmind monitor")
	fmt.Println("  GET /api/events/services - Service lifecycle events (supports ?follow=true for SSE)")
	fmt.Println("  POST /api/issues   - Create new beads issue (for follow-ups)")
	fmt.Println("  GET /api/gaps      - Gap tracker stats (total, recurring, by-skill)")
	fmt.Println("  GET /api/reflect   - Reflect suggestions (synthesis, promote, stale)")
	fmt.Println("  GET /api/kb-health - Knowledge hygiene signals (synthesis, promote, stale, investigation-promotion)")
	fmt.Println("  GET /api/attention - Unified attention API (beads + git collectors, role-aware)")
	fmt.Println("  POST /api/attention/verify - Mark issue as verified or needs_fix (persisted to JSONL)")
	fmt.Println("  GET /api/errors    - Error pattern analysis (recent errors, recurring patterns)")
	fmt.Println("  GET /api/hotspot   - Hotspot analysis (fix density, investigation clusters)")
	fmt.Println("  GET /api/orchestrator-sessions - Active orchestrator sessions")
	fmt.Println("  GET /api/frontier   - Decidability frontier (ready, blocked, active, stuck)")
	fmt.Println("  GET /api/decisions  - Decision center items (absorb_knowledge, give_approvals, answer_questions, handle_failures)")
	fmt.Println("  GET /api/pending-reviews - Agents with unreviewed synthesis recommendations")
	fmt.Println("  POST /api/dismiss-review - Dismiss a specific recommendation")
	fmt.Println("  GET/PUT /api/config - User configuration settings")
	fmt.Println("  GET /api/changelog - Aggregated changelog (?days=7&project=all)")
	fmt.Println("  GET /api/file      - Read file contents (?path=/path/to/file)")
	fmt.Println("  POST /api/approve  - Approve agent's work")
	fmt.Println("  GET /health        - Health check")
	fmt.Println("\nPress Ctrl+C to stop")

	// HTTP/2 is automatically enabled when using TLS with Go's http package
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	server := &http.Server{Addr: addr, Handler: mux, TLSConfig: tlsConfig}
	return server.ListenAndServeTLS("", "")
}

// handleChangelog returns aggregated changelog data across ecosystem repos.
// Query parameters:
//   - days: Number of days to include (default: 7)
//   - project: Project to filter (default: "all" for all repos)
func handleChangelog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	daysStr := r.URL.Query().Get("days")
	project := r.URL.Query().Get("project")

	// Default values
	days := 7
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}
	if project == "" {
		project = "all"
	}

	// Get changelog data using the reusable function from changelog.go
	result, err := GetChangelog(days, project)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get changelog: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode changelog: %v", err), http.StatusInternalServerError)
		return
	}
}
