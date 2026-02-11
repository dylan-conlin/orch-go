// Package process provides utilities for managing OS processes.
package process

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProcessStartTime returns the start time of a process by PID.
// Uses ps to query the elapsed time (etime) and calculates the start time.
func ProcessStartTime(pid int) (time.Time, error) {
	if pid <= 0 {
		return time.Time{}, fmt.Errorf("invalid PID: %d", pid)
	}

	// Use ps to get elapsed time in seconds
	cmd := exec.Command("ps", "-o", "etime=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get process start time for PID %d: %w", pid, err)
	}

	elapsed, err := parseEtime(strings.TrimSpace(string(output)))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse etime for PID %d: %w", pid, err)
	}

	return time.Now().Add(-elapsed), nil
}

// parseEtime parses ps etime format: [[DD-]HH:]MM:SS
func parseEtime(etime string) (time.Duration, error) {
	if etime == "" {
		return 0, fmt.Errorf("empty etime")
	}

	var days, hours, minutes, seconds int

	// Check for days component (DD-...)
	if idx := strings.Index(etime, "-"); idx != -1 {
		d, err := strconv.Atoi(etime[:idx])
		if err != nil {
			return 0, fmt.Errorf("failed to parse days from %q: %w", etime, err)
		}
		days = d
		etime = etime[idx+1:]
	}

	parts := strings.Split(etime, ":")
	switch len(parts) {
	case 3: // HH:MM:SS
		h, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("failed to parse hours from %q: %w", etime, err)
		}
		hours = h
		m, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes from %q: %w", etime, err)
		}
		minutes = m
		s, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds from %q: %w", etime, err)
		}
		seconds = s
	case 2: // MM:SS
		m, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("failed to parse minutes from %q: %w", etime, err)
		}
		minutes = m
		s, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("failed to parse seconds from %q: %w", etime, err)
		}
		seconds = s
	default:
		return 0, fmt.Errorf("unexpected etime format: %q", etime)
	}

	total := time.Duration(days)*24*time.Hour +
		time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second

	return total, nil
}
