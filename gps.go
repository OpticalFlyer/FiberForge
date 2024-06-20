package main

import (
	"bufio"
	//"fmt"
	"sync"

	"github.com/adrianmo/go-nmea"
	"github.com/tarm/serial"
)

type GPS struct {
	done       chan struct{}
	wg         sync.WaitGroup
	serialPort *serial.Port
	running    bool
	latitude   float64
	longitude  float64
	altitude   float64
	speed      float64
	course     float64
	HDOP       float64
}

func NewGPS() *GPS {
	return &GPS{running: false}
}

func (gps *GPS) StartGPS() error {
	// Configure the serial port
	config := &serial.Config{
		Name: "/dev/cu.usbserial-0213AF16", // or "COM3" for Windows
		Baud: 9600,
	}

	// Open the serial port
	port, err := serial.OpenPort(config)
	if err != nil {
		return err
	}

	// Assign the opened port to the GPS struct
	gps.serialPort = port

	// Read data from the serial port
	scanner := bufio.NewScanner(port)

	// Create a new 'done' channel
	gps.done = make(chan struct{})

	gps.wg.Add(1)
	go func() {
		defer gps.wg.Done()

		for {
			select {
			case <-gps.done:
				return
			default:
				if scanner.Scan() {
					line := scanner.Text()

					// Parse NMEA sentence
					sentence, err := nmea.Parse(line)
					if err != nil {
						continue // Ignore invalid sentences
					}

					// Print GPS data based on sentence type
					switch s := sentence.(type) {
					case nmea.GGA:
						//fmt.Printf("Time: %s, Latitude: %f, Longitude: %f, Altitude: %f, HDOP: %f\n", s.Time, s.Latitude, s.Longitude, s.Altitude, s.HDOP)
						gps.latitude = s.Latitude
						gps.longitude = s.Longitude
						gps.altitude = s.Altitude
						gps.HDOP = s.HDOP
					case nmea.RMC:
						//fmt.Printf("Time: %s, Latitude: %f, Longitude: %f, Speed: %f, Course: %f\n", s.Time, s.Latitude, s.Longitude, s.Speed, s.Course)
						gps.latitude = s.Latitude
						gps.longitude = s.Longitude
						gps.speed = s.Speed
						gps.course = s.Course
					case nmea.GLL:
						//fmt.Printf("Time: %s, Latitude: %f, Longitude: %f\n", s.Time, s.Latitude, s.Longitude)
						gps.latitude = s.Latitude
						gps.longitude = s.Longitude
					}
				}
			}
		}
	}()

	gps.running = true

	return nil
}

func (gps *GPS) StopGPS() {
	close(gps.done)
	gps.wg.Wait() // Wait for the Go routine to exit

	// Close the serial port
	if gps.serialPort != nil {
		gps.serialPort.Close()
	}

	gps.running = false
}
