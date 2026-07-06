package bandit

import (
	"testing"

	"github.com/jullss/banners-rotation/internal/domain"
)

func TestChoose_UnshownBannerFirst(t *testing.T) {
	stats := []domain.Stat{
		{BannerID: 1, Shows: 10, Clicks: 5},
		{BannerID: 2, Shows: 0, Clicks: 0},
		{BannerID: 3, Shows: 8, Clicks: 3},
	}
	got := Choose(stats)
	if got != 2 {
		t.Errorf("expected banner 2 (unshown), got %d", got)
	}
}

func TestChoose_PopularBannerWins(t *testing.T) {
	stats := []domain.Stat{
		{BannerID: 1, Shows: 100, Clicks: 1},
		{BannerID: 2, Shows: 100, Clicks: 80},
		{BannerID: 3, Shows: 100, Clicks: 2},
	}

	counts := make(map[int64]int)
	for range 1000 {
		id := Choose(stats)
		counts[id]++
		stats[id-1].Shows++
		if id == 2 {
			stats[id-1].Clicks++
		}
	}

	if counts[2] <= counts[1] || counts[2] <= counts[3] {
		t.Errorf("expected banner 2 to be chosen most often, got counts: %v", counts)
	}
}

func TestChoose_AllEqual_EachShownAtLeastOnce(t *testing.T) {
	stats := []domain.Stat{
		{BannerID: 1, Shows: 0, Clicks: 0},
		{BannerID: 2, Shows: 0, Clicks: 0},
		{BannerID: 3, Shows: 0, Clicks: 0},
	}

	seen := make(map[int64]bool)
	for range 30 {
		id := Choose(stats)
		seen[id] = true
		for i := range stats {
			if stats[i].BannerID == id {
				stats[i].Shows++
				break
			}
		}
	}

	for _, s := range stats {
		if !seen[s.BannerID] {
			t.Errorf("banner %d was never chosen", s.BannerID)
		}
	}
}
