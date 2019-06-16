package nfc

// MFRC522 spec can be found here: https://www.nxp.com/docs/en/data-sheet/MFRC522.pdf
// MIFARE Ultralight C spec: https://www.nxp.com/docs/en/data-sheet/MF0ICU2.pdf

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ecc1/spi"
	"github.com/jdevelop/golang-rpi-extras/rf522/commands"
	"github.com/jdevelop/gpio"
	rpio "github.com/jdevelop/gpio/rpi"
	"github.com/sirupsen/logrus"
)

var NoCardErr = errors.New("no card detected")

type RFID struct {
	ResetPin    gpio.Pin
	IrqPin      gpio.Pin
	antennaGain int
	MaxSpeedHz  int
	spiDev      *spi.Device
	stop        chan interface{}
}

func (rfid *RFID) ReadCardID() (string, error) {
	if err := rfid.Init(); err != nil {
		return "", err
	}
	if _, err := rfid.Request(); err != nil {
		return "", err
	}
	data, err := rfid.AntiColl()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func MakeRFID(busId, deviceId, maxSpeed, resetPin, irqPin int) (device *RFID, err error) {

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

	dev := &RFID{
		spiDev:      spiDev,
		MaxSpeedHz:  maxSpeed,
		antennaGain: 7,
		stop:        make(chan interface{}, 1),
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
	dev.IrqPin = pin
	dev.IrqPin.PullUp()

	err = dev.Init()

	device = dev

	return
}

func (r *RFID) Init() (err error) {
	err = r.Reset()
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
	err = r.SetAntenna(true)
	if err != nil {
		return
	}
	return
}

func (r *RFID) Close() error {
	r.stop <- true
	close(r.stop)
	return r.spiDev.Close()
}

func (r *RFID) writeSpiData(dataIn []byte) (out []byte, err error) {
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

func (r *RFID) devWrite(address int, data byte) (err error) {
	newData := [2]byte{(byte(address) << 1) & 0x7E, data}
	_, err = r.writeSpiData(newData[:])
	return
}

func (r *RFID) devRead(address int) (result byte, err error) {
	data := [2]byte{((byte(address) << 1) & 0x7E) | 0x80, 0}
	rb, err := r.writeSpiData(data[:])
	result = rb[1]
	return
}

func (r *RFID) setBitmask(address, mask int) (err error) {
	current, err := r.devRead(address)
	if err != nil {
		return
	}
	err = r.devWrite(address, current|byte(mask))
	return
}

func (r *RFID) clearBitmask(address, mask int) (err error) {
	current, err := r.devRead(address)
	if err != nil {
		return
	}
	err = r.devWrite(address, current&^byte(mask))
	return

}

func (r *RFID) SetAntennaGain(gain int) {
	if 0 <= gain && gain <= 7 {
		r.antennaGain = gain
	}
}

func (r *RFID) Reset() (err error) {
	err = r.devWrite(commands.CommandReg, commands.PCD_RESETPHASE)
	return
}

func (r *RFID) SetAntenna(state bool) (err error) {
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

func (r *RFID) cardWrite(command byte, data []byte) (backData []byte, backLength int, err error) {
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
		logrus.Error("E2")
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

func (r *RFID) Request() (backBits int, err error) {
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

func (r *RFID) Wait() (err error) {
	irqChannel := make(chan bool)
	r.IrqPin.BeginWatch(gpio.EdgeFalling, func() {
		defer func() {
			if recover() != nil {
				err = errors.New("panic")
			}
		}()
		irqChannel <- true
	})

	defer func() {
		r.IrqPin.EndWatch()
		close(irqChannel)
	}()

	err = r.Init()
	if err != nil {
		return
	}
	err = r.devWrite(commands.CommIrqReg, 0x00)
	if err != nil {
		return
	}
	err = r.devWrite(commands.CommIEnReg, 0xA0)
	if err != nil {
		return
	}
	logrus.SetLevel(logrus.ErrorLevel)

interruptLoop:
	for {
		err = r.devWrite(commands.FIFODataReg, 0x26)
		if err != nil {
			return
		}
		err = r.devWrite(commands.CommandReg, 0x0C)
		if err != nil {
			return
		}
		err = r.devWrite(commands.BitFramingReg, 0x87)
		if err != nil {
			return
		}
		select {
		case <-r.stop:
			return errors.New("stop signal")
		case _ = <-irqChannel:
			logrus.Debugln("Interrupt!")
			break interruptLoop
		case <-time.After(100 * time.Millisecond):
			// do nothing
		}
	}
	return
}

func (r *RFID) AntiColl() ([]byte, error) {
	var backData, backData2 []byte
	var uid = make([]byte, 7) // used when the UID is a 7 byte UID.

	err := r.devWrite(commands.BitFramingReg, 0x00)

	backData, _, err = r.cardWrite(commands.PCD_TRANSCEIVE, []byte{0x93, 0x20}[:])

	if err != nil {
		logrus.Error("Card write ", err)
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
	logrus.Debugf("UID currently %v", hex.EncodeToString(uid))
	logrus.Debug("cascade l2 required!")
	cmd := []byte{0x93, 0x70, backData[0], backData[1], backData[2], backData[3], backData[4]}
	cascadeCRC, err := r.CRC(cmd)
	if err != nil {
		logrus.Warn(err)
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
		logrus.Error("Card write ", err)
		return nil, err
	}

	if len(backData2) != 5 {
		return nil, errors.New(fmt.Sprintf("Back data expected 5, actual %d", len(backData2)))
	}

	crc = byte(0)
	for _, v := range backData2[:4] {
		crc = crc ^ v
	}

	logrus.Debug("Back data ", printBytes(backData2), ", CRC ", printBytes([]byte{crc}))
	if crc != backData2[4] {
		return nil, errors.New(fmt.Sprintf("CRC mismatch, expected %02x actual %02x", crc, backData[4]))
	}
	copy(uid[3:], backData2[:5])
	logrus.Debugf("Found uid %v", hex.EncodeToString(uid))
	return uid, nil
}

func (r *RFID) CRC(inData []byte) (res []byte, err error) {
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

func (r *RFID) StopCrypto() (err error) {
	err = r.clearBitmask(commands.Status2Reg, 0x08)
	return
}
