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
	TrackList struct {
		Data []struct {
			Id uint64 `json:"id"`
		} `json:"data"`
	} `json:"tracks"`
	TitleString string `json:"title"`
}

func (a Album) Tracks() []string {
	var tracks []string
	for _, value := range a.TrackList.Data {
		tracks = append(tracks, fmt.Sprint(value.Id))
	}
	return tracks
}

func (a Album) Title() string {
	return a.TitleString
}

func (a Album) Artist() string {
	return a.ArtistContainer.Name
}

func (a Album) String() string {
	return fmt.Sprintf("ID: %v, artist: %v, title: %v, tracks: %v", a.Identifier, a.Artist(), a.Title(), len(a.TrackList.Data))
}

func GetAlbum(albumId string) (*Album, error) {
	return getAlbumInfo(albumId)
}

func (a Album) CoverArt() *image.Image {
	return fetchCoverArt(a.Cover)
}

func (a Album) Id() string {
	return fmt.Sprintf("a-%v", a.Identifier)
}

func getAlbumInfo(albumId string) (*Album, error) {
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
