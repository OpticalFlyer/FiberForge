package main

import (
	"image"
	"image/color"
	"log"
	"math"

	"github.com/flywave/go-earcut"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	MinPolygonSize = 5 // Minimum size threshold for drawing polygons
)

var (
	whiteImage = ebiten.NewImage(3, 3)
)

func isPolygonTooSmall(points []struct{ x, y float64 }) bool {
	if len(points) < 2 {
		return true
	}
	minX, minY := points[0].x, points[0].y
	maxX, maxY := points[0].x, points[0].y
	for _, p := range points {
		if p.x < minX {
			minX = p.x
		}
		if p.x > maxX {
			maxX = p.x
		}
		if p.y < minY {
			minY = p.y
		}
		if p.y > maxY {
			maxY = p.y
		}
	}
	width := maxX - minX
	height := maxY - minY
	return width < MinPolygonSize || height < MinPolygonSize
}

func drawFilledPolygon(screen *ebiten.Image, points []struct{ x, y float64 }, fillColor color.Color) {
	if len(points) < 3 {
		log.Printf("Not enough points to form a polygon: %+v", points)
		return // A polygon must have at least 3 points
	}

	// Simplify the polygon
	points = simplifyPolygon(points, 0.1)

	// Check if the polygon is too small to draw
	if isPolygonTooSmall(points) {
		log.Printf("Polygon too small to draw: %+v", points)
		return
	}

	// Remove duplicate points
	points = removeDuplicatePoints(points)

	// Convert points to vertices
	vertices := make([]ebiten.Vertex, len(points))
	for i, p := range points {
		vertices[i] = ebiten.Vertex{
			DstX:   float32(p.x),
			DstY:   float32(p.y),
			SrcX:   1,
			SrcY:   1,
			ColorR: float32(fillColor.(color.RGBA).R) / 255,
			ColorG: float32(fillColor.(color.RGBA).G) / 255,
			ColorB: float32(fillColor.(color.RGBA).B) / 255,
			ColorA: float32(fillColor.(color.RGBA).A) / 255,
		}
	}

	// Triangulate the polygon using earcut
	indices, err := earcutPolygon(points)
	if err != nil {
		log.Printf("Failed to triangulate polygon: %+v", points)
		return
	}

	// Draw the filled polygon
	screen.DrawTriangles(vertices, indices, whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image), &ebiten.DrawTrianglesOptions{})

	// Optionally, draw the outline of the polygon
	for i := 0; i < len(points); i++ {
		next := (i + 1) % len(points)
		vector.StrokeLine(screen, float32(points[i].x), float32(points[i].y), float32(points[next].x), float32(points[next].y), 2, color.RGBA{0x00, 0x00, 0x00, 0xff}, false)
	}
}

// removeDuplicatePoints removes duplicate points from the polygon
func removeDuplicatePoints(points []struct{ x, y float64 }) []struct{ x, y float64 } {
	uniquePoints := []struct{ x, y float64 }{}
	pointMap := make(map[struct{ x, y float64 }]bool)

	for _, point := range points {
		if _, exists := pointMap[point]; !exists {
			pointMap[point] = true
			uniquePoints = append(uniquePoints, point)
		} else {
			log.Printf("Removing duplicate point: %+v", point)
		}
	}

	return uniquePoints
}

// earcutPolygon performs ear clipping triangulation on the given polygon points using the earcut algorithm
func earcutPolygon(points []struct{ x, y float64 }) ([]uint16, error) {
	// Flatten the points for earcut
	var coords []float64
	for _, point := range points {
		coords = append(coords, point.x, point.y)
	}

	// Call the earcut implementation
	indices, err := earcut.Earcut(coords, nil, 2)
	if err != nil {
		return nil, err
	}

	// Convert to uint16
	var uint16Indices []uint16
	for _, index := range indices {
		uint16Indices = append(uint16Indices, uint16(index))
	}

	return uint16Indices, nil
}

// BEGIN: Polygon simplification using Ramer-Douglas-Peucker algorithm
func simplifyPolygon(points []struct{ x, y float64 }, tolerance float64) []struct{ x, y float64 } {
	if len(points) < 3 {
		return points
	}

	// Simplify the polygon using the Ramer-Douglas-Peucker algorithm
	return rdpSimplify(points, tolerance)
}

func rdpSimplify(points []struct{ x, y float64 }, epsilon float64) []struct{ x, y float64 } {
	if len(points) < 2 {
		return points
	}

	dmax := 0.0
	index := 0
	end := len(points) - 1

	for i := 1; i < end; i++ {
		d := perpendicularDistance(points[i], points[0], points[end])
		if d > dmax {
			index = i
			dmax = d
		}
	}

	if dmax > epsilon {
		recResults1 := rdpSimplify(points[:index+1], epsilon)
		recResults2 := rdpSimplify(points[index:], epsilon)

		return append(recResults1[:len(recResults1)-1], recResults2...)
	} else {
		return []struct{ x, y float64 }{points[0], points[end]}
	}
}

func perpendicularDistance(point, lineStart, lineEnd struct{ x, y float64 }) float64 {
	dx := lineEnd.x - lineStart.x
	dy := lineEnd.y - lineStart.y

	if dx == 0 && dy == 0 {
		return distance(point, lineStart)
	}

	return math.Abs(dy*point.x-dx*point.y+lineEnd.x*lineStart.y-lineEnd.y*lineStart.x) /
		math.Sqrt(dx*dx+dy*dy)
}

func distance(p1, p2 struct{ x, y float64 }) float64 {
	return math.Sqrt((p1.x-p2.x)*(p1.x-p2.x) + (p1.y-p2.y)*(p1.y-p2.y))
}

// END: Polygon simplification using Ramer-Douglas-Peucker algorithm
