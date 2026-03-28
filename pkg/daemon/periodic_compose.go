package daemon

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan-conlin/orch-go/pkg/compose"
	"github.com/dylan-conlin/orch-go/pkg/orient"
)

// ComposeResult holds the outcome of a periodic composition run.
type ComposeResult struct {
	Error         error
	Message       string
	Composed      bool
	BriefCount    int
	ClustersFound int
	DigestPath    string
}

// ComposeService abstracts brief counting, composition, and digest writing for testability.
type ComposeService interface {
	CountUndigestedBriefs() (int, error)
	Compose() (*compose.Digest, error)
	WriteDigest(d *compose.Digest) (string, error)
}

// RunPeriodicCompose runs brief composition if due and threshold is met.
// Returns nil if not due.
func (d *Daemon) RunPeriodicCompose() *ComposeResult {
	if !d.Scheduler.IsDue(TaskCompose) {
		return nil
	}

	svc := d.ComposeService
	if svc == nil {
		svc = &defaultComposeService{}
	}

	result := runComposition(svc, d.Config.ComposeThreshold)
	d.Scheduler.MarkRun(TaskCompose)
	return result
}

func runComposition(svc ComposeService, threshold int) *ComposeResult {
	count, err := svc.CountUndigestedBriefs()
	if err != nil {
		return &ComposeResult{
			Error:   err,
			Message: fmt.Sprintf("failed to count undigested briefs: %v", err),
		}
	}

	if count < threshold {
		return &ComposeResult{
			BriefCount: count,
			Message:    fmt.Sprintf("below threshold: %d briefs (need %d)", count, threshold),
		}
	}

	digest, err := svc.Compose()
	if err != nil {
		return &ComposeResult{
			Error:      err,
			BriefCount: count,
			Message:    fmt.Sprintf("composition failed: %v", err),
		}
	}

	path, err := svc.WriteDigest(digest)
	if err != nil {
		return &ComposeResult{
			Error:      err,
			BriefCount: count,
			Message:    fmt.Sprintf("failed to write digest: %v", err),
		}
	}

	return &ComposeResult{
		Composed:      true,
		BriefCount:    digest.BriefsComposed,
		ClustersFound: digest.ClustersFound,
		DigestPath:    path,
		Message:       fmt.Sprintf("composed %d briefs into %d clusters → %s", digest.BriefsComposed, digest.ClustersFound, path),
	}
}

// defaultComposeService is the production implementation using the filesystem.
type defaultComposeService struct{}

func (s *defaultComposeService) CountUndigestedBriefs() (int, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("getting working directory: %w", err)
	}

	briefsDir := filepath.Join(projectDir, ".kb", "briefs")
	digestsDir := filepath.Join(projectDir, ".kb", "digests")

	briefs, err := compose.LoadBriefs(briefsDir)
	if err != nil {
		return 0, err
	}

	digested := orient.DigestedBriefIDs(digestsDir)
	count := 0
	for _, b := range briefs {
		if !digested[b.ID] {
			count++
		}
	}
	return count, nil
}

func (s *defaultComposeService) Compose() (*compose.Digest, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	briefsDir := filepath.Join(projectDir, ".kb", "briefs")
	threadsDir := filepath.Join(projectDir, ".kb", "threads")
	return compose.Compose(briefsDir, threadsDir)
}

func (s *defaultComposeService) WriteDigest(d *compose.Digest) (string, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	digestsDir := filepath.Join(projectDir, ".kb", "digests")
	return compose.WriteDigest(d, digestsDir)
}
