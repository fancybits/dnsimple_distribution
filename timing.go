package main

import (
	"sort"
	"time"
)

type Timing struct {
	StartAt  time.Time     `json:"start_at"`
	Duration time.Duration `json:"duration"`
}

func NewTiming() *Timing {
	return &Timing{
		StartAt: time.Now(),
	}
}

type Timings []*Timing

func (t Timings) Median() time.Duration {
	if len(t) == 0 {
		return 0
	}

	if len(t) == 1 {
		return t[0].Duration
	}

	sorted := make(Timings, len(t))
	copy(sorted, t)
	// Sort the timings slice
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Duration < sorted[j].Duration
	})

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1].Duration + sorted[mid].Duration) / 2
	}
	return sorted[mid].Duration
}

func (t Timings) Mean() time.Duration {
	if len(t) == 0 {
		return 0
	}

	var total time.Duration
	for _, timing := range t {
		total += timing.Duration
	}
	return total / time.Duration(len(t))
}

func (t *Timing) Stop() *Timing {
	t.Duration = time.Since(t.StartAt)
	return t
}
