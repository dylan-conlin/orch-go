// Package testutil provides utilities for hardening flaky tests.
package testutil

import (
	"testing"
	"time"
)

// DefaultTimeout is the default timeout for WaitFor operations.
const DefaultTimeout = 2 * time.Second

// DefaultInterval is the default polling interval for WaitFor operations.
const DefaultInterval = 10 * time.Millisecond

// WaitFor polls the condition function until it returns true or the timeout is reached.
// This is the recommended way to wait for asynchronous operations in tests,
// replacing arbitrary time.Sleep calls that lead to flaky tests.
//
// Example usage:
//
//	var result string
//	go func() {
//	    // async operation
//	    result = "done"
//	}()
//	testutil.WaitFor(t, func() bool {
//	    return result == "done"
//	}, "result to be set")
func WaitFor(t testing.TB, condition func() bool, description string) {
	t.Helper()
	WaitForWithTimeout(t, condition, DefaultTimeout, description)
}

// WaitForWithTimeout polls the condition function until it returns true or the timeout is reached.
// Use this when you need a custom timeout.
func WaitForWithTimeout(t testing.TB, condition func() bool, timeout time.Duration, description string) {
	t.Helper()
	WaitForWithTimeoutAndInterval(t, condition, timeout, DefaultInterval, description)
}

// WaitForWithTimeoutAndInterval polls the condition function until it returns true,
// checking at the specified interval until the timeout is reached.
func WaitForWithTimeoutAndInterval(t testing.TB, condition func() bool, timeout, interval time.Duration, description string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		if condition() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s (timeout: %v)", description, timeout)
		}
		time.Sleep(interval)
	}
}

// Eventually returns true if the condition becomes true within the timeout.
// Unlike WaitFor, this does not fail the test on timeout - it returns false.
// Use this when you want to check a condition without failing immediately.
func Eventually(condition func() bool, timeout time.Duration) bool {
	return EventuallyWithInterval(condition, timeout, DefaultInterval)
}

// EventuallyWithInterval returns true if the condition becomes true within the timeout,
// checking at the specified interval.
func EventuallyWithInterval(condition func() bool, timeout, interval time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if condition() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(interval)
	}
}

// WaitForCount waits until the counter function returns at least the expected count.
// This is useful for waiting for a specific number of async callbacks.
//
// Example usage:
//
//	var count int
//	var mu sync.Mutex
//	testutil.WaitForCount(t, func() int {
//	    mu.Lock()
//	    defer mu.Unlock()
//	    return count
//	}, 3, "3 callbacks")
func WaitForCount(t testing.TB, counter func() int, expected int, description string) {
	t.Helper()
	WaitForWithTimeout(t, func() bool {
		return counter() >= expected
	}, DefaultTimeout, description)
}

// AssertEventually asserts that the condition eventually becomes true.
// This is like WaitFor but reads better for assertion-style tests.
func AssertEventually(t testing.TB, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	description := "condition to be true"
	if len(msgAndArgs) > 0 {
		if s, ok := msgAndArgs[0].(string); ok {
			description = s
		}
	}
	WaitForWithTimeout(t, condition, timeout, description)
}

// YieldForGoroutine yields the processor to allow other goroutines to start.
// This is useful in tests that need to ensure a goroutine has started blocking
// on a condition variable or channel before proceeding with the test.
//
// This uses runtime.Gosched() followed by a minimal sleep to handle scheduler
// variations. It's more reliable than a fixed sleep alone.
//
// Example usage:
//
//	var wg sync.WaitGroup
//	wg.Add(1)
//	go func() {
//	    defer wg.Done()
//	    slot, _ = pool.Acquire(ctx) // blocks
//	}()
//	testutil.YieldForGoroutine() // Let goroutine start blocking
//	pool.Release(slot1)          // Now release
//	wg.Wait()
func YieldForGoroutine() {
	// Yield to scheduler to let other goroutines run
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond)
	}
}
