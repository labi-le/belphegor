package node

import (
	"github.com/labi-le/belphegor/internal/peer"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/storage"
)

type Storage = storage.SyncMap[domain.NodeID, *peer.Peer]
