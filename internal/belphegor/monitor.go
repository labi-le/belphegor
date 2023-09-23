package belphegor

import (
	"belphegor/pkg/clipboard"
	"bytes"
	"errors"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"time"
)

func monitorClipboard(node *Node, cp clipboard.Manager, delay time.Duration, externalUpdateChan Channel) {
	var (
		clipboardChan = make(chan []byte)
		// New node will always send its clipboard
		currentClipboard []byte
	)
	defer close(clipboardChan)

	go func() {
		for range time.Tick(delay) {
			log.Trace().Msg("scan local clipboard")
			select {
			case clip := <-externalUpdateChan.Get():
				if len(clip) > 0 {
					log.Trace().Msg("received external clipboard update")
					currentClipboard = clip
				}
			default:
				newestClipboard := fetchLocalClipboard(cp)
				if !bytes.Equal(newestClipboard, currentClipboard) {
					currentClipboard = newestClipboard
					clipboardChan <- currentClipboard
				}
			}
		}

	}()

	for clip := range clipboardChan {
		log.Trace().Msg("local clipboard data changed")
		node.Broadcast(NewMessage(clip))
	}
}

func receiveDataFromNode(node *Node, conn net.Conn, cp clipboard.Manager, localClipboard Channel) {
	remoteIP := IP(conn.RemoteAddr().(*net.TCPAddr).IP.String())
	defer func() {
		log.Info().Msgf("close connection: %s", remoteIP)
		node.storage.Delete(remoteIP)
	}()
	for {
		msg := NewMessage(nil)

		err := decode(conn, msg)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Trace().Msg("connection closed")
				return
			}
			var opErr *net.OpError
			if errors.As(err, &opErr) {
				log.Trace().Err(opErr).Msg("connection closed")
				return
			}

			log.Error().Msgf("failed to decode clipboard data: %s", err)
			break
		}

		node.lastMessage = *msg

		_ = cp.Set(msg.Data.Raw)

		localClipboard.Set(msg.Data.Raw)

		log.Debug().Msgf("received: %s from: %s", msg.Header.ID, remoteIP)

		node.Broadcast(msg, remoteIP)
	}
}

func fetchLocalClipboard(c clipboard.Manager) []byte {
	clip, _ := c.Get()
	return clip
}
