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
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/account"
	"github.com/dylan-conlin/orch-go/pkg/beads"
	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/focus"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/port"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/usage"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

// DefaultServePort is the port orch serve listens on.
// This is infrastructure, not a project dev server.
const DefaultServePort = 3348

var (
	servePort int
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
  GET /api/servers   - Servers status across projects
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
	addr := fmt.Sprintf("http://127.0.0.1:%d/health", portNum)

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
	fmt.Printf("   URL:    http://127.0.0.1:%d\n", portNum)
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/agents    - Active agents")
	fmt.Println("  GET /api/events    - SSE event stream")
	fmt.Println("  GET /api/agentlog  - Agent lifecycle events")
	fmt.Println("  GET /api/usage     - Claude Max usage")
	fmt.Println("  GET /api/focus     - Focus and drift status")
	fmt.Println("  GET /api/beads     - Beads stats")
	fmt.Println("  GET /api/servers   - Project servers status")
	fmt.Println("  GET /health        - Health check")

	return nil
}

func runServe(portNum int) error {
	// Set default directory for beads socket discovery
	// This is needed because serve may run from any working directory
	if sourceDir != "" && sourceDir != "unknown" {
		beads.DefaultDir = sourceDir
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
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

	// GET /api/servers - returns servers status across projects
	mux.HandleFunc("/api/servers", corsHandler(handleServers))

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// pprof handlers for CPU profiling (useful for debugging CPU runaway)
	// Access at: http://127.0.0.1:3348/debug/pprof/
	mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)

	addr := fmt.Sprintf(":%d", portNum)
	fmt.Printf("Starting orch-go API server on http://127.0.0.1%s\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/agents    - List of active agents from OpenCode/tmux")
	fmt.Println("  GET /api/events    - SSE proxy for OpenCode events")
	fmt.Println("  GET /api/agentlog  - Agent lifecycle events (supports ?follow=true for SSE)")
	fmt.Println("  GET /api/usage     - Claude Max usage stats")
	fmt.Println("  GET /api/focus     - Current focus and drift status")
	fmt.Println("  GET /api/beads     - Beads stats (ready, blocked, open)")
	fmt.Println("  GET /api/servers   - Servers status across projects")
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode agents: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleEvents proxies the OpenCode SSE stream to the client.
// It connects to http://127.0.0.1:4096/event and forwards events.
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

	// Try RPC client first, using sourceDir (build-time) as the base directory
	// since serve may run from any working directory
	var stats *beads.Stats
	var err error

	socketPath, socketErr := beads.FindSocketPath(sourceDir)
	if socketErr == nil {
		client := beads.NewClient(socketPath)
		if connErr := client.Connect(); connErr == nil {
			defer client.Close()
			stats, err = client.Stats()
			if err != nil {
				// Fall through to CLI fallback on RPC error
				stats, err = beads.FallbackStats()
			}
		} else {
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
