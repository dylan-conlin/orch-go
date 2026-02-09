package main

import "github.com/dylan-conlin/orch-go/pkg/beads"

// withBeadsClient creates a beads RPC client, connects it, and closes it after fn returns.
func withBeadsClient(projectDir string, fn func(*beads.Client) error, opts ...beads.Option) error {
	return beads.WithConnected(projectDir, fn, opts...)
}

// withBeadsFallback runs an RPC operation and falls back to CLI when RPC fails.
func withBeadsFallback(projectDir string, fn func(*beads.Client) error, fallback func() error, opts ...beads.Option) error {
	return beads.WithFallback(projectDir, fn, fallback, opts...)
}
