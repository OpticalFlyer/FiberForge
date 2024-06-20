package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

type Osm struct {
	XMLName xml.Name `xml:"osm"`
	Nodes   []Node   `xml:"node"`
	Ways    []Way    `xml:"way"`
}

type Node struct {
	XMLName xml.Name `xml:"node"`
	Id      int      `xml:"id,attr"`
	Lat     float64  `xml:"lat,attr"`
	Lon     float64  `xml:"lon,attr"`
}

type Way struct {
	XMLName xml.Name  `xml:"way"`
	Id      int       `xml:"id,attr"`
	Nodes   []WayNode `xml:"nd"`
	Tags    []Tag     `xml:"tag"`
}

type WayNode struct {
	XMLName xml.Name `xml:"nd"`
	Ref     int      `xml:"ref,attr"`
}

type Tag struct {
	XMLName xml.Name `xml:"tag"`
	Key     string   `xml:"k,attr"`
	Value   string   `xml:"v,attr"`
}

func main() {
	// Define the bounding box (LEFT, BOTTOM, RIGHT, TOP)
	bbox := "-107.075945,39.642421,-107.062191,39.650517"

	// Define the URL for the API request
	url := fmt.Sprintf("https://api.openstreetmap.org/api/0.6/map?bbox=%s", bbox)

	// Send the GET request to the OpenStreetMap API
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	// Read the response body into a []byte
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Unmarshal the XML response into an Osm struct
	var osm Osm
	if err := xml.Unmarshal(body, &osm); err != nil {
		panic(err)
	}

	// Print the IDs of all the nodes and ways in the response
	for _, node := range osm.Nodes {
		fmt.Printf("Node ID: %d, Lat: %f, Lon: %f, Name: %s\n", node.Id, node.Lat, node.Lon, node.XMLName)
	}
	for _, way := range osm.Ways {
		fmt.Printf("Way ID: %d, Name: %s\n", way.Id, way.XMLName)
	}
}
