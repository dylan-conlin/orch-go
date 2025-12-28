package servers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_FullConfig(t *testing.T) {
	tmpDir := t.TempDir()
	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	configContent := `servers:
  - name: web
    type: command
    command: bun run dev
    port: 5173
    workdir: web
    health:
      type: http
      path: /health
      interval: 10s
      timeout: 3s
      retries: 5
    env:
      NODE_ENV: development
  - name: api
    type: command
    command: go run ./cmd/server
    port: 3000
    health:
      type: tcp
      interval: 5s
  - name: db
    type: docker
    image: postgres:15
    port: 5432
    health:
      type: command
      command: pg_isready -U postgres
    depends_on:
      - redis
  - name: redis
    type: docker
    image: redis:7
    port: 6379
`
	if err := os.WriteFile(filepath.Join(orchDir, "servers.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Servers) != 4 {
		t.Errorf("Expected 4 servers, got %d", len(cfg.Servers))
	}

	// Check web server
	web := cfg.GetServer("web")
	if web == nil {
		t.Fatal("web server not found")
	}
	if web.Type != TypeCommand {
		t.Errorf("web.Type = %q, want %q", web.Type, TypeCommand)
	}
	if web.Command != "bun run dev" {
		t.Errorf("web.Command = %q, want %q", web.Command, "bun run dev")
	}
	if web.Port != 5173 {
		t.Errorf("web.Port = %d, want 5173", web.Port)
	}
	if web.Workdir != "web" {
		t.Errorf("web.Workdir = %q, want %q", web.Workdir, "web")
	}
	if web.Health == nil {
		t.Fatal("web.Health is nil")
	}
	if web.Health.Type != HealthHTTP {
		t.Errorf("web.Health.Type = %q, want %q", web.Health.Type, HealthHTTP)
	}
	if web.Health.Path != "/health" {
		t.Errorf("web.Health.Path = %q, want %q", web.Health.Path, "/health")
	}
	if web.Health.Interval.Duration() != 10*time.Second {
		t.Errorf("web.Health.Interval = %s, want 10s", web.Health.Interval.Duration())
	}
	if web.Health.Timeout.Duration() != 3*time.Second {
		t.Errorf("web.Health.Timeout = %s, want 3s", web.Health.Timeout.Duration())
	}
	if web.Health.Retries != 5 {
		t.Errorf("web.Health.Retries = %d, want 5", web.Health.Retries)
	}
	if web.Env["NODE_ENV"] != "development" {
		t.Errorf("web.Env[NODE_ENV] = %q, want %q", web.Env["NODE_ENV"], "development")
	}

	// Check Docker server
	db := cfg.GetServer("db")
	if db == nil {
		t.Fatal("db server not found")
	}
	if db.Type != TypeDocker {
		t.Errorf("db.Type = %q, want %q", db.Type, TypeDocker)
	}
	if db.Image != "postgres:15" {
		t.Errorf("db.Image = %q, want %q", db.Image, "postgres:15")
	}
	if len(db.DependsOn) != 1 || db.DependsOn[0] != "redis" {
		t.Errorf("db.DependsOn = %v, want [redis]", db.DependsOn)
	}

	// Check api server gets defaults
	api := cfg.GetServer("api")
	if api == nil {
		t.Fatal("api server not found")
	}
	if api.Health.Interval.Duration() != 5*time.Second {
		t.Errorf("api.Health.Interval = %s, want 5s", api.Health.Interval.Duration())
	}
	// Check default timeout is applied
	if api.Health.Timeout.Duration() != 2*time.Second {
		t.Errorf("api.Health.Timeout = %s, want 2s (default)", api.Health.Timeout.Duration())
	}
}

func TestLoad_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Minimal config - only required fields
	configContent := `servers:
  - name: web
    command: npm start
    port: 3000
    health:
      type: http
`
	if err := os.WriteFile(filepath.Join(orchDir, "servers.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	web := cfg.GetServer("web")
	if web == nil {
		t.Fatal("web server not found")
	}

	// Check defaults are applied
	if web.Type != TypeCommand {
		t.Errorf("Type default: got %q, want %q", web.Type, TypeCommand)
	}
	if web.Workdir != "." {
		t.Errorf("Workdir default: got %q, want %q", web.Workdir, ".")
	}
	if web.Health.Path != "/" {
		t.Errorf("Health.Path default: got %q, want %q", web.Health.Path, "/")
	}
	if web.Health.Interval.Duration() != 5*time.Second {
		t.Errorf("Health.Interval default: got %s, want 5s", web.Health.Interval.Duration())
	}
	if web.Health.Timeout.Duration() != 2*time.Second {
		t.Errorf("Health.Timeout default: got %s, want 2s", web.Health.Timeout.Duration())
	}
	if web.Health.Retries != 3 {
		t.Errorf("Health.Retries default: got %d, want 3", web.Health.Retries)
	}
}

func TestLoad_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Servers) != 0 {
		t.Errorf("Expected empty config, got %d servers", len(cfg.Servers))
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Servers: []Server{
			{
				Name:    "web",
				Type:    TypeCommand,
				Command: "npm start",
				Port:    3000,
				Health: &HealthCheck{
					Type:     HealthHTTP,
					Path:     "/health",
					Interval: Duration(10 * time.Second),
					Timeout:  Duration(3 * time.Second),
				},
			},
		},
	}

	if err := Save(tmpDir, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	configPath := DefaultPath(tmpDir)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", configPath)
	}

	// Reload and verify
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load after Save failed: %v", err)
	}

	if len(loaded.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(loaded.Servers))
	}
	if loaded.GetServerPort("web") != 3000 {
		t.Errorf("web port = %d, want 3000", loaded.GetServerPort("web"))
	}
}

func TestGetServer(t *testing.T) {
	cfg := &Config{
		Servers: []Server{
			{Name: "web", Port: 3000},
			{Name: "api", Port: 8080},
		},
	}

	if s := cfg.GetServer("web"); s == nil || s.Port != 3000 {
		t.Error("GetServer(web) failed")
	}
	if s := cfg.GetServer("api"); s == nil || s.Port != 8080 {
		t.Error("GetServer(api) failed")
	}
	if s := cfg.GetServer("notexist"); s != nil {
		t.Error("GetServer(notexist) should return nil")
	}
}

func TestGetServerPort(t *testing.T) {
	cfg := &Config{
		Servers: []Server{
			{Name: "web", Port: 3000},
		},
	}

	if port := cfg.GetServerPort("web"); port != 3000 {
		t.Errorf("GetServerPort(web) = %d, want 3000", port)
	}
	if port := cfg.GetServerPort("notexist"); port != 0 {
		t.Errorf("GetServerPort(notexist) = %d, want 0", port)
	}
}

func TestServerNames(t *testing.T) {
	cfg := &Config{
		Servers: []Server{
			{Name: "web"},
			{Name: "api"},
			{Name: "db"},
		},
	}

	names := cfg.ServerNames()
	if len(names) != 3 {
		t.Errorf("ServerNames() returned %d names, want 3", len(names))
	}

	expected := map[string]bool{"web": true, "api": true, "db": true}
	for _, name := range names {
		if !expected[name] {
			t.Errorf("Unexpected server name: %s", name)
		}
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Servers: []Server{
			{
				Name:    "web",
				Type:    TypeCommand,
				Command: "npm start",
				Port:    3000,
			},
			{
				Name:  "db",
				Type:  TypeDocker,
				Image: "postgres:15",
				Port:  5432,
			},
			{
				Name:         "launchd-service",
				Type:         TypeLaunchd,
				LaunchdLabel: "com.example.service",
				Port:         8080,
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate failed on valid config: %v", err)
	}
}

func TestValidate_DuplicateName(t *testing.T) {
	cfg := &Config{
		Servers: []Server{
			{Name: "web", Type: TypeCommand, Command: "npm start", Port: 3000},
			{Name: "web", Type: TypeCommand, Command: "npm run dev", Port: 3001},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate should fail on duplicate names")
	}
	if err.Error() != "duplicate server name: web" {
		t.Errorf("Wrong error: %v", err)
	}
}

func TestValidate_DuplicatePort(t *testing.T) {
	cfg := &Config{
		Servers: []Server{
			{Name: "web", Type: TypeCommand, Command: "npm start", Port: 3000},
			{Name: "api", Type: TypeCommand, Command: "go run main.go", Port: 3000},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate should fail on duplicate ports")
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		server  Server
		wantErr string
	}{
		{
			name:    "missing name",
			server:  Server{Type: TypeCommand, Command: "npm start", Port: 3000},
			wantErr: "name is required",
		},
		{
			name:    "missing port",
			server:  Server{Name: "web", Type: TypeCommand, Command: "npm start"},
			wantErr: "port is required",
		},
		{
			name:    "command type missing command",
			server:  Server{Name: "web", Type: TypeCommand, Port: 3000},
			wantErr: "command is required",
		},
		{
			name:    "docker type missing image",
			server:  Server{Name: "db", Type: TypeDocker, Port: 5432},
			wantErr: "image is required",
		},
		{
			name:    "launchd type missing label",
			server:  Server{Name: "svc", Type: TypeLaunchd, Port: 8080},
			wantErr: "launchd_label is required",
		},
		{
			name:    "invalid type",
			server:  Server{Name: "web", Type: "invalid", Port: 3000},
			wantErr: "invalid server type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.server.Validate()
			if err == nil {
				t.Errorf("Validate should fail: %s", tt.wantErr)
			}
		})
	}
}

func TestHealthCheck_Validate(t *testing.T) {
	tests := []struct {
		name       string
		health     HealthCheck
		serverType ServerType
		wantErr    bool
	}{
		{
			name:       "valid http",
			health:     HealthCheck{Type: HealthHTTP, Path: "/health", Interval: Duration(5 * time.Second), Timeout: Duration(2 * time.Second)},
			serverType: TypeCommand,
			wantErr:    false,
		},
		{
			name:       "valid tcp",
			health:     HealthCheck{Type: HealthTCP, Interval: Duration(5 * time.Second), Timeout: Duration(2 * time.Second)},
			serverType: TypeCommand,
			wantErr:    false,
		},
		{
			name:       "valid command",
			health:     HealthCheck{Type: HealthCommand, Command: "curl localhost", Interval: Duration(5 * time.Second), Timeout: Duration(2 * time.Second)},
			serverType: TypeDocker,
			wantErr:    false,
		},
		{
			name:       "command health missing command",
			health:     HealthCheck{Type: HealthCommand, Interval: Duration(5 * time.Second)},
			serverType: TypeCommand,
			wantErr:    true,
		},
		{
			name:       "invalid health type",
			health:     HealthCheck{Type: "invalid"},
			serverType: TypeCommand,
			wantErr:    true,
		},
		{
			name:       "timeout exceeds interval",
			health:     HealthCheck{Type: HealthHTTP, Interval: Duration(2 * time.Second), Timeout: Duration(5 * time.Second)},
			serverType: TypeCommand,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.health.Validate(tt.serverType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDuration_YAML(t *testing.T) {
	tmpDir := t.TempDir()
	orchDir := filepath.Join(tmpDir, ".orch")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test various duration formats
	configContent := `servers:
  - name: web
    command: npm start
    port: 3000
    health:
      type: http
      interval: 30s
      timeout: 500ms
`
	if err := os.WriteFile(filepath.Join(orchDir, "servers.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	web := cfg.GetServer("web")
	if web.Health.Interval.Duration() != 30*time.Second {
		t.Errorf("Interval = %s, want 30s", web.Health.Interval.Duration())
	}
	if web.Health.Timeout.Duration() != 500*time.Millisecond {
		t.Errorf("Timeout = %s, want 500ms", web.Health.Timeout.Duration())
	}
}
