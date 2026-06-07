package eval

import (
	"testing"
	"time"
)

type testCase struct {
	Name     string
	Category string
}

func TestSample_Count(t *testing.T) {
	pool := []testCase{
		{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}, {Name: "e"},
	}
	got := Sample(pool, 3, 42, func(c testCase) string { return c.Name })
	if len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
}

func TestSample_CountExceedsPool(t *testing.T) {
	pool := []testCase{{Name: "a"}, {Name: "b"}}
	got := Sample(pool, 5, 42, func(c testCase) string { return c.Name })
	if len(got) != 2 {
		t.Errorf("len = %d, want 2 (pool size)", len(got))
	}
}

func TestSample_Reproducible(t *testing.T) {
	pool := make([]testCase, 20)
	for i := range pool {
		pool[i] = testCase{Name: string(rune('A' + i))}
	}
	got1 := Sample(pool, 8, 100, func(c testCase) string { return c.Name })
	got2 := Sample(pool, 8, 100, func(c testCase) string { return c.Name })

	for i := range got1 {
		if got1[i].Name != got2[i].Name {
			t.Errorf("same seed should produce same order: index %d differs", i)
		}
	}
}

func TestSample_DifferentSeeds(t *testing.T) {
	pool := make([]testCase, 20)
	for i := range pool {
		pool[i] = testCase{Name: string(rune('A' + i))}
	}
	got1 := Sample(pool, 8, 1, func(c testCase) string { return c.Name })
	got2 := Sample(pool, 8, 2, func(c testCase) string { return c.Name })

	same := true
	for i := range got1 {
		if got1[i].Name != got2[i].Name {
			same = false
			break
		}
	}
	if same {
		t.Error("different seeds should produce different results (probabilistically)")
	}
}

func TestISOWeekSeed(t *testing.T) {
	// Verify that ISOWeekSeed returns a deterministic value for the same week.
	now := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	seed1 := isoWeekSeed(now)
	seed2 := isoWeekSeed(now.Add(12 * time.Hour))
	if seed1 != seed2 {
		t.Errorf("same week should give same seed: %d vs %d", seed1, seed2)
	}

	// Different week → different seed (almost certainly)
	nextWeek := now.Add(7 * 24 * time.Hour)
	seed3 := isoWeekSeed(nextWeek)
	if seed1 == seed3 {
		t.Errorf("different weeks should give different seeds (usually): %d == %d", seed1, seed3)
	}
}
