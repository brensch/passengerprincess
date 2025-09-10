package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/brensch/passengerprincess/pkg/db"
	"github.com/google/uuid"
	"gorm.io/gorm/logger"
)

const (
	// US bounding box (approximate)
	USMinLat = 24.396308 // Florida Keys
	USMaxLat = 49.384358 // Northern border
	USMinLng = -125.0    // West coast
	USMaxLng = -66.93457 // East coast

	// Earth radius in meters
	EarthRadiusM = 6371000

	NumSuperchargers = 10000
	NumPlaces        = 100000
	MaxPlacesPerSC   = 20
	MinPlacesPerSC   = 1
	PlaceRadiusM     = 500 // Places within 500m of supercharger
)

func main() {
	fmt.Println("🚗 PassengerPrincess Data Generator")
	fmt.Printf("Generating %d superchargers and %d places...\n", NumSuperchargers, NumPlaces)

	// Initialize database
	config := &db.Config{
		DatabasePath: "passengerprincess.db",
		LogLevel:     logger.Info,
	}

	if err := db.Initialize(config); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Get database service
	service := db.GetDefaultService()

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	start := time.Now()

	// Generate superchargers
	fmt.Println("📍 Generating superchargers...")
	superchargers := generateSuperchargers()

	// Create superchargers in database
	fmt.Println("💾 Writing superchargers to database...")
	if err := createSuperchargers(service, superchargers); err != nil {
		log.Fatalf("Failed to create superchargers: %v", err)
	}

	// Generate and create places
	fmt.Println("🏪 Generating and writing places...")
	if err := generateAndCreatePlaces(service, superchargers); err != nil {
		log.Fatalf("Failed to create places: %v", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("✅ Complete! Generated %d superchargers and %d places in %v\n",
		NumSuperchargers, NumPlaces, elapsed)

	// Print some stats
	printStats(service)
}

func generateSuperchargers() []db.Supercharger {
	superchargers := make([]db.Supercharger, NumSuperchargers)

	for i := 0; i < NumSuperchargers; i++ {
		lat := randomFloat(USMinLat, USMaxLat)
		lng := randomFloat(USMinLng, USMaxLng)

		superchargers[i] = db.Supercharger{
			PlaceID:   generateID(),
			Name:      fmt.Sprintf("Supercharger-%s", generateShortID()),
			Address:   fmt.Sprintf("Address %d, US", i+1),
			Latitude:  lat,
			Longitude: lng,
		}

		if (i+1)%1000 == 0 {
			fmt.Printf("  Generated %d/%d superchargers\n", i+1, NumSuperchargers)
		}
	}

	return superchargers
}

func createSuperchargers(service *db.Service, superchargers []db.Supercharger) error {
	batchSize := 1000
	for i := 0; i < len(superchargers); i += batchSize {
		end := i + batchSize
		if end > len(superchargers) {
			end = len(superchargers)
		}

		batch := superchargers[i:end]
		if err := service.Supercharger.CreateBatch(batch); err != nil {
			return fmt.Errorf("failed to create batch %d-%d: %v", i, end-1, err)
		}

		fmt.Printf("  Wrote %d/%d superchargers\n", end, len(superchargers))
	}

	return nil
}

func generateAndCreatePlaces(service *db.Service, superchargers []db.Supercharger) error {
	placesPerSC := calculatePlacesPerSupercharger()
	totalGenerated := 0
	batchSize := 500
	var placeBatch []db.Place
	var associationPairs []struct {
		PlaceID        string
		SuperchargerID string
	}

	for i, sc := range superchargers {
		numPlaces := placesPerSC[i]

		// Generate places for this supercharger
		for j := 0; j < numPlaces; j++ {
			lat, lng := randomPointWithinRadius(sc.Latitude, sc.Longitude, PlaceRadiusM)

			placeID := generateID()
			place := db.Place{
				PlaceID:          placeID,
				Name:             fmt.Sprintf("Place-%s", generateShortID()),
				Address:          fmt.Sprintf("Place Address %d-%d", i+1, j+1),
				Latitude:         lat,
				Longitude:        lng,
				PrimaryType:      randomPlaceType(),
				Rating:           randomFloat(3.0, 5.0),
				UserRatingsTotal: rand.Intn(100) + 1,
				DisplayName:      fmt.Sprintf("Display Place-%s", generateShortID()),
			}

			placeBatch = append(placeBatch, place)
			associationPairs = append(associationPairs, struct {
				PlaceID        string
				SuperchargerID string
			}{placeID, sc.PlaceID})

			totalGenerated++

			// Process batch when it's full
			if len(placeBatch) >= batchSize {
				if err := processBatch(service, placeBatch, associationPairs); err != nil {
					return fmt.Errorf("failed to process batch: %v", err)
				}
				placeBatch = placeBatch[:0]
				associationPairs = associationPairs[:0]
			}
		}

		if (i+1)%500 == 0 {
			fmt.Printf("  Generated places for %d/%d superchargers (%d total places)\n",
				i+1, len(superchargers), totalGenerated)
		}
	}

	// Process remaining places
	if len(placeBatch) > 0 {
		if err := processBatch(service, placeBatch, associationPairs); err != nil {
			return fmt.Errorf("failed to process final batch: %v", err)
		}
	}

	fmt.Printf("  Generated %d total places\n", totalGenerated)
	return nil
}

func processBatch(service *db.Service, places []db.Place, associations []struct {
	PlaceID        string
	SuperchargerID string
}) error {
	// Create all places in batch
	if err := service.Place.CreateBatch(places); err != nil {
		return fmt.Errorf("failed to create place batch: %v", err)
	}

	// Create associations efficiently using batch method
	if err := service.Place.BatchAssociateWithSuperchargers(associations); err != nil {
		return fmt.Errorf("failed to batch associate places with superchargers: %v", err)
	}

	return nil
}

func calculatePlacesPerSupercharger() []int {
	remaining := NumPlaces
	placesPerSC := make([]int, NumSuperchargers)

	// First pass: give each supercharger the minimum
	for i := 0; i < NumSuperchargers; i++ {
		placesPerSC[i] = MinPlacesPerSC
		remaining -= MinPlacesPerSC
	}

	// Second pass: randomly distribute remaining places
	for remaining > 0 {
		idx := rand.Intn(NumSuperchargers)
		if placesPerSC[idx] < MaxPlacesPerSC {
			placesPerSC[idx]++
			remaining--
		}
	}

	return placesPerSC
}

func randomPointWithinRadius(centerLat, centerLng float64, radiusM int) (float64, float64) {
	// Convert radius to degrees (approximate)
	radiusDeg := float64(radiusM) / (EarthRadiusM * math.Pi / 180)

	// Random angle and distance
	angle := rand.Float64() * 2 * math.Pi
	distance := rand.Float64() * radiusDeg

	// Calculate new point
	lat := centerLat + distance*math.Cos(angle)
	lng := centerLng + distance*math.Sin(angle)/math.Cos(centerLat*math.Pi/180)

	return lat, lng
}

func randomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func generateID() string {
	return uuid.New().String()
}

func generateShortID() string {
	return uuid.New().String()[:8]
}

func randomPlaceType() string {
	types := []string{
		"restaurant", "gas_station", "convenience_store", "shopping_mall",
		"coffee_shop", "hotel", "hospital", "pharmacy", "bank", "atm",
		"grocery_store", "park", "museum", "library", "gym", "movie_theater",
	}
	return types[rand.Intn(len(types))]
}

func printStats(service *db.Service) {
	fmt.Println("\n📊 Database Statistics:")

	// Count superchargers
	superchargers, err := service.Supercharger.GetAll(100000, 0) // Large limit to get all
	if err != nil {
		fmt.Printf("Error counting superchargers: %v\n", err)
	} else {
		fmt.Printf("Total superchargers: %d\n", len(superchargers))
	}

	// Count places
	places, err := service.Place.GetAll(200000, 0) // Large limit to get all
	if err != nil {
		fmt.Printf("Error counting places: %v\n", err)
	} else {
		fmt.Printf("Total places: %d\n", len(places))
	}
}
