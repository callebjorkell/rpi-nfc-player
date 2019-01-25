package nfc

import (
	"encoding/hex"
	"log"
	"periph.io/x/periph/conn/spi/spireg"
	"periph.io/x/periph/experimental/devices/mfrc522"
	"periph.io/x/periph/experimental/devices/mfrc522/commands"
	"periph.io/x/periph/host"
	"periph.io/x/periph/host/rpi"
	"time"
)

const DefaultKey = "FFFFFFFFFFFF"

func ReadCard() {
	// init periph
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// open the first SPI port
	p, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer p.Close()

	rfid, err := mfrc522.NewSPI(p, rpi.P1_22, rpi.P1_18)
	if err != nil {
		log.Fatal(err)
	}
	defer rfid.Halt()

	hexKey, _ := hex.DecodeString(DefaultKey)
	var key [6]byte
	copy(key[:], hexKey)

	timedOut := false
	cb := make(chan []byte)
	go func() {
		log.Printf("Started %s", rfid.String())

		for {
			// Trying to read data from sector 1 block 0
			data, err := rfid.ReadCard(10*time.Second, byte(commands.PICC_AUTHENT1B), 1, 0, key)

			// If main thread timed out just exiting.
			if timedOut {
				log.Println("quitting")
				return
			}

			// Some devices tend to send wrong data while RFID chip is already detected
			// but still "too far" from a receiver.
			// Especially some cheap CN clones which you can find on GearBest, AliExpress, etc.
			// This will suppress such errors.
			if err != nil {
				log.Println("bad shit happened:",err)
				continue
			}

			cb <- data
		}
	}()

	for {
		select {
		case <-time.After(10 * time.Second):
			timedOut = true
			return
		case data := <-cb:
			log.Printf("Read some weird data: %v\n", data)
			return
		}
	}
}
