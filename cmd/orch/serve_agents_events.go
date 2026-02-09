package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// handleEvents proxies the OpenCode SSE stream to the client.
// It connects to http://127.0.0.1:4096/event and forwards events.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
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

	// Connect to OpenCode SSE stream using request context so upstream is canceled
	// immediately when the dashboard client disconnects.
	opencodeURL := s.ServerURL + "/event"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, opencodeURL, nil)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Failed to create OpenCode request: %s\"}\n\n", err.Error())
		flusher.Flush()
		return
	}
	resp, err := serveOpenCodeSSEHTTPClient.Do(req)
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
		line, err := reader.ReadString('\n')
		if err != nil {
			if ctx.Err() != nil {
				return
			}
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

// handleAgentlog returns agent lifecycle events from ~/.orch/events.jsonl.
// Without query params: returns last 100 events as JSON array.
// With ?follow=true: streams new events via SSE.
func (s *Server) handleAgentlog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	follow := r.URL.Query().Get("follow") == "true"

	if follow {
		s.handleAgentlogSSE(w, r)
	} else {
		s.handleAgentlogJSON(w, r)
	}
}

// handleAgentlogJSON returns the last 100 events as JSON array.
func (s *Server) handleAgentlogJSON(w http.ResponseWriter, r *http.Request) {
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
func (s *Server) handleAgentlogSSE(w http.ResponseWriter, r *http.Request) {
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

func reopenIfLogRotated(logPath string, file **os.File, reader **bufio.Reader) error {
	if file == nil || *file == nil {
		return nil
	}

	openInfo, err := (*file).Stat()
	if err != nil {
		return err
	}

	currentInfo, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if os.SameFile(openInfo, currentInfo) {
		return nil
	}

	_ = (*file).Close()

	newFile, err := os.Open(logPath)
	if err != nil {
		return err
	}

	*file = newFile
	*reader = bufio.NewReader(newFile)
	return nil
}

// readLastNEvents reads the last n events from a JSONL file.
func readLastNEvents(path string, n int) ([]events.Event, error) {
	var allEvents []events.Event
	err := events.ReadCompactedJSONL(path, func(line string) error {
		var event events.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil // Skip invalid lines
		}
		allEvents = append(allEvents, event)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Return last n events
	if len(allEvents) > n {
		return allEvents[len(allEvents)-n:], nil
	}
	return allEvents, nil
}
