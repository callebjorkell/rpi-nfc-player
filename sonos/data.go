package sonos

import (
	"encoding/json"
	"fmt"
	"github.com/callebjorkell/rpi-nfc-player/deezer"
)

type TrackLocation int
type TrackType int

type CardInfo struct {
	// The ID of the card itself
	ID string `json:"id"`
	// AlbumID contains the Deezer album ID if applicable
	AlbumID *uint64 `json:"albumId,omitempty"`
	// PlaylistID contains the Deezer playlist ID if applicable
	PlaylistID *uint64 `json:"playlistId,omitempty"`
	// State is the last seen state of the card. If none exists, the state will be nil.
	State *CardStatus `json:"state,omitempty"`
}

func (p CardInfo) String() string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Sprintf("ID: %v, album: %v", p.ID, p.AlbumID)
	}
	return string(b)
}

type CardStatus struct {
	// CurrentTrack is the index of the track in the tracks array of the card.
	CurrentTrack int `json:"current_track"`
	// CurrentPosition is the position in the current track.
	CurrentPosition string `json:"current_position"`
}

func FromAlbum(album *deezer.Album, cardId string) *CardInfo {
	albumId := album.Identifier
	return &CardInfo{
		ID:         cardId,
		AlbumID:    &albumId,
		State:      nil,
		PlaylistID: nil,
	}
}

func FromPlaylist(p *deezer.Playlist, cardId string) *CardInfo {
	playlistId := p.Identifier
	return &CardInfo{
		ID:         cardId,
		PlaylistID: &playlistId,
		State:      nil,
		AlbumID:    nil,
	}
}