package maps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"

	"github.com/brensch/passengerprincess/pkg/db"
)

// Making the endpoint and client package-level variables allows us to
// mock them during testing without changing the function's signature.
var (
	placesAPIEndpoint    = "https://places.googleapis.com/v1/places:searchText"
	placesNearbyEndpoint = "https://places.googleapis.com/v1/places:searchNearby"
	placeDetailsEndpoint = "https://places.googleapis.com/v1/places"
	httpClient           = &http.Client{}
)

// requestBody represents the JSON structure for the Google Places API searchText request.
type requestBody struct {
	TextQuery           string               `json:"textQuery"`
	LocationBias        *LocationBias        `json:"locationBias,omitempty"`
	LocationRestriction *LocationRestriction `json:"locationRestriction,omitempty"`
	PageToken           *string              `json:"pageToken,omitempty"`
}

// nearbyRequestBody represents the JSON structure for the Google Places API searchNearby request.
type nearbyRequestBody struct {
	IncludedTypes        []string                   `json:"includedTypes,omitempty"`
	ExcludedTypes        []string                   `json:"excludedTypes,omitempty"`
	IncludedPrimaryTypes []string                   `json:"includedPrimaryTypes,omitempty"`
	ExcludedPrimaryTypes []string                   `json:"excludedPrimaryTypes,omitempty"`
	MaxResultCount       *int                       `json:"maxResultCount,omitempty"`
	LocationRestriction  *NearbyLocationRestriction `json:"locationRestriction"`
	RankPreference       *string                    `json:"rankPreference,omitempty"`
	LanguageCode         *string                    `json:"languageCode,omitempty"`
	RegionCode           *string                    `json:"regionCode,omitempty"`
}

// NearbyLocationRestriction represents the location restriction for nearby search
type NearbyLocationRestriction struct {
	Circle Circle `json:"circle"`
}

type LocationBias struct {
	Circle Circle `json:"circle"`
}

type LocationRestriction struct {
	Rectangle Rectangle `json:"rectangle"`
}

type Rectangle struct {
	Low  Point `json:"low"`
	High Point `json:"high"`
}

type Circle struct {
	Center Point   `json:"center"`
	Radius float64 `json:"radius"`
}

type Point struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// apiResponse defines the structure for unmarshalling the API's JSON response.
// We only care about the place IDs.
type apiResponse struct {
	Places        []*PlaceDetails `json:"places"`
	NextPageToken string          `json:"nextPageToken,omitempty"`
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
	GoogleMapsUri          *string         `json:"googleMapsUri,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// circleToRectangle converts a circle to a rectangle that fully encompasses the circle
func circleToRectangle(circle Circle) Rectangle {
	// Convert radius from meters to degrees (approximate)
	// 1 degree of latitude ≈ 111,320 meters
	// 1 degree of longitude varies by latitude, but we'll use an approximation
	latDegreeInMeters := 111320.0
	lonDegreeInMeters := 111320.0 * math.Cos(circle.Center.Latitude*math.Pi/180)

	// Calculate the latitude and longitude offsets
	latOffset := circle.Radius / latDegreeInMeters
	lonOffset := circle.Radius / lonDegreeInMeters

	// Create the rectangle that encompasses the circle
	return Rectangle{
		Low: Point{
			Latitude:  circle.Center.Latitude - latOffset,
			Longitude: circle.Center.Longitude - lonOffset,
		},
		High: Point{
			Latitude:  circle.Center.Latitude + latOffset,
			Longitude: circle.Center.Longitude + lonOffset,
		},
	}
}

// GetPlacesViaNearbySearch queries the Google Places API (Nearby Search - New) to find all places
// of specified types within a circular search area.
func GetPlacesViaNearbySearch(ctx context.Context, broker *db.Service, apiKey string, includedTypes []string, fieldMask string, targetCircle Circle, maxResults int, ipAddress string) ([]*PlaceDetails, error) {
	maxResults = 20
	reqBody := nearbyRequestBody{
		IncludedTypes: includedTypes,
		LocationRestriction: &NearbyLocationRestriction{
			Circle: targetCircle,
		},
		MaxResultCount: &maxResults,
	}

	if maxResults > 0 {
		reqBody.MaxResultCount = &maxResults
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", placesNearbyEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", apiKey)
	req.Header.Set("X-Goog-FieldMask", fieldMask)

	// Execute the request
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

	var places []*PlaceDetails
	for _, p := range apiResp.Places {
		if p.ID == "" {
			return nil, fmt.Errorf("place ID is missing for a place")
		}
		places = append(places, p)
	}

	// Log the API call
	if broker != nil {
		sku := determineSKU(placesNearbyEndpoint, fieldMask)
		logEntry := &db.MapsCallLog{
			SKU:       sku,
			IPAddress: ipAddress,
			Details:   fmt.Sprintf("Nearby search for types: %v, center: %.6f,%.6f, radius: %.0f", includedTypes, targetCircle.Center.Latitude, targetCircle.Center.Longitude, targetCircle.Radius),
		}
		if len(places) > 0 {
			logEntry.PlaceID = &places[0].ID
		}
		if err := broker.MapsCallLog.Create(logEntry); err != nil {
			// Log error but don't fail the API call
			fmt.Printf("Failed to log maps call: %v\n", err)
		}
	}

	return places, nil
}

// GetPlacesViaTextSearch queries the Google Places API (Text Search - New) to find all places
// matching a query within a specified circular search area. It now takes a 'circle' struct directly.
func GetPlacesViaTextSearch(ctx context.Context, broker *db.Service, apiKey, query, fieldMask string, targetCircle Circle, strict bool, ipAddress string) ([]*PlaceDetails, error) {
	reqBody := requestBody{
		TextQuery: query,
	}

	if strict {
		// Use rectangle restriction when strict is true
		rectangle := circleToRectangle(targetCircle)
		reqBody.LocationRestriction = &LocationRestriction{Rectangle: rectangle}
	} else {
		// Use circle bias when strict is false
		reqBody.LocationBias = &LocationBias{Circle: targetCircle}
	}

	// iterate to get all pages of results up to max 3 pages (60 results)
	var allPlaces []*PlaceDetails
	for i := 0; i < 3; i++ {

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
		// we add nextpagetoken to get more results
		fieldMask = fieldMask + ",nextPageToken"
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
			allPlaces = append(allPlaces, p)
		}

		// Log the API call
		if broker != nil {
			sku := determineSKU(placesAPIEndpoint, fieldMask)
			logEntry := &db.MapsCallLog{
				SKU:       sku,
				IPAddress: ipAddress,
				Details:   fmt.Sprintf("Text search for query: %s, center: %.6f,%.6f, radius: %.0f, page: %d", query, targetCircle.Center.Latitude, targetCircle.Center.Longitude, targetCircle.Radius, i+1),
			}
			if len(apiResp.Places) > 0 {
				logEntry.PlaceID = &apiResp.Places[0].ID
			}
			if err := broker.MapsCallLog.Create(logEntry); err != nil {
				// Log error but don't fail the API call
				fmt.Printf("Failed to log maps call: %v\n", err)
			}
		}

		// If there's no next page token, we are done
		if apiResp.NextPageToken == "" {
			break
		}

		// Set the next page token for the next iteration
		reqBody.PageToken = &apiResp.NextPageToken
	}

	return allPlaces, nil
}

// GetPlaceDetails retrieves essential place information from Google Places API given a place ID
func GetPlaceDetails(ctx context.Context, broker *db.Service, apiKey, placeID, fieldMask string, ipAddress string) (*PlaceDetails, error) {
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

	// Log the API call
	if broker != nil {
		sku := determineSKU(placeDetailsEndpoint, fieldMask)
		logEntry := &db.MapsCallLog{
			SKU:       sku,
			PlaceID:   &placeID,
			IPAddress: ipAddress,
			Details:   fmt.Sprintf("Place details for place ID: %s", placeID),
		}
		if err := broker.MapsCallLog.Create(logEntry); err != nil {
			// Log error but don't fail the API call
			fmt.Printf("Failed to log maps call: %v\n", err)
		}
	}

	return &placeDetails, nil
}
