package sonos

import (
	"encoding/xml"
	"fmt"
)

const deezerServiceId = "SA_RINCON519_X_#Svc519-0-Token"

type didlDesc struct {
	ID        string `xml:"id,attr"`
	NameSpace string `xml:"nameSpace,attr"`
	Value     string `xml:",chardata"`
}

type didlItem struct {
	ID         string   `xml:"id,attr"`
	ParentID   string   `xml:"parentID,attr"`
	Restricted string   `xml:"restricted,attr"`
	Title      string   `xml:"dc:title"`
	Class      string   `xml:"upnp:class"`
	Desc       didlDesc `xml:"desc"`
}

type didlPayload struct {
	XMLName   xml.Name
	Dc        string   `xml:"xmlns:dc,attr"`
	Upnp      string   `xml:"xmlns:upnp,attr"`
	R         string   `xml:"xmlns:r,attr"`
	Ns        string   `xml:"xmlns,attr"`
	Item      *didlItem `xml:"item"`
	Container *didlItem `xml:"container"`
}


const (
 	trackClass = "object.item.audioItem.musicTrack"
 	albumClass = "object.container.album"
 	playlistClass = "object.container.playlistContainer"
)
func CreateTrackMetadata(deezerId string) ([]byte, error) {
	return createDidl(fmt.Sprintf("00032020%v", deezerId), trackClass)
}

func CreateAlbumMetadata(deezerId uint64) ([]byte, error) {
	return createDidl(fmt.Sprintf("0004206calbum-%v", deezerId), albumClass)
}

func CreatePlaylistMetadata(deezerId uint64) ([]byte, error) {
	return createDidl(fmt.Sprintf("0006206cplaylist_spotify%%3aplaylist-%v", deezerId), playlistClass)
}

func createDidl(ID, class string) ([]byte, error) {
	item := &didlItem{
		ID:         ID,
		ParentID:   "-1",
		Restricted: "true",
		Class:      class,
		Desc: didlDesc{
			ID:        "cdudn",
			NameSpace: "urn:schemas-rinconnetworks-com:metadata-1-0/",
			Value:     deezerServiceId,
		},
	}
	didl := didlPayload{
		XMLName: xml.Name{Local: "DIDL-Lite"},
		Dc:      "http://purl.org/dc/elements/1.1/",
		Upnp:    "urn:schemas-upnp-org:metadata-1-0/upnp/",
		R:       "urn:schemas-rinconnetworks-com:metadata-1-0/",
		Ns:      "urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/",
	}

	if class == trackClass {
		didl.Item = item
	} else {
		didl.Container = item
	}
	return xml.Marshal(didl)
}
