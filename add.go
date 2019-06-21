package main

import (
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
	"github.com/callebjorkell/rpi-nfc-player/sonos"
	log "github.com/sirupsen/logrus"
)

func addAlbum(id uint32) {
	a, err := deezer.AlbumInfo(fmt.Sprint(id))
	if err != nil {
		log.Error(err)
		return
	}

	var cardId string
	if *albumCardId != "" {
		cardId = *albumCardId
	} else {
		id, err := readSingleCard()
		if err != nil {
			log.Fatal(err)
		}
		cardId = id
	}
	p := sonos.FromAlbum(a, cardId)

	db.StoreCard(*p)
}
