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
	"strings"
	"time"
)

// image size of 50x81.6mm (85.60 mm × 53.98 with 2mm margin on each side) at 600 DPI
// = 1181 x 1928 pix

const height = 1928
const width = 1181
const artSize = 755
const strokeSize = 4

const uriBase = "https://api.deezer.com/album"
const fontFile = "/usr/share/fonts/truetype/msttcorefonts/Comic_Sans_MS_Bold.ttf"

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

func init() {
	rand.Seed(time.Now().Unix())
}

func AlbumInfo(albumId string) (*Album, error) {
	return getInfo(albumId)
}

func CreateLabel(albumId string, out io.Writer) error {
	c, err := getInfo(albumId)
	if err != nil {
		return fmt.Errorf("could not fetch album info: %v", err.Error())
	}

	logrus.Debugf("Generating label for %v (%v - %v)", c.Id, c.Artist, c.Title)
	img := defaultArt
	if c.Cover != "" {
		img, err = fetchAlbumArt(c.Cover)
		if err != nil {
			return fmt.Errorf("could not fetch album art: %v", err.Error())
		}
	} else {
		logrus.Debugln("Using the default album art")
	}

	l := gg.NewContext(width, height)

	scaled := resize.Resize(artSize, 0, *img, resize.Lanczos3)
	l.SetRGB(1, 1, 1)
	l.Clear()

	origin := width / 2
	l.DrawImageAnchored(scaled, origin, origin, 0.5, 0.5)

	frame, err := getFrame()
	if err != nil {
		return err
	}
	l.DrawImage(frame, 0, 0)

	col := colors[rand.Int()%len(colors)]
	l.SetHexColor(col)
	if err := renderString(l, strings.ToUpper(c.Title), 96, 1250); err != nil {
		return err
	}

	l.SetHexColor(col + "60")
	if err := renderString(l, strings.ToUpper(c.Artist.Name), 64, 1700); err != nil {
		return err
	}

	logrus.Debugln("Render album %v to a PNG", c.Id)
	if err := l.EncodePNG(out); err != nil {
		return fmt.Errorf("could not render PNG: %v", err.Error())
	}
	return nil
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
	lines := c.WordWrap(s, width-(width/10))
	for i, line := range lines {
		c.Push()
		w := float64(width / 2)
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
