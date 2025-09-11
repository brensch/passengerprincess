package maps

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"testing"
)

func TestGetRoute(t *testing.T) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		t.Skip("MAPS_API_KEY environment variable not set")
	}

	origin := "New York, NY"
	destination := "Boston, MA"

	result, err := GetRoute(apiKey, origin, destination)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	if result == nil {
		t.Fatal("GetRoute returned nil result")
	}

	if result.DistanceMeters <= 0 {
		t.Errorf("Expected positive distance, got %d", result.DistanceMeters)
	}

	if result.Duration <= 0 {
		t.Errorf("Expected positive duration, got %v", result.Duration)
	}

	if result.EncodedPolyline == "" {
		t.Error("Expected non-empty encoded polyline")
	}

	t.Log(result.EncodedPolyline)

}

// TestPolylineToCircles_Visualization tests the PolylineToCircles function and
// generates an HTML file to visualize the results on a Leaflet map.
func TestPolylineToCircles_Visualization(t *testing.T) {
	curDir, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current directory")
	}
	t.Logf("Current directory: %s", curDir)

	// Read the encoded polyline from a file
	encodedPolyline, err := os.ReadFile(curDir + "/polyline.txt")
	if err != nil {
		t.Fatalf("Failed to read polyline.txt: %v", err)
	}
	radius := 10000.0

	circles, err := PolylineToCircles(string(encodedPolyline), radius)
	if err != nil {
		t.Fatalf("PolylineToCircles failed: %v", err)
	}

	decodedPath, err := DecodePolyline(string(encodedPolyline))
	if err != nil {
		t.Fatalf("DecodePolyline failed: %v", err)
	}

	err = generateHTMLMap(circles, decodedPath)
	if err != nil {
		t.Fatalf("Failed to generate HTML map: %v", err)
	}

	t.Logf("Successfully generated route_visualization.html")
}

// generateHTMLMap creates an HTML file with a map visualizing the circles and polyline.
func generateHTMLMap(circles []Circle, path []Center) error {
	// Marshal circle and path data to JSON to be safely embedded in JavaScript.
	circlesJSON, err := json.Marshal(circles)
	if err != nil {
		return fmt.Errorf("failed to marshal circles: %w", err)
	}

	// Leaflet needs [[lat, lng], [lat, lng], ...] format.
	pathForJS := make([][]float64, len(path))
	for i, p := range path {
		pathForJS[i] = []float64{p.Latitude, p.Longitude}
	}
	pathJSON, err := json.Marshal(pathForJS)
	if err != nil {
		return fmt.Errorf("failed to marshal path: %w", err)
	}

	// Data to be passed to the HTML template.
	// Use template.JS to prevent the template engine from escaping the JSON string.
	// This ensures it's treated as a JavaScript literal, not a string.
	data := struct {
		CirclesJSON template.JS
		PathJSON    template.JS
	}{
		CirclesJSON: template.JS(circlesJSON),
		PathJSON:    template.JS(pathJSON),
	}

	// Create and parse the HTML template.
	tmpl, err := template.New("map").Parse(mapTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse html template: %w", err)
	}

	// Create the output file.
	file, err := os.Create("route_visualization.html")
	if err != nil {
		return fmt.Errorf("failed to create html file: %w", err)
	}
	defer file.Close()

	// Execute the template with the data and write to the file.
	return tmpl.Execute(file, data)
}

// mapTemplate is the HTML and JavaScript for the map visualization using Leaflet.
const mapTemplate = `
<!DOCTYPE html>
<html>
  <head>
    <title>Route Visualization (Leaflet)</title>
    <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"/>
    <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
    <style>
      #map {
        height: 100vh;
        width: 100%;
      }
      html, body {
        height: 100%;
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <div id="map"></div>
    <script>
      (function() {
        const circlesData = {{.CirclesJSON}};
        const pathData = {{.PathJSON}};
        
        if (pathData.length === 0) {
            console.error("No path data to display.");
            return;
        }

        // Initialize the map
        const map = L.map('map');

        // Add a tile layer (OpenStreetMap)
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            maxZoom: 19,
            attribution: '© OpenStreetMap contributors'
        }).addTo(map);

        // Draw the original polyline path
        const routePath = L.polyline(pathData, {color: 'red'}).addTo(map);

        // Adjust map bounds to fit the whole path
        if (pathData.length > 1) {
            map.fitBounds(routePath.getBounds());
        } else if (pathData.length === 1) {
            map.setView(pathData[0], 13); // Center on the single point with a zoom level of 13
        }

        // Draw the covering circles
        circlesData.forEach(circleInfo => {
          L.circle([circleInfo.center.latitude, circleInfo.center.longitude], {
            color: 'blue',
            fillColor: '#0000FF',
            fillOpacity: 0.2,
            weight: 1,
            radius: circleInfo.radius
          }).addTo(map);
        });
      })();
    </script>
  </body>
</html>
`
