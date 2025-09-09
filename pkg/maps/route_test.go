package maps

import (
	"os"
	"testing"
)

func TestGetRoute(t *testing.T) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		t.Skip("MAPS_API_KEY environment variable not set")
	}

	origin := "New York, NY"
	destination := "Boston, MA"

	result, err := GetRoute(apiKey, origin, destination)
	if err != nil {
		t.Fatalf("GetRoute failed: %v", err)
	}

	if result == nil {
		t.Fatal("GetRoute returned nil result")
	}

	if result.DistanceMeters <= 0 {
		t.Errorf("Expected positive distance, got %d", result.DistanceMeters)
	}

	if result.Duration <= 0 {
		t.Errorf("Expected positive duration, got %v", result.Duration)
	}

	if result.EncodedPolyline == "" {
		t.Error("Expected non-empty encoded polyline")
	}

}
