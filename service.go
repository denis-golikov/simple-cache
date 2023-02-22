package simple_cache

import (
	"context"
	"sync"
	"time"
)

const reindexBatchSize = 100

type Service struct {
	capacity      int
	entryTtl      uint64
	entries       []*entry
	index         map[string]int
	reindexCursor int
	timeService   TimeService
	lock          sync.RWMutex
}

func NewService(
	ctx context.Context,
	capacity int,
	entryTtl time.Duration,
	reindexInterval time.Duration,
	reindexDuration time.Duration,
	timeService TimeService,
) *Service {
	result := &Service{
		capacity:    capacity,
		entryTtl:    uint64(entryTtl.Milliseconds()),
		entries:     make([]*entry, capacity, capacity),
		index:       make(map[string]int, capacity),
		timeService: timeService,
		lock:        sync.RWMutex{},
	}

	go result.asyncReindex(ctx, reindexInterval, reindexDuration)

	return result
}

func (s *Service) Set(key string, value any) {
	s.lock.RLock()
	_, isKeyExist := s.index[key]
	s.lock.RUnlock()
	if isKeyExist {
		return
	}

	s.store(key, value)
}

func (s *Service) Get(key string) any {
	s.lock.RLock()
	i, isKeyExist := s.index[key]
	s.lock.RUnlock()
	if isKeyExist {
		s.lock.Lock()
		defer s.lock.Unlock()

		s.entries[i].weight++
		return s.entries[i].value
	}

	return nil
}

func (s *Service) store(key string, value any) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.entries[s.capacity-1] != nil {
		delete(s.index, s.entries[s.capacity-1].key)
	}

	entryForStore := &entry{
		key:    key,
		value:  value,
		weight: uint64(s.timeService.GetCurrentTimestamp()) + s.entryTtl,
	}

	i := 0
	for i < s.capacity {
		entry := s.entries[i]
		if entry == nil {
			s.entries[i] = entryForStore

			break
		}

		if entry.weight > entryForStore.weight {
			i++
			continue
		}

		for _, reindexEntry := range s.entries[i : s.capacity-1] {
			if reindexEntry == nil {
				break
			}
			s.index[reindexEntry.key]++
		}

		copy(s.entries[i+1:], s.entries[i:s.capacity-1])
		s.entries[i] = entryForStore

		break
	}

	s.index[key] = i
}

func (s *Service) asyncReindex(ctx context.Context, interval time.Duration, duration time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reindex(duration)
		}
	}
}

func (s *Service) reindex(duration time.Duration) {
	if s.reindexCursor == 0 {
		s.reindexCursor = s.capacity - 1
	}

	reindexUntil := time.Now().Add(duration)
	for s.reindexCursor > 0 {
		if s.reindexCursor%reindexBatchSize == 0 && time.Now().After(reindexUntil) {
			return
		}

		if s.entries[s.reindexCursor] == nil {
			s.reindexCursor--
			continue
		}

		currentIndex := s.reindexCursor
		beforeIndex := s.reindexCursor - 1
		if s.entries[beforeIndex].weight <= s.entries[currentIndex].weight {
			s.lock.Lock()
			s.index[s.entries[currentIndex].key] = beforeIndex
			s.index[s.entries[beforeIndex].key] = currentIndex
			s.entries[beforeIndex], s.entries[currentIndex] = s.entries[currentIndex], s.entries[beforeIndex]
			s.lock.Unlock()
		}
		s.reindexCursor--
	}
}
