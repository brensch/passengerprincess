package maps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// earthRadiusMeters is the mean radius of Earth in meters, used for distance calculations.
	earthRadiusMeters = 6371000
)

// --- Custom Result Struct ---

// EncodedPolyline contains the string representation of the route path.
type EncodedPolyline struct {
	EncodedPolyline string `json:"encodedPolyline"`
}

// RouteInfo is a cleaner, consolidated structure for returning the final result.
type RouteInfo struct {
	DistanceMeters  int
	Duration        time.Duration
	EncodedPolyline string
	// Enhanced data for traffic-aware routing
	TravelAdvisory RouteTravelAdvisory `json:"travelAdvisory,omitempty"`
}

// Enhanced route structures for traffic-aware routing
type EnhancedRouteRequest struct {
	Origin            LocationRequest `json:"origin"`
	Destination       LocationRequest `json:"destination"`
	TravelMode        string          `json:"travelMode"`
	RoutingPreference string          `json:"routingPreference,omitempty"`
	ExtraComputations []string        `json:"extraComputations,omitempty"`
	PolylineQuality   string          `json:"polylineQuality,omitempty"`
	PolylineEncoding  string          `json:"polylineEncoding,omitempty"`
	DepartureTime     string          `json:"departureTime,omitempty"`
}

type LocationRequest struct {
	Address string     `json:"address,omitempty"`
	LatLng  *LatLngReq `json:"latLng,omitempty"`
}

type LatLngReq struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type EnhancedRouteResponse struct {
	Routes []EnhancedRoute `json:"routes"`
}

type EnhancedRoute struct {
	Polyline       EncodedPolyline     `json:"polyline"`
	Legs           []EnhancedRouteLeg  `json:"legs"`
	TravelAdvisory RouteTravelAdvisory `json:"travelAdvisory,omitempty"`
	Duration       string              `json:"duration"`
	DistanceMeters int                 `json:"distanceMeters"`
}

type EnhancedRouteLeg struct {
	Polyline       EncodedPolyline        `json:"polyline"`
	Steps          []EnhancedRouteStep    `json:"steps"`
	TravelAdvisory RouteLegTravelAdvisory `json:"travelAdvisory,omitempty"`
	Duration       string                 `json:"duration"`
	DistanceMeters int                    `json:"distanceMeters"`
}

type EnhancedRouteStep struct {
	Polyline       EncodedPolyline `json:"polyline"`
	StaticDuration string          `json:"staticDuration"`
	DistanceMeters int             `json:"distanceMeters"`
}

type RouteTravelAdvisory struct {
	SpeedReadingIntervals []SpeedReadingInterval `json:"speedReadingIntervals,omitempty"`
}

type RouteLegTravelAdvisory struct {
	SpeedReadingIntervals []SpeedReadingInterval `json:"speedReadingIntervals,omitempty"`
}

type SpeedReadingInterval struct {
	StartPolylinePointIndex int    `json:"startPolylinePointIndex,omitempty"`
	EndPolylinePointIndex   int    `json:"endPolylinePointIndex"`
	Speed                   string `json:"speed"`
}

// GetRoute takes an API key and two location strings, then returns
// information about the route with traffic-aware routing.
func GetRoute(apiKey, origin, destination string) (*RouteInfo, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is missing. Please set the GOOGLE_MAPS_API_KEY environment variable")
	}

	// Get enhanced route data with traffic information
	enhancedRoute, err := getEnhancedRouteData(apiKey, origin, destination)
	if err != nil {
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	if len(enhancedRoute.Routes) == 0 {
		return nil, fmt.Errorf("no route data returned")
	}

	route := enhancedRoute.Routes[0]

	// Parse the duration string
	durationSeconds := parseDurationString(route.Duration)

	return &RouteInfo{
		DistanceMeters:  route.DistanceMeters,
		Duration:        time.Duration(durationSeconds) * time.Second,
		EncodedPolyline: route.Polyline.EncodedPolyline,
		TravelAdvisory:  route.TravelAdvisory,
	}, nil
}

// getEnhancedRouteData fetches traffic-aware route data from Google Routes API
func getEnhancedRouteData(apiKey, origin, destination string) (*EnhancedRouteResponse, error) {
	routesRequest := EnhancedRouteRequest{
		Origin: LocationRequest{
			Address: origin,
		},
		Destination: LocationRequest{
			Address: destination,
		},
		TravelMode:        "DRIVE",
		RoutingPreference: "TRAFFIC_AWARE_OPTIMAL",
		ExtraComputations: []string{"TRAFFIC_ON_POLYLINE"},
		PolylineQuality:   "HIGH_QUALITY",
		PolylineEncoding:  "ENCODED_POLYLINE",
		DepartureTime:     time.Now().Add(1 * time.Minute).Format(time.RFC3339),
	}

	requestBody, err := json.Marshal(routesRequest)
	if err != nil {
		return nil, err
	}

	apiURL := "https://routes.googleapis.com/directions/v2:computeRoutes"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", apiKey)
	req.Header.Set("X-Goog-FieldMask", "routes.duration,routes.distanceMeters,routes.polyline.encodedPolyline,routes.travelAdvisory.speedReadingIntervals")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("routes API error: %s", string(body))
	}

	var routesData EnhancedRouteResponse
	if err := json.Unmarshal(body, &routesData); err != nil {
		return nil, err
	}

	return &routesData, nil
}

// parseDurationString parses duration strings like "2420s" to seconds
func parseDurationString(durationStr string) int {
	// Parse duration strings like "2420s" to seconds
	if strings.HasSuffix(durationStr, "s") {
		if val, err := strconv.Atoi(strings.TrimSuffix(durationStr, "s")); err == nil {
			return val
		}
	}
	// Fallback: try to parse as plain number
	if val, err := strconv.Atoi(durationStr); err == nil {
		return val
	}
	return 0
}

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
