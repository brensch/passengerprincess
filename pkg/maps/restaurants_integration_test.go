package maps

import (
	"context"
	"os"
	"testing"

	"github.com/brensch/passengerprincess/pkg/db"
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
	targetCircle := Circle{
		Center: Center{
			Latitude:  40.7128, // New York City
			Longitude: -74.0060,
		},
		Radius: 1000.0,
	}

	// Call the real API
	places, err := GetPlacesViaTextSearch(context.Background(), apiKey, query, "places.id", targetCircle)
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
	places, err = GetPlacesViaTextSearch(context.Background(), apiKey, query, FieldMaskRestaurantTextSearch, targetCircle)
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
			t.Logf("Place %d display name: %s", i, *place.DisplayName)
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
			t.Logf("Place %d primary type display name: %s", i, *place.PrimaryTypeDisplayName)
		}
	}

	t.Logf("Successfully retrieved %d places for query '%s'", len(places), query)
}

// TestGetSuperchargerWithCacheRestaurants makes an actual call to Google Places API
// and verifies it returns valid supercharger details. This test requires MAPS_API_KEY environment variable.
// Run with: MAPS_API_KEY=your_key go test -run TestGetSuperchargerWithCacheRestaurants ./pkg/maps
func TestGetSuperchargerWithCacheRestaurants(t *testing.T) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		t.Skip("MAPS_API_KEY not set, skipping integration test")
	}

	// Initialize in-memory database for testing
	err := db.Initialize(&db.Config{
		DatabasePath: ":memory:",
		LogLevel:     4, // Silent
	})
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	broker := db.GetDefaultService()

	// Test with a known place ID (Googleplex)
	placeID := "ChIJj61dQgK6j4AR4GeTYWZsKWw"

	// Call the cached version (will fetch from API and cache in DB)
	supercharger, err := GetSuperchargerWithCache(context.Background(), broker, apiKey, placeID)
	if err != nil {
		t.Fatalf("GetSuperchargerWithCache failed: %v", err)
	}

	// Verify we got a valid response
	if supercharger == nil {
		t.Fatal("GetSuperchargerWithCache returned nil")
	}

	// Verify essential fields are present
	if supercharger.PlaceID == "" {
		t.Error("Expected non-empty place ID")
	}

	if supercharger.Name == "" {
		t.Error("Expected non-empty place name")
	}

	if supercharger.Address == "" {
		t.Error("Expected non-empty formatted address")
	}

	// Verify location coordinates are reasonable
	if supercharger.Latitude < -90 || supercharger.Latitude > 90 {
		t.Errorf("Latitude %f is out of valid range [-90, 90]", supercharger.Latitude)
	}

	if supercharger.Longitude < -180 || supercharger.Longitude > 180 {
		t.Errorf("Longitude %f is out of valid range [-180, 180]", supercharger.Longitude)
	}

	// Verify place ID matches what we requested
	if supercharger.PlaceID != placeID {
		t.Errorf("Expected place ID %s, got %s", placeID, supercharger.PlaceID)
	}

	t.Logf("Successfully retrieved details for place: %s", supercharger.Name)
	t.Logf("Address: %s", supercharger.Address)
	t.Logf("Location: %.6f, %.6f", supercharger.Latitude, supercharger.Longitude)

	// Test caching: Call again, should get from database this time
	t.Logf("Testing cache - calling again for same place ID...")
	supercharger2, err := GetSuperchargerWithCache(context.Background(), broker, apiKey, placeID)
	if err != nil {
		t.Fatalf("Second call to GetSuperchargerWithCache failed: %v", err)
	}

	// Verify the cached result matches the original
	if supercharger.PlaceID != supercharger2.PlaceID ||
		supercharger.Name != supercharger2.Name ||
		supercharger.Address != supercharger2.Address ||
		supercharger.Latitude != supercharger2.Latitude ||
		supercharger.Longitude != supercharger2.Longitude {
		t.Error("Cached result doesn't match original API result")
	}

	t.Logf("Cache test passed - data retrieved from database on second call")
}
