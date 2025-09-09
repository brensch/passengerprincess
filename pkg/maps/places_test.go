package maps

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestGetPlaceIDsViaTextSearch_WithRealResponse(t *testing.T) {
	// Read the real API response from file
	responseFile := "google_places_api_response.txt"
	data, err := os.ReadFile(responseFile)
	if err != nil {
		t.Skipf("Could not read response file %s: %v", responseFile, err)
	}

	// Parse the response file
	responseStr := string(data)
	lines := strings.Split(responseStr, "\n")

	// Find the empty line that separates headers from body
	bodyStart := -1
	for i, line := range lines {
		if line == "" {
			bodyStart = i + 1
			break
		}
	}

	if bodyStart == -1 || bodyStart >= len(lines) {
		t.Fatal("Could not find body in response file")
	}

	// Extract status line and headers
	statusLine := lines[0]
	headers := make(http.Header)
	for i := 1; i < bodyStart-1; i++ {
		if lines[i] == "" {
			continue
		}
		parts := strings.SplitN(lines[i], ": ", 2)
		if len(parts) == 2 {
			headers.Add(parts[0], parts[1])
		}
	}

	// Extract body
	body := strings.Join(lines[bodyStart:], "\n")

	// Parse status code from status line
	statusParts := strings.Fields(statusLine)
	if len(statusParts) < 2 {
		t.Fatal("Invalid status line")
	}
	statusCode := 200 // default
	if statusParts[1] != "200" {
		// Try to parse as int
		if code, err := strconv.Atoi(statusParts[1]); err == nil {
			statusCode = code
		}
	}

	// Create a test server that returns the real response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Copy all headers from the real response
		for key, values := range headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Set status code
		w.WriteHeader(statusCode)

		// Write body
		w.Write([]byte(body))
	}))
	defer server.Close()

	// Temporarily replace the package-level variables
	originalEndpoint := placesAPIEndpoint
	originalClient := httpClient
	placesAPIEndpoint = server.URL + "/v1/places:searchText"
	httpClient = server.Client()
	defer func() {
		placesAPIEndpoint = originalEndpoint
		httpClient = originalClient
	}()

	// Test the function with the same parameters used to capture the real response
	targetCircle := circle{
		Center: center{
			Latitude:  40.7128, // New York City
			Longitude: -74.0060,
		},
		Radius: 1000.0,
	}
	placeIDs, err := GetPlaceIDsViaTextSearch("test-api-key", "pizza", targetCircle)
	if err != nil {
		t.Fatalf("GetPlaceIDsViaTextSearch failed: %v", err)
	}

	// Verify we got some place IDs
	if len(placeIDs) == 0 {
		t.Error("Expected some place IDs, got empty slice")
	}

	// Verify the structure of place IDs (should be strings)
	for i, id := range placeIDs {
		if id == "" {
			t.Errorf("Place ID at index %d is empty", i)
		}
		if !strings.HasPrefix(id, "ChIJ") {
			t.Errorf("Place ID %s doesn't look like a valid Google Place ID", id)
		}
	}

	t.Logf("Successfully parsed %d place IDs from real API response", len(placeIDs))
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || contains(s[1:], substr) || (len(s) > 0 && s[:len(substr)] == substr))
}
