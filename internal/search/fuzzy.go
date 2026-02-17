package search

import (
	"strings"
	"unicode"
)

type Match struct {
	Score int
	Index int
}

func Fuzzy(query string, targets []string, usageScores []int) []Match {
	if query == "" {
		matches := make([]Match, len(targets))
		for i := range targets {
			matches[i] = Match{Score: usageScores[i] * 100, Index: i}
		}
		sortByScore(matches)
		return matches
	}

	query = strings.ToLower(query)
	var matches []Match

	// Minimum score threshold to filter garbage matches.
	// A prefix match scores 5000+, a contains match scores 2000+,
	// so 50 means we at least need reasonable subsequence hits.
	minScore := 50

	for i, target := range targets {
		matchScore := score(query, strings.ToLower(target))
		if matchScore >= minScore {
			matchScore += usageScores[i] * 100
			matches = append(matches, Match{Score: matchScore, Index: i})
		}
	}

	sortByScore(matches)
	return matches
}

func score(query, target string) int {
	if target == query {
		return 10000
	}
	if strings.HasPrefix(target, query) {
		return 5000 + len(query)*10
	}
	if strings.Contains(target, query) {
		return 2000 + len(query)*5
	}

	queryIndex := 0
	consecutive := 0
	maxConsecutive := 0
	total := 0
	wordBoundaryBonus := 0
	firstCharBonus := 0

	for targetIndex := 0; targetIndex < len(target) && queryIndex < len(query); targetIndex++ {
		if target[targetIndex] == query[queryIndex] {
			total += 10
			consecutive++
			if consecutive > maxConsecutive {
				maxConsecutive = consecutive
			}
			if queryIndex == 0 && targetIndex == 0 {
				firstCharBonus = 50
			}
			if targetIndex > 0 && isWordBoundary(rune(target[targetIndex-1])) {
				wordBoundaryBonus += 25
			}
			queryIndex++
		} else {
			consecutive = 0
		}
	}

	if queryIndex < len(query) {
		return 0
	}

	return total + maxConsecutive*30 + wordBoundaryBonus + firstCharBonus
}

func isWordBoundary(r rune) bool {
	return r == ' ' || r == '-' || r == '_' || r == '.' || r == '/' || r == '\\' || unicode.IsUpper(r)
}

func sortByScore(matches []Match) {
	for i := 1; i < len(matches); i++ {
		current := i
		for current > 0 && matches[current].Score > matches[current-1].Score {
			matches[current], matches[current-1] = matches[current-1], matches[current]
			current--
		}
	}
}
