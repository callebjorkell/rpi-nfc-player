package deezer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
	"github.com/sirupsen/logrus"
	"image"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// image size of 50x81.6mm (85.60 mm × 53.98 with 2mm margin on each side) at 600 DPI
// = 1181 x 1928 pix

const (
	a4Width          = 4962
	a4Height         = 7014
	horizontalLabels = 3
	verticalLabels   = 3
	LabelsPerSheet   = horizontalLabels * verticalLabels

	labelHeight = 1928
	labelWidth  = 1181
	artSize     = 755
	strokeSize  = 4

	uriBase = "https://api.deezer.com/album"
	// this thing doesn't exist on the raspberry. Fix to add a proper font path if one wants to generate labels on there
	fontFile = "/usr/share/fonts/truetype/msttcorefonts/Comic_Sans_MS_Bold.ttf"
)

var defaultArt = getDefaultArt()
var colors = []string{
	"#0048BA",
	"#D3212D",
	"#32CD32",
	"#F4C2C2",
	"#8A2BE2",
	"#FF7E00",
	"#FDEE00",
}

func init() {
	rand.Seed(time.Now().Unix())
}

type Album struct {
	Id     int    `json:"id"`
	Cover  string `json:"cover_xl"`
	Artist struct {
		Name string `json:"name"`
	} `json:"artist"`
	TrackList struct {
		Data []struct {
			Id int `json:"id"`
		} `json:"data"`
	} `json:"tracks"`
	Title string `json:"title"`
}

func (a *Album) Tracks() []string {
	var tracks []string
	for _, value := range a.TrackList.Data {
		tracks = append(tracks, fmt.Sprint(value.Id))
	}
	return tracks
}

func (a Album) String() string {
	return fmt.Sprintf("ID: %v, artist: %v, title: %v, tracks: %v", a.Id, a.Artist.Name, a.Title, len(a.TrackList.Data))
}

func AlbumInfo(albumId string) (*Album, error) {
	return getInfo(albumId)
}

func CreateLabelSheet(albums []string, out io.Writer) error {
	if len(albums) > LabelsPerSheet {
		return fmt.Errorf("too many albums for a single sheet. Max: %v, got: %v", LabelsPerSheet, len(albums))
	}
	l := gg.NewContext(a4Width, a4Height) // A4 size @ 600 dpi

	baseX := (a4Width - (horizontalLabels * labelWidth)) / 2
	baseY := (a4Height - (verticalLabels * labelHeight)) / 2

	wg := sync.WaitGroup{}
	drawing := &sync.Mutex{}

	l.SetRGB(1, 1, 1)
	l.Clear()
	l.SetRGB(0, 0, 0)
	l.SetLineWidth(4)

	for index, albumId := range albums {
		wg.Add(1)
		go func(index int, albumId string) {
			defer wg.Done()
			c, _ := renderLabelContext(albumId)
			logrus.Debugf("Rendering album label for %v at index %v", albumId, index)
			x := baseX + (index % horizontalLabels * labelWidth)
			y := baseY + (index / verticalLabels * labelHeight)

			drawing.Lock()
			l.DrawImage(c.Image(), x, y)
			drawCutMark(l, x, y)
			drawCutMark(l, x, y+labelHeight)
			drawCutMark(l, x+labelWidth, y)
			drawCutMark(l, x+labelWidth, y+labelHeight)
			l.Stroke()
			drawing.Unlock()
		}(index, albumId)
	}
	wg.Wait()

	logrus.Debugln("Rendering album sheet to a PNG")
	if err := l.EncodePNG(out); err != nil {
		return fmt.Errorf("could not render PNG: %v", err.Error())
	}
	return nil
}

func CreateLabel(albumId string, out io.Writer) error {
	l, err := renderLabelContext(albumId)
	if err != nil {
		return err
	}

	logrus.Debugf("Render album %v to a PNG", albumId)
	if err := l.EncodePNG(out); err != nil {
		return fmt.Errorf("could not render PNG: %v", err.Error())
	}
	return nil
}

func drawCutMark(l *gg.Context, x, y int) {
	fx := float64(x)
	fy := float64(y)
	l.DrawLine(fx-30, fy, 30+fx, fy)
	l.DrawLine(fx, fy-30, fx, fy+30)
}

func renderLabelContext(albumId string) (*gg.Context, error) {
	c, err := getInfo(albumId)
	if err != nil {
		return nil, fmt.Errorf("could not fetch album info: %v", err.Error())
	}

	logrus.Debugf("Generating label for %v (%v - %v)", c.Id, c.Artist, c.Title)
	img := defaultArt
	if c.Cover != "" {
		img, err = fetchAlbumArt(c.Cover)
		if err != nil {
			return nil, fmt.Errorf("could not fetch album art: %v", err.Error())
		}
	} else {
		logrus.Debugln("Using the default album art")
	}

	l := gg.NewContext(labelWidth, labelHeight)

	scaled := resize.Resize(artSize, 0, *img, resize.Lanczos3)
	l.SetRGB(1, 1, 1)
	l.Clear()

	origin := labelWidth / 2
	l.DrawImageAnchored(scaled, origin, origin, 0.5, 0.5)

	frame, err := getFrame()
	if err != nil {
		return nil, err
	}
	l.DrawImage(frame, 0, 0)

	col := colors[rand.Int()%len(colors)]
	l.SetHexColor(col)
	r := regexp.MustCompile("\\(.*\\)")
	title := r.ReplaceAllString(c.Title, "")

	l.Push()
	l.DrawRectangle(0, 1200, float64(labelWidth), 465)
	l.Clip()
	if err := renderString(l, strings.ToUpper(title), 96, 1250); err != nil {
		return nil, err
	}
	l.ResetClip()
	l.Pop()

	l.SetHexColor(col + "60")
	if err := renderString(l, strings.ToUpper(c.Artist.Name), 64, 1700); err != nil {
		return nil, err
	}
	return l, nil
}

func getFrame() (image.Image, error) {
	const frames = 7
	frameName := fmt.Sprintf("img/frame%d.png", rand.Int()%frames)
	return loadImage(frameName)
}

func getDefaultArt() *image.Image {
	img, err := loadImage("img/defaultArt.png")
	if err != nil {
		logrus.Error("Could not find the default album art")
	}
	return &img
}

func loadImage(fileName string) (image.Image, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)

	return img, err
}

func renderString(c *gg.Context, s string, size, y float64) error {
	if err := c.LoadFontFace(fontFile, size); err != nil {
		return fmt.Errorf("could not load the font: %v", err.Error())
	}
	lines := c.WordWrap(s, labelWidth-(labelWidth/10))
	for i, line := range lines {
		c.Push()
		w := float64(labelWidth / 2)
		h := y + float64(i)*size*1.2

		c.SetRGB(0.2, 0.2, 0.2)
		for dy := -strokeSize; dy <= strokeSize; dy++ {
			for dx := -strokeSize; dx <= strokeSize; dx++ {
				if dx*dx+dy*dy >= strokeSize*strokeSize {
					// give it rounded corners
					continue
				}
				x := w + float64(dx)
				y := h + float64(dy)
				c.DrawStringAnchored(line, x, y, 0.5, 0.5)
			}
		}
		c.SetRGB(1, 1, 1)
		c.DrawStringAnchored(line, w, h, 0.5, 0.5)
		c.Pop()
		c.DrawStringAnchored(line, w, h, 0.5, 0.5)
	}

	return nil
}

func getInfo(albumId string) (*Album, error) {
	u := fmt.Sprintf("%s/%s", uriBase, albumId)
	res, err := http.DefaultClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	c := new(Album)
	body, _ := ioutil.ReadAll(res.Body)
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, err
	}
	if c.Artist.Name == "" && c.Title == "" {
		return c, errors.New("album info is empty")
	}
	return c, nil
}

func fetchAlbumArt(uri string) (*image.Image, error) {
	res, err := http.DefaultClient.Get(uri)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	img, _, err := image.Decode(res.Body)
	return &img, err
}
