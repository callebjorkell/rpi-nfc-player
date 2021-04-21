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
		if card.AlbumID != nil && *card.AlbumID > 0 {
			a, err := getAlbum(*card.AlbumID)
			if err != nil {
				log.Warn(err)
				continue
			}
			lists = append(lists, a)
		}
		if card.PlaylistID != nil && *card.PlaylistID > 0 {
			p, err := getPlaylist(*card.PlaylistID)
			if err != nil {
				log.Warn(err)
				continue
			}
			lists = append(lists, p)
		}
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

	for _, l := range *labelCardId {
		trackList, err := getPlayable(labelAlbumId, labelPlaylistId, l)
		if err != nil {
			log.Fatal(err)
		}
		generateLabel(trackList)
	}
}

func getPlayable(givenAlbumId, givenPlaylistId *uint64, cardId string) (deezer.Playable, error) {
	if givenAlbumId != nil && *givenAlbumId > 0 {
		return getAlbum(*givenAlbumId)
	}
	if givenPlaylistId != nil && *givenPlaylistId > 0 {
		return getPlaylist(*givenPlaylistId)
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
			return getAlbum(*card.AlbumID)
		}
		if card.PlaylistID != nil && *card.PlaylistID > 0 {
			return getPlaylist(*card.PlaylistID)
		}
	}
	return nil, fmt.Errorf("couldn't get a card with id %v", cardId)
}

func getPlaylist(id uint64) (deezer.Playable, error) {
	return deezer.GetPlaylist(fmt.Sprintf("%v", id))
}

func getAlbum(id uint64) (deezer.Playable, error) {
	return deezer.GetAlbum(fmt.Sprintf("%v", id))
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
