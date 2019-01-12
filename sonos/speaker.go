package sonos

import (
	"fmt"
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/soap"
	"log"
)

type SonosSpeaker struct {
	control *service
	content *service
	info    *service
	name    string
}

func New(name string) (*SonosSpeaker, error) {
	d, err := goupnp.DiscoverDevices("urn:schemas-upnp-org:device:ZonePlayer:1")
	if err != nil {
		log.Fatal(err)
	}
	for _, dev := range d {
		root, err := goupnp.DeviceByURL(dev.Location)
		if err != nil {
			log.Printf("Could not retrieve %v, speaker went away?", dev.Location)
		}

		s, err := getService(root, "DeviceProperties")
		if err != nil {
			return nil, err
		}

		out := struct {
			CurrentZoneName string
		}{}
		if err := s.Action("GetZoneAttributes", nil, &out); err != nil {
			return nil, err
		}
		if out.CurrentZoneName == name {
			control, err := getService(root, "AVTransport")
			if err != nil {
				return nil, err
			}
			content, err := getService(root, "ContentDirectory")
			if err != nil {
				return nil, err
			}
			return &SonosSpeaker{
				control,
				content,
				s,
				name,
			}, nil
		}
	}
	return nil, fmt.Errorf("no speakers found for zone %v", name)
}

func (s *SonosSpeaker) PlayDeezer(track string) {
	metadata, err := CreateMetadata(track)
	if err != nil {
		panic(err)
	}
	uri := fmt.Sprintf("x-sonos-http:tr:%v.mp3", track)
	in := struct {
		UpdateID                        string
		NumberOfURIs                    string
		EnqueuedURIs                    string
		EnqueuedURIsMetaData            string
		DesiredFirstTrackNumberEnqueued string
		EnqueueAsNext                   string
		ObjectID                        string
		InstanceID                      string
	}{
		"0", "1", uri, string(metadata), "1", "0", "Q:0", "0",
	}
	s.control.Action("AddMultipleURIsToQueue", in, nil)
}

func (s *SonosSpeaker) Play() {
	in := struct {
		InstanceID string
		Speed      string
	}{
		"0",
		"1",
	}
	s.control.Action("Play", in, nil)
}

func (s *SonosSpeaker) Name() string {
	return s.name
}

func (s *SonosSpeaker) MediaInfo() {
	in := struct {
		InstanceID string
	}{
		"0",
	}
	s.control.Action("GetPositionInfo", &in, nil)
}

func (s *SonosSpeaker) Search(name string) {
	in := struct {
		ObjectID       string
		BrowseFlag     string
		Filter         string
		StartingIndex  string
		RequestedCount string
		SortCriteria   string
	}{
		name,
		"BrowseDirectChildren",
		"*",
		"0",
		"100",
		"",
	}
	s.content.Action("Browse", &in, nil)
}

func getService(dev *goupnp.RootDevice, id string) (*service, error) {
	namespace := fmt.Sprintf("urn:schemas-upnp-org:service:%v:1", id)
	s := dev.Device.FindService(namespace)
	if len(s) > 1 {
		return nil, fmt.Errorf("got %v services instead of the expected maximum of 1", len(s))
	}
	if len(s) == 0 {
		return nil, nil
	}

	return &service{
		SOAPClient: s[0].NewSOAPClient(),
		namespace:  namespace,
	}, nil
}

type service struct {
	*soap.SOAPClient
	namespace string
}

func (s *service) Action(name string, in interface{}, out interface{}) error {
	return s.SOAPClient.PerformAction(s.namespace, name, in, out)
}
