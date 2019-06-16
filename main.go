package main

import (
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
	"github.com/callebjorkell/rpi-nfc-player/nfc"
	"github.com/callebjorkell/rpi-nfc-player/sonos"
	"github.com/callebjorkell/rpi-nfc-player/ui"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var db = nfc.GetDB()

var (
	app   = kingpin.New("nfc-player", "Music player that plays deezer playlists on a sonos speakes with the help of NFC cards, a Raspberry Pi and some buttons.")
	debug = app.Flag("debug", "Turn on debug logging.").Bool()
	start = app.Command("start", "Start the music player and start listening for NFC cards.")

	add         = app.Command("add", "Construct and add a new playlist to a card.")
	playlist    = add.Command("playlist", "Add a deezer playlist.")
	playlistId  = playlist.Arg("id", "The ID of the album that should be added.").Required().Uint32()
	album       = add.Command("album", "Add a deezer album.")
	albumId     = album.Arg("id", "The ID of the album that should be added.").Required().Uint32()
	albumCardId = album.Flag("cardId", "Manually specify the card id to be used.").String()

	dump       = app.Command("dump", "Read a card and dump all the available information onto standard out.")
	dumpCardId = dump.Flag("cardId", "Manually specify the card id to be used.").String()
	dumpList   = dump.Flag("list", "Dump a short list of all the cards in the database").Bool()

	search       = app.Command("search", "Search for albums on deezer")
	searchString = search.Arg("query", "The string to search on.").Required().String()

	label        = app.Command("label", "Create a label for a card.")
	labelAlbumId = label.Flag("id", "The id of the album that should be created. If not provided, a card will be requested.").Uint32()
	labelCardId  = label.Flag("cardId", "Manually specify the card that the label should be printed for").String()
)

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-signalChan:
			os.Exit(0)
		}
	}()

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("%v: Try --help\n", err.Error())
		os.Exit(1)
	}

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	if *debug {
		log.Info("Enabling debug output...")
		log.SetLevel(log.DebugLevel)
	}

	switch cmd {
	case start.FullCommand():
		startServer()
	case album.FullCommand():
		addAlbum(*albumId)
	case playlist.FullCommand():
		addPlaylist(*playlistId)
	case dump.FullCommand():
		if *dumpList == true {
			dumpAll()
		} else {
			dumpCard(*dumpCardId)
		}
	case search.FullCommand():
		searchAlbum()
	case label.FullCommand():
		createLabel()
	default:
		kingpin.FatalUsage("Unrecognized command")
	}
}

func dumpAll() {
	c, err := db.ReadAll()
	if err != nil {
		panic(err)
	}

	if len(*c) > 0 {
		fmt.Println("        ID │   AlbumId   │ PlaylistId │ Tracks ")
		fmt.Println("───────────┼─────────────┼────────────┼────────")
	} else {
		fmt.Println("No cards found in the database...")
	}
	for _, card := range *c {
		fmt.Printf("%10v │ %11v │ %10v │ %4v \n", card.ID, *card.AlbumID, card.PlaylistID, len(card.Tracks))
	}
}

func dumpCard(cardId string) {
	if cardId == "" {
		log.Error("No card specified")
		return
	}

	p, err := db.ReadCard(cardId)
	if err != nil {
		log.Error(err)
		return
	}
	fmt.Println(p.String())
}

func addPlaylist(id uint32) {

}

func addAlbum(id uint32) {
	a, err := deezer.AlbumInfo(fmt.Sprint(id))
	if err != nil {
		log.Error(err)
		return
	}

	var cardId string
	if albumCardId != nil {
		cardId = *albumCardId
	}
	p := sonos.FromAlbum(a, cardId)

	db.StoreCard(*p)
}

func createLabel() {
	id := getAlbumId(*labelAlbumId, *labelCardId)

	generateLabel(id)
}

func getAlbumId(givenAlbumId uint32, cardId string) uint32 {
	if givenAlbumId > 0 {
		return givenAlbumId
	}
	if cardId != "" {
		card, err := db.ReadCard(cardId)
		if err == nil {
			if card.AlbumID != nil && *card.AlbumID > 0 {
				return uint32(*card.AlbumID)
			}
		}
		log.Error("Couldn't get a card with id ", cardId)
	}

	panic("implement this")
	// TODO: read a card to figure out what is what.
}

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
		fmt.Println("        ID │ Artist - Title")
		fmt.Println("───────────┼────────────────────")
		for _, v := range r.Data {
			fmt.Printf("%10v │ %v - %v\n", v.Id, checkLength(v.Artist.Name, 50), checkLength(v.Title, 75))

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

func generateLabel(id uint32) {
	f, err := os.Create("label.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := deezer.CreateLabel(fmt.Sprintf("%d", id), f); err != nil {
		panic(err)
	}
}

type CardState int

const (
	Activated   CardState = 0
	Deactivated CardState = 1
)

type Event struct {
	CardID string
	State  CardState
}

func startServer() {
	tiger := ui.GetTiger()
	buttons := ui.InitButtons()
	led := ui.GetColorLED()
	s, err := sonos.New("Guest Room")
	if err != nil {
		log.Fatal(err)
	}

	checkTiger := tigerCheck(tiger, led)
	checkTiger()
	play := false

	go func() {
		for {
			select {
			case b, ok := <-buttons:
				if !ok {
					log.Error("Button channel has closed. Stopping event loop.")
					return
				}

				switch b.Button {
				case ui.TigerSwitch:
					if b.Pressed {
						if !play {
							tiger.On()
							led.Red()
						}
					} else {
						tiger.Off()
						led.Off()
					}
				case ui.Red:
					if b.Pressed && play {
						s.Previous()
					}
				case ui.Blue:
					if b.Pressed && play {
						s.Next()
					}
				}
				fmt.Println(b.String())
			}
		}
	}()

	events := cardChannel()
	for {
		card, open := <-events
		if !open {
			return
		}

		if card.State == Activated {
			fmt.Printf("Card %v activated\n", card.CardID)
			led.Yellow()
			play = true

			p, err := db.ReadCard(card.CardID)
			if err != nil {
				fmt.Println(err)
				continue
			}
			s.SetPlaylist(p)
			// apparently this returns before the player is ready
			time.Sleep(500 * time.Millisecond)

			s.Play()

			led.Green()

		} else {
			fmt.Println("Card removed...")
			s.Pause()
			led.Off()
			play = false
			checkTiger()
		}
	}
}

func tigerCheck(tiger *ui.Tiger, led *ui.ColorLed) func() {
	return func() {
		if ui.IsPressed(ui.TigerSwitch) {
			log.Info("Tiger switched on already, enabling tiger.")
			led.Red()
			tiger.On()
		}
	}
}

func cardChannel() <-chan Event {
	//s, err := sonos.New("Guest Room")
	//if err != nil {
	//	log.Fatal(err)
	//}

	//log.Println(s.Name(), "found")
	//
	//s.SetPlaylist(sonos.Playlist{
	//	ID:    123456,
	//	State: nil,
	//	Tracks: []sonos.Track{
	//		makeTrack("tr%3A63534071"),
	//		makeTrack("tr%3A404209842"),
	//		makeTrack("tr%3A404209862"),
	//		makeTrack("tr%3A404209892"),
	//	},
	//})
	//
	//s.Play()
	//time.Sleep(2 * time.Second)
	//stat, err := s.MediaInfo()
	//fmt.Printf("1: %v\n", stat)
	//
	//s.Next()
	//s.Next()
	//time.Sleep(2 * time.Second)
	//stat, err = s.MediaInfo()
	//fmt.Printf("2: %v\n", stat)
	//
	//s.Previous()
	//s.Play()
	//time.Sleep(10 * time.Second)
	//stat, err = s.MediaInfo()
	//fmt.Printf("3: %v\n", stat)
	//
	//s.Pause()

	//ui.Interact()
	reader, err := nfc.MakeRFID(0, 0, 100000, 22, 18)
	if err != nil {
		log.Fatal(err)
	}
	events := make(chan Event, 10)
	go func() {
		defer close(events)
		lastConfirmedId, lastSeenId := "", ""
		debounceIndex := 0
		for {
			time.Sleep(150 * time.Millisecond)

			id, err := reader.ReadCardID()
			if err != nil {
				log.Debugf("error when reading card ID: %v", err)
			}

			log.Debugf("ID: %v, lastSeen: %v, lastConfirmed: %v, debounce: %v", id, lastSeenId, lastConfirmedId, debounceIndex)

			if lastSeenId != id {
				lastSeenId = id
				debounceIndex = 0
				continue
			}

			if lastConfirmedId == id {
				continue
			}

			// debounce the card, in case we have half reads, or multiple cards. This means there is a slight lag to
			// start playing, but it also means it's way more stable when it actually does...
			debounceIndex++
			if debounceIndex >= 4 {
				if id == "" {
					// There is no card currently
					log.Debugln("Sending deactivation event")
					events <- Event{State: Deactivated, CardID: ""}
				} else {
					log.Debugf("Sending activation event for card %v", id)
					events <- Event{State: Activated, CardID: id}

					// there seems to be some issues with reading sometimes. Not sure why that would be, but here we
					// sleep as an extra countermeasure against "bounce". Since a card was just added, we might
					// as well let it get going before reading again.
					time.Sleep(1000 * time.Millisecond)
				}
				lastConfirmedId = id
				debounceIndex = 0
			}
		}
	}()
	return events
}
