package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// logDaemonEvent logs an event to the events file and prints a warning on failure.
// This consolidates the repeated pattern of constructing an events.Event, calling
// logger.Log, and printing a warning if it fails.
func logDaemonEvent(logger *events.Logger, eventType string, data map[string]interface{}) {
	event := events.Event{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	if err := logger.Log(event); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log %s event: %v\n", eventType, err)
	}
}

// subsystemResult describes the outcome of a periodic daemon subsystem (cleanup,
// recovery, server recovery, dead session detection). It provides a uniform
// interface for the three-branch logging pattern:
//
//	if err       → log error to stderr + event file
//	else if hit  → log info to stdout + event file
//	else verbose → log debug to stdout (no event)
type subsystemResult struct {
	// Name is the human-readable subsystem name (e.g. "Cleanup", "Recovery").
	Name string
	// EventType is the event type for the events file (e.g. "daemon.cleanup").
	EventType string
	// Error from the subsystem, if any.
	Error error
	// Message is the human-readable summary produced by the subsystem.
	Message string
	// HasActivity is true when the subsystem did meaningful work (e.g. count > 0).
	HasActivity bool
	// Data is the event payload. Always logged when Error != nil or HasActivity.
	Data map[string]interface{}
}

// logSubsystemResult handles the three-branch logging pattern shared by all
// periodic daemon subsystems. It replaces ~35 lines of boilerplate per call site.
func logSubsystemResult(logger *events.Logger, timestamp string, verbose bool, r subsystemResult) {
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "[%s] %s error: %v\n", timestamp, r.Name, r.Error)
		logDaemonEvent(logger, r.EventType, r.Data)
		return
	}
	if r.HasActivity {
		fmt.Printf("[%s] %s: %s\n", timestamp, r.Name, r.Message)
		logDaemonEvent(logger, r.EventType, r.Data)
		return
	}
	if verbose {
		fmt.Printf("[%s] %s: %s\n", timestamp, r.Name, r.Message)
	}
}
