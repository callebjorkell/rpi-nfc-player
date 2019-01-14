package nfc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
				assert.Equal(t, uint8(1), tr.Volume())
			},
		},
		{
			"volume 7",
			0x9C,
			func(t *testing.T, tr Track) {
				assert.Equal(t, uint8(7), tr.Volume())
			},
		},
		{
			"volume 5",
			0x54,
			func(t *testing.T, tr Track) {
				assert.Equal(t, uint8(5), tr.Volume())
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
