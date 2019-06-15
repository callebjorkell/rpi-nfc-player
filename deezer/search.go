package deezer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const searchBase = "https://api.deezer.com/search/album?q="

type searchContent struct {
	Data []struct {
		Id     int    `json:"id"`
		Title  string `json:"title"`
		Artist struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"data"`
	Total int `json:"total"`
}

func AlbumSearch(queryString string) (*searchContent, error) {
	u := fmt.Sprintf("%s%s", searchBase, url.QueryEscape(queryString))
	res, err := http.DefaultClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	c := new(searchContent)
	body, _ := ioutil.ReadAll(res.Body)
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, err
	}
	return c, nil
}
