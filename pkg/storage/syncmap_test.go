package storage_test

import (
	"sync"
	"testing"

	"github.com/labi-le/belphegor/pkg/storage"
)

func TestSyncMap_AddGetExist(t *testing.T) {
	m := storage.NewSyncMapStorage[string, int]()

	if _, ok := m.Get("missing"); ok {
		t.Fatal("Get on empty map must be false")
	}
	if m.Exist("missing") {
		t.Fatal("Exist on empty map must be false")
	}

	m.Add("a", 1)
	v, ok := m.Get("a")
	if !ok || v != 1 {
		t.Fatalf("Get(a) = %d,%v want 1,true", v, ok)
	}
	if !m.Exist("a") {
		t.Fatal("Exist(a) must be true after Add")
	}
	if m.Len() != 1 {
		t.Fatalf("Len = %d, want 1", m.Len())
	}
}

func TestSyncMap_AddDuplicateKeepsFirstAndCount(t *testing.T) {
	m := storage.NewSyncMapStorage[string, int]()
	m.Add("k", 1)
	m.Add("k", 2) // LoadOrStore keeps the existing value; count must not grow

	if m.Len() != 1 {
		t.Fatalf("Len = %d, want 1 after duplicate Add", m.Len())
	}
	if v, _ := m.Get("k"); v != 1 {
		t.Fatalf("Get(k) = %d, want 1 (first write wins)", v)
	}
}

func TestSyncMap_Delete(t *testing.T) {
	m := storage.NewSyncMapStorage[string, int]()
	m.Add("a", 1)

	m.Delete("a")
	if _, ok := m.Get("a"); ok {
		t.Fatal("Get after Delete must be false")
	}
	if m.Len() != 0 {
		t.Fatalf("Len = %d, want 0 after Delete", m.Len())
	}

	m.Delete("missing") // no-op, count must stay
	if m.Len() != 0 {
		t.Fatalf("Len = %d, want 0 after deleting missing key", m.Len())
	}
}

func TestSyncMap_Tap(t *testing.T) {
	m := storage.NewSyncMapStorage[string, int]()
	want := map[string]int{"a": 1, "b": 2, "c": 3}
	for k, v := range want {
		m.Add(k, v)
	}

	got := map[string]int{}
	m.Tap(func(k string, v int) bool {
		got[k] = v
		return true
	})

	if len(got) != len(want) {
		t.Fatalf("Tap visited %d entries, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("Tap saw %s=%d, want %d", k, got[k], v)
		}
	}
}

func TestSyncMap_TapEarlyStop(t *testing.T) {
	m := storage.NewSyncMapStorage[string, int]()
	m.Add("a", 1)
	m.Add("b", 2)
	m.Add("c", 3)

	visited := 0
	m.Tap(func(string, int) bool {
		visited++
		return false // stop after the first
	})
	if visited != 1 {
		t.Fatalf("Tap visited %d entries, want 1 (early stop)", visited)
	}
}

func TestSyncMap_ConcurrentAddCountsEachKeyOnce(t *testing.T) {
	m := storage.NewSyncMapStorage[int, int]()
	const n = 100

	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			m.Add(i, i)
			m.Add(i, i) // duplicate; must not double-count
		})
	}
	wg.Wait()

	if m.Len() != n {
		t.Fatalf("Len = %d, want %d after concurrent adds", m.Len(), n)
	}
}
