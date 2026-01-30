package domain

import (
	"strconv"

	"github.com/labi-le/belphegor/pkg/id"
	"github.com/rs/zerolog"
)

type NodeID id.Unique

func NewNodeID() NodeID {
	return NodeID(id.New())
}

func (n NodeID) String() string {
	return strconv.FormatInt(n.Int64(), 10)
}

func (n NodeID) Int64() int64 {
	return int64(n)
}

func (n NodeID) Zero() bool {
	return n == 0
}

func (n NodeID) MarshalZerologObject(e *zerolog.Event) {
	e.Int64("node_id", n.Int64())
}
