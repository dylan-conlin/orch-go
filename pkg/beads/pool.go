package beads

import (
	"sync"
)

// Pool manages lazy-initialized beads clients for multiple project directories.
// This enables the dashboard to query beads from different projects without
// creating a new connection for each request.
//
// Usage:
//
//	pool := beads.NewPool()
//	client := pool.GetOrCreate("/path/to/project") // Returns nil if daemon unavailable
//	if client != nil {
//	    stats, _ := client.Stats()
//	}
type Pool struct {
	mu        sync.Mutex
	clients   map[string]*Client
	attempted map[string]bool // Tracks directories we've tried (to avoid repeated connection attempts)
}

// NewPool creates a new BeadsClientPool.
func NewPool() *Pool {
	return &Pool{
		clients:   make(map[string]*Client),
		attempted: make(map[string]bool),
	}
}

// GetOrCreate returns a beads client for the given project directory.
// If no client exists, it attempts to create one by finding and connecting
// to the beads daemon socket for that directory.
//
// Returns nil if:
// - The directory has no .beads/bd.sock
// - The beads daemon is not running for that project
// - Connection to the daemon failed
//
// This is safe for concurrent use.
func (p *Pool) GetOrCreate(projectDir string) *Client {
	// Normalize empty string to DefaultDir
	if projectDir == "" {
		projectDir = DefaultDir
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if we already have a client for this directory
	if client, ok := p.clients[projectDir]; ok {
		return client
	}

	// Check if we've already attempted and failed for this directory
	if p.attempted[projectDir] {
		return nil
	}

	// Mark as attempted before trying (prevents repeated failed attempts)
	p.attempted[projectDir] = true

	// Try to find the socket path
	socketPath, err := FindSocketPath(projectDir)
	if err != nil {
		// No socket found - daemon not running or no .beads directory
		return nil
	}

	// Create client with auto-reconnect
	client := NewClient(socketPath, WithAutoReconnect(3), WithCwd(projectDir))

	// Try to connect
	if err := client.Connect(); err != nil {
		// Connection failed - daemon may not be running
		return nil
	}

	// Store the connected client
	p.clients[projectDir] = client
	return client
}

// Stats returns pool statistics for debugging/monitoring.
// Returns (connected count, attempted count, list of tracked directories).
func (p *Pool) Stats() (int, int, []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	connectedCount := len(p.clients)
	attemptedCount := len(p.attempted)

	dirs := make([]string, 0, attemptedCount)
	for dir := range p.attempted {
		dirs = append(dirs, dir)
	}

	return connectedCount, attemptedCount, dirs
}

// CloseAll closes all clients in the pool and clears the pool.
// Should be called when the server is shutting down.
func (p *Pool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		if client != nil {
			client.Close()
		}
	}

	// Clear maps
	p.clients = make(map[string]*Client)
	p.attempted = make(map[string]bool)
}

// Reset clears the attempted cache for a specific directory,
// allowing a fresh connection attempt on next GetOrCreate.
// Useful when you know the daemon has been (re)started.
func (p *Pool) Reset(projectDir string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close existing client if any
	if client, ok := p.clients[projectDir]; ok {
		if client != nil {
			client.Close()
		}
		delete(p.clients, projectDir)
	}

	// Clear attempted flag
	delete(p.attempted, projectDir)
}
