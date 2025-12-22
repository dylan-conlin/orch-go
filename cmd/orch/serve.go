package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/opencode"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
	"github.com/dylan-conlin/orch-go/pkg/tmux"
	"github.com/dylan-conlin/orch-go/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	servePort int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server for the beads-ui dashboard",
	Long: `Start an HTTP API server that provides endpoints for the beads-ui dashboard.

Endpoints:
  GET /api/agents  - Returns JSON list of active agents from OpenCode/tmux
  GET /api/events  - Proxies the OpenCode SSE stream for real-time updates

The server runs on port 3333 by default.

Examples:
  orch-go serve              # Start server on port 3333
  orch-go serve --port 8080  # Start server on port 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServe(servePort)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 3333, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func runServe(port int) error {
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

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting orch-go API server on http://127.0.0.1%s\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/agents    - List of active agents from OpenCode/tmux")
	fmt.Println("  GET /api/events    - SSE proxy for OpenCode events")
	fmt.Println("  GET /api/agentlog  - Agent lifecycle events (supports ?follow=true for SSE)")
	fmt.Println("  GET /health        - Health check")
	fmt.Println("\nPress Ctrl+C to stop")

	return http.ListenAndServe(addr, mux)
}

// AgentAPIResponse is the JSON structure returned by /api/agents.
type AgentAPIResponse struct {
	ID         string             `json:"id"`
	SessionID  string             `json:"session_id,omitempty"`
	BeadsID    string             `json:"beads_id,omitempty"`
	BeadsTitle string             `json:"beads_title,omitempty"`
	Skill      string             `json:"skill,omitempty"`
	Status     string             `json:"status"` // "active", "completed", etc.
	Runtime    string             `json:"runtime,omitempty"`
	Window     string             `json:"window,omitempty"`
	Synthesis  *SynthesisResponse `json:"synthesis,omitempty"`
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

// handleAgents returns JSON list of active agents from OpenCode/tmux and completed workspaces.
func handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir, _ := os.Getwd()
	client := opencode.NewClient(serverURL)

	// Get active sessions from OpenCode
	sessions, err := client.ListSessions(projectDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list sessions: %v", err), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	var agents []AgentAPIResponse

	// Add active sessions from OpenCode
	for _, s := range sessions {
		createdAt := time.Unix(s.Time.Created/1000, 0)
		runtime := now.Sub(createdAt)

		agent := AgentAPIResponse{
			ID:        s.Title,
			SessionID: s.ID,
			Status:    "active",
			Runtime:   formatDuration(runtime),
		}

		// Derive beadsID and skill from session title
		if s.Title != "" {
			agent.BeadsID = extractBeadsIDFromTitle(s.Title)
			agent.Skill = extractSkillFromTitle(s.Title)
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
					Status:  "active",
					Window:  win.Target,
				})
			}
		}
	}

	// Add completed workspaces (those with SYNTHESIS.md)
	workspaceDir := filepath.Join(projectDir, ".orch", "workspace")
	if entries, err := os.ReadDir(workspaceDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			workspacePath := filepath.Join(workspaceDir, entry.Name())
			synthesisPath := filepath.Join(workspacePath, "SYNTHESIS.md")

			// Check if SYNTHESIS.md exists (indicates completion)
			if _, err := os.Stat(synthesisPath); err == nil {
				// Check if already in active list
				alreadyIn := false
				for _, a := range agents {
					if a.ID == entry.Name() {
						alreadyIn = true
						break
					}
				}

				if !alreadyIn {
					agent := AgentAPIResponse{
						ID:     entry.Name(),
						Status: "completed",
					}

					// Read session ID from workspace
					if sessionID := spawn.ReadSessionID(workspacePath); sessionID != "" {
						agent.SessionID = sessionID
					}

					// Parse synthesis
					if synthesis, err := verify.ParseSynthesis(workspacePath); err == nil {
						agent.Synthesis = &SynthesisResponse{
							TLDR:           synthesis.TLDR,
							Outcome:        synthesis.Outcome,
							Recommendation: synthesis.Recommendation,
							DeltaSummary:   summarizeDelta(synthesis.Delta),
							NextActions:    synthesis.NextActions,
						}
					}

					// Derive beadsID from workspace name
					agent.BeadsID = extractBeadsIDFromTitle(entry.Name())
					agent.Skill = extractSkillFromTitle(entry.Name())

					agents = append(agents, agent)
				}
			}
		}
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
