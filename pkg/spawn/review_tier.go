package spawn

// Review tier constants.
// Each tier represents an increasing level of orchestrator review rigor.
// Declared at spawn time, determines how thoroughly the orchestrator reviews completion.
const (
	ReviewAuto   = "auto"   // Auto-close: minimal orchestrator involvement
	ReviewScan   = "scan"   // Quick scan: skim synthesis + verify gates passed
	ReviewReview = "review" // Full review: read synthesis, check diff, verify gates
	ReviewDeep   = "deep"   // Deep review: explain-back, behavioral verification
)

// reviewTierOrder maps tiers to their numeric order for comparison.
var reviewTierOrder = map[string]int{
	ReviewAuto:   0,
	ReviewScan:   1,
	ReviewReview: 2,
	ReviewDeep:   3,
}

// SkillReviewTierDefaults maps skills to their default review tier.
var SkillReviewTierDefaults = map[string]string{
	// auto: minimal review — knowledge capture, issue bookkeeping
	"capture-knowledge": ReviewAuto,
	"issue-creation":    ReviewAuto,

	// scan: quick scan — knowledge-producing skills
	"investigation":  ReviewScan,
	"probe":          ReviewScan,
	"research":       ReviewScan,
	"codebase-audit": ReviewScan,
	"design-session": ReviewScan,
	"ux-audit":       ReviewScan,

	// review: full review — implementation and architecture skills
	"feature-impl":         ReviewReview,
	"systematic-debugging": ReviewReview,
	"architect":            ReviewReview,
	"reliability-testing":  ReviewReview,

	// deep: deep review — visual/interactive verification required
	"debug-with-playwright": ReviewDeep,
}

// IssueTypeMinReviewTier maps issue types to their minimum review tier.
// The actual tier is max(skill_tier, issue_type_minimum).
var IssueTypeMinReviewTier = map[string]string{
	"feature":  ReviewReview,
	"bug":      ReviewReview,
	"decision": ReviewReview,
	// task, question, investigation, probe: no minimum
}

// DefaultReviewTier returns the default review tier for a skill and issue type.
// The tier is max(skill_default, issue_type_minimum).
// Returns ReviewReview for unknown skills (conservative default).
func DefaultReviewTier(skillName, issueType string) string {
	skillTier, ok := SkillReviewTierDefaults[skillName]
	if !ok {
		skillTier = ReviewReview // Conservative default for unknown skills
	}

	issueMin, ok := IssueTypeMinReviewTier[issueType]
	if !ok {
		return skillTier // No issue type minimum
	}

	return MaxReviewTier(skillTier, issueMin)
}

// CompareReviewTiers compares two review tiers.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
// Unknown tiers are treated as review (conservative).
func CompareReviewTiers(a, b string) int {
	orderA := tierToOrder(a)
	orderB := tierToOrder(b)
	if orderA < orderB {
		return -1
	}
	if orderA > orderB {
		return 1
	}
	return 0
}

// MaxReviewTier returns the higher of two review tiers.
func MaxReviewTier(a, b string) string {
	if CompareReviewTiers(a, b) >= 0 {
		return a
	}
	return b
}

// IsValidReviewTier returns true if the tier string is a valid review tier.
func IsValidReviewTier(tier string) bool {
	_, ok := reviewTierOrder[tier]
	return ok
}

// tierToOrder returns the numeric order for a review tier.
// Unknown tiers default to review order (conservative).
func tierToOrder(tier string) int {
	if order, ok := reviewTierOrder[tier]; ok {
		return order
	}
	return reviewTierOrder[ReviewReview] // Conservative default
}
