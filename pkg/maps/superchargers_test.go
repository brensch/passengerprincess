package maps

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brensch/passengerprincess/pkg/db"
	"gorm.io/gorm/logger"
)

func TestGetSuperchargersOnRoute(t *testing.T) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		t.Skip("MAPS_API_KEY not set")
	}

	// Create database file in test-databases directory
	timestamp := time.Now().Format("20060102_150405")
	dbFile := filepath.Join("test-databases", fmt.Sprintf("TestGetSuperchargersOnRoute_%s.db", timestamp))

	// Ensure the directory exists
	os.MkdirAll("test-databases", 0755)

	// Initialize file-based database
	err := db.Initialize(&db.Config{
		DatabasePath: dbFile,
		LogLevel:     logger.Error,
	})
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	t.Logf("Database created at: %s", dbFile)

	broker := db.GetDefaultService()

	start := "mountain view, california"
	end := "morgan hill, california"

	t.Logf("Finding superchargers on route from %s to %s", start, end)

	result, err := GetSuperchargersOnRoute(context.Background(), broker, apiKey, start, end)
	if err != nil {
		t.Fatalf("GetSuperchargersOnRoute failed: %v", err)
	}

	route := result.Route
	superchargers := result.Superchargers
	circles := result.SearchCircles

	t.Logf("Found %d superchargers on route :", len(superchargers))
	for i, sc := range superchargers {
		t.Logf("%d: %s at %s (Lat: %.6f, Lon: %.6f)", i+1, sc.Name, sc.Address, sc.Latitude, sc.Longitude)
	}

	// Generate HTML visualization
	err = generateSuperchargerHTMLMap(route, superchargers, circles)
	if err != nil {
		t.Fatalf("Failed to generate HTML map: %v", err)
	}
	t.Logf("Successfully generated supercharger_route_visualization.html")

	t.Logf("running again to check caching...")
	resultCached, err := GetSuperchargersOnRoute(context.Background(), broker, apiKey, start, end)
	if err != nil {
		t.Fatalf("GetSuperchargersOnRoute failed: %v", err)
	}

	superchargersCached := resultCached.Superchargers

	t.Logf("Found %d superchargers on route from Salinas to Gonzales:", len(superchargersCached))
	for i, sc := range superchargersCached {
		t.Logf("%d: %s at %s (Lat: %.6f, Lon: %.6f)", i+1, sc.Name, sc.Address, sc.Latitude, sc.Longitude)
	}

	// make sure they're the same superchargers
	for _, sc := range superchargers {
		found := false
		for _, scCached := range superchargersCached {
			if sc.PlaceID == scCached.PlaceID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Supercharger %s not found in cached results", sc.PlaceID)
		}
	}

	if len(superchargers) == 0 {
		t.Error("Expected to find at least one supercharger on the route")
	}

	// Close the database after all operations are complete
	defer db.Close()
}

// generateSuperchargerHTMLMap creates an HTML file with a map visualizing the route and superchargers.
func generateSuperchargerHTMLMap(route *RouteInfo, superchargers []*db.Supercharger, circles []Circle) error {
	// Decode the polyline to get the path
	decodedPath, err := DecodePolyline(route.EncodedPolyline)
	if err != nil {
		return fmt.Errorf("failed to decode polyline: %w", err)
	}

	// Convert path to JavaScript format
	pathForJS := make([][]float64, len(decodedPath))
	for i, p := range decodedPath {
		pathForJS[i] = []float64{p.Latitude, p.Longitude}
	}
	pathJSON, err := json.Marshal(pathForJS)
	if err != nil {
		return fmt.Errorf("failed to marshal path: %w", err)
	}

	// Convert superchargers to JavaScript format
	superchargersForJS := make([]map[string]interface{}, len(superchargers))
	for i, sc := range superchargers {
		superchargersForJS[i] = map[string]interface{}{
			"name":      sc.Name,
			"address":   sc.Address,
			"latitude":  sc.Latitude,
			"longitude": sc.Longitude,
			"placeId":   sc.PlaceID,
		}
	}
	superchargersJSON, err := json.Marshal(superchargersForJS)
	if err != nil {
		return fmt.Errorf("failed to marshal superchargers: %w", err)
	}

	// Convert circles to JavaScript format
	circlesJSON, err := json.Marshal(circles)
	if err != nil {
		return fmt.Errorf("failed to marshal circles: %w", err)
	}

	// Data to be passed to the HTML template
	data := struct {
		PathJSON          template.JS
		SuperchargersJSON template.JS
		CirclesJSON       template.JS
		RouteDistance     int
		RouteDistanceKm   float64
		RouteDuration     string
	}{
		PathJSON:          template.JS(pathJSON),
		SuperchargersJSON: template.JS(superchargersJSON),
		CirclesJSON:       template.JS(circlesJSON),
		RouteDistance:     route.DistanceMeters,
		RouteDistanceKm:   float64(route.DistanceMeters) / 1000.0,
		RouteDuration:     route.Duration.String(),
	}

	// Create and parse the HTML template
	tmpl, err := template.New("superchargerMap").Parse(superchargerMapTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse html template: %w", err)
	}

	// Create the output file
	file, err := os.Create("supercharger_route_visualization.html")
	if err != nil {
		return fmt.Errorf("failed to create html file: %w", err)
	}
	defer file.Close()

	// Execute the template with the data and write to the file
	return tmpl.Execute(file, data)
}

// superchargerMapTemplate is the HTML and JavaScript for the supercharger map visualization using Leaflet.
const superchargerMapTemplate = `
<!DOCTYPE html>
<html>
  <head>
    <title>Supercharger Route Visualization</title>
    <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"/>
    <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
    <style>
      #map {
        height: 90vh;
        width: 100%;
      }
      #info {
        height: 10vh;
        padding: 10px;
        background-color: #f0f0f0;
        border-bottom: 1px solid #ccc;
        font-family: Arial, sans-serif;
      }
      html, body {
        height: 100%;
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <div id="info">
      <h3>Route Information</h3>
      <p><strong>Distance:</strong> {{.RouteDistance}} meters ({{printf "%.2f" .RouteDistanceKm}} km)</p>
      <p><strong>Duration:</strong> {{.RouteDuration}}</p>
      <p><strong>Superchargers Found:</strong> <span id="charger-count"></span></p>
      <p><strong>Search Circles:</strong> <span id="circle-count"></span></p>
    </div>
    <div id="map"></div>
    <script>
      (function() {
        const pathData = {{.PathJSON}};
        const superchargersData = {{.SuperchargersJSON}};
        const circlesData = {{.CirclesJSON}};
        
        // Update counts
        document.getElementById('charger-count').textContent = superchargersData.length;
        document.getElementById('circle-count').textContent = circlesData.length;
        
        if (pathData.length === 0) {
            console.error("No path data to display.");
            return;
        }

        // Initialize the map with a default view first
        const map = L.map('map').setView([47.0, -122.4], 10); // Set a default center and zoom

        // Add a tile layer (OpenStreetMap)
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            maxZoom: 19,
            attribution: '© OpenStreetMap contributors'
        }).addTo(map);

        // Draw the route path
        const routePath = L.polyline(pathData, {
            color: 'blue',
            weight: 4,
            opacity: 0.7
        }).addTo(map);

        // Add search circles
        const circleMarkers = [];
        circlesData.forEach((circleInfo, index) => {
          const circle = L.circle([circleInfo.center.latitude, circleInfo.center.longitude], {
            color: 'green',
            fillColor: '#00FF00',
            fillOpacity: 0.1,
            weight: 2,
            radius: circleInfo.radius
          }).addTo(map).bindPopup('<b>Search Circle ' + (index + 1) + '</b><br>Radius: ' + circleInfo.radius + 'm');
          circleMarkers.push(circle);
        });

        // Add start and end markers
        const routeMarkers = [];
        if (pathData.length > 0) {
            const startMarker = L.marker(pathData[0]).addTo(map)
                .bindPopup('<b>Start</b>')
                .openPopup();
            routeMarkers.push(startMarker);
            
            if (pathData.length > 1) {
                const endMarker = L.marker(pathData[pathData.length - 1]).addTo(map)
                    .bindPopup('<b>End</b>');
                routeMarkers.push(endMarker);
            }
        }

        // Add supercharger markers
        const superchargerMarkers = [];
        superchargersData.forEach(charger => {
            const marker = L.marker([charger.latitude, charger.longitude], {
                icon: L.icon({
                    iconUrl: 'data:image/svg+xml;base64,' + btoa(
                        '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="red">' +
                        '<path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/>' +
                        '</svg>'
                    ),
                    iconSize: [32, 32],
                    iconAnchor: [16, 32],
                    popupAnchor: [0, -32]
                })
            }).addTo(map)
                .bindPopup(
                    '<div>' +
                    '<h4>' + charger.name + '</h4>' +
                    '<p><strong>Address:</strong> ' + charger.address + '</p>' +
                    '<p><strong>Coordinates:</strong> ' + charger.latitude.toFixed(6) + ', ' + charger.longitude.toFixed(6) + '</p>' +
                    '<p><strong>Place ID:</strong> ' + charger.placeId + '</p>' +
                    '</div>'
                );
            superchargerMarkers.push(marker);
        });

        // Fit map bounds to show the entire route and all elements
        const allLayers = [routePath, ...routeMarkers, ...superchargerMarkers, ...circleMarkers];
        const group = new L.featureGroup(allLayers);
        
        if (group.getLayers().length > 0) {
            map.fitBounds(group.getBounds().pad(0.1));
        }
      })();
    </script>
  </body>
</html>
`
