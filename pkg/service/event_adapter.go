package service

import (
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
)

// EventLoggerAdapter adapts pkg/events.Logger to the EventLogger interface.
type EventLoggerAdapter struct {
	logger *events.Logger
}

// NewEventLoggerAdapter creates a new adapter for event logging.
func NewEventLoggerAdapter(logger *events.Logger) *EventLoggerAdapter {
	return &EventLoggerAdapter{logger: logger}
}

// LogServiceCrashed logs a service crash event.
func (a *EventLoggerAdapter) LogServiceCrashed(serviceName, projectPath string, oldPID, newPID int) error {
	return a.logger.Log(events.Event{
		Type:      events.EventTypeServiceCrashed,
		SessionID: serviceName,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"service_name": serviceName,
			"project_path": projectPath,
			"old_pid":      oldPID,
			"new_pid":      newPID,
		},
	})
}

// LogServiceRestarted logs a service restart event.
func (a *EventLoggerAdapter) LogServiceRestarted(serviceName, projectPath string, newPID, restartCount int, autoRestart bool) error {
	return a.logger.Log(events.Event{
		Type:      events.EventTypeServiceRestarted,
		SessionID: serviceName,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"service_name":  serviceName,
			"project_path":  projectPath,
			"new_pid":       newPID,
			"restart_count": restartCount,
			"auto_restart":  autoRestart,
		},
	})
}

// LogServiceStarted logs a service start event.
func (a *EventLoggerAdapter) LogServiceStarted(serviceName, projectPath string, pid int) error {
	return a.logger.Log(events.Event{
		Type:      events.EventTypeServiceStarted,
		SessionID: serviceName,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"service_name": serviceName,
			"project_path": projectPath,
			"pid":          pid,
		},
	})
}

// LogServiceUnresponsive logs a service unresponsive event (process alive but not serving).
func (a *EventLoggerAdapter) LogServiceUnresponsive(serviceName, projectPath string, pid, consecutiveFailures int) error {
	return a.logger.Log(events.Event{
		Type:      events.EventTypeServiceUnresponsive,
		SessionID: serviceName,
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"service_name":         serviceName,
			"project_path":         projectPath,
			"pid":                  pid,
			"consecutive_failures": consecutiveFailures,
		},
	})
}
