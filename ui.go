package main

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

func (g *Game) handleTextInput() {
	// Create a buffer to store the input characters
	buffer := make([]rune, 0, 16)

	// Get input characters
	buffer = ebiten.AppendInputChars(buffer)

	// Process printable characters
	for _, char := range buffer {
		g.TextBoxText += strings.ToUpper(string(char))
		g.TextBoxText = strings.TrimSpace(g.TextBoxText)
	}

	// Process backspace key
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(g.TextBoxText) > 0 {
			g.TextBoxText = g.TextBoxText[:len(g.TextBoxText)-1]
		}
	}
}

func (g *Game) drawText(screen *ebiten.Image, x, y float64, clr color.Color, textStr string) {
	if len(g.TextBoxText) > 0 {
		fontFace := basicfont.Face7x13
		textWidth := font.MeasureString(fontFace, textStr).Ceil()
		textHeight := fontFace.Metrics().Ascent.Ceil()

		textImage := ebiten.NewImage(textWidth, textHeight)
		text.Draw(textImage, textStr, fontFace, 0, 10, clr)

		textOpts := &ebiten.DrawImageOptions{}
		textOpts.GeoM.Translate(x, y)

		screen.DrawImage(textImage, textOpts)
	}
}

func (g *Game) DrawTextbox(screen *ebiten.Image, screenWidth, screenHeight int) {
	// Set the textbox dimensions and position
	boxWidth := int(0.8 * float64(screenWidth))
	if boxWidth > 800 {
		boxWidth = 800
	}
	boxHeight := 24
	boxX := (screenWidth - boxWidth) / 2
	boxY := screenHeight - boxHeight - 50

	// Create a new image for the textbox background
	bgColor := color.RGBA{50, 50, 50, 200}
	bgImg := ebiten.NewImage(boxWidth, boxHeight)
	bgImg.Fill(bgColor)

	// Draw the textbox background onto the screen
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(boxX), float64(boxY))
	screen.DrawImage(bgImg, op)

	textX := float64(boxX) + 10
	textY := float64(boxY) + float64(boxHeight)/2 - 5

	textColor := color.White
	g.drawText(screen, textX, textY, textColor, g.TextBoxText)
}

func drawCrosshair(screen *ebiten.Image, x, y, size float32, clr color.Color) {
	halfSize := size / 2
	vector.StrokeLine(screen, float32(x)-halfSize, float32(y), float32(x)+halfSize, float32(y), 1, clr, false)
	vector.StrokeLine(screen, float32(x), float32(y)-halfSize, float32(x), float32(y)+halfSize, 1, clr, false)
}

func drawSquareCrosshair(screen *ebiten.Image, x, y, squareSize, crosshairSize float32, clr color.Color) {
	halfSquareSize := squareSize / 2
	halfCrosshairSize := crosshairSize / 2

	// Draw the square
	vector.StrokeLine(screen, x-halfSquareSize, y-halfSquareSize, x+halfSquareSize, y-halfSquareSize, 1, clr, false)
	vector.StrokeLine(screen, x+halfSquareSize, y-halfSquareSize, x+halfSquareSize, y+halfSquareSize, 1, clr, false)
	vector.StrokeLine(screen, x+halfSquareSize, y+halfSquareSize, x-halfSquareSize, y+halfSquareSize, 1, clr, false)
	vector.StrokeLine(screen, x-halfSquareSize, y+halfSquareSize, x-halfSquareSize, y-halfSquareSize, 1, clr, false)

	// Draw the crosshair lines
	vector.StrokeLine(screen, x-halfCrosshairSize, y, x-halfSquareSize, y, 1, clr, false)
	vector.StrokeLine(screen, x+halfSquareSize, y, x+halfCrosshairSize, y, 1, clr, false)
	vector.StrokeLine(screen, x, y-halfCrosshairSize, x, y-halfSquareSize, 1, clr, false)
	vector.StrokeLine(screen, x, y+halfSquareSize, x, y+halfCrosshairSize, 1, clr, false)

}
