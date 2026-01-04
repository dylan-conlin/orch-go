package testutil

import (
	"sync"
	"testing"
	"time"
)

func TestWaitFor_SucceedsImmediately(t *testing.T) {
	WaitFor(t, func() bool {
		return true
	}, "immediate condition")
}

func TestWaitFor_SucceedsAfterDelay(t *testing.T) {
	start := time.Now()
	ready := make(chan struct{})

	go func() {
		time.Sleep(50 * time.Millisecond)
		close(ready)
	}()

	var done bool
	go func() {
		<-ready
		done = true
	}()

	WaitFor(t, func() bool {
		return done
	}, "delayed condition")

	elapsed := time.Since(start)
	if elapsed < 50*time.Millisecond {
		t.Errorf("WaitFor returned too quickly: %v", elapsed)
	}
}

func TestWaitForWithTimeout_TimesOut(t *testing.T) {
	// Create a fake T that captures the fatal call
	fakeT := &fakeT{}

	// WaitForWithTimeout calls Fatalf which panics, so we recover
	defer func() {
		if r := recover(); r != nil {
			// Expected panic from fakeT.Fatalf
		}
	}()

	WaitForWithTimeout(fakeT, func() bool {
		return false // Never true
	}, 50*time.Millisecond, "impossible condition")

	if !fakeT.failed {
		t.Error("Expected WaitForWithTimeout to fail on timeout")
	}
	if fakeT.failMsg == "" {
		t.Error("Expected failure message")
	}
}

func TestEventually_ReturnsTrue(t *testing.T) {
	var ready bool
	go func() {
		time.Sleep(20 * time.Millisecond)
		ready = true
	}()

	result := Eventually(func() bool {
		return ready
	}, 200*time.Millisecond)

	if !result {
		t.Error("Eventually should return true when condition becomes true")
	}
}

func TestEventually_ReturnsFalse(t *testing.T) {
	result := Eventually(func() bool {
		return false
	}, 50*time.Millisecond)

	if result {
		t.Error("Eventually should return false when condition never becomes true")
	}
}

func TestWaitForCount(t *testing.T) {
	var count int
	var mu sync.Mutex

	// Simulate async increments
	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			count++
			mu.Unlock()
		}
	}()

	WaitForCount(t, func() int {
		mu.Lock()
		defer mu.Unlock()
		return count
	}, 3, "3 increments")
}

// fakeT implements testing.TB for testing timeout behavior
type fakeT struct {
	testing.TB
	failed  bool
	failMsg string
}

func (f *fakeT) Helper() {}

func (f *fakeT) Fatalf(format string, args ...interface{}) {
	f.failed = true
	f.failMsg = format
	panic("fatalf called") // Simulate test termination
}
