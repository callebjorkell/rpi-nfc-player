package ui

import (
	"fmt"
	"os"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
	"sync"
	"time"
)

func init() {
	if _, err := host.Init(); err != nil {
		handleErr(err)
	}
}

var buttonStates [3]bool

const (
	redPin         = "GPIO21"
	bluePin        = "GPIO20"
	tigerSwitchPin = "GPIO16"
	tigerPin       = "GPIO23"
)

func handleErr(err error) {
	fmt.Println(err)
	os.Exit(1)
}

type ButtonEvent struct {
	Pressed bool
	Button  Button
}

func (b ButtonEvent) String() string {
	action := "pressed"
	if !b.Pressed {
		action = "released"
	}
	return fmt.Sprintf("Button %v was %v", b.Button.String(), action)
}

type Tiger struct {
	pin gpio.PinIO
}

func (t Tiger) On() {
	t.pin.Out(gpio.Low)
}

func (t Tiger) Off() {
	t.pin.Out(gpio.High)
}

func GetTiger() *Tiger {
	pin := gpioreg.ByName(tigerPin)
	t := Tiger{pin: pin}
	t.Off()

	return &t
}

type Button int

const (
	Red         Button = 0
	Blue        Button = 1
	TigerSwitch Button = 2
)

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

func InitButtons() chan ButtonEvent {
	redButton := gpioreg.ByName(redPin)
	blueButton := gpioreg.ByName(bluePin)
	tigerSwitch := gpioreg.ByName(tigerSwitchPin)

	c := make(chan ButtonEvent, 10)
	initialized := sync.WaitGroup{}
	initialized.Add(3)
	go handleButton(blueButton, Blue, c, &initialized)
	go handleButton(redButton, Red, c, &initialized)
	go handleButton(tigerSwitch, TigerSwitch, c, &initialized)
	initialized.Wait()
	return c
}

// IsPressed returns true if the button is currently pressed, and false otherwize
func IsPressed(button Button) bool {
	return buttonStates[button]
}

func handleButton(b gpio.PinIO, t Button, c chan ButtonEvent, initialized *sync.WaitGroup) {
	fmt.Println("Handling button ", b.Name())
	if err := b.In(gpio.PullUp, gpio.BothEdges); err != nil {
		handleErr(err)
	}

	last := b.Read()
	saveState(t, last)
	initialized.Done()
	for {
		// read and debounce
		if !b.WaitForEdge(-1) {
			continue
		}

		l := b.Read()
		if l == last {
			continue
		}

		time.Sleep(50 * time.Millisecond)
		if l == b.Read() {
			last = l
			saveState(t, l)
			c <- ButtonEvent{
				Pressed: l == gpio.Low,
				Button:  t,
			}
		}
	}
}

func saveState(t Button, l gpio.Level) {
	pressed := l == gpio.Low
	buttonStates[t] = pressed
}
