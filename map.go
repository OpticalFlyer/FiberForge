package main

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"math"
	"net/http"

	"github.com/hajimehoshi/ebiten/v2"
)

func latLngToTile(lat, lng float64, zoom int) (int, int) {
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2, float64(zoom))
	xtile := int((lng + 180.0) / 360.0 * n)
	ytile := int((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n)
	return xtile, ytile
}

func downloadTileImage(x, y, zoom int) (*ebiten.Image, error) {
	url := fmt.Sprintf("https://mt1.google.com/vt/lyrs=s&x=%d&y=%d&z=%d", x, y, zoom)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: %s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(img), nil
}
