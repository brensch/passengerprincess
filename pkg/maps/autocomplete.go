package maps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AutocompleteRequest represents the request body for Places API v1 autocomplete
type AutocompleteRequest struct {
	Input                string        `json:"input"`
	SessionToken         string        `json:"sessionToken,omitempty"`
	IncludedPrimaryTypes []string      `json:"includedPrimaryTypes,omitempty"`
	LocationBias         *LocationBias `json:"locationBias,omitempty"`
}

// AutocompleteResponse represents the response from Google Places API v1
type AutocompleteResponse struct {
	Suggestions []Suggestion `json:"suggestions"`
}

// Suggestion represents a single suggestion in the response
type Suggestion struct {
	PlacePrediction *PlacePrediction `json:"placePrediction,omitempty"`
	QueryPrediction *QueryPrediction `json:"queryPrediction,omitempty"`
}

// PlacePrediction represents a place prediction
type PlacePrediction struct {
	PlaceID string   `json:"placeId"`
	Text    Text     `json:"text"`
	Types   []string `json:"types,omitempty"`
}

// QueryPrediction represents a query prediction
type QueryPrediction struct {
	Text Text `json:"text"`
}

// Text represents text with matches
type Text struct {
	Text string `json:"text"`
}

// AutocompletePrediction represents a simplified prediction for our API
type AutocompletePrediction struct {
	Description string   `json:"description"`
	PlaceID     string   `json:"place_id"`
	Types       []string `json:"types"`
}

// GetAutocompleteSuggestions fetches place autocomplete suggestions from Google Places API v1
func GetAutocompleteSuggestions(ctx context.Context, apiKey, input string, sessionToken string) ([]AutocompletePrediction, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is missing")
	}

	if input == "" {
		return nil, fmt.Errorf("input is required")
	}

	// Create request body
	requestBody := AutocompleteRequest{
		Input: input,
	}

	// Add session token if provided
	if sessionToken != "" {
		requestBody.SessionToken = sessionToken
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	apiURL := "https://places.googleapis.com/v1/places:autocomplete"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", apiKey)
	req.Header.Set("X-Goog-FieldMask", "suggestions.placePrediction.placeId,suggestions.placePrediction.text,suggestions.placePrediction.types")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	// Parse response
	var autocompleteResp AutocompleteResponse
	if err := json.Unmarshal(body, &autocompleteResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our simplified format
	var predictions []AutocompletePrediction
	for _, suggestion := range autocompleteResp.Suggestions {
		if suggestion.PlacePrediction != nil {
			prediction := AutocompletePrediction{
				Description: suggestion.PlacePrediction.Text.Text,
				PlaceID:     suggestion.PlacePrediction.PlaceID,
				Types:       suggestion.PlacePrediction.Types,
			}
			predictions = append(predictions, prediction)
		}
	}

	return predictions, nil
}
