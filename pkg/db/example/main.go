package main

import (
	"fmt"
	"log"
	"time"

	"github.com/brensch/passengerprincess/pkg/db"
	"gorm.io/gorm/logger"
)

func main() {
	// Initialize the database with custom configuration
	config := &db.Config{
		DatabasePath: "example.db",
		LogLevel:     logger.Info,
	}

	if err := db.Initialize(config); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Get the service instance
	service := db.GetDefaultService()

	// Example: Create a new place
	place := &db.Restaurant{
		PlaceID:            "ChIJN1t_tDeuEmsRUsoyG83frY4",
		Name:               "Example Restaurant",
		Address:            "123 Example St, Sydney NSW 2000, Australia",
		Latitude:           -33.8688,
		Longitude:          151.2093,
		Rating:             4.5,
		UserRatingsTotal:   150,
		PrimaryType:        "restaurant",
		PrimaryTypeDisplay: "Restaurant",
		DisplayName:        "Example Restaurant",
		LastUpdated:        time.Now(),
	}

	if err := service.Restaurant.Create(place); err != nil {
		log.Printf("Error creating place: %v", err)
	} else {
		fmt.Printf("Created place: %s\n", place.Name)
	}

	// Example: Create a supercharger
	supercharger := &db.Supercharger{
		PlaceID:     "ChIJexample_supercharger_id",
		Name:        "Example Supercharger",
		Address:     "456 Charging St, Sydney NSW 2000, Australia",
		Latitude:    -33.8700,
		Longitude:   151.2100,
		LastUpdated: time.Now(),
	}

	if err := service.Supercharger.Create(supercharger); err != nil {
		log.Printf("Error creating supercharger: %v", err)
	} else {
		fmt.Printf("Created supercharger: %s\n", supercharger.Name)
	}

	// Example: Associate place with supercharger
	placeAssociationOps := service.GetRestaurantAssociationOps()
	if err := placeAssociationOps.AddAssociation(place.PlaceID, supercharger.PlaceID); err != nil {
		log.Printf("Error creating association: %v", err)
	} else {
		fmt.Println("Created association between place and supercharger")
	}

	// Example: Retrieve place with superchargers
	retrievedPlace, err := service.Restaurant.GetByIDWithSuperchargers(place.PlaceID)
	if err != nil {
		log.Printf("Error retrieving place: %v", err)
	} else {
		fmt.Printf("Retrieved place: %s with %d superchargers\n",
			retrievedPlace.Name, len(retrievedPlace.Superchargers))
	}

	// Example: Search places by name
	searchResults, err := service.Restaurant.Search("Example", 10)
	if err != nil {
		log.Printf("Error searching places: %v", err)
	} else {
		fmt.Printf("Found %d places matching 'Example'\n", len(searchResults))
	}

	// Example: Log a maps API call
	mapsLog := &db.MapsCallLog{
		SKU:       "placesearch",
		Timestamp: time.Now(),
		PlaceID:   &place.PlaceID,
		Details:   "Searched for restaurants near location",
	}

	if err := service.MapsCallLog.Create(mapsLog); err != nil {
		log.Printf("Error creating maps log: %v", err)
	} else {
		fmt.Printf("Created maps call log entry with ID: %d\n", mapsLog.ID)
	}

	// Example: Record cache hit
	cacheHit := &db.CacheHit{
		ObjectID:    place.PlaceID,
		Hit:         true,
		LastUpdated: time.Now(),
		Type:        "place_details",
	}

	if err := service.CacheHit.Upsert(cacheHit); err != nil {
		log.Printf("Error recording cache hit: %v", err)
	} else {
		fmt.Println("Recorded cache hit")
	}

	// Example: Get cache hit rate
	hitRate, err := service.CacheHit.GetHitRate("place_details")
	if err != nil {
		log.Printf("Error calculating hit rate: %v", err)
	} else {
		fmt.Printf("Cache hit rate for place_details: %.2f%%\n", hitRate*100)
	}

	// Example: Transaction usage
	err = service.Transaction(func(txService *db.Service) error {
		// Create multiple records in a transaction
		place2 := &db.Restaurant{
			PlaceID:   "example_place_2",
			Name:      "Another Place",
			Address:   "789 Another St",
			Latitude:  -33.8600,
			Longitude: 151.2000,
		}

		if err := txService.Restaurant.Create(place2); err != nil {
			return err
		}

		log2 := &db.RouteCallLog{
			Origin:      "Sydney",
			Destination: "Melbourne",
			Timestamp:   time.Now(),
			IPAddress:   "127.0.0.1",
		}

		return txService.RouteCallLog.Create(log2)
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
	} else {
		fmt.Println("Transaction completed successfully")
	}

	// Example: Get counts
	restaurantCount, _ := service.Restaurant.Count()
	superchargerCount, _ := service.Supercharger.Count()
	mapsLogCount, _ := service.MapsCallLog.Count()

	fmt.Printf("\nDatabase statistics:\n")
	fmt.Printf("Restaurants: %d\n", restaurantCount)
	fmt.Printf("Superchargers: %d\n", superchargerCount)
	fmt.Printf("Maps call logs: %d\n", mapsLogCount)
}
