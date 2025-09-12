package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

func TestInitialize(t *testing.T) {
	// Create database file in test-databases directory
	timestamp := time.Now().Format("20060102_150405")
	dbFile := filepath.Join("test-databases", fmt.Sprintf("TestInitialize_%s.db", timestamp))

	// Ensure the directory exists
	os.MkdirAll("test-databases", 0755)

	err := Initialize(&Config{
		DatabasePath: dbFile,
		LogLevel:     logger.Error,
	})
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer Close()

	t.Logf("Database created at: %s", dbFile)

	// Check if tables exist
	if !DB.Migrator().HasTable(&Supercharger{}) {
		t.Error("Supercharger table not created")
	}
	if !DB.Migrator().HasTable(&Restaurant{}) {
		t.Error("Restaurant table not created")
	}
	if !DB.Migrator().HasTable("restaurant_superchargers") {
		t.Error("Join table not created")
	}

	// Test repositories
	service := GetDefaultService()

	// Test Supercharger
	sc := &Supercharger{
		PlaceID:   "test_sc",
		Name:      "Test Supercharger",
		Address:   "Test Address",
		Latitude:  37.7749,
		Longitude: -122.4194,
	}
	err = service.Supercharger.Create(sc)
	if err != nil {
		t.Fatalf("Failed to create supercharger: %v", err)
	}

	retrieved, err := service.Supercharger.GetByID("test_sc")
	if err != nil {
		t.Fatalf("Failed to get supercharger: %v", err)
	}
	if retrieved.PlaceID != "test_sc" || retrieved.Name != "Test Supercharger" {
		t.Error("Retrieved supercharger does not match")
	}

	// Test Restaurant
	rest := &Restaurant{
		PlaceID:            "test_rest",
		Name:               "Test Restaurant",
		Address:            "Test Address",
		Latitude:           37.7749,
		Longitude:          -122.4194,
		Rating:             4.5,
		UserRatingsTotal:   100,
		PrimaryType:        "restaurant",
		PrimaryTypeDisplay: "Restaurant",
		DisplayName:        "Test Restaurant",
	}
	err = service.Restaurant.Create(rest)
	if err != nil {
		t.Fatalf("Failed to create restaurant: %v", err)
	}

	retrievedRest, err := service.Restaurant.GetByID("test_rest")
	if err != nil {
		t.Fatalf("Failed to get restaurant: %v", err)
	}
	if retrievedRest.PlaceID != "test_rest" || retrievedRest.Name != "Test Restaurant" {
		t.Error("Retrieved restaurant does not match")
	}

	// Test association
	err = service.Restaurant.AssociateWithSupercharger("test_rest", "test_sc")
	if err != nil {
		t.Fatalf("Failed to associate: %v", err)
	}

	restWithSCs, err := service.Restaurant.GetByIDWithSuperchargers("test_rest")
	if err != nil {
		t.Fatalf("Failed to get restaurant with superchargers: %v", err)
	}
	if len(restWithSCs.Superchargers) != 1 || restWithSCs.Superchargers[0].PlaceID != "test_sc" {
		t.Error("Association not working correctly")
	}
}

func TestSuperchargerRepository(t *testing.T) {
	// Create database file in test-databases directory
	timestamp := time.Now().Format("20060102_150405")
	dbFile := filepath.Join("test-databases", fmt.Sprintf("TestSuperchargerRepository_%s.db", timestamp))

	// Ensure the directory exists
	os.MkdirAll("test-databases", 0755)

	err := Initialize(&Config{
		DatabasePath: dbFile,
		LogLevel:     logger.Error,
	})
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer Close()

	t.Logf("Database created at: %s", dbFile)

	service := GetDefaultService()

	// Create test data
	scs := []Supercharger{
		{PlaceID: "sc1", Name: "SC1", Address: "Addr1", Latitude: 1, Longitude: 1},
		{PlaceID: "sc2", Name: "SC2", Address: "Addr2", Latitude: 2, Longitude: 2},
	}

	err = service.Supercharger.CreateBatch(scs)
	if err != nil {
		t.Fatalf("Failed to create batch superchargers: %v", err)
	}

	// Test GetByID
	sc, err := service.Supercharger.GetByID("sc1")
	if err != nil {
		t.Fatalf("Failed to get supercharger: %v", err)
	}
	if sc.Name != "SC1" {
		t.Error("GetByID failed")
	}

	// Test GetAll
	all, err := service.Supercharger.GetAll(10, 0)
	if err != nil || len(all) != 2 {
		t.Fatalf("Failed to get all superchargers: %v", err)
	}

	// Test GetByLocation
	located, err := service.Supercharger.GetByLocation(0, 3, 0, 3)
	if err != nil || len(located) != 2 {
		t.Fatalf("Failed to get superchargers by location: %v", err)
	}
}

func TestRestaurantRepository(t *testing.T) {
	// Create database file in test-databases directory
	timestamp := time.Now().Format("20060102_150405")
	dbFile := filepath.Join("test-databases", fmt.Sprintf("TestRestaurantRepository_%s.db", timestamp))

	// Ensure the directory exists
	os.MkdirAll("test-databases", 0755)

	err := Initialize(&Config{
		DatabasePath: dbFile,
		LogLevel:     logger.Error,
	})
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer Close()

	t.Logf("Database created at: %s", dbFile)

	service := GetDefaultService()

	// Create test data
	rests := []Restaurant{
		{
			PlaceID:            "r1",
			Name:               "Rest1",
			Address:            "Addr1",
			Latitude:           1,
			Longitude:          1,
			Rating:             4.0,
			UserRatingsTotal:   50,
			PrimaryType:        "restaurant",
			PrimaryTypeDisplay: "Restaurant",
			DisplayName:        "Rest1",
		},
		{
			PlaceID:            "r2",
			Name:               "Rest2",
			Address:            "Addr2",
			Latitude:           2,
			Longitude:          2,
			Rating:             4.5,
			UserRatingsTotal:   75,
			PrimaryType:        "restaurant",
			PrimaryTypeDisplay: "Restaurant",
			DisplayName:        "Rest2",
		},
	}

	for _, r := range rests {
		err = service.Restaurant.Create(&r)
		if err != nil {
			t.Fatalf("Failed to create restaurant: %v", err)
		}
	}

	// Test GetByID
	r, err := service.Restaurant.GetByID("r1")
	if err != nil {
		t.Fatalf("Failed to get restaurant: %v", err)
	}
	if r.Name != "Rest1" {
		t.Error("GetByID failed")
	}

	// Test Search
	results, err := service.Restaurant.Search("Rest", 10)
	if err != nil || len(results) != 2 {
		t.Fatalf("Failed to search restaurants: %v", err)
	}

	// Test Count
	count, err := service.Restaurant.Count()
	if err != nil || count != 2 {
		t.Fatalf("Failed to count restaurants: %v", err)
	}
}
