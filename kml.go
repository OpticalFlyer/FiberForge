package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"image/color"
	"io"
	"os"
	"strconv"
	"strings"
)

type KML struct {
	XMLName   xml.Name   `xml:"kml"`
	Documents []Document `xml:"Document"`
	Folders   []Folder   `xml:"Folder"` // Folders without a document
}

type Document struct {
	XMLName    xml.Name    `xml:"Document"`
	Folders    []Folder    `xml:"Folder"`
	Documents  []Document  `xml:"Document"`  // Handle nested documents
	Placemarks []Placemark `xml:"Placemark"` // Placemarks without a folder
	Styles     []Style     `xml:"Style"`
	StyleMaps  []StyleMap  `xml:"StyleMap"`
}

type Folder struct {
	XMLName    xml.Name    `xml:"Folder"`
	Name       string      `xml:"name"`
	Placemarks []Placemark `xml:"Placemark"`
	Folders    []Folder    `xml:"Folder"`   // Handle nested folders
	Documents  []Document  `xml:"Document"` // Handle nested documents
}

type Placemark struct {
	Name       string     `xml:"name"`
	StyleURL   string     `xml:"styleUrl"`
	Style      Style      `xml:"Style"`
	Point      Point      `xml:"Point"`
	LineString LineString `xml:"LineString"`
}

type Point struct {
	Coordinates string `xml:"coordinates"`
}

type LineString struct {
	Coordinates string `xml:"coordinates"`
}

type StyleMap struct {
	XMLName xml.Name `xml:"StyleMap"`
	ID      string   `xml:"id,attr"`
	Pairs   []Pair   `xml:"Pair"`
}

type Pair struct {
	Key      string `xml:"key"`
	StyleURL string `xml:"styleUrl"`
}

type Style struct {
	XMLName   xml.Name  `xml:"Style"`
	ID        string    `xml:"id,attr"`
	IconStyle IconStyle `xml:"IconStyle"`
	LineStyle LineStyle `xml:"LineStyle"`
}

type IconStyle struct {
	Scale   float64 `xml:"scale"`
	Icon    Icon    `xml:"Icon"`
	HotSpot HotSpot `xml:"hotSpot"`
}

type Icon struct {
	Href string `xml:"href"`
}

type HotSpot struct {
	X      float64 `xml:"x,attr"`
	Y      float64 `xml:"y,attr"`
	XUnits string  `xml:"xunits,attr"`
	YUnits string  `xml:"yunits,attr"`
}

type LineStyle struct {
	Color string  `xml:"color"`
	Width float64 `xml:"width"`
}

func processFoldersAndDocuments(folders []Folder, documents []Document, game *Game) error {
	// Process Folders
	for _, folder := range folders {
		err := processPlacemarks(folder.Placemarks, game)
		if err != nil {
			return err
		}

		// Recursively process nested folders and documents
		err = processFoldersAndDocuments(folder.Folders, folder.Documents, game)
		if err != nil {
			return err
		}
	}

	// Process Documents
	for _, document := range documents {
		// Update the StyleMap for each Document.StyleMaps
		convertedStyleMap := convertStyleMapsToMap(document.StyleMaps)
		for id, pairs := range convertedStyleMap {
			if _, exists := game.StyleMap[id]; !exists {
				game.StyleMap[id] = pairs
				fmt.Printf("Added StyleMap %s - normal: %s, highlight: %s\n", id, pairs["normal"], pairs["highlight"])
			} else {
				for k, v := range pairs {
					game.StyleMap[id][k] = v
				}
			}
		}

		// Update the convertedMap for each Document.Styles
		convertedStyle := convertStylesToMap(document.Styles)
		for id, styleEntry := range convertedStyle {
			if _, exists := game.Styles[id]; !exists {
				game.Styles[id] = styleEntry
				fmt.Printf("Added Style %s - Color: %s, Width: %f\n", id, styleEntry.Color, styleEntry.Width)
			} else {
				game.Styles[id] = styleEntry
			}
		}

		// Process Placemarks within the document with no folder
		err := processPlacemarks(document.Placemarks, game)
		if err != nil {
			return err
		}

		// Recursively process folders and documents in the document
		err = processFoldersAndDocuments(document.Folders, document.Documents, game)
		if err != nil {
			return err
		}
	}

	return nil
}

/*
Sometimes there is an embedded style in the placemark
<Style><LineStyle><color>FF00ffff</color><width>5</width></LineStyle></Style>
*/
func processPlacemarks(placemarks []Placemark, game *Game) error {
	for _, placemark := range placemarks {

		// Skip points for now.  Only processing lines.
		if len(placemark.LineString.Coordinates) == 0 {
			continue
		}

		rawLineString := strings.TrimSpace(placemark.LineString.Coordinates)
		coordinates := strings.Split(strings.TrimSpace(rawLineString), " ")

		var line PolyLine

		styleURL := placemark.StyleURL
		if len(styleURL) > 0 { // Either a StyleMap or Style link
			if styleURL[0] == '#' {
				styleURL = styleURL[1:] // Strip leading #
			}

			if _, exists := game.StyleMap[styleURL]; !exists { // Not a StyleMap link
				line.Color, _ = hexStringToColor(game.Styles[styleURL].Color)
				line.Width = game.Styles[styleURL].Width
			} else { // StyleMap link
				line.Color, _ = hexStringToColor(game.Styles[game.StyleMap[styleURL]["normal"]].Color)
				line.Width = game.Styles[game.StyleMap[styleURL]["normal"]].Width
			}
		} else { // Embedded style?
			if len(placemark.Style.LineStyle.Color) > 0 {
				line.Color, _ = hexStringToColor(placemark.Style.LineStyle.Color)
			} else {
				line.Color = color.RGBA{0, 0, 0, 255}
			}
			if placemark.Style.LineStyle.Width > 0 {
				line.Width = float32(placemark.Style.LineStyle.Width)
			} else {
				line.Width = 1.0
			}
		}

		// Make sure we always have a minimum line width of 1
		if line.Width < 1 {
			line.Width = 1
		}

		for _, coordinate := range coordinates {
			latLon := strings.Split(coordinate, ",")
			if len(latLon) >= 2 {
				lat, err := strconv.ParseFloat(latLon[1], 64)
				if err != nil {
					return err
				}

				lon, err := strconv.ParseFloat(latLon[0], 64)
				if err != nil {
					return err
				}

				dist := 0.0
				if len(line.Points) > 0 {
					dist = haversine(line.Points[len(line.Points)-1].Lat, line.Points[len(line.Points)-1].Lon, lat, lon, EarthRadiusFT)

				}
				line.Points = append(line.Points, LinePoint{Lat: lat, Lon: lon, Dist: dist})
			}
		}
		fmt.Printf("Added line with %d points, Style: %s, Line Width: %f\n", len(line.Points), styleURL, line.Width)
		game.Lines = append(game.Lines, line)
	}

	return nil
}

func convertStyleMapsToMap(styleMaps []StyleMap) map[string]map[string]string {
	styleMapMap := make(map[string]map[string]string)

	for _, styleMap := range styleMaps {
		pairMap := make(map[string]string)
		for _, pair := range styleMap.Pairs {
			styleURL := pair.StyleURL
			if len(styleURL) > 0 && styleURL[0] == '#' {
				styleURL = styleURL[1:]
			}

			pairMap[pair.Key] = styleURL
		}
		styleMapMap[styleMap.ID] = pairMap
	}

	return styleMapMap
}

func convertStylesToMap(styles []Style) map[string]PolyLineStyle {
	convertedMap := make(map[string]PolyLineStyle)

	for _, style := range styles {
		convertedMap[style.ID] = PolyLineStyle{
			Color: style.LineStyle.Color,
			Width: float32(style.LineStyle.Width),
		}
	}

	return convertedMap
}

func hexStringToColor(hex string) (color.RGBA, error) {
	if len(hex) != 8 {
		return color.RGBA{}, fmt.Errorf("invalid color string")
	}

	a, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}

	b, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}

	g, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}

	r, err := strconv.ParseUint(hex[6:8], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}

	return color.RGBA{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: uint8(a),
	}, nil
}

func LoadKMLFile(filename string, game *Game) error {
	var kmlData []byte
	var err error

	if strings.HasSuffix(strings.ToLower(filename), ".kmz") {
		// Read KMZ file
		r, err := zip.OpenReader(filename)
		if err != nil {
			return err
		}
		defer r.Close()

		// Find the KML file inside the KMZ archive
		for _, f := range r.File {
			if strings.HasSuffix(strings.ToLower(f.Name), ".kml") {
				rc, err := f.Open()
				if err != nil {
					return err
				}
				defer rc.Close()

				kmlData, err = io.ReadAll(rc)
				if err != nil {
					return err
				}

				break
			}
		}

		if kmlData == nil {
			return fmt.Errorf("no KML file found in the KMZ archive")
		}

	} else {
		// Read KML file
		kmlData, err = os.ReadFile(filename)
		if err != nil {
			return err
		}
	}

	var kml KML
	err = xml.Unmarshal(kmlData, &kml)
	if err != nil {
		return err
	}

	// Process the Folders at the KML level
	err = processFoldersAndDocuments(kml.Folders, nil, game)
	if err != nil {
		return err
	}

	// Process the Documents at the KML level
	err = processFoldersAndDocuments(nil, kml.Documents, game)
	if err != nil {
		return err
	}

	return nil
}
