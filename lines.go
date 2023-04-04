package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

func dashedLine(screen *ebiten.Image, x0, y0, x1, y1, dashLength, gapLength, strokeWidth float32, clr color.Color) {
	dx := x1 - x0
	dy := y1 - y0
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	angle := math.Atan2(float64(dy), float64(dx))

	interval := dashLength + gapLength
	dashes := int(length / interval)
	segment := interval * float32(dashes)
	bookend := (length - segment) / 2

	//fmt.Printf("Length: %f, Bookend: %f, Segment: %f\n", length, bookend, segment)

	startX := x0
	startY := y0
	endX := x0 + bookend*float32(math.Cos(angle))
	endY := y0 + bookend*float32(math.Sin(angle))

	vector.StrokeLine(screen, startX, startY, endX, endY, strokeWidth, color.RGBA{0, 0, 255, 255}, false)

	for i := float32(bookend); i < segment; i += interval {
		startX := x0 + i*float32(math.Cos(angle))
		startY := y0 + i*float32(math.Sin(angle))
		endX := x0 + (i+dashLength)*float32(math.Cos(angle))
		endY := y0 + (i+dashLength)*float32(math.Sin(angle))

		vector.StrokeLine(screen, startX, startY, endX, endY, strokeWidth, clr, false)
	}

	startX = x0 + (bookend+segment)*float32(math.Cos(angle))
	startY = y0 + (bookend+segment)*float32(math.Sin(angle))
	endX = x0 + length*float32(math.Cos(angle))
	endY = y0 + length*float32(math.Sin(angle))

	vector.StrokeLine(screen, startX, startY, endX, endY, strokeWidth, color.RGBA{255, 0, 0, 255}, false)
}

func textDashedLine(screen *ebiten.Image, x0, y0, x1, y1, dashLength, gapLength, strokeWidth float32, clr color.Color, textStr string) {
	dx := x1 - x0
	dy := y1 - y0
	length := math.Sqrt(float64(dx*dx + dy*dy))
	angle := math.Atan2(float64(dy), float64(dx))
	interval := float64(dashLength + gapLength)
	offset := float64(dashLength + gapLength/2)

	dashes := int(length / interval)
	segment := interval * float64(dashes)
	bookend := (length - segment) / 2

	dashedLine(screen, x0, y0, x1, y1, dashLength, gapLength, strokeWidth, clr)

	gapCount := int(length / interval)
	for i := 0; i < gapCount; i++ {
		gapCenterX := float64(x0) + float64(i)*interval*math.Cos(angle) + (offset+bookend)*math.Cos(angle)
		gapCenterY := float64(y0) + float64(i)*interval*math.Sin(angle) + (offset+bookend)*math.Sin(angle)

		rotatedText(screen, gapCenterX, gapCenterY, angle, clr, textStr, -5)
	}

	rotatedText(screen, float64(x0)+length/2*math.Cos(angle), float64(y0)+length/2*math.Sin(angle), angle, clr, "SEGMENT LABEL", -20)
}

func rotatedText(screen *ebiten.Image, x, y, angle float64, clr color.Color, textStr string, offset float64) {
	if angle > 1.57 || angle < -1.57 {
		angle = math.Pi + angle
	}
	fontFace := basicfont.Face7x13
	textWidth := font.MeasureString(fontFace, textStr).Ceil()
	textHeight := fontFace.Metrics().Ascent.Ceil()

	textImage := ebiten.NewImage(textWidth, textHeight)
	text.Draw(textImage, textStr, fontFace, 0, 10, clr)

	textOpts := &ebiten.DrawImageOptions{}
	textOpts.GeoM.Translate(-float64(textWidth/2), offset)
	textOpts.GeoM.Rotate(angle)
	textOpts.GeoM.Translate(x, y)

	screen.DrawImage(textImage, textOpts)
}
