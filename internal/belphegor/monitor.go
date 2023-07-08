package belphegor

import (
	"belphegor/pkg/clipboard"
	"bytes"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"time"
)

func monitorClipboard(node *Node, cp clipboard.Manager, delay time.Duration, externalUpdateChan <-chan []byte) {
	var (
		clipboardChan = make(chan []byte)
	)
	defer close(clipboardChan)

	localClipboard := fetchLocalClipboard(cp)

	go func() {
		for range time.Tick(delay * time.Second) {
			select {
			case clip := <-externalUpdateChan:
				if len(clip) > 0 {
					log.Trace().Msg("received external clipboard update")
					localClipboard = clip
				}
			default:
				newClipboard := fetchLocalClipboard(cp)
				log.Trace().Msgf("new cp: %d, old cp: %d", len(newClipboard), len(localClipboard))
				if !bytes.Equal(newClipboard, localClipboard) {
					localClipboard = newClipboard
					clipboardChan <- localClipboard
				}
			}
		}

	}()

	for clip := range clipboardChan {
		log.Trace().Msg("local clipboard data changed")
		node.Broadcast(NewMessage(clip))
	}
}

func handleClipboardData(node *Node, conn net.Conn, cp clipboard.Manager, externalUpdateChan chan []byte) {
	ip := NodeIP(conn.RemoteAddr().(*net.TCPAddr).IP.String())
	defer func() {
		log.Info().Msgf("close connection: %s", ip)
		node.storage.Delete(ip)
	}()
	for {
		msg := NewMessage(nil)

		err := decode(conn, msg)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok || opErr != io.EOF {
				return
			}

			log.Error().Msgf("failed to decode clipboard data: %s", err)
			break
		}

		node.lastMessage = msg

		cp.Set(msg.Data)

		externalUpdateChan <- msg.Data

		log.Debug().Msgf("received: %s from: %s", msg.Header.ID, ip)

		node.Broadcast(msg)
	}
}

func fetchLocalClipboard(c clipboard.Manager) []byte {
	clip, _ := c.Get()
	return clip
}
