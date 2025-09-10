package db

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// US bounding box coordinates (approximate)
const (
	// Continental US bounds
	minLatUS = 24.396308 // Southern tip of Florida
	maxLatUS = 49.384358 // Northern border with Canada
	minLonUS = -125.0    // West coast
	maxLonUS = -66.93457 // East coast
)

// BenchmarkWriteRandomData benchmarks writing 10k superchargers and 100k places
func BenchmarkWriteRandomData(b *testing.B) {
	// Create in-memory SQLite database for benchmarking
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Disable logging for performance
	})
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&Place{}, &Supercharger{})
	if err != nil {
		b.Fatalf("Failed to migrate database: %v", err)
	}

	// Create service
	service := NewService(db)

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Clear database between runs
		db.Exec("DELETE FROM place_superchargers")
		db.Exec("DELETE FROM places")
		db.Exec("DELETE FROM superchargers")

		// Generate and write data
		err := generateAndWriteRandomData(service)
		if err != nil {
			b.Fatalf("Failed to generate and write data: %v", err)
		}
	}
}

// generateAndWriteRandomData generates 10k superchargers and 100k places
func generateAndWriteRandomData(service *Service) error {
	const numSuperchargers = 10000
	var totalPlaces int

	// Generate superchargers and their associated places
	for i := 0; i < numSuperchargers; i++ {
		// Generate random supercharger in US
		supercharger := generateRandomSupercharger()

		// Create supercharger
		err := service.Supercharger.Create(&supercharger)
		if err != nil {
			return fmt.Errorf("failed to create supercharger %d: %w", i, err)
		}

		// Generate random number of places (1-20) near this supercharger
		numPlaces := rand.Intn(20) + 1 // 1 to 20 places
		totalPlaces += numPlaces

		places := make([]Place, numPlaces)
		for j := 0; j < numPlaces; j++ {
			places[j] = generateRandomPlaceNear(supercharger.Latitude, supercharger.Longitude)
		}

		// Create places in batch
		for _, place := range places {
			err := service.Place.Create(&place)
			if err != nil {
				return fmt.Errorf("failed to create place for supercharger %d: %w", i, err)
			}

			// Create association between place and supercharger
			assocOps := NewPlaceAssociationOperations(service.db)
			err = assocOps.AddAssociation(place.PlaceID, supercharger.PlaceID)
			if err != nil {
				return fmt.Errorf("failed to create association: %w", err)
			}
		}
	}

	// Verify we're close to 100k places (should be between 10k and 200k)
	if totalPlaces < 10000 || totalPlaces > 200000 {
		return fmt.Errorf("unexpected number of places generated: %d (expected ~100k)", totalPlaces)
	}

	return nil
}

// generateRandomSupercharger creates a supercharger with random coordinates in the US
func generateRandomSupercharger() Supercharger {
	lat := minLatUS + rand.Float64()*(maxLatUS-minLatUS)
	lon := minLonUS + rand.Float64()*(maxLonUS-minLonUS)

	return Supercharger{
		PlaceID:     generateFastID(),
		Name:        fmt.Sprintf("Supercharger_%s", generateFastID()[:8]),
		Address:     fmt.Sprintf("Address_%s", generateFastID()[:8]),
		Latitude:    lat,
		Longitude:   lon,
		LastUpdated: time.Now(),
	}
}

// generateRandomPlaceNear creates a place within 500m of the given coordinates
func generateRandomPlaceNear(centerLat, centerLon float64) Place {
	// Generate random point within 500m radius
	// 500m is approximately 0.0045 degrees latitude
	// Longitude varies by latitude, but we'll use an approximation
	radiusInDegrees := 0.0045

	// Generate random angle and distance
	angle := rand.Float64() * 2 * math.Pi
	distance := rand.Float64() * radiusInDegrees

	// Calculate new coordinates
	lat := centerLat + distance*math.Cos(angle)
	lon := centerLon + distance*math.Sin(angle)/math.Cos(centerLat*math.Pi/180)

	// Generate random place data
	placeTypes := []string{"restaurant", "gas_station", "lodging", "tourist_attraction", "shopping_mall", "convenience_store"}
	primaryType := placeTypes[rand.Intn(len(placeTypes))]

	return Place{
		PlaceID:            generateFastID(),
		Name:               fmt.Sprintf("Place_%s", generateFastID()[:8]),
		Address:            fmt.Sprintf("Address_%s", generateFastID()[:8]),
		Latitude:           lat,
		Longitude:          lon,
		Rating:             1.0 + rand.Float64()*4.0, // 1.0 to 5.0
		UserRatingsTotal:   rand.Intn(1000),
		PrimaryType:        primaryType,
		PrimaryTypeDisplay: primaryType,
		DisplayName:        fmt.Sprintf("Display_%s", generateFastID()[:8]),
		LastUpdated:        time.Now(),
	}
}

// generateFastID generates a fast pseudo-UUID using current time and random numbers
// This is much faster than proper UUID generation for benchmarking purposes
func generateFastID() string {
	// Use current nanosecond time and random number for uniqueness
	now := time.Now().UnixNano()
	random := rand.Int63()
	return fmt.Sprintf("%016x-%016x", now, random)
}

// BenchmarkWriteRandomDataBatched benchmarks the same data but with batched inserts
func BenchmarkWriteRandomDataBatched(b *testing.B) {
	// Create in-memory SQLite database for benchmarking
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Disable logging for performance
	})
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&Place{}, &Supercharger{})
	if err != nil {
		b.Fatalf("Failed to migrate database: %v", err)
	}

	// Create service
	service := NewService(db)

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Clear database between runs
		db.Exec("DELETE FROM place_superchargers")
		db.Exec("DELETE FROM places")
		db.Exec("DELETE FROM superchargers")

		// Generate and write data with batching
		err := generateAndWriteRandomDataBatched(service)
		if err != nil {
			b.Fatalf("Failed to generate and write data: %v", err)
		}
	}
}

// generateAndWriteRandomDataBatched generates data with batched inserts for better performance
func generateAndWriteRandomDataBatched(service *Service) error {
	const numSuperchargers = 10000
	const batchSize = 1000

	// Generate all superchargers first
	superchargers := make([]Supercharger, numSuperchargers)
	for i := 0; i < numSuperchargers; i++ {
		superchargers[i] = generateRandomSupercharger()
	}

	// Insert superchargers in batches
	for i := 0; i < len(superchargers); i += batchSize {
		end := i + batchSize
		if end > len(superchargers) {
			end = len(superchargers)
		}

		err := service.db.CreateInBatches(superchargers[i:end], batchSize).Error
		if err != nil {
			return fmt.Errorf("failed to create supercharger batch: %w", err)
		}
	}

	// Generate places and associations
	var allPlaces []Place
	var associations []struct {
		PlaceID        string
		SuperchargerID string
	}

	for _, supercharger := range superchargers {
		// Generate random number of places (1-20) near this supercharger
		numPlaces := rand.Intn(20) + 1 // 1 to 20 places

		for j := 0; j < numPlaces; j++ {
			place := generateRandomPlaceNear(supercharger.Latitude, supercharger.Longitude)
			allPlaces = append(allPlaces, place)
			associations = append(associations, struct {
				PlaceID        string
				SuperchargerID string
			}{
				PlaceID:        place.PlaceID,
				SuperchargerID: supercharger.PlaceID,
			})
		}
	}

	// Insert places in batches
	for i := 0; i < len(allPlaces); i += batchSize {
		end := i + batchSize
		if end > len(allPlaces) {
			end = len(allPlaces)
		}

		err := service.db.CreateInBatches(allPlaces[i:end], batchSize).Error
		if err != nil {
			return fmt.Errorf("failed to create place batch: %w", err)
		}
	}

	// Create associations in batches
	for i := 0; i < len(associations); i += batchSize {
		end := i + batchSize
		if end > len(associations) {
			end = len(associations)
		}

		// Use transaction for batch associations
		err := service.db.Transaction(func(tx *gorm.DB) error {
			txAssocOps := NewPlaceAssociationOperations(tx)
			for j := i; j < end; j++ {
				err := txAssocOps.AddAssociation(associations[j].PlaceID, associations[j].SuperchargerID)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to create association batch: %w", err)
		}
	}

	return nil
}
