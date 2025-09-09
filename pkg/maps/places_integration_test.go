package maps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
)

// TestCaptureRealAPIResponse makes an actual call to Google Places API and saves the response
// to a file for use in tests. This test requires a valid MAPS_API_KEY environment variable.
// Run with: go test -run TestCaptureRealAPIResponse ./pkg/maps
func TestCaptureRealAPIResponse(t *testing.T) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		t.Skip("MAPS_API_KEY not set, skipping real API test")
	}

	// Test parameters
	query := "pizza"
	targetCircle := circle{
		Center: center{
			Latitude:  40.7128, // New York City
			Longitude: -74.0060,
		},
		Radius: 1000.0,
	}

	// Construct the request body
	reqBody := requestBody{
		TextQuery:    query,
		LocationBias: locationBias{Circle: targetCircle},
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", placesAPIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", apiKey)
	req.Header.Set("X-Goog-FieldMask", "places.id")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Create a buffer to capture the full response
	var responseBuffer bytes.Buffer

	// Write status line
	responseBuffer.WriteString(fmt.Sprintf("HTTP/%d.%d %d %s\n",
		resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, resp.Status))

	// Write headers
	for key, values := range resp.Header {
		for _, value := range values {
			responseBuffer.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}

	// Write empty line separating headers from body
	responseBuffer.WriteString("\n")

	// Write body
	responseBuffer.Write(bodyBytes)

	// Save to file
	filename := "google_places_api_response.txt"
	err = os.WriteFile(filename, responseBuffer.Bytes(), 0644)
	if err != nil {
		t.Fatalf("Failed to write response to file: %v", err)
	}

	t.Logf("Response saved to %s", filename)
	t.Logf("Status: %s", resp.Status)
	t.Logf("Body length: %d bytes", len(bodyBytes))

	// Also try to parse the response to verify it's valid
	if resp.StatusCode == http.StatusOK {
		var apiResp apiResponse
		if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		} else {
			t.Logf("Found %d places", len(apiResp.Places))
			for i, place := range apiResp.Places {
				t.Logf("Place %d: %s", i+1, place.ID)
			}
		}
	}
}
