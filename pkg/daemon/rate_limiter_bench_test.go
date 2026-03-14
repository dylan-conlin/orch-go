package daemon

import (
	"context"
	"testing"
	"time"
)

func BenchmarkRateLimiter_CanSpawn_Empty(b *testing.B) {
	r := NewRateLimiter(20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.CanSpawn()
	}
}

func BenchmarkRateLimiter_CanSpawn_HalfFull(b *testing.B) {
	r := NewRateLimiter(20)
	for i := 0; i < 10; i++ {
		r.RecordSpawn()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.CanSpawn()
	}
}

func BenchmarkRateLimiter_CanSpawn_Full(b *testing.B) {
	r := NewRateLimiter(20)
	for i := 0; i < 20; i++ {
		r.RecordSpawn()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.CanSpawn()
	}
}

func BenchmarkRateLimiter_RecordSpawn(b *testing.B) {
	r := NewRateLimiter(20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.RecordSpawn()
	}
}

func BenchmarkWorkerPool_TryAcquire_Available(b *testing.B) {
	p := NewWorkerPool(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slot := p.TryAcquire()
		if slot != nil {
			p.Release(slot)
		}
	}
}

func BenchmarkWorkerPool_TryAcquire_AtCapacity(b *testing.B) {
	p := NewWorkerPool(1)
	slot := p.TryAcquire()
	_ = slot
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.TryAcquire() // should return nil every time
	}
}

func BenchmarkWorkerPool_Acquire_Available(b *testing.B) {
	p := NewWorkerPool(1000)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slot, _ := p.Acquire(ctx)
		if slot != nil {
			p.Release(slot)
		}
	}
}

func BenchmarkWorkerPool_Reconcile(b *testing.B) {
	p := NewWorkerPool(10)
	// Pre-fill with 5 slots
	for i := 0; i < 5; i++ {
		p.TryAcquire()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Reconcile(5) // No change - same count
	}
}

func BenchmarkCheckPreSpawnGates_AllPass(b *testing.B) {
	d := &Daemon{
		Config:      Config{MaxSpawnsPerHour: 20},
		RateLimiter: NewRateLimiter(20),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.CheckPreSpawnGates()
	}
}

func BenchmarkCheckPreSpawnGates_RateLimited(b *testing.B) {
	d := &Daemon{
		Config:      Config{MaxSpawnsPerHour: 2},
		RateLimiter: NewRateLimiter(2),
	}
	// Fill up the rate limiter
	d.RateLimiter.RecordSpawn()
	d.RateLimiter.RecordSpawn()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.CheckPreSpawnGates()
	}
}

func BenchmarkRateLimiter_Prune(b *testing.B) {
	r := NewRateLimiter(100)
	// Set up a scenario with old + new entries
	baseTime := time.Now()
	r.nowFunc = func() time.Time { return baseTime.Add(-2 * time.Hour) }
	for i := 0; i < 50; i++ {
		r.RecordSpawn()
	}
	r.nowFunc = func() time.Time { return baseTime }
	for i := 0; i < 50; i++ {
		r.RecordSpawn()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.prune()
	}
}
