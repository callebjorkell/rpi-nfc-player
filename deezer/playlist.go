package deezer

import (
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
)

const playlistUriBase = "https://api.deezer.com/playlist"

type Playlist struct {
	Identifier uint64 `json:"id"`
	Cover      string `json:"picture_xl"`
	TrackList  struct {
		Data []struct {
			Id uint64 `json:"id"`
		} `json:"data"`
	} `json:"tracks"`
	TitleString string `json:"title"`
}

func (p Playlist) Tracks() []string {
	var tracks []string
	for _, value := range p.TrackList.Data {
		tracks = append(tracks, fmt.Sprint(value.Id))
	}
	return tracks
}

func (p Playlist) String() string {
	return fmt.Sprintf("ID: %v, title: %v, tracks: %v", p.Id, p.Title, len(p.TrackList.Data))
}

func (p Playlist) CoverArt() *image.Image {
	return fetchCoverArt(p.Cover)
}

func (p Playlist) Artist() string {
	return ""
}

func (p Playlist) Title() string {
	return p.TitleString
}

func (p Playlist) Id() string {
	return fmt.Sprintf("pl-%v", p.Identifier)
}

func GetPlaylist(id string) (*Playlist, error) {
	return getPlaylistInfo(id)
}

func getPlaylistInfo(playlistId string) (*Playlist, error) {
	u := fmt.Sprintf("%s/%s", playlistUriBase, playlistId)
	res, err := http.DefaultClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	c := new(Playlist)
	body, _ := ioutil.ReadAll(res.Body)
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, err
	}
	if c.TitleString == "" {
		return c, fmt.Errorf("title info is empty for playlist %v", playlistId)
	}
	return c, nil
}
