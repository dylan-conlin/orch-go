package orch

import (
	"fmt"
	"os"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/events"
	"github.com/dylan-conlin/orch-go/pkg/spawn"
)

func GatherSpawnContext(skillContent, task, orientationFrame, beadsID, projectDir, workspaceName, skillName string, skipArtifactCheck, gateOnGap, skipGapGate bool, gapThreshold int) (
	kbContext string, gapAnalysis *spawn.GapAnalysis, hasInjectedModels bool, primaryModelPath string, crossRepoModelDir string, err error) {
	stalenessMeta := &spawn.StalenessEventMeta{SpawnID: workspaceName, AgentSkill: skillName}
	if skipArtifactCheck {
		fmt.Println("Skipping context check (--skip-artifact-check)")
		return "", nil, false, "", "", nil
	}
	requires := spawn.ParseSkillRequires(skillContent)
	if requires != nil && requires.HasRequirements() {
		fmt.Printf("Gathering context (skill requires: %s)\n", requires.String())
		kbContext = spawn.GatherRequiredContext(requires, task, beadsID, projectDir, stalenessMeta)
		gapAnalysis = spawn.AnalyzeGaps(nil, task, projectDir)
	} else {
		gapResult := runPreSpawnKBCheckFull(task, orientationFrame, projectDir, stalenessMeta)
		kbContext = gapResult.Context
		gapAnalysis = gapResult.GapAnalysis
		if gapResult.FormatResult != nil {
			hasInjectedModels = gapResult.FormatResult.HasInjectedModels
			if hasInjectedModels {
				primaryModelPath = extractPrimaryModelPath(gapResult.FormatResult)
			}
			crossRepoModelDir = gapResult.FormatResult.CrossRepoModelDir
		}
	}
	if err := checkGapGating(gapAnalysis, gateOnGap, skipGapGate, gapThreshold); err != nil {
		return "", nil, false, "", "", err
	}
	if gapAnalysis != nil && gapAnalysis.HasGaps {
		recordGapForLearning(gapAnalysis, skillContent, task)
	}
	if skipGapGate && gapAnalysis != nil && gapAnalysis.ShouldBlockSpawn(gapThreshold) {
		fmt.Fprintf(os.Stderr, "⚠️  Bypassing gap gate (--skip-gap-gate): context quality %d\n", gapAnalysis.ContextQuality)
		logger := events.NewLogger(events.DefaultLogPath())
		event := events.Event{
			Type: "gap.gate.bypassed", Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{"task": task, "context_quality": gapAnalysis.ContextQuality, "beads_id": beadsID, "skill": skillContent},
		}
		if err := logger.Log(event); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log gap bypass: %v\n", err)
		}
	}
	return kbContext, gapAnalysis, hasInjectedModels, primaryModelPath, crossRepoModelDir, nil
}

func extractPrimaryModelPath(formatResult *spawn.KBContextFormatResult) string {
	if formatResult == nil {
		return ""
	}
	return formatResult.PrimaryModelPath
}

func runPreSpawnKBCheck(task, orientationFrame, projectDir string, stalenessMeta *spawn.StalenessEventMeta) string {
	result := runPreSpawnKBCheckFull(task, orientationFrame, projectDir, stalenessMeta)
	return result.Context
}

func runPreSpawnKBCheckFull(task, orientationFrame, projectDir string, stalenessMeta *spawn.StalenessEventMeta) *GapCheckResult {
	gcr := &GapCheckResult{}
	var keywords string
	if orientationFrame != "" {
		keywords = spawn.ExtractKeywordsWithContext(task, orientationFrame, 5)
	} else {
		keywords = spawn.ExtractKeywords(task, 3)
	}
	if keywords == "" {
		gcr.GapAnalysis = spawn.AnalyzeGaps(nil, task, projectDir)
		if gcr.GapAnalysis.ShouldWarnAboutGaps() {
			fmt.Fprintf(os.Stderr, "%s", gcr.GapAnalysis.FormatProminentWarning())
		}
		return gcr
	}
	fmt.Printf("Checking kb context for: %q\n", keywords)
	result, err := spawn.RunKBContextCheckForDir(keywords, projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
		return gcr
	}
	if result == nil || !result.HasMatches {
		firstKeyword := spawn.ExtractKeywords(task, 1)
		if firstKeyword != "" && firstKeyword != keywords {
			fmt.Printf("Trying broader search for: %q\n", firstKeyword)
			result, err = spawn.RunKBContextCheckForDir(firstKeyword, projectDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: kb context check failed: %v\n", err)
				return gcr
			}
		}
	}
	gcr.GapAnalysis = spawn.AnalyzeGaps(result, keywords, projectDir)
	if gcr.GapAnalysis.ShouldWarnAboutGaps() {
		fmt.Fprintf(os.Stderr, "%s", gcr.GapAnalysis.FormatProminentWarning())
	}
	if result == nil || !result.HasMatches {
		fmt.Println("No prior knowledge found.")
		return gcr
	}
	fmt.Printf("Found %d relevant context entries - including in spawn context.\n", len(result.Matches))
	maxChars := spawn.MaxKBContextChars
	if spawn.TaskIsScoped(task) {
		originalCount := len(result.Matches)
		result.Matches = spawn.FilterForScopedTask(result.Matches)
		result.HasMatches = len(result.Matches) > 0
		maxChars = spawn.ScopedMaxKBContextChars
		fmt.Printf("Scoped task detected: filtered %d → %d matches (budget: %dk chars)\n", originalCount, len(result.Matches), maxChars/1000)
		if !result.HasMatches {
			fmt.Println("No relevant context after scoped filtering.")
			return gcr
		}
	}
	formatResult := spawn.FormatContextForSpawnWithLimitAndMeta(result, maxChars, projectDir, stalenessMeta)
	gcr.FormatResult = formatResult
	contextContent := formatResult.Content
	if gapSummary := gcr.GapAnalysis.FormatGapSummary(); gapSummary != "" {
		contextContent = gapSummary + "\n\n" + contextContent
	}
	gcr.Context = contextContent
	return gcr
}

func checkGapGating(gapAnalysis *spawn.GapAnalysis, gateEnabled, skipGate bool, threshold int) error {
	if !gateEnabled || skipGate {
		return nil
	}
	if gapAnalysis == nil {
		return nil
	}
	if threshold <= 0 {
		threshold = spawn.DefaultGateThreshold
	}
	if gapAnalysis.ShouldBlockSpawn(threshold) {
		fmt.Fprintf(os.Stderr, "%s", gapAnalysis.FormatGateBlockMessage())
		return fmt.Errorf("spawn blocked: context quality %d is below threshold %d", gapAnalysis.ContextQuality, threshold)
	}
	return nil
}

func recordGapForLearning(gapAnalysis *spawn.GapAnalysis, skill, task string) {
	tracker, err := spawn.LoadTracker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load gap tracker: %v\n", err)
		return
	}
	tracker.RecordGap(gapAnalysis, skill, task)
	suggestions := tracker.FindRecurringGaps()
	if len(suggestions) > 0 {
		hasHighPriority := false
		for _, s := range suggestions {
			if s.Priority == "high" && s.Count >= spawn.RecurrenceThreshold {
				hasHighPriority = true
				break
			}
		}
		if hasHighPriority {
			fmt.Fprintf(os.Stderr, "%s", spawn.FormatSuggestions(suggestions))
		}
	}
	if err := tracker.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save gap tracker: %v\n", err)
	}
}
