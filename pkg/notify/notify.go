// Package notify provides desktop notification functionality.
//
// Notifications can be disabled via ~/.orch/config.yaml:
//
//	notifications:
//	  enabled: false
package notify

import (
	"fmt"

	"github.com/dylan-conlin/orch-go/pkg/userconfig"
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
	enabled bool
}

// New creates a new Notifier with the given backend.
// The notifier is enabled by default.
func New(backend Backend) *Notifier {
	return &Notifier{backend: backend, enabled: true}
}

// Default creates a Notifier with the default beeep backend.
// Checks ~/.orch/config.yaml for notifications.enabled setting.
func Default() *Notifier {
	enabled := true
	if cfg, err := userconfig.Load(); err == nil {
		enabled = cfg.NotificationsEnabled()
	}
	return &Notifier{backend: &BeeepBackend{}, enabled: enabled}
}

// SessionComplete sends a notification that a session has completed.
// If workspace is provided, it's used in the title; otherwise uses sessionID.
// Returns nil immediately if notifications are disabled.
func (n *Notifier) SessionComplete(sessionID, workspace string) error {
	if !n.enabled {
		return nil
	}
	name := workspace
	if name == "" {
		name = sessionID
	}
	title := fmt.Sprintf("Agent Complete: %s", name)
	message := fmt.Sprintf("Session %s completed", sessionID)
	return n.backend.Notify(title, message, "")
}

// Error sends an error notification.
// Returns nil immediately if notifications are disabled.
func (n *Notifier) Error(message string) error {
	if !n.enabled {
		return nil
	}
	return n.backend.Notify("OpenCode Error", message, "")
}

// IsEnabled returns whether notifications are enabled.
func (n *Notifier) IsEnabled() bool {
	return n.enabled
}

// SetEnabled sets whether notifications are enabled.
func (n *Notifier) SetEnabled(enabled bool) {
	n.enabled = enabled
}
