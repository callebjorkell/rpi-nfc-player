package deezer

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	searchBase = "https://api.deezer.com/search"
)

type SearchContent struct {
	Data []struct {
		Id     uint64 `json:"id"`
		Type   string `json:"type"`
		Title  string `json:"title"`
		Artist struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"data"`
	Total int `json:"total"`
}

func Search(queryString string) (*SearchContent, error) {
	albums, err := search(fmt.Sprintf("%s/album?q=%s", searchBase, url.QueryEscape(queryString)))
	if err != nil {
		return nil, err
	}
	playlists, err := search(fmt.Sprintf("%s/playlist?q=%s", searchBase, url.QueryEscape(queryString)))
	if err != nil {
		return nil, err
	}

	return &SearchContent{
		Data:  append(albums.Data, playlists.Data...),
		Total: albums.Total + playlists.Total,
	}, nil
}

func search(u string) (*SearchContent, error) {
	log.Debug("Searching on ", u)
	res, err := http.DefaultClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	c := new(SearchContent)
	body, _ := ioutil.ReadAll(res.Body)
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, err
	}
	return c, nil
}
