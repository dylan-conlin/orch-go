package cache

import (
	"strings"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	bounds := NewCache(128, 5*time.Second)

	if bounds.MaxSize() != 128 {
		t.Fatalf("MaxSize() = %d, want 128", bounds.MaxSize())
	}
	if bounds.TTL() != 5*time.Second {
		t.Fatalf("TTL() = %v, want 5s", bounds.TTL())
	}
}

func TestNewCacheE(t *testing.T) {
	t.Run("invalid max size", func(t *testing.T) {
		_, err := NewCacheE(0, time.Second)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "maxSize must be > 0") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid ttl", func(t *testing.T) {
		_, err := NewCacheE(1, 0)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "ttl must be > 0") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNewNamedCacheEIncludesName(t *testing.T) {
	_, err := NewNamedCacheE("example cache", 0, time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "example cache maxSize must be > 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}
