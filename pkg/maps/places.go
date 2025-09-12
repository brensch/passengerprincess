package maps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	Places []*PlaceDetails `json:"places"`
}

// DisplayNameObj represents the display name object from Google Places API
type DisplayNameObj struct {
	Text         string `json:"text"`
	LanguageCode string `json:"languageCode,omitempty"`
}

// PlaceDetails represents the essential place information from Google Places API
type PlaceDetails struct {
	ID                     string          `json:"id"`
	DisplayName            *DisplayNameObj `json:"displayName"`
	FormattedAddress       *string         `json:"formattedAddress,omitempty"`
	Location               *Location       `json:"location,omitempty"`
	PrimaryType            *string         `json:"primaryType,omitempty"`
	PrimaryTypeDisplayName *DisplayNameObj `json:"primaryTypeDisplayName,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GetPlacesViaTextSearch queries the Google Places API (Text Search - New) to find all places
// matching a query within a specified circular search area. It now takes a 'circle' struct directly.
func GetPlacesViaTextSearch(ctx context.Context, apiKey, query, fieldMask string, targetCircle Circle) ([]*PlaceDetails, error) {
	reqBody := requestBody{
		TextQuery:    query,
		LocationBias: LocationBias{Circle: targetCircle},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", placesAPIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	// The FieldMask is crucial for performance and cost-effectiveness.
	// It tells Google to only return the data we absolutely need.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", apiKey)
	req.Header.Set("X-Goog-FieldMask", fieldMask)

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

	for _, p := range apiResp.Places {
		if p.ID == "" {
			return nil, fmt.Errorf("place ID is missing for a place")
		}
	}

	return apiResp.Places, nil
}

// GetPlaceDetails retrieves essential place information from Google Places API given a place ID
func GetPlaceDetails(ctx context.Context, apiKey, placeID, fieldMask string) (*PlaceDetails, error) {
	url := fmt.Sprintf("%s/%s", placeDetailsEndpoint, placeID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("X-Goog-Api-Key", apiKey)
	req.Header.Set("X-Goog-FieldMask", fieldMask)

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
	err = json.Unmarshal(bodyBytes, &placeDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response json: %w", err)
	}

	return &placeDetails, nil
}
