package search

import (
	"testing"
)

func TestDefaultCaps(t *testing.T) {
	caps := DefaultCaps()
	cases := []struct {
		cat string
		min int
	}{
		{"Commands", 1},
		{"Applications", 1},
		{"Files", 1},
		{"Folders", 1},
		{"Clipboard", 1},
		{"System", 1},
		{"Web", 1},
	}
	for _, tc := range cases {
		if caps[tc.cat] < tc.min {
			t.Errorf("DefaultCaps[%q] = %d, want >= %d", tc.cat, caps[tc.cat], tc.min)
		}
	}
}

func TestRankAndCap_SortsByScore(t *testing.T) {
	items := []Scored[string]{
		{Item: "low", Score: 100, Cat: "A"},
		{Item: "high", Score: 9000, Cat: "A"},
		{Item: "mid", Score: 500, Cat: "A"},
	}
	caps := map[string]int{"A": 10}
	got := RankAndCap(items, caps)
	if len(got) != 3 {
		t.Fatalf("want 3 results, got %d", len(got))
	}
	if got[0] != "high" || got[1] != "mid" || got[2] != "low" {
		t.Errorf("wrong order: %v", got)
	}
}

func TestRankAndCap_AppliesCap(t *testing.T) {
	items := []Scored[string]{
		{Item: "a1", Score: 9000, Cat: "Apps"},
		{Item: "a2", Score: 8000, Cat: "Apps"},
		{Item: "a3", Score: 7000, Cat: "Apps"},
		{Item: "f1", Score: 6000, Cat: "Files"},
	}
	caps := map[string]int{"Apps": 2, "Files": 5}
	got := RankAndCap(items, caps)
	if len(got) != 3 {
		t.Fatalf("want 3 results (2 Apps + 1 Files), got %d", len(got))
	}
	if got[0] != "a1" || got[1] != "a2" || got[2] != "f1" {
		t.Errorf("wrong results: %v", got)
	}
}

func TestRankAndCap_UncappedCategory(t *testing.T) {
	items := []Scored[string]{
		{Item: "calc", Score: 5000, Cat: "Calculator"},
		{Item: "app", Score: 4000, Cat: "Applications"},
	}
	caps := map[string]int{"Applications": 1}
	// Calculator is not in caps — should always pass through
	got := RankAndCap(items, caps)
	if len(got) != 2 {
		t.Fatalf("want 2 results, got %d", len(got))
	}
}

func TestRankAndCap_CrossCategoryOrdering(t *testing.T) {
	// A highly relevant file should outrank a low-relevance app
	items := []Scored[string]{
		{Item: "relevant-file", Score: 8000, Cat: "Files"},
		{Item: "weak-app", Score: 200, Cat: "Applications"},
	}
	caps := map[string]int{"Files": 5, "Applications": 5}
	got := RankAndCap(items, caps)
	if len(got) != 2 {
		t.Fatalf("want 2 results, got %d", len(got))
	}
	if got[0] != "relevant-file" {
		t.Errorf("expected file to rank first, got %v", got[0])
	}
}

func TestFuzzyScoreExactBeatsPrefix(t *testing.T) {
	exact := score("chrome", "chrome")
	prefix := score("chrome", "chromebook")
	if exact <= prefix {
		t.Errorf("exact match (%d) should beat prefix match (%d)", exact, prefix)
	}
}

func TestFuzzyScorePrefixBeatsSubstring(t *testing.T) {
	prefix := score("not", "notepad")
	sub := score("not", "windows notifier")
	if prefix <= sub {
		t.Errorf("prefix match (%d) should beat substring match (%d)", prefix, sub)
	}
}
