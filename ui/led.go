package ui

import (
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
)

type ColorLed struct {
	r gpio.PinIO
	g gpio.PinIO
	b gpio.PinIO
}

func (c *ColorLed) Green() {
	c.Off()
	c.g.Out(gpio.Low)
}

func (c *ColorLed) Blue() {
	c.Off()
	c.b.Out(gpio.Low)
}

func (c *ColorLed) Red() {
	c.Off()
	c.r.Out(gpio.Low)
}

func (c *ColorLed) Purple() {
	c.Off()
	c.r.Out(gpio.Low)
	c.b.Out(gpio.Low)
}

func (c *ColorLed) Yellow() {
	c.Off()
	c.r.Out(gpio.Low)
	c.g.Out(gpio.Low)
}

func (c *ColorLed) Cyan() {
	c.Off()
	c.g.Out(gpio.Low)
	c.b.Out(gpio.Low)
}

func (c *ColorLed) White() {
	c.r.Out(gpio.Low)
	c.g.Out(gpio.Low)
	c.b.Out(gpio.Low)
}

func (c *ColorLed) Off() {
	c.r.Out(gpio.High)
	c.g.Out(gpio.High)
	c.b.Out(gpio.High)
}

func GetColorLED() ColorLed {
	redLED := gpioreg.ByName("GPIO6")
	greenLED := gpioreg.ByName("GPIO5")
	blueLED := gpioreg.ByName("GPIO13")

	c := ColorLed{r: redLED, g: greenLED, b: blueLED}
	c.Off()
	return c
}
