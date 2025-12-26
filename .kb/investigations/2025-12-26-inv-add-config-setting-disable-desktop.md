# Investigation: Add Config Setting to Disable Desktop Notifications

**Status:** Complete
**Created:** 2025-12-26

## TLDR

Added a `notifications.enabled` setting to `~/.orch/config.yaml` allowing users to disable desktop notifications. Created `pkg/userconfig` package for global user config and updated `pkg/notify` to check this setting.

## What I Tried

1. Analyzed existing config structure in `pkg/config/config.go` (project-level config)
2. Analyzed `pkg/account/account.go` for global config pattern (`~/.orch/accounts.yaml`)
3. Found notification code in `pkg/notify/notify.go` and `pkg/opencode/service.go`
4. Created new `pkg/userconfig` package to manage `~/.orch/config.yaml`
5. Updated `pkg/notify` to check the config before sending notifications

## What I Observed

- Project config is at `.orch/config.yaml` (per-project)
- Global config already exists at `~/.orch/config.yaml` with `backend` and `auto_export_transcript` settings
- No Go code was loading the global config - needed to create a new package
- Notifications are sent from `pkg/opencode/service.go` via `notify.Notifier.SessionComplete()`

## Implementation

### New Package: `pkg/userconfig`

- `Load()` - Loads config from `~/.orch/config.yaml`, returns defaults if file doesn't exist
- `Save()` - Saves config to `~/.orch/config.yaml`
- `DefaultConfig()` - Returns config with notifications enabled by default
- `Config.NotificationsEnabled()` - Returns whether notifications are enabled (defaults to true)

### Updated Package: `pkg/notify`

- Added `enabled` field to `Notifier` struct
- `Default()` now checks userconfig and sets `enabled` accordingly
- `SessionComplete()` and `Error()` return nil immediately if disabled
- Added `IsEnabled()` and `SetEnabled()` methods for control

### Config Format

```yaml
# ~/.orch/config.yaml
backend: opencode
auto_export_transcript: true
notifications:
  enabled: false  # Set to false to disable desktop notifications
```

## Test Performed

- Ran `go test ./pkg/userconfig/...` - All 6 tests pass
- Ran `go test ./pkg/notify/...` - All 8 tests pass
- Ran `go build ./...` - Build succeeds

## Conclusion

The implementation is complete. Users can now disable desktop notifications by adding `notifications.enabled: false` to their `~/.orch/config.yaml` file. The setting defaults to `true` (enabled) for backwards compatibility.
