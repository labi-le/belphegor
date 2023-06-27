package belphegor

import (
	"belphegor/pkg/clipboard"
	"bytes"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"time"
)

func monitorClipboard(node *Node, cp clipboard.Manager, delay time.Duration, externalUpdateChan chan []byte) {
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
					log.Debug().Msgf("received external clipboard update: %s", clip)
					localClipboard = clip
				}
			default:
				newClipboard := fetchLocalClipboard(cp)
				if !bytes.Equal(newClipboard, localClipboard) {
					localClipboard = newClipboard
					clipboardChan <- localClipboard
				}
			}
		}

	}()

	for clip := range clipboardChan {
		log.Debug().Msgf("local clipboard data changed: %s", clip)
		node.Broadcast(NewMessage(clip))
	}
}

func handleClipboardData(node *Node, conn net.Conn, cp clipboard.Manager, externalUpdateChan chan []byte) {
	for {
		var msg Message
		err := decode(conn, &msg)
		if err == io.EOF {
			log.Warn().Msgf("client %s is disconnected", conn.RemoteAddr().String())
			return
		}

		if err != nil {
			log.Error().Msgf("failed to decode clipboard data: %s", err)
			continue
		}

		node.lastMessage = msg

		cp.Set(msg.Data)

		externalUpdateChan <- msg.Data

		log.Debug().Msgf("received: %s from: %s", msg.Header.ID, conn.RemoteAddr().String())

		node.Broadcast(msg)
	}
}

func fetchLocalClipboard(c clipboard.Manager) []byte {
	clip, _ := c.Get()
	return clip
}
