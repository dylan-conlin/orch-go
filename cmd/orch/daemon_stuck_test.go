package main

import (
	"testing"
	"time"
)

func TestCheckDaemonStuck(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		lastSpawn      time.Time
		lastCompletion time.Time
		lastNotify     time.Time
		threshold      time.Duration
		cooldown       time.Duration
		want           bool
	}{
		{
			name:           "stuck: both timestamps old, no recent notification",
			lastSpawn:      now.Add(-15 * time.Minute),
			lastCompletion: now.Add(-12 * time.Minute),
			lastNotify:     time.Time{}, // zero = never notified
			threshold:      10 * time.Minute,
			cooldown:       30 * time.Minute,
			want:           true,
		},
		{
			name:           "not stuck: recent spawn",
			lastSpawn:      now.Add(-5 * time.Minute),
			lastCompletion: now.Add(-15 * time.Minute),
			lastNotify:     time.Time{},
			threshold:      10 * time.Minute,
			cooldown:       30 * time.Minute,
			want:           false,
		},
		{
			name:           "not stuck: recent completion",
			lastSpawn:      now.Add(-15 * time.Minute),
			lastCompletion: now.Add(-5 * time.Minute),
			lastNotify:     time.Time{},
			threshold:      10 * time.Minute,
			cooldown:       30 * time.Minute,
			want:           false,
		},
		{
			name:           "not stuck: zero lastSpawn (daemon just started)",
			lastSpawn:      time.Time{},
			lastCompletion: now.Add(-15 * time.Minute),
			lastNotify:     time.Time{},
			threshold:      10 * time.Minute,
			cooldown:       30 * time.Minute,
			want:           false,
		},
		{
			name:           "not stuck: zero lastCompletion (no completions yet)",
			lastSpawn:      now.Add(-15 * time.Minute),
			lastCompletion: time.Time{},
			lastNotify:     time.Time{},
			threshold:      10 * time.Minute,
			cooldown:       30 * time.Minute,
			want:           false,
		},
		{
			name:           "cooldown active: recently notified",
			lastSpawn:      now.Add(-15 * time.Minute),
			lastCompletion: now.Add(-15 * time.Minute),
			lastNotify:     now.Add(-10 * time.Minute), // within 30min cooldown
			threshold:      10 * time.Minute,
			cooldown:       30 * time.Minute,
			want:           false,
		},
		{
			name:           "cooldown expired: can notify again",
			lastSpawn:      now.Add(-45 * time.Minute),
			lastCompletion: now.Add(-45 * time.Minute),
			lastNotify:     now.Add(-35 * time.Minute), // cooldown expired
			threshold:      10 * time.Minute,
			cooldown:       30 * time.Minute,
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkDaemonStuck(tt.lastSpawn, tt.lastCompletion, tt.lastNotify, tt.threshold, tt.cooldown)
			if got != tt.want {
				t.Errorf("checkDaemonStuck() = %v, want %v", got, tt.want)
			}
		})
	}
}
