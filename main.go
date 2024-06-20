package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/atotto/clipboard"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type PointObject struct {
	Lat, Lon  float64
	Color     color.RGBA
	IconImage *ebiten.Image
	Scale     float64
	HotSpot   HotSpot
}

type LinePoint struct {
	Lat, Lon, Dist float64
}

type PolyLine struct {
	Points []LinePoint
	Color  color.RGBA
	Width  float32
}

type PolyLineStyle struct {
	Color string
	Width float32
}

type IconStyleData struct {
	ID      string
	Color   string
	Scale   float64
	Href    string
	HotSpot HotSpot
}

type PolyPoint struct {
	Lat, Lon float64
}

type PolygonObject struct {
	Points []PolyPoint
}

type Game struct {
	ScreenWidth    int
	ScreenHeight   int
	basemap        string
	TextBoxText    string
	LastCmdText    string
	Points         []PointObject
	Line           PolyLine
	Lines          []PolyLine
	PolygonObject  PolygonObject
	Polygons       []PolygonObject
	StyleMap       map[string]map[string]string
	Styles         map[string]PolyLineStyle
	IconStyles     map[string]IconStyleData
	IconImages     map[string]*ebiten.Image
	PL_activated   bool
	PO_activated   bool
	POL_activated  bool
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
	numSegments    int
	emptyTile      *ebiten.Image
	offscreenImage *ebiten.Image
	needRedraw     bool
}

func Initialize() (*Game, error) {
	g := &Game{}
	g.centerLat = 35.156072
	g.centerLon = -90.051911
	g.zoom = 5
	g.basemap = GOOGLEAERIAL

	g.Line.Color = color.RGBA{0, 255, 255, 255}
	g.Line.Width = 3.0
	g.StyleMap = make(map[string]map[string]string)
	g.Styles = make(map[string]PolyLineStyle)
	g.IconStyles = make(map[string]IconStyleData)

	g.tileCache = NewTileImageCache()

	g.emptyTile = ebiten.NewImage(256, 256)
	solidColor := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	g.emptyTile.Fill(solidColor)

	g.gps = NewGPS()

	g.ScreenWidth = 1024
	g.ScreenHeight = 768
	g.offscreenImage = ebiten.NewImage(g.ScreenWidth, g.ScreenHeight)
	g.needRedraw = true

	// For polygon drawing
	whiteImage.Fill(color.White)

	return g, nil
}

func (g *Game) Update() error {
	if droppedFiles := ebiten.DroppedFiles(); droppedFiles != nil {
		err := LoadKMLDroppedFiles(droppedFiles, g)
		if err != nil {
			log.Println(err)
		}
		g.needRedraw = true
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && g.PL_activated {
		mouseX, mouseY := ebiten.CursorPosition()
		lat, lon := screenCoordsToLatLng(mouseX, mouseY, g)
		dist := 0.0
		if len(g.Line.Points) > 0 {
			prevPoint := len(g.Line.Points) - 1
			dist = haversine(g.Line.Points[prevPoint].Lat, g.Line.Points[prevPoint].Lon, lat, lon, EarthRadiusFT)

		}
		g.Line.Points = append(g.Line.Points, LinePoint{Lat: lat, Lon: lon, Dist: dist})
	} else if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && g.PO_activated {
		mouseX, mouseY := ebiten.CursorPosition()
		lat, lon := screenCoordsToLatLng(mouseX, mouseY, g)
		clr := color.RGBA{255, 255, 255, 255}
		g.Points = append(g.Points, PointObject{Lat: lat, Lon: lon, Color: clr, Scale: 1.0, IconImage: nil})
		g.needRedraw = true
	} else if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && g.POL_activated {
		mouseX, mouseY := ebiten.CursorPosition()
		lat, lon := screenCoordsToLatLng(mouseX, mouseY, g)
		g.PolygonObject.Points = append(g.PolygonObject.Points, PolyPoint{Lat: lat, Lon: lon})
		g.needRedraw = true
	}

	if inpututil.IsKeyJustReleased(ebiten.KeySpace) || inpututil.IsKeyJustReleased(ebiten.KeyEnter) {
		if g.PL_activated { // Save new line
			g.PL_activated = false
			if len(g.Line.Points) > 0 {
				g.Lines = append(g.Lines, g.Line)
				g.Line.Points = nil
			}
		} else if g.POL_activated { // Save new polygon
			g.POL_activated = false
			if len(g.PolygonObject.Points) > 2 {
				g.Polygons = append(g.Polygons, g.PolygonObject)
				g.PolygonObject.Points = nil
			}
		} else if g.TextBoxText == "PL" || g.TextBoxText == "" && g.LastCmdText == "PL" { // Start new line
			g.PL_activated = true
			g.LastCmdText = "PL"
		} else if g.PO_activated { // End point mode
			g.PO_activated = false
		} else if g.TextBoxText == "PO" || g.TextBoxText == "" && g.LastCmdText == "PO" { // Start new point
			g.PO_activated = true
			g.LastCmdText = "PO"
		} else if g.TextBoxText == "POL" || g.TextBoxText == "" && g.LastCmdText == "POL" { // Start new point
			g.POL_activated = true
			g.LastCmdText = "POL"
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
		} else if g.TextBoxText == "MAPIMPORT" {
			//homeDir, _ := os.UserHomeDir()
			//LoadKMLFile(filepath.Join(homeDir, "test.kml"), g)
			clipboardContent, err := clipboard.ReadAll()
			if err != nil {
				fmt.Printf("Error reading clipboard: %v\n", err)
			}

			LoadKMLFile(clipboardContent, g)
		}
		g.TextBoxText = ""

		g.needRedraw = true
	} else {
		g.handleTextInput()
	}

	// Determine if line segment is clicked
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && !g.PL_activated && !g.PO_activated && !g.POL_activated {
		mouseX, mouseY := ebiten.CursorPosition()
		//lat, lon := screenCoordsToLatLng(mouseX, mouseY, g)

		threshold := 5.0 // Pixels

		// Iterate through each PointObject
		for index, point := range g.Points {
			// Convert the point's lat/lon to screen coordinates
			pointX, pointY := latLngToScreenCoords(point.Lat, point.Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)

			// Calculate the distance from the click to the point
			dx := float32(mouseX) - pointX
			dy := float32(mouseY) - pointY
			distance := math.Sqrt(float64(dx*dx + dy*dy))

			// Check if the distance is within the selection radius
			if distance <= threshold {
				fmt.Printf("Point %d selected\n", index)
			}
		}

		// Iterate through each segment in the PolyLine
		for _, polyLine := range g.Lines {
			for i := 0; i < len(polyLine.Points)-1; i++ {
				// Convert the segment's start and end points from lat/lon to screen coordinates
				startX, startY := latLngToScreenCoords(polyLine.Points[i].Lat, polyLine.Points[i].Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)
				endX, endY := latLngToScreenCoords(polyLine.Points[i+1].Lat, polyLine.Points[i+1].Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)

				// Calculate the distance from the mouse click to the current line segment
				distance := pointLineSegmentDistance(float64(mouseX), float64(mouseY), float64(startX), float64(startY), float64(endX), float64(endY))

				if distance <= threshold {
					// The user clicked on a line segment
					fmt.Printf("Clicked close to line segment between points %d and %d\n", i, i+1)
				}
			}
		}
	}

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

		g.needRedraw = true
	}

	// Panning
	tileWidth := 360 / math.Pow(2, float64(g.zoom))
	panSpeed := tileWidth * 0.5

	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.centerLon -= panSpeed
		g.needRedraw = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.centerLon += panSpeed
		g.needRedraw = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.centerLat += panSpeed
		g.needRedraw = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.centerLat -= panSpeed
		g.needRedraw = true
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
		g.needRedraw = true
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
	if g.needRedraw {
		g.needRedraw = false // Reset the flag
		g.offscreenImage.Clear()

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
				if drawTile(g.offscreenImage, g.emptyTile, &g.tileCache, startTileX+i, startTileY+j, g.zoom, g.basemap, op) {
					g.needRedraw = true
				}
			}
		}

		// Draw Lines
		g.numSegments = 0
		for _, line := range g.Lines {
			numPoints := len(line.Points)
			g.numSegments += numPoints - 1
			if numPoints > 0 {
				for i, j := 0, 1; j < numPoints; i, j = i+1, j+1 {
					//label := fmt.Sprintf("%.0f'", line.Points[j].Dist)
					//textDashedLine(screen, line.Points[i].Lat, line.Points[i].Lon, line.Points[j].Lat, line.Points[j].Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight, dashLength, gapLength, line.Width, line.Color, "144F", label)
					solidLine(g.offscreenImage, line.Points[i].Lat, line.Points[i].Lon, line.Points[j].Lat, line.Points[j].Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight, line.Width, line.Color)
				}
			}
		}

		// Draw point objects
		if len(g.Points) > 0 {
			for _, point := range g.Points {
				pointX, pointY := latLngToScreenCoords(point.Lat, point.Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)

				// Check if the point is within the screen bounds
				if pointX >= 0 && pointX <= float32(g.ScreenWidth) && pointY >= 0 && pointY <= float32(g.ScreenHeight) {
					if point.IconImage != nil {
						// Draw the icon with the hotspot offset from the bottom-left corner
						op := &ebiten.DrawImageOptions{}

						// Calculate the offset based on hotspot values
						//offsetX := float32(point.HotSpot.X * float64(point.IconImage.Bounds().Dx()))
						//offsetY := float32(point.HotSpot.Y * float64(point.IconImage.Bounds().Dy()))

						// If hotspot x and y are both 0, center the icon on pointX and pointY
						if point.HotSpot.X == 0 && point.HotSpot.Y == 0 {
							centerX := float32(point.IconImage.Bounds().Dx()) / 2
							centerY := float32(point.IconImage.Bounds().Dy()) / 2
							op.GeoM.Translate(float64(pointX-centerX), float64(pointY-centerY))
						} else {
							// Apply the hotspot offset
							op.GeoM.Translate(float64(pointX)-point.HotSpot.X, float64(pointY)-float64(point.IconImage.Bounds().Dy())+point.HotSpot.Y)
						}

						g.offscreenImage.DrawImage(point.IconImage, op)
					} else {
						// Draw a circle if there's no icon
						pointRadius := 5.0
						pointColor := point.Color

						vector.DrawFilledCircle(g.offscreenImage, pointX, pointY, float32(pointRadius), pointColor, false)
						vector.StrokeCircle(g.offscreenImage, pointX, pointY, float32(pointRadius), 2, color.RGBA{0, 0, 0, 255}, false)
					}
				}
			}
		}

		// Loop through all polygons in g.Polygons and render them
		for _, polygon := range g.Polygons {
			if len(polygon.Points) > 2 {
				screenPoints := make([]struct{ x, y float64 }, len(polygon.Points))
				for i, pt := range polygon.Points {
					x32, y32 := latLngToScreenCoords(pt.Lat, pt.Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)
					screenPoints[i] = struct{ x, y float64 }{float64(x32), float64(y32)}
				}
				drawFilledPolygon(g.offscreenImage, screenPoints, color.RGBA{0x00, 0xff, 0x00, 0x4D}) // Green filled polygon
			}
		}
	}

	// Draw the off-screen image to the screen
	screen.DrawImage(g.offscreenImage, nil)

	// Draw currently active polygon
	if g.POL_activated && len(g.PolygonObject.Points) > 0 {
		screenPoints := make([]struct{ x, y float64 }, len(g.PolygonObject.Points))
		for i, pt := range g.PolygonObject.Points {
			x32, y32 := latLngToScreenCoords(pt.Lat, pt.Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)
			screenPoints[i] = struct{ x, y float64 }{float64(x32), float64(y32)}
		}

		mouseX, mouseY := ebiten.CursorPosition()
		screenX, screenY := screenCoordsToLatLng(mouseX, mouseY, g)
		x32, y32 := latLngToScreenCoords(screenX, screenY, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)

		// Check if the mouse coordinates are the same as the last point
		lastPoint := screenPoints[len(screenPoints)-1]
		if !(float64(x32) == lastPoint.x && float64(y32) == lastPoint.y) {
			// Include the mouse coordinates as the last point
			screenPoints = append(screenPoints, struct{ x, y float64 }{float64(x32), float64(y32)})
		}

		if len(screenPoints) > 2 {
			drawFilledPolygon(screen, screenPoints, color.RGBA{0x00, 0xff, 0x00, 0x4D}) // Green filled polygon
		} else {
			// Draw a line from the first point to the mouse cursor
			vector.StrokeLine(screen, float32(screenPoints[0].x), float32(screenPoints[0].y), float32(x32), float32(y32), 2, color.RGBA{0x00, 0x00, 0x00, 0xff}, false)
		}
	}

	// Draw currently active line
	dashLength, gapLength := float32(20), float32(40)
	numPoints := len(g.Line.Points)
	if numPoints > 0 {
		for i, j := 0, 1; j < numPoints; i, j = i+1, j+1 {
			label := fmt.Sprintf("%.0f'", g.Line.Points[j].Dist)
			textDashedLine(screen, g.Line.Points[i].Lat, g.Line.Points[i].Lon, g.Line.Points[j].Lat, g.Line.Points[j].Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight, dashLength, gapLength, g.Line.Width, g.Line.Color, "144F", label)
		}
		mouseX, mouseY := ebiten.CursorPosition()
		screenX, screenY := screenCoordsToLatLng(mouseX, mouseY, g)
		dist := haversine(g.Line.Points[numPoints-1].Lat, g.Line.Points[numPoints-1].Lon, screenX, screenY, EarthRadiusFT)
		label := fmt.Sprintf("%.0f'", dist)
		textDashedLine(screen, g.Line.Points[numPoints-1].Lat, g.Line.Points[numPoints-1].Lon, screenX, screenY, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight, dashLength, gapLength, g.Line.Width, g.Line.Color, "144F", label)
	}

	/*// Draw point objects
	if len(g.Points) > 0 {
		for _, point := range g.Points {
			pointX, pointY := latLngToScreenCoords(point.Lat, point.Lon, g.centerLat, g.centerLon, float64(g.zoom), g.ScreenWidth, g.ScreenHeight)

			// Check if the point is within the screen bounds
			if pointX >= 0 && pointX <= float32(g.ScreenWidth) && pointY >= 0 && pointY <= float32(g.ScreenHeight) {
				pointRadius := 5.0
				pointColor := point.Color

				vector.DrawFilledCircle(screen, pointX, pointY, float32(pointRadius), pointColor, false)
				vector.StrokeCircle(screen, pointX, pointY, float32(pointRadius), 2, color.RGBA{0, 0, 0, 255}, false)
			}

		}
	}*/

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

	if g.PO_activated {
		pointRadius := 5.0
		pointColor := color.RGBA{128, 128, 128, 26}

		vector.DrawFilledCircle(screen, float32(mouseX), float32(mouseY), float32(pointRadius), pointColor, false)
	}

	// Draw the crosshair at the mouse position
	if g.PL_activated || g.PO_activated || g.POL_activated {
		drawCrosshair(screen, float32(mouseX), float32(mouseY), 100, color.RGBA{255, 255, 255, 255})
	} else {
		drawSquareCrosshair(screen, float32(mouseX), float32(mouseY), 10, 100, color.RGBA{255, 255, 255, 255})
	}

	mouseX, mouseY = ebiten.CursorPosition()
	lat, lon := screenCoordsToLatLng(mouseX, mouseY, g)
	debugString := fmt.Sprintf("Zoom: %d, Coords: %f, %f\n%d Points, %d Lines (%d Segments)\n%d Styles, %d Style Maps\n%.0f FPS",
		g.zoom, lat, lon, len(g.Points), len(g.Lines), g.numSegments, len(g.Styles), len(g.StyleMap), ebiten.ActualFPS())
	ebitenutil.DebugPrint(screen, debugString)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	if g.ScreenWidth != outsideWidth || g.ScreenHeight != outsideHeight {
		fmt.Println("Resizing to", outsideWidth, outsideHeight)
		// Recreate the off-screen image with the new dimensions
		g.offscreenImage = ebiten.NewImage(outsideWidth, outsideHeight)

		g.needRedraw = true
	}

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

	ebiten.SetWindowSize(fiberforge.ScreenWidth, fiberforge.ScreenHeight)
	ebiten.SetWindowTitle("CAD/GIS Experiment")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	if err := ebiten.RunGame(fiberforge); err != nil {
		fmt.Println(err)
	}
}
