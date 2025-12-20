// Package notify provides desktop notification functionality.
package notify

import (
	"fmt"

	"github.com/gen2brain/beeep"
)

// Backend is the interface for sending notifications.
// This abstraction allows for mocking in tests.
type Backend interface {
	Notify(title, message, icon string) error
}

// BeeepBackend wraps the beeep library for notifications.
type BeeepBackend struct{}

// Notify sends a desktop notification using beeep.
func (b *BeeepBackend) Notify(title, message, icon string) error {
	return beeep.Notify(title, message, icon)
}

// Notifier handles sending desktop notifications.
type Notifier struct {
	backend Backend
}

// New creates a new Notifier with the given backend.
func New(backend Backend) *Notifier {
	return &Notifier{backend: backend}
}

// Default creates a Notifier with the default beeep backend.
func Default() *Notifier {
	return &Notifier{backend: &BeeepBackend{}}
}

// SessionComplete sends a notification that a session has completed.
// If workspace is provided, it's used in the title; otherwise uses sessionID.
func (n *Notifier) SessionComplete(sessionID, workspace string) error {
	name := workspace
	if name == "" {
		name = sessionID
	}
	title := fmt.Sprintf("Agent Complete: %s", name)
	message := fmt.Sprintf("Session %s completed", sessionID)
	return n.backend.Notify(title, message, "")
}

// Error sends an error notification.
func (n *Notifier) Error(message string) error {
	return n.backend.Notify("OpenCode Error", message, "")
}
