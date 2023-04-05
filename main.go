package main

import (
	"fmt"
	"image/color"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

type Game struct {
	ScreenWidth  int
	ScreenHeight int
	TextBoxText  string
	LastCmdText  string
	Points       []struct{ X, Y float32 }
	Lines        [][]struct{ X, Y float32 }
	PL_activated bool
	mapTile      *ebiten.Image
}

func Initialize() (*Game, error) {
	state := &Game{}
	latitude := 35.4382223973447
	longitude := -89.82503869316511
	zoom := 19

	xtile, ytile := latLngToTile(latitude, longitude, zoom)
	img, err := downloadTileImage(xtile, ytile, zoom)
	if err != nil {
		return nil, err
	}

	// Initialize your variables here
	state.mapTile = img

	return state, nil
}

func (g *Game) Update() error {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && g.PL_activated {
		mouseX, mouseY := ebiten.CursorPosition()
		g.Points = append(g.Points, struct{ X, Y float32 }{X: float32(mouseX), Y: float32(mouseY)})
	}

	if inpututil.IsKeyJustReleased(ebiten.KeySpace) || inpututil.IsKeyJustReleased(ebiten.KeyEnter) {
		if g.PL_activated {
			g.PL_activated = false
			if len(g.Points) > 0 {
				g.Lines = append(g.Lines, g.Points)
				g.Points = nil
			}
		} else if g.TextBoxText == "PL" || g.TextBoxText == "" && g.LastCmdText == "PL" {
			g.PL_activated = true
			g.LastCmdText = "PL"
		}
		g.TextBoxText = ""
	} else {
		g.handleTextInput()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 25, 255})

	// Draw your images or graphics here
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(50, 50)
	screen.DrawImage(g.mapTile, op)

	dashLength, gapLength := float32(20), float32(40)

	// Draw completed lines
	for _, line := range g.Lines {
		numPoints := len(line)
		if numPoints > 0 {
			for i, j := 0, 1; j < numPoints; i, j = i+1, j+1 {
				textDashedLine(screen, line[i].X, line[i].Y, line[j].X, line[j].Y, dashLength, gapLength, 3, color.RGBA{255, 255, 255, 255}, "UG")
			}
		}
	}

	// Draw currently active line
	numPoints := len(g.Points)
	if numPoints > 0 {
		for i, j := 0, 1; j < numPoints; i, j = i+1, j+1 {
			textDashedLine(screen, g.Points[i].X, g.Points[i].Y, g.Points[j].X, g.Points[j].Y, dashLength, gapLength, 3, color.RGBA{255, 255, 255, 255}, "UG")
		}
		mouseX, mouseY := ebiten.CursorPosition()
		textDashedLine(screen, g.Points[numPoints-1].X, g.Points[numPoints-1].Y, float32(mouseX), float32(mouseY), dashLength, gapLength, 3, color.RGBA{255, 255, 255, 255}, "UG")
	}

	g.DrawTextbox(screen, g.ScreenWidth, g.ScreenHeight)

	// Get the current mouse position
	mouseX, mouseY := ebiten.CursorPosition()

	// Draw the crosshair at the mouse position
	if g.PL_activated {
		drawCrosshair(screen, float32(mouseX), float32(mouseY), 100, color.RGBA{255, 255, 255, 255})
	} else {
		drawSquareCrosshair(screen, float32(mouseX), float32(mouseY), 10, 100, color.RGBA{255, 255, 255, 255})
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	g.ScreenWidth = outsideWidth
	g.ScreenHeight = outsideHeight
	return outsideWidth, outsideHeight
}

func main() {
	fiberforge, err := Initialize()
	if err != nil {
		log.Fatalf("Error creating app state: %v", err)
	}

	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowTitle("CAD/GIS Experiment")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	if err := ebiten.RunGame(fiberforge); err != nil {
		fmt.Println(err)
	}
}

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
