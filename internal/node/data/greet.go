package data

import (
	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/pool"
)

var (
	greetPool = initGreetPool()
)

func initGreetPool() *pool.ObjectPool[*Greet] {
	p := pool.NewObjectPool[*Greet](10)
	p.New = func() *Greet {
		return NewGreetFromProto(&types.GreetMessage{
			Version: internal.Version,
		})
	}

	return p
}

type Greet struct {
	*types.GreetMessage
}

func NewGreet(metadata *types.Device) *Greet {
	gp := greetPool.Acquire()
	gp.Device = metadata

	return gp
}

func (m *Greet) Release() {
	greetPool.Release(m)
}

func NewGreetFromProto(m *types.GreetMessage) *Greet {
	return &Greet{GreetMessage: m}
}
