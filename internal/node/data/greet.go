package data

import (
	"github.com/labi-le/belphegor/internal/types"
	"io"
)

type Greet struct {
	*types.GreetMessage
}

func NewGreet(metadata *MetaData) *Greet {
	gp := Greet{&types.GreetMessage{}}
	if metadata != nil {
		gp.Device = metadata.Kind()
	}

	return &gp
}

func NewGreetFromReader(reader io.Reader) (*Greet, error) {
	gp := NewGreet(nil)

	if err := DecodeReader(reader, gp); err != nil {
		return gp, err
	}

	return gp, nil
}

func (g *Greet) MetaData() *MetaData {
	return MetaDataFromKind(g.Device)
}
