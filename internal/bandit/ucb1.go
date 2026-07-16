package bandit

import (
	"math"

	"github.com/jullss/banners-rotation/internal/domain"
)

type Chooser interface {
	Choose(stats []domain.Stat) int64
}

type UCB1 struct{}

func (UCB1) Choose(stats []domain.Stat) int64 {
	return Choose(stats)
}

func score(clicks, shows, totalShows int64) float64 {
	if shows == 0 {
		return math.Inf(1)
	}
	return float64(clicks)/float64(shows) + math.Sqrt(2*math.Log(float64(totalShows))/float64(shows))
}

func Choose(stats []domain.Stat) int64 {
	var bestID int64
	bestScore := math.Inf(-1)

	var totalShows int64
	for _, s := range stats {
		totalShows += s.Shows
	}

	for _, s := range stats {
		sc := score(s.Clicks, s.Shows, totalShows)
		if sc > bestScore {
			bestScore = sc
			bestID = s.BannerID
		}
	}
	return bestID
}
