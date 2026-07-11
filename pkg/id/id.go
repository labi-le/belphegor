package id

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/bwmarrin/snowflake"
	"github.com/cespare/xxhash"
)

type Unique = int64

const (
	nodeIDBits  = 10
	nodeIDShift = 12
	nodeIDMask  = 1<<nodeIDBits - 1
	nodeIDCount = 1 << nodeIDBits
)

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
	return Author(id) == MyID
}

func getNodeID() int64 {
	if v, ok := os.LookupEnv("BELPHEGOR_NODE_ID"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			nid := int64(n) % nodeIDCount
			if nid < 0 {
				nid += nodeIDCount
			}
			return nid
		}
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return 1
	}

	for _, i := range interfaces {
		if (i.Flags&net.FlagUp) != 0 && i.HardwareAddr != nil && len(i.HardwareAddr) > 0 {
			h := xxhash.New()
			if _, writeErr := h.Write(i.HardwareAddr); writeErr != nil {
				panic(fmt.Sprintf("failed to generate node id: %s", writeErr))
			}
			return int64(h.Sum64() % nodeIDCount)
		}
	}

	return 1 // fallback
}

func Author(id Unique) Unique {
	return (id >> nodeIDShift) & nodeIDMask
}
