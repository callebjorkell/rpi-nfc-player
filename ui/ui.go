package ui

import (
	"fmt"
)

const (
	Red         Button = 0
	Blue        Button = 1
	TigerSwitch Button = 2
)

type ColorLed interface {
	Purple()
	Yellow()
	Cyan()
	Red()
	Green()
	Blue()
	Off()
}

type Button int

type Tiger interface {
	Off()
	On()
}

func (b ButtonEvent) String() string {
	action := "pressed"
	if !b.Pressed {
		action = "released"
	}
	return fmt.Sprintf("Button %v was %v", b.Button.String(), action)
}

func (b Button) String() string {
	if b == Red {
		return "red"
	}
	if b == Blue {
		return "blue"
	}
	if b == TigerSwitch {
		return "tiger switch"
	}
	return ""
}

type ButtonEvent struct {
	Pressed bool
	Button  Button
}
