package notification

import (
	"fmt"
	"github.com/gen2brain/beeep"
	"github.com/rs/zerolog/log"
)

type Notifier interface {
	Notify(message string, v ...any)
}

type BeepDecorator struct {
	Title string
}

func (b BeepDecorator) Notify(message string, v ...any) {
	if err := beeep.Notify(b.Title, fmt.Sprintf(message, v...), ""); err != nil {
		log.Err(err).Msg("failed to send system notification")
	}
}

type NullNotifier struct{}

func (n NullNotifier) Notify(string, ...any) {}
