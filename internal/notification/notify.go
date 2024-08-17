package notification

import (
	"fmt"
	"github.com/gen2brain/beeep"
)

type Notifier interface {
	Notify(message string, v ...any)
}

type BeepDecorator struct {
	Title string
}

func (b BeepDecorator) Notify(message string, v ...any) {
	beeep.Notify(b.Title, fmt.Sprintf(message, v...), "")
}

type NullNotifier struct{}

func (n NullNotifier) Notify(string, ...any) {}
