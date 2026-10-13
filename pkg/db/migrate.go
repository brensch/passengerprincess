package db

import (
	"fmt"

	"gorm.io/gorm"
)

// migrateData applies schema changes that gorm's AutoMigrate cannot express.
// Every step checks the current schema first, so it is safe to run on every start.
func migrateData(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := moveNonSuperchargers(tx); err != nil {
			return fmt.Errorf("move non-superchargers: %w", err)
		}
		if err := tx.Exec("UPDATE superchargers SET last_verified = last_updated WHERE last_verified IS NULL").Error; err != nil {
			return fmt.Errorf("backfill last_verified: %w", err)
		}
		return nil
	})
}

// moveNonSuperchargers moves places flagged with the is_supercharger column into
// non_superchargers, keeping only their IDs, and drops the column.
func moveNonSuperchargers(tx *gorm.DB) error {
	if !tx.Migrator().HasColumn(&Supercharger{}, "is_supercharger") {
		return nil
	}
	statements := []string{
		"INSERT OR IGNORE INTO non_superchargers (place_id) SELECT place_id FROM superchargers WHERE NOT is_supercharger",
		"DELETE FROM restaurant_supercharger_mappings WHERE supercharger_id IN (SELECT place_id FROM superchargers WHERE NOT is_supercharger)",
		"DELETE FROM superchargers WHERE NOT is_supercharger",
		"ALTER TABLE superchargers DROP COLUMN is_supercharger",
	}
	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
