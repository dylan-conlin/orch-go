package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Enable pprof for CPU profiling
	"os"
	"path"
	"path/filepath"
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
	"github.com/dylan-conlin/orch-go/pkg/process"
	"github.com/dylan-conlin/orch-go/pkg/service"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
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
	servePort       int
	serveWithDaemon bool // Enable daemon alongside serve
)

// Server holds all dependencies for HTTP handlers, enabling unit testing
// without global state. Created in runServe and passed to registerRoutes.
type Server struct {
	ServerURL       string
	SourceDir       string
	Version         string
	ServerStartTime time.Time

	BeadsClient   *beads.Client
	BeadsClientMu sync.RWMutex

	ServiceMonitor   *service.ServiceMonitor
	ServiceMonitorMu sync.RWMutex

	LikelyDoneCache *attention.LikelyDoneCache
	Materializer    *materializer.Materializer
	ResourceMonitor *resourceMonitor

	BdLimiter       *bdLimiter
	BeadsCache      *beadsCache
	BeadsStatsCache *beadsStatsCache
	KBHealthCache   *kbHealthCache
	WorkspaceCache  *globalWorkspaceCacheType
}

func (s *Server) currentProjectDir() (string, error) {
	if s.SourceDir != "" && s.SourceDir != "unknown" {
		return s.SourceDir, nil
	}
	return os.Getwd()
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server for the beads-ui dashboard",
	Long: `Start an HTTP API server that provides endpoints for the beads-ui dashboard.

This is orchestration infrastructure (persistent monitoring), NOT a project
dev server. Use 'orch serve status' to check health and 'orch serve restart'
for managed restarts.

Endpoints:
  GET /             - Dashboard UI (static build from web/build)
  GET /api/agents    - Returns JSON list of active agents from OpenCode/tmux
  GET /api/events    - Proxies the OpenCode SSE stream for real-time updates
  GET /api/agentlog  - Agent lifecycle events
  GET /api/usage     - Claude Max usage stats
  GET /api/usage/cost - API cost tracking (Sonnet token usage)
  GET /api/focus     - Current focus and drift status
  GET /api/beads     - Beads stats (ready, blocked, open)
  GET /api/beads/ready - List of ready issues for queue visibility
  GET /api/beads/{id}/attempts - Attempt history for a beads issue
  GET /api/beads/{id}/completion - Completion details (message, commits, artifacts)
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
  GET /api/operator-health - Behavioral system health dashboard metrics
  GET /api/outcomes - Normalized outcome metrics (skills, durations, abandonment, throughput)
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
  orch-go serve --daemon     # Start server with daemon (overrides config)
  orch-go serve status       # Check if server is running
  orch-go serve restart      # Restart via overmind/launchd manager`,
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

var serveRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart orch serve through its process manager",
	Long: `Restart orch serve using a managed lifecycle path.

Restart order:
  1. overmind restart api (development default)
  2. launchctl kickstart for known launchd labels (legacy compatibility)

If orch serve is running unmanaged (for example via nohup), this command
returns a remediation message instead of spawning another orphan process.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServeRestart()
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", DefaultServePort, "Port to check/listen on")
	serveCmd.Flags().BoolVar(&serveWithDaemon, "daemon", false, "Enable daemon alongside serve (overrides config)")
	serveStatusCmd.Flags().IntVarP(&servePort, "port", "p", DefaultServePort, "Port to check")

	serveCmd.AddCommand(serveStatusCmd)
	serveCmd.AddCommand(serveRestartCmd)
	rootCmd.AddCommand(serveCmd)
}

// runServeStatus checks if the orch serve API is running on the given port.
func runServeStatus(portNum int) error {
	addr := fmt.Sprintf("https://localhost:%d/health", portNum)

	// Skip TLS verification for self-signed localhost cert
	client := &http.Client{Timeout: 2 * time.Second, Transport: serveOutboundTLSTransport}

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
	fmt.Println("  GET /             - Dashboard UI (static build from web/build)")
	fmt.Println("  GET /api/agents    - Active agents")
	fmt.Println("  GET /api/events    - SSE event stream")
	fmt.Println("  GET /api/agentlog  - Agent lifecycle events")
	fmt.Println("  GET /api/usage     - Claude Max usage")
	fmt.Println("  GET /api/focus     - Focus and drift status")
	fmt.Println("  GET /api/beads     - Beads stats")
	fmt.Println("  GET /api/beads/ready - Ready issues list")
	fmt.Println("  GET /api/beads/graph - Full dependency graph (nodes + edges)")
	fmt.Println("  POST /api/beads/close - Close a beads issue")
	fmt.Println("  POST /api/beads/update - Update priority and labels for a beads issue")
	fmt.Println("  GET /api/beads/{id}/attempts - Attempt history for a beads issue")
	fmt.Println("  GET /api/beads/{id}/completion - Completion details (message, commits, artifacts)")
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
	fmt.Println("  GET /api/operator-health - Behavioral system health metrics")
	fmt.Println("  GET /api/outcomes - Normalized outcome metrics")
	fmt.Println("  GET /api/frontier  - Decidability frontier")
	fmt.Println("  GET /api/decisions - Decision center items")
	fmt.Println("  GET /api/changelog - Aggregated changelog")
	fmt.Println("  GET /health        - Health check")

	return nil
}

func runServe(portNum int) error {
	// Record server start time for agent death diagnostics
	srvStartTime := time.Now()

	// Check if daemon should be enabled based on config and flag
	cfg, configErr := userconfig.Load()
	if configErr != nil {
		return fmt.Errorf("failed to load config: %w", configErr)
	}

	// Determine if daemon should run: --daemon flag overrides config
	daemonEnabled := serveWithDaemon || cfg.DaemonEnabled()

	if !daemonEnabled {
		fmt.Println("Daemon auto-start disabled (supervised-first workflow)")
		fmt.Println("Use 'orch serve --daemon' or set daemon.enabled: true in config to enable")
	}

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

	// Startup sweep: reconcile ledger against live PIDs, remove stale entries,
	// and kill orphaned bun processes. This closes the "restart window" where
	// stale agents accumulate.
	startupResult := process.StartupSweepWithReconciliation()
	if startupResult.Error != nil {
		fmt.Printf("Warning: startup sweep failed: %v\n", startupResult.Error)
	} else {
		if startupResult.LedgerStaleRemoved > 0 || startupResult.OrphanProcessesKilled > 0 {
			fmt.Printf("Startup sweep: removed %d stale ledger entries", startupResult.LedgerStaleRemoved)
			if startupResult.OrphanProcessesKilled > 0 {
				fmt.Printf(", killed %d orphaned processes", startupResult.OrphanProcessesKilled)
			}
			fmt.Println()
		}
	}

	// Initialize persistent beads client with auto-reconnect.
	// This avoids per-request connection overhead and handles daemon restarts.
	// Use 5s timeout (not 30s default) to fail fast when daemon dies.
	var initBeadsClient *beads.Client
	err := beads.Do(sourceDir, func(client *beads.Client) error {
		initBeadsClient = client
		return nil
	},
		beads.WithAutoReconnect(3),
		beads.WithTimeout(5*time.Second),
	)
	if err == nil {
		if connErr := initBeadsClient.Connect(); connErr != nil {
			// Non-fatal: handlers will fallback to CLI if client is nil
			fmt.Printf("Warning: beads daemon not available, using CLI fallback: %v\n", connErr)
			initBeadsClient = nil
		}
	}

	// Initialize bd subprocess limiter to prevent stampede under load.
	// Two-layer protection: singleflight deduplication + hard concurrency cap (configurable).
	// Without this, dashboard polling can spawn hundreds of concurrent bd subprocesses.
	bdLim := newBdLimiter()
	fmt.Printf("Initialized bd subprocess limiter (max %d concurrent)\n", maxBdConcurrent)

	// Start limiter stats logging in background (every 60s)
	limiterStop := make(chan struct{})
	go logLimiterStats(bdLim, 60*time.Second, limiterStop)
	defer close(limiterStop)

	// Initialize beads cache to prevent CPU spikes from excessive bd spawning.
	// Without caching, each /api/agents request spawns 20+ bd processes for 600+ workspaces.
	bCache := newBeadsCache(defaultBeadsCacheMaxEntries, defaultOpenIssuesTTL)

	// Initialize beads stats cache to prevent slow API responses.
	// Without caching, /api/beads spawns bd stats (~1.5s) on every request.
	bsCache := newBeadsStatsCache(defaultBeadsStatsCacheMaxEntries, defaultBeadsStatsCacheTTL)

	// Initialize kb health cache to prevent slow API responses.
	// kb reflect can be slow with many artifacts, so we cache with 5-minute TTL.
	kbCache := newKBHealthCache(defaultKBHealthCacheMaxEntries, defaultKBHealthCacheTTL)

	// Initialize likely done cache to prevent slow API responses.
	// Git log scanning and workspace checks can be slow, so we cache with 5-minute TTL.
	ldCache := attention.NewLikelyDoneCache(attention.DefaultLikelyDoneCacheMaxEntries, attention.DefaultLikelyDoneCacheTTL)

	// Start service monitoring daemon (Phase 1 MVP: crash detection + auto-restart)
	// Polls overmind status every 10s, tracks PIDs, emits crash notifications, auto-restarts services
	notifier := notify.Default()
	eventLogger := events.NewDefaultLogger()
	eventAdapter := service.NewEventLoggerAdapter(eventLogger)
	svcMonitor := service.NewMonitor(sourceDir, notifier, eventAdapter, 10*time.Second, true)
	svcMonitor.Start()
	fmt.Println("Started service monitor (polling every 10s, auto-restart enabled)")

	// Start SSE materializer to keep state.db fresh from OpenCode events.
	// Subscribes to SSE stream and writes is_processing, session_updated_at,
	// and token counts to state.db in real-time (~2s freshness target).
	mat := materializer.New(materializer.Config{
		ServerURL: serverURL,
	})
	ctx := context.Background()
	if err := mat.Start(ctx); err != nil {
		fmt.Printf("Warning: failed to start SSE materializer: %v\n", err)
		mat = nil
	} else {
		fmt.Println("Started SSE materializer (state.db real-time sync)")
		defer mat.Stop()
	}

	// Create the Server with all dependencies.
	// Handlers access these via the receiver instead of globals.
	s := &Server{
		ServerURL:       serverURL,
		SourceDir:       sourceDir,
		Version:         version,
		ServerStartTime: srvStartTime,
		BeadsClient:     initBeadsClient,
		ServiceMonitor:  svcMonitor,
		LikelyDoneCache: ldCache,
		Materializer:    mat,
		ResourceMonitor: newResourceMonitor(eventLogger),
		BdLimiter:       bdLim,
		BeadsCache:      bCache,
		BeadsStatsCache: bsCache,
		KBHealthCache:   kbCache,
		WorkspaceCache:  globalWorkspaceCacheInstance,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Load TLS certificate from embedded bytes
	cert, err := tls.X509KeyPair(certs.CertPEM, certs.KeyPEM)
	if err != nil {
		return fmt.Errorf("failed to load embedded TLS certificate: %w", err)
	}

	addr := fmt.Sprintf(":%d", portNum)
	fmt.Printf("Starting orch-go API server on https://localhost%s (HTTP/2 with TLS)\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET /             - Dashboard UI (static build from web/build)")
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
	fmt.Println("  POST /api/beads/update - Update priority and labels for a beads issue")
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
	fmt.Println("  GET /api/operator-health - Behavioral system health metrics")
	fmt.Println("  GET /api/outcomes - Normalized outcome metrics")
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

// corsHandler wraps an HTTP handler with CORS headers and version info.
func (s *Server) corsHandler(h http.HandlerFunc) http.HandlerFunc {
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
		w.Header().Set("X-Orch-Version", s.Version)
		w.Header().Set("X-Cache-Time", time.Now().Format(time.RFC3339))

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		h(w, r)
	}
}

// registerRoutes wires all HTTP handlers onto the given mux.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	c := s.corsHandler

	mux.HandleFunc("/api/agents", c(s.handleAgents))
	mux.HandleFunc("/api/events", c(s.handleEvents))
	mux.HandleFunc("/api/agentlog", c(s.handleAgentlog))
	mux.HandleFunc("/api/usage", c(s.handleUsage))
	mux.HandleFunc("/api/usage/cost", c(s.handleUsageCost))
	mux.HandleFunc("/api/focus", c(s.handleFocus))
	mux.HandleFunc("/api/beads", c(s.handleBeads))
	mux.HandleFunc("/api/beads/ready", c(s.handleBeadsReady))
	mux.HandleFunc("/api/beads/graph", c(s.handleBeadsGraph))
	mux.HandleFunc("/api/beads/", c(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/beads/close" && r.Method == http.MethodPost {
			s.handleBeadsClose(w, r)
			return
		}
		if r.URL.Path == "/api/beads/update" && r.Method == http.MethodPost {
			s.handleBeadsUpdate(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/attempts") {
			s.handleBeadsAttempts(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/completion") {
			s.handleBeadsCompletion(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))
	mux.HandleFunc("/api/questions", c(s.handleQuestions))
	mux.HandleFunc("/api/servers", c(s.handleServers))
	mux.HandleFunc("/api/services", c(s.handleServices))
	mux.HandleFunc("/api/services/", c(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/logs") {
			s.handleServiceLogs(w, r)
		} else if r.URL.Path == "/api/services" || r.URL.Path == "/api/services/" {
			s.handleServices(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))
	mux.HandleFunc("/api/events/services", c(s.handleServiceEvents))
	mux.HandleFunc("/api/issues", c(s.handleIssues))
	mux.HandleFunc("/api/daemon", c(s.handleDaemon))
	mux.HandleFunc("/api/gaps", c(s.handleGaps))
	mux.HandleFunc("/api/reflect", c(s.handleReflect))
	mux.HandleFunc("/api/kb-health", c(s.handleKBHealth))
	mux.HandleFunc("/api/kb/artifacts", c(s.handleKBArtifacts))
	mux.HandleFunc("/api/kb/model-probes", c(s.handleKBModelProbes))
	mux.HandleFunc("/api/attention", c(s.handleAttention))
	mux.HandleFunc("/api/attention/likely-done", c(s.handleLikelyDone))
	mux.HandleFunc("/api/attention/verify", c(s.handleAttentionVerify))
	mux.HandleFunc("/api/attention/verify-failed/clear", c(s.handleVerifyFailedClear))
	mux.HandleFunc("/api/attention/verify-failed/reset-status", c(s.handleVerifyFailedResetStatus))
	mux.HandleFunc("/api/kb/artifact/content", c(s.handleKBArtifactContent))
	mux.HandleFunc("/api/errors", c(s.handleErrors))
	mux.HandleFunc("/api/operator-health", c(s.handleOperatorHealth))
	mux.HandleFunc("/api/outcomes", c(s.handleOutcomes))
	mux.HandleFunc("/api/pending-reviews", c(s.handlePendingReviews))
	mux.HandleFunc("/api/dismiss-review", c(s.handleDismissReview))
	mux.HandleFunc("/api/config", c(s.handleConfig))
	mux.HandleFunc("/api/config/daemon", c(s.handleDaemonConfig))
	mux.HandleFunc("/api/config/drift", c(s.handleConfigDrift))
	mux.HandleFunc("/api/config/regenerate", c(s.handleConfigRegenerate))
	mux.HandleFunc("/api/changelog", c(s.handleChangelog))
	mux.HandleFunc("/api/cache/invalidate", c(s.handleCacheInvalidate))
	mux.HandleFunc("/api/hotspot", c(s.handleHotspot))
	mux.HandleFunc("/api/orchestrator-sessions", c(s.handleOrchestratorSessions))
	mux.HandleFunc("/api/file", c(s.handleFile))
	mux.HandleFunc("/api/screenshots", c(s.handleScreenshots))
	mux.HandleFunc("/api/context", c(s.handleContext))
	mux.HandleFunc("/api/frontier", c(s.handleFrontier))
	mux.HandleFunc("/api/decisions", c(s.handleDecisions))
	mux.HandleFunc("/api/session/", c(s.handleSessionMessages))
	mux.HandleFunc("/api/approve", c(s.handleApprove))
	mux.HandleFunc("/api/deliverables/", c(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/deliverables/")
		if path == "override" && r.Method == http.MethodPost {
			s.handleDeliverablesOverride(w, r)
		} else if path != "" && !strings.Contains(path, "/") {
			s.handleDeliverablesStatus(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))

	// Health check — includes bd limiter stats and materializer status
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{"status": "ok"}
		if s.ResourceMonitor != nil {
			response["resources"] = s.ResourceMonitor.sampleAndCheck()
		}
		if limiterStats := s.getLimiterStats(); limiterStats != nil {
			response["bd_limiter"] = limiterStats
		}
		if s.Materializer != nil {
			response["materializer"] = s.Materializer.Status()
		}
		json.NewEncoder(w).Encode(response)
	})

	// pprof handlers for CPU profiling
	mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)

	// Dashboard static files and SPA fallback.
	mux.HandleFunc("/", s.handleDashboardStatic)
}

func (s *Server) handleDashboardStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}

	if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/debug/") {
		http.NotFound(w, r)
		return
	}

	projectDir, err := s.currentProjectDir()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to resolve project directory: %v", err), http.StatusInternalServerError)
		return
	}

	buildDir := filepath.Join(projectDir, "web", "build")
	indexPath := filepath.Join(buildDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			http.Error(
				w,
				fmt.Sprintf("dashboard build not found at %s; run 'cd %s/web && bun run build'", buildDir, projectDir),
				http.StatusServiceUnavailable,
			)
			return
		}

		http.Error(w, fmt.Sprintf("failed to access dashboard build: %v", err), http.StatusInternalServerError)
		return
	}

	cleanPath := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(r.URL.Path, "/")), "/")
	if cleanPath == "" {
		cleanPath = "index.html"
	}

	targetPath := filepath.Join(buildDir, filepath.FromSlash(cleanPath))
	if info, err := os.Stat(targetPath); err == nil && !info.IsDir() {
		http.ServeFile(w, r, targetPath)
		return
	} else if err != nil && !os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("failed to access dashboard file: %v", err), http.StatusInternalServerError)
		return
	}

	// Missing assets (for example .js/.css) should stay 404.
	if path.Ext(cleanPath) != "" {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, indexPath)
}

// handleChangelog returns aggregated changelog data across ecosystem repos.
// Query parameters:
//   - days: Number of days to include (default: 7)
//   - project: Project to filter (default: "all" for all repos)
func (s *Server) handleChangelog(w http.ResponseWriter, r *http.Request) {
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
