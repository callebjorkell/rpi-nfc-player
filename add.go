package main

import (
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
	"github.com/callebjorkell/rpi-nfc-player/sonos"
	log "github.com/sirupsen/logrus"
)

func addAlbum(id uint64) {
	a, err := deezer.GetAlbum(fmt.Sprint(id))
	if err != nil {
		log.Error(err)
		return
	}

	cardId := getCardId()
	p := sonos.FromAlbum(a, cardId)

	db.StoreCard(p)
}

func addPlaylist(id uint64) {
	p, err := deezer.GetPlaylist(fmt.Sprint(id))
	if err != nil {
		log.Error(err)
		return
	}

	cardId := getCardId()
	pl := sonos.FromPlaylist(p, cardId)

	db.StoreCard(pl)
}

func getCardId() string {
	if *addCardId != "" {
		return *addCardId
	} else {
		id, err := readSingleCard()
		if err != nil {
			log.Fatal(err)
		}
		return id
	}
}