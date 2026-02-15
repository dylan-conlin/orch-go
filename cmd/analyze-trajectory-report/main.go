package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type CommitType string

const (
	TypeFeat      CommitType = "feat"
	TypeFix       CommitType = "fix"
	TypeInv       CommitType = "inv"
	TypeArchitect CommitType = "architect"
	TypeBdSync    CommitType = "bd sync"
	TypeChore     CommitType = "chore"
	TypeRefactor  CommitType = "refactor"
	TypeTest      CommitType = "test"
	TypeDocs      CommitType = "docs"
	TypeWip       CommitType = "wip"
	TypeOther     CommitType = "other"
)

type KnowledgeArtifact struct {
	Path string
	Type string
}

type DayStats struct {
	Date               string              `json:"date"`
	TotalCommits       int                 `json:"total_commits"`
	CommitsByType      map[CommitType]int  `json:"commits_by_type"`
	KnowledgeArtifacts []KnowledgeArtifact `json:"knowledge_artifacts"`
	FeatureCommits     []string            `json:"feature_commits"`
	FixCommits         []string            `json:"fix_commits"`
	NetLOC             int                 `json:"net_loc"`
	FilesChanged       int                 `json:"files_changed"`
}

type Timeline struct {
	StartDate string               `json:"start_date"`
	EndDate   string               `json:"end_date"`
	TotalDays int                  `json:"total_days"`
	Days      map[string]*DayStats `json:"days"`
}

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

	fmt.Println("| Date | Total | feat | fix | inv | arch | bd | chore | refactor | test | docs | wip | other | Fix:Feat |")
	fmt.Println("|------|-------|------|-----|-----|------|-------|----------|------|------|-----|-------|----------|")

	for _, date := range dates {
		day := r.timeline.Days[date]

		fixCount := day.CommitsByType[TypeFix]
		featCount := day.CommitsByType[TypeFeat]
		ratio := "-"
		if featCount > 0 {
			ratio = fmt.Sprintf("%.2f", float64(fixCount)/float64(featCount))
		}

		fmt.Printf("| %s | %d | %d | %d | %d | %d | %d | %d | %d | %d | %d | %d | %d | %s |\n",
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
		)
	}
	fmt.Println()
}

func (r *ReportGenerator) printChurningPeriods() {
	fmt.Println("## Churning vs Productive Periods")
	fmt.Println()

	type dayAnalysis struct {
		Date           string
		Commits        int
		FixFeatRatio   float64
		Investigations int
		Score          float64
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

	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].Score > analyses[j].Score
	})

	fmt.Println("### Top 15 Churning Days")
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
}

func (r *ReportGenerator) printKnowledgeArtifacts() {
	fmt.Println("## Knowledge Artifacts")
	fmt.Println()

	artifactsByType := make(map[string]int)

	for _, day := range r.timeline.Days {
		for _, artifact := range day.KnowledgeArtifacts {
			artifactsByType[artifact.Type]++
		}
	}

	total := 0
	for _, count := range artifactsByType {
		total += count
	}

	fmt.Printf("**Total:** %d (sampled)\n\n", total)
	fmt.Printf("- Investigations: %d\n", artifactsByType["investigation"])
	fmt.Printf("- Decisions: %d\n", artifactsByType["decision"])
	fmt.Printf("- Models: %d\n", artifactsByType["model"])
	fmt.Println()
}

func (r *ReportGenerator) printFeatureVsFixAnalysis() {
	fmt.Println("## Feature vs Fix Patterns")
	fmt.Println()

	var dates []string
	for date := range r.timeline.Days {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	fmt.Println("### Days with 3+ Features and Subsequent Fixes")
	fmt.Println()
	fmt.Println("| Date | Features | Day 0 Fixes | Day +1 | Day +2 | Day +3 | Total | Pattern |")
	fmt.Println("|------|----------|-------------|--------|--------|--------|-------|---------|")

	for i, date := range dates {
		day := r.timeline.Days[date]

		if day.CommitsByType[TypeFeat] < 3 {
			continue
		}

		fixes := []int{day.CommitsByType[TypeFix]}

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
			pattern = "Moderate"
		}

		fixStr := make([]string, 4)
		for i := 0; i < 4; i++ {
			if i < len(fixes) {
				fixStr[i] = fmt.Sprintf("%d", fixes[i])
			} else {
				fixStr[i] = "-"
			}
		}

		fmt.Printf("| %s | %d | %s | %s | %s | %s | %d | %s |\n",
			date,
			day.CommitsByType[TypeFeat],
			fixStr[0], fixStr[1], fixStr[2], fixStr[3],
			totalFixes,
			pattern)
	}
	fmt.Println()
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

func main() {
	timeline, err := loadTimeline("/tmp/timeline.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading timeline: %v\n", err)
		os.Exit(1)
	}

	generator := &ReportGenerator{timeline: timeline}
	generator.GenerateReport()
}
