package main

import (
	"math"
)

const EarthRadiusKM float64 = 6371.0     // Earth radius in kilometers
const EarthRadiusFT float64 = 20902231.0 // Earth radius in feet

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

func haversine(lat1, lon1, lat2, lon2, EarthRadius float64) float64 {
	// Convert decimal degrees to radians
	lat1, lon1 = toRadians(lat1), toRadians(lon1)
	lat2, lon2 = toRadians(lat2), toRadians(lon2)

	// Calculate the differences between the latitudes and longitudes
	dLat := lat2 - lat1
	dLon := lon2 - lon1

	// Calculate the Haversine formula
	a := math.Pow(math.Sin(dLat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(dLon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Calculate the distance
	distance := EarthRadius * c
	return distance
}
