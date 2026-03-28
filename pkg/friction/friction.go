// Package friction parses Friction: comments from beads issues.jsonl
// and produces aggregate metrics: category breakdown, recurring sources,
// per-skill friction rates, and weekly trends.
package friction

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Entry represents a single parsed friction comment.
type Entry struct {
	IssueID   string
	Assignee  string // agent assignee, e.g. "og-feat-xxx-27mar-abc1"
	Skill     string // extracted skill prefix, e.g. "feat", "debug", "inv"
	Category  string // tooling, ceremony, bug, gap, capacity
	Message   string // the message after "Friction: <category>: "
	CreatedAt time.Time
}

// Report holds the aggregated friction metrics.
type Report struct {
	TotalIssues       int                `json:"total_issues"`
	TotalComments     int                `json:"total_comments"`     // all Friction: comments including "none"
	FrictionCount     int                `json:"friction_count"`     // actual friction (non-none)
	NoneCount         int                `json:"none_count"`         // "Friction: none"
	FrictionRate      float64            `json:"friction_rate"`      // friction / total comments
	Categories        []CategoryCount    `json:"categories"`
	TopSources        []Source           `json:"top_sources"`
	SkillRates        []SkillRate        `json:"skill_rates"`
	WeeklyTrend       []WeekBucket       `json:"weekly_trend"`
	Days              int                `json:"days"`
	Entries           []Entry            `json:"entries,omitempty"` // populated with --detail
}

// CategoryCount is a category with its count and percentage.
type CategoryCount struct {
	Category   string  `json:"category"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// Source represents a recurring friction source (clustered by keyword).
type Source struct {
	Pattern string `json:"pattern"`
	Count   int    `json:"count"`
	Example string `json:"example"`
}

// SkillRate shows friction rate per skill type.
type SkillRate struct {
	Skill         string  `json:"skill"`
	Total         int     `json:"total"`         // total Friction: comments for this skill
	FrictionCount int     `json:"friction_count"` // non-none friction
	Rate          float64 `json:"rate"`
}

// WeekBucket holds friction counts for a calendar week.
type WeekBucket struct {
	Week          string  `json:"week"` // "2026-W13"
	Total         int     `json:"total"`
	FrictionCount int     `json:"friction_count"`
	Rate          float64 `json:"rate"`
}

// issue is a minimal struct for parsing the JSONL fields we need.
type issue struct {
	ID        string    `json:"id"`
	Assignee  string    `json:"assignee"`
	Comments  []comment `json:"comments"`
}

type comment struct {
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

// ParseJSONL reads issues.jsonl and extracts all Friction: comments.
// Returns friction entries and total friction comment count (including "none").
func ParseJSONL(path string, since time.Time) (entries []Entry, noneCount int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB lines

	for scanner.Scan() {
		var iss issue
		if err := json.Unmarshal(scanner.Bytes(), &iss); err != nil {
			continue // skip malformed lines
		}
		for _, c := range iss.Comments {
			if !strings.HasPrefix(c.Text, "Friction:") {
				continue
			}

			t, _ := time.Parse(time.RFC3339Nano, c.CreatedAt)
			if !since.IsZero() && t.Before(since) {
				continue
			}

			rest := strings.TrimSpace(c.Text[len("Friction:"):])

			if rest == "none" {
				noneCount++
				continue
			}

			cat, msg := parseCategory(rest)
			skill := extractSkill(iss.Assignee)

			entries = append(entries, Entry{
				IssueID:   iss.ID,
				Assignee:  iss.Assignee,
				Skill:     skill,
				Category:  cat,
				Message:   msg,
				CreatedAt: t,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("scanning %s: %w", path, err)
	}
	return entries, noneCount, nil
}

// parseCategory extracts category and message from "tooling: some description".
func parseCategory(text string) (category, message string) {
	idx := strings.Index(text, ":")
	if idx == -1 {
		// No colon — treat whole text as message, category unknown
		return "unknown", text
	}
	cat := strings.TrimSpace(text[:idx])
	msg := strings.TrimSpace(text[idx+1:])
	return strings.ToLower(cat), msg
}

// extractSkill pulls the skill prefix from an assignee like "og-feat-xxx-27mar-abc1".
func extractSkill(assignee string) string {
	if assignee == "" {
		return "unknown"
	}
	parts := strings.SplitN(assignee, "-", 3)
	if len(parts) < 2 {
		return "unknown"
	}
	return parts[1] // "feat", "debug", "inv", "arch", "research", "work"
}

// Aggregate computes the full friction report from parsed entries.
func Aggregate(entries []Entry, noneCount int, days int) *Report {
	totalComments := len(entries) + noneCount
	frictionRate := 0.0
	if totalComments > 0 {
		frictionRate = float64(len(entries)) / float64(totalComments)
	}

	r := &Report{
		TotalComments: totalComments,
		FrictionCount: len(entries),
		NoneCount:     noneCount,
		FrictionRate:  frictionRate,
		Days:          days,
	}

	r.Categories = computeCategories(entries)
	r.TopSources = computeTopSources(entries, 5)
	r.SkillRates = computeSkillRates(entries, noneCount)
	r.WeeklyTrend = computeWeeklyTrend(entries, noneCount)

	// Count distinct issues that had friction comments
	issueSet := make(map[string]bool)
	for _, e := range entries {
		issueSet[e.IssueID] = true
	}
	r.TotalIssues = len(issueSet)

	return r
}

func computeCategories(entries []Entry) []CategoryCount {
	counts := make(map[string]int)
	for _, e := range entries {
		counts[e.Category]++
	}

	var result []CategoryCount
	for cat, count := range counts {
		result = append(result, CategoryCount{
			Category:   cat,
			Count:      count,
			Percentage: float64(count) / float64(len(entries)) * 100,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	return result
}

// computeTopSources clusters friction messages by recurring keyword patterns.
func computeTopSources(entries []Entry, limit int) []Source {
	// Extract key phrases from messages and cluster by common patterns
	patterns := make(map[string][]string) // pattern -> example messages

	keywords := []struct {
		pattern string
		match   func(string) bool
	}{
		{"tab indentation / Edit tool", func(s string) bool {
			l := strings.ToLower(s)
			return strings.Contains(l, "tab") && (strings.Contains(l, "edit") || strings.Contains(l, "indentation"))
		}},
		{"git ignore / workspace staging", func(s string) bool {
			l := strings.ToLower(s)
			return (strings.Contains(l, "gitignore") || strings.Contains(l, "git add")) && strings.Contains(l, "workspace")
		}},
		{"governance hooks blocked valid action", func(s string) bool {
			l := strings.ToLower(s)
			return (strings.Contains(l, "governance") || strings.Contains(l, "hook")) && (strings.Contains(l, "block") || strings.Contains(l, "denied") || strings.Contains(l, "prevent"))
		}},
		{"pre-existing build break / compile error", func(s string) bool {
			l := strings.ToLower(s)
			return (strings.Contains(l, "build") || strings.Contains(l, "compile")) && (strings.Contains(l, "break") || strings.Contains(l, "error") || strings.Contains(l, "fail"))
		}},
		{"stale .go files / broad test failures", func(s string) bool {
			l := strings.ToLower(s)
			return (strings.Contains(l, "stale") && strings.Contains(l, ".go")) ||
				(strings.Contains(l, "go test") && strings.Contains(l, "unrelated"))
		}},
		{"concurrent agent file conflicts", func(s string) bool {
			l := strings.ToLower(s)
			return (strings.Contains(l, "concurrent") || strings.Contains(l, "parallel")) && (strings.Contains(l, "agent") || strings.Contains(l, "revert") || strings.Contains(l, "conflict"))
		}},
		{"chflags / immutable file locks", func(s string) bool {
			l := strings.ToLower(s)
			return strings.Contains(l, "chflags") || strings.Contains(l, "immutable") || strings.Contains(l, "uchg") || strings.Contains(l, "locked")
		}},
		{"kb create investigation missing flag", func(s string) bool {
			l := strings.ToLower(s)
			return strings.Contains(l, "kb create") || (strings.Contains(l, "investigation") && strings.Contains(l, "manually"))
		}},
		{"linter/hook reverted or deleted files", func(s string) bool {
			l := strings.ToLower(s)
			return (strings.Contains(l, "linter") || strings.Contains(l, "hook")) && (strings.Contains(l, "revert") || strings.Contains(l, "delet"))
		}},
		{"git stash / settings.json permissions", func(s string) bool {
			l := strings.ToLower(s)
			return strings.Contains(l, "git stash") || (strings.Contains(l, "settings.json") && strings.Contains(l, "permission"))
		}},
	}

	for _, e := range entries {
		matched := false
		for _, kw := range keywords {
			if kw.match(e.Message) {
				patterns[kw.pattern] = append(patterns[kw.pattern], e.Message)
				matched = true
				break
			}
		}
		if !matched {
			patterns["other"] = append(patterns["other"], e.Message)
		}
	}

	var sources []Source
	for pattern, msgs := range patterns {
		if pattern == "other" {
			continue
		}
		sources = append(sources, Source{
			Pattern: pattern,
			Count:   len(msgs),
			Example: truncate(msgs[0], 120),
		})
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Count > sources[j].Count
	})

	if len(sources) > limit {
		sources = sources[:limit]
	}
	return sources
}

func computeSkillRates(entries []Entry, noneCount int) []SkillRate {
	// We need per-skill total (friction + none). We only have friction entries
	// and a global none count. Approximate: distribute noneCount proportionally
	// by skill based on friction count distribution. This is imperfect but
	// issues.jsonl doesn't have skill info on none entries.

	// Actually, we can get skill from ALL friction comments including none.
	// But we only have entries for non-none. Let's compute what we have.
	skillFriction := make(map[string]int)
	for _, e := range entries {
		skillFriction[e.Skill]++
	}

	var rates []SkillRate
	for skill, count := range skillFriction {
		rates = append(rates, SkillRate{
			Skill:         skill,
			FrictionCount: count,
		})
	}
	sort.Slice(rates, func(i, j int) bool {
		return rates[i].FrictionCount > rates[j].FrictionCount
	})
	return rates
}

// ComputeSkillRatesWithNone computes skill rates using both friction entries
// and a none-by-skill map parsed from the JSONL.
func ComputeSkillRatesWithNone(entries []Entry, noneBySkill map[string]int) []SkillRate {
	skillFriction := make(map[string]int)
	for _, e := range entries {
		skillFriction[e.Skill]++
	}

	allSkills := make(map[string]bool)
	for s := range skillFriction {
		allSkills[s] = true
	}
	for s := range noneBySkill {
		allSkills[s] = true
	}

	var rates []SkillRate
	for skill := range allSkills {
		fc := skillFriction[skill]
		nc := noneBySkill[skill]
		total := fc + nc
		rate := 0.0
		if total > 0 {
			rate = float64(fc) / float64(total)
		}
		rates = append(rates, SkillRate{
			Skill:         skill,
			Total:         total,
			FrictionCount: fc,
			Rate:          rate,
		})
	}
	sort.Slice(rates, func(i, j int) bool {
		return rates[i].FrictionCount > rates[j].FrictionCount
	})
	return rates
}

func computeWeeklyTrend(entries []Entry, _ int) []WeekBucket {
	type weekData struct {
		friction int
	}
	weeks := make(map[string]*weekData)

	for _, e := range entries {
		y, w := e.CreatedAt.ISOWeek()
		key := fmt.Sprintf("%d-W%02d", y, w)
		if weeks[key] == nil {
			weeks[key] = &weekData{}
		}
		weeks[key].friction++
	}

	var result []WeekBucket
	for week, data := range weeks {
		result = append(result, WeekBucket{
			Week:          week,
			FrictionCount: data.friction,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Week < result[j].Week
	})
	return result
}

// ParseJSONLFull reads issues.jsonl and extracts all Friction: comments,
// including "none" entries with their skill info for accurate per-skill rates.
func ParseJSONLFull(path string, since time.Time) (entries []Entry, noneBySkill map[string]int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	noneBySkill = make(map[string]int)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var iss issue
		if err := json.Unmarshal(scanner.Bytes(), &iss); err != nil {
			continue
		}
		for _, c := range iss.Comments {
			if !strings.HasPrefix(c.Text, "Friction:") {
				continue
			}

			t, _ := time.Parse(time.RFC3339Nano, c.CreatedAt)
			if !since.IsZero() && t.Before(since) {
				continue
			}

			rest := strings.TrimSpace(c.Text[len("Friction:"):])
			skill := extractSkill(iss.Assignee)

			if rest == "none" {
				noneBySkill[skill]++
				continue
			}

			cat, msg := parseCategory(rest)
			entries = append(entries, Entry{
				IssueID:   iss.ID,
				Assignee:  iss.Assignee,
				Skill:     skill,
				Category:  cat,
				Message:   msg,
				CreatedAt: t,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("scanning %s: %w", path, err)
	}
	return entries, noneBySkill, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
