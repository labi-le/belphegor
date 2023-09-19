package clipboard

import "sync"

type ts struct {
	defaultManager Manager
	sync.RWMutex
}

func (t *ts) Get() ([]byte, error) {
	t.RLock()
	defer t.RUnlock()

	return t.defaultManager.Get()
}

func (t *ts) Set(data []byte) error {
	t.RLock()
	defer t.RUnlock()

	return t.defaultManager.Set(data)
}

func NewThreadSafe() Manager {
	return &ts{defaultManager: NewManager()}
}
