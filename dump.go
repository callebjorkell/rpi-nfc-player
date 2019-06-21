package main

import (
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
	log "github.com/sirupsen/logrus"
	"strconv"
)

func dumpAll() {
	c, err := db.ReadAll()
	if err != nil {
		panic(err)
	}

	if len(*c) > 0 {
		fmt.Println("            ID │   AlbumId   │ PlaylistId │ Tracks ")
		fmt.Println("───────────────┼─────────────┼────────────┼────────")
	} else {
		fmt.Println("No cards found in the database...")
	}
	for _, card := range *c {
		fmt.Printf("%14v │ %11v │ %10v │ %4v \n", card.ID, *card.AlbumID, card.PlaylistID, len(card.Tracks))
	}
}

func dumpCard(cardId string) {
	if cardId == "" {
		id, err := readSingleCard()
		if err != nil {
			log.Fatal(err)
		}
		cardId = id
	}

	p, err := db.ReadCard(cardId)
	if err != nil {
		log.Error(err)
		return
	}

	if *dumpInfo {
		a, err := deezer.AlbumInfo(strconv.Itoa(*p.AlbumID))
		if err != nil {
			log.Error(err)
			return
		}
		fmt.Println(a)
	} else {
		fmt.Println(p)
	}
}
