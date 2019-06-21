package main

import (
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
)

func createSheet() {
	cards, err := db.ReadAll()
	var ids []string
	if err == nil {
		for _, card := range *cards {
			if card.AlbumID != nil && *card.AlbumID > 0 {
				ids = append(ids, strconv.Itoa(*card.AlbumID))
			}
		}
	}

	step := deezer.LabelsPerSheet
	for i := 0; i*step < len(ids); i++ {
		f, err := os.Create(fmt.Sprintf("sheet%v.png", i))
		if err != nil {
			panic(err)
		}

		index := i * step
		if err := deezer.CreateLabelSheet(ids[index:min(index+step, len(ids))], f); err != nil {
			panic(err)
		}
		f.Close()
	}
}

func createLabel() {
	if *sheet {
		createSheet()
		return
	}

	id := getLabelAlbumId(*labelAlbumId, *labelCardId)

	generateLabel(id)
}

func getLabelAlbumId(givenAlbumId uint32, cardId string) uint32 {
	if givenAlbumId > 0 {
		return givenAlbumId
	}

	if cardId == "" {
		if read, err := readSingleCard(); err != nil {
			log.Fatal(err)
		} else {
			cardId = read
		}
	}

	card, err := db.ReadCard(cardId)
	if err == nil {
		if card.AlbumID != nil && *card.AlbumID > 0 {
			return uint32(*card.AlbumID)
		}
	}
	panic(fmt.Errorf("couldn't get a card with id %v", cardId))
}

func generateLabel(id uint32) {
	file := fmt.Sprintf("%v.png", id)
	log.Infof("Generating label for album %v into %v", albumId, file)

	f, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := deezer.CreateLabel(fmt.Sprintf("%d", id), f); err != nil {
		panic(err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
