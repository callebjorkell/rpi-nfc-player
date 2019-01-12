package sonos

import (
	"encoding/xml"
	"fmt"
)

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
	XMLName xml.Name
	Dc      string   `xml:"xmlns:dc,attr"`
	Upnp    string   `xml:"xmlns:upnp,attr"`
	R       string   `xml:"xmlns:r,attr"`
	Ns      string   `xml:"xmlns,attr"`
	Item    didlItem `xml:"item"`
}

func CreateMetadata(deezerId string) ([]byte, error) {
	service := "519"
	didl := didlPayload{
		XMLName: xml.Name{Local: "DIDL-Lite"},
		Dc:      "http://purl.org/dc/elements/1.1/",
		Upnp:    "urn:schemas-upnp-org:metadata-1-0/upnp/",
		R:       "urn:schemas-rinconnetworks-com:metadata-1-0/",
		Ns:      "urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/",
		Item: didlItem{
			ID:         fmt.Sprintf("00032020%v", deezerId),
			ParentID:   "-1",
			Restricted: "true",
			Title:      "",
			Class:      "object.item.audioItem.musicTrack",
			Desc: didlDesc{
				ID:        "cdudn",
				NameSpace: "urn:schemas-rinconnetworks-com:metadata-1-0/",
				Value:     fmt.Sprintf("SA_RINCON%v_X_#Svc%v-0-Token", service, service),
			},
		},
	}
	return xml.Marshal(didl)
}
