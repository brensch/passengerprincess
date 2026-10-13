package db

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// legacySchema is the schema before non-superchargers moved to their own table and
// superchargers gained verification tracking.
var legacySchema = []string{
	"CREATE TABLE `restaurants` (`place_id` text,`name` text,`address` text,`latitude` real,`longitude` real,`rating` real,`user_ratings_total` integer,`primary_type` text,`primary_type_display` text,`display_name` text,`google_maps_uri` text,`last_updated` datetime DEFAULT CURRENT_TIMESTAMP,PRIMARY KEY (`place_id`))",
	"CREATE TABLE `superchargers` (`place_id` text,`name` text,`address` text,`latitude` real,`longitude` real,`google_maps_uri` text,`last_updated` datetime DEFAULT CURRENT_TIMESTAMP,`is_supercharger` numeric,PRIMARY KEY (`place_id`))",
	"CREATE TABLE `restaurant_supercharger_mappings` (`restaurant_id` text,`supercharger_id` text,`distance` real,PRIMARY KEY (`restaurant_id`,`supercharger_id`))",
	"CREATE INDEX `idx_supercharger_id` ON `restaurant_supercharger_mappings`(`supercharger_id`)",
	"INSERT INTO superchargers (place_id, name, latitude, longitude, last_updated, is_supercharger) VALUES ('sc', 'Tesla Supercharger', 1, 2, '2026-06-25 07:00:47', true)",
	"INSERT INTO superchargers (place_id, name, latitude, longitude, last_updated, is_supercharger) VALUES ('not-sc', 'Tesla Service Center', 1, 2, '2026-06-25 07:00:47', false)",
	"INSERT INTO restaurants (place_id, name, latitude, longitude, last_updated) VALUES ('r1', 'Diner', 1, 2, '2026-06-25 07:00:47')",
	"INSERT INTO restaurant_supercharger_mappings (restaurant_id, supercharger_id, distance) VALUES ('r1', 'sc', 42)",
}

func TestInitializeMigratesLegacySchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	for _, stmt := range legacySchema {
		if err := legacy.Exec(stmt).Error; err != nil {
			t.Fatalf("legacy schema %q: %v", stmt, err)
		}
	}
	sqlDB, _ := legacy.DB()
	sqlDB.Close()

	// Running twice checks the migration is safe to repeat on every start.
	for run := 1; run <= 2; run++ {
		if err := Initialize(&Config{DatabasePath: path, LogLevel: logger.Silent}); err != nil {
			t.Fatalf("Initialize run %d: %v", run, err)
		}
		s := GetDefaultService()

		if DB.Migrator().HasColumn(&Supercharger{}, "is_supercharger") {
			t.Errorf("run %d: is_supercharger column still exists", run)
		}
		if isNon, err := s.Supercharger.IsNonSupercharger("not-sc"); err != nil || !isNon {
			t.Errorf("run %d: IsNonSupercharger(not-sc) = %v, %v; want true", run, isNon, err)
		}
		if _, err := s.Supercharger.GetByID("not-sc"); err == nil {
			t.Errorf("run %d: non-supercharger still in superchargers", run)
		}

		sc, err := s.Supercharger.GetByID("sc")
		if err != nil {
			t.Fatalf("run %d: GetByID(sc): %v", run, err)
		}
		if !sc.LastVerified.Equal(sc.LastUpdated) || sc.LastUpdated.IsZero() {
			t.Errorf("run %d: last_verified = %v, want backfilled from last_updated %v", run, sc.LastVerified, sc.LastUpdated)
		}
		if sc.RestaurantSearchIDs != nil {
			t.Errorf("run %d: restaurant_search_ids = %v, want none", run, sc.RestaurantSearchIDs)
		}

		restaurants, err := s.Supercharger.GetRestaurantsForSupercharger("sc")
		if err != nil || len(restaurants) != 1 || restaurants[0].Distance != 42 {
			t.Errorf("run %d: restaurants = %v, %v; want r1 at 42m", run, restaurants, err)
		}

		// Legacy rows are old, so they must count as due for refresh and verification.
		if due, err := s.Supercharger.ListUpdatedBefore(sc.LastUpdated.AddDate(0, 0, 1), 10); err != nil || len(due) != 1 {
			t.Errorf("run %d: ListUpdatedBefore = %v, %v; want sc", run, due, err)
		}

		if err := Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}
