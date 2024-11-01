package node

import (
	"github.com/labi-le/belphegor/internal/node/data"
	"github.com/labi-le/belphegor/pkg/storage"
)

type Storage = storage.SyncMap[data.UniqueID, *Peer]
