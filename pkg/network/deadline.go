package network

import (
	"time"
)

type Deadline struct {
	Read  time.Duration
	Write time.Duration
}
