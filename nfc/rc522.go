package nfc

import (
	"log"
	"periph.io/x/periph/conn/spi/spireg"
	"periph.io/x/periph/experimental/devices/mfrc522"
	"periph.io/x/periph/host"
	"periph.io/x/periph/host/rpi"
)

// MFRC522 spec can be found here: https://www.nxp.com/docs/en/data-sheet/MFRC522.pdf
// MIFARE Ultralight C spec: https://www.nxp.com/docs/en/data-sheet/MF0ICU2.pdf

func init() {
	// init periph
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}
}

func openSPI() (*mfrc522.Dev, error) {
	// open the first SPI port
	p, err := spireg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer p.Close()

	return mfrc522.NewSPI(p, rpi.P1_22, rpi.P1_18)
}

func ReadCardID() (int, error) {
	spi, err := openSPI()
	if err != nil {
		log.Fatal(err)
	}
	defer spi.Halt()

	return -1, nil
}