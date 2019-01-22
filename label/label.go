package label

import (
	"encoding/json"
	"fmt"
	"github.com/fogleman/gg"
	"image"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"github.com/nfnt/resize"
)

// image size of 50x81.6mm (85.60 mm × 53.98 with 2mm margin on each side) at 600 DPI
// = 1181 x 1928 pix

const height = 1928
const width = 1181
const artSize = 900

var uriBase = "https://api.deezer.com/album"
var fontFile = "/usr/share/fonts/truetype/msttcorefonts/Comic_Sans_MS_Bold.ttf"

type content struct {
	Cover  string `json:"cover_xl"`
	Artist struct {
		Name string `json:"name"`
	} `json:"artist"`
	Title string `json:"title"`
}

func CreateLabel(albumId string, out io.Writer) error {
	c, err := getInfo(albumId)
	if err != nil {
		return fmt.Errorf("could not fetch album info: %v", err.Error())
	}

	img, err := fetchAlbumArt(c.Cover)
	if err != nil {
		return fmt.Errorf("could not fetch album art: %v", err.Error())
	}

	l := gg.NewContext(width, height)

	scaled := resize.Resize(artSize, 0, *img, resize.Lanczos3)
	l.SetRGB(1, 1, 1)
	l.Fill()
	origin := width / 2
	l.DrawImageAnchored(scaled, origin, origin, 0.5, 0.5)
	l.SetRGB(0, 0, 0)

	if err := renderString(l, strings.ToUpper(c.Artist.Name), 112, 1300); err != nil {
		return err
	}
	l.SetRGB(0.4, 0.4, 0.4)
	if err := renderString(l, strings.ToUpper(c.Title), 72, 1550); err != nil {
		return err
	}

	if err := l.EncodePNG(out); err != nil {
		return fmt.Errorf("could not render PNG: %v", err.Error())
	}
	return nil
}

func renderString(c *gg.Context, s string, size, y float64) error {
	if err := c.LoadFontFace(fontFile, size); err != nil {
		return fmt.Errorf("could not load the font: %v", err.Error())
	}
	lines := c.WordWrap(s, width - (width/10))
	for i, line := range lines {
		c.DrawStringAnchored(line, float64(width/2), y + float64(i) * size * 1.2, 0.5, 0.5)
	}
	return nil
}

func getInfo(albumId string) (*content, error) {
	u := fmt.Sprintf("%s/%s", uriBase, albumId)
	res, err := http.DefaultClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	c := new(content)
	body, _ := ioutil.ReadAll(res.Body)
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, err
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
