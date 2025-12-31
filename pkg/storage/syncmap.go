package storage

import (
	"sync"
	"sync/atomic"
)

type SyncMap[key any, val any] struct {
	m     sync.Map
	count atomic.Int64
}

// NewSyncMapStorage creates a new SyncMap.
func NewSyncMapStorage[key any, val any]() *SyncMap[key, val] {
	return &SyncMap[key, val]{m: sync.Map{}}
}

func (s *SyncMap[key, val]) Add(k key, v val) {
	_, loaded := s.m.LoadOrStore(k, v)
	if !loaded {
		s.count.Add(1)
	}
}

func (s *SyncMap[key, val]) Delete(k key) {
	_, loaded := s.m.LoadAndDelete(k)
	if loaded {
		s.count.Add(-1)
	}
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
}

func (s *SyncMap[key, val]) Len() int {
	return int(s.count.Load())
}
