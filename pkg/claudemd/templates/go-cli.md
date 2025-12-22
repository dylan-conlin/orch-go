# {{.ProjectName}}

[Brief project description - fill in]

## Development

```bash
make build   # Build binary
make test    # Run tests
make install # Install to ~/bin
```

## Architecture

```
cmd/
├── {{.ProjectName}}/    # CLI entry point
│   └── main.go

pkg/
├── [package]/           # Core packages
│   ├── [package].go
│   └── [package]_test.go
```

[Describe key directories and packages]

## Commands

[List main CLI commands]

```bash
{{.ProjectName}} --help  # Show available commands
```

## Testing

```bash
go test ./...            # Run all tests
go test -v ./pkg/...     # Verbose package tests
```

## Gotchas

[Document project-specific gotchas]

- [Gotcha 1]
- [Gotcha 2]
