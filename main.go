package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
	centerLat    float64
	centerLon    float64
	zoom         int
	tileCache    TileImageCache
}

func Initialize() (*Game, error) {
	g := &Game{}
	g.centerLat = 35.156072
	g.centerLon = -90.051911
	g.zoom = 5

	g.tileCache = NewTileImageCache()
	return g, nil
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

	// Zooming
	_, scrollY := ebiten.Wheel()

	// Set the scroll threshold
	scrollThreshold := 0.2

	// Zoom in or out based on the scrollY value
	if scrollY > scrollThreshold {
		g.zoom++
	} else if scrollY < -scrollThreshold {
		g.zoom--
	}

	// Clamp the zoom level within a valid range (e.g., 0-20 for Google Maps)
	g.zoom = int(math.Max(0, math.Min(21, float64(g.zoom))))

	// Panning
	tileWidth := 360 / math.Pow(2, float64(g.zoom))
	panSpeed := tileWidth * 0.5

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.centerLon -= panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.centerLon += panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.centerLat += panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.centerLat -= panSpeed
	}

	// Clamp the coordinates to valid values
	g.centerLat = math.Min(math.Max(g.centerLat, -85.05112878), 85.05112878)
	g.centerLon = math.Min(math.Max(g.centerLon, -180), 180)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 25, 255})

	// Calculate the center pixel coordinates of the game window
	centerX := g.ScreenWidth / 2
	centerY := g.ScreenHeight / 2

	// Get the tile coordinates and pixel coordinates of the center point
	tileX, tileY := latLngToTile(g.centerLat, g.centerLon, g.zoom)
	pixelX, pixelY := latLngToTilePixel(g.centerLat, g.centerLon, g.zoom)

	// Calculate the tile offset to center the pixel coordinates in the game window
	tileOffsetX := centerX - pixelX
	tileOffsetY := centerY - pixelY

	// Calculate the number of tiles needed to cover the window horizontally and vertically
	numHorizontalTiles := (g.ScreenWidth / 256) + 2
	numVerticalTiles := (g.ScreenHeight / 256) + 2

	// Calculate the starting tile coordinates based on the center tile
	startTileX := tileX - numHorizontalTiles/2
	startTileY := tileY - numVerticalTiles/2

	// Draw the tiles within the window
	for i := 0; i < numHorizontalTiles; i++ {
		for j := 0; j < numVerticalTiles; j++ {
			op := &ebiten.DrawImageOptions{}
			tileOffsetXForTile := tileOffsetX + ((i - numHorizontalTiles/2) * 256)
			tileOffsetYForTile := tileOffsetY + ((j - numVerticalTiles/2) * 256)
			op.GeoM.Translate(float64(tileOffsetXForTile), float64(tileOffsetYForTile))
			drawTile(screen, &g.tileCache, startTileX+i, startTileY+j, g.zoom, op)
		}
	}

	// Draw Lines

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

	debugString := fmt.Sprintf("Zoom: %d", g.zoom)
	ebitenutil.DebugPrint(screen, debugString)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	g.ScreenWidth = outsideWidth
	g.ScreenHeight = outsideHeight
	return outsideWidth, outsideHeight
}

func main() {
	fiberforge, err := Initialize()
	if err != nil {
		log.Fatalf("Error initializing program: %v", err)
	}

	startWorkerPool(10)

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
