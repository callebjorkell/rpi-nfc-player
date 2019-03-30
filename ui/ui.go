package ui

import (
	"fmt"
	"os"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

func handleErr(err error) {
	fmt.Println(err)
	os.Exit(1)
}

type buttonEvent struct {
	pressed bool
	name    string
}

func Interact() {
	if _, err := host.Init(); err != nil {
		handleErr(err)
	}
	fmt.Println(gpioreg.All())
	red := gpioreg.ByName("GPIO21")
	if err := red.In(gpio.PullUp, gpio.BothEdges); err != nil {
		handleErr(err)
	}

	blue := gpioreg.ByName("GPIO20")
	if err := blue.In(gpio.PullUp, gpio.BothEdges); err != nil {
		handleErr(err)
	}

	led := gpioreg.ByName("GPIO4")

	c := make(chan buttonEvent, 10)
	go handleButton(red, c)
	go handleButton(blue, c)
	for {
		select {
		case e := <-c:
			led.Out(gpio.Level(e.pressed))
			fmt.Printf("I see button events! %v\n", e)
		}
	}
}

func handleButton(b gpio.PinIO, c chan buttonEvent) {
	fmt.Println("Handling button ", b.Name())
	for {
		if !b.WaitForEdge(-1) {
			continue
		}
		l := b.Read()
		e := buttonEvent{
			pressed: l == gpio.Low,
			name:    b.Name(),
		}
		c <- e
	}
}
