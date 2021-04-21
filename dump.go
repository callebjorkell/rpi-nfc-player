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
		fmt.Println("            ID │   AlbumId   │   PlaylistId    │ Title")
		fmt.Println("───────────────┼─────────────┼─────────────────┼─────────────────────────────────────────")
	} else {
		fmt.Println("No cards found in the database...")
	}
	for _, card := range *c {
		var album, playlist string
		title := ""
		if card.AlbumID != nil {
			album = fmt.Sprintf("%v", *card.AlbumID)
			a, err := deezer.GetAlbum(fmt.Sprintf("%v", *card.AlbumID))
			if err == nil {
				title = fmt.Sprintf("%v - %v", a.Artist(), a.Title())
			}
		}
		if card.PlaylistID != nil {
			playlist = fmt.Sprintf("%v", *card.PlaylistID)
			p, err := deezer.GetPlaylist(fmt.Sprintf("%v", *card.PlaylistID))
			if err == nil {
				title = p.Title()
			}
		}
		if len(title) > 40 {
			title = fmt.Sprintf("%.39v…", title)
		}
		fmt.Printf("%14v │ %11v │ %15v │ %v\n", card.ID, album, playlist, title)
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
