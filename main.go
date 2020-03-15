package main

import (
	"errors"
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/nfc"
	"github.com/callebjorkell/rpi-nfc-player/sonos"
	"github.com/callebjorkell/rpi-nfc-player/ui"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var db = nfc.GetDB()

var (
	app     = kingpin.New("rpi-nfc-player", "Music player that plays Deezer albums on a Sonos speaker with the help of NFC cards, a Raspberry Pi and some buttons.")
	debug   = app.Flag("debug", "Turn on debug logging.").Bool()
	start   = app.Command("start", "Start the music player and start listening for NFC cards.")
	speaker = start.Flag("speaker", "The name of the speaker that the player should control.").Required().String()
	refresh = start.Flag("refresh", "Refresh playlist/album content from deezer with regular intervals.").Default("false").Bool()

	add           = app.Command("add", "Construct and add a new playlist to a card.")
	addAlbumId    = add.Flag("albumId", "The ID of the album that should be added.").Uint64()
	addPlaylistId = add.Flag("playlistId", "The ID of the playlist that should be added.").Uint64()
	addCardId     = add.Flag("cardId", "Manually specify the card id to be used.").String()

	dump       = app.Command("dump", "Read a card and dump all the available information onto standard out.")
	dumpCardId = dump.Flag("cardId", "Manually specify the card id to be used.").String()
	dumpInfo   = dump.Flag("albumInfo", "Dump information about the album the card points to instead of the data on the card.").Bool()
	dumpList   = dump.Flag("list", "Dump a short list of all the cards in the database").Bool()

	search       = app.Command("search", "Search for albums on deezer")
	searchString = search.Arg("query", "The string to search on.").Required().String()

	label           = app.Command("label", "Create a label for a card.")
	labelAlbumId    = label.Flag("albumId", "The id of the album that should be created. If not provided, a card will be requested.").Uint64()
	labelPlaylistId = label.Flag("playlistId", "The id of the playlist that should be created. If not provided, a card will be requested.").Uint64()
	labelCardId     = label.Flag("cardId", "Manually specify the card that the label should be printed for.").String()
	sheet           = label.Flag("sheet", "Render all labels in the database onto A4 sized sheets for batch printing. Using this ignores the cardId and id flags if set.").Bool()
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
		if *addAlbumId != 0 {
			storeAlbum(*addAlbumId, *addCardId)
		} else if *addPlaylistId != 0 {
			storePlaylist(*addPlaylistId, *addCardId)
		} else {
			kingpin.FatalUsage("One of albumid or playlistid must be specified")
		}
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

func refreshCards() {
	log.Debug("Starting card refresh loop")
	for {
		log.Info("Refreshing card entries")
		entries, err := db.ReadAll()
		if err != nil {
			log.Error("Encountered an error, will retry: ", err)
			<- time.After(10 * time.Minute)
			continue
		}

		for _, e := range *entries {
			if e.AlbumID != nil {
				log.Debug("Refreshing album ", *e.AlbumID)
				storeAlbum(*e.AlbumID, e.ID)
			} else if e.PlaylistID != nil {
				log.Debug("Refreshing playlist ", *e.PlaylistID)
				storePlaylist(*e.PlaylistID, e.ID)
			} else {
				log.Warn("Cannot refresh data for ", e)
			}

			//don't spam the APIs
			<- time.After(250 * time.Millisecond)
		}
		<- time.After(24 * time.Hour)
	}
}

func startServer() {
	s, err := sonos.New(*speaker)
	if *refresh {
		go refreshCards()
	}
	if err != nil {
		log.Fatal(err)
	}
	tiger := ui.InitTiger()
	buttons := ui.InitButtons()
	led := ui.GetColorLED()

	checkTiger := tigerCheck(tiger, led)
	checkTiger()
	play := false
	playSync := &sync.Mutex{}

	go func() {
		for {
			select {
			case b, ok := <-buttons:
				if !ok {
					log.Error("Button channel has closed. Stopping event loop.")
					return
				}

				playSync.Lock()
				handleButton(&b, play, tiger, led, s)
				playSync.Unlock()
			case <-time.After(time.Second):
				// allow the scheduler to run
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

		playSync.Lock()
		play = isPlaying(&card)
		playSync.Unlock()

		handleCard(&card, lastActive, led, s)

		// should not need to lock here, since the state is not written other than above in this same loop.
		if play {
			lastActive = card.CardID
		} else {
			checkTiger()
		}
	}
}

func isPlaying(event *nfc.CardEvent) bool {
	return event.State == nfc.Activated
}

func handleCard(card *nfc.CardEvent, lastActive string, led *ui.ColorLed, speaker *sonos.SonosSpeaker) {
	if card.State == nfc.Activated {
		log.Infof("Card %v activated", card.CardID)
		led.Purple()

		p, err := db.ReadCard(card.CardID)
		if err != nil {
			log.Errorln(err)
			return
		}
		speaker.SetPlaylist(p)
		// apparently this returns before the player is ready sometimes
		time.Sleep(750 * time.Millisecond)

		speaker.Play()

		led.Green()
	} else {
		log.Infoln("Card removed...")
		if state, err := speaker.MediaInfo(); err == nil {
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

				if err := db.StoreCard(&p); err != nil {
					log.Warn("Could not update playlist state: ", err)
				} else {
					log.Debugf("Updated card %v with state %v", lastActive, p.State)
				}
			}
		}
		speaker.Pause()
		led.Off()
	}
}

func handleButton(b *ui.ButtonEvent, playing bool, tiger *ui.Tiger, led *ui.ColorLed, speaker *sonos.SonosSpeaker) {
	log.Debugln(b)
	switch b.Button {
	case ui.TigerSwitch:
		if b.Pressed {
			if !playing {
				tiger.On()
				led.Red()
			}
		} else {
			tiger.Off()
			if playing {
				led.Green()
			} else {
				led.Off()
			}
		}
	case ui.Red:
		if b.Pressed && playing {
			led.Yellow()
			speaker.Previous()
			time.Sleep(400 * time.Millisecond)
			led.Green()
		}
	case ui.Blue:
		if b.Pressed && playing {
			led.Cyan()
			speaker.Next()
			time.Sleep(400 * time.Millisecond)
			led.Green()
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
