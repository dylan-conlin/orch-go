package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type ReportGenerator struct {
	timeline *Timeline
}

func (r *ReportGenerator) GenerateReport() {
	fmt.Println("# System Trajectory Analysis: Dec 19, 2025 - Feb 12, 2026")
	fmt.Println()
	fmt.Printf("**Period:** %s to %s (%d days)\n", r.timeline.StartDate, r.timeline.EndDate, r.timeline.TotalDays)
	fmt.Printf("**Total Commits:** %d\n", r.totalCommits())
	fmt.Println()

	r.printOverallStats()
	r.printDayByDayBreakdown()
	r.printChurningPeriods()
	r.printKnowledgeArtifacts()
	r.printFeatureVsFixAnalysis()
}

func (r *ReportGenerator) totalCommits() int {
	total := 0
	for _, day := range r.timeline.Days {
		total += day.TotalCommits
	}
	return total
}

func (r *ReportGenerator) printOverallStats() {
	fmt.Println("## Overall Statistics")
	fmt.Println()

	// Aggregate commit types
	typeStats := make(map[CommitType]int)
	for _, day := range r.timeline.Days {
		for ctype, count := range day.CommitsByType {
			typeStats[ctype] += count
		}
	}

	// Sort by count
	type typeStat struct {
		Type  CommitType
		Count int
	}
	var stats []typeStat
	for t, c := range typeStats {
		stats = append(stats, typeStat{t, c})
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	fmt.Println("### Commits by Type")
	fmt.Println()
	fmt.Println("| Type | Count | Percentage |")
	fmt.Println("|------|-------|------------|")
	total := r.totalCommits()
	for _, s := range stats {
		pct := float64(s.Count) / float64(total) * 100
		fmt.Printf("| %s | %d | %.1f%% |\n", s.Type, s.Count, pct)
	}
	fmt.Println()

	// Fix:Feat ratio
	fixCount := typeStats[TypeFix]
	featCount := typeStats[TypeFeat]
	if featCount > 0 {
		ratio := float64(fixCount) / float64(featCount)
		fmt.Printf("**Fix:Feat Ratio:** %.2f:1 (%d fixes / %d features)\n", ratio, fixCount, featCount)
		fmt.Println()
	}
}

func (r *ReportGenerator) printDayByDayBreakdown() {
	fmt.Println("## Day-by-Day Breakdown")
	fmt.Println()

	// Sort dates
	var dates []string
	for date := range r.timeline.Days {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	fmt.Println("| Date | Total | feat | fix | inv | architect | bd sync | chore | refactor | test | docs | wip | other | Fix:Feat | Knowledge |")
	fmt.Println("|------|-------|------|-----|-----|-----------|---------|-------|----------|------|------|-----|-------|----------|-----------|")

	for _, date := range dates {
		day := r.timeline.Days[date]

		fixCount := day.CommitsByType[TypeFix]
		featCount := day.CommitsByType[TypeFeat]
		ratio := "-"
		if featCount > 0 {
			ratio = fmt.Sprintf("%.2f", float64(fixCount)/float64(featCount))
		}

		kbCount := len(day.KnowledgeArtifacts)
		kbStr := "-"
		if kbCount > 0 {
			kbStr = fmt.Sprintf("%d", kbCount)
		}

		fmt.Printf("| %s | %d | %d | %d | %d | %d | %d | %d | %d | %d | %d | %d | %d | %s | %s |\n",
			date,
			day.TotalCommits,
			day.CommitsByType[TypeFeat],
			day.CommitsByType[TypeFix],
			day.CommitsByType[TypeInv],
			day.CommitsByType[TypeArchitect],
			day.CommitsByType[TypeBdSync],
			day.CommitsByType[TypeChore],
			day.CommitsByType[TypeRefactor],
			day.CommitsByType[TypeTest],
			day.CommitsByType[TypeDocs],
			day.CommitsByType[TypeWip],
			day.CommitsByType[TypeOther],
			ratio,
			kbStr,
		)
	}
	fmt.Println()
}

func (r *ReportGenerator) printChurningPeriods() {
	fmt.Println("## Churning vs Productive Periods")
	fmt.Println()

	// Define churning as days with:
	// 1. High fix:feat ratio (>0.7)
	// 2. High commit count (>30)
	// 3. Many investigations

	type dayAnalysis struct {
		Date           string
		Commits        int
		FixFeatRatio   float64
		Investigations int
		Score          float64 // churn score
	}

	var analyses []dayAnalysis

	for date, day := range r.timeline.Days {
		fixCount := day.CommitsByType[TypeFix]
		featCount := day.CommitsByType[TypeFeat]
		invCount := day.CommitsByType[TypeInv]

		ratio := 0.0
		if featCount > 0 {
			ratio = float64(fixCount) / float64(featCount)
		}

		// Churn score: weighted combination
		score := 0.0
		if day.TotalCommits > 30 {
			score += float64(day.TotalCommits-30) * 0.5
		}
		if ratio > 0.7 {
			score += (ratio - 0.7) * 100
		}
		score += float64(invCount) * 2

		analyses = append(analyses, dayAnalysis{
			Date:           date,
			Commits:        day.TotalCommits,
			FixFeatRatio:   ratio,
			Investigations: invCount,
			Score:          score,
		})
	}

	// Sort by churn score
	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].Score > analyses[j].Score
	})

	fmt.Println("### Top 15 Churning Days (by churn score)")
	fmt.Println()
	fmt.Println("| Date | Commits | Fix:Feat | Investigations | Churn Score | Assessment |")
	fmt.Println("|------|---------|----------|----------------|-------------|------------|")

	for i := 0; i < 15 && i < len(analyses); i++ {
		a := analyses[i]

		assessment := "Productive"
		if a.Score > 30 {
			assessment = "High Churn"
		} else if a.Score > 15 {
			assessment = "Moderate Churn"
		}

		ratioStr := "-"
		if a.FixFeatRatio > 0 {
			ratioStr = fmt.Sprintf("%.2f", a.FixFeatRatio)
		}

		fmt.Printf("| %s | %d | %s | %d | %.1f | %s |\n",
			a.Date, a.Commits, ratioStr, a.Investigations, a.Score, assessment)
	}
	fmt.Println()

	// Also show most productive days (low churn, high feat)
	sort.Slice(analyses, func(i, j int) bool {
		// Productive = high feat, low fix, low inv
		scoreI := float64(r.timeline.Days[analyses[i].Date].CommitsByType[TypeFeat]) -
			float64(r.timeline.Days[analyses[i].Date].CommitsByType[TypeFix])*0.5 -
			float64(analyses[i].Investigations)
		scoreJ := float64(r.timeline.Days[analyses[j].Date].CommitsByType[TypeFeat]) -
			float64(r.timeline.Days[analyses[j].Date].CommitsByType[TypeFix])*0.5 -
			float64(analyses[j].Investigations)
		return scoreI > scoreJ
	})

	fmt.Println("### Top 10 Productive Days (high features, low fixes)")
	fmt.Println()
	fmt.Println("| Date | Commits | Features | Fixes | Investigations | Net Feature Progress |")
	fmt.Println("|------|---------|----------|-------|----------------|----------------------|")

	for i := 0; i < 10 && i < len(analyses); i++ {
		a := analyses[i]
		day := r.timeline.Days[a.Date]
		netProgress := day.CommitsByType[TypeFeat] - day.CommitsByType[TypeFix]

		if day.CommitsByType[TypeFeat] == 0 {
			continue // skip days with no features
		}

		fmt.Printf("| %s | %d | %d | %d | %d | %d |\n",
			a.Date, a.Commits,
			day.CommitsByType[TypeFeat],
			day.CommitsByType[TypeFix],
			a.Investigations,
			netProgress)
	}
	fmt.Println()
}

func (r *ReportGenerator) printKnowledgeArtifacts() {
	fmt.Println("## Knowledge Artifacts Created")
	fmt.Println()

	// Aggregate by type
	artifactsByType := make(map[string][]KnowledgeArtifact)
	artifactsByDate := make(map[string][]KnowledgeArtifact)

	for date, day := range r.timeline.Days {
		for _, artifact := range day.KnowledgeArtifacts {
			artifactsByType[artifact.Type] = append(artifactsByType[artifact.Type], artifact)
			artifactsByDate[date] = append(artifactsByDate[date], artifact)
		}
	}

	fmt.Printf("**Total Knowledge Artifacts:** %d\n",
		len(artifactsByType["investigation"])+len(artifactsByType["decision"])+len(artifactsByType["model"]))
	fmt.Println()

	fmt.Println("### By Type")
	fmt.Println()
	fmt.Printf("- **Investigations:** %d\n", len(artifactsByType["investigation"]))
	fmt.Printf("- **Decisions:** %d\n", len(artifactsByType["decision"]))
	fmt.Printf("- **Models:** %d\n", len(artifactsByType["model"]))
	fmt.Println()

	// Find days with most knowledge creation
	type dateKB struct {
		Date  string
		Count int
	}
	var dateKBs []dateKB
	for date, artifacts := range artifactsByDate {
		dateKBs = append(dateKBs, dateKB{date, len(artifacts)})
	}
	sort.Slice(dateKBs, func(i, j int) bool {
		return dateKBs[i].Count > dateKBs[j].Count
	})

	fmt.Println("### Days with Most Knowledge Creation")
	fmt.Println()
	fmt.Println("| Date | Artifacts Created |")
	fmt.Println("|------|-------------------|")
	for i := 0; i < 10 && i < len(dateKBs); i++ {
		if dateKBs[i].Count > 0 {
			fmt.Printf("| %s | %d |\n", dateKBs[i].Date, dateKBs[i].Count)
		}
	}
	fmt.Println()
}

func (r *ReportGenerator) printFeatureVsFixAnalysis() {
	fmt.Println("## Feature Introduction vs Fix Cascades")
	fmt.Println()

	// Analyze whether features trigger fix cascades
	// Look for patterns: feature commit followed by multiple fixes

	var dates []string
	for date := range r.timeline.Days {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	fmt.Println("### Feature-Heavy Days and Subsequent Fix Activity")
	fmt.Println()
	fmt.Println("Analyzing days with 3+ feature commits and tracking fixes in the following 3 days:")
	fmt.Println()
	fmt.Println("| Date | Features | Fixes Day 0 | Fixes Day +1 | Fixes Day +2 | Fixes Day +3 | Total Fixes | Pattern |")
	fmt.Println("|------|----------|-------------|--------------|--------------|--------------|-------------|---------|")

	for i, date := range dates {
		day := r.timeline.Days[date]

		if day.CommitsByType[TypeFeat] < 3 {
			continue
		}

		fixes := []int{day.CommitsByType[TypeFix]}

		// Get fixes from next 3 days
		for j := 1; j <= 3 && i+j < len(dates); j++ {
			nextDate := dates[i+j]
			nextDay := r.timeline.Days[nextDate]
			fixes = append(fixes, nextDay.CommitsByType[TypeFix])
		}

		totalFixes := 0
		for _, f := range fixes {
			totalFixes += f
		}

		pattern := "Stable"
		if totalFixes > day.CommitsByType[TypeFeat]*2 {
			pattern = "Fix Cascade"
		} else if totalFixes > day.CommitsByType[TypeFeat] {
			pattern = "Moderate Fixes"
		}

		fixStr := make([]string, len(fixes))
		for i, f := range fixes {
			if i < len(fixes) {
				fixStr[i] = fmt.Sprintf("%d", f)
			}
		}

		// Pad to 4 columns
		for len(fixStr) < 4 {
			fixStr = append(fixStr, "-")
		}

		fmt.Printf("| %s | %d | %s | %s | %s | %s | %d | %s |\n",
			date,
			day.CommitsByType[TypeFeat],
			fixStr[0], fixStr[1], fixStr[2], fixStr[3],
			totalFixes,
			pattern)
	}
	fmt.Println()

	fmt.Println("### Notable Feature Commits")
	fmt.Println()
	fmt.Println("Sample of feature commits from high-activity days:")
	fmt.Println()

	count := 0
	for _, date := range dates {
		day := r.timeline.Days[date]

		if len(day.FeatureCommits) > 0 && day.TotalCommits > 20 {
			fmt.Printf("**%s** (%d commits):\n", date, day.TotalCommits)
			for i, msg := range day.FeatureCommits {
				if i >= 5 {
					fmt.Printf("  - ... and %d more\n", len(day.FeatureCommits)-5)
					break
				}
				// Truncate long messages
				if len(msg) > 80 {
					msg = msg[:77] + "..."
				}
				fmt.Printf("  - %s\n", msg)
			}
			fmt.Println()
			count++
			if count >= 10 {
				break
			}
		}
	}
}

func loadTimeline(path string) (*Timeline, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var timeline Timeline
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&timeline); err != nil {
		return nil, err
	}

	return &timeline, nil
}

func runReport() {
	timeline, err := loadTimeline("/tmp/timeline.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading timeline: %v\n", err)
		os.Exit(1)
	}

	generator := &ReportGenerator{timeline: timeline}
	generator.GenerateReport()
}
