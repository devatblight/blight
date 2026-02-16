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
		s := score(query, strings.ToLower(target))
		if s >= minScore {
			s += usageScores[i] * 100
			matches = append(matches, Match{Score: s, Index: i})
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

	qi := 0
	consecutive := 0
	maxConsecutive := 0
	total := 0
	wordBoundaryBonus := 0
	firstCharBonus := 0

	for ti := 0; ti < len(target) && qi < len(query); ti++ {
		if target[ti] == query[qi] {
			total += 10
			consecutive++
			if consecutive > maxConsecutive {
				maxConsecutive = consecutive
			}
			if qi == 0 && ti == 0 {
				firstCharBonus = 50
			}
			if ti > 0 && isWordBoundary(rune(target[ti-1])) {
				wordBoundaryBonus += 25
			}
			qi++
		} else {
			consecutive = 0
		}
	}

	if qi < len(query) {
		return 0
	}

	return total + maxConsecutive*30 + wordBoundaryBonus + firstCharBonus
}

func isWordBoundary(r rune) bool {
	return r == ' ' || r == '-' || r == '_' || r == '.' || r == '/' || r == '\\' || unicode.IsUpper(r)
}

func sortByScore(matches []Match) {
	for i := 1; i < len(matches); i++ {
		j := i
		for j > 0 && matches[j].Score > matches[j-1].Score {
			matches[j], matches[j-1] = matches[j-1], matches[j]
			j--
		}
	}
}
