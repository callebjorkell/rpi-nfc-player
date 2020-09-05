//+build !pi

package ui

import (
	"github.com/sirupsen/logrus"
	"time"
)

var ch = make(chan ButtonEvent, 1)

func InitButtons() <-chan ButtonEvent {
	go func() {
		<-time.After(20 * time.Second)
		ch <- ButtonEvent{
			Pressed: true,
			Button:  TigerSwitch,
		}
	}()
	return ch
}

func InitTiger() Tiger {
	return cliTiger{}
}

func GetColorLED() ColorLed {
	return cliLed{}
}

type cliLed struct{}

func (cliLed) Purple() {
	logrus.Println("LED: Purple")
}

func (cliLed) Yellow() {
	logrus.Println("LED: Yellow")
}

func (cliLed) Cyan() {
	logrus.Println("LED: Cyan")
}

func (cliLed) Red() {
	logrus.Println("LED: Red")
}

func (cliLed) Green() {
	logrus.Println("LED: Green")
}

func (cliLed) Blue() {
	logrus.Println("LED: Blue")
}

func (cliLed) Off() {
	logrus.Println("LED: Off")
}

type cliTiger struct{}

func (cliTiger) Off() {
	logrus.Println("Tiger deactivated")
}

func (cliTiger) On() {
	logrus.Println("Tiger activated")
	ch <- ButtonEvent{
		Pressed: false,
		Button:  TigerSwitch,
	}
}
