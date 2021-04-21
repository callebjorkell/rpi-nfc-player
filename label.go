package main

import (
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
	"github.com/callebjorkell/rpi-nfc-player/sonos"
	log "github.com/sirupsen/logrus"
	"os"
)

func createSheet(cardIds *[]string) {
	var cards = new([]sonos.CardInfo)
	log.Debug("Cards: ", cardIds)
	if cardIds != nil && len(*cardIds) > 0 {
		for _, c := range *cardIds {
			playlist, err := db.ReadCard(c)
			if err != nil {
				log.Warn(err)
				continue
			}

			*cards = append(*cards, playlist)
		}
	} else {
		all, _ := db.ReadAll()
		cards = all
	}
	var lists []deezer.Playable
	for _, card := range *cards {
		p, err := card.ToPlayable()
		if err != nil {
			log.Warn(err)
			continue
		}
		lists = append(lists, p)
	}

	step := deezer.LabelsPerSheet
	for i := 0; i*step < len(lists); i++ {
		f, err := os.Create(fmt.Sprintf("sheet%v.png", i))
		if err != nil {
			panic(err)
		}

		index := i * step
		if err := deezer.CreateLabelSheet(lists[index:min(index+step, len(lists))], f); err != nil {
			panic(err)
		}
		f.Close()
	}
}

func createLabel() {
	if *sheet {
		createSheet(labelCardId)
		return
	}

	if labelAlbumId != nil && *labelAlbumId > 0 {
		p, err := deezer.GetAlbum(fmt.Sprintf("%v", labelAlbumId))
		if err != nil {
			log.Fatal(err)
		}
		generateLabel(p)
	} else if labelPlaylistId != nil && *labelPlaylistId > 0 {
		p, err := deezer.GetPlaylist(fmt.Sprintf("%v", labelPlaylistId))
		if err != nil {
			log.Fatal(err)
		}
		generateLabel(p)
	} else if len(*labelCardId) > 0 {
		for _, l := range *labelCardId {
			p, err := getPlayable(l)
			if err != nil {
				log.Fatal(err)
			}
			generateLabel(p)
		}
	} else {
		if read, err := readSingleCard(); err != nil {
			log.Fatal(err)
		} else {
			p, err := getPlayable(read)
			if err != nil {
				log.Fatal(err)
			}
			generateLabel(p)
		}
	}
}

func getPlayable(cardId string) (deezer.Playable, error) {
	card, err := db.ReadCard(cardId)
	if err != nil {
		return nil, fmt.Errorf("couldn't get a card with id %v", cardId)
	}
	return card.ToPlayable()
}

func generateLabel(t deezer.Playable) {
	file := fmt.Sprintf("%v.png", t.Id())
	log.Infof("Generating label for %v into %v", t.Id(), file)

	f, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := deezer.CreateLabel(t, f); err != nil {
		panic(err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
