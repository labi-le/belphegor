package clipboard

import (
	"github.com/labi-le/belphegor/pkg/clipboard/api"
	"sync"
)

type ThreadSafe struct {
	defaultManager api.Manager
	sync.RWMutex
}

func (t *ThreadSafe) Get() ([]byte, error) {
	t.RWMutex.Lock()
	defer t.RWMutex.Unlock()

	return t.defaultManager.Get()
}

func (t *ThreadSafe) Set(data []byte) error {
	t.RWMutex.Lock()
	defer t.RWMutex.Unlock()

	return t.defaultManager.Set(data)
}

func NewThreadSafe() *ThreadSafe {
	return &ThreadSafe{defaultManager: New()}
}
