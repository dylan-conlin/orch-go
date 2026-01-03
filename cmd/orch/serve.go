package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof" // Enable pprof for CPU profiling
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/daemon"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/usage"
	"github.com/dylan-conlin/orch-go/pkg/userconfig"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

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
	addr := fmt.Sprintf("http://localhost:%d/health", portNum)

	client := &http.Client{
		Timeout: 2 * time.Second,
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

	fmt.Printf("✅ API server is running on port %d\n", portNum)
	fmt.Printf("   Status: %s\n", health.Status)
	fmt.Printf("   URL:    http://localhost:%d\n", portNum)
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

	// Initialize persistent beads client with auto-reconnect.
	// This avoids per-request connection overhead and handles daemon restarts.
	socketPath, err := beads.FindSocketPath(sourceDir)
	if err == nil {
		beadsClient = beads.NewClient(socketPath, beads.WithAutoReconnect(3))
		if connErr := beadsClient.Connect(); connErr != nil {
			// Non-fatal: handlers will fallback to CLI if client is nil
			fmt.Printf("Warning: beads daemon not available, using CLI fallback: %v\n", connErr)
			beadsClient = nil
		}
	}

	mux := http.NewServeMux()

	// CORS middleware wrapper
	corsHandler := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Allow requests from SvelteKit dev server and any localhost
			origin := r.Header.Get("Origin")
			if origin == "" || strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// pprof handlers for CPU profiling (useful for debugging CPU runaway)
	// Access at: http://localhost:3348/debug/pprof/
	mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)

	addr := fmt.Sprintf(":%d", portNum)
	fmt.Printf("Starting orch-go API server on http://localhost%s\n", addr)
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
	fmt.Println("  GET /api/pending-reviews - Agents with unreviewed synthesis recommendations")
	fmt.Println("  POST /api/dismiss-review - Dismiss a specific recommendation")
	fmt.Println("  GET/PUT /api/config - User configuration settings")
	fmt.Println("  GET /api/changelog - Aggregated changelog (?days=7&project=all)")
	fmt.Println("  GET /health        - Health check")
	fmt.Println("\nPress Ctrl+C to stop")

	return http.ListenAndServe(addr, mux)
}

// AgentAPIResponse is the JSON structure returned by /api/agents.
type AgentAPIResponse struct {
	ID           string             `json:"id"`
	SessionID    string             `json:"session_id,omitempty"`
	BeadsID      string             `json:"beads_id,omitempty"`
	BeadsTitle   string             `json:"beads_title,omitempty"`
	Skill        string             `json:"skill,omitempty"`
	Status       string             `json:"status"`            // "active", "idle", "completed", etc.
	Phase        string             `json:"phase,omitempty"`   // "Planning", "Implementing", "Complete", etc.
	Task         string             `json:"task,omitempty"`    // Task description from beads issue
	Project      string             `json:"project,omitempty"` // Project name (orch-go, skillc, etc.)
	Runtime      string             `json:"runtime,omitempty"`
	Window       string             `json:"window,omitempty"`
	IsProcessing bool               `json:"is_processing,omitempty"` // True if actively generating response
	SpawnedAt    string             `json:"spawned_at,omitempty"`    // ISO 8601 timestamp
	UpdatedAt    string             `json:"updated_at,omitempty"`    // ISO 8601 timestamp
	Synthesis    *SynthesisResponse `json:"synthesis,omitempty"`
	CloseReason  string             `json:"close_reason,omitempty"` // Beads close reason, fallback when synthesis is null
	GapAnalysis  *GapAPIResponse    `json:"gap_analysis,omitempty"` // Context gap analysis from spawn time
	Tokens       *opencode.TokenStats `json:"tokens,omitempty"`      // Token usage for the session
}

// GapAPIResponse represents gap analysis data for the API.
type GapAPIResponse struct {
	HasGaps        bool `json:"has_gaps"`
	ContextQuality int  `json:"context_quality"`
	ShouldWarn     bool `json:"should_warn"`
	MatchCount     int  `json:"match_count,omitempty"`
	Constraints    int  `json:"constraints,omitempty"`
	Decisions      int  `json:"decisions,omitempty"`
	Investigations int  `json:"investigations,omitempty"`
}

// SynthesisResponse is a condensed version of verify.Synthesis for the API.
// Uses the D.E.K.N. structure: Delta, Evidence, Knowledge, Next.
type SynthesisResponse struct {
	// Header fields
	TLDR           string `json:"tldr,omitempty"`
	Outcome        string `json:"outcome,omitempty"`        // success, partial, blocked, failed
	Recommendation string `json:"recommendation,omitempty"` // close, continue, escalate

	// Condensed sections
	DeltaSummary string   `json:"delta_summary,omitempty"` // e.g., "3 files created, 2 modified, 5 commits"
	NextActions  []string `json:"next_actions,omitempty"`  // Follow-up items
}

// workspaceCache stores pre-computed workspace metadata to avoid repeated directory scans.
// Built once per request and used for all lookups within that request.
type workspaceCache struct {
	// beadsToWorkspace maps beadsID → workspace path (absolute)
	beadsToWorkspace map[string]string
	// beadsToProjectDir maps beadsID → PROJECT_DIR from SPAWN_CONTEXT.md
	beadsToProjectDir map[string]string
	// workspaceEntries stores directory entries for reuse
	workspaceEntries []os.DirEntry
	// workspaceDir is the base workspace directory path
	workspaceDir string
	// workspaceEntryToPath maps directory entry name → absolute workspace path
	// This is needed for multi-project scenarios where entries come from different projects
	workspaceEntryToPath map[string]string
}

// extractUniqueProjectDirs collects unique project directories from OpenCode sessions.
// Returns a deduplicated slice of directory paths that have active agents.
// This enables multi-project workspace aggregation for cross-project agent visibility.
func extractUniqueProjectDirs(sessions []opencode.Session, currentProjectDir string) []string {
	seen := make(map[string]bool)
	var dirs []string

	// Always include current project directory
	if currentProjectDir != "" {
		seen[currentProjectDir] = true
		dirs = append(dirs, currentProjectDir)
	}

	// Add unique directories from sessions
	for _, s := range sessions {
		dir := s.Directory
		if dir == "" {
			continue
		}

		// Normalize path (resolve any symlinks, clean path)
		dir = filepath.Clean(dir)

		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}

	return dirs
}

// buildMultiProjectWorkspaceCache builds workspace caches for multiple project directories
// and merges them into a unified cache. Scans in parallel for performance.
// This enables cross-project agent visibility by aggregating workspace metadata
// from all projects with active OpenCode sessions.
func buildMultiProjectWorkspaceCache(projectDirs []string) *workspaceCache {
	if len(projectDirs) == 0 {
		return &workspaceCache{
			beadsToWorkspace:  make(map[string]string),
			beadsToProjectDir: make(map[string]string),
		}
	}

	// If only one project directory, use the simpler single-project scan
	if len(projectDirs) == 1 {
		return buildWorkspaceCache(projectDirs[0])
	}

	// Build caches in parallel using goroutines
	type cacheResult struct {
		cache *workspaceCache
	}
	results := make(chan cacheResult, len(projectDirs))

	for _, dir := range projectDirs {
		go func(projectDir string) {
			cache := buildWorkspaceCache(projectDir)
			results <- cacheResult{cache: cache}
		}(dir)
	}

	// Merge all caches into a unified cache
	merged := &workspaceCache{
		beadsToWorkspace:     make(map[string]string),
		beadsToProjectDir:    make(map[string]string),
		workspaceEntryToPath: make(map[string]string),
	}

	for i := 0; i < len(projectDirs); i++ {
		result := <-results

		// Merge beadsToWorkspace map (later entries don't overwrite earlier ones)
		for beadsID, wsPath := range result.cache.beadsToWorkspace {
			if _, exists := merged.beadsToWorkspace[beadsID]; !exists {
				merged.beadsToWorkspace[beadsID] = wsPath
			}
		}

		// Merge beadsToProjectDir map
		for beadsID, projDir := range result.cache.beadsToProjectDir {
			if _, exists := merged.beadsToProjectDir[beadsID]; !exists {
				merged.beadsToProjectDir[beadsID] = projDir
			}
		}

		// Merge workspaceEntryToPath map (for multi-project workspace path resolution)
		for entryName, wsPath := range result.cache.workspaceEntryToPath {
			if _, exists := merged.workspaceEntryToPath[entryName]; !exists {
				merged.workspaceEntryToPath[entryName] = wsPath
			}
		}

		// Merge workspace entries (for completed workspace scanning)
		merged.workspaceEntries = append(merged.workspaceEntries, result.cache.workspaceEntries...)

		// Keep track of workspace dir for backward compatibility
		// (use first non-empty workspace dir)
		if merged.workspaceDir == "" && result.cache.workspaceDir != "" {
			merged.workspaceDir = result.cache.workspaceDir
		}
	}

	return merged
}

// buildWorkspaceCache scans the workspace directory once and builds lookup maps.
// This replaces multiple calls to findWorkspaceByBeadsID which each scanned all 400+ directories.
func buildWorkspaceCache(projectDir string) *workspaceCache {
	cache := &workspaceCache{
		beadsToWorkspace:     make(map[string]string),
		beadsToProjectDir:    make(map[string]string),
		workspaceDir:         filepath.Join(projectDir, ".orch", "workspace"),
		workspaceEntryToPath: make(map[string]string),
	}

	entries, err := os.ReadDir(cache.workspaceDir)
	if err != nil {
		return cache // Empty cache if directory doesn't exist
	}
	cache.workspaceEntries = entries

	// Single scan: extract beads ID and project dir from each workspace
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(cache.workspaceDir, dirName)
		spawnContextPath := filepath.Join(dirPath, "SPAWN_CONTEXT.md")

		// Store entry name to absolute path mapping for multi-project support
		cache.workspaceEntryToPath[dirName] = dirPath

		// Read SPAWN_CONTEXT.md once to extract both beads ID and PROJECT_DIR
		content, err := os.ReadFile(spawnContextPath)
		if err != nil {
			continue // Skip workspaces without SPAWN_CONTEXT.md
		}
		contentStr := string(content)

		var beadsID, agentProjectDir string

		// Parse once, extracting both pieces of info
		for _, line := range strings.Split(contentStr, "\n") {
			lineTrimmed := strings.TrimSpace(line)

			// Extract beads ID from "spawned from beads issue: **xxx**" or "bd comment xxx"
			if strings.Contains(strings.ToLower(line), "spawned from beads issue:") {
				// Pattern: "spawned from beads issue: **orch-go-xxxx**"
				// Extract the beads ID between ** markers or after the colon
				if idx := strings.Index(line, "**"); idx != -1 {
					rest := line[idx+2:]
					if endIdx := strings.Index(rest, "**"); endIdx != -1 {
						beadsID = rest[:endIdx]
					}
				}
			} else if strings.HasPrefix(lineTrimmed, "bd comment ") {
				// Pattern: "bd comment orch-go-xxxx ..."
				parts := strings.Fields(lineTrimmed)
				if len(parts) >= 3 {
					beadsID = parts[2]
				}
			}

			// Extract PROJECT_DIR
			if strings.HasPrefix(lineTrimmed, "PROJECT_DIR:") {
				agentProjectDir = strings.TrimSpace(strings.TrimPrefix(lineTrimmed, "PROJECT_DIR:"))
			}
		}

		// Store in cache if beads ID found
		if beadsID != "" {
			cache.beadsToWorkspace[beadsID] = dirPath
			if agentProjectDir != "" {
				cache.beadsToProjectDir[beadsID] = agentProjectDir
			}
		}
	}

	return cache
}

// lookupWorkspace returns the workspace path for a beads ID (O(1) lookup).
func (c *workspaceCache) lookupWorkspace(beadsID string) string {
	return c.beadsToWorkspace[beadsID]
}

// lookupProjectDir returns the PROJECT_DIR for a beads ID (O(1) lookup).
func (c *workspaceCache) lookupProjectDir(beadsID string) string {
	return c.beadsToProjectDir[beadsID]
}

// lookupWorkspacePathByEntry returns the absolute workspace path for a directory entry name.
// This is used in multi-project scenarios where workspace entries come from different projects.
func (c *workspaceCache) lookupWorkspacePathByEntry(entryName string) string {
	if path, ok := c.workspaceEntryToPath[entryName]; ok {
		return path
	}
	// Fallback to single-project path construction
	if c.workspaceDir != "" {
		return filepath.Join(c.workspaceDir, entryName)
	}
	return ""
}

// handleAgents returns JSON list of active agents from OpenCode/tmux and completed workspaces.
func handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use sourceDir (set at build time) since serve may run from any working directory
	projectDir := sourceDir
	if projectDir == "" || projectDir == "unknown" {
		projectDir, _ = os.Getwd()
	}

	client := opencode.NewClient(serverURL)

	// Get active sessions from OpenCode
	// Don't filter by directory - show all sessions across all projects
	// (serve process CWD may not match project directory)
	sessions, err := client.ListSessions("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list sessions: %v", err), http.StatusInternalServerError)
		return
	}

	// Build multi-project workspace cache for cross-project agent visibility.
	// This aggregates workspace metadata from all projects with active sessions,
	// enabling the dashboard to show correct status for agents spawned with --workdir.
	// Previously: Only scanned current project's .orch/workspace/
	// Now: Scans all unique project directories found in OpenCode sessions
	projectDirs := extractUniqueProjectDirs(sessions, projectDir)
	wsCache := buildMultiProjectWorkspaceCache(projectDirs)

	now := time.Now()
	agents := []AgentAPIResponse{} // Initialize as empty slice, not nil, to return [] instead of null

	// Collect beads IDs for batch fetching
	var beadsIDsToFetch []string
	seenBeadsIDs := make(map[string]bool)

	// Track project directories for cross-project agents
	// Key: beadsID, Value: projectDir from workspace SPAWN_CONTEXT.md
	beadsProjectDirs := make(map[string]string)

	// Add active sessions from OpenCode
	// Filter: only show sessions updated in the last 10 minutes as "active"
	// Sessions idle > 30 min are filtered out AFTER checking beads Phase status
	// (completed agents should still be shown regardless of activity time)
	activeThreshold := 10 * time.Minute
	displayThreshold := 30 * time.Minute

	// Track which agents need post-filtering by beads ID (idle > displayThreshold)
	// These will be filtered out after Phase check unless Phase: Complete
	pendingFilterByBeadsID := make(map[string]bool)

	// Track agents by title to deduplicate (OpenCode can have multiple sessions with same title)
	// Keep the most recently updated session for each title
	seenTitles := make(map[string]int) // title -> index in agents slice

	for _, s := range sessions {
		createdAt := time.Unix(s.Time.Created/1000, 0)
		updatedAt := time.Unix(s.Time.Updated/1000, 0)
		runtime := now.Sub(createdAt)
		timeSinceUpdate := now.Sub(updatedAt)

		// Determine status based on recent activity
		status := "active"
		if timeSinceUpdate > activeThreshold {
			status = "idle" // Session exists but hasn't had recent activity
		}

		// NOTE: IsProcessing is now populated client-side via SSE session.status events.
		// Previously we called client.IsSessionProcessing(s.ID) here, but that makes
		// an HTTP call per session which caused 125% CPU when dashboard polled frequently.
		// The frontend already receives busy/idle state from OpenCode SSE and updates
		// is_processing in real-time, so we don't need to fetch it here.

		agent := AgentAPIResponse{
			ID:           s.Title,
			SessionID:    s.ID,
			Status:       status,
			Runtime:      formatDuration(runtime),
			SpawnedAt:    createdAt.Format(time.RFC3339),
			UpdatedAt:    updatedAt.Format(time.RFC3339),
			IsProcessing: false, // Populated client-side via SSE
		}

		// Derive beadsID and skill from session title
		if s.Title != "" {
			agent.BeadsID = extractBeadsIDFromTitle(s.Title)
			agent.Skill = extractSkillFromTitle(s.Title)
			agent.Project = extractProjectFromBeadsID(agent.BeadsID)

			// Collect beads ID for batch fetch
			if agent.BeadsID != "" && !seenBeadsIDs[agent.BeadsID] {
				beadsIDsToFetch = append(beadsIDsToFetch, agent.BeadsID)
				seenBeadsIDs[agent.BeadsID] = true

				// For cross-project agent visibility: use cached PROJECT_DIR
				// This replaces expensive directory scanning with O(1) lookup
				if agentProjectDir := wsCache.lookupProjectDir(agent.BeadsID); agentProjectDir != "" {
					beadsProjectDirs[agent.BeadsID] = agentProjectDir
				}
			}
		}

		// Only include sessions that were spawned via orch spawn (have beads ID)
		// This filters out interactive/ad-hoc OpenCode sessions
		if agent.BeadsID == "" {
			continue
		}

		// Track if this agent should be filtered after Phase check
		// Don't filter yet - we need to check beads Phase: Complete first
		if status == "idle" && timeSinceUpdate > displayThreshold {
			pendingFilterByBeadsID[agent.BeadsID] = true
		}

		// Deduplicate by title - keep the most recently updated session
		// OpenCode can have multiple sessions with the same title (e.g., resumed agents)
		if existingIdx, exists := seenTitles[s.Title]; exists {
			// Compare updated_at to keep the more recent session
			existingUpdatedAt, _ := time.Parse(time.RFC3339, agents[existingIdx].UpdatedAt)
			if updatedAt.After(existingUpdatedAt) {
				// Replace the existing agent with this newer one
				agents[existingIdx] = agent
			}
			// Skip appending since we either replaced or kept the existing one
			continue
		}

		seenTitles[s.Title] = len(agents)
		agents = append(agents, agent)
	}

	// Add tmux-only agents
	workersSessions, _ := tmux.ListWorkersSessions()
	for _, sessionName := range workersSessions {
		windows, _ := tmux.ListWindows(sessionName)
		for _, win := range windows {
			if win.Name == "servers" || win.Name == "zsh" {
				continue
			}

			beadsID := extractBeadsIDFromWindowName(win.Name)
			skill := extractSkillFromWindowName(win.Name)
			project := extractProjectFromBeadsID(beadsID)

			// Check if already in agents list
			alreadyIn := false
			for _, a := range agents {
				if (beadsID != "" && a.BeadsID == beadsID) || (a.ID != "" && strings.Contains(win.Name, a.ID)) {
					alreadyIn = true
					break
				}
			}

			if !alreadyIn {
				agents = append(agents, AgentAPIResponse{
					ID:      win.Name,
					BeadsID: beadsID,
					Skill:   skill,
					Project: project,
					Status:  "active",
					Window:  win.Target,
				})

				// Collect beads ID for batch fetch
				if beadsID != "" && !seenBeadsIDs[beadsID] {
					beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
					seenBeadsIDs[beadsID] = true

					// For cross-project agent visibility: use cached PROJECT_DIR
					// This replaces expensive directory scanning with O(1) lookup
					if agentProjectDir := wsCache.lookupProjectDir(beadsID); agentProjectDir != "" {
						beadsProjectDirs[beadsID] = agentProjectDir
					}
				}
			}
		}
	}

	// Add completed workspaces (those with SYNTHESIS.md or light-tier completions)
	// Reuse cached workspace entries to avoid redundant directory reads
	// Multi-project support: entries may come from different project workspace directories
	if len(wsCache.workspaceEntries) > 0 {
		for _, entry := range wsCache.workspaceEntries {
			if !entry.IsDir() {
				continue
			}

			// Check if already in active list
			// Active session IDs have format "workspace [beads-id]", workspace names don't
			alreadyIn := false
			workspaceName := entry.Name()
			for _, a := range agents {
				if a.ID == workspaceName || strings.HasPrefix(a.ID, workspaceName+" ") {
					alreadyIn = true
					break
				}
			}

			if alreadyIn {
				continue
			}

			// Use the lookup method for multi-project support
			workspacePath := wsCache.lookupWorkspacePathByEntry(entry.Name())
			synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
			hasSynthesis := false

			// Check if SYNTHESIS.md exists (indicates full-tier completion)
			if _, err := os.Stat(synthesisPath); err == nil {
				hasSynthesis = true
			}

			// Only add workspaces that have SYNTHESIS.md for now
			// Light-tier completions will be detected via Phase: Complete in beads comments
			if !hasSynthesis {
				// For light-tier, check if there's a SPAWN_CONTEXT.md (indicates it's a valid spawn)
				spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
				if _, err := os.Stat(spawnContextPath); err != nil {
					continue // Not a valid spawn workspace
				}
			}

			agent := AgentAPIResponse{
				ID:     entry.Name(),
				Status: "completed",
			}

			// Set updated_at from workspace name date suffix or file modification time
			// This ensures proper sorting in archive section
			if parsedDate := extractDateFromWorkspaceName(entry.Name()); !parsedDate.IsZero() {
				agent.UpdatedAt = parsedDate.Format(time.RFC3339)
			} else if hasSynthesis {
				// Fallback to file modification time of SYNTHESIS.md
				if info, err := os.Stat(synthesisPath); err == nil {
					agent.UpdatedAt = info.ModTime().Format(time.RFC3339)
				}
			} else {
				// For light-tier, use SPAWN_CONTEXT.md modification time
				spawnContextPath := filepath.Join(workspacePath, "SPAWN_CONTEXT.md")
				if info, err := os.Stat(spawnContextPath); err == nil {
					agent.UpdatedAt = info.ModTime().Format(time.RFC3339)
				}
			}

			// Read session ID from workspace
			if sessionID := spawn.ReadSessionID(workspacePath); sessionID != "" {
				agent.SessionID = sessionID
			}

			// Parse synthesis (only for full-tier)
			if hasSynthesis {
				if synthesis, err := verify.ParseSynthesis(workspacePath); err == nil {
					agent.Synthesis = &SynthesisResponse{
						TLDR:           synthesis.TLDR,
						Outcome:        synthesis.Outcome,
						Recommendation: synthesis.Recommendation,
						DeltaSummary:   summarizeDelta(synthesis.Delta),
						NextActions:    synthesis.NextActions,
					}
				}
			}

			// Extract beadsID from workspace SPAWN_CONTEXT.md (more reliable than parsing name)
			agent.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
			// Fallback to extracting from workspace name if SPAWN_CONTEXT.md doesn't have it
			if agent.BeadsID == "" {
				agent.BeadsID = extractBeadsIDFromTitle(entry.Name())
			}
			agent.Skill = extractSkillFromTitle(entry.Name())

			// Extract PROJECT_DIR from workspace for cross-project agent visibility
			// This allows fetching beads comments from the correct project's database
			agentProjectDir := extractProjectDirFromWorkspace(workspacePath)

			// Collect beads ID for batch fetch (to get close_reason for light-tier)
			if agent.BeadsID != "" && !seenBeadsIDs[agent.BeadsID] {
				beadsIDsToFetch = append(beadsIDsToFetch, agent.BeadsID)
				seenBeadsIDs[agent.BeadsID] = true
				// Store project directory for cross-project comment fetching
				if agentProjectDir != "" {
					beadsProjectDirs[agent.BeadsID] = agentProjectDir
				}
			}

			agents = append(agents, agent)
		}
	}

	// Batch fetch beads data (phase from comments, task from issues, close_reason for completed)
	// This is the same pattern used by orch status for efficiency
	if len(beadsIDsToFetch) > 0 {
		// Fetch all open issues in one call
		openIssues, _ := verify.ListOpenIssues()

		// Batch fetch all issues (including closed) for close_reason
		// Uses bd show which works for any issue status
		allIssues, _ := verify.GetIssuesBatch(beadsIDsToFetch)

		// Batch fetch comments for all beads IDs
		// Use project-aware batch fetch for cross-project agent visibility
		commentsMap := verify.GetCommentsBatchWithProjectDirs(beadsIDsToFetch, beadsProjectDirs)

		// Populate phase, task, and close_reason for each agent
		for i := range agents {
			if agents[i].BeadsID == "" {
				continue
			}

			// Get task from open issue title first
			if issue, ok := openIssues[agents[i].BeadsID]; ok {
				agents[i].Task = truncate(issue.Title, 60)
			}

			// If not in open issues, try all issues (for closed ones)
			if agents[i].Task == "" {
				if issue, ok := allIssues[agents[i].BeadsID]; ok {
					agents[i].Task = truncate(issue.Title, 60)
					// For completed agents without synthesis, use close_reason as fallback
					if agents[i].Synthesis == nil && issue.CloseReason != "" {
						agents[i].CloseReason = issue.CloseReason
					}
				}
			}

			// Get phase from comments
			if comments, ok := commentsMap[agents[i].BeadsID]; ok {
				phaseStatus := verify.ParsePhaseFromComments(comments)
				if phaseStatus.Found {
					agents[i].Phase = phaseStatus.Phase
					// Update status to completed if phase is Complete.
					// Phase: Complete is the definitive signal that the agent's work is done,
					// regardless of whether the OpenCode session is still open. An open session
					// just means the agent hasn't called /exit yet, but the work is complete.
					// If an agent is resumed after Phase: Complete, a new Phase comment
					// (e.g., "Phase: Implementing") would supersede this.
					if strings.EqualFold(phaseStatus.Phase, "Complete") {
						agents[i].Status = "completed"
					}
				}
			}

			// For agents not yet marked completed, check workspace for SYNTHESIS.md
			// This handles untracked agents (--no-track) which have fake beads IDs
			// and won't have Phase: Complete in beads comments.
			// SYNTHESIS.md presence is a definitive signal that the agent completed,
			// regardless of whether the OpenCode session is still open.
			if agents[i].Status != "completed" {
				// Use cached workspace lookup instead of scanning directories
				workspacePath := wsCache.lookupWorkspace(agents[i].BeadsID)
				if checkWorkspaceSynthesis(workspacePath) {
					agents[i].Status = "completed"
				}
			}

			// For completed agents, also check close_reason if synthesis is null
			if agents[i].Status == "completed" && agents[i].Synthesis == nil && agents[i].CloseReason == "" {
				if issue, ok := allIssues[agents[i].BeadsID]; ok && issue.CloseReason != "" {
					agents[i].CloseReason = issue.CloseReason
				}
			}
		}

		// Fetch gap analysis from spawn events for each agent
		gapAnalysisMap := getGapAnalysisFromEvents(beadsIDsToFetch)
		for i := range agents {
			if agents[i].BeadsID == "" {
				continue
			}
			if gapData, ok := gapAnalysisMap[agents[i].BeadsID]; ok {
				agents[i].GapAnalysis = gapData
			}
		}

		// Post-Phase filtering: remove agents that were idle > displayThreshold
		// and are NOT Phase: Complete. This deferred filtering ensures completed
		// agents are shown regardless of activity time.
		filtered := make([]AgentAPIResponse, 0, len(agents))
		for _, agent := range agents {
			if pendingFilterByBeadsID[agent.BeadsID] && agent.Status != "completed" {
				// Skip idle agents that are not completed
				continue
			}
			filtered = append(filtered, agent)
		}
		agents = filtered
	}

	// Fetch token usage for agents with valid session IDs
	// Parallelized to avoid sequential HTTP calls causing ~20s delays with 200+ agents.
	// Uses goroutines with semaphore to limit concurrent requests.
	type tokenResult struct {
		index  int
		tokens *opencode.TokenStats
	}
	tokenChan := make(chan tokenResult, len(agents))
	
	// Limit concurrent HTTP requests to avoid overwhelming the OpenCode server
	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)
	
	var wg sync.WaitGroup
	for i := range agents {
		// Skip agents without session ID or completed agents (token data is static for completed)
		if agents[i].SessionID == "" || agents[i].Status == "completed" {
			continue
		}
		
		wg.Add(1)
		go func(idx int, sessionID string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore
			
			tokens, err := client.GetSessionTokens(sessionID)
			if err == nil && tokens != nil {
				tokenChan <- tokenResult{index: idx, tokens: tokens}
			}
		}(i, agents[i].SessionID)
	}
	
	// Wait for all goroutines to complete, then close channel
	go func() {
		wg.Wait()
		close(tokenChan)
	}()
	
	// Collect results
	for result := range tokenChan {
		agents[result.index].Tokens = result.tokens
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode agents: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleEvents proxies the OpenCode SSE stream to the client.
// It connects to http://localhost:4096/event and forwards events.
func handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Connect to OpenCode SSE stream
	opencodeURL := serverURL + "/event"
	resp, err := http.Get(opencodeURL)
	if err != nil {
		// Send error as SSE event
		fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Failed to connect to OpenCode: %s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer resp.Body.Close()

	// Check if OpenCode returned an error
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(w, "event: error\ndata: {\"error\": \"OpenCode returned status %d\"}\n\n", resp.StatusCode)
		flusher.Flush()
		return
	}

	// Send connected event
	fmt.Fprintf(w, "event: connected\ndata: {\"source\": \"%s\"}\n\n", opencodeURL)
	flusher.Flush()

	// Create a done channel to handle client disconnect
	ctx := r.Context()

	// Read and forward SSE events
	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Connection closed by OpenCode
					fmt.Fprintf(w, "event: disconnected\ndata: {\"reason\": \"upstream closed\"}\n\n")
					flusher.Flush()
					return
				}
				// Read error
				fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Read error: %s\"}\n\n", err.Error())
				flusher.Flush()
				return
			}

			// Forward the line as-is (preserves SSE format)
			if strings.HasPrefix(line, "data:") {
				fmt.Printf("Forwarding SSE event: %s", line)
			}
			fmt.Fprint(w, line)
			flusher.Flush()
		}
	}
}

// handleAgentlog returns agent lifecycle events from ~/.orch/events.jsonl.
// Without query params: returns last 100 events as JSON array.
// With ?follow=true: streams new events via SSE.
func handleAgentlog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	follow := r.URL.Query().Get("follow") == "true"

	if follow {
		handleAgentlogSSE(w, r)
	} else {
		handleAgentlogJSON(w, r)
	}
}

// handleAgentlogJSON returns the last 100 events as JSON array.
func handleAgentlogJSON(w http.ResponseWriter, r *http.Request) {
	logPath := events.DefaultLogPath()

	eventList, err := readLastNEvents(logPath, 100)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty array if file doesn't exist yet
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(eventList); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode events: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleAgentlogSSE streams new events via SSE as they are appended to events.jsonl.
func handleAgentlogSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	logPath := events.DefaultLogPath()
	ctx := r.Context()

	// Open file for reading
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Send connected event, file doesn't exist yet
			fmt.Fprintf(w, "event: connected\ndata: {\"source\": \"%s\", \"status\": \"waiting\"}\n\n", logPath)
			flusher.Flush()
		} else {
			fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Failed to open log file: %s\"}\n\n", err.Error())
			flusher.Flush()
			return
		}
	} else {
		defer file.Close()
	}

	// Send connected event
	fmt.Fprintf(w, "event: connected\ndata: {\"source\": \"%s\"}\n\n", logPath)
	flusher.Flush()

	// Seek to end of file to only stream new events
	if file != nil {
		file.Seek(0, io.SeekEnd)
	}

	// Poll for new events
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	reader := bufio.NewReader(file)
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		case <-ticker.C:
			// Try to read new lines
			if file == nil {
				// Try to open file if it didn't exist before
				file, err = os.Open(logPath)
				if err != nil {
					continue // File still doesn't exist
				}
				file.Seek(0, io.SeekEnd)
				reader = bufio.NewReader(file)
			}

			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						break // No more data, wait for next poll
					}
					fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Read error: %s\"}\n\n", err.Error())
					flusher.Flush()
					return
				}

				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// Validate it's valid JSON and forward as SSE event
				var event events.Event
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					continue // Skip invalid lines
				}

				fmt.Fprintf(w, "event: agentlog\ndata: %s\n\n", line)
				flusher.Flush()
			}
		}
	}
}

// readLastNEvents reads the last n events from a JSONL file.
func readLastNEvents(path string, n int) ([]events.Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var allEvents []events.Event
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip invalid lines
		}
		allEvents = append(allEvents, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last n events
	if len(allEvents) > n {
		return allEvents[len(allEvents)-n:], nil
	}
	return allEvents, nil
}

// getProjectAPIPort returns the allocated API port for the current project.
// Returns 0 if no allocation exists or on error.
func getProjectAPIPort() int {
	projectDir, err := os.Getwd()
	if err != nil {
		return 0
	}
	projectName := filepath.Base(projectDir)

	registry, err := port.New("")
	if err != nil {
		return 0
	}

	alloc := registry.Find(projectName, "api")
	if alloc == nil {
		return 0
	}

	return alloc.Port
}

// UsageAPIResponse is the JSON structure returned by /api/usage.
type UsageAPIResponse struct {
	Account         string  `json:"account"`                       // Account email
	AccountName     string  `json:"account_name,omitempty"`        // Account name from accounts.yaml (e.g., "personal", "work")
	FiveHour        float64 `json:"five_hour_percent"`             // 5-hour session usage %
	FiveHourReset   string  `json:"five_hour_reset,omitempty"`     // Human-readable time until 5-hour reset
	Weekly          float64 `json:"weekly_percent"`                // 7-day weekly usage %
	WeeklyReset     string  `json:"weekly_reset,omitempty"`        // Human-readable time until weekly reset
	WeeklyOpus      float64 `json:"weekly_opus_percent,omitempty"` // 7-day Opus-specific usage %
	WeeklyOpusReset string  `json:"weekly_opus_reset,omitempty"`   // Human-readable time until Opus weekly reset
	Error           string  `json:"error,omitempty"`               // Error message if any
}

// handleUsage returns Claude Max usage stats.
func handleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := usage.FetchUsage()

	resp := UsageAPIResponse{}

	if info.Error != "" {
		resp.Error = info.Error
	} else {
		resp.Account = info.Email
		resp.AccountName = lookupAccountName(info.Email)
		if info.FiveHour != nil {
			resp.FiveHour = info.FiveHour.Utilization
			resp.FiveHourReset = info.FiveHour.TimeUntilReset()
		}
		if info.SevenDay != nil {
			resp.Weekly = info.SevenDay.Utilization
			resp.WeeklyReset = info.SevenDay.TimeUntilReset()
		}
		if info.SevenDayOpus != nil {
			resp.WeeklyOpus = info.SevenDayOpus.Utilization
			resp.WeeklyOpusReset = info.SevenDayOpus.TimeUntilReset()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode usage: %v", err), http.StatusInternalServerError)
		return
	}
}

// lookupAccountName finds the account name from ~/.orch/accounts.yaml by matching email.
// Returns the account name (e.g., "personal", "work") if found, empty string otherwise.
func lookupAccountName(email string) string {
	if email == "" {
		return ""
	}

	cfg, err := account.LoadConfig()
	if err != nil {
		return ""
	}

	// Find account by matching email
	for name, acc := range cfg.Accounts {
		if acc.Email == email {
			return name
		}
	}

	return ""
}

// FocusAPIResponse is the JSON structure returned by /api/focus.
type FocusAPIResponse struct {
	Goal       string `json:"goal,omitempty"`
	BeadsID    string `json:"beads_id,omitempty"`
	SetAt      string `json:"set_at,omitempty"`
	IsDrifting bool   `json:"is_drifting"`
	HasFocus   bool   `json:"has_focus"`
}

// handleFocus returns current focus and drift status.
func handleFocus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	store, err := focus.New("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load focus: %v", err), http.StatusInternalServerError)
		return
	}

	resp := FocusAPIResponse{}

	f := store.Get()
	if f != nil {
		resp.HasFocus = true
		resp.Goal = f.Goal
		resp.BeadsID = f.BeadsID
		resp.SetAt = f.SetAt

		// Check drift by getting active agents from current sessions
		client := opencode.NewClient(serverURL)
		sessions, _ := client.ListSessions("")

		var activeIssues []string
		for _, s := range sessions {
			if beadsID := extractBeadsIDFromTitle(s.Title); beadsID != "" {
				activeIssues = append(activeIssues, beadsID)
			}
		}

		drift := store.CheckDrift(activeIssues)
		resp.IsDrifting = drift.IsDrifting
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode focus: %v", err), http.StatusInternalServerError)
		return
	}
}

// BeadsAPIResponse is the JSON structure returned by /api/beads.
type BeadsAPIResponse struct {
	TotalIssues    int     `json:"total_issues"`
	OpenIssues     int     `json:"open_issues"`
	InProgress     int     `json:"in_progress_issues"`
	BlockedIssues  int     `json:"blocked_issues"`
	ReadyIssues    int     `json:"ready_issues"`
	ClosedIssues   int     `json:"closed_issues"`
	AvgLeadTimeHrs float64 `json:"avg_lead_time_hours,omitempty"`
	Error          string  `json:"error,omitempty"`
}

// handleBeads returns beads stats by shelling out to `bd stats --json`.
func handleBeads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use persistent RPC client (with auto-reconnect), fallback to CLI if unavailable
	var stats *beads.Stats
	var err error

	if beadsClient != nil {
		stats, err = beadsClient.Stats()
		if err != nil {
			// Fall through to CLI fallback on RPC error
			stats, err = beads.FallbackStats()
		}
	} else {
		stats, err = beads.FallbackStats()
	}

	if err != nil {
		resp := BeadsAPIResponse{Error: fmt.Sprintf("Failed to get bd stats: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := BeadsAPIResponse{
		TotalIssues:    stats.Summary.TotalIssues,
		OpenIssues:     stats.Summary.OpenIssues,
		InProgress:     stats.Summary.InProgressIssues,
		BlockedIssues:  stats.Summary.BlockedIssues,
		ReadyIssues:    stats.Summary.ReadyIssues,
		ClosedIssues:   stats.Summary.ClosedIssues,
		AvgLeadTimeHrs: stats.Summary.AvgLeadTimeHours,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode beads: %v", err), http.StatusInternalServerError)
		return
	}
}

// ReadyIssueResponse represents a ready issue for the dashboard queue.
type ReadyIssueResponse struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Priority  int      `json:"priority"`
	IssueType string   `json:"issue_type"`
	Labels    []string `json:"labels,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
}

// BeadsReadyAPIResponse is the JSON structure returned by /api/beads/ready.
type BeadsReadyAPIResponse struct {
	Issues []ReadyIssueResponse `json:"issues"`
	Count  int                  `json:"count"`
	Error  string               `json:"error,omitempty"`
}

// handleBeadsReady returns list of ready issues for dashboard queue visibility.
func handleBeadsReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use persistent RPC client (with auto-reconnect), fallback to CLI if unavailable
	var issues []beads.Issue
	var err error

	if beadsClient != nil {
		issues, err = beadsClient.Ready(nil)
		if err != nil {
			// Fall through to CLI fallback on RPC error
			issues, err = beads.FallbackReady()
		}
	} else {
		issues, err = beads.FallbackReady()
	}

	if err != nil {
		resp := BeadsReadyAPIResponse{
			Issues: []ReadyIssueResponse{},
			Count:  0,
			Error:  fmt.Sprintf("Failed to get ready issues: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert beads.Issue to ReadyIssueResponse
	readyIssues := make([]ReadyIssueResponse, 0, len(issues))
	for _, issue := range issues {
		readyIssues = append(readyIssues, ReadyIssueResponse{
			ID:        issue.ID,
			Title:     issue.Title,
			Priority:  issue.Priority,
			IssueType: issue.IssueType,
			Labels:    issue.Labels,
			CreatedAt: issue.CreatedAt,
		})
	}

	resp := BeadsReadyAPIResponse{
		Issues: readyIssues,
		Count:  len(readyIssues),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode beads ready: %v", err), http.StatusInternalServerError)
		return
	}
}

// ServerPortInfo represents a port allocation for a server.
type ServerPortInfo struct {
	Service string `json:"service"`
	Port    int    `json:"port"`
	Purpose string `json:"purpose,omitempty"`
}

// ServerProjectInfo represents a project's server status.
type ServerProjectInfo struct {
	Project string           `json:"project"`
	Ports   []ServerPortInfo `json:"ports"`
	Running bool             `json:"running"`
	Session string           `json:"session,omitempty"` // tmux session name
}

// ServersAPIResponse is the JSON structure returned by /api/servers.
type ServersAPIResponse struct {
	Projects     []ServerProjectInfo `json:"projects"`
	TotalCount   int                 `json:"total_count"`
	RunningCount int                 `json:"running_count"`
	StoppedCount int                 `json:"stopped_count"`
}

// handleServers returns servers status across projects.
func handleServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load port registry
	reg, err := port.New("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load port registry: %v", err), http.StatusInternalServerError)
		return
	}

	allocs := reg.List()

	// Group allocations by project
	projectMap := make(map[string][]port.Allocation)
	for _, alloc := range allocs {
		projectMap[alloc.Project] = append(projectMap[alloc.Project], alloc)
	}

	// Get list of running workers sessions
	runningSessions, _ := tmux.ListWorkersSessions()
	runningSessionSet := make(map[string]bool)
	for _, sess := range runningSessions {
		runningSessionSet[sess] = true
	}

	// Build project info list
	var projects []ServerProjectInfo
	runningCount := 0

	for projectName, projectPorts := range projectMap {
		sessionName := tmux.GetWorkersSessionName(projectName)
		running := runningSessionSet[sessionName]

		// Convert port allocations
		var ports []ServerPortInfo
		for _, p := range projectPorts {
			ports = append(ports, ServerPortInfo{
				Service: p.Service,
				Port:    p.Port,
				Purpose: p.Purpose,
			})
		}

		projects = append(projects, ServerProjectInfo{
			Project: projectName,
			Ports:   ports,
			Running: running,
			Session: sessionName,
		})

		if running {
			runningCount++
		}
	}

	resp := ServersAPIResponse{
		Projects:     projects,
		TotalCount:   len(projects),
		RunningCount: runningCount,
		StoppedCount: len(projects) - runningCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode servers: %v", err), http.StatusInternalServerError)
		return
	}
}

// DaemonAPIResponse is the JSON structure returned by /api/daemon.
type DaemonAPIResponse struct {
	Running       bool   `json:"running"`                   // Whether the daemon is currently running
	Status        string `json:"status,omitempty"`          // "running", "stalled", or empty if not running
	LastPoll      string `json:"last_poll,omitempty"`       // ISO 8601 timestamp of last poll
	LastPollAgo   string `json:"last_poll_ago,omitempty"`   // Human-readable time since last poll
	LastSpawn     string `json:"last_spawn,omitempty"`      // ISO 8601 timestamp of last spawn
	LastSpawnAgo  string `json:"last_spawn_ago,omitempty"`  // Human-readable time since last spawn
	ReadyCount    int    `json:"ready_count"`               // Number of issues ready to process
	CapacityMax   int    `json:"capacity_max"`              // Maximum concurrent agents
	CapacityUsed  int    `json:"capacity_used"`             // Currently active agents
	CapacityFree  int    `json:"capacity_free"`             // Available slots for spawning
	IssuesPerHour int    `json:"issues_per_hour,omitempty"` // Approximate processing rate (future)
}

// handleDaemon returns the daemon status from ~/.orch/daemon-status.json.
// If the daemon is not running (file doesn't exist), returns running: false.
func handleDaemon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := DaemonAPIResponse{
		Running: false,
	}

	// Try to read daemon status file
	status, err := daemon.ReadStatusFile()
	if err == nil && status != nil {
		resp.Running = true
		resp.Status = status.Status
		resp.ReadyCount = status.ReadyCount
		resp.CapacityMax = status.Capacity.Max
		resp.CapacityUsed = status.Capacity.Active
		resp.CapacityFree = status.Capacity.Available

		// Format timestamps
		if !status.LastPoll.IsZero() {
			resp.LastPoll = status.LastPoll.Format(time.RFC3339)
			resp.LastPollAgo = formatDurationAgo(time.Since(status.LastPoll))
		}
		if !status.LastSpawn.IsZero() {
			resp.LastSpawn = status.LastSpawn.Format(time.RFC3339)
			resp.LastSpawnAgo = formatDurationAgo(time.Since(status.LastSpawn))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode daemon status: %v", err), http.StatusInternalServerError)
		return
	}
}

// formatDurationAgo formats a duration into a human-readable "X ago" string.
func formatDurationAgo(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// CreateIssueRequest is the JSON request body for POST /api/issues.
type CreateIssueRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	IssueType   string   `json:"issue_type,omitempty"` // task, bug, etc.
	Priority    int      `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	ParentID    string   `json:"parent_id,omitempty"` // Optional parent issue for follow-ups
}

// CreateIssueResponse is the JSON response for POST /api/issues.
type CreateIssueResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handleIssues handles POST /api/issues - creates a new beads issue.
// This is used by the dashboard to create follow-up issues from synthesis recommendations.
func handleIssues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := CreateIssueResponse{Success: false, Error: fmt.Sprintf("Invalid request body: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Validate title
	if req.Title == "" {
		resp := CreateIssueResponse{Success: false, Error: "Title is required"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Use persistent RPC client (with auto-reconnect), fallback to CLI if unavailable
	var issue *beads.Issue
	var err error

	if beadsClient != nil {
		issue, err = beadsClient.Create(&beads.CreateArgs{
			Title:       req.Title,
			Description: req.Description,
			IssueType:   req.IssueType,
			Priority:    req.Priority,
			Labels:      req.Labels,
		})
		if err != nil {
			// Fall through to CLI fallback on RPC error
			issue, err = beads.FallbackCreate(req.Title, req.Description, req.IssueType, req.Priority, req.Labels)
		}
	} else {
		issue, err = beads.FallbackCreate(req.Title, req.Description, req.IssueType, req.Priority, req.Labels)
	}

	if err != nil {
		resp := CreateIssueResponse{Success: false, Error: fmt.Sprintf("Failed to create issue: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := CreateIssueResponse{
		ID:      issue.ID,
		Title:   issue.Title,
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// checkWorkspaceSynthesis checks if a workspace has a non-empty SYNTHESIS.md file.
// This is used to detect completion for untracked agents (--no-track) where
// there's no beads issue to check Phase: Complete.
func checkWorkspaceSynthesis(workspacePath string) bool {
	if workspacePath == "" {
		return false
	}
	synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")
	info, err := os.Stat(synthesisPath)
	if err != nil {
		return false
	}
	// SYNTHESIS.md must exist and be non-empty
	return info.Size() > 0
}

// getGapAnalysisFromEvents reads spawn events and extracts gap analysis data for given beads IDs.
// Returns a map of beads ID -> GapAPIResponse.
func getGapAnalysisFromEvents(beadsIDs []string) map[string]*GapAPIResponse {
	result := make(map[string]*GapAPIResponse)
	if len(beadsIDs) == 0 {
		return result
	}

	// Build a set of beads IDs for fast lookup
	beadsIDSet := make(map[string]bool)
	for _, id := range beadsIDs {
		beadsIDSet[id] = true
	}

	// Read events file
	logPath := events.DefaultLogPath()
	file, err := os.Open(logPath)
	if err != nil {
		return result
	}
	defer file.Close()

	// Scan events for spawn events matching our beads IDs
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		// Only process spawn events
		if event.Type != "session.spawned" {
			continue
		}

		// Check if this event is for one of our beads IDs
		beadsID, ok := event.Data["beads_id"].(string)
		if !ok || !beadsIDSet[beadsID] {
			continue
		}

		// Already have gap analysis for this beads ID? Skip (we want the most recent)
		// Since we read chronologically, later entries overwrite earlier ones
		if _, exists := result[beadsID]; exists {
			// We could skip, but let's allow overwrites for resumptions
		}

		// Extract gap analysis data from event
		gapData := extractGapAnalysisFromEvent(event.Data)
		if gapData != nil {
			result[beadsID] = gapData
		}
	}

	return result
}

// GapsAPIResponse is the JSON structure returned by /api/gaps.
type GapsAPIResponse struct {
	TotalGaps         int                    `json:"total_gaps"`
	RecurringPatterns int                    `json:"recurring_patterns"`
	BySkill           map[string]int         `json:"by_skill"`
	RecentGaps        int                    `json:"recent_gaps,omitempty"`       // Gaps in last 7 days
	Suggestions       []GapSuggestionSummary `json:"suggestions,omitempty"`       // Top recurring gap suggestions
	Error             string                 `json:"error,omitempty"`
}

// ReflectAPIResponse is the JSON structure returned by /api/reflect.
// It exposes the reflect-suggestions.json data with synthesis/promote/stale info.
type ReflectAPIResponse struct {
	Timestamp string                   `json:"timestamp"`
	Synthesis []ReflectSynthesisSummary `json:"synthesis"`
	Refine    []ReflectRefineSummary   `json:"refine,omitempty"`
	Error     string                   `json:"error,omitempty"`
}

// ReflectRefineSummary represents a kn entry that refines an existing principle.
type ReflectRefineSummary struct {
	ID         string   `json:"id"`
	Content    string   `json:"content"`
	Principle  string   `json:"principle"`
	MatchTerms []string `json:"match_terms"`
	Suggestion string   `json:"suggestion"`
}

// ReflectSynthesisSummary represents a topic with accumulated investigations.
type ReflectSynthesisSummary struct {
	Topic          string   `json:"topic"`
	Count          int      `json:"count"`
	Investigations []string `json:"investigations"`
	Suggestion     string   `json:"suggestion"`
}

// GapSuggestionSummary is a condensed version of LearningSuggestion for the API.
type GapSuggestionSummary struct {
	Query      string `json:"query"`
	Count      int    `json:"count"`
	Priority   string `json:"priority"`
	Suggestion string `json:"suggestion"`
}

// handleGaps returns gap tracker statistics from ~/.orch/gap-tracker.json.
func handleGaps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tracker, err := spawn.LoadTracker()
	if err != nil {
		resp := GapsAPIResponse{Error: fmt.Sprintf("Failed to load gap tracker: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Calculate by_skill breakdown
	bySkill := tracker.GetSkillGapRates()

	// Find recurring patterns (gaps that occurred 3+ times)
	suggestions := tracker.FindRecurringGaps()

	// Count recent gaps (last 7 days)
	recentGaps := 0
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	for _, event := range tracker.Events {
		if event.Timestamp.After(weekAgo) {
			recentGaps++
		}
	}

	// Convert suggestions to API format (top 5)
	var apiSuggestions []GapSuggestionSummary
	for i, s := range suggestions {
		if i >= 5 {
			break
		}
		apiSuggestions = append(apiSuggestions, GapSuggestionSummary{
			Query:      s.Query,
			Count:      s.Count,
			Priority:   s.Priority,
			Suggestion: s.Suggestion,
		})
	}

	resp := GapsAPIResponse{
		TotalGaps:         len(tracker.Events),
		RecurringPatterns: len(suggestions),
		BySkill:           bySkill,
		RecentGaps:        recentGaps,
		Suggestions:       apiSuggestions,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode gaps: %v", err), http.StatusInternalServerError)
		return
	}
}

// extractGapAnalysisFromEvent extracts gap analysis data from a spawn event's data map.
func extractGapAnalysisFromEvent(data map[string]interface{}) *GapAPIResponse {
	// Check if gap data exists
	hasGaps, ok := data["gap_has_gaps"].(bool)
	if !ok {
		return nil
	}

	contextQuality := 0
	if cq, ok := data["gap_context_quality"].(float64); ok {
		contextQuality = int(cq)
	}

	shouldWarn := false
	if sw, ok := data["gap_should_warn"].(bool); ok {
		shouldWarn = sw
	}

	matchCount := 0
	if mc, ok := data["gap_match_total"].(float64); ok {
		matchCount = int(mc)
	}

	constraints := 0
	if c, ok := data["gap_match_constraints"].(float64); ok {
		constraints = int(c)
	}

	decisions := 0
	if d, ok := data["gap_match_decisions"].(float64); ok {
		decisions = int(d)
	}

	investigations := 0
	if i, ok := data["gap_match_investigations"].(float64); ok {
		investigations = int(i)
	}

	return &GapAPIResponse{
		HasGaps:        hasGaps,
		ContextQuality: contextQuality,
		ShouldWarn:     shouldWarn,
		MatchCount:     matchCount,
		Constraints:    constraints,
		Decisions:      decisions,
		Investigations: investigations,
	}
}

// handleReflect returns reflect suggestions from ~/.orch/reflect-suggestions.json.
// This exposes synthesis/promote/stale data for kb reflect UI integration.
func handleReflect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read reflect-suggestions.json from ~/.orch/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		resp := ReflectAPIResponse{Error: fmt.Sprintf("Failed to get home directory: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	reflectPath := filepath.Join(homeDir, ".orch", "reflect-suggestions.json")
	data, err := os.ReadFile(reflectPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty response if file doesn't exist yet
			resp := ReflectAPIResponse{
				Timestamp: "",
				Synthesis: []ReflectSynthesisSummary{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		resp := ReflectAPIResponse{Error: fmt.Sprintf("Failed to read reflect-suggestions.json: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Parse the raw JSON structure
	var rawReflect struct {
		Timestamp string `json:"timestamp"`
		Synthesis []struct {
			Topic          string   `json:"topic"`
			Count          int      `json:"count"`
			Investigations []string `json:"investigations"`
			Suggestion     string   `json:"suggestion"`
		} `json:"synthesis"`
		Refine []struct {
			ID         string   `json:"id"`
			Content    string   `json:"content"`
			Principle  string   `json:"principle"`
			MatchTerms []string `json:"match_terms"`
			Suggestion string   `json:"suggestion"`
		} `json:"refine"`
	}

	if err := json.Unmarshal(data, &rawReflect); err != nil {
		resp := ReflectAPIResponse{Error: fmt.Sprintf("Failed to parse reflect-suggestions.json: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert to API response format
	var synthesis []ReflectSynthesisSummary
	for _, s := range rawReflect.Synthesis {
		synthesis = append(synthesis, ReflectSynthesisSummary{
			Topic:          s.Topic,
			Count:          s.Count,
			Investigations: s.Investigations,
			Suggestion:     s.Suggestion,
		})
	}

	// Convert refine data
	var refine []ReflectRefineSummary
	for _, r := range rawReflect.Refine {
		refine = append(refine, ReflectRefineSummary{
			ID:         r.ID,
			Content:    r.Content,
			Principle:  r.Principle,
			MatchTerms: r.MatchTerms,
			Suggestion: r.Suggestion,
		})
	}

	resp := ReflectAPIResponse{
		Timestamp: rawReflect.Timestamp,
		Synthesis: synthesis,
		Refine:    refine,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode reflect: %v", err), http.StatusInternalServerError)
		return
	}
}

// ErrorEvent represents an error event for the API.
type ErrorEvent struct {
	Type       string `json:"type"`                  // "session.error" or "agent.abandoned"
	SessionID  string `json:"session_id,omitempty"`  // Session ID if available
	BeadsID    string `json:"beads_id,omitempty"`    // Beads issue ID if available
	Timestamp  string `json:"timestamp"`             // ISO 8601 timestamp
	Message    string `json:"message,omitempty"`     // Error message or abandon reason
	Workspace  string `json:"workspace,omitempty"`   // Workspace path if available
	Skill      string `json:"skill,omitempty"`       // Skill type if known
	RecurCount int    `json:"recur_count,omitempty"` // How many times this error pattern has occurred
}

// ErrorPattern represents a recurring error pattern.
type ErrorPattern struct {
	Pattern    string   `json:"pattern"`     // Error message pattern (may be truncated/normalized)
	Count      int      `json:"count"`       // Number of occurrences
	LastSeen   string   `json:"last_seen"`   // ISO 8601 timestamp of most recent occurrence
	BeadsIDs   []string `json:"beads_ids"`   // Affected beads issues
	Suggestion string   `json:"suggestion"`  // Remediation suggestion
}

// ErrorsAPIResponse is the JSON structure returned by /api/errors.
type ErrorsAPIResponse struct {
	TotalErrors     int            `json:"total_errors"`               // Total error events
	ErrorsLast24h   int            `json:"errors_last_24h"`            // Errors in last 24 hours
	ErrorsLast7d    int            `json:"errors_last_7d"`             // Errors in last 7 days
	AbandonedCount  int            `json:"abandoned_count"`            // Total agent.abandoned events
	SessionErrors   int            `json:"session_errors"`             // Total session.error events
	RecentErrors    []ErrorEvent   `json:"recent_errors,omitempty"`    // Last 20 error events
	Patterns        []ErrorPattern `json:"patterns,omitempty"`         // Recurring error patterns
	ByType          map[string]int `json:"by_type"`                    // Breakdown by error type
	Error           string         `json:"error,omitempty"`            // Error message if any
}

// handleErrors returns error pattern analysis from ~/.orch/events.jsonl.
func handleErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logPath := events.DefaultLogPath()

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty response if file doesn't exist
			resp := ErrorsAPIResponse{
				ByType: make(map[string]int),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		resp := ErrorsAPIResponse{Error: fmt.Sprintf("Failed to open events file: %v", err)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer file.Close()

	now := time.Now()
	day24h := now.Add(-24 * time.Hour)
	days7 := now.Add(-7 * 24 * time.Hour)

	var allErrors []ErrorEvent
	byType := make(map[string]int)
	patternCounts := make(map[string]*ErrorPattern)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		// Only process error-related events
		if event.Type != events.EventTypeSessionError && event.Type != "agent.abandoned" {
			continue
		}

		ts := time.Unix(event.Timestamp, 0)
		errorEvent := ErrorEvent{
			Type:      event.Type,
			SessionID: event.SessionID,
			Timestamp: ts.Format(time.RFC3339),
		}

		// Extract details based on event type
		if event.Type == events.EventTypeSessionError {
			if msg, ok := event.Data["error"].(string); ok {
				errorEvent.Message = msg
			}
		} else if event.Type == "agent.abandoned" {
			if reason, ok := event.Data["reason"].(string); ok {
				errorEvent.Message = reason
			}
			if beadsID, ok := event.Data["beads_id"].(string); ok {
				errorEvent.BeadsID = beadsID
			}
			if workspace, ok := event.Data["workspace_path"].(string); ok {
				errorEvent.Workspace = filepath.Base(workspace)
			}
			if agentID, ok := event.Data["agent_id"].(string); ok {
				// Extract skill from agent_id pattern like "og-feat-xxx" or "og-debug-xxx"
				errorEvent.Skill = extractSkillFromAgentID(agentID)
			}
		}

		allErrors = append(allErrors, errorEvent)
		byType[event.Type]++

		// Track patterns for recurring error detection
		patternKey := normalizeErrorMessage(errorEvent.Message)
		if patternKey != "" {
			if p, exists := patternCounts[patternKey]; exists {
				p.Count++
				p.LastSeen = errorEvent.Timestamp
				if errorEvent.BeadsID != "" && !containsString(p.BeadsIDs, errorEvent.BeadsID) {
					p.BeadsIDs = append(p.BeadsIDs, errorEvent.BeadsID)
				}
			} else {
				beadsIDs := []string{}
				if errorEvent.BeadsID != "" {
					beadsIDs = append(beadsIDs, errorEvent.BeadsID)
				}
				patternCounts[patternKey] = &ErrorPattern{
					Pattern:  patternKey,
					Count:    1,
					LastSeen: errorEvent.Timestamp,
					BeadsIDs: beadsIDs,
				}
			}
		}
	}

	// Count errors by time window
	var errorsLast24h, errorsLast7d int
	for _, e := range allErrors {
		ts, _ := time.Parse(time.RFC3339, e.Timestamp)
		if ts.After(day24h) {
			errorsLast24h++
		}
		if ts.After(days7) {
			errorsLast7d++
		}
	}

	// Get recent errors (last 20, most recent first)
	recentErrors := allErrors
	if len(recentErrors) > 20 {
		recentErrors = recentErrors[len(recentErrors)-20:]
	}
	// Reverse to show most recent first
	for i, j := 0, len(recentErrors)-1; i < j; i, j = i+1, j-1 {
		recentErrors[i], recentErrors[j] = recentErrors[j], recentErrors[i]
	}

	// Convert patterns map to slice and sort by count
	var patterns []ErrorPattern
	for _, p := range patternCounts {
		if p.Count >= 2 { // Only include patterns that occurred 2+ times
			p.Suggestion = suggestRemediation(p.Pattern)
			patterns = append(patterns, *p)
		}
	}
	// Sort by count descending
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].Count > patterns[i].Count {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}
	// Limit to top 10 patterns
	if len(patterns) > 10 {
		patterns = patterns[:10]
	}

	resp := ErrorsAPIResponse{
		TotalErrors:    len(allErrors),
		ErrorsLast24h:  errorsLast24h,
		ErrorsLast7d:   errorsLast7d,
		AbandonedCount: byType["agent.abandoned"],
		SessionErrors:  byType[events.EventTypeSessionError],
		RecentErrors:   recentErrors,
		Patterns:       patterns,
		ByType:         byType,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode errors: %v", err), http.StatusInternalServerError)
		return
	}
}

// extractSkillFromAgentID extracts the skill type from an agent ID.
// Agent IDs have patterns like "og-feat-xxx", "og-debug-xxx", "og-inv-xxx".
func extractSkillFromAgentID(agentID string) string {
	parts := strings.Split(agentID, "-")
	if len(parts) < 2 {
		return ""
	}
	// Map short prefixes to skill names
	switch parts[1] {
	case "feat":
		return "feature-impl"
	case "debug":
		return "systematic-debugging"
	case "inv":
		return "investigation"
	case "arch":
		return "architect"
	case "work":
		return "design-session"
	default:
		return parts[1]
	}
}

// normalizeErrorMessage normalizes an error message for pattern matching.
// Removes specific identifiers to group similar errors together.
func normalizeErrorMessage(msg string) string {
	if msg == "" {
		return ""
	}
	// Truncate long messages
	if len(msg) > 100 {
		msg = msg[:100]
	}
	// Simple normalization - could be enhanced with regex for IDs, paths, etc.
	return strings.TrimSpace(msg)
}

// containsString checks if a string is in a slice.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// suggestRemediation provides a remediation suggestion based on error pattern.
func suggestRemediation(pattern string) string {
	lower := strings.ToLower(pattern)
	switch {
	case strings.Contains(lower, "stall"):
		return "Check agent for long-running operations or API timeouts"
	case strings.Contains(lower, "timeout"):
		return "Review API response times or increase timeout limits"
	case strings.Contains(lower, "capacity"):
		return "Increase daemon capacity or check for stuck agents"
	case strings.Contains(lower, "daemon"):
		return "Check daemon logs at ~/.orch/daemon.log"
	case strings.Contains(lower, "context"):
		return "Review spawn context for missing or incorrect information"
	case strings.Contains(lower, "connection"):
		return "Check network connectivity or API endpoint availability"
	default:
		return "Review agent workspace for more details"
	}
}

// PendingReviewItem represents a single synthesis recommendation pending review.
type PendingReviewItem struct {
	WorkspaceID string `json:"workspace_id"`
	BeadsID     string `json:"beads_id"`
	Index       int    `json:"index"`       // Index of the recommendation (0-based)
	Text        string `json:"text"`        // The recommendation text
	Reviewed    bool   `json:"reviewed"`    // Whether this item has been reviewed
	ActedOn     bool   `json:"acted_on"`    // Whether an issue was created
	Dismissed   bool   `json:"dismissed"`   // Whether this was dismissed
}

// PendingReviewAgent represents an agent with pending synthesis reviews.
type PendingReviewAgent struct {
	WorkspaceID          string              `json:"workspace_id"`
	WorkspacePath        string              `json:"workspace_path"`
	BeadsID              string              `json:"beads_id"`
	TLDR                 string              `json:"tldr,omitempty"`
	TotalRecommendations int                 `json:"total_recommendations"`
	UnreviewedCount      int                 `json:"unreviewed_count"`
	Items                []PendingReviewItem `json:"items"`
	IsLightTier          bool                `json:"is_light_tier,omitempty"` // True if this was a light tier spawn (no synthesis by design)
}

// PendingReviewsAPIResponse is the JSON structure returned by /api/pending-reviews.
type PendingReviewsAPIResponse struct {
	Agents           []PendingReviewAgent `json:"agents"`
	TotalAgents      int                  `json:"total_agents"`
	TotalUnreviewed  int                  `json:"total_unreviewed"`
}

// handlePendingReviews returns pending synthesis reviews.
// This includes both full-tier agents with SYNTHESIS.md and light-tier agents
// that have completed (Phase: Complete) but have no synthesis by design.
func handlePendingReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Scan workspaces for SYNTHESIS.md and review state
	workspaceDir := filepath.Join(sourceDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		// No workspace directory is fine - just return empty response
		resp := PendingReviewsAPIResponse{
			Agents:          []PendingReviewAgent{},
			TotalAgents:     0,
			TotalUnreviewed: 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	var agents []PendingReviewAgent
	totalUnreviewed := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(workspaceDir, dirName)

		// Check for SYNTHESIS.md (full-tier agents)
		synthesisPath := filepath.Join(dirPath, "SYNTHESIS.md")
		hasSynthesis := false
		if _, err := os.Stat(synthesisPath); err == nil {
			hasSynthesis = true
		}

		// Check for light-tier completion (no synthesis by design)
		isLightComplete, lightBeadsID := isLightTierComplete(dirPath)

		// Skip workspaces that are neither full-tier with synthesis nor light-tier complete
		if !hasSynthesis && !isLightComplete {
			continue
		}

		// Handle full-tier agents with synthesis
		if hasSynthesis {
			// Parse synthesis
			synthesis, err := verify.ParseSynthesis(dirPath)
			if err != nil || synthesis == nil {
				continue
			}

			// Skip if no recommendations
			if len(synthesis.NextActions) == 0 {
				continue
			}

			// Load review state
			reviewState, err := verify.LoadReviewState(dirPath)
			if err != nil {
				reviewState = &verify.ReviewState{}
			}

			// Extract beads ID from SPAWN_CONTEXT
			beadsID := extractBeadsIDFromWorkspace(dirPath)

			// Build item list
			var items []PendingReviewItem
			unreviewedCount := 0

			for i, action := range synthesis.NextActions {
				actedOn := contains(reviewState.ActedOn, i)
				dismissed := contains(reviewState.Dismissed, i)
				reviewed := actedOn || dismissed

				if !reviewed {
					unreviewedCount++
				}

				items = append(items, PendingReviewItem{
					WorkspaceID: dirName,
					BeadsID:     beadsID,
					Index:       i,
					Text:        action,
					Reviewed:    reviewed,
					ActedOn:     actedOn,
					Dismissed:   dismissed,
				})
			}

			// Only include agents with unreviewed recommendations
			if unreviewedCount > 0 {
				agents = append(agents, PendingReviewAgent{
					WorkspaceID:          dirName,
					WorkspacePath:        dirPath,
					BeadsID:              beadsID,
					TLDR:                 synthesis.TLDR,
					TotalRecommendations: len(synthesis.NextActions),
					UnreviewedCount:      unreviewedCount,
					Items:                items,
					IsLightTier:          false,
				})
				totalUnreviewed += unreviewedCount
			}
		} else if isLightComplete {
			// Handle light-tier agents (no synthesis by design)
			// Light-tier completions appear with a special indicator and no items
			// They still need orchestrator acknowledgment via beads close

			// Load review state to check if already acknowledged
			reviewState, err := verify.LoadReviewState(dirPath)
			if err != nil {
				reviewState = &verify.ReviewState{}
			}

			// Skip if already reviewed (acknowledged)
			if reviewState.LightTierAcknowledged {
				continue
			}

			// Create a single "pseudo-item" indicating the light tier completion needs review
			items := []PendingReviewItem{
				{
					WorkspaceID: dirName,
					BeadsID:     lightBeadsID,
					Index:       0,
					Text:        "Light tier agent completed - no synthesis produced (by design). Review and close via orch complete.",
					Reviewed:    false,
					ActedOn:     false,
					Dismissed:   false,
				},
			}

			agents = append(agents, PendingReviewAgent{
				WorkspaceID:          dirName,
				WorkspacePath:        dirPath,
				BeadsID:              lightBeadsID,
				TLDR:                 "Light tier completion - review agent output directly",
				TotalRecommendations: 1,
				UnreviewedCount:      1,
				Items:                items,
				IsLightTier:          true,
			})
			totalUnreviewed++
		}
	}

	resp := PendingReviewsAPIResponse{
		Agents:          agents,
		TotalAgents:     len(agents),
		TotalUnreviewed: totalUnreviewed,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode pending reviews: %v", err), http.StatusInternalServerError)
		return
	}
}

// contains checks if a slice contains a value.
func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// isLightTierWorkspace checks if a workspace is a light tier spawn.
// Light tier workspaces have a .tier file containing "light".
func isLightTierWorkspace(workspacePath string) bool {
	tierPath := filepath.Join(workspacePath, ".tier")
	data, err := os.ReadFile(tierPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "light"
}

// isLightTierComplete checks if a light tier workspace has Phase: Complete in beads comments.
// Returns true if the workspace is light tier AND has Phase: Complete.
func isLightTierComplete(workspacePath string) (isComplete bool, beadsID string) {
	if !isLightTierWorkspace(workspacePath) {
		return false, ""
	}

	// Extract beads ID from SPAWN_CONTEXT.md
	beadsID = extractBeadsIDFromWorkspace(workspacePath)
	if beadsID == "" {
		return false, ""
	}

	// Get comments for this beads ID
	comments, err := verify.GetComments(beadsID)
	if err != nil {
		return false, beadsID
	}

	// Check for Phase: Complete
	phaseStatus := verify.ParsePhaseFromComments(comments)
	return phaseStatus.Found && strings.EqualFold(phaseStatus.Phase, "Complete"), beadsID
}

// DismissReviewRequest is the request body for POST /api/dismiss-review.
type DismissReviewRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Index       int    `json:"index"` // Index of the recommendation to dismiss
}

// DismissReviewResponse is the response for POST /api/dismiss-review.
type DismissReviewResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
}

// handleDismissReview dismisses a synthesis recommendation.
func handleDismissReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DismissReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	if req.WorkspaceID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   "workspace_id is required",
		})
		return
	}

	// Build workspace path
	workspacePath := filepath.Join(sourceDir, ".orch", "workspace", req.WorkspaceID)

	// Check for light-tier workspace - these don't have SYNTHESIS.md by design.
	// Light-tier dismissals set LightTierAcknowledged instead of tracking individual recommendations.
	if isLightTierWorkspace(workspacePath) {
		reviewState, err := verify.LoadReviewState(workspacePath)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(DismissReviewResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to load review state: %v", err),
			})
			return
		}

		// Check if already acknowledged
		if reviewState.LightTierAcknowledged {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(DismissReviewResponse{
				Success: true,
				Message: "Already acknowledged",
			})
			return
		}

		// Mark as acknowledged
		reviewState.LightTierAcknowledged = true
		reviewState.WorkspaceID = req.WorkspaceID
		reviewState.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
		if reviewState.ReviewedAt.IsZero() {
			reviewState.ReviewedAt = time.Now()
		}

		if err := verify.SaveReviewState(workspacePath, reviewState); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(DismissReviewResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to save review state: %v", err),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: true,
			Message: "Light tier completion acknowledged",
		})
		return
	}

	// Full-tier path: Load existing review state
	reviewState, err := verify.LoadReviewState(workspacePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to load review state: %v", err),
		})
		return
	}

	// Parse synthesis to get total recommendations
	synthesis, err := verify.ParseSynthesis(workspacePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse synthesis: %v", err),
		})
		return
	}

	// Validate index
	if req.Index < 0 || req.Index >= len(synthesis.NextActions) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid index %d (total recommendations: %d)", req.Index, len(synthesis.NextActions)),
		})
		return
	}

	// Check if already reviewed
	if contains(reviewState.ActedOn, req.Index) || contains(reviewState.Dismissed, req.Index) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: true,
			Message: "Already reviewed",
		})
		return
	}

	// Add to dismissed
	reviewState.Dismissed = append(reviewState.Dismissed, req.Index)
	reviewState.TotalRecommendations = len(synthesis.NextActions)
	reviewState.WorkspaceID = req.WorkspaceID
	if reviewState.ReviewedAt.IsZero() {
		reviewState.ReviewedAt = time.Now()
	}

	// Extract beads ID if not set
	if reviewState.BeadsID == "" {
		reviewState.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
	}

	// Save updated state
	if err := verify.SaveReviewState(workspacePath, reviewState); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DismissReviewResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to save review state: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DismissReviewResponse{
		Success: true,
		Message: fmt.Sprintf("Dismissed recommendation %d", req.Index),
	})
}

// ConfigAPIResponse is the JSON structure returned by GET /api/config.
type ConfigAPIResponse struct {
	Backend              string `json:"backend"`
	AutoExportTranscript bool   `json:"auto_export_transcript"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
	ConfigPath           string `json:"config_path,omitempty"` // Path to config file for reference
}

// ConfigUpdateRequest is the JSON structure for PUT /api/config.
type ConfigUpdateRequest struct {
	Backend              *string `json:"backend,omitempty"`
	AutoExportTranscript *bool   `json:"auto_export_transcript,omitempty"`
	NotificationsEnabled *bool   `json:"notifications_enabled,omitempty"`
}

// handleConfig handles GET and PUT requests for user configuration.
// GET returns current config from ~/.orch/config.yaml
// PUT updates specified fields in the config
func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleConfigGet(w, r)
	case http.MethodPut:
		handleConfigPut(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleConfigGet returns the current user configuration.
func handleConfigGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := userconfig.Load()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	resp := ConfigAPIResponse{
		Backend:              cfg.Backend,
		AutoExportTranscript: cfg.AutoExportTranscript,
		NotificationsEnabled: cfg.NotificationsEnabled(),
		ConfigPath:           userconfig.ConfigPath(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode config: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleConfigPut updates the user configuration with the provided values.
func handleConfigPut(w http.ResponseWriter, r *http.Request) {
	var req ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Load existing config
	cfg, err := userconfig.Load()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Update only the fields that were provided
	if req.Backend != nil {
		cfg.Backend = *req.Backend
	}
	if req.AutoExportTranscript != nil {
		cfg.AutoExportTranscript = *req.AutoExportTranscript
	}
	if req.NotificationsEnabled != nil {
		cfg.Notifications.Enabled = req.NotificationsEnabled
	}

	// Save the updated config
	if err := userconfig.Save(cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the updated config
	resp := ConfigAPIResponse{
		Backend:              cfg.Backend,
		AutoExportTranscript: cfg.AutoExportTranscript,
		NotificationsEnabled: cfg.NotificationsEnabled(),
		ConfigPath:           userconfig.ConfigPath(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode config: %v", err), http.StatusInternalServerError)
		return
	}
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
