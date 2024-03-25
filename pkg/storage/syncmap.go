package storage

import (
	"sync"
)

type SyncMap[key any, val any] struct {
	m sync.Map
}

// NewSyncMapStorage creates a new SyncMap.
func NewSyncMapStorage[key any, val any]() *SyncMap[key, val] {
	return &SyncMap[key, val]{m: sync.Map{}}
}

func (s *SyncMap[key, val]) Add(k key, v val) {
	s.m.Store(k, v)
}

func (s *SyncMap[key, val]) Delete(k key) {
	s.m.Delete(k)
}

func (s *SyncMap[key, val]) Get(k key) (val, bool) {
	v, ok := s.m.Load(k)
	if !ok {
		return v.(val), false
	}
	return v.(val), true
}

func (s *SyncMap[key, val]) Exist(k key) bool {
	_, ok := s.m.Load(k)
	return ok
}

func (s *SyncMap[key, val]) Tap(fn func(key, val)) {
	var wg sync.WaitGroup
	s.m.Range(func(k, v any) bool {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn(k.(key), v.(val))
		}()
		return true
	})

	wg.Wait()
}
