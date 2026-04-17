package search

import "sort"

// DefaultCaps returns the standard per-category result limits used by the ranking pass.
func DefaultCaps() map[string]int {
	return map[string]int{
		"Commands":     6,
		"Applications": 8,
		"Files":        6,
		"Folders":      4,
		"Clipboard":    6,
		"System":       5,
		"Web":          1,
	}
}

// Scored wraps any value with a relevance score and a category label.
type Scored[T any] struct {
	Item  T
	Score int
	Cat   string
}

// RankAndCap sorts a slice of Scored items by descending score, then caps each
// category to the limit defined in caps. Items whose category is not in caps
// are passed through uncapped (use this for special categories like Calculator).
func RankAndCap[T any](items []Scored[T], caps map[string]int) []T {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})
	counts := make(map[string]int, len(caps))
	out := make([]T, 0, len(items))
	for _, s := range items {
		if cap, hasCap := caps[s.Cat]; hasCap {
			if counts[s.Cat] >= cap {
				continue
			}
		}
		counts[s.Cat]++
		out = append(out, s.Item)
	}
	return out
}
