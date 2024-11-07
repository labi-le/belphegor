package node

import "github.com/labi-le/belphegor/internal/types/domain"

// Channel is an interface for managing clipboard data.
type Channel chan *domain.Message
