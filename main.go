package main

import (
	"errors"
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
	"github.com/callebjorkell/rpi-nfc-player/nfc"
	"github.com/callebjorkell/rpi-nfc-player/sonos"
	"github.com/callebjorkell/rpi-nfc-player/ui"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var db = nfc.GetDB()

var (
	app   = kingpin.New("nfc-player", "Music player that plays deezer playlists on a sonos speakes with the help of NFC cards, a Raspberry Pi and some buttons.")
	debug = app.Flag("debug", "Turn on debug logging.").Bool()
	start = app.Command("start", "Start the music player and start listening for NFC cards.")

	add         = app.Command("add", "Construct and add a new playlist to a card.")
	albumId     = add.Arg("id", "The ID of the album that should be added.").Required().Uint32()
	albumCardId = add.Flag("cardId", "Manually specify the card id to be used.").String()

	dump       = app.Command("dump", "Read a card and dump all the available information onto standard out.")
	dumpCardId = dump.Flag("cardId", "Manually specify the card id to be used.").String()
	dumpInfo   = dump.Flag("albumInfo", "Dump information about the album the card points to instead of the data on the card.").Bool()
	dumpList   = dump.Flag("list", "Dump a short list of all the cards in the database").Bool()

	search       = app.Command("search", "Search for albums on deezer")
	searchString = search.Arg("query", "The string to search on.").Required().String()

	label        = app.Command("label", "Create a label for a card.")
	labelAlbumId = label.Flag("id", "The id of the album that should be created. If not provided, a card will be requested.").Uint32()
	labelCardId  = label.Flag("cardId", "Manually specify the card that the label should be printed for.").String()
	sheet        = label.Flag("sheet", "Render all labels in the database onto A4 sized sheets for batch printing. Using this ignores the cardId and id flags if set.").Bool()
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
	case add.FullCommand():
		addAlbum(*albumId)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
		defer f.Close()

		index := i * step
		if err := deezer.CreateLabelSheet(ids[index:min(index+step, len(ids))], f); err != nil {
			panic(err)
		}
	}
}

func dumpAll() {
	c, err := db.ReadAll()
	if err != nil {
		panic(err)
	}

	if len(*c) > 0 {
		fmt.Println("            ID │   AlbumId   │ PlaylistId │ Tracks ")
		fmt.Println("───────────────┼─────────────┼────────────┼────────")
	} else {
		fmt.Println("No cards found in the database...")
	}
	for _, card := range *c {
		fmt.Printf("%14v │ %11v │ %10v │ %4v \n", card.ID, *card.AlbumID, card.PlaylistID, len(card.Tracks))
	}
}

func dumpCard(cardId string) {
	if cardId == "" {
		id, err := readSingleCard()
		if err != nil {
			log.Fatal(err)
		}
		cardId = id
	}

	p, err := db.ReadCard(cardId)
	if err != nil {
		log.Error(err)
		return
	}

	if *dumpInfo {
		a, err := deezer.AlbumInfo(strconv.Itoa(*p.AlbumID))
		if err != nil {
			log.Error(err)
			return
		}
		fmt.Println(a)
	} else {
		fmt.Println(p)
	}
}

func addAlbum(id uint32) {
	a, err := deezer.AlbumInfo(fmt.Sprint(id))
	if err != nil {
		log.Error(err)
		return
	}

	var cardId string
	if *albumCardId != "" {
		cardId = *albumCardId
	} else {
		id, err := readSingleCard()
		if err != nil {
			log.Fatal(err)
		}
		cardId = id
	}
	p := sonos.FromAlbum(a, cardId)

	db.StoreCard(*p)
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

func readSingleCard() (string, error) {
	c, err := nfc.CreateReader()
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	fmt.Println("Please add a card to read...")

	for {
		select {
		case cardEvent, ok := <-c.Events():
			if !ok {
				return "", errors.New("card channel closed unexpectedly")
			}
			if cardEvent.State == nfc.Activated {
				log.Debugf("Read card %v", cardEvent.CardID)
				return cardEvent.CardID, nil
			}
		case <-time.After(20 * time.Second):
			return "", errors.New("no card found")
		}
	}
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

	reader, err := nfc.CreateReader()
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()
	lastActive := ""
	for {
		card, open := <-reader.Events()
		if !open {
			return
		}

		if card.State == nfc.Activated {
			log.Infof("Card %v activated", card.CardID)
			led.Yellow()
			play = true

			// save this so that we can fetch it later and update the state
			lastActive = card.CardID

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
			log.Infoln("Card removed...")
			if state, err := s.MediaInfo(); err == nil {
				if p, err := db.ReadCard(lastActive); err == nil {
					i, err := strconv.Atoi(state.Track)
					if err != nil {
						log.Warnf("Could not parse current track: %v", err.Error())
						i = 1
					}

					p.State = &sonos.PlaylistState{
						CurrentTrack:    i,
						CurrentPosition: state.RelTime,
					}

					if err := db.StoreCard(p); err != nil {
						log.Warn("Could not update playlist state: ", err)
					} else {
						log.Debugf("Updated card %v with state %v", lastActive, p.State)
					}
				}
			}
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
