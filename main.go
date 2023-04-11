package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/lukeroth/gdal"
)

type Game struct {
	ScreenWidth    int
	ScreenHeight   int
	basemap        string
	TextBoxText    string
	LastCmdText    string
	Points         []struct{ Lat, Lon, Dist float64 }
	Lines          [][]struct{ Lat, Lon, Dist float64 }
	PL_activated   bool
	centerLat      float64
	centerLon      float64
	zoom           int
	tileCache      TileImageCache
	panning        bool
	previousMouseX int
	previousMouseY int
	panStartMouseX int
	panStartMouseY int
	panStartLat    float64
	panStartLon    float64
	gps            *GPS
	geoTiff        gdal.Dataset
	geoTiffImage   *ebiten.Image
}

func Initialize() (*Game, error) {
	g := &Game{}
	g.centerLat = 35.156072
	g.centerLon = -90.051911
	g.zoom = 5
	g.basemap = OSM

	g.tileCache = NewTileImageCache()

	g.gps = NewGPS()

	// Load the GeoTIFF file
	//g.LoadGeoTIFF("Memphis SEC.tif")

	return g, nil
}

func (g *Game) Update() error {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && g.PL_activated {
		mouseX, mouseY := ebiten.CursorPosition()
		lat, lon := screenCoordsToLatLng(mouseX, mouseY, g)
		if len(g.Points) > 0 {
			prevPoint := len(g.Points) - 1
			dist := haversine(g.Points[prevPoint].Lat, g.Points[prevPoint].Lon, lat, lon, EarthRadiusFT)
			g.Points = append(g.Points, struct{ Lat, Lon, Dist float64 }{Lat: lat, Lon: lon, Dist: dist})
		} else {
			g.Points = append(g.Points, struct{ Lat, Lon, Dist float64 }{Lat: lat, Lon: lon, Dist: 0.0})
		}
		//fmt.Printf("Added Point: %f, %f\n", lat, lon)
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
		} else if g.TextBoxText == "STARTGPS" {
			if !g.gps.running {
				g.gps.StartGPS() // Call StartGPS on the GPS instance
			}
			g.TextBoxText = ""
		} else if g.TextBoxText == "STOPGPS" {
			if g.gps.running {
				g.gps.StopGPS() // Call StopGPS on the GPS instance
			}
			g.TextBoxText = ""
		} else if g.TextBoxText == "GOOGLEHYBRID" {
			g.basemap = GOOGLEHYBRID
			g.tileCache = NewTileImageCache()
		} else if g.TextBoxText == "GOOGLEAERIAL" {
			g.basemap = GOOGLEAERIAL
			g.tileCache = NewTileImageCache()
		} else if g.TextBoxText == "BINGHYBRID" {
			g.basemap = BINGHYBRID
			g.tileCache = NewTileImageCache()
		} else if g.TextBoxText == "BINGAERIAL" {
			g.basemap = BINGAERIAL
			g.tileCache = NewTileImageCache()
		} else if g.TextBoxText == "OSM" {
			g.basemap = OSM
			g.tileCache = NewTileImageCache()
		}
		g.TextBoxText = ""
	} else {
		g.handleTextInput()
	}

	// Determine if line segment is clicked
	/*if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && !g.PL_activated {
		mouseX, mouseY := ebiten.CursorPosition()
		//lat, lon := screenCoordsToLatLng(mouseX, mouseY, g)

		threshold := 5.0 // Adjust the threshold distance as needed

		for _, line := range g.Lines {
			for i := 0; i < len(line)-1; i++ {
				startX, startY := latLngToScreenCoords(line[i].Lat, line[i].Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)
				endX, endY := latLngToScreenCoords(line[i+1].Lat, line[i+1].Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)
				distance := pointLineSegmentDistance(float64(mouseX), float64(mouseY), float64(startX), float64(startY), float64(endX), float64(endY))
				if distance <= threshold {
					// The user clicked on a line segment
					fmt.Printf("Clicked on line segment %d\n", i)
				}
			}
		}
	}*/

	// Zoomers...
	_, scrollY := ebiten.Wheel()
	scrollThreshold := 0.2
	mouseX, mouseY := ebiten.CursorPosition()

	if scrollY > scrollThreshold || scrollY < -scrollThreshold {
		// Calculate the world coordinates before zooming
		preZoomLat, preZoomLon := screenCoordsToLatLng(mouseX, mouseY, g)

		if scrollY > scrollThreshold {
			g.zoom++
		} else if scrollY < -scrollThreshold {
			g.zoom--
		}

		// Clamp the zoom level within a valid range (e.g., 0-21 for Google Maps)
		g.zoom = int(math.Max(0, math.Min(21, float64(g.zoom))))

		// Calculate the world coordinates after zooming
		postZoomLat, postZoomLon := screenCoordsToLatLng(mouseX, mouseY, g)

		// Adjust the center latitude and longitude to keep the world coordinates at the mouse position locked
		g.centerLat += preZoomLat - postZoomLat
		g.centerLon += preZoomLon - postZoomLon
	}

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

	// Panning with middle mouse button
	mouseX, mouseY = ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if !g.panning {
			g.panning = true
			g.panStartMouseX, g.panStartMouseY = mouseX, mouseY
			g.panStartLat, g.panStartLon = screenCoordsToLatLng(mouseX, mouseY, g)
		} else {
			postZoomLat, postZoomLon := screenCoordsToLatLng(mouseX, mouseY, g)
			g.centerLat += g.panStartLat - postZoomLat
			g.centerLon += g.panStartLon - postZoomLon
		}
	} else {
		g.panning = false
	}

	// Store previous mouse coordinates
	g.previousMouseX, g.previousMouseY = mouseX, mouseY

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
	numHorizontalTiles := int(math.Ceil(float64(g.ScreenWidth)/256)) + 2
	numVerticalTiles := int(math.Ceil(float64(g.ScreenHeight)/256)) + 2

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
			drawTile(screen, &g.tileCache, startTileX+i, startTileY+j, g.zoom, g.basemap, op)
		}
	}

	// Draw Lines

	dashLength, gapLength := float32(20), float32(40)
	clr := color.RGBA{255, 255, 255, 255}

	// Draw completed lines
	for _, line := range g.Lines {
		numPoints := len(line)
		if numPoints > 0 {
			for i, j := 0, 1; j < numPoints; i, j = i+1, j+1 {
				label := fmt.Sprintf("%.0f'", line[j].Dist)
				textDashedLine(screen, line[i].Lat, line[i].Lon, line[j].Lat, line[j].Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight, dashLength, gapLength, 3, clr, "144F", label)
			}
		}
	}

	// Draw currently active line
	numPoints := len(g.Points)
	if numPoints > 0 {
		for i, j := 0, 1; j < numPoints; i, j = i+1, j+1 {
			label := fmt.Sprintf("%.0f'", g.Points[j].Dist)
			textDashedLine(screen, g.Points[i].Lat, g.Points[i].Lon, g.Points[j].Lat, g.Points[j].Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight, dashLength, gapLength, 3, clr, "144F", label)
		}
		mouseX, mouseY := ebiten.CursorPosition()
		screenX, screenY := screenCoordsToLatLng(mouseX, mouseY, g)
		dist := haversine(g.Points[numPoints-1].Lat, g.Points[numPoints-1].Lon, screenX, screenY, EarthRadiusFT)
		label := fmt.Sprintf("%.0f'", dist)
		textDashedLine(screen, g.Points[numPoints-1].Lat, g.Points[numPoints-1].Lon, screenX, screenY, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight, dashLength, gapLength, 3, clr, "144F", label)
	}

	// GEOTIFF
	//g.DrawGeoTiff(screen)

	// Draw the current GPS position
	if g.gps.running {
		gpsX, gpsY := latLngToScreenCoords(g.gps.latitude, g.gps.longitude, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)
		gpsCircleRadius := 10.0 * g.gps.HDOP
		gpsCircleColor := color.RGBA{0, 0, 255, 179}

		vector.DrawFilledCircle(screen, gpsX, gpsY, float32(gpsCircleRadius), gpsCircleColor, false)
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

	mouseX, mouseY = ebiten.CursorPosition()
	lat, lon := screenCoordsToLatLng(mouseX, mouseY, g)
	debugString := fmt.Sprintf("Zoom: %d, Coords: %f, %f", g.zoom, lat, lon)
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
