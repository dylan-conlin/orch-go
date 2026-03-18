package daemon

import "github.com/dylan-conlin/orch-go/pkg/digest"

// Re-export gate types from pkg/digest for backward compatibility.
type DigestTypeStats = digest.TypeStats
type AdaptiveThreshold = digest.AdaptiveThreshold
type DigestFeedbackState = digest.FeedbackState

// Re-export gate constants.
const (
	MinProductsForAdaptation = digest.MinProductsForAdaptation
	MaturityWindowDays       = digest.MaturityWindowDays
	LowReadRateThreshold     = digest.LowReadRateThreshold
	HighStarRateThreshold    = digest.HighStarRateThreshold
)

// Re-export gate functions.
var (
	NewDigestFeedbackState   = digest.NewFeedbackState
	DigestFeedbackStatePath  = digest.FeedbackStatePath
	SaveDigestFeedbackState  = digest.SaveFeedbackState
	LoadDigestFeedbackState  = digest.LoadFeedbackState
)
