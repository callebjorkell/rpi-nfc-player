package ui

import (
	"fmt"
	"github.com/stianeikeland/go-rpio/v4"
	"os"
	"time"
)

func Interact() {
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer rpio.Close()
	p := rpio.Pin(4)
	p.Output()
	for i := 0; i < 10; i++ {
		p.Toggle()
		fmt.Println("Blinky blinky")
		time.Sleep(2 * time.Second)
	}
}
