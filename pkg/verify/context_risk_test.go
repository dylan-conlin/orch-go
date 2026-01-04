package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAssessContextRisk(t *testing.T) {
	tests := []struct {
		name            string
		totalTokens     int
		hasUncommitted  bool
		uncommittedCount int
		isProcessing    bool
		expectedLevel   ContextRiskLevel
		expectRisk      bool
	}{
		{
			name:          "low tokens, no uncommitted work",
			totalTokens:   50000,
			hasUncommitted: false,
			isProcessing:  true,
			expectedLevel: RiskNone,
			expectRisk:    false,
		},
		{
			name:          "high tokens (warning level), no uncommitted work",
			totalTokens:   160000,
			hasUncommitted: false,
			isProcessing:  true,
			expectedLevel: RiskNone, // No risk without uncommitted work at warning level
			expectRisk:    false,
		},
		{
			name:           "high tokens (warning level), with uncommitted work",
			totalTokens:    160000,
			hasUncommitted: true,
			uncommittedCount: 3,
			isProcessing:   true,
			expectedLevel:  RiskWarning,
			expectRisk:     true,
		},
		{
			name:           "critical tokens, no uncommitted work",
			totalTokens:    185000,
			hasUncommitted: false,
			isProcessing:   true,
			expectedLevel:  RiskCritical,
			expectRisk:     true,
		},
		{
			name:           "critical tokens, with uncommitted work",
			totalTokens:    185000,
			hasUncommitted: true,
			uncommittedCount: 5,
			isProcessing:   true,
			expectedLevel:  RiskCritical,
			expectRisk:     true,
		},
		{
			name:           "low tokens, significant uncommitted work, idle agent",
			totalTokens:    50000,
			hasUncommitted: true,
			uncommittedCount: 10,
			isProcessing:   false,
			expectedLevel:  RiskWarning,
			expectRisk:     true,
		},
		{
			name:           "low tokens, few uncommitted files, idle agent",
			totalTokens:    50000,
			hasUncommitted: true,
			uncommittedCount: 2,
			isProcessing:   false,
			expectedLevel:  RiskNone, // Not enough uncommitted files to warn
			expectRisk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			tmpDir := t.TempDir()
			
			// Initialize a git repo if we need uncommitted work checks
			if tt.hasUncommitted {
				// Initialize git repo
				gitInit := exec.Command("git", "init")
				gitInit.Dir = tmpDir
				if err := gitInit.Run(); err != nil {
					t.Skipf("git init failed: %v", err)
				}
				
				// Configure git for the test
				gitConfig1 := exec.Command("git", "config", "user.email", "test@test.com")
				gitConfig1.Dir = tmpDir
				gitConfig1.Run()
				
				gitConfig2 := exec.Command("git", "config", "user.name", "Test")
				gitConfig2.Dir = tmpDir
				gitConfig2.Run()
				
				// Create and commit an initial file
				initialFile := filepath.Join(tmpDir, "initial.txt")
				os.WriteFile(initialFile, []byte("initial"), 0644)
				
				gitAdd := exec.Command("git", "add", ".")
				gitAdd.Dir = tmpDir
				gitAdd.Run()
				
				gitCommit := exec.Command("git", "commit", "-m", "initial")
				gitCommit.Dir = tmpDir
				gitCommit.Run()
				
				// Create uncommitted files
				for i := 0; i < tt.uncommittedCount; i++ {
					file := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
					os.WriteFile(file, []byte("uncommitted"), 0644)
				}
			} else {
				// No git repo - just use empty dir
				tmpDir = ""
			}

			// Assess the risk
			risk := AssessContextRisk(tt.totalTokens, tmpDir, tt.isProcessing)

			// Check risk level
			if risk.Level != tt.expectedLevel {
				t.Errorf("expected level %q, got %q", tt.expectedLevel, risk.Level)
			}

			// Check if at risk matches expectation
			if risk.IsAtRisk() != tt.expectRisk {
				t.Errorf("expected IsAtRisk()=%v, got %v", tt.expectRisk, risk.IsAtRisk())
			}

			// Check token stats are populated
			if risk.TokenUsage != tt.totalTokens {
				t.Errorf("expected TokenUsage=%d, got %d", tt.totalTokens, risk.TokenUsage)
			}
		})
	}
}

func TestHasUncommittedWork(t *testing.T) {
	// Test with empty directory
	t.Run("empty directory", func(t *testing.T) {
		has, count := HasUncommittedWork("")
		if has || count != 0 {
			t.Errorf("expected no uncommitted work for empty dir, got has=%v count=%d", has, count)
		}
	})

	// Test with non-git directory
	t.Run("non-git directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		has, count := HasUncommittedWork(tmpDir)
		if has || count != 0 {
			t.Errorf("expected no uncommitted work for non-git dir, got has=%v count=%d", has, count)
		}
	})

	// Test with clean git repo
	t.Run("clean git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Initialize git repo
		gitInit := exec.Command("git", "init")
		gitInit.Dir = tmpDir
		if err := gitInit.Run(); err != nil {
			t.Skipf("git init failed: %v", err)
		}
		
		// Configure git
		exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
		exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
		
		// Create and commit a file
		file := filepath.Join(tmpDir, "file.txt")
		os.WriteFile(file, []byte("content"), 0644)
		exec.Command("git", "-C", tmpDir, "add", ".").Run()
		exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

		has, count := HasUncommittedWork(tmpDir)
		if has || count != 0 {
			t.Errorf("expected no uncommitted work for clean repo, got has=%v count=%d", has, count)
		}
	})

	// Test with uncommitted changes
	t.Run("uncommitted changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Initialize git repo
		gitInit := exec.Command("git", "init")
		gitInit.Dir = tmpDir
		if err := gitInit.Run(); err != nil {
			t.Skipf("git init failed: %v", err)
		}
		
		// Configure git
		exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
		exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()
		
		// Create and commit an initial file
		file := filepath.Join(tmpDir, "file.txt")
		os.WriteFile(file, []byte("content"), 0644)
		exec.Command("git", "-C", tmpDir, "add", ".").Run()
		exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()
		
		// Create uncommitted files
		os.WriteFile(filepath.Join(tmpDir, "new1.txt"), []byte("new"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "new2.txt"), []byte("new"), 0644)

		has, count := HasUncommittedWork(tmpDir)
		if !has {
			t.Error("expected uncommitted work to be detected")
		}
		if count != 2 {
			t.Errorf("expected 2 uncommitted files, got %d", count)
		}
	})
}

func TestContextExhaustionRisk_FormatMethods(t *testing.T) {
	tests := []struct {
		name             string
		risk             ContextExhaustionRisk
		expectedStatus   string
		expectedEmoji    string
		expectedAtRisk   bool
		expectedAlert    bool
	}{
		{
			name:           "no risk",
			risk:           ContextExhaustionRisk{Level: RiskNone},
			expectedStatus: "",
			expectedEmoji:  "",
			expectedAtRisk: false,
			expectedAlert:  false,
		},
		{
			name:           "warning with uncommitted work",
			risk:           ContextExhaustionRisk{Level: RiskWarning, HasUncommittedWork: true},
			expectedStatus: "AT-RISK",
			expectedEmoji:  "⚠️",
			expectedAtRisk: true,
			expectedAlert:  true,
		},
		{
			name:           "warning without uncommitted work",
			risk:           ContextExhaustionRisk{Level: RiskWarning, HasUncommittedWork: false},
			expectedStatus: "HIGH-TOK",
			expectedEmoji:  "⚠️",
			expectedAtRisk: true,
			expectedAlert:  false,
		},
		{
			name:           "critical with uncommitted work",
			risk:           ContextExhaustionRisk{Level: RiskCritical, HasUncommittedWork: true},
			expectedStatus: "CRITICAL",
			expectedEmoji:  "🚨",
			expectedAtRisk: true,
			expectedAlert:  true,
		},
		{
			name:           "critical without uncommitted work",
			risk:           ContextExhaustionRisk{Level: RiskCritical, HasUncommittedWork: false},
			expectedStatus: "HIGH-CTX",
			expectedEmoji:  "🚨",
			expectedAtRisk: true,
			expectedAlert:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.risk.FormatRiskStatus(); got != tt.expectedStatus {
				t.Errorf("FormatRiskStatus() = %q, want %q", got, tt.expectedStatus)
			}
			if got := tt.risk.FormatRiskEmoji(); got != tt.expectedEmoji {
				t.Errorf("FormatRiskEmoji() = %q, want %q", got, tt.expectedEmoji)
			}
			if got := tt.risk.IsAtRisk(); got != tt.expectedAtRisk {
				t.Errorf("IsAtRisk() = %v, want %v", got, tt.expectedAtRisk)
			}
			if got := tt.risk.ShouldAlert(); got != tt.expectedAlert {
				t.Errorf("ShouldAlert() = %v, want %v", got, tt.expectedAlert)
			}
		})
	}
}
