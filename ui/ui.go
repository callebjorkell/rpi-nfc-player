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

	push := rpio.Pin(26)
	push.PullUp()
	push.Detect(rpio.FallEdge)
	defer push.Detect(rpio.NoEdge)

	count := 0
	for {
		if !push.EdgeDetected() {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		count++
		p.Toggle()
		fmt.Println("I see pressed buttons! ", count)
	}
}
