package deezer

import (
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
)

const albumUriBase = "https://api.deezer.com/album"

type Album struct {
	Identifier      uint64 `json:"id"`
	Cover           string `json:"cover_xl"`
	ArtistContainer struct {
		Name string `json:"name"`
	} `json:"artist"`
	TitleString string `json:"title"`
}

func (a Album) Title() string {
	return a.TitleString
}

func (a Album) Artist() string {
	return a.ArtistContainer.Name
}

func (a Album) String() string {
	return fmt.Sprintf("ID: %v, artist: %v, title: %v", a.Identifier, a.Artist(), a.Title())
}

func (a Album) CoverArt() *image.Image {
	return fetchCoverArt(a.Cover)
}

func (a Album) Id() string {
	return fmt.Sprintf("a-%v", a.Identifier)
}

func GetAlbum(albumId string) (*Album, error) {
	u := fmt.Sprintf("%s/%s", albumUriBase, albumId)
	res, err := http.DefaultClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	c := new(Album)
	body, _ := ioutil.ReadAll(res.Body)
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, err
	}
	if c.ArtistContainer.Name == "" && c.TitleString == "" {
		return c, fmt.Errorf("album info is empty for album %v", albumId)
	}
	return c, nil
}
