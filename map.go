package main

import (
	"bytes"
	"fmt"
	"image/color"
	"image/jpeg"
	"io"
	"math"
	"net/http"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
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
}

var downloadQueue = make(chan DownloadRequest, 100)

func startWorkerPool(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go tileDownloader()
	}
}

func tileDownloader() {
	for req := range downloadQueue {
		img, err := downloadTileImage(req.tileX, req.tileY, req.zoom)
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

/*func drawTile(screen *ebiten.Image, tileCache TileImageCache, tileX, tileY, zoom int, op *ebiten.DrawImageOptions) {
	cachedImg, ok := tileCache.Get(zoom, tileX, tileY)
	if ok {
		screen.DrawImage(cachedImg, op)
	} else {
		img, err := downloadTileImage(tileX, tileY, zoom)
		if err != nil {
			img := ebiten.NewImage(256, 256)
			solidColor := color.RGBA{R: 0, G: 0, B: 0, A: 255}
			img.Fill(solidColor)
			tileCache.Set(zoom, tileX, tileY, img)
			screen.DrawImage(img, op)
		} else {
			tileCache.Set(zoom, tileX, tileY, img)
			screen.DrawImage(img, op)
		}
	}
}*/

/*func drawTile(screen *ebiten.Image, tileCache *TileImageCache, tileX, tileY, zoom int, op *ebiten.DrawImageOptions) {
	cachedImg, ok := tileCache.Get(zoom, tileX, tileY)
	if ok {
		screen.DrawImage(cachedImg, op)
	} else {
		// Create a black placeholder tile and draw it
		img := ebiten.NewImage(256, 256)
		solidColor := color.RGBA{R: 0, G: 0, B: 0, A: 255}
		img.Fill(solidColor)
		screen.DrawImage(img, op)

		// Launch a goroutine to download the tile and update the cache
		go func(tileCache *TileImageCache, zoom, tileX, tileY int) {
			img, err := downloadTileImage(tileX, tileY, zoom)
			if err == nil {
				tileCache.Set(zoom, tileX, tileY, img)
			}
		}(tileCache, zoom, tileX, tileY)
	}
}*/

func drawTile(screen *ebiten.Image, tileCache *TileImageCache, tileX, tileY, zoom int, op *ebiten.DrawImageOptions) {
	cachedImg, ok := tileCache.Get(zoom, tileX, tileY)
	if ok {
		screen.DrawImage(cachedImg, op)
	} else {
		// Create a black placeholder tile and draw it
		img := ebiten.NewImage(256, 256)
		solidColor := color.RGBA{R: 0, G: 0, B: 0, A: 255}
		img.Fill(solidColor)
		screen.DrawImage(img, op)

		// Add a download request to the queue
		downloadQueue <- DownloadRequest{
			tileCache: tileCache,
			zoom:      zoom,
			tileX:     tileX,
			tileY:     tileY,
		}
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

	deltaLon := offsetX * metersPerPixel / 6371000 * 180 / math.Pi
	deltaLat := -offsetY * metersPerPixel / 6371000 * 180 / math.Pi

	clickedLat := centerLat + deltaLat
	clickedLon := centerLon + deltaLon

	return clickedLat, clickedLon
}

func downloadTileImage(x, y, zoom int) (*ebiten.Image, error) {
	url := fmt.Sprintf("https://mt1.google.com/vt/lyrs=s,h&x=%d&y=%d&z=%d", x, y, zoom)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(img), nil
}
