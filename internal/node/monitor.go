package node

import (
	"bytes"
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

// NewClipboardMonitor creates a new instance of ClipboardMonitor.
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
	var (
		clipboardChan    = make(chan []byte)
		currentClipboard []byte
	)

	// first scan
	//clipboardChan <- cm.fetchLocalClipboard()

	defer close(clipboardChan)

	go func() {
		for {
			//log.Trace().Msg("scan local clipboard")
			select {
			case clip := <-cm.updateChan:
				if len(clip) > 0 {
					log.Trace().Str("clipboardMonitor.Receive", "received external clipboard update").Send()
					currentClipboard = clip
				}
			case <-time.After(cm.scanInterval):
				if newestClipboard := cm.fetchLocalClipboard(); !bytes.Equal(newestClipboard, currentClipboard) {
					currentClipboard = newestClipboard
					clipboardChan <- currentClipboard
				}
			}
		}
	}()

	for clip := range clipboardChan {
		log.Trace().Str("clipboardMonitor.Receive", "received external clipboard update").Send()
		cm.node.Broadcast(MessageFrom(clip), "")
	}
}

// fetchLocalClipboard returns the current value of the local clipboard.
func (cm *ClipboardMonitor) fetchLocalClipboard() []byte {
	clip, _ := cm.cm.Get()
	return clip
}
