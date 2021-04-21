package main

import (
	"fmt"
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
		title := card.Title
		album := card.AlbumIDString()
		playlist := card.PlaylistIDString()
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
		a, err := p.ToPlayable()
		if err != nil {
			log.Error(err)
			return
		}
		fmt.Println(a)
	} else {
		fmt.Println(p)
	}
}
