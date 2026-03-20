package gates

// ConcurrencyCheck returns (activeCount, maxAgents) for capacity validation.
// Nil means no concurrency check is available (skip the gate).
type ConcurrencyCheck func() (activeCount int, maxAgents int)
