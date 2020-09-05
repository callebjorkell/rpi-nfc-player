package nfc

import (
	"io"
)

const (
	Activated   CardState = 0
	Deactivated CardState = 1
)

type CardReader interface {
	io.Closer
	Events() <-chan CardEvent
}

type CardState int
type CardEvent struct {
	CardID string
	State  CardState
}
