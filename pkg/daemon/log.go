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

// maxLogSize is the threshold above which the daemon log is rotated on startup.
const maxLogSize = 10 * 1024 * 1024 // 10 MB

// rotateIfNeeded checks the log file size and rotates it if above maxLogSize.
// Rotation renames daemon.log → daemon.log.1 (overwriting any previous backup).
func rotateIfNeeded(logPath string) {
	info, err := os.Stat(logPath)
	if err != nil {
		return // file doesn't exist or can't stat — nothing to rotate
	}
	if info.Size() < maxLogSize {
		return
	}
	// Rename current log to .1 (os.Rename overwrites destination on Unix)
	_ = os.Rename(logPath, logPath+".1")
}

// stdoutIsLogFile reports whether stdout is already pointing at the given log file.
// This happens when launchd's StandardOutPath redirects stdout to daemon.log —
// writing to both stdout and the file directly would double every log line.
func stdoutIsLogFile(logPath string) bool {
	stdoutInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	logInfo, err := os.Stat(logPath)
	if err != nil {
		return false
	}
	return os.SameFile(stdoutInfo, logInfo)
}

// NewDaemonLogger creates a logger that writes to both stdout and ~/.orch/daemon.log.
// If the log file cannot be opened, falls back to stdout only.
// Rotates the log file on startup if it exceeds 10 MB.
//
// When launchd redirects stdout to daemon.log (StandardOutPath), the logger
// detects this and skips the direct file write to avoid double logging.
func NewDaemonLogger() *DaemonLogger {
	logPath := DaemonLogPath()
	if logPath == "" {
		return &DaemonLogger{out: os.Stdout}
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return &DaemonLogger{out: os.Stdout}
	}

	// Check before rotation: if stdout already points to daemon.log
	// (launchd StandardOutPath), skip direct file write to avoid doubling.
	if stdoutIsLogFile(logPath) {
		return &DaemonLogger{out: os.Stdout}
	}

	// Rotate before opening
	rotateIfNeeded(logPath)

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
