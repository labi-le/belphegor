package id

import (
	"fmt"
	"net"
	"sync"

	"github.com/bwmarrin/snowflake"
	"github.com/cespare/xxhash"
)

type Unique = int64

var (
	MyID      = getNodeID()
	generator = new(idGenerator)
)

type idGenerator struct {
	node *snowflake.Node
	once sync.Once
}

func (g *idGenerator) nextID() int64 {
	g.once.Do(func() {
		node, err := snowflake.NewNode(MyID)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize snowflake node: %s", err))
		}
		g.node = node
	})
	return g.node.Generate().Int64()
}

func New() Unique {
	return generator.nextID()
}

func Mine(id Unique) bool {
	node := (id >> 12) & 0x3FF
	return node == MyID
}

func getNodeID() int64 {
	interfaces, err := net.Interfaces()
	if err != nil {
		return 1
	}

	for _, i := range interfaces {
		if (i.Flags&net.FlagUp) != 0 && i.HardwareAddr != nil && len(i.HardwareAddr) > 0 {
			h := xxhash.New()
			if _, err := h.Write(i.HardwareAddr); err != nil {
				panic(fmt.Sprintf("failed to generate node id: %s", err))
			}
			return int64(h.Sum64() % 1024)
		}
	}

	return 1 // fallback
}

func Author(id Unique) Unique {
	return (id >> 12) & 0x3FF
}
