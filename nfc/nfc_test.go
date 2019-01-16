package nfc

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWrite(t *testing.T) {
	p := Payload{
		ID: 130,
		Tracks: []Track{
			{TrackID: 80912383, Info: 0x10},
			{TrackID: 3123, Info: 0x10},
			{TrackID: 142341444, Info: 0x10},
			{TrackID: 13411355, Info: 0x10},
			{TrackID: 789789789, Info: 0x10},
			{TrackID: 4123577, Info: 0x10},
			{TrackID: 4, Info: 0x10},
			{TrackID: 651436, Info: 0x10},
			{TrackID: 789, Info: 0x40},
			{TrackID: 5148780, Info: 0x10},
			{TrackID: 4294967295, Info: 0x10},
			{TrackID: 131324, Info: 0x10},
			{TrackID: 312333, Info: 0x10},
			{TrackID: 14144, Info: 0x10},
			{TrackID: 23133341, Info: 0x10},
		},
	}

	buf := bytes.Buffer{}
	err := p.Write(&buf)
	assert.NoError(t, err)

	t.Logf("able to store %v tracks in %v bytes\n", len(p.Tracks), len(buf.Bytes()))

	other, err := NewPayload(&buf)
	assert.NoError(t, err)
	assert.Equal(t, p.ID, other.ID)
	assert.Equal(t, len(p.Tracks), len(other.Tracks))
	for i, track := range p.Tracks {
		assert.Equal(t, track.TrackID, other.Tracks[i].TrackID)
		assert.Equal(t, track.Info, other.Tracks[i].Info)
	}
}

func TestPayloadInfo(t *testing.T) {
	tests := []struct {
		name      string
		info      uint8
		assertion func(t *testing.T, tr Track)
	}{
		{
			"deezer location",
			0x00,
			func(t *testing.T, tr Track) {
				assert.Equal(t, Deezer, tr.Location())
			},
		},
		{
			"local location",
			0x40,
			func(t *testing.T, tr Track) {
				assert.Equal(t, Local, tr.Location())
			},
		},
		{
			"audiobook type",
			0x60,
			func(t *testing.T, tr Track) {
				assert.Equal(t, AudioBook, tr.Type())
			},
		},
		{
			"music type",
			0x00,
			func(t *testing.T, tr Track) {
				assert.Equal(t, Music, tr.Type())
			},
		},
		{
			"volume 1",
			0x04,
			func(t *testing.T, tr Track) {
				assert.Equal(t, 25, tr.Volume(100))
			},
		},
		{
			"volume 7",
			0x9C,
			func(t *testing.T, tr Track) {
				assert.Equal(t, 50, tr.Volume(50))
			},
		},
		{
			"volume 5",
			0x54,
			func(t *testing.T, tr Track) {
				assert.Equal(t, 6, tr.Volume(9)) // rounding causes this to return 6 even though it is 6.75
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := Track{
				tc.info,
				97812389,
			}
			tc.assertion(t, tr)
		})
	}
}
