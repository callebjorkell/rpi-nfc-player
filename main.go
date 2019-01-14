package main

import (
	"github.com/callebjorkell/nfc-player/sonos"
	"log"
)

func main() {
	s, err := sonos.New("Guest Room")
	if err != nil {
		log.Fatal(err)
	}

	log.Println(s.Name(), "found")
	s.PlayDeezer("tr%3A63534071")
}
