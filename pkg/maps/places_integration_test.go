package maps

import (
	"os"
	"testing"
)

// TestGetPlaceIDsViaTextSearch makes an actual call to Google Places API
// and verifies it returns valid place IDs. This test requires MAPS_API_KEY environment variable.
// Run with: MAPS_API_KEY=your_key go test -run TestGetPlaceIDsViaTextSearch ./pkg/maps
func TestGetPlaceIDsViaTextSearch(t *testing.T) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		t.Skip("MAPS_API_KEY not set, skipping integration test")
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

	// Call the real API
	placeIDs, err := GetPlaceIDsViaTextSearch(apiKey, query, targetCircle)
	if err != nil {
		t.Fatalf("GetPlaceIDsViaTextSearch failed: %v", err)
	}

	// Verify we got some results
	if len(placeIDs) == 0 {
		t.Error("Expected some place IDs, got empty slice")
	}

	// Verify each place ID looks valid (Google Place IDs start with "ChIJ")
	for i, id := range placeIDs {
		if id == "" {
			t.Errorf("Place ID at index %d is empty", i)
		}
		if len(id) < 10 {
			t.Errorf("Place ID %s seems too short to be valid", id)
		}
	}

	t.Logf("Successfully retrieved %d place IDs for query '%s'", len(placeIDs), query)
}
