package main

import (
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
	"github.com/callebjorkell/rpi-nfc-player/sonos"
	log "github.com/sirupsen/logrus"
)

func storeAlbum(id uint64, cardId string) {
	a, err := deezer.GetAlbum(fmt.Sprint(id))
	if err != nil {
		log.Error(err)
		return
	}

	if cardId == "" {
		cardId = getCardId()
	}
	p := sonos.FromAlbum(a, cardId)

	db.StoreCard(p)
}

func storePlaylist(id uint64, cardId string) {
	p, err := deezer.GetPlaylist(fmt.Sprint(id))
	if err != nil {
		log.Error(err)
		return
	}

	if cardId == "" {
		cardId = getCardId()
	}
	pl := sonos.FromPlaylist(p, cardId)

	db.StoreCard(pl)
}

func getCardId() string {
	id, err := readSingleCard()
	if err != nil {
		log.Fatal(err)
	}
	return id
}
