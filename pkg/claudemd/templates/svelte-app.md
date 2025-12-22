# {{.ProjectName}}

[Brief project description - fill in]

## Development

```bash
bun install   # Install dependencies
bun dev       # Start dev server (port {{.PortWeb}})
bun build     # Build for production
bun test      # Run tests
```

## Architecture

```
src/
├── routes/          # SvelteKit routes
│   ├── +page.svelte
│   └── +layout.svelte
├── lib/             # Shared components and utilities
│   ├── components/
│   └── stores/
└── app.html         # HTML template
```

[Describe key directories and components]

## Routes

| Route | Description |
|-------|-------------|
| `/` | [Home page] |
| `/[route]` | [Description] |

## Ports

- **Dev server:** {{.PortWeb}}
{{if .PortAPI}}- **API server:** {{.PortAPI}}{{end}}

## Testing

```bash
bun test                 # Run unit tests
bun run test:e2e         # Run Playwright tests
```

## Gotchas

[Document project-specific gotchas]

- [Gotcha 1]
- [Gotcha 2]
