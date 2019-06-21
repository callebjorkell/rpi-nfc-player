package main

import (
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
	log "github.com/sirupsen/logrus"
)

func searchAlbum() {
	r, err := deezer.AlbumSearch(*searchString)
	if err != nil {
		log.Error(err)
		return
	}

	if len(r.Data) > 0 {
		if len(r.Data) < r.Total {
			fmt.Printf("Too many matches (%v). Only showing the first %v.\n\n", r.Total, len(r.Data))
		}
		fmt.Println("            ID │ Artist - Title")
		fmt.Println("───────────────┼────────────────────")
		for _, v := range r.Data {
			fmt.Printf("%14v │ %v - %v\n", v.Id, checkLength(v.Artist.Name, 50), checkLength(v.Title, 75))

		}
	} else {
		fmt.Println("No matches. Try a different query string.")
	}
}

func checkLength(s string, l int) string {
	if len(s) > l {
		return s[:l] + "…"
	}
	return s
}
