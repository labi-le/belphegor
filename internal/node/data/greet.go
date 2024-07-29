package data

import (
	"github.com/labi-le/belphegor/internal"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/pool"
	"io"
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

func NewGreet(metadata *MetaData) *Greet {
	gp := greetPool.Acquire()
	gp.Device = metadata.Kind()

	return gp
}

func NewGreetFromReader(reader io.Reader) (*Greet, error) {
	gp := greetPool.Acquire()

	if err := DecodeReader(reader, gp); err != nil {
		return gp, err
	}

	return gp, nil
}

func (g *Greet) Release() {
	greetPool.Release(g)
}

func NewGreetFromProto(m *types.GreetMessage) *Greet {
	return &Greet{GreetMessage: m}
}

func (g *Greet) MetaData() *MetaData {
	return MetaDataFromKind(g.Device)
}
