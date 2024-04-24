package node

import (
	"github.com/labi-le/belphegor/internal/node/data"
)

// Channel is an interface for managing clipboard data.
type Channel chan *data.Message
