// Package daemon provides autonomous overnight processing capabilities.
// Digest functionality has been extracted to pkg/digest.
// This file re-exports types for backward compatibility and provides
// the daemon-integrated RunPeriodicDigest method.
package daemon

import (
	"fmt"

	"github.com/dylan-conlin/orch-go/pkg/digest"
)

// Re-export types from pkg/digest for backward compatibility.
type DigestProductType = digest.ProductType
type DigestProductState = digest.ProductState
type DigestProduct = digest.Product
type DigestSource = digest.Source
type DigestState = digest.State
type DigestStats = digest.Stats
type DigestStatsResponse = digest.StatsResponse
type DigestArtifactChange = digest.ArtifactChange
type DigestService = digest.Service
type DigestResult = digest.Result
type DigestListOpts = digest.ListOpts
type DigestStore = digest.Store

// Re-export constants.
const (
	DigestTypeThreadProgression = digest.TypeThreadProgression
	DigestTypeModelUpdate       = digest.TypeModelUpdate
	DigestTypeModelProbe        = digest.TypeModelProbe
	DigestTypeDecisionBrief     = digest.TypeDecisionBrief

	DigestStateNew      = digest.StateNew
	DigestStateRead     = digest.StateRead
	DigestStateStarred  = digest.StateStarred
	DigestStateArchived = digest.StateArchived

	SignificanceLow    = digest.SignificanceLow
	SignificanceMedium = digest.SignificanceMedium
	SignificanceHigh   = digest.SignificanceHigh

	ThreadDeltaWordThreshold = digest.ThreadDeltaWordThreshold
)

// Re-export constructors and functions.
var (
	NewDigestStore          = digest.NewStore
	NewDefaultDigestService = digest.NewDefaultService
	LoadDigestState         = digest.LoadState
	SaveDigestState         = digest.SaveState
)

// RunPeriodicDigest scans KB artifacts for changes and creates digest products.
// This is the daemon-integrated entry point that checks scheduling.
func (d *Daemon) RunPeriodicDigest() *digest.Result {
	if !d.Scheduler.IsDue(TaskDigest) {
		return nil
	}

	svc := d.Digest
	if svc == nil {
		return &digest.Result{
			Error:   fmt.Errorf("digest service not configured"),
			Message: "Digest: service not configured",
		}
	}

	result := digest.RunDigest(svc, d.DigestDir, d.DigestStatePath)

	d.Scheduler.MarkRun(TaskDigest)
	return result
}
