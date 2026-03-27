package daemonconfig

// DecisionTier represents the autonomy level for a daemon decision.
// Classification is based on reversibility and blast radius.
type DecisionTier int

const (
	// TierAutonomous decisions are low-risk and reversible. The daemon acts and logs.
	TierAutonomous DecisionTier = iota
	// TierProposeAndAct decisions are medium-risk. The daemon acts immediately
	// but offers a veto window (1-24h depending on category).
	TierProposeAndAct
	// TierGenuineDecision decisions are high-risk or irreversible. The daemon
	// blocks and creates a beads issue with decision:pending label.
	TierGenuineDecision
)

// String returns the string representation of a DecisionTier.
func (t DecisionTier) String() string {
	switch t {
	case TierAutonomous:
		return "autonomous"
	case TierProposeAndAct:
		return "propose-and-act"
	case TierGenuineDecision:
		return "genuine-decision"
	default:
		return "autonomous"
	}
}

// DecisionClass identifies a specific type of daemon decision.
// 19 decision types across 6 categories, classified by reversibility and blast radius.
type DecisionClass int

const (
	// Spawn category — issue selection and routing
	DecisionSelectIssue      DecisionClass = iota // Pick next issue from ready queue
	DecisionInferSkill                            // Infer skill for an issue
	DecisionInferModel                            // Infer model for an issue
	DecisionRouteExtraction                       // Route to knowledge extraction
	DecisionArchitectEscalate                     // Escalate issue to architect review

	// Completion category — closing agent work
	DecisionAutoCompleteLight // Auto-complete light-tier agent
	DecisionAutoCompleteFull  // Auto-complete full-tier agent
	DecisionLabelForReview    // Label issue for human review

	// Knowledge category — creating knowledge artifacts
	DecisionCreateSynthesisIssue  // Create synthesis follow-up issue
	DecisionCreateModelDriftIssue // Create model drift detection issue
	DecisionCreateAgreementIssue  // Create agreement check issue

	// Lifecycle category — agent lifecycle management
	DecisionResetOrphan      // Reset orphaned agent
	DecisionResumeStuck      // Resume stuck agent
	DecisionFlagPhaseTimeout // Flag agent phase timeout

	// Compliance category — compliance level changes
	DecisionDowngradeCompliance // Downgrade compliance level

	// Work graph category — backlog and plan management
	DecisionDetectDuplicate // Detect duplicate issues
	DecisionSurfaceRemoval  // Surface removal candidates
	DecisionCullBacklog     // Cull stale backlog items
	DecisionAdvancePlan     // Advance coordination plan phase
)

// String returns the string representation of a DecisionClass.
func (c DecisionClass) String() string {
	switch c {
	case DecisionSelectIssue:
		return "select_issue"
	case DecisionInferSkill:
		return "infer_skill"
	case DecisionInferModel:
		return "infer_model"
	case DecisionRouteExtraction:
		return "route_extraction"
	case DecisionArchitectEscalate:
		return "architect_escalate"
	case DecisionAutoCompleteLight:
		return "auto_complete_light"
	case DecisionAutoCompleteFull:
		return "auto_complete_full"
	case DecisionLabelForReview:
		return "label_for_review"
	case DecisionCreateSynthesisIssue:
		return "create_synthesis_issue"
	case DecisionCreateModelDriftIssue:
		return "create_model_drift_issue"
	case DecisionCreateAgreementIssue:
		return "create_agreement_issue"
	case DecisionResetOrphan:
		return "reset_orphan"
	case DecisionResumeStuck:
		return "resume_stuck"
	case DecisionFlagPhaseTimeout:
		return "flag_phase_timeout"
	case DecisionDowngradeCompliance:
		return "downgrade_compliance"
	case DecisionDetectDuplicate:
		return "detect_duplicate"
	case DecisionSurfaceRemoval:
		return "surface_removal"
	case DecisionCullBacklog:
		return "cull_backlog"
	case DecisionAdvancePlan:
		return "advance_plan"
	default:
		return "unknown"
	}
}

// Category returns the category name for a DecisionClass.
func (c DecisionClass) Category() string {
	switch {
	case c <= DecisionArchitectEscalate:
		return "spawn"
	case c <= DecisionLabelForReview:
		return "completion"
	case c <= DecisionCreateAgreementIssue:
		return "knowledge"
	case c <= DecisionFlagPhaseTimeout:
		return "lifecycle"
	case c == DecisionDowngradeCompliance:
		return "compliance"
	default:
		return "work_graph"
	}
}

// baseTier returns the default tier for a decision class at ComplianceStandard.
// Classification is based on reversibility (can we undo it?) and blast radius
// (how much does it affect?).
func baseTier(class DecisionClass) DecisionTier {
	switch class {
	// Tier 1: Low risk, easily reversible, narrow blast radius
	case DecisionSelectIssue, // Just picks from queue — next cycle picks differently
		DecisionInferSkill,       // Inference can be overridden by label
		DecisionInferModel,       // Inference can be overridden by label
		DecisionResetOrphan,      // Orphan is already stuck — reset is recovery
		DecisionFlagPhaseTimeout, // Flagging is advisory
		DecisionDetectDuplicate:  // Detection is advisory, no action taken
		return TierAutonomous

	// Tier 2: Medium risk, reversible but wider blast radius
	case DecisionRouteExtraction,      // Creates issue — can be closed
		DecisionArchitectEscalate,     // Creates issue — can be closed
		DecisionAutoCompleteLight,     // Closes issue — can be reopened
		DecisionAutoCompleteFull,      // Closes issue — can be reopened
		DecisionLabelForReview,        // Adds label — can be removed
		DecisionCreateSynthesisIssue,  // Creates issue — can be closed
		DecisionCreateModelDriftIssue, // Creates issue — can be closed
		DecisionCreateAgreementIssue,  // Creates issue — can be closed
		DecisionResumeStuck,           // Sends message to agent — message is permanent
		DecisionSurfaceRemoval,        // Creates issue — can be closed
		DecisionCullBacklog,           // Closes issues — can be reopened
		DecisionAdvancePlan:           // Advances plan — can be reverted
		return TierProposeAndAct

	// Tier 3: High risk, hard to reverse, system-wide effect
	case DecisionDowngradeCompliance: // Changes compliance level — affects all future decisions
		return TierGenuineDecision

	default:
		return TierProposeAndAct // safe default for unknown classes
	}
}

// adjustForCompliance modulates a base tier by the compliance level.
//   - strict: promotes one level (more cautious)
//   - standard: no change (uses base tiers)
//   - relaxed: demotes one level (more autonomous)
//   - autonomous: everything T1 except base-T3 → T2 (safety floor)
func adjustForCompliance(base DecisionTier, level ComplianceLevel) DecisionTier {
	switch level {
	case ComplianceStrict:
		if base < TierGenuineDecision {
			return base + 1
		}
		return TierGenuineDecision

	case ComplianceStandard:
		return base

	case ComplianceRelaxed:
		if base > TierAutonomous {
			return base - 1
		}
		return TierAutonomous

	case ComplianceAutonomous:
		if base == TierGenuineDecision {
			return TierProposeAndAct // safety floor: T3 never goes to T1
		}
		return TierAutonomous

	default:
		return base
	}
}

// ClassifyDecision returns the effective decision tier for a given decision class
// and compliance level. This follows the DeriveX pattern established by the other
// compliance functions (DeriveReviewThreshold, etc.).
//
// The tier is computed as: baseTier(class) adjusted by compliance level.
func ClassifyDecision(class DecisionClass, level ComplianceLevel) DecisionTier {
	return adjustForCompliance(baseTier(class), level)
}
