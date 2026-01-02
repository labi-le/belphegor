package network

import (
	"fmt"
	"time"
)

type Deadline struct {
	Read  time.Duration
	Write time.Duration
}

type ReadDeadline interface {
	SetReadDeadline(time.Time) error
}

type WriteDeadline interface {
	SetWriteDeadline(time.Time) error
}

type RWDeadline interface {
	ReadDeadline
	WriteDeadline
}

func SetDeadline(conn RWDeadline, dd Deadline) error {
	if err := SetReadDeadline(conn, dd); err != nil {
		return err
	}

	if err := SetWriteDeadline(conn, dd); err != nil {
		return err
	}

	return nil
}

func SetReadDeadline(conn ReadDeadline, dd Deadline) error {
	if dd.Read != 0 {
		if err := conn.SetReadDeadline(time.Now().Add(dd.Read)); err != nil {
			return fmt.Errorf("set read deadline: %w", err)
		}

	}

	return nil
}

func SetWriteDeadline(conn WriteDeadline, dd Deadline) error {
	if dd.Write != 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(dd.Write)); err != nil {
			return fmt.Errorf("set write deadline: %w", err)
		}

	}

	return nil
}
