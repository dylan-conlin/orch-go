package daemonconfig

import "testing"

func TestDecisionTierString(t *testing.T) {
	tests := []struct {
		tier DecisionTier
		want string
	}{
		{TierAutonomous, "autonomous"},
		{TierProposeAndAct, "propose-and-act"},
		{TierGenuineDecision, "genuine-decision"},
		{DecisionTier(99), "autonomous"}, // unknown defaults to autonomous (safe: most cautious behavior is to act)
	}
	for _, tt := range tests {
		if got := tt.tier.String(); got != tt.want {
			t.Errorf("DecisionTier(%d).String() = %q, want %q", tt.tier, got, tt.want)
		}
	}
}

func TestDecisionClassString(t *testing.T) {
	tests := []struct {
		class DecisionClass
		want  string
	}{
		{DecisionSelectIssue, "select_issue"},
		{DecisionInferSkill, "infer_skill"},
		{DecisionInferModel, "infer_model"},
		{DecisionRouteExtraction, "route_extraction"},
		{DecisionArchitectEscalate, "architect_escalate"},
		{DecisionAutoCompleteLight, "auto_complete_light"},
		{DecisionAutoCompleteFull, "auto_complete_full"},
		{DecisionLabelForReview, "label_for_review"},
		{DecisionCreateSynthesisIssue, "create_synthesis_issue"},
		{DecisionCreateModelDriftIssue, "create_model_drift_issue"},
		{DecisionCreateAgreementIssue, "create_agreement_issue"},
		{DecisionResetOrphan, "reset_orphan"},
		{DecisionResumeStuck, "resume_stuck"},
		{DecisionFlagPhaseTimeout, "flag_phase_timeout"},
		{DecisionDowngradeCompliance, "downgrade_compliance"},
		{DecisionDetectDuplicate, "detect_duplicate"},
		{DecisionSurfaceRemoval, "surface_removal"},
		{DecisionCullBacklog, "cull_backlog"},
		{DecisionAdvancePlan, "advance_plan"},
	}
	for _, tt := range tests {
		if got := tt.class.String(); got != tt.want {
			t.Errorf("DecisionClass(%d).String() = %q, want %q", tt.class, got, tt.want)
		}
	}
}

func TestDecisionClassCategory(t *testing.T) {
	tests := []struct {
		class DecisionClass
		want  string
	}{
		{DecisionSelectIssue, "spawn"},
		{DecisionInferSkill, "spawn"},
		{DecisionInferModel, "spawn"},
		{DecisionRouteExtraction, "spawn"},
		{DecisionArchitectEscalate, "spawn"},
		{DecisionAutoCompleteLight, "completion"},
		{DecisionAutoCompleteFull, "completion"},
		{DecisionLabelForReview, "completion"},
		{DecisionCreateSynthesisIssue, "knowledge"},
		{DecisionCreateModelDriftIssue, "knowledge"},
		{DecisionCreateAgreementIssue, "knowledge"},
		{DecisionResetOrphan, "lifecycle"},
		{DecisionResumeStuck, "lifecycle"},
		{DecisionFlagPhaseTimeout, "lifecycle"},
		{DecisionDowngradeCompliance, "compliance"},
		{DecisionDetectDuplicate, "work_graph"},
		{DecisionSurfaceRemoval, "work_graph"},
		{DecisionCullBacklog, "work_graph"},
		{DecisionAdvancePlan, "work_graph"},
	}
	for _, tt := range tests {
		if got := tt.class.Category(); got != tt.want {
			t.Errorf("DecisionClass(%d).Category() = %q, want %q", tt.class, got, tt.want)
		}
	}
}

func TestClassifyDecision_BaseTiers(t *testing.T) {
	// At ComplianceStandard, base tiers should be used as-is
	tests := []struct {
		class DecisionClass
		want  DecisionTier
	}{
		// Tier 1 (autonomous) decisions at standard compliance
		{DecisionSelectIssue, TierAutonomous},
		{DecisionInferSkill, TierAutonomous},
		{DecisionInferModel, TierAutonomous},
		{DecisionResetOrphan, TierAutonomous},
		{DecisionFlagPhaseTimeout, TierAutonomous},
		{DecisionDetectDuplicate, TierAutonomous},

		// Tier 2 (propose-and-act) decisions at standard compliance
		{DecisionRouteExtraction, TierProposeAndAct},
		{DecisionArchitectEscalate, TierProposeAndAct},
		{DecisionAutoCompleteLight, TierProposeAndAct},
		{DecisionAutoCompleteFull, TierProposeAndAct},
		{DecisionLabelForReview, TierProposeAndAct},
		{DecisionCreateSynthesisIssue, TierProposeAndAct},
		{DecisionCreateModelDriftIssue, TierProposeAndAct},
		{DecisionCreateAgreementIssue, TierProposeAndAct},
		{DecisionResumeStuck, TierProposeAndAct},
		{DecisionSurfaceRemoval, TierProposeAndAct},
		{DecisionCullBacklog, TierProposeAndAct},
		{DecisionAdvancePlan, TierProposeAndAct},

		// Tier 3 (genuine-decision) decisions at standard compliance
		{DecisionDowngradeCompliance, TierGenuineDecision},
	}
	for _, tt := range tests {
		got := ClassifyDecision(tt.class, ComplianceStandard)
		if got != tt.want {
			t.Errorf("ClassifyDecision(%v, standard) = %v, want %v", tt.class, got, tt.want)
		}
	}
}

func TestClassifyDecision_StrictPromotesAll(t *testing.T) {
	// Strict promotes all tiers one level (more cautious)
	tests := []struct {
		class DecisionClass
		want  DecisionTier
	}{
		// Base T1 → T2 under strict
		{DecisionSelectIssue, TierProposeAndAct},
		{DecisionInferSkill, TierProposeAndAct},

		// Base T2 → T3 under strict
		{DecisionAutoCompleteLight, TierGenuineDecision},
		{DecisionArchitectEscalate, TierGenuineDecision},

		// Base T3 stays T3 (can't go higher)
		{DecisionDowngradeCompliance, TierGenuineDecision},
	}
	for _, tt := range tests {
		got := ClassifyDecision(tt.class, ComplianceStrict)
		if got != tt.want {
			t.Errorf("ClassifyDecision(%v, strict) = %v, want %v", tt.class, got, tt.want)
		}
	}
}

func TestClassifyDecision_RelaxedDemotesOne(t *testing.T) {
	// Relaxed demotes one level (more autonomous)
	tests := []struct {
		class DecisionClass
		want  DecisionTier
	}{
		// Base T1 stays T1 (can't go lower)
		{DecisionSelectIssue, TierAutonomous},

		// Base T2 → T1 under relaxed
		{DecisionAutoCompleteLight, TierAutonomous},
		{DecisionArchitectEscalate, TierAutonomous},

		// Base T3 → T2 under relaxed
		{DecisionDowngradeCompliance, TierProposeAndAct},
	}
	for _, tt := range tests {
		got := ClassifyDecision(tt.class, ComplianceRelaxed)
		if got != tt.want {
			t.Errorf("ClassifyDecision(%v, relaxed) = %v, want %v", tt.class, got, tt.want)
		}
	}
}

func TestClassifyDecision_AutonomousMakesAllT1ExceptBaseT3(t *testing.T) {
	// Autonomous: everything T1 except base-T3 → T2
	tests := []struct {
		class DecisionClass
		want  DecisionTier
	}{
		// Base T1 stays T1
		{DecisionSelectIssue, TierAutonomous},
		{DecisionInferSkill, TierAutonomous},

		// Base T2 → T1
		{DecisionAutoCompleteLight, TierAutonomous},
		{DecisionArchitectEscalate, TierAutonomous},
		{DecisionCreateSynthesisIssue, TierAutonomous},

		// Base T3 → T2 (not T1 — safety floor)
		{DecisionDowngradeCompliance, TierProposeAndAct},
	}
	for _, tt := range tests {
		got := ClassifyDecision(tt.class, ComplianceAutonomous)
		if got != tt.want {
			t.Errorf("ClassifyDecision(%v, autonomous) = %v, want %v", tt.class, got, tt.want)
		}
	}
}

func TestClassifyDecision_AllClassesCovered(t *testing.T) {
	// Ensure every DecisionClass from 0 to DecisionAdvancePlan returns a valid tier
	for class := DecisionSelectIssue; class <= DecisionAdvancePlan; class++ {
		for _, level := range []ComplianceLevel{ComplianceStrict, ComplianceStandard, ComplianceRelaxed, ComplianceAutonomous} {
			tier := ClassifyDecision(class, level)
			if tier < TierAutonomous || tier > TierGenuineDecision {
				t.Errorf("ClassifyDecision(%v, %v) = %v, outside valid range", class, level, tier)
			}
		}
	}
}
