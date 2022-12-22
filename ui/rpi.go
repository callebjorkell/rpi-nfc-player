//+build pi

package ui

import (
	"github.com/sirupsen/logrus"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
	"sync"
	"time"
)

const (
	redPin         = "GPIO21"
	bluePin        = "GPIO20"
	tigerSwitchPin = "GPIO16"
	tigerPin       = "GPIO23"
)

func init() {
	if _, err := host.Init(); err != nil {
		logrus.Fatalln("Unable to initialize periph:", err)
	}
}

var buttonStates [3]bool

type tiger struct {
	pin gpio.PinIO
}

type colorLed struct {
	r gpio.PinIO
	g gpio.PinIO
	b gpio.PinIO
}

func (c *colorLed) Green() {
	c.Off()
	c.g.Out(gpio.Low)
}

func (c *colorLed) Blue() {
	c.Off()
	c.b.Out(gpio.Low)
}

func (c *colorLed) Red() {
	c.Off()
	c.r.Out(gpio.Low)
}

func (c *colorLed) Purple() {
	c.Off()
	c.r.Out(gpio.Low)
	c.b.Out(gpio.Low)
}

func (c *colorLed) Yellow() {
	c.Off()
	c.r.Out(gpio.Low)
	c.g.Out(gpio.Low)
}

func (c *colorLed) Cyan() {
	c.Off()
	c.g.Out(gpio.Low)
	c.b.Out(gpio.Low)
}

func (c *colorLed) White() {
	c.r.Out(gpio.Low)
	c.g.Out(gpio.Low)
	c.b.Out(gpio.Low)
}

func (c *colorLed) Off() {
	c.r.Out(gpio.High)
	c.g.Out(gpio.High)
	c.b.Out(gpio.High)
}

func GetColorLED() ColorLed {
	logrus.Infoln("Initializing LED")

	redLED := gpioreg.ByName("GPIO6")
	greenLED := gpioreg.ByName("GPIO5")
	blueLED := gpioreg.ByName("GPIO13")

	c := colorLed{r: redLED, g: greenLED, b: blueLED}
	c.Off()
	return &c
}

func (t tiger) On() {
	t.pin.Out(gpio.Low)
}

func (t tiger) Off() {
	t.pin.Out(gpio.High)
}

// InitTiger fetches and resets the tiger pin
func InitTiger() Tiger {
	pin := gpioreg.ByName(tigerPin)
	t := tiger{pin: pin}
	t.Off()

	return &t
}

// InitButtons initializes all the button pins and fetches a button event channel
func InitButtons() <-chan ButtonEvent {
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

func handleButton(b gpio.PinIO, t Button, c chan ButtonEvent, initialized *sync.WaitGroup) {
	logrus.Debugln("Handling button ", b.Name())
	if err := b.In(gpio.PullUp, gpio.BothEdges); err != nil {
		logrus.Fatal(err)
	}

	last := gpio.High
	if t == TigerSwitch {
		// Make sure that we always get an initial event for the tiger if it's on by saying that the last state
		// was turned off.
		saveState(t, last)
	} else {
		last := b.Read()
		saveState(t, last)
	}

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
