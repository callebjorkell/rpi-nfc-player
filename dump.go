package main

import (
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
	log "github.com/sirupsen/logrus"
)

func dumpAll() {
	c, err := db.ReadAll()
	if err != nil {
		panic(err)
	}

	if len(*c) > 0 {
		fmt.Println("            ID │   AlbumId   │   PlaylistId    │ Tracks ")
		fmt.Println("───────────────┼─────────────┼─────────────────┼────────")
	} else {
		fmt.Println("No cards found in the database...")
	}
	for _, card := range *c {
		var album, playlist string
		if card.AlbumID != nil {
			album = fmt.Sprintf("%v", *card.AlbumID)
		}
		if card.PlaylistID != nil {
			playlist = fmt.Sprintf("%v", *card.PlaylistID)
		}
		fmt.Printf("%14v │ %11v │ %15v │ %4v \n", card.ID, album, playlist, len(card.Tracks))
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
		a, err := deezer.GetAlbum(fmt.Sprintf("%v", *p.AlbumID))
		if err != nil {
			log.Error(err)
			return
		}
		fmt.Println(a)
	} else {
		fmt.Println(p)
	}
}
