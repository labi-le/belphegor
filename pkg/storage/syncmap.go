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
		var zeroVal val
		return zeroVal, false
	}
	return v.(val), true
}

func (s *SyncMap[key, val]) Exist(k key) bool {
	_, ok := s.m.Load(k)
	return ok
}

func (s *SyncMap[key, val]) Tap(fn func(key, val) bool) {
	var wg sync.WaitGroup
	s.m.Range(func(k, v any) bool {
		typedKey, okKey := k.(key)
		if !okKey {
			panic("unexpected key type in SyncMap during Tap")
		}
		typedVal, okVal := v.(val)
		if !okVal {
			panic("unexpected value type in SyncMap during Tap")
		}

		return fn(typedKey, typedVal)
	})

	wg.Wait()
}
