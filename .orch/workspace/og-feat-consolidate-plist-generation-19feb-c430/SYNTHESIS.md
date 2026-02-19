# Synthesis: Consolidate Plist Generation (orch-go-1107)

## What Changed

Consolidated all plist XML generation into `pkg/daemonconfig/plist.go`, eliminating three copies of the same logic scattered across `cmd/orch/`.

### Before
- `cmd/orch/config_cmd.go`: `PlistData` type, `plistTemplate` const, `findOrchPath()`, `getPlistPath()`, template rendering
- `cmd/orch/serve_system.go`: `PlistDataAPI` type, `plistTemplateAPI` const, `findOrchPathForAPI()`, `buildPlistDataForAPI()`, template rendering
- `cmd/orch/doctor.go`: `parsePlistValues()` function (90 lines)

### After
- `pkg/daemonconfig/plist.go`: Single source of truth — `PlistData`, `PlistTemplate`, `GeneratePlistXML()`, `GetPlistPath()`, `FindOrchPath()`, `BuildPATH()`, `ParsePlistValues()`
- `cmd/orch/config_cmd.go`: Uses `daemonconfig.*` for all plist operations
- `cmd/orch/serve_system.go`: Uses `buildPlistData()` + `daemonconfig.GeneratePlistXML()` (no more duplicated type/template/helpers)
- `cmd/orch/doctor.go`: Uses `daemonconfig.ParsePlistValues()` instead of local copy

## Lines Removed
- ~200 lines of duplicated code removed across 3 files
- 0 new lines in cmd/orch (net reduction)

## Test Coverage
- `pkg/daemonconfig/plist_test.go`: 8 tests covering all exported functions
- `cmd/orch/doctor_test.go`: 3 existing tests updated to use `daemonconfig.ParsePlistValues()`
- All tests pass: `go test ./pkg/daemonconfig/ && go test ./cmd/orch/`

## Decisions
- Kept `buildPlistData(cfg)` in `config_cmd.go` since it bridges `userconfig.Config` → `daemonconfig.PlistData` (caller-level concern)
- Left `getDoctorPlistPath()` in doctor.go — it's for `com.orch.doctor` (different service), not `com.orch.daemon`
- Left `runDoctorInstall()` inline plist in doctor.go — different service with different template

## Verification
```bash
go build ./cmd/orch/   # ✅
go vet ./cmd/orch/     # ✅
go test ./pkg/daemonconfig/  # ✅ 8/8 pass
go test ./cmd/orch/ -run "TestParsePlist|TestConfigDrift"  # ✅ all pass
```
