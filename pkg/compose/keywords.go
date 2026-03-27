package compose

import (
	"regexp"
	"sort"
	"strings"
)

// Common English stopwords plus markdown/code artifacts to filter out.
var stopwords = map[string]bool{
	// Articles and determiners
	"a": true, "an": true, "the": true, "this": true, "that": true, "these": true, "those": true,
	// Pronouns
	"i": true, "you": true, "he": true, "she": true, "it": true, "we": true, "they": true,
	"me": true, "him": true, "her": true, "us": true, "them": true, "my": true, "your": true,
	"his": true, "its": true, "our": true, "their": true, "mine": true, "yours": true,
	// Prepositions
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true, "with": true,
	"by": true, "from": true, "about": true, "into": true, "through": true, "during": true,
	"before": true, "after": true, "above": true, "below": true, "between": true, "under": true,
	"over": true, "against": true, "without": true, "within": true,
	// Conjunctions
	"and": true, "but": true, "or": true, "nor": true, "so": true, "yet": true, "both": true,
	// Verbs (common auxiliary/linking)
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "shall": true, "should": true, "may": true, "might": true,
	"can": true, "could": true, "must": true,
	// Common verbs
	"get": true, "got": true, "make": true, "made": true, "go": true, "went": true, "come": true,
	"take": true, "give": true, "say": true, "said": true, "know": true, "see": true, "think": true,
	"just": true, "also": true, "like": true,
	// Adverbs and others
	"not": true, "no": true, "yes": true, "very": true, "too": true, "more": true, "most": true,
	"only": true, "even": true, "still": true, "already": true, "here": true, "there": true,
	"when": true, "where": true, "how": true, "what": true, "which": true, "who": true, "why": true,
	"if": true, "then": true, "than": true, "because": true, "as": true, "while": true, "each": true,
	"all": true, "any": true, "some": true, "every": true, "much": true, "many": true, "such": true,
	"other": true, "same": true, "own": true, "well": true, "now": true,
	// Brief-specific noise
	"brief": true, "briefs": true, "question": true, "answer": true, "first": true, "last": true,
	"new": true, "one": true, "two": true, "three": true, "four": true, "five": true,
	"doesn": true, "don": true, "didn": true, "isn": true, "wasn": true, "aren": true,
	"won": true, "wouldn": true, "couldn": true, "shouldn": true, "hadn": true, "hasn": true,
	// Common reasoning/discussion vocabulary
	"actually": true, "real": true, "really": true, "instead": true, "means": true,
	"never": true, "way": true, "thing": true, "things": true, "put": true,
	"right": true, "wrong": true, "point": true, "part": true, "different": true,
	"turn": true, "found": true, "look": true, "looking": true, "needed": true,
	"needs": true, "enough": true, "let": true, "want": true, "wanted": true,
	"start": true, "started": true, "end": true, "try": true, "tried": true,
	"use": true, "used": true, "using": true, "show": true, "shows": true,
	"change": true, "changed": true, "changes": true, "run": true, "running": true,
	"runs": true, "read": true, "write": true, "set": true, "call": true,
	"called": true, "back": true, "long": true, "small": true, "big": true,
	"good": true, "bad": true, "fine": true, "hard": true, "easy": true,
	"either": true, "nothing": true, "something": true, "everything": true,
	"anywhere": true, "keep": true, "left": true, "another": true,
	"single": true, "whole": true, "full": true, "exist": true, "exists": true,
	"find": true, "contain": true, "contains": true, "require": true, "requires": true,
	"work": true, "working": true, "works": true, "fix": true, "fixed": true,
	"open": true, "check": true, "checked": true, "create": true, "created": true,
	"add": true, "added": true, "adding": true, "itself": true,
	"context": true, "across": true, "worth": true, "line": true, "lines": true,
	"pass": true, "passes": true, "passed": true, "fail": true, "fails": true,
	"test": true, "tests": true, "testing": true,
	"file": true, "files": true, "path": true, "code": true, "build": true,
	"time": true, "problem": true, "reason": true, "case": true,
	"sure": true, "possible": true, "clear": true, "clean": true,
}

var wordRe = regexp.MustCompile(`[a-z]+(?:-[a-z]+)*`)

// ExtractKeywords pulls significant words from text, lowercased and deduplicated.
// Words shorter than 3 characters and stopwords are excluded.
func ExtractKeywords(text string) []string {
	lower := strings.ToLower(text)
	words := wordRe.FindAllString(lower, -1)

	seen := make(map[string]bool)
	var keywords []string
	for _, w := range words {
		if len(w) < 3 || stopwords[w] || seen[w] {
			continue
		}
		seen[w] = true
		keywords = append(keywords, w)
	}

	sort.Strings(keywords)
	return keywords
}

// KeywordOverlap returns the keywords shared between two sets.
func KeywordOverlap(a, b []string) []string {
	set := make(map[string]bool, len(a))
	for _, w := range a {
		set[w] = true
	}

	var overlap []string
	seen := make(map[string]bool)
	for _, w := range b {
		if set[w] && !seen[w] {
			overlap = append(overlap, w)
			seen[w] = true
		}
	}

	sort.Strings(overlap)
	return overlap
}

// WeightedKeywords returns keywords sorted by frequency (TF-like weighting).
// Words appearing more often in the text rank higher.
func WeightedKeywords(text string) []KeywordWeight {
	lower := strings.ToLower(text)
	words := wordRe.FindAllString(lower, -1)

	freq := make(map[string]int)
	for _, w := range words {
		if len(w) < 3 || stopwords[w] {
			continue
		}
		freq[w]++
	}

	var result []KeywordWeight
	for w, count := range freq {
		result = append(result, KeywordWeight{Word: w, Count: count})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Word < result[j].Word
	})

	return result
}

// KeywordWeight pairs a keyword with its frequency count.
type KeywordWeight struct {
	Word  string
	Count int
}

// MaxKeywordsPerBrief caps keywords per brief after mid-band filtering.
const MaxKeywordsPerBrief = 25

// FilterCommonKeywords applies mid-band document frequency filtering:
// removes words that appear in too many briefs (ubiquitous) AND words
// that appear in only one brief (too unique to cluster on). The remaining
// "mid-band" words are shared by some briefs but not all — exactly the
// vocabulary that reveals meaningful clusters.
func FilterCommonKeywords(briefs []*Brief, maxFreq float64) {
	// Only apply DF filtering with enough briefs for frequency to be meaningful.
	if len(briefs) < 10 {
		return
	}

	// Count document frequency (how many briefs contain each keyword)
	docFreq := make(map[string]int)
	for _, b := range briefs {
		seen := make(map[string]bool)
		for _, kw := range b.Keywords {
			if !seen[kw] {
				docFreq[kw]++
				seen[kw] = true
			}
		}
	}

	maxDF := int(float64(len(briefs)) * maxFreq)
	minDF := 2 // must appear in at least 2 briefs to be useful for clustering

	// Build set of mid-band keywords
	midBand := make(map[string]bool)
	for kw, count := range docFreq {
		if count >= minDF && count <= maxDF {
			midBand[kw] = true
		}
	}

	// Filter each brief's keywords to mid-band only, cap at MaxKeywordsPerBrief
	for _, b := range briefs {
		var filtered []string
		for _, kw := range b.Keywords {
			if midBand[kw] {
				filtered = append(filtered, kw)
			}
		}
		if len(filtered) > MaxKeywordsPerBrief {
			filtered = filtered[:MaxKeywordsPerBrief]
		}
		b.Keywords = filtered
	}
}
