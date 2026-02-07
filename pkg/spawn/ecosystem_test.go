package spawn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectDomain(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name       string
		projectDir string
		want       string
	}{
		{
			name:       "personal in Documents/personal",
			projectDir: filepath.Join(home, "Documents", "personal", "orch-go"),
			want:       DomainPersonal,
		},
		{
			name:       "work in Documents/work",
			projectDir: filepath.Join(home, "Documents", "work", "SendCutSend", "scs-api"),
			want:       DomainWork,
		},
		{
			name:       "work in ~/work",
			projectDir: filepath.Join(home, "work", "project"),
			want:       DomainWork,
		},
		{
			name:       "unknown path defaults to personal",
			projectDir: filepath.Join(home, "other", "project"),
			want:       DomainPersonal,
		},
		{
			name:       "root path defaults to personal",
			projectDir: "/tmp/project",
			want:       DomainPersonal,
		},
		{
			name:       "empty path defaults to personal",
			projectDir: "",
			want:       DomainPersonal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectDomain(tt.projectDir)
			if got != tt.want {
				t.Errorf("DetectDomain(%q) = %q, want %q", tt.projectDir, got, tt.want)
			}
		})
	}
}

func TestGetEcosystemRepos(t *testing.T) {
	tests := []struct {
		name           string
		domain         string
		wantNil        bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:         "personal domain returns orch ecosystem",
			domain:       DomainPersonal,
			wantNil:      false,
			wantContains: []string{"orch-go", "kb-cli", "beads", "glass", "skillc"},
		},
		{
			name:         "work domain returns work ecosystem",
			domain:       DomainWork,
			wantNil:      false,
			wantContains: []string{"scs-special-projects"},
		},
		{
			name:    "unknown domain returns nil",
			domain:  "unknown",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEcosystemRepos(tt.domain)

			if tt.wantNil {
				if got != nil {
					t.Errorf("GetEcosystemRepos(%q) = %v, want nil", tt.domain, got)
				}
				return
			}

			if got == nil {
				t.Errorf("GetEcosystemRepos(%q) = nil, want non-nil", tt.domain)
				return
			}

			for _, repo := range tt.wantContains {
				if !got[repo] {
					t.Errorf("GetEcosystemRepos(%q) missing %q", tt.domain, repo)
				}
			}

			for _, repo := range tt.wantNotContain {
				if got[repo] {
					t.Errorf("GetEcosystemRepos(%q) should not contain %q", tt.domain, repo)
				}
			}
		})
	}
}

func TestIsEcosystemRepo(t *testing.T) {
	tests := []struct {
		name string
		repo string
		want bool
	}{
		{name: "orch-go is ecosystem", repo: "orch-go", want: true},
		{name: "kb-cli is ecosystem", repo: "kb-cli", want: true},
		{name: "beads is ecosystem", repo: "beads", want: true},
		{name: "glass is ecosystem", repo: "glass", want: true},
		{name: "skillc is ecosystem", repo: "skillc", want: true},
		{name: "unknown is not ecosystem", repo: "unknown-repo", want: false},
		{name: "scs-special-projects is not in expanded list", repo: "scs-special-projects", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEcosystemRepo(tt.repo)
			if got != tt.want {
				t.Errorf("IsEcosystemRepo(%q) = %v, want %v", tt.repo, got, tt.want)
			}
		})
	}
}
