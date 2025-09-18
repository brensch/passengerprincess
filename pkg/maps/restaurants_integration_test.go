package maps

import (
	"context"
	"os"
	"testing"
)

// TestGetPlaceDetailsViaTextSearch makes an actual call to Google Places API
// and verifies it returns valid place IDs. This test requires MAPS_API_KEY environment variable.
// Run with: MAPS_API_KEY=your_key go test -run TestGetPlaceDetailsViaTextSearch ./pkg/maps
func TestGetPlaceDetailsViaTextSearch(t *testing.T) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		t.Skip("MAPS_API_KEY not set, skipping integration test")
	}

	// Test parameters
	query := "pizza"
	targetCircle := Circle{
		Center: Point{
			Latitude:  40.7128, // New York City
			Longitude: -74.0060,
		},
		Radius: 1000.0,
	}

	// Call the real API
	places, err := GetPlacesViaTextSearch(context.Background(), apiKey, query, "places.id", targetCircle, true)
	if err != nil {
		t.Fatalf("GetPlaceIDsViaTextSearch failed: %v", err)
	}

	// Verify we got some results
	if len(places) == 0 {
		t.Error("Expected some places, got empty slice")
	}

	// Verify each place ID looks valid (Google Place IDs start with "ChIJ")
	for i, place := range places {
		if place.ID == "" {
			t.Errorf("Place ID at index %d is empty", i)
		}
		if len(place.ID) < 10 {
			t.Errorf("Place ID %s seems too short to be valid", place.ID)
		}
	}

	// do 1 pro request to make sure all fields are populated
	places, err = GetPlacesViaTextSearch(context.Background(), apiKey, query, FieldMaskRestaurantTextSearch, targetCircle, false)
	if err != nil {
		t.Fatalf("GetPlaceIDsViaTextSearch failed: %v", err)
	}

	// Verify we got some results
	if len(places) == 0 {
		t.Error("Expected some places, got empty slice")
	}

	// Verify each place has valid ID and optional fields
	for i, place := range places {
		if place.ID == "" {
			t.Errorf("Place ID at index %d is empty", i)
		}
		if len(place.ID) < 10 {
			t.Errorf("Place ID %s seems too short to be valid", place.ID)
		}
		// Optional fields
		if place.DisplayName != nil {
			t.Logf("Place %d display name: %s", i, place.DisplayName.Text)
		}
		if place.FormattedAddress != nil {
			t.Logf("Place %d address: %s", i, *place.FormattedAddress)
		}
		if place.Location != nil {
			t.Logf("Place %d location: %.6f, %.6f", i, place.Location.Latitude, place.Location.Longitude)
		}
		if place.PrimaryType != nil {
			t.Logf("Place %d primary type: %s", i, *place.PrimaryType)
		}
		if place.PrimaryTypeDisplayName != nil {
			t.Logf("Place %d primary type display name: %s", i, place.PrimaryTypeDisplayName.Text)
		}
	}

	t.Logf("Successfully retrieved %d places for query '%s'", len(places), query)
}
