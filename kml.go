package main

import (
	"encoding/xml"
	"io/ioutil"
	"strconv"
	"strings"
)

type KML struct {
	XMLName  xml.Name `xml:"kml"`
	Document Document `xml:"Document"`
}

type Document struct {
	XMLName xml.Name `xml:"Document"`
	Folders []Folder `xml:"Folder"`
}

type Folder struct {
	XMLName    xml.Name    `xml:"Folder"`
	Name       string      `xml:"name"`
	Placemarks []Placemark `xml:"Placemark"`
	Folders    []Folder    `xml:"Folder"` // Handle nested folders
}

type Placemark struct {
	Name       string     `xml:"name"`
	LineString LineString `xml:"LineString"`
}

type LineString struct {
	Coordinates string `xml:"coordinates"`
}

/*func LoadKMLFile(filename string) (*Document, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var doc Document
	err = xml.Unmarshal(data, &doc)
	if err != nil {
		return nil, err
	}

	return &doc, nil
}*/

func processFolders(folders []Folder, game *Game) error {
	for _, folder := range folders {
		for _, placemark := range folder.Placemarks {
			lineString := placemark.LineString.Coordinates
			coordinates := strings.Split(strings.TrimSpace(lineString), " ")
			var points []struct{ Lat, Lon, Dist float64 }

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
					//points = append(points, struct{ Lat, Lon, Dist float64 }{Lat: lat, Lon: lon, Dist: 0.0})
					if len(points) > 0 {
						dist := haversine(points[len(points)-1].Lat, points[len(points)-1].Lon, lat, lon, EarthRadiusFT)
						points = append(points, struct{ Lat, Lon, Dist float64 }{Lat: lat, Lon: lon, Dist: dist})
					} else {
						points = append(points, struct{ Lat, Lon, Dist float64 }{Lat: lat, Lon: lon, Dist: 0.0})
					}
				}
			}
			game.Lines = append(game.Lines, points)
		}

		// Recursively process nested folders
		if len(folder.Folders) > 0 {
			err := processFolders(folder.Folders, game)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func LoadKMLFile(filename string, game *Game) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var kml KML
	err = xml.Unmarshal(data, &kml)
	if err != nil {
		return err
	}

	// Process the data
	err = processFolders(kml.Document.Folders, game)
	if err != nil {
		return err
	}

	return nil
}
