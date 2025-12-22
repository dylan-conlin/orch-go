# {{.ProjectName}}

[Brief project description - fill in]

## Development

```bash
uv sync              # Install dependencies
uv run {{.ProjectName}} --help  # Run CLI
uv run pytest        # Run tests
```

## Architecture

```
src/{{.ProjectName}}/
├── __init__.py
├── cli.py           # CLI entry point
├── [module].py      # Core modules
└── [module]_test.py
```

[Describe key directories and modules]

## Commands

```bash
{{.ProjectName}} --help          # Show available commands
{{.ProjectName}} [command]       # Run specific command
```

[List main CLI commands]

## Testing

```bash
uv run pytest              # Run all tests
uv run pytest -v           # Verbose output
uv run pytest --cov        # With coverage
```

## Gotchas

[Document project-specific gotchas]

- [Gotcha 1]
- [Gotcha 2]
