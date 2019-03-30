package main

import (
	"github.com/callebjorkell/rpi-nfc-player/nfc"
	"github.com/callebjorkell/rpi-nfc-player/sonos"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func makeTrack(id string) sonos.Track {
	return sonos.Track{
		ID:       id,
		Type:     sonos.Music,
		Location: sonos.Deezer,
		Volume:   100,
	}
}

func main() {
	//s, err := sonos.New("Guest Room")
	//if err != nil {
	//	log.Fatal(err)
	//}

	//log.Println(s.Name(), "found")
	//
	//s.SetPlaylist(sonos.Playlist{
	//	ID:    123456,
	//	State: nil,
	//	Tracks: []sonos.Track{
	//		makeTrack("tr%3A63534071"),
	//		makeTrack("tr%3A404209842"),
	//		makeTrack("tr%3A404209862"),
	//		makeTrack("tr%3A404209892"),
	//	},
	//})
	//
	//s.Play()
	//time.Sleep(2 * time.Second)
	//stat, err := s.MediaInfo()
	//fmt.Printf("1: %v\n", stat)
	//
	//s.Next()
	//s.Next()
	//time.Sleep(2 * time.Second)
	//stat, err = s.MediaInfo()
	//fmt.Printf("2: %v\n", stat)
	//
	//s.Previous()
	//s.Play()
	//time.Sleep(10 * time.Second)
	//stat, err = s.MediaInfo()
	//fmt.Printf("3: %v\n", stat)
	//
	//s.Pause()

	//f, err := os.Create("label.png")
	//if err != nil {
	//	panic(err)
	//}
	//defer f.Close()
	//
	//if err := label.CreateLabel("11428738", f); err != nil {
	//	panic(err)
	//}

	//ui.Interact()
	rfid, err := nfc.MakeRFID(0, 0, 1000000, 22, 18)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for {
			uuid, err := rfid.ReadCardID()
			if err != nil {
				log.Println("shit:", err)
			} else {
				log.Println("ID: ", string(uuid))
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-signalChan:
		os.Exit(0)
	}
}
