// Package daemon provides autonomous overnight processing capabilities.
package daemon

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// DaemonLogger writes daemon output to both stdout and a persistent log file.
// The log file (~/.orch/daemon.log) survives process detachment and orphaned
// file descriptors — unlike stdout which depends on the launching process.
type DaemonLogger struct {
	file *os.File
	out  io.Writer
}

// DaemonLogPath returns the path to the daemon log file.
func DaemonLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".orch", "daemon.log")
}

// NewDaemonLogger creates a logger that writes to both stdout and ~/.orch/daemon.log.
// If the log file cannot be opened, falls back to stdout only.
func NewDaemonLogger() *DaemonLogger {
	logPath := DaemonLogPath()
	if logPath == "" {
		return &DaemonLogger{out: os.Stdout}
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return &DaemonLogger{out: os.Stdout}
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return &DaemonLogger{out: os.Stdout}
	}

	return &DaemonLogger{
		file: f,
		out:  io.MultiWriter(os.Stdout, f),
	}
}

// Printf writes formatted output to both stdout and the log file.
func (l *DaemonLogger) Printf(format string, a ...interface{}) {
	fmt.Fprintf(l.out, format, a...)
}

// Errorf writes formatted output to both stderr and the log file.
func (l *DaemonLogger) Errorf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprint(os.Stderr, msg)
	if l.file != nil {
		fmt.Fprint(l.file, msg)
	}
}

// Stamp writes a timestamped log line (e.g., "[15:04:05] message\n").
func (l *DaemonLogger) Stamp(format string, a ...interface{}) {
	ts := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(l.out, "[%s] %s\n", ts, msg)
}

// Close closes the log file. Safe to call multiple times.
func (l *DaemonLogger) Close() {
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
}

// Writer returns the underlying io.Writer for use with fmt.Fprintf.
func (l *DaemonLogger) Writer() io.Writer {
	return l.out
}
