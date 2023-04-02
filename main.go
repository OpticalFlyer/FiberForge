package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Points []struct{ X, Y float32 }
}

func (g *Game) Update() error {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		g.Points = append(g.Points, struct{ X, Y float32 }{X: float32(mouseX), Y: float32(mouseY)})
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	dashLength, gapLength := float32(20), float32(40)

	numPoints := len(g.Points)
	if numPoints > 0 {
		for i, j := 0, 1; j < numPoints; i, j = i+1, j+1 {
			textDashedLine(screen, g.Points[i].X, g.Points[i].Y, g.Points[j].X, g.Points[j].Y, dashLength, gapLength, 3, color.RGBA{255, 255, 255, 255}, "UG")
		}
		mouseX, mouseY := ebiten.CursorPosition()
		textDashedLine(screen, g.Points[numPoints-1].X, g.Points[numPoints-1].Y, float32(mouseX), float32(mouseY), dashLength, gapLength, 3, color.RGBA{255, 255, 255, 255}, "UG")
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowTitle("Dashed Line Experiment")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(&Game{}); err != nil {
		fmt.Println(err)
	}
}
