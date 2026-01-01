package network

import (
	"fmt"
	"time"
)

type Deadline struct {
	Read  time.Duration
	Write time.Duration
}

type RWDeadline interface {
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

func SetDeadline(conn RWDeadline, dd Deadline) error {
	if dd.Write != 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(dd.Write)); err != nil {
			return fmt.Errorf("set write deadline: %w", err)
		}

	}

	if dd.Read != 0 {
		if err := conn.SetReadDeadline(time.Now().Add(dd.Read)); err != nil {
			return fmt.Errorf("set read deadline: %w", err)
		}

	}

	return nil
}
