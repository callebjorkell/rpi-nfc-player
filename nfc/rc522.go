package nfc

// MFRC522 spec can be found here: https://www.nxp.com/docs/en/data-sheet/MFRC522.pdf
// MIFARE Ultralight C spec: https://www.nxp.com/docs/en/data-sheet/MF0ICU2.pdf

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ecc1/spi"
	"github.com/jdevelop/golang-rpi-extras/rf522/commands"
	"github.com/jdevelop/gpio"
	rpio "github.com/jdevelop/gpio/rpi"
	log "github.com/sirupsen/logrus"
)

var NoCardErr = errors.New("no card detected")
var stateLock sync.Mutex
var active bool

type CardState int
type CardEvent struct {
	CardID string
	State  CardState
}

const (
	Activated   CardState = 0
	Deactivated CardState = 1
)

type rfid struct {
	ResetPin    gpio.Pin
	antennaGain int
	MaxSpeedHz  int
	spiDev      *spi.Device
}

type CardReader interface {
	io.Closer
	Events() <-chan CardEvent
}

type cardReader struct {
	events <-chan CardEvent
	rfid   *rfid
	stop   chan interface{}
}

func (c cardReader) Events() <-chan CardEvent {
	return c.events
}

func (c cardReader) Close() error {
	close(c.stop)
	defer func() {
		stateLock.Lock()
		active = false
		stateLock.Unlock()
	}()
	return c.rfid.Close()
}

func CreateReader() (CardReader, error) {
	stateLock.Lock()
	if active {
		return nil, errors.New("reader already in use")
	} else {
		active = true
	}
	stateLock.Unlock()

	// the IRQ pin is actually connected on the board, but I could never get it to work properly. So now we're
	// polling instead. Might come back to it at some point if I feel like losing another couple of days.
	reader, err := makeRFID(0, 0, 100000, 22, 18)
	if err != nil {
		log.Fatal(err)
	}

	events := make(chan CardEvent, 10)
	c := cardReader{
		rfid:   reader,
		events: events,
		stop:   make(chan interface{}),
	}

	go func() {
		defer close(events)
		lastConfirmedId, lastSeenId := "", ""
		debounceIndex := 0
		for {
			select {
			case <-c.stop:
				log.Debugln("CardReader stopped. Returning.")
				return
			case <-time.After(150 * time.Millisecond):
				// just do another loop
			}

			id, err := reader.readCardId()
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
					events <- CardEvent{State: Deactivated, CardID: ""}
				} else {
					log.Debugf("Sending activation event for card %v", id)
					events <- CardEvent{State: Activated, CardID: id}

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
	return c, nil
}

func (r *rfid) readCardId() (string, error) {
	if err := r.init(); err != nil {
		return "", err
	}
	if _, err := r.request(); err != nil {
		return "", err
	}
	data, err := r.antiColl()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func makeRFID(busId, deviceId, maxSpeed, resetPin, irqPin int) (device *rfid, err error) {
	spiDev, err := spi.Open(fmt.Sprintf("/dev/spidev%d.%d", busId, deviceId), maxSpeed, 0)

	if err != nil {
		return
	}

	err = spiDev.SetLSBFirst(false)
	if err != nil {
		spiDev.Close()
		return
	}

	err = spiDev.SetBitsPerWord(8)

	if err != nil {
		spiDev.Close()
		return
	}

	dev := &rfid{
		spiDev:      spiDev,
		MaxSpeedHz:  maxSpeed,
		antennaGain: 7,
	}

	pin, err := rpio.OpenPin(resetPin, gpio.ModeOutput)
	if err != nil {
		spiDev.Close()
		return
	}
	dev.ResetPin = pin
	dev.ResetPin.Set()

	pin, err = rpio.OpenPin(irqPin, gpio.ModeInput)
	if err != nil {
		spiDev.Close()
		return
	}

	err = dev.init()

	device = dev

	return
}

func (r *rfid) init() (err error) {
	err = r.reset()
	if err != nil {
		return
	}
	err = r.devWrite(0x2A, 0x8D)
	if err != nil {
		return
	}
	err = r.devWrite(0x2B, 0x3E)
	if err != nil {
		return
	}
	err = r.devWrite(0x2D, 30)
	if err != nil {
		return
	}
	err = r.devWrite(0x2C, 0)
	if err != nil {
		return
	}
	err = r.devWrite(0x15, 0x40)
	if err != nil {
		return
	}
	err = r.devWrite(0x11, 0x3D)
	if err != nil {
		return
	}
	err = r.devWrite(0x26, byte(r.antennaGain)<<4)
	if err != nil {
		return
	}
	err = r.setAntenna(true)
	if err != nil {
		return
	}
	return
}

func (r *rfid) Close() error {
	return r.spiDev.Close()
}

func (r *rfid) writeSpiData(dataIn []byte) (out []byte, err error) {
	out = make([]byte, len(dataIn))
	copy(out, dataIn)
	err = r.spiDev.Transfer(out)
	return
}

func printBytes(data []byte) (res string) {
	res = "["
	for _, v := range data[0 : len(data)-1] {
		res = res + fmt.Sprintf("%02x, ", byte(v))
	}
	res = res + fmt.Sprintf("%02x", data[len(data)-1])
	res = res + "]"
	return
}

func (r *rfid) devWrite(address int, data byte) (err error) {
	newData := [2]byte{(byte(address) << 1) & 0x7E, data}
	_, err = r.writeSpiData(newData[:])
	return
}

func (r *rfid) devRead(address int) (result byte, err error) {
	data := [2]byte{((byte(address) << 1) & 0x7E) | 0x80, 0}
	rb, err := r.writeSpiData(data[:])
	result = rb[1]
	return
}

func (r *rfid) setBitmask(address, mask int) (err error) {
	current, err := r.devRead(address)
	if err != nil {
		return
	}
	err = r.devWrite(address, current|byte(mask))
	return
}

func (r *rfid) clearBitmask(address, mask int) (err error) {
	current, err := r.devRead(address)
	if err != nil {
		return
	}
	err = r.devWrite(address, current&^byte(mask))
	return

}

func (r *rfid) setAntennaGain(gain int) {
	if 0 <= gain && gain <= 7 {
		r.antennaGain = gain
	}
}

func (r *rfid) reset() (err error) {
	err = r.devWrite(commands.CommandReg, commands.PCD_RESETPHASE)
	return
}

func (r *rfid) setAntenna(state bool) (err error) {
	if state {
		current, err := r.devRead(commands.TxControlReg)
		if err != nil {
			return err
		}
		if current&0x03 == 0 {
			err = r.setBitmask(commands.TxControlReg, 0x03)
		}
	} else {
		err = r.clearBitmask(commands.TxControlReg, 0x03)
	}
	return
}

func (r *rfid) cardWrite(command byte, data []byte) (backData []byte, backLength int, err error) {
	backData = make([]byte, 0)
	backLength = -1
	irqEn := byte(0x00)
	irqWait := byte(0x00)

	switch command {
	case commands.PCD_AUTHENT:
		irqEn = 0x12
		irqWait = 0x10
	case commands.PCD_TRANSCEIVE:
		irqEn = 0x77
		irqWait = 0x30
	}

	r.devWrite(commands.CommIEnReg, irqEn|0x80)
	r.clearBitmask(commands.CommIrqReg, 0x80)
	r.setBitmask(commands.FIFOLevelReg, 0x80)
	r.devWrite(commands.CommandReg, commands.PCD_IDLE)

	for _, v := range data {
		r.devWrite(commands.FIFODataReg, v)
	}

	r.devWrite(commands.CommandReg, command)

	if command == commands.PCD_TRANSCEIVE {
		r.setBitmask(commands.BitFramingReg, 0x80)
	}

	i := 2000
	n := byte(0)

	for ; i > 0; i-- {
		n, err = r.devRead(commands.CommIrqReg)
		if err != nil {
			return
		}
		if n&(irqWait|1) != 0 {
			break
		}
	}

	r.clearBitmask(commands.BitFramingReg, 0x80)

	if i == 0 {
		err = errors.New("can't read data after 2000 loops")
		return
	}

	if d, err1 := r.devRead(commands.ErrorReg); err1 != nil || d&0x1B != 0 {
		err = err1
		log.Error("E2")
		return
	}

	if n&irqEn&0x01 == 1 {
		err = errors.New("IRQ error")
		return
	}

	if command == commands.PCD_TRANSCEIVE {
		n, err = r.devRead(commands.FIFOLevelReg)
		if err != nil {
			return
		}
		lastBits, err1 := r.devRead(commands.ControlReg)
		if err1 != nil {
			err = err1
			return
		}
		lastBits = lastBits & 0x07
		if lastBits != 0 {
			backLength = (int(n)-1)*8 + int(lastBits)
		} else {
			backLength = int(n) * 8
		}

		if n == 0 {
			n = 1
		}

		if n > 16 {
			n = 16
		}

		for i := byte(0); i < n; i++ {
			byteVal, err1 := r.devRead(commands.FIFODataReg)
			if err1 != nil {
				err = err1
				return
			}
			backData = append(backData, byteVal)
		}

	}

	return
}

func (r *rfid) request() (backBits int, err error) {
	backBits = 0
	err = r.devWrite(commands.BitFramingReg, 0x07)
	if err != nil {
		return
	}

	_, backBits, err = r.cardWrite(commands.PCD_TRANSCEIVE, []byte{0x26}[:])
	if err != nil {
		return -1, NoCardErr
	}
	if backBits != 0x10 {
		err = errors.New(fmt.Sprintf("wrong number of bits %d", backBits))
	}

	return
}

func (r *rfid) antiColl() ([]byte, error) {
	var backData, backData2 []byte
	var uid = make([]byte, 7) // used when the UID is a 7 byte UID.

	err := r.devWrite(commands.BitFramingReg, 0x00)

	backData, _, err = r.cardWrite(commands.PCD_TRANSCEIVE, []byte{0x93, 0x20}[:])

	if err != nil {
		log.Error("Card write ", err)
		return nil, err
	}

	if len(backData) != 5 {
		return nil, errors.New(fmt.Sprintf("Back data expected 5, actual %d", len(backData)))
	}

	crc := byte(0)

	for _, v := range backData[:4] {
		crc = crc ^ v
	}

	if crc != backData[4] {
		return nil, errors.New(fmt.Sprintf("CRC mismatch, expected %02x actual %02x", crc, backData[4]))
	}
	if backData[0] != 0x88 {
		return backData[:4], nil
	}

	copy(uid, backData[1:4])
	log.Debugf("UID currently %v", hex.EncodeToString(uid))
	log.Debug("cascade l2 required!")
	cmd := []byte{0x93, 0x70, backData[0], backData[1], backData[2], backData[3], backData[4]}
	cascadeCRC, err := r.crc(cmd)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	buf := make([]byte, 9)
	copy(buf, cmd)
	buf[7] = cascadeCRC[0]
	buf[8] = cascadeCRC[1]

	backDataT, _, err := r.cardWrite(commands.PCD_TRANSCEIVE, buf)
	if err != nil {
		return nil, err
	}

	if backDataT[0] != 0x04 {
		return nil, errors.New(fmt.Sprintf("Unexpected L2 Anticoll response: %02x", backDataT[0]))
	}

	backData2, _, err = r.cardWrite(commands.PCD_TRANSCEIVE, []byte{0x95, 0x20}[:])

	if err != nil {
		log.Error("Card write ", err)
		return nil, err
	}

	if len(backData2) != 5 {
		return nil, errors.New(fmt.Sprintf("Back data expected 5, actual %d", len(backData2)))
	}

	crc = byte(0)
	for _, v := range backData2[:4] {
		crc = crc ^ v
	}

	log.Debug("Back data ", printBytes(backData2), ", CRC ", printBytes([]byte{crc}))
	if crc != backData2[4] {
		return nil, errors.New(fmt.Sprintf("CRC mismatch, expected %02x actual %02x", crc, backData[4]))
	}
	copy(uid[3:], backData2[:5])
	log.Debugf("Found uid %v", hex.EncodeToString(uid))
	return uid, nil
}

func (r *rfid) crc(inData []byte) (res []byte, err error) {
	res = []byte{0, 0}
	err = r.clearBitmask(commands.DivIrqReg, 0x04)
	if err != nil {
		return
	}
	err = r.setBitmask(commands.FIFOLevelReg, 0x80)
	if err != nil {
		return
	}
	for _, v := range inData {
		r.devWrite(commands.FIFODataReg, v)
	}
	err = r.devWrite(commands.CommandReg, commands.PCD_CALCCRC)
	if err != nil {
		return
	}
	for i := byte(0xFF); i > 0; i-- {
		n, err1 := r.devRead(commands.DivIrqReg)
		if err1 != nil {
			err = err1
			return
		}
		if n&0x04 > 0 {
			break
		}
	}
	lsb, err := r.devRead(commands.CRCResultRegL)
	if err != nil {
		return
	}
	res[0] = lsb

	msb, err := r.devRead(commands.CRCResultRegM)
	if err != nil {
		return
	}
	res[1] = msb
	return
}
