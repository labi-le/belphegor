package belphegor

import (
	"belphegor/pkg/clipboard"
	"bytes"
	"github.com/rs/zerolog/log"
	"time"
)

type Handler interface {
	Start()
}

// ClipboardMonitor отвечает за мониторинг локального буфера обмена.
type ClipboardMonitor struct {
	cm           clipboard.Manager
	updateChan   Channel
	node         *Node
	scanInterval time.Duration
}

// NewClipboardMonitor создает новый экземпляр ClipboardMonitor.
func NewClipboardMonitor(node *Node, cp clipboard.Manager, interval time.Duration, extUpdateChan Channel) Handler {
	return &ClipboardMonitor{
		node:         node,
		cm:           cp,
		scanInterval: interval,
		updateChan:   extUpdateChan,
	}
}

// Start начинает мониторинг буфера обмена.
func (cm *ClipboardMonitor) Start() {
	var (
		clipboardChan    = make(chan []byte)
		currentClipboard []byte
	)

	defer close(clipboardChan)

	go func() {
		for range time.Tick(cm.scanInterval) {
			log.Trace().Msg("scan local clipboard")
			select {
			case clip := <-cm.updateChan.Get():
				if len(clip) > 0 {
					log.Trace().Msg("received external clipboard update")
					currentClipboard = clip
				}
			default:
				newestClipboard := cm.fetchLocalClipboard()
				if !bytes.Equal(newestClipboard, currentClipboard) {
					currentClipboard = newestClipboard
					clipboardChan <- currentClipboard
				}
			}
		}
	}()

	for clip := range clipboardChan {
		log.Trace().Msg("local clipboard data changed")
		cm.node.Broadcast(NewMessage(clip))
	}
}

// fetchLocalClipboard возвращает текущее значение локального буфера обмена.
func (cm *ClipboardMonitor) fetchLocalClipboard() []byte {
	clip, _ := cm.cm.Get()
	return clip
}
