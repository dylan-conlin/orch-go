package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Enable pprof for CPU profiling
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/beads"
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

	// beadsClient is a persistent RPC client for beads operations.
	// Initialized at startup with auto-reconnect enabled.
	beadsClient *beads.Client
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
  GET /api/focus     - Current focus and drift status
  GET /api/beads     - Beads stats (ready, blocked, open)
  GET /api/beads/ready - List of ready issues for queue visibility
  GET /api/servers   - Servers status across projects
  GET /api/daemon    - Daemon status (running, capacity, last poll)
  GET /api/gaps      - Gap tracker stats (total, recurring, by-skill)
  GET /api/reflect   - Reflect suggestions (synthesis, promote, stale)
  GET /api/errors    - Error pattern analysis (recent errors, recurring patterns)
  GET /api/hotspot   - Hotspot analysis (fix density, investigation clusters)
  GET/PUT /api/config - User configuration settings (~/.orch/config.yaml)
  GET /api/changelog - Aggregated changelog (?days=7&project=all)
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
	fmt.Println("  GET /api/servers   - Project servers status")
	fmt.Println("  POST /api/issues   - Create new beads issue (for follow-ups)")
	fmt.Println("  GET /api/daemon    - Daemon status (running, capacity, last poll)")
	fmt.Println("  GET /api/gaps      - Gap tracker stats")
	fmt.Println("  GET /api/reflect   - Reflect suggestions")
	fmt.Println("  GET /api/errors    - Error pattern analysis")
	fmt.Println("  GET /api/changelog - Aggregated changelog")
	fmt.Println("  GET /health        - Health check")

	return nil
}

func runServe(portNum int) error {
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
	socketPath, err := beads.FindSocketPath(sourceDir)
	if err == nil {
		beadsClient = beads.NewClient(socketPath,
			beads.WithAutoReconnect(3),
			beads.WithTimeout(5*time.Second),
		)
		if connErr := beadsClient.Connect(); connErr != nil {
			// Non-fatal: handlers will fallback to CLI if client is nil
			fmt.Printf("Warning: beads daemon not available, using CLI fallback: %v\n", connErr)
			beadsClient = nil
		}
	}

	// Initialize beads cache to prevent CPU spikes from excessive bd spawning.
	// Without caching, each /api/agents request spawns 20+ bd processes for 600+ workspaces.
	globalBeadsCache = newBeadsCache()

	// Initialize beads stats cache to prevent slow API responses.
	// Without caching, /api/beads spawns bd stats (~1.5s) on every request.
	globalBeadsStatsCache = newBeadsStatsCache()

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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")

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

	// GET /api/focus - returns current focus and drift status
	mux.HandleFunc("/api/focus", corsHandler(handleFocus))

	// GET /api/beads - returns beads stats (ready, blocked, open issues)
	mux.HandleFunc("/api/beads", corsHandler(handleBeads))

	// GET /api/beads/ready - returns list of ready issues for dashboard queue visibility
	mux.HandleFunc("/api/beads/ready", corsHandler(handleBeadsReady))

	// GET /api/servers - returns servers status across projects
	mux.HandleFunc("/api/servers", corsHandler(handleServers))

	// POST /api/issues - creates a new beads issue (for follow-up from synthesis)
	mux.HandleFunc("/api/issues", corsHandler(handleIssues))

	// GET /api/daemon - returns daemon status (running, capacity, last poll)
	mux.HandleFunc("/api/daemon", corsHandler(handleDaemon))

	// GET /api/gaps - returns gap tracker statistics
	mux.HandleFunc("/api/gaps", corsHandler(handleGaps))

	// GET /api/reflect - returns reflect suggestions for kb reflect UI
	mux.HandleFunc("/api/reflect", corsHandler(handleReflect))

	// GET /api/errors - returns error pattern analysis
	mux.HandleFunc("/api/errors", corsHandler(handleErrors))

	// GET /api/pending-reviews - returns agents with unreviewed synthesis recommendations
	mux.HandleFunc("/api/pending-reviews", corsHandler(handlePendingReviews))

	// POST /api/dismiss-review - dismiss a specific recommendation
	mux.HandleFunc("/api/dismiss-review", corsHandler(handleDismissReview))

	// GET/PUT /api/config - user configuration settings
	mux.HandleFunc("/api/config", corsHandler(handleConfig))

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

	// GET /api/context - returns current tmux cwd and resolved projects for "follow orchestrator" filtering
	mux.HandleFunc("/api/context", corsHandler(handleContext))

	// GET /api/session/{sessionID}/messages - proxies OpenCode session messages for activity feed history
	// Uses prefix matching to extract sessionID from path
	mux.HandleFunc("/api/session/", corsHandler(handleSessionMessages))

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// pprof handlers for CPU profiling (useful for debugging CPU runaway)
	// Access at: https://localhost:3348/debug/pprof/
	mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)

	// TLS certificate paths (relative to source directory)
	certFile := filepath.Join(sourceDir, "pkg", "certs", "cert.pem")
	keyFile := filepath.Join(sourceDir, "pkg", "certs", "key.pem")

	addr := fmt.Sprintf(":%d", portNum)
	fmt.Printf("Starting orch-go API server on https://localhost%s (HTTP/2 with TLS)\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/agents    - List of active agents from OpenCode/tmux")
	fmt.Println("  GET /api/events    - SSE proxy for OpenCode events")
	fmt.Println("  GET /api/agentlog  - Agent lifecycle events (supports ?follow=true for SSE)")
	fmt.Println("  GET /api/usage     - Claude Max usage stats")
	fmt.Println("  GET /api/focus     - Current focus and drift status")
	fmt.Println("  GET /api/beads     - Beads stats (ready, blocked, open)")
	fmt.Println("  GET /api/beads/ready - List of ready issues for queue visibility")
	fmt.Println("  GET /api/servers   - Servers status across projects")
	fmt.Println("  POST /api/issues   - Create new beads issue (for follow-ups)")
	fmt.Println("  GET /api/gaps      - Gap tracker stats (total, recurring, by-skill)")
	fmt.Println("  GET /api/reflect   - Reflect suggestions (synthesis, promote, stale)")
	fmt.Println("  GET /api/errors    - Error pattern analysis (recent errors, recurring patterns)")
	fmt.Println("  GET /api/hotspot   - Hotspot analysis (fix density, investigation clusters)")
	fmt.Println("  GET /api/orchestrator-sessions - Active orchestrator sessions")
	fmt.Println("  GET /api/pending-reviews - Agents with unreviewed synthesis recommendations")
	fmt.Println("  POST /api/dismiss-review - Dismiss a specific recommendation")
	fmt.Println("  GET/PUT /api/config - User configuration settings")
	fmt.Println("  GET /api/changelog - Aggregated changelog (?days=7&project=all)")
	fmt.Println("  GET /api/file      - Read file contents (?path=/path/to/file)")
	fmt.Println("  GET /health        - Health check")
	fmt.Println("\nPress Ctrl+C to stop")

	// HTTP/2 is automatically enabled when using TLS with Go's http package
	return http.ListenAndServeTLS(addr, certFile, keyFile, mux)
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
