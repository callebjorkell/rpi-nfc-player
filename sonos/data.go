package sonos

import (
	"encoding/json"
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
)

type TrackLocation int
type TrackType int

const (
	Deezer TrackLocation = 0
	Local  TrackLocation = 1

	Music     TrackType = 0
	AudioBook TrackType = 1 // not really used (yet?)
)

type Playlist struct {
	// The ID of the card itself
	ID string `json:"id"`
	// AlbumID contains the Deezer album ID if applicable
	AlbumID *uint64 `json:"albumId,omitempty"`
	// PlaylistID contains the Deezer playlist ID if applicable
	PlaylistID *uint64 `json:"playlistId,omitempty"`
	// Tracks is the collection of tracks that should be played when this card is detected. Is only used if the
	Tracks []Track `json:"tracks"`
	// State is the last seen state of the card. If none exists, the state will be nil.
	State *PlaylistState `json:"state,omitempty"`
}

func (p Playlist) String() string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Sprintf("ID: %v, album: %v", p.ID, p.AlbumID)
	}
	return string(b)
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

type PlaylistState struct {
	// CurrentTrack is the index of the track in the tracks array of the card.
	CurrentTrack int `json:"current_track"`
	// CurrentPosition is the position in the current track.
	CurrentPosition string `json:"current_position"`
}

func FromAlbum(album *deezer.Album, cardId string) *Playlist {
	var tracks []Track
	for _, trackId := range album.Tracks() {
		tracks = append(tracks, makeTrack(trackId))
	}
	albumId := album.Identifier
	return &Playlist{
		ID:         cardId,
		AlbumID:    &albumId,
		State:      nil,
		Tracks:     tracks,
		PlaylistID: nil,
	}
}

func FromPlaylist(p *deezer.Playlist, cardId string) *Playlist {
	var tracks []Track
	for _, trackId := range p.Tracks() {
		tracks = append(tracks, makeTrack(trackId))
	}
	playlistId := p.Identifier
	return &Playlist{
		ID:         cardId,
		PlaylistID: &playlistId,
		State:      nil,
		Tracks:     tracks,
		AlbumID:    nil,
	}
}

func makeTrack(id string) Track {
	return Track{
		ID:       fmt.Sprintf("%v%v", "tr%3A", id),
		Type:     Music,
		Location: Deezer,
		Volume:   100,
	}
}
