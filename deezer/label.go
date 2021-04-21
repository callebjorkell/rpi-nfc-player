package deezer

import (
	"fmt"
	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
	"github.com/sirupsen/logrus"
	"image"
	"io"
	"math"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
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

// image size of 50x81.6mm (85.60 mm × 53.98 with 2mm margin on each side) at 600/2 DPI
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

	lineWidth  = 4
	strokeSize = 4

	// the scale at which the above measurements should be rendered/drawn. Smaller scaling saves time.
	renderScale = .75

	// this thing doesn't exist on the raspberry. Fix to add a proper font path if one wants to generate labels on there
	fontFile = "/usr/share/fonts/truetype/msttcorefonts/Comic_Sans_MS_Bold.ttf"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func scaleI(size int) int {
	return int(math.RoundToEven(renderScale * float64(size)))
}

func scaleF(size float64) float64 {
	return math.RoundToEven(renderScale * size)
}

func CreateLabelSheet(trackLists []Playable, out io.Writer) error {
	if len(trackLists) > LabelsPerSheet {
		return fmt.Errorf("too many albums for a single sheet. Max: %v, got: %v", LabelsPerSheet, len(trackLists))
	}
	l := gg.NewContext(scaleI(a4Width), scaleI(a4Height)) // A4 size @ 600 dpi

	width := scaleI(labelWidth)
	height := scaleI(labelHeight)
	baseX := (scaleI(a4Width) - (horizontalLabels * width)) / 2
	baseY := (scaleI(a4Height) - (verticalLabels * height)) / 2

	wg := sync.WaitGroup{}
	drawing := &sync.Mutex{}

	l.SetRGB(1, 1, 1)
	l.Clear()
	l.SetRGB(0, 0, 0)
	l.SetLineWidth(scaleF(lineWidth))

	for index, p := range trackLists {
		wg.Add(1)
		go func(index int, p Playable) {
			defer wg.Done()
			c, _ := renderLabelContext(p)
			logrus.Debugf("Rendering label for %v at index %v", p.Id(), index)
			x := baseX + (index % horizontalLabels * width)
			y := baseY + (index / verticalLabels * height)

			drawing.Lock()
			l.DrawImage(c.Image(), x, y)
			drawCutMark(l, x, y)
			drawCutMark(l, x, y+height)
			drawCutMark(l, x+width, y)
			drawCutMark(l, x+width, y+height)
			l.Stroke()
			drawing.Unlock()
		}(index, p)
	}
	wg.Wait()

	logrus.Debugln("Rendering album sheet to a PNG")
	if err := l.EncodePNG(out); err != nil {
		return fmt.Errorf("could not render PNG: %v", err.Error())
	}
	return nil
}

func CreateLabel(t Playable, out io.Writer) error {
	l, err := renderLabelContext(t)
	if err != nil {
		return err
	}

	logrus.Debugf("Render album %v to a PNG", t.Id())
	if err := l.EncodePNG(out); err != nil {
		return fmt.Errorf("could not render PNG: %v", err.Error())
	}
	return nil
}

func drawCutMark(l *gg.Context, x, y int) {
	fx := float64(x)
	fy := float64(y)
	length := scaleF(30)
	l.DrawLine(fx-length, fy, length+fx, fy)
	l.DrawLine(fx, fy-length, fx, fy+length)
}

func renderLabelContext(t Playable) (*gg.Context, error) {
	logrus.Debugf("Generating label for %v (%v)", t.Id(), t.FullTitle())
	img := t.CoverArt()

	width := scaleI(labelWidth)
	height := scaleI(labelHeight)
	l := gg.NewContext(width, height)

	scaled := resize.Resize(uint(scaleI(artSize)), 0, *img, resize.Lanczos3)
	l.SetRGB(1, 1, 1)
	l.Clear()

	origin := width / 2
	l.DrawImageAnchored(scaled, origin, origin, 0.5, 0.5)

	frame, err := getFrame()
	if err != nil {
		return nil, err
	}
	l.DrawImage(frame, 0, 0)

	col := colors[rand.Int()%len(colors)]
	l.SetHexColor(col)
	r := regexp.MustCompile("\\(.*\\)")
	title := r.ReplaceAllString(t.Title(), "")

	l.Push()
	l.DrawRectangle(0, scaleF(1200), float64(width), scaleF(465))
	l.Clip()
	if err := renderString(l, strings.ToUpper(title), scaleF(96), scaleF(1250)); err != nil {
		return nil, err
	}
	l.ResetClip()
	l.Pop()

	l.SetHexColor(col + "60")
	if err := renderString(l, strings.ToUpper(t.Artist()), scaleF(64), scaleF(1700)); err != nil {
		return nil, err
	}
	return l, nil
}

func getFrame() (image.Image, error) {
	const frames = 7
	frameName := fmt.Sprintf("img/frame%d.png", rand.Int()%frames)
	frame, err := loadImage(frameName)
	if err != nil {
		return nil, err
	}

	width := scaleI(frame.Bounds().Dx())
	return resize.Resize(uint(width), 0, frame, resize.Lanczos3), nil
}

func getDefaultArt() *image.Image {
	img, err := loadImage("img/defaultArt.png")
	if err != nil {
		logrus.Error("Could not find the default album art")
	}
	width := scaleI(img.Bounds().Dx())
	sized := resize.Resize(uint(width), 0, img, resize.Lanczos3)

	return &sized
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

	width := scaleF(labelWidth)
	stroke := scaleF(strokeSize)

	lines := c.WordWrap(s, width-(width/10))
	for i, line := range lines {
		c.Push()
		w := width / 2
		h := y + float64(i)*size*1.2

		c.SetRGB(0.2, 0.2, 0.2)
		for dy := -stroke; dy <= stroke; dy++ {
			for dx := -stroke; dx <= stroke; dx++ {
				if dx*dx+dy*dy >= stroke*stroke {
					// give it rounded corners
					continue
				}
				x := w + dx
				y := h + dy
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
