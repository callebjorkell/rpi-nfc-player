package deezer

import (
	log "github.com/sirupsen/logrus"
	"image"
	"net/http"
)

type Playable interface {
	String() string
	CoverArt() *image.Image
	Id() string
	Artist() string
	Title() string
}

func fetchCoverArt(uri string) *image.Image {
	if uri == "" {
		return defaultArt
	}
	res, err := http.DefaultClient.Get(uri)
	if err != nil {
		log.Debug(err)
		return defaultArt
	}
	defer res.Body.Close()

	img, _, err := image.Decode(res.Body)
	if err != nil {
		log.Debug(err)
		return defaultArt
	}
	return &img
}
