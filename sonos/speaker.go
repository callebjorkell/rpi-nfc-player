package sonos

import (
	"fmt"
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/soap"
	"github.com/sirupsen/logrus"
	"log"
	"strconv"
	"strings"
)

type SonosSpeaker struct {
	control *service
	content *service
	info    *service
	name    string
}

type State struct {
	Track         string
	TrackDuration string
	RelTime       string
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

func (s *SonosSpeaker) SetPlaylist(playlist Playlist) {
	s.Clear()

	// chunk the queue in 16 item chunks to allow them to be added to sonos without errors.
	pLen := len(playlist.Tracks)
	const cSize = 16

	for i := 0; i < pLen; i += cSize {
		l := i + cSize
		if l > pLen {
			l = pLen
		}
		s.addChunk(playlist.Tracks[i:l], i+1)
	}

	if playlist.State != nil {
		logrus.Debugf("Resuming the previous state from track %v", playlist.State.CurrentTrack)
		s.SetTrack(playlist.State.CurrentTrack)
	}
}

func (s *SonosSpeaker) SetTrack(position int) {
	in := struct {
		InstanceID string
		Unit       string
		Target     string
	}{
		"0",
		"TRACK_NR",
		strconv.Itoa(position),
	}

	s.control.Action("Seek", in, nil)
}

// addChunk adds a chunk of tracks to the sonos speaker. Note that if this goes over 16 in size, there are probably going to be problems.
func (s *SonosSpeaker) addChunk(tracks []Track, index int) {
	if len(tracks) == 0 {
		return
	}

	uri := strings.Builder{}
	metadata := strings.Builder{}
	for i, track := range tracks {
		if track.Location != Deezer {
			panic("Only deezer is supported now")
		}
		m, err := CreateMetadata(track.ID)
		if err != nil {
			panic(err)
		}

		u := fmt.Sprintf("x-sonos-http:tr:%v.mp3", track.ID)

		if i != 0 {
			metadata.WriteString(" ")
			uri.WriteString(" ")
		}

		metadata.Write(m)
		uri.WriteString(u)
	}

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
		"0", strconv.Itoa(len(tracks)), uri.String(), metadata.String(), strconv.Itoa(index), "0", "Q:0", "0",
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

func (s *SonosSpeaker) Clear() {
	s.simpleCommand("RemoveAllTracksFromQueue")
}

func (s *SonosSpeaker) Previous() {
	s.simpleCommand("Previous")
}

func (s *SonosSpeaker) Next() {
	s.simpleCommand("Next")
}

func (s *SonosSpeaker) Pause() {
	s.simpleCommand("Pause")
}

func (s *SonosSpeaker) simpleCommand(action string) {
	in := struct {
		InstanceID string
	}{
		"0",
	}
	s.control.Action(action, in, nil)
}

func (s *SonosSpeaker) Name() string {
	return s.name
}

func (s *SonosSpeaker) MediaInfo() (State, error) {
	in := struct {
		InstanceID string
	}{
		"0",
	}
	//<u:GetPositionInfoResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><Track>1</Track><TrackDuration>0:04:21</TrackDuration><TrackMetaData>&lt;DIDL-Lite xmlns:dc=&quot;http://purl.org/dc/elements/1.1/&quot; xmlns:upnp=&quot;urn:schemas-upnp-org:metadata-1-0/upnp/&quot; xmlns:r=&quot;urn:schemas-rinconnetworks-com:metadata-1-0/&quot; xmlns=&quot;urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/&quot;&gt;&lt;item id=&quot;-1&quot; parentID=&quot;-1&quot; restricted=&quot;true&quot;&gt;&lt;res protocolInfo=&quot;sonos.com-http:*:audio/mpeg:*&quot; duration=&quot;0:04:21&quot;&gt;x-sonos-http:tr%3a63534071.mp3?sid=2&amp;amp;flags=8224&amp;amp;sn=2&lt;/res&gt;&lt;r:streamContent&gt;&lt;/r:streamContent&gt;&lt;upnp:albumArtURI&gt;/getaa?s=1&amp;amp;u=x-sonos-http%3atr%253a63534071.mp3%3fsid%3d2%26flags%3d8224%26sn%3d2&lt;/upnp:albumArtURI&gt;&lt;dc:title&gt;Weatherman&lt;/dc:title&gt;&lt;upnp:class&gt;object.item.audioItem.musicTrack&lt;/upnp:class&gt;&lt;dc:creator&gt;Dead Sara&lt;/dc:creator&gt;&lt;upnp:album&gt;Dead Sara&lt;/upnp:album&gt;&lt;/item&gt;&lt;/DIDL-Lite&gt;</TrackMetaData><TrackURI>x-sonos-http:tr%3a63534071.mp3?sid=2&amp;flags=8224&amp;sn=2</TrackURI><RelTime>0:00:50</RelTime><AbsTime>NOT_IMPLEMENTED</AbsTime><RelCount>2147483647</RelCount><AbsCount>2147483647</AbsCount></u:GetPositionInfoResponse>
	out := State{}
	err := s.control.Action("GetPositionInfo", &in, &out)
	return out, err
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
