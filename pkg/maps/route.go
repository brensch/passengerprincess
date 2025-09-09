package maps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// The API endpoint for computing routes.
	routesAPIEndpoint = "https://routes.googleapis.com/directions/v2:computeRoutes"
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
