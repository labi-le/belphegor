package notification

import (
	"fmt"

	"github.com/gen2brain/beeep"
	"github.com/labi-le/belphegor/internal/console/icon"
	"github.com/rs/zerolog/log"
)

type Notifier interface {
	Notify(message string, v ...any)
}

type BeepDecorator struct {
	Title string
}

func (b BeepDecorator) Notify(message string, v ...any) {
	if err := beeep.Notify(b.Title, fmt.Sprintf(message, v...), icon.FullSize); err != nil {
		log.Err(err).Msg("failed to send system notification")
	}
}

func New(enable bool) Notifier {
	if enable {
		const name = "belphegor"

		beeep.AppName = name
		return BeepDecorator{
			Title: name,
		}
	}

	return new(NullNotifier)
}

type NullNotifier struct{}

func (n NullNotifier) Notify(string, ...any) {}
