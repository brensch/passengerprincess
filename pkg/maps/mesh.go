package maps

import (
	"fmt"
	"math"
	"strings"
)

// CreateMesh takes lat/lon bounds and generates an efficient hexagonal grid of
// overlapping circles of the specified radius within that rectangular area.
// This version uses the correct spacing for a full COVERING, eliminating all gaps.
func CreateMesh(latMin, latMax, lonMin, lonMax float64, radius int) ([]circle, error) {
	if radius <= 0 {
		return nil, fmt.Errorf("radius must be a positive integer")
	}

	var targets []circle
	r := float64(radius)

	// --- Coordinate and Dimension Calculations ---
	const metersPerDegreeLat = 111320.0
	avgLatRad := (latMin + latMax) / 2.0 * math.Pi / 180.0
	metersPerDegreeLon := metersPerDegreeLat * math.Cos(avgLatRad)

	if metersPerDegreeLon == 0 {
		metersPerDegreeLon = metersPerDegreeLat
	}

	heightMeters := (latMax - latMin) * metersPerDegreeLat
	widthMeters := (lonMax - lonMin) * metersPerDegreeLon

	// --- Correct Hexagonal COVERING Spacing ---
	// The optimal distance between centers for a hexagonal covering lattice is sqrt(3) * r.
	distanceBetweenCenters := r * math.Sqrt(3)

	// dx is the horizontal distance between centers in the same row.
	dx := distanceBetweenCenters

	// dy is the vertical distance between rows.
	dy := distanceBetweenCenters * math.Sqrt(3) / 2.0 // This simplifies to 1.5 * r

	// xOffset is the horizontal shift for alternating rows.
	xOffset := dx / 2.0

	// --- Grid Generation ---
	row := 0
	for y := 0.0; y <= heightMeters; y += dy {
		currentXOffset := 0.0
		if row%2 != 0 {
			currentXOffset = xOffset
		}

		for x := currentXOffset; x <= widthMeters; x += dx {
			latOffset := y / metersPerDegreeLat
			lonOffset := x / metersPerDegreeLon

			targets = append(targets, circle{
				Center: center{
					Latitude:  latMin + latOffset,
					Longitude: lonMin + lonOffset,
				},
				Radius: float64(radius),
			})
		}
		row++
	}

	return targets, nil
}

func VisualiseMeshHTML(lat, lon float64, targets []circle) string {
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

	return html
}
