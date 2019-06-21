package ui

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
	"sync"
	"time"
)

func init() {
	if _, err := host.Init(); err != nil {
		logrus.Fatalln("Unable to initialize periph:", err)
	}
}

var buttonStates [3]bool

type Button int

const (
	redPin                = "GPIO21"
	bluePin               = "GPIO20"
	tigerSwitchPin        = "GPIO16"
	tigerPin              = "GPIO23"
	Red            Button = 0
	Blue           Button = 1
	TigerSwitch    Button = 2
)

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

type Tiger struct {
	pin gpio.PinIO
}

func (t Tiger) On() {
	t.pin.Out(gpio.Low)
}

func (t Tiger) Off() {
	t.pin.Out(gpio.High)
}

// InitTiger fetches and resets the tiger pin
func InitTiger() *Tiger {
	pin := gpioreg.ByName(tigerPin)
	t := Tiger{pin: pin}
	t.Off()

	return &t
}

// InitButtons initializes all the button pins and fetches a button event channel
func InitButtons() chan ButtonEvent {
	logrus.Infoln("Initializing buttons")
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

func IsPressed(button Button) bool {
	return buttonStates[button]
}

func handleButton(b gpio.PinIO, t Button, c chan ButtonEvent, initialized *sync.WaitGroup) {
	logrus.Debugln("Handling button ", b.Name())
	if err := b.In(gpio.PullUp, gpio.BothEdges); err != nil {
		logrus.Fatal(err)
	}

	last := b.Read()
	saveState(t, last)
	initialized.Done()
	for {
		// wait for the edge
		if !b.WaitForEdge(time.Second) {
			continue
		}

		// debounce
		l := b.Read()
		if l == last {
			continue
		}

		time.Sleep(50 * time.Millisecond)
		if l == b.Read() {
			// ... and handle
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
