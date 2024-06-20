package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	GOOGLEHYBRID = "GOOGLEHYBRID"
	GOOGLEAERIAL = "GOOGLEAERIAL"
	BINGHYBRID   = "BINGHYBRID"
	BINGAERIAL   = "BINGAERIAL"
	OSM          = "OSM"
)

type TileImageCache struct {
	cache map[int]map[int]map[int]*ebiten.Image
	mu    sync.Mutex
}

func NewTileImageCache() TileImageCache {
	return TileImageCache{
		cache: make(map[int]map[int]map[int]*ebiten.Image),
	}
}

type DownloadRequest struct {
	tileCache *TileImageCache
	zoom      int
	tileX     int
	tileY     int
	basemap   string
}

var downloadQueue = make(chan DownloadRequest, 100)

func startWorkerPool(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go tileDownloader()
	}
}

func tileDownloader() {
	for req := range downloadQueue {
		img, err := downloadTileImage(req.tileX, req.tileY, req.zoom, req.basemap)
		if err == nil {
			req.tileCache.Set(req.zoom, req.tileX, req.tileY, img)
		}
	}
}

func (cache *TileImageCache) Set(zoom, x, y int, img *ebiten.Image) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if _, ok := cache.cache[zoom]; !ok {
		cache.cache[zoom] = make(map[int]map[int]*ebiten.Image)
	}
	if _, ok := cache.cache[zoom][x]; !ok {
		cache.cache[zoom][x] = make(map[int]*ebiten.Image)
	}
	cache.cache[zoom][x][y] = img
}

func (cache *TileImageCache) Get(zoom, x, y int) (*ebiten.Image, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if _, ok := cache.cache[zoom]; !ok {
		return nil, false
	}
	if _, ok := cache.cache[zoom][x]; !ok {
		return nil, false
	}
	img, ok := cache.cache[zoom][x][y]
	return img, ok
}

func drawTile(screen *ebiten.Image, emptyTile *ebiten.Image, tileCache *TileImageCache, tileX, tileY, zoom int, basemap string, op *ebiten.DrawImageOptions) bool {
	cachedImg, ok := tileCache.Get(zoom, tileX, tileY)
	if ok {
		screen.DrawImage(cachedImg, op)
		return false
	} else {
		// Draw the empty tile
		screen.DrawImage(emptyTile, op)

		// Add a download request to the queue
		downloadQueue <- DownloadRequest{
			tileCache: tileCache,
			zoom:      zoom,
			tileX:     tileX,
			tileY:     tileY,
			basemap:   basemap,
		}
		return true
	}
}

func latLngToTile(lat, lng float64, zoom int) (int, int) {
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2, float64(zoom))
	xtile := int((lng + 180.0) / 360.0 * n)
	ytile := int((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n)
	return xtile, ytile
}

func latLngToTilePixel(lat, lng float64, zoom int) (int, int) {
	// Calculate the pixel coordinates inside the tile
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2, float64(zoom))
	pixelX := int((lng+180.0)/360.0*n*256.0) % 256
	pixelY := int((1.0-math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi)/2.0*n*256.0) % 256

	return pixelX, pixelY
}

func screenCoordsToLatLng(screenX, screenY int, game *Game) (float64, float64) {
	centerLat := game.centerLat
	centerLon := game.centerLon
	zoom := float64(game.zoom)

	tileSize := 256.0
	scale := tileSize * math.Pow(2, zoom)

	metersPerPixel := math.Cos(centerLat*math.Pi/180.0) * 2 * math.Pi * 6371000 / scale

	offsetX := float64(screenX - game.ScreenWidth/2)
	offsetY := float64(screenY - game.ScreenHeight/2)

	deltaLon := offsetX * metersPerPixel / (6371000 * math.Cos(centerLat*math.Pi/180.0)) * 180 / math.Pi
	deltaLat := -offsetY * metersPerPixel / 6371000 * 180 / math.Pi

	clickedLat := centerLat + deltaLat
	clickedLon := centerLon + deltaLon

	return clickedLat, clickedLon
}

func latLngToScreenCoords(lat, lng, centerLat, centerLon, zoom float64, screenWidth, screenHeight int) (float32, float32) {
	// Web Mercator projection formulas
	tileSize := 256.0
	//scale := tileSize * math.Pow(2, zoom)
	worldSize := tileSize * math.Pow(2, zoom)
	origin := worldSize / 2

	// Convert center lat/lon to pixel coordinates
	centerX := origin + centerLon*math.Pi/180.0*origin/math.Pi
	centerY := origin - math.Log(math.Tan((centerLat*math.Pi/180.0+math.Pi/2)/2))*origin/math.Pi

	// Convert lat/lon to pixel coordinates
	x := origin + lng*math.Pi/180.0*origin/math.Pi
	y := origin - math.Log(math.Tan((lat*math.Pi/180.0+math.Pi/2)/2))*origin/math.Pi

	// Calculate screen coordinates
	screenX := int(x - centerX + float64(screenWidth)/2)
	screenY := int(y - centerY + float64(screenHeight)/2)

	return float32(screenX), float32(screenY)
}

func buildTilePath(basemap string, zoom, x, y int) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	var fileExtension string
	if basemap == OSM {
		fileExtension = "png"
	} else {
		fileExtension = "jpg"
	}
	tilePath := filepath.Join(homeDir, ".fiberforge", "tilecache", basemap, fmt.Sprintf("%d-%d-%d.%s", zoom, x, y, fileExtension))
	return tilePath, nil
}

func saveTileToDisk(tilePath string, data []byte) error {
	// Ensure the directory exists
	dir := filepath.Dir(tilePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	// Write the file
	err := os.WriteFile(tilePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func downloadTileImage(x, y, zoom int, basemap string) (*ebiten.Image, error) {
	tilePath, err := buildTilePath(basemap, zoom, x, y)
	if err != nil {
		fmt.Printf("Failed to build tile path: %s\n", err)
	}

	if _, err := os.Stat(tilePath); err == nil {
		// Tile exists, load from disk
		fileData, err := os.ReadFile(tilePath)
		if err != nil {
			return nil, err
		}
		// Depending on the basemap, decode the image accordingly
		var img image.Image
		if basemap == OSM {
			img, err = png.Decode(bytes.NewReader(fileData))
		} else {
			img, err = jpeg.Decode(bytes.NewReader(fileData))
		}
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(img), nil
	}

	var url string
	if basemap == BINGHYBRID {
		q := getQuadKey(zoom, x, y)
		url = fmt.Sprintf("http://ecn.t1.tiles.virtualearth.net/tiles/h%s.jpeg?g=129&mkt=en-US&shading=hill&stl=H", q)
	} else if basemap == BINGAERIAL {
		q := getQuadKey(zoom, x, y)
		url = fmt.Sprintf("http://ecn.t1.tiles.virtualearth.net/tiles/a%s.jpeg?g=129&mkt=en-US&shading=hill&stl=H", q)
	} else if basemap == GOOGLEAERIAL {
		url = fmt.Sprintf("https://mt1.google.com/vt/lyrs=s&x=%d&y=%d&z=%d", x, y, zoom)
	} else if basemap == GOOGLEHYBRID {
		url = fmt.Sprintf("https://mt1.google.com/vt/lyrs=s,h&x=%d&y=%d&z=%d", x, y, zoom)
	} else {
		url = fmt.Sprintf("https://tile.openstreetmap.org/%d/%d/%d.png", zoom, x, y)
	}

	/*resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()*/

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "GeoForge/alpha")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Something went wrong: %s\n", resp.Status)
		return nil, fmt.Errorf("failed to download image: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var img image.Image
	if basemap == OSM {
		img, err = png.Decode(bytes.NewReader(data))
	} else {
		img, err = jpeg.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return nil, err
	}

	if err := saveTileToDisk(tilePath, data); err != nil {
		fmt.Println("Failed to save tile to disk:", err)
	}

	return ebiten.NewImageFromImage(img), nil
}

func pointLineSegmentDistance(x, y, x1, y1, x2, y2 float64) float64 {
	// Calculate the squared length of the line segment
	l2 := math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2)
	if l2 == 0 {
		return math.Sqrt(math.Pow(x-x1, 2) + math.Pow(y-y1, 2))
	}

	// Calculate the projection of the point onto the line segment
	t := ((x-x1)*(x2-x1) + (y-y1)*(y2-y1)) / l2
	t = math.Max(0, math.Min(1, t))

	// Calculate the projection point
	projX := x1 + t*(x2-x1)
	projY := y1 + t*(y2-y1)

	// Calculate the distance from the point to the projection point
	return math.Sqrt(math.Pow(x-projX, 2) + math.Pow(y-projY, 2))
}

func getQuadKey(zoom, tileX, tileY int) string {
	var quadKey string
	for i := zoom; i > 0; i-- {
		var digit int
		mask := 1 << (i - 1)
		if (tileX & mask) != 0 {
			digit += 1
		}
		if (tileY & mask) != 0 {
			digit += 2
		}
		quadKey += fmt.Sprintf("%d", digit)
	}
	return quadKey
}
