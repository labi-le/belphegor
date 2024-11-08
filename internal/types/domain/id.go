package domain

import (
	"github.com/bwmarrin/snowflake"
	"github.com/rs/zerolog/log"
	"sync"
)

type UniqueID = int64

type idGenerator struct {
	node *snowflake.Node
	once sync.Once
}

var generator = &idGenerator{}

func (g *idGenerator) nextID() int64 {
	g.once.Do(func() {
		node, err := snowflake.NewNode(1)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to initialize snowflake node")
		}
		g.node = node
	})
	return g.node.Generate().Int64()
}

func NewID() UniqueID {
	return generator.nextID()
}
