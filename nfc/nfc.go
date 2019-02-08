package nfc

import (
	"encoding/binary"
)

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

type Card struct {
	// The ID of the card itself
	ID int
	// Tracks is the collection of tracks that should be played when this card is detected.
	Tracks []Track `json:"tracks"`
	// State is the last seen state of the card. If none exists, the state will be nil. If the state is stale (card has
	// changed since last state save) it will also be nilled.
	State *State `json:"state,omitempty"`
}

type Track struct {
	// Location is the location (source) of the track.
	Location TrackLocation `json:"location"`
	// Type is the type of track
	Type TrackType `json:"type"`
	// Volume is the percentage of volume that should be used.
	Volume int `json:"volume"`
	// TrackID is the ID of the track. The format for this depends on the location.
	ID string `json:"id"`
}

type State struct {
	// CurrentTrack is the index of the track in the tracks array of the card.
	CurrentTrack int `json:"current_track"`
	// CurrentPosition is the position in seconds in the current track.
	CurrentPosition int `json:"current_position"`
}

func (t Track) TrackLocation() TrackLocation {
	return t.Location
}

func (t Track) TrackType() TrackType {
	return t.Type
}

func (t Track) TrackVolume(base int) int {
	return t.Volume * base / 100
}