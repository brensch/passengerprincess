package maps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Making the endpoint and client package-level variables allows us to
// mock them during testing without changing the function's signature.
var (
	placesAPIEndpoint = "https://places.googleapis.com/v1/places:searchText"
	httpClient        = &http.Client{}
)

// requestBody represents the JSON structure for the Google Places API searchText request.
type requestBody struct {
	TextQuery    string       `json:"textQuery"`
	LocationBias locationBias `json:"locationBias"`
}

type locationBias struct {
	Circle circle `json:"circle"`
}

type circle struct {
	Center center  `json:"center"`
	Radius float64 `json:"radius"`
}

type center struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// apiResponse defines the structure for unmarshalling the API's JSON response.
// We only care about the place IDs.
type apiResponse struct {
	Places []place `json:"places"`
}

type place struct {
	ID string `json:"id"`
}

// GetPlaceIDsViaTextSearch queries the Google Places API (Text Search - New) to find all place IDs
// matching a query within a specified circular search area. It now takes a 'circle' struct directly.
func GetPlaceIDsViaTextSearch(apiKey, query string, targetCircle circle) ([]string, error) {
	reqBody := requestBody{
		TextQuery:    query,
		LocationBias: locationBias{Circle: targetCircle},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", placesAPIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	// The FieldMask is crucial for performance and cost-effectiveness.
	// It tells Google to only return the data we absolutely need.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", apiKey)
	req.Header.Set("X-Goog-FieldMask", "places.id")

	// 5. Execute the request using the package-level client.
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Google Places API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google places api returned an error. status: %s, body: %s", resp.Status, string(bodyBytes))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response json: %w", err)
	}

	placeIDs := make([]string, 0, len(apiResp.Places))
	for _, p := range apiResp.Places {
		placeIDs = append(placeIDs, p.ID)
	}

	return placeIDs, nil
}
