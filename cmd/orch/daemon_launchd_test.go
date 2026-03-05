package main

import (
	"testing"
)

func TestGetUID(t *testing.T) {
	uid, err := getUID()
	if err != nil {
		t.Fatalf("getUID() error: %v", err)
	}
	if uid == "" {
		t.Fatal("getUID() returned empty string")
	}
	// UID should be a numeric string
	for _, c := range uid {
		if c < '0' || c > '9' {
			t.Fatalf("getUID() returned non-numeric: %s", uid)
		}
	}
}

func TestIsServiceLoaded(t *testing.T) {
	// This is a smoke test — just verify it doesn't panic.
	// The result depends on whether the daemon is actually loaded.
	_ = isServiceLoaded()
}
