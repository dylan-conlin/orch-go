package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// handleServiceEvents returns service lifecycle events from ~/.orch/events.jsonl.
// Without query params: returns last 100 service events as JSON array.
// With ?follow=true: streams new service events via SSE.
func (s *Server) handleServiceEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	follow := r.URL.Query().Get("follow") == "true"

	if follow {
		s.handleServiceEventsSSE(w, r)
	} else {
		s.handleServiceEventsJSON(w, r)
	}
}

// handleServiceEventsJSON returns the last 100 service events as JSON array.
func (s *Server) handleServiceEventsJSON(w http.ResponseWriter, r *http.Request) {
	logPath := events.DefaultLogPath()

	allEvents, err := readLastNEvents(logPath, 1000) // Read more to filter
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

	// Filter to service.* events only
	serviceEvents := filterServiceEvents(allEvents)

	// Return last 100
	if len(serviceEvents) > 100 {
		serviceEvents = serviceEvents[len(serviceEvents)-100:]
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(serviceEvents); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode events: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleServiceEventsSSE streams new service events via SSE as they are appended to events.jsonl.
func (s *Server) handleServiceEventsSSE(w http.ResponseWriter, r *http.Request) {
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
	var file *os.File
	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

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

			// Reopen if current log file was rotated.
			if err := reopenIfLogRotated(logPath, &file, &reader); err != nil {
				continue
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

				// Validate it's valid JSON and check if it's a service event
				var event events.Event
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					continue // Skip invalid lines
				}

				// Filter to service.* events only
				if !strings.HasPrefix(event.Type, "service.") {
					continue
				}

				fmt.Fprintf(w, "event: servicelog\ndata: %s\n\n", line)
				flusher.Flush()
			}
		}
	}
}

// filterServiceEvents filters events to only service lifecycle events.
func filterServiceEvents(allEvents []events.Event) []events.Event {
	var serviceEvents []events.Event
	for _, e := range allEvents {
		if strings.HasPrefix(e.Type, "service.") {
			serviceEvents = append(serviceEvents, e)
		}
	}
	return serviceEvents
}

// handleServiceLogs returns logs for a specific service from overmind echo.
// URL pattern: /api/services/{name}/logs
func (s *Server) handleServiceLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract service name from path
	// Path is like: /api/services/api/logs or /api/services/web/logs
	path := strings.TrimPrefix(r.URL.Path, "/api/services/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "logs" {
		http.Error(w, "Invalid path. Expected: /api/services/{name}/logs", http.StatusBadRequest)
		return
	}
	serviceName := parts[0]

	// Get logs from overmind echo
	logs, err := getServiceLogs(serviceName, s.SourceDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get logs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"service": serviceName,
		"logs":    logs,
		"count":   len(logs),
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode logs: %v", err), http.StatusInternalServerError)
		return
	}
}

// getServiceLogs executes overmind echo <service> and returns the last 100 lines.
func getServiceLogs(serviceName, projectPath string) ([]string, error) {
	// Run overmind echo <service> from the project directory
	cmd := exec.Command("overmind", "echo", serviceName)
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		// If overmind isn't running or service not found, return empty
		return []string{}, nil
	}

	// Parse output into lines
	lines := strings.Split(string(output), "\n")

	// Remove empty lines
	var nonEmptyLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	// Return last 100 lines
	if len(nonEmptyLines) > 100 {
		return nonEmptyLines[len(nonEmptyLines)-100:], nil
	}
	return nonEmptyLines, nil
}
