package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	Points []struct{ X, Y float64 }
}

func (g *Game) Update() error {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		fmt.Printf("Left button released at (%d, %d)\n", mouseX, mouseY)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)

	mouseX, mouseY := ebiten.CursorPosition()
	textDashedLine(screen, 200, 20, float32(mouseX), float32(mouseY), 20, 40, 3, color.RGBA{0, 0, 0, 255}, "UG")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowSize(1024, 768)
	ebiten.SetWindowTitle("Dashed Line Experiment")
	//ebiten.SetWindowResizable(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(&Game{}); err != nil {
		fmt.Println(err)
	}
}
