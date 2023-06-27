package belphegor

import (
	"belphegor/pkg/clipboard"
	"belphegor/pkg/ip"
	"bytes"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
	"io"
	"net"
	"time"
)

type Header struct {
	ID   uuid.UUID
	From string
}

type Message struct {
	Header Header
	Data   []byte
}

func (m Message) Write(w io.Writer) (int, error) {
	return w.Write(encode(m))
}

func NewMessage(data []byte, addr string) Message {
	return Message{Data: data, Header: Header{
		ID:   uuid.New(),
		From: addr,
	}}
}

func (m Message) IsDuplicate(msg Message) bool {
	return m.Header.ID == msg.Header.ID && m.Header.From == msg.Header.From && bytes.Equal(m.Data, msg.Data)
}

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
		node.Broadcast(NewMessage(clip, ip.GetOutboundIP()))
	}
}

func handleClipboardData(node *Node, conn net.Conn, cp clipboard.Manager, externalUpdateChan chan []byte) {
	for {
		var msg Message
		err := decode(conn, &msg)
		if err == io.EOF {
			log.Warn().Msgf("client %s is disconnected", conn.RemoteAddr().String())
			node.Close(conn)

			return
		}

		if err != nil {
			log.Error().Msgf("failed to decode clipboard data: %s", err)
			continue
		}

		node.lastMessage = msg

		cp.Set(msg.Data)

		externalUpdateChan <- msg.Data

		log.Debug().Msgf("received: %s from: %s", msg.Header.From, conn.RemoteAddr().String())

		node.Broadcast(msg)
	}
}

func fetchLocalClipboard(c clipboard.Manager) []byte {
	clip, _ := c.Get()
	return clip
}

func encode(src interface{}) []byte {
	encoded, err := msgpack.Marshal(src)
	if err != nil {
		log.Error().Msgf("failed to encode clipboard data: %s", err)
	}

	return encoded
}

func decode(r io.Reader, dst interface{}) error {
	return msgpack.NewDecoder(r).Decode(dst)
}
