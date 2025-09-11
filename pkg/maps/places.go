package maps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/brensch/passengerprincess/pkg/db"
	"gorm.io/gorm"
)

// Making the endpoint and client package-level variables allows us to
// mock them during testing without changing the function's signature.
var (
	placesAPIEndpoint    = "https://places.googleapis.com/v1/places:searchText"
	placeDetailsEndpoint = "https://places.googleapis.com/v1/places"
	httpClient           = &http.Client{}
)

// requestBody represents the JSON structure for the Google Places API searchText request.
type requestBody struct {
	TextQuery    string       `json:"textQuery"`
	LocationBias LocationBias `json:"locationBias"`
}

type LocationBias struct {
	Circle Circle `json:"circle"`
}

type Circle struct {
	Center Center  `json:"center"`
	Radius float64 `json:"radius"`
}

type Center struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// apiResponse defines the structure for unmarshalling the API's JSON response.
// We only care about the place IDs.
type apiResponse struct {
	Places []Place `json:"places"`
}

type Place struct {
	ID string `json:"id"`
}

// PlaceDetails represents the essential place information from Google Places API
type PlaceDetails struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	FormattedAddress string   `json:"formattedAddress,omitempty"`
	Location         Location `json:"location,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GetPlaceIDsViaTextSearch queries the Google Places API (Text Search - New) to find all place IDs
// matching a query within a specified circular search area. It now takes a 'circle' struct directly.
func GetPlaceIDsViaTextSearch(apiKey, query string, targetCircle Circle) ([]string, error) {
	reqBody := requestBody{
		TextQuery:    query,
		LocationBias: LocationBias{Circle: targetCircle},
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

// GetPlaceDetails retrieves essential place information from Google Places API given a place ID
func GetPlaceDetails(apiKey, placeID string) (*PlaceDetails, error) {
	url := fmt.Sprintf("%s/%s", placeDetailsEndpoint, placeID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("X-Goog-Api-Key", apiKey)
	req.Header.Set("X-Goog-FieldMask", "id,name,formattedAddress,location")

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

	var placeDetails PlaceDetails
	if err := json.Unmarshal(bodyBytes, &placeDetails); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response json: %w", err)
	}

	return &placeDetails, nil
}

// GetPlaceDetailsWithCache retrieves place details with database caching
// First checks the database, then falls back to API if not found
func GetPlaceDetailsWithCache(broker *db.Service, apiKey, placeID string) (*PlaceDetails, error) {
	// First try to get from database
	place, err := broker.Place.GetByID(placeID)
	if err == nil {
		// Found in database, convert to PlaceDetails format
		return &PlaceDetails{
			ID:               place.PlaceID,
			Name:             place.Name,
			FormattedAddress: place.Address,
			Location: Location{
				Latitude:  place.Latitude,
				Longitude: place.Longitude,
			},
		}, nil
	}

	// Check if error is "not found" (expected when place doesn't exist in DB)
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query place from database: %w", err)
	}

	// Not found in database, fetch from API
	placeDetails, err := GetPlaceDetails(apiKey, placeID)
	if err != nil {
		return nil, err
	}

	// Store in database for future use
	dbPlace := &db.Place{
		PlaceID:   placeDetails.ID,
		Name:      placeDetails.Name,
		Address:   placeDetails.FormattedAddress,
		Latitude:  placeDetails.Location.Latitude,
		Longitude: placeDetails.Location.Longitude,
	}

	if err := broker.Place.Create(dbPlace); err != nil {
		// Log the error but don't fail the request since we already have the data
		fmt.Printf("Warning: failed to cache place %s in database: %v\n", placeID, err)
	}

	return placeDetails, nil
}
