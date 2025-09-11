package maps

import (
	"context"
	"os"
	"testing"

	"github.com/brensch/passengerprincess/pkg/db"
)

func TestGetSuperchargersOnRoute_BostonToFramingham(t *testing.T) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		t.Skip("MAPS_API_KEY not set")
	}

	// Initialize in-memory database
	err := db.Initialize(&db.Config{
		DatabasePath: ":memory:",
		LogLevel:     4, // Silent
	})
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	broker := db.GetDefaultService()

	superchargers, err := GetSuperchargersOnRoute(context.Background(), broker, apiKey, "Morgan Hill, CA", "Gilroy, CA")
	if err != nil {
		t.Fatalf("GetSuperchargersOnRoute failed: %v", err)
	}

	t.Logf("Found %d superchargers on route from Morgan Hill to Gilroy:", len(superchargers))
	for i, sc := range superchargers {
		t.Logf("%d: %s at %s (Lat: %.6f, Lon: %.6f)", i+1, sc.Name, sc.Address, sc.Latitude, sc.Longitude)
	}

	t.Logf("running again to check caching...")
	superchargersCached, err := GetSuperchargersOnRoute(context.Background(), broker, apiKey, "Morgan Hill, CA", "Gilroy, CA")
	if err != nil {
		t.Fatalf("GetSuperchargersOnRoute failed: %v", err)
	}

	t.Logf("Found %d superchargers on route from Morgan Hill to Gilroy:", len(superchargersCached))
	for i, sc := range superchargersCached {
		t.Logf("%d: %s at %s (Lat: %.6f, Lon: %.6f)", i+1, sc.Name, sc.Address, sc.Latitude, sc.Longitude)
	}

	// make sure they're the same superchargers
	for _, sc := range superchargers {
		found := false
		for _, scCached := range superchargersCached {
			if sc.PlaceID == scCached.PlaceID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Supercharger %s not found in cached results", sc.PlaceID)
		}
	}

	if len(superchargers) == 0 {
		t.Error("Expected to find at least one supercharger on the route")
	}
}
