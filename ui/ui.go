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
	redButton := gpioreg.ByName("GPIO21")
	blueButton := gpioreg.ByName("GPIO20")
	tigerSwitch := gpioreg.ByName("GPIO16")

	redLED := gpioreg.ByName("GPIO6")
	greenLED := gpioreg.ByName("GPIO5")
	blueLED := gpioreg.ByName("GPIO13")

	go doRGB(redLED, greenLED, blueLED)
	tiger := gpioreg.ByName("GPIO23")

	c := make(chan buttonEvent, 10)
	go handleButton(blueButton, c)
	go triggerTiger(redButton, tiger)
	go handleButton(tigerSwitch, c)
	for {
		select {
		case e := <-c:
			fmt.Printf("I see button events! %v\n", e)
		}
	}
}

func doRGB(r gpio.PinIO, g gpio.PinIO, b gpio.PinIO) {
	leds := []gpio.PinIO{r,g,b}
	index := 0
	for _, l := range leds {
		l.Out(gpio.High)
	}
	for {
		leds[index].Out(gpio.Low)
		fmt.Printf("%d ",index)
		time.Sleep(time.Second)
		leds[index].Out(gpio.High)

		index++
		index = index%3
	}
}

func triggerTiger(b gpio.PinIO, tiger gpio.PinIO) {
	fmt.Println("Handling tiger button ", b.Name())
	tiger.Out(gpio.Low)
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
			if l != gpio.Low {
				continue
			}

			fmt.Println("enabling tiger")
			tiger.Out(gpio.Low)
			time.Sleep(time.Second)
			tiger.Out(gpio.High)
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
