// Package daemon provides autonomous overnight processing capabilities.
// This file implements per-cycle caching for expensive daemon queries.
//
// Problem: Multiple periodic tasks (recovery, orphan detection, phase timeout,
// question detection) independently call GetActiveAgents() each cycle. Each call
// queries beads for in_progress issues and fetches comments in batch — 4 redundant
// round-trips per OODA cycle.
//
// Solution: cachedAgentDiscoverer wraps an AgentDiscoverer and caches the
// GetActiveAgents() result for the duration of one cycle. Session checks
// (HasExistingSession, HasExistingSessionOrError) are NOT cached because they
// check live infrastructure state (OpenCode sessions, tmux windows) that may
// change between tasks within a single cycle.
package daemon

// cachedAgentDiscoverer wraps an AgentDiscoverer and caches GetActiveAgents()
// after the first call. Session checks delegate directly to the inner discoverer.
type cachedAgentDiscoverer struct {
	inner  AgentDiscoverer
	agents []ActiveAgent
	err    error
	loaded bool
}

// newCachedAgentDiscoverer creates a new caching wrapper around the given discoverer.
func newCachedAgentDiscoverer(inner AgentDiscoverer) *cachedAgentDiscoverer {
	return &cachedAgentDiscoverer{inner: inner}
}

func (c *cachedAgentDiscoverer) GetActiveAgents() ([]ActiveAgent, error) {
	if !c.loaded {
		c.agents, c.err = c.inner.GetActiveAgents()
		c.loaded = true
	}
	return c.agents, c.err
}

func (c *cachedAgentDiscoverer) HasExistingSession(beadsID string) bool {
	return c.inner.HasExistingSession(beadsID)
}

func (c *cachedAgentDiscoverer) HasExistingSessionOrError(beadsID string) (bool, error) {
	return c.inner.HasExistingSessionOrError(beadsID)
}

// BeginCycle wraps d.Agents with a caching layer so that all periodic tasks
// within the same OODA cycle share a single GetActiveAgents() result.
// Must be paired with EndCycle() to restore the original discoverer.
func (d *Daemon) BeginCycle() {
	d.uncachedAgents = d.Agents
	d.Agents = newCachedAgentDiscoverer(d.uncachedAgents)
}

// EndCycle restores the original (uncached) AgentDiscoverer.
func (d *Daemon) EndCycle() {
	if d.uncachedAgents != nil {
		d.Agents = d.uncachedAgents
		d.uncachedAgents = nil
	}
}
