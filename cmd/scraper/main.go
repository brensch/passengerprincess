package main

import (
	"os"

	"github.com/brensch/passengerprincess/pkg/maps"
)

func main() {
	// Fixed bounds for a 5km x 5km area around Mountain View
	latMin := 37.2
	latMax := 37.9
	lonMin := -122.6
	lonMax := -121.8
	radius := 1000

	targets, err := maps.CreateMesh(latMin, latMax, lonMin, lonMax, radius)
	if err != nil {
		panic(err)
	}

	if len(targets) == 0 {
		panic("CreateMesh returned no targets")
	}

	// Generate HTML using VisualiseMeshHTML
	html := maps.VisualiseMeshHTML(latMax, lonMin, targets)

	// Write HTML to file
	err = os.WriteFile("mesh.html", []byte(html), 0644)
	if err != nil {
		panic(err)
	}

}
