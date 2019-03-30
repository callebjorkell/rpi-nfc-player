package ui

import (
	"fmt"
	"os"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
	"time"
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
	red := gpioreg.ByName("GPIO21")
	blue := gpioreg.ByName("GPIO20")
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
	if err := b.In(gpio.PullUp, gpio.BothEdges); err != nil {
		handleErr(err)
	}
	last := b.Read()
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
			c <- buttonEvent{
				pressed: l == gpio.Low,
				name:    b.Name(),
			}
		}
	}
}
