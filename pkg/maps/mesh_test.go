package maps

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
)

func TestCreateMesh(t *testing.T) {
	lat := 40.7128
	lon := -74.0060
	radius := 1000

	latMin := lat - float64(radius)/111320
	latMax := lat + float64(radius)/111320
	lonMin := lon - float64(radius)/(111320*math.Cos(lat*math.Pi/180))
	lonMax := lon + float64(radius)/(111320*math.Cos(lat*math.Pi/180))

	targets, err := CreateMesh(latMin, latMax, lonMin, lonMax, radius)
	if err != nil {
		t.Fatalf("CreateMesh failed: %v", err)
	}

	if len(targets) == 0 {
		t.Fatal("CreateMesh returned no targets")
	}

	for _, target := range targets {
		if target.Center.Latitude == 0 && target.Center.Longitude == 0 {
			t.Error("Expected non-zero lat/lon for search targets")
		}
	}
}

// TestCreateMeshAndVisualise runs the create mesh over a fixed 5km x 5km area around Mountain View then generates an HTML file
// for visualization
func TestCreateMeshAndVisualise(t *testing.T) {
	lat := 37.3861
	lon := -122.0839
	radius := 1000

	// Fixed bounds for a 5km x 5km area around Mountain View
	latMin := 37.2
	latMax := 37.9
	lonMin := -122.6
	lonMax := -121.8

	targets, err := CreateMesh(latMin, latMax, lonMin, lonMax, radius)
	if err != nil {
		t.Fatalf("CreateMesh failed: %v", err)
	}

	if len(targets) == 0 {
		t.Fatal("CreateMesh returned no targets")
	}

	t.Log(len(targets))

	// Generate circles JSON for Leaflet visualization
	var builder strings.Builder
	builder.WriteString("[")
	for i, target := range targets {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf(`{"lat": %f, "lon": %f, "radius": %f}`, target.Center.Latitude, target.Center.Longitude, target.Radius))
	}
	builder.WriteString("]")
	circlesJSON := builder.String()

	// Generate HTML file with embedded JSON
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>Circle Visualization</title>
  <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
  <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
</head>
<body>
  <div id="map" style="height: 600px;"></div>
  <script>
    var map = L.map('map').setView([%f, %f], 12);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png').addTo(map);

    var circles = %s;

    circles.forEach(circle => {
      L.circle([circle.lat, circle.lon], {
        color: 'blue',
        fillColor: 'blue',
        fillOpacity: 0.2,
        radius: circle.radius
      }).addTo(map);
    });
  </script>
</body>
</html>`, lat, lon, circlesJSON)

	// Write HTML to file
	err = os.WriteFile("mesh.html", []byte(html), 0644)
	if err != nil {
		t.Fatalf("Failed to write HTML file: %v", err)
	}

	t.Logf("HTML file 'mesh.html' generated with %d circles. Open it directly in your browser to visualize overlapping circles.", len(targets))
}
