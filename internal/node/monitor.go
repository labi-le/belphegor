package node

import (
	"bytes"
	"github.com/labi-le/belphegor/internal/types"
	"github.com/labi-le/belphegor/pkg/clipboard"
	"github.com/rs/zerolog/log"
	"time"
)

type Handler interface {
	Receive()
}

// ClipboardMonitor responsible for monitoring the local clipboard.
type ClipboardMonitor struct {
	cm           clipboard.Manager
	updateChan   Channel
	node         *Node
	scanInterval time.Duration
}

func NewClipboardMonitor(
	node *Node,
	cp clipboard.Manager,
	interval time.Duration,
	extUpdateChan Channel,
) *ClipboardMonitor {
	return &ClipboardMonitor{
		node:         node,
		cm:           cp,
		scanInterval: interval,
		updateChan:   extUpdateChan,
	}
}

// Receive starts monitoring the clipboard and subsequently sending data to other nodes
func (cm *ClipboardMonitor) Receive() {
	const op = "clipboardMonitor.Receive"
	var (
		clipboardChan    = make(chan *types.Message)
		currentClipboard = MessageFrom(cm.fetchLocalClipboard())
	)

	// first scan
	//clipboardChan <- cm.fetchLocalClipboard()

	defer close(clipboardChan)

	go func() {
		for {
			//log.Trace().Msg("scan local clipboard")
			select {
			case clip := <-cm.updateChan:
				currentClipboard = clip
			case <-time.After(cm.scanInterval):
				if newestClipboard := cm.fetchLocalClipboard(); !bytes.Equal(newestClipboard, currentClipboard.Data.Raw) {
					currentClipboard = MessageFrom(newestClipboard)
					clipboardChan <- currentClipboard
				}
			}
		}
	}()

	for clip := range clipboardChan {
		log.Trace().Str(op, "local clipboard data changed").Send()
		cm.node.Broadcast(clip, clip.Header.From)
	}
}

// fetchLocalClipboard returns the current value of the local clipboard.
func (cm *ClipboardMonitor) fetchLocalClipboard() []byte {
	clip, _ := cm.cm.Get()
	return clip
}
