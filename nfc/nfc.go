package nfc

import (
	"encoding/binary"
	"io"
)

// Bought NXP MIFARE Ultralight® C 144 Byte RFID card, so only have 144 bytes to play with (137 if you ask nfc tools on
// android). This means we need to get somewhat creative in the storing of data unless I resort to storing just an id
// and then maintaining some sort of database later on. Currently "creative" should work fine.
type TrackLocation int
type TrackType int

const (
	Deezer TrackLocation = 0
	Local  TrackLocation = 1
)
const (
	Music     TrackType = 0
	AudioBook TrackType = 1
)

var order = binary.BigEndian

type Payload struct {
	// The ID of the card itself, can prove useful if the database approach is needed.
	ID uint16
	// Tracks is the collection of tracks that should be played when this card is detected.
	Tracks []Track
}

type Track struct {
	// Info is a bitmap with track information following this bit layout
	// * 0-1, track location. 0 = deezer, 1 = local. 2 and 3 are reserved.
	// * 2, track type, 0 = music, 1 = audiobook
	// * 3-5, volume as a 3 bit integer.
	//        Handled like fraction (val + 1/ 8 * vol) of whatever volume the application sets as the base. Default 3.
	// * 6-7, reserved.
	Info uint8
	// The ID of the track. For deezer this is a one to one mapping. The local files will need mapping
	TrackID uint32
}

func (t Track) Location() TrackLocation {
	return TrackLocation(t.Info >> 6)
}

func (t Track) Type() TrackType {
	return TrackType(t.Info >> 5 & 0x01)
}

func (t Track) Volume(base int) int {
	multiplier := int(t.Info>>2&0x07 + 1)
	return multiplier * base / 8
}

func (p Payload) Write(w io.Writer) error {
	if err := binary.Write(w, order, p.ID); err != nil {
		return err
	}
	tracks := uint8(len(p.Tracks))
	if err := binary.Write(w, order, tracks); err != nil {
		return err
	}
	for _, t := range p.Tracks {
		if err := binary.Write(w, order, t.Info); err != nil {
			return err
		}
		if err := binary.Write(w, order, t.TrackID); err != nil {
			return err
		}
	}
	return nil
}

func NewPayload(r io.Reader) (*Payload, error) {
	p := Payload{}
	if err := binary.Read(r, order, &p.ID); err != nil {
		return nil, err
	}
	var tracks uint8
	if err := binary.Read(r, order, &tracks); err != nil {
		return nil, err
	}
	for i := 0; i < int(tracks); i++ {
		t := Track{}
		if err := binary.Read(r, order, &t.Info); err != nil {
			return nil, err
		}
		if err := binary.Read(r, order, &t.TrackID); err != nil {
			return nil, err
		}
		p.Tracks = append(p.Tracks, t)
	}

	return &p, nil
}
