package nfc

import (
	"github.com/callebjorkell/rpi-nfc-player/sonos"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestReadWriteCard(t *testing.T) {
	db, err := NewDB()
	if err != nil {
		t.Fatal(err)
	}

	c := sonos.Playlist{
		ID: rand.Int(),
		Tracks: []sonos.Track{
			{
				ID:       "abeautifulid",
				Location: 1,
				Volume:   100,
				Type:     1,
			},
		},
	}

	if err := db.StoreCard(c); err != nil{
		t.Fatal(err)
	}
	defer db.DeleteCard(c.ID)

	b, err := db.ReadCard(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, c, b)
}
