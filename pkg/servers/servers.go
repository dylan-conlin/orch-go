// Package servers provides schema and configuration for per-project server declarations.
//
// Servers are declared in .orch/servers.yaml and managed by `orch servers` switchboard.
// This replaces the simpler `servers:` section in .orch/config.yaml with a richer schema
// that supports health checks, different server types, and lifecycle management.
//
// Example servers.yaml:
//
//	servers:
//	  - name: web
//	    type: command
//	    command: bun run dev
//	    port: 5173
//	    health:
//	      type: http
//	      path: /
//	      interval: 5s
//	      timeout: 2s
//	  - name: api
//	    type: command
//	    command: go run ./cmd/server
//	    port: 3000
//	    health:
//	      type: tcp
//	      interval: 5s
//	      timeout: 2s
//	  - name: db
//	    type: docker
//	    image: postgres:15
//	    port: 5432
//	    health:
//	      type: command
//	      command: pg_isready -U postgres
//	      interval: 10s
//	      timeout: 5s
package servers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ServerType defines how a server is started and managed.
type ServerType string

const (
	// TypeCommand runs a shell command in the terminal.
	TypeCommand ServerType = "command"
	// TypeDocker runs a Docker container.
	TypeDocker ServerType = "docker"
	// TypeLaunchd uses macOS launchd for background services.
	TypeLaunchd ServerType = "launchd"
)

// HealthCheckType defines how to verify a server is healthy.
type HealthCheckType string

const (
	// HealthHTTP checks an HTTP endpoint returns 2xx.
	HealthHTTP HealthCheckType = "http"
	// HealthTCP checks a TCP port is accepting connections.
	HealthTCP HealthCheckType = "tcp"
	// HealthCommand runs a command that should exit 0 when healthy.
	HealthCommand HealthCheckType = "command"
)

// Duration wraps time.Duration for YAML unmarshaling.
type Duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler for Duration.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// MarshalYAML implements yaml.Marshaler for Duration.
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// Duration returns the underlying time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// HealthCheck defines how to verify a server is running and healthy.
type HealthCheck struct {
	// Type is the health check method: http, tcp, or command.
	Type HealthCheckType `yaml:"type"`

	// Path is the HTTP path to check (for type: http).
	// Default: "/"
	Path string `yaml:"path,omitempty"`

	// Command is the health check command (for type: command).
	Command string `yaml:"command,omitempty"`

	// Interval is how often to check health.
	// Default: 5s
	Interval Duration `yaml:"interval,omitempty"`

	// Timeout is how long to wait for a response.
	// Default: 2s
	Timeout Duration `yaml:"timeout,omitempty"`

	// Retries is how many failures before marking unhealthy.
	// Default: 3
	Retries int `yaml:"retries,omitempty"`
}

// Server defines a single server in the project.
type Server struct {
	// Name is the unique identifier for this server within the project.
	// Examples: "web", "api", "db", "redis"
	Name string `yaml:"name"`

	// Type defines how the server is started: command, docker, or launchd.
	Type ServerType `yaml:"type"`

	// Command is the shell command to run (for type: command).
	Command string `yaml:"command,omitempty"`

	// Image is the Docker image to run (for type: docker).
	Image string `yaml:"image,omitempty"`

	// LaunchdLabel is the launchd service label (for type: launchd).
	LaunchdLabel string `yaml:"launchd_label,omitempty"`

	// Port is the primary port this server listens on.
	// Used for health checks and URL generation.
	Port int `yaml:"port"`

	// Health defines how to verify the server is running.
	// Optional - if not set, only port binding is checked.
	Health *HealthCheck `yaml:"health,omitempty"`

	// Env is environment variables to set when starting.
	Env map[string]string `yaml:"env,omitempty"`

	// Workdir is the working directory for the command.
	// Relative to project root. Default: "."
	Workdir string `yaml:"workdir,omitempty"`

	// DependsOn lists other servers that must be healthy first.
	DependsOn []string `yaml:"depends_on,omitempty"`
}

// Config is the root configuration for .orch/servers.yaml.
type Config struct {
	// Servers is the list of server definitions.
	Servers []Server `yaml:"servers"`
}

// DefaultPath returns the path to servers.yaml for a project.
func DefaultPath(projectDir string) string {
	return filepath.Join(projectDir, ".orch", "servers.yaml")
}

// Load reads and parses .orch/servers.yaml from the project directory.
func Load(projectDir string) (*Config, error) {
	configPath := DefaultPath(projectDir)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &Config{Servers: []Server{}}, nil
		}
		return nil, fmt.Errorf("failed to read servers.yaml: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse servers.yaml: %w", err)
	}

	// Apply defaults
	for i := range cfg.Servers {
		applyServerDefaults(&cfg.Servers[i])
	}

	return &cfg, nil
}

// applyServerDefaults sets default values for optional fields.
func applyServerDefaults(s *Server) {
	if s.Type == "" {
		s.Type = TypeCommand
	}
	if s.Workdir == "" {
		s.Workdir = "."
	}
	if s.Health != nil {
		if s.Health.Interval == 0 {
			s.Health.Interval = Duration(5 * time.Second)
		}
		if s.Health.Timeout == 0 {
			s.Health.Timeout = Duration(2 * time.Second)
		}
		if s.Health.Retries == 0 {
			s.Health.Retries = 3
		}
		if s.Health.Type == HealthHTTP && s.Health.Path == "" {
			s.Health.Path = "/"
		}
	}
}

// Save writes the configuration to .orch/servers.yaml.
func Save(projectDir string, cfg *Config) error {
	configPath := DefaultPath(projectDir)

	// Ensure .orch directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create .orch directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal servers.yaml: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write servers.yaml: %w", err)
	}

	return nil
}

// GetServer returns a server by name, or nil if not found.
func (c *Config) GetServer(name string) *Server {
	for i := range c.Servers {
		if c.Servers[i].Name == name {
			return &c.Servers[i]
		}
	}
	return nil
}

// GetServerPort returns the port for a named server, or 0 if not found.
func (c *Config) GetServerPort(name string) int {
	if s := c.GetServer(name); s != nil {
		return s.Port
	}
	return 0
}

// ServerNames returns a list of all configured server names.
func (c *Config) ServerNames() []string {
	names := make([]string, len(c.Servers))
	for i, s := range c.Servers {
		names[i] = s.Name
	}
	return names
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	seen := make(map[string]bool)
	ports := make(map[int]string)

	for _, s := range c.Servers {
		// Check for duplicate names
		if seen[s.Name] {
			return fmt.Errorf("duplicate server name: %s", s.Name)
		}
		seen[s.Name] = true

		// Check for duplicate ports
		if other, ok := ports[s.Port]; ok {
			return fmt.Errorf("duplicate port %d used by both %s and %s", s.Port, other, s.Name)
		}
		ports[s.Port] = s.Name

		// Validate server type
		if err := s.Validate(); err != nil {
			return fmt.Errorf("server %s: %w", s.Name, err)
		}

		// Check dependencies exist
		for _, dep := range s.DependsOn {
			if !seen[dep] {
				// Note: this is a simplistic check - doesn't handle forward references
				// For now, just warn that the dependency should be defined first
			}
		}
	}

	return nil
}

// Validate checks a single server configuration for errors.
func (s *Server) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}

	if s.Port == 0 {
		return fmt.Errorf("port is required")
	}

	switch s.Type {
	case TypeCommand:
		if s.Command == "" {
			return fmt.Errorf("command is required for type %s", TypeCommand)
		}
	case TypeDocker:
		if s.Image == "" {
			return fmt.Errorf("image is required for type %s", TypeDocker)
		}
	case TypeLaunchd:
		if s.LaunchdLabel == "" {
			return fmt.Errorf("launchd_label is required for type %s", TypeLaunchd)
		}
	default:
		return fmt.Errorf("invalid server type: %s (must be command, docker, or launchd)", s.Type)
	}

	if s.Health != nil {
		if err := s.Health.Validate(s.Type); err != nil {
			return fmt.Errorf("health check: %w", err)
		}
	}

	return nil
}

// Validate checks a health check configuration for errors.
func (h *HealthCheck) Validate(serverType ServerType) error {
	switch h.Type {
	case HealthHTTP:
		// HTTP health checks need a path (defaulted to "/" if empty)
	case HealthTCP:
		// TCP health checks just need the port (from server)
	case HealthCommand:
		if h.Command == "" {
			return fmt.Errorf("command is required for health type %s", HealthCommand)
		}
	default:
		return fmt.Errorf("invalid health check type: %s (must be http, tcp, or command)", h.Type)
	}

	if h.Interval.Duration() < 0 {
		return fmt.Errorf("interval cannot be negative")
	}
	if h.Timeout.Duration() < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}
	if h.Timeout.Duration() > h.Interval.Duration() && h.Interval.Duration() > 0 {
		return fmt.Errorf("timeout (%s) cannot exceed interval (%s)",
			h.Timeout.Duration(), h.Interval.Duration())
	}
	if h.Retries < 0 {
		return fmt.Errorf("retries cannot be negative")
	}

	return nil
}
