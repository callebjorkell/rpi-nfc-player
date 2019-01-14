package nfc

// Bought NXP MIFARE Ultralight® C 144 Byte RFID card, so only have 144 bytes to play with (137 if you ask nfc tools on
// android). This means we need to get somewhat creative in the storing of data unless I resort to storing just an id
// and then maintaining some sort of database later on. Currently "creative" should work fine.
const (
	Deezer    uint8 = 0
	Local     uint8 = 1
	Music     uint8 = 0
	AudioBook uint8 = 1
)

type Payload struct {
	// The ID of the card itself, can prove useful if the database approach is needed.
	ID uint8
	// Tracks is the collection of tracks that should be played when this card is detected.
	Tracks []Track
}

type Track struct {
	// Info is a bitmap with track information following this bit layout
	// * 0-1, track location. 0 = deezer, 1 = local. 2 and 3 are reserved.
	// * 2, track type, 0 = music, 1 = audiobook
	// * 3-5, volume as a 3 bit integer. Handled like a multiplier of whatever volume the application sets as the base.
	// * 6-7, reserved.
	Info uint8
	// The ID of the track. For deezer this is a one to one mapping. The local files will need mapping
	TrackID uint32
}

func (t Track) Location() uint8 {
	return t.Info >> 6
}

func (t Track) Type() uint8 {
	return (t.Info >> 5) & 0x01
}

func (t Track) Volume() uint8 {
	return (t.Info >> 2) & 0x07
}
