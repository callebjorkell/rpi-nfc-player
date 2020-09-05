package sonos

import (
	"fmt"
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/soap"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

/*
 * I would probably have given up on trying to figure out most of this stuff if it wasn't for the excellent PHP library
 * made by Craig Duncan (https://github.com/duncan3dc/sonos). This served as a blueprint for what to pass and where
 * to make the sonos speakers do my bidding :)
 */

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
		logrus.Fatal(err)
	}
	logrus.Debugf("Inspecting %v devices", len(d))
	for _, dev := range d {
		root, err := goupnp.DeviceByURL(dev.Location)
		if err != nil {
			logrus.Errorf("Could not retrieve %v, speaker went away?", dev.Location)
			continue
		}
		logrus.Debugf("Checking device: %v", root.Device.FriendlyName)

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
	start := time.Now()
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

	s.SetRepeat(true)

	if playlist.State != nil {
		logrus.Debugf("Resuming the previous state from track %v", playlist.State.CurrentTrack)
		s.SetTrack(playlist.State.CurrentTrack)
	}
	logrus.Infof("Added %v tracks to the playlist in %v", len(playlist.Tracks), time.Now().Sub(start))
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

func (s *SonosSpeaker) SetRepeat(repeat bool) {
	mode := "NORMAL"
	if repeat {
		mode = "REPEAT_ALL"
	}
	in := struct {
		InstanceID  string
		NewPlayMode string
	}{
		"0",
		mode, // or NORMAL
	}

	s.control.Action("SetPlayMode", in, nil)
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
	if err := s.control.Action("AddMultipleURIsToQueue", in, nil); err != nil {
		logrus.Warn("Could not queue the playlist: ", err)
	}
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
