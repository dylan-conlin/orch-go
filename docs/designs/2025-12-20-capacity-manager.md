# Design: Capacity Manager for Multi-Account Coordination

**Status:** Draft
**Date:** 2025-12-20
**Author:** Agent

## Problem Statement

The daemon needs to coordinate spawning across multiple Claude Max accounts to:
1. Avoid rate limits by switching to accounts with available capacity
2. Track in-flight agents per account (not just API usage)
3. Queue requests when all accounts are exhausted
4. Emit events for monitoring capacity changes

### Success Criteria

- Can acquire slot from least-used account
- Queues when all accounts at limit
- Auto-switches to fresh account when current exhausted
- Thread-safe for concurrent access
- Unit tests with mock usage data

## Approach

### Core Design

Create a new `pkg/capacity/` package with:

1. **CapacityManager** - Central coordinator for account capacity
2. **Slot** - Represents an acquired agent slot
3. **Options** - Configuration for thresholds and timeouts

### Key Components

```go
// pkg/capacity/manager.go

type CapacityManager struct {
    mu           sync.Mutex
    accounts     map[string]*AccountState
    inFlight     map[string]int        // accountName -> active agent count
    capacityFunc CapacityFunc          // for testing
    eventHandler EventHandler          // for monitoring
    threshold    float64               // switch threshold (default 80%)
    maxPerAcct   int                   // max concurrent per account
}

type AccountState struct {
    Name     string
    Capacity *account.CapacityInfo
    InFlight int
    LastUsed time.Time
}

type Slot struct {
    AccountName string
    AcquiredAt  time.Time
}

// Core operations
func (m *CapacityManager) AcquireSlot(ctx context.Context) (*Slot, error)
func (m *CapacityManager) ReleaseSlot(slot *Slot) error
func (m *CapacityManager) Status() []AccountState
func (m *CapacityManager) Refresh() error
```

### Account Selection Algorithm

When `AcquireSlot` is called:

1. Refresh capacity for all accounts (if stale)
2. Filter accounts below threshold (80% by default)
3. Filter accounts below max concurrent limit
4. Select account with most remaining capacity
5. If no accounts available, queue with timeout
6. Increment in-flight count
7. Emit `capacity.acquired` event
8. Return Slot

When `ReleaseSlot` is called:

1. Decrement in-flight count
2. Emit `capacity.released` event

### Queue Behavior

When all accounts are exhausted:

```go
func (m *CapacityManager) AcquireSlot(ctx context.Context) (*Slot, error) {
    m.mu.Lock()
    
    for {
        // Try to find available account
        acct := m.selectBestAccount()
        if acct != nil {
            m.inFlight[acct.Name]++
            m.mu.Unlock()
            return &Slot{AccountName: acct.Name, AcquiredAt: time.Now()}, nil
        }
        
        // No account available - wait for release or timeout
        m.mu.Unlock()
        
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-m.waitChan:
            // Account may be available now
            m.mu.Lock()
            continue
        }
    }
}
```

### Event Emission

```go
type Event struct {
    Type      string                 // capacity.acquired, capacity.released, capacity.low, capacity.exhausted
    Timestamp time.Time
    Account   string
    Data      map[string]interface{}
}

type EventHandler func(Event)
```

### Thread Safety

- Use `sync.Mutex` for all state access
- Use `sync.Cond` or channel for queue waiting
- Capacity refresh happens under lock (short-lived)

### Configuration

```go
type Options struct {
    Threshold    float64       // Switch at this % remaining (default 80%)
    MaxPerAcct   int           // Max concurrent per account (default 3)
    QueueTimeout time.Duration // Max wait in queue (default 5m)
    RefreshRate  time.Duration // How often to refresh capacity (default 1m)
}
```

## Data Model

### AccountState

```go
type AccountState struct {
    Name          string
    Email         string
    Capacity      *account.CapacityInfo
    InFlight      int
    TotalAcquired int       // lifetime count
    LastUsed      time.Time
    LastRefresh   time.Time
}
```

## Testing Strategy

1. **Unit tests** with mock capacity function
2. **Concurrent tests** with race detector
3. **Queue timeout tests**
4. **Event emission tests**

Mock approach:
```go
type CapacityFunc func(name string) (*account.CapacityInfo, error)

func NewWithCapacityFunc(fn CapacityFunc, opts Options) *CapacityManager {
    return &CapacityManager{
        capacityFunc: fn,
        // ...
    }
}
```

## Implementation Plan

1. Create `pkg/capacity/manager.go` with CapacityManager struct
2. Implement `AcquireSlot` with selection algorithm
3. Implement `ReleaseSlot` with event emission
4. Implement queue behavior with context timeout
5. Add thread-safe tests with race detector
6. Wire into daemon for integration

## Alternatives Considered

### Alternative 1: Extend account package

Put capacity management in `pkg/account/`. Rejected because:
- account package is already large (700+ lines)
- Capacity management is a distinct concern
- Cleaner separation of responsibilities

### Alternative 2: Global semaphore approach

Use a simple semaphore per account. Rejected because:
- Doesn't account for actual API capacity
- No intelligent account selection
- No queue behavior

## Open Questions

None - design is straightforward based on requirements.

## Security Considerations

- No new credentials handling (uses existing account tokens)
- Thread safety prevents race conditions
- Queue timeout prevents indefinite blocking

## Performance Requirements

- `AcquireSlot` should complete in <100ms when accounts available
- Capacity refresh should be cached (not on every acquire)
- Queue wait uses efficient signaling (not polling)
