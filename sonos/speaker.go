package sonos

import (
	"fmt"
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/soap"
	"github.com/sirupsen/logrus"
	"strconv"
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
	uid     string
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
				root.Device.UDN[5:], // trim away the "uuid:" prefix
			}, nil
		}
	}
	return nil, fmt.Errorf("no speakers found for zone %v", name)
}

func (s *SonosSpeaker) setAVTransportToQueue() {
	in := struct {
		InstanceID         string
		CurrentURI         string
		CurrentURIMetaData string
	}{
		"0",
		fmt.Sprintf("x-rincon-queue:%v#0", s.uid),
		"",
	}

	if err := s.control.Action("SetAVTransportURI", in, nil); err != nil {
		logrus.Warn("Could not properly set the queue as the AV soure: ", err)
	}
}

// SetPlaylist clears the queue and then adds the given playlist for the speaker. Will use the order:
// * Album
// * Playlist
// * Tracks
// and use the first one that has been set. Repeat will also be set.
func (s *SonosSpeaker) SetPlaylist(playlist CardInfo) {
	s.Clear()

	if playlist.AlbumID != nil {
		s.playAlbum(*playlist.AlbumID)
	} else if playlist.PlaylistID != nil {
		s.playPlaylist(*playlist.PlaylistID)
	} else {
		logrus.Errorf("No content for playlist %v. Try to re-provision it?", playlist.ID)
	}

	s.setAVTransportToQueue()

	s.SetRepeat(true)

	if playlist.State != nil {
		logrus.Debugf("Resuming the previous state from track %v", playlist.State.CurrentTrack)
		s.Seek(playlist.State.CurrentTrack)
	}
}

func (s *SonosSpeaker) playAlbum(id uint64) {
	logrus.Debug("Queueing album ", id)
	m, err := CreateAlbumMetadata(id)
	if err != nil {
		logrus.Warn("Unable to generate DIDL: ", err)
		return
	}
	uri := fmt.Sprintf("x-rincon-cpcontainer:0004206calbum-%v", id)
	s.enqueue(uri, m)
}

func (s *SonosSpeaker) playPlaylist(id uint64) {
	logrus.Debug("Queueing playlist ", id)
	m, err := CreatePlaylistMetadata(id)
	if err != nil {
		logrus.Warn("Unable to generate DIDL: ", err)
		return
	}
	uri := fmt.Sprintf("x-rincon-cpcontainer:0006206cplaylist_spotify%%3aplaylist-%v", id)
	s.enqueue(uri, m)
}

func (s *SonosSpeaker) enqueue(uri string, m []byte) {
	in := struct {
		InstanceID                      string
		EnqueuedURI                     string
		EnqueuedURIMetaData             string
		DesiredFirstTrackNumberEnqueued string
		EnqueueAsNext                   string
	}{
		"0", uri, string(m), "0", "0",
	}
	s.control.Action("AddURIToQueue", in, nil)
}

func (s *SonosSpeaker) Seek(position int) {
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
