package maps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	// The API endpoint for computing routes.
	routesAPIEndpoint = "https://routes.googleapis.com/directions/v2:computeRoutes"
	// earthRadiusMeters is the mean radius of Earth in meters, used for distance calculations.
	earthRadiusMeters = 6371000
)

// --- Structs for JSON Unmarshalling ---
// These structs are designed to capture the relevant parts of the Google Maps API response.

// ComputeRoutesResponse is the top-level structure of the API's JSON response.
type ComputeRoutesResponse struct {
	Routes []Route `json:"routes"`
}

// Route represents a single route from origin to destination.
type Route struct {
	DistanceMeters int             `json:"distanceMeters"`
	Duration       string          `json:"duration"`
	Polyline       EncodedPolyline `json:"polyline"`
}

// EncodedPolyline contains the string representation of the route path.
type EncodedPolyline struct {
	EncodedPolyline string `json:"encodedPolyline"`
}

// --- Custom Result Struct ---

// RouteInfo is a cleaner, consolidated structure for returning the final result.
type RouteInfo struct {
	DistanceMeters  int
	Duration        time.Duration
	EncodedPolyline string
}

// GetRoute takes an API key and two location strings, then returns
// information about the most basic (and thus cheapest) route.
func GetRoute(apiKey, origin, destination string) (*RouteInfo, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is missing. Please set the GOOGLE_MAPS_API_KEY environment variable")
	}

	// 1. Construct the request payload as a map.
	// This will be marshalled into a JSON string.
	requestBodyMap := map[string]interface{}{
		"origin": map[string]string{
			"address": origin,
		},
		"destination": map[string]string{
			"address": destination,
		},
		"travelMode": "DRIVE",
		// By NOT requesting any advanced features (like computeAlternativeRoutes,
		// routeModifiers, languageCode, etc.), we ensure we are billed for the
		// basic "Routes: Compute Routes Essentials" SKU.
	}

	// Marshal the map into a JSON byte slice.
	jsonBody, err := json.Marshal(requestBodyMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 2. Create the HTTP POST request.
	req, err := http.NewRequest("POST", routesAPIEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create new http request: %w", err)
	}

	// 3. Set the required headers for authentication and efficiency.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", apiKey)
	// Use a FieldMask to request only the specific fields we need. This is a best
	// practice that can reduce response latency and cost.
	req.Header.Set("X-Goog-FieldMask", "routes.distanceMeters,routes.duration,routes.polyline.encodedPolyline")

	// 4. Execute the request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// 5. Read the response body.
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for non-200 status codes.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 6. Unmarshal the JSON response into our Go structs.
	var apiResponse ComputeRoutesResponse
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response JSON: %w", err)
	}

	// Ensure at least one route was returned.
	if len(apiResponse.Routes) == 0 {
		return nil, fmt.Errorf("no routes found between origin and destination")
	}

	// 8. Parse the duration string (e.g., "1800s") into a time.Duration type.
	// The 's' suffix must be removed first.
	durationString := apiResponse.Routes[0].Duration
	parsedDuration, err := time.ParseDuration(strings.TrimSuffix(durationString, "s") + "s")
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration string '%s': %w", durationString, err)
	}

	// 9. Consolidate the results into our custom RouteInfo struct.
	result := &RouteInfo{
		DistanceMeters:  apiResponse.Routes[0].DistanceMeters,
		Duration:        parsedDuration,
		EncodedPolyline: apiResponse.Routes[0].Polyline.EncodedPolyline,
	}

	return result, nil
}

// --- NEW FUNCTION: Polyline to Circles ---

// interpolatePoints takes a list of points and inserts additional points at regular intervals along the path.
func interpolatePoints(points []Center, intervalMeters float64) []Center {
	var densePoints []Center
	if len(points) == 0 {
		return densePoints
	}
	densePoints = append(densePoints, points[0])
	for i := 0; i < len(points)-1; i++ {
		p1 := points[i]
		p2 := points[i+1]
		dist := haversineDistance(p1, p2)
		if dist <= intervalMeters {
			densePoints = append(densePoints, p2)
			continue
		}
		numSegments := int(math.Ceil(dist / intervalMeters))
		for j := 1; j < numSegments; j++ {
			fraction := float64(j) / float64(numSegments)
			lat := p1.Latitude + fraction*(p2.Latitude-p1.Latitude)
			lng := p1.Longitude + fraction*(p2.Longitude-p1.Longitude)
			densePoints = append(densePoints, Center{Latitude: lat, Longitude: lng})
		}
		densePoints = append(densePoints, p2)
	}
	return densePoints
}

// PolylineToCircles takes an encoded polyline string and a radius, then returns
// a series of Circle objects that completely cover the route.
func PolylineToCircles(encodedPolyline string, radius float64) ([]Circle, error) {
	if radius <= 0 {
		return nil, fmt.Errorf("radius must be a positive number")
	}

	points, err := DecodePolyline(encodedPolyline)
	if err != nil {
		return nil, fmt.Errorf("failed to decode polyline: %w", err)
	}

	points = interpolatePoints(points, 100.0) // Interpolate points every 100 meters for better coverage

	if len(points) == 0 {
		return []Circle{}, nil // Return empty slice if polyline has no points
	}

	var circles []Circle
	// Start with a circle at the very first point of the route.
	firstCircle := Circle{Center: points[0], Radius: radius}
	circles = append(circles, firstCircle)
	lastCircleCenter := points[0]

	// Iterate through the rest of the points to cover the path.
	for i := 1; i < len(points); i++ {
		currentPoint := points[i]
		// Calculate the distance from the center of the last placed circle.
		distance := haversineDistance(lastCircleCenter, currentPoint)

		// If the current point is outside the last circle's radius, we need to
		// place a new circle to cover this segment of the route.
		if distance > radius {
			newCircle := Circle{Center: currentPoint, Radius: radius}
			circles = append(circles, newCircle)
			lastCircleCenter = currentPoint
		}
	}

	// Ensure the last point has a circle
	if lastCircleCenter != points[len(points)-1] {
		newCircle := Circle{Center: points[len(points)-1], Radius: radius}
		circles = append(circles, newCircle)
	}

	return circles, nil
}

// DecodePolyline converts an encoded polyline string into a slice of geographic points.
func DecodePolyline(encoded string) ([]Center, error) {
	var points []Center
	var lat, lng, index int

	for index < len(encoded) {
		// Decode latitude
		var change, latChange int
		var shift uint = 0
		for {
			if index >= len(encoded) {
				return nil, fmt.Errorf("polyline string is malformed")
			}
			b := int(encoded[index]) - 63
			index++
			change |= (b & 0x1f) << shift
			shift += 5
			if b&0x20 == 0 {
				break
			}
		}
		if change&1 == 1 {
			latChange = ^(change >> 1)
		} else {
			latChange = change >> 1
		}
		lat += latChange

		// Decode longitude
		change, shift = 0, 0
		var lngChange int
		for {
			if index >= len(encoded) {
				return nil, fmt.Errorf("polyline string is malformed")
			}
			b := int(encoded[index]) - 63
			index++
			change |= (b & 0x1f) << shift
			shift += 5
			if b&0x20 == 0 {
				break
			}
		}
		if change&1 == 1 {
			lngChange = ^(change >> 1)
		} else {
			lngChange = change >> 1
		}
		lng += lngChange

		points = append(points, Center{
			Latitude:  float64(lat) / 1e5,
			Longitude: float64(lng) / 1e5,
		})
	}

	return points, nil
}

// haversineDistance calculates the shortest distance over the earth's surface
// between two geographic points in meters.
func haversineDistance(p1, p2 Center) float64 {
	lat1 := p1.Latitude * math.Pi / 180
	lon1 := p1.Longitude * math.Pi / 180
	lat2 := p2.Latitude * math.Pi / 180
	lon2 := p2.Longitude * math.Pi / 180

	dLat := lat2 - lat1
	dLon := lon2 - lon1

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}
