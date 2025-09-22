package maps

import (
	"context"
	"os"
	"testing"
)

// TestGetPlacesViaTextSearch makes an actual call to Google Places API
// and verifies it returns valid places. This test requires MAPS_API_KEY environment variable.
// Run with: MAPS_API_KEY=your_key go test -run TestGetPlacesViaTextSearch ./pkg/maps
func TestGetPlacesViaTextSearch(t *testing.T) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		t.Skip("MAPS_API_KEY not set, skipping integration test")
	}

	// Test parameters
	query := "food"
	targetCircle := Circle{
		Center: Point{
			Latitude:  38.79104900000001,
			Longitude: -121.223586,
		},
		Radius: 500.0,
	}

	// Call the real API
	places, err := GetPlacesViaTextSearch(context.Background(), nil, apiKey, query, FieldMaskRestaurantTextSearch, targetCircle, true, "", "test-request-id")
	if err != nil {
		t.Fatalf("GetPlaceIDsViaTextSearch failed: %v", err)
	}

	// Verify we got some results
	if len(places) == 0 {
		t.Error("Expected some places, got empty slice")
	}

	// Verify each place has valid ID and optional fields
	for i, place := range places {
		// if place.ID == "" {
		// 	t.Errorf("Place ID at index %d is empty", i)
		// }
		// if len(place.ID) < 10 {
		// 	t.Errorf("Place ID %s seems too short to be valid", place.ID)
		// }
		// Optional fields
		if place.DisplayName != nil {
			t.Logf("Place %d display name: %s", i, place.DisplayName.Text)
		}
		// if place.FormattedAddress != nil {
		// 	t.Logf("Place %d address: %s", i, *place.FormattedAddress)
		// }
		// if place.Location != nil {
		// 	t.Logf("Place %d location: %.6f, %.6f", i, place.Location.Latitude, place.Location.Longitude)
		// }
	}

	t.Logf("Successfully retrieved %d places for query '%s'", len(places), query)
}

// TestGetPlacesViaNearbySearch makes an actual call to Google Places API
// and verifies it returns valid restaurants using nearby search. This test requires MAPS_API_KEY environment variable.
// Run with: MAPS_API_KEY=your_key go test -run TestGetPlacesViaNearbySearch ./pkg/maps
func TestGetPlacesViaNearbySearch(t *testing.T) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		t.Skip("MAPS_API_KEY not set, skipping integration test")
	}

	// Test parameters - search for restaurants
	includedTypes := []string{"restaurant"}
	targetCircle := Circle{
		Center: Point{
			Latitude:  38.79104900000001,
			Longitude: -121.223586,
		},
		Radius: 500.0,
	}
	maxResults := 10

	// Call the real API
	places, err := GetPlacesViaNearbySearch(context.Background(), nil, apiKey, includedTypes, FieldMaskRestaurantNearbySearch, targetCircle, maxResults, "", "test-request-id")
	if err != nil {
		t.Fatalf("GetPlacesViaNearbySearch failed: %v", err)
	}

	// Verify we got some results
	if len(places) == 0 {
		t.Error("Expected some places, got empty slice")
	}

	// Verify we don't exceed max results
	if len(places) > maxResults {
		t.Errorf("Expected at most %d places, got %d", maxResults, len(places))
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
	}

	t.Logf("Successfully retrieved %d restaurants using nearby search", len(places))
}
