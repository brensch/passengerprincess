package maps

import (
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

// TestGetPlaceDetails makes an actual call to Google Places API
// and verifies it returns valid place details. This test requires MAPS_API_KEY environment variable.
// Run with: MAPS_API_KEY=your_key go test -run TestGetPlaceDetails ./pkg/maps
func TestGetPlaceDetails(t *testing.T) {
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
	placeDetails, err := GetPlaceDetailsWithCache(broker, apiKey, placeID)
	if err != nil {
		t.Fatalf("GetPlaceDetailsWithCache failed: %v", err)
	}

	// Verify we got a valid response
	if placeDetails == nil {
		t.Fatal("GetPlaceDetailsWithCache returned nil")
	}

	// Verify essential fields are present
	if placeDetails.ID == "" {
		t.Error("Expected non-empty place ID")
	}

	if placeDetails.Name == "" {
		t.Error("Expected non-empty place name")
	}

	if placeDetails.FormattedAddress == "" {
		t.Error("Expected non-empty formatted address")
	}

	// Verify location coordinates are reasonable
	if placeDetails.Location.Latitude < -90 || placeDetails.Location.Latitude > 90 {
		t.Errorf("Latitude %f is out of valid range [-90, 90]", placeDetails.Location.Latitude)
	}

	if placeDetails.Location.Longitude < -180 || placeDetails.Location.Longitude > 180 {
		t.Errorf("Longitude %f is out of valid range [-180, 180]", placeDetails.Location.Longitude)
	}

	// Verify place ID matches what we requested
	if placeDetails.ID != placeID {
		t.Errorf("Expected place ID %s, got %s", placeID, placeDetails.ID)
	}

	t.Logf("Successfully retrieved details for place: %s", placeDetails.Name)
	t.Logf("Address: %s", placeDetails.FormattedAddress)
	t.Logf("Location: %.6f, %.6f", placeDetails.Location.Latitude, placeDetails.Location.Longitude)

	// Test caching: Call again, should get from database this time
	t.Logf("Testing cache - calling again for same place ID...")
	placeDetails2, err := GetPlaceDetailsWithCache(broker, apiKey, placeID)
	if err != nil {
		t.Fatalf("Second call to GetPlaceDetailsWithCache failed: %v", err)
	}

	// Verify the cached result matches the original
	if placeDetails.ID != placeDetails2.ID ||
		placeDetails.Name != placeDetails2.Name ||
		placeDetails.FormattedAddress != placeDetails2.FormattedAddress ||
		placeDetails.Location.Latitude != placeDetails2.Location.Latitude ||
		placeDetails.Location.Longitude != placeDetails2.Location.Longitude {
		t.Error("Cached result doesn't match original API result")
	}

	t.Logf("Cache test passed - data retrieved from database on second call")
}
