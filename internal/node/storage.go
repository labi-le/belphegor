package node

import (
	"github.com/labi-le/belphegor/pkg/id"
	"github.com/labi-le/belphegor/pkg/storage"
)

type Storage = storage.SyncMap[id.Unique, *Peer]
