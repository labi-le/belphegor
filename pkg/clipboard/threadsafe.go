package clipboard

import "sync"

type ThreadSafe struct {
	defaultManager Manager
	sync.RWMutex
}

func (t *ThreadSafe) Get() ([]byte, error) {
	t.RLock()
	defer t.RUnlock()

	return t.defaultManager.Get()
}

func (t *ThreadSafe) Set(data []byte) error {
	t.RLock()
	defer t.RUnlock()

	return t.defaultManager.Set(data)
}

func (t *ThreadSafe) Name() string {
	return t.defaultManager.Name()
}

func NewThreadSafe() *ThreadSafe {
	return &ThreadSafe{defaultManager: NewManager()}
}
