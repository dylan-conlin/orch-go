// stats_inference.go - Skill inference accuracy tracking for stats aggregation.
// Correlates spawn.skill_inferred events with agent completion/abandonment outcomes.
package main

import "sort"

// skillInferenceRecord tracks a single skill inference for outcome correlation.
type skillInferenceRecord struct {
	issueID       string
	inferredSkill string
	method        string // "label", "title", "description", "type"
}

func (a *statsAggregator) processSkillInferred(event StatsEvent) {
	// Record all inferences (even outside window) for outcome correlation,
	// since the completion event may be in-window while inference was earlier.
	if data := event.Data; data != nil {
		issueID, _ := data["issue_id"].(string)
		inferredSkill, _ := data["inferred_skill"].(string)
		if issueID == "" || inferredSkill == "" {
			return
		}

		// Determine inference method
		method := "type" // default fallback
		if hadLabel, ok := data["had_skill_label"].(bool); ok && hadLabel {
			method = "label"
		} else if hadTitle, ok := data["had_title_match"].(bool); ok && hadTitle {
			method = "title"
		} else if usedDesc, ok := data["used_description_heuristic"].(bool); ok && usedDesc {
			method = "description"
		}

		a.skillInferences[issueID] = &skillInferenceRecord{
			issueID:       issueID,
			inferredSkill: inferredSkill,
			method:        method,
		}
	}
}

func (a *statsAggregator) calcSkillInferenceStats() {
	if len(a.skillInferences) == 0 {
		return
	}

	// Aggregate by method and skill
	methodStats := make(map[string]*InferenceMethodStats) // method -> stats
	skillStats := make(map[string]*InferenceSkillStats)   // skill -> stats

	for issueID, rec := range a.skillInferences {
		// Check if this inference led to a completion or abandonment in-window
		completed := a.completedBeadsIDs[issueID]
		abandoned := a.abandonedBeadsIDs[issueID]

		if !completed && !abandoned {
			continue // No outcome yet, skip
		}

		a.report.SkillInferenceStats.TotalInferences++

		// By method
		ms, ok := methodStats[rec.method]
		if !ok {
			ms = &InferenceMethodStats{Method: rec.method}
			methodStats[rec.method] = ms
		}
		ms.Inferences++

		// By skill
		ss, ok := skillStats[rec.inferredSkill]
		if !ok {
			ss = &InferenceSkillStats{Skill: rec.inferredSkill}
			skillStats[rec.inferredSkill] = ss
		}
		ss.Inferences++

		if completed {
			a.report.SkillInferenceStats.Completed++
			ms.Completed++
			ss.Completed++
		}
		if abandoned {
			a.report.SkillInferenceStats.Abandoned++
			ms.Abandoned++
			ss.Abandoned++
		}
	}

	// Compute rates
	total := a.report.SkillInferenceStats.Completed + a.report.SkillInferenceStats.Abandoned
	if total > 0 {
		a.report.SkillInferenceStats.SuccessRate = float64(a.report.SkillInferenceStats.Completed) / float64(total) * 100
	}

	for _, ms := range methodStats {
		t := ms.Completed + ms.Abandoned
		if t > 0 {
			ms.SuccessRate = float64(ms.Completed) / float64(t) * 100
		}
		a.report.SkillInferenceStats.ByMethod = append(a.report.SkillInferenceStats.ByMethod, *ms)
	}
	sort.Slice(a.report.SkillInferenceStats.ByMethod, func(i, j int) bool {
		return a.report.SkillInferenceStats.ByMethod[i].Inferences > a.report.SkillInferenceStats.ByMethod[j].Inferences
	})

	for _, ss := range skillStats {
		t := ss.Completed + ss.Abandoned
		if t > 0 {
			ss.SuccessRate = float64(ss.Completed) / float64(t) * 100
		}
		a.report.SkillInferenceStats.BySkill = append(a.report.SkillInferenceStats.BySkill, *ss)
	}
	sort.Slice(a.report.SkillInferenceStats.BySkill, func(i, j int) bool {
		return a.report.SkillInferenceStats.BySkill[i].Inferences > a.report.SkillInferenceStats.BySkill[j].Inferences
	})
}
