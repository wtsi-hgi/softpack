package utils

import (
	"errors"
	"sync"
)

var ErrInvalidBuildTime = errors.New("cannot add build time, one or more invalid values passed")

type BuildTimes struct {
	mu sync.RWMutex

	sum   int64
	items int64
}

func (b *BuildTimes) AddBuildTime(start, end int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if start == 0 || end == 0 {
		return ErrInvalidBuildTime
	}

	b.items++
	b.sum += end - start

	return nil
}

func (b *BuildTimes) GetAverageBuildTime() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.items == 0 {
		return 0
	}

	return b.sum / b.items
}
