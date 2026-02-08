package boundedlifetimetest

import (
	"context"
	"os/exec"
	"time"
)

func badGoroutine() {
	go func() { // want "goroutine must accept a context.Context parameter"
	}()
}

func goodGoroutine(ctx context.Context) {
	go func(ctx context.Context) {
		_ = ctx
	}(ctx)
}

func badExec() {
	_ = exec.Command("echo", "hello") // want "use exec.CommandContext with context.WithTimeout instead of exec.Command"
}

func badCommandContext(parent context.Context) {
	_ = exec.CommandContext(parent, "echo", "hello") // want "exec.CommandContext must receive a context created by context.WithTimeout"
}

func goodCommandContext(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "echo", "hello")
}

type memoryCache struct {
	entries map[string]string // want "map cache field \"entries\" requires max-size bound and eviction logic"
}

type resultCache struct {
	cache      map[string]string
	maxEntries int
}

func (c *resultCache) evictOne() {
	for key := range c.cache {
		delete(c.cache, key)
		return
	}
}

type noEvictionCache struct {
	maxEntries int
	entries    map[string]string // want "map cache field \"entries\" requires eviction logic that deletes entries"
}

func (c *noEvictionCache) set(key, value string) {
	c.entries[key] = value
}

type noBoundCache struct {
	entries map[string]string // want "map cache field \"entries\" requires max-size bound"
}

func (c *noBoundCache) evictOne() {
	for key := range c.entries {
		delete(c.entries, key)
		return
	}
}

type mapStore struct {
	entries map[string]string
}
