package deezer

import (
	"fmt"
	"testing"
)

func TestAlbumSearch(t *testing.T) {
	a, err := AlbumSearch("ultra bra")

	fmt.Println(a, err)
}

func TestAlbumInfo(t *testing.T) {
	a, err := AlbumInfo("11838808")

	fmt.Println(a, err)
}

func TestCreateLabel(t *testing.T) {
	err := CreateLabel("81536992", NopWriter{})

	fmt.Println(err)
}

type NopWriter struct {
}

func (w NopWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
