package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/lukeroth/gdal"
)

func (g *Game) LoadGeoTIFF(path string) error {
	geoTiff, err := gdal.Open(path, gdal.ReadOnly)
	if err != nil {
		return err
	}
	g.geoTiff = geoTiff

	sizeX := g.geoTiff.RasterXSize()
	sizeY := g.geoTiff.RasterYSize()
	img := image.NewRGBA(image.Rect(0, 0, sizeX, sizeY))

	for bandIndex := 1; bandIndex <= g.geoTiff.RasterCount(); bandIndex++ {
		band := g.geoTiff.RasterBand(bandIndex)
		buffer := make([]uint8, sizeX*sizeY)
		err := band.IO(gdal.Read, 0, 0, sizeX, sizeY, buffer, sizeX, sizeY, 0, 0)
		if err != nil {
			return err
		}

		// Set the channel for each pixel in the image
		for y := 0; y < sizeY; y++ {
			for x := 0; x < sizeX; x++ {
				img.SetRGBA(x, y, color.RGBA{
					R: buffer[y*sizeX+x],
					G: buffer[y*sizeX+x],
					B: buffer[y*sizeX+x],
					A: 255,
				})
			}
		}
	}

	g.geoTiffImage = ebiten.NewImageFromImage(img)

	return nil
}

func (g *Game) DrawGeoTiff(screen *ebiten.Image) {
	geoTransform := g.geoTiff.GeoTransform()
	topLeftLon, topLeftLat := geoTransform[0], geoTransform[3]
	pixelWidth, pixelHeight := geoTransform[1], -geoTransform[5]
	cols := g.geoTiff.RasterXSize()
	rows := g.geoTiff.RasterYSize()
	bottomRightLon := topLeftLon + (pixelWidth * float64(cols))
	bottomRightLat := topLeftLat + (pixelHeight * float64(rows))

	topLeftX, topLeftY := latLngToScreenCoords(topLeftLat, topLeftLon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)
	bottomRightX, bottomRightY := latLngToScreenCoords(bottomRightLat, bottomRightLon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)

	//geoTiffImage := ebiten.NewImageFromImage(geoTiffImage)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale((float64(bottomRightX)-float64(topLeftX))/float64(cols), (float64(bottomRightY)-float64(topLeftY))/float64(rows))
	op.GeoM.Translate(float64(topLeftX), float64(topLeftY))
	screen.DrawImage(g.geoTiffImage, op)
}
