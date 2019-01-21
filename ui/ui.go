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

func Interact() {
	if _, err := host.Init(); err != nil {
		handleErr(err)
	}
	fmt.Println(gpioreg.All())
	button := gpioreg.ByName("GPIO26")
	if err := button.In(gpio.PullUp, gpio.FallingEdge); err != nil {
		handleErr(err)
	}

	led := gpioreg.ByName("GPIO4")

	count := 0
	for {
		if !button.WaitForEdge(-1) {
			continue
		}
		fmt.Println("Button was: ", button.Read())
		count++
		led.Out(!led.Read())
		fmt.Println("I see pressed buttons! ", count)
	}
}
