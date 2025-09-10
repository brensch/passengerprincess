package db

import (
	"gorm.io/gorm"
)

// Service provides a unified interface to all database operations
type Service struct {
	Place        *PlaceRepository
	Supercharger *SuperchargerRepository
	MapsCallLog  *MapsCallLogRepository
	CacheHit     *CacheHitRepository
	RouteCallLog *RouteCallLogRepository
	db           *gorm.DB
}

// NewService creates a new database service with all repositories
func NewService(db *gorm.DB) *Service {
	return &Service{
		Place:        NewPlaceRepository(db),
		Supercharger: NewSuperchargerRepository(db),
		MapsCallLog:  NewMapsCallLogRepository(db),
		CacheHit:     NewCacheHitRepository(db),
		RouteCallLog: NewRouteCallLogRepository(db),
		db:           db,
	}
}

// GetDefaultService returns a service using the global DB instance
func GetDefaultService() *Service {
	if DB == nil {
		panic("database not initialized - call Initialize() first")
	}
	return NewService(DB)
}

// Transaction executes a function within a database transaction
func (s *Service) Transaction(fn func(*Service) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txService := NewService(tx)
		return fn(txService)
	})
}

// PlaceAssociationOperations provides operations for place-supercharger associations using GORM's many-to-many
type PlaceAssociationOperations struct {
	db *gorm.DB
}

// NewPlaceAssociationOperations creates operations for place-supercharger relationships
func NewPlaceAssociationOperations(db *gorm.DB) *PlaceAssociationOperations {
	return &PlaceAssociationOperations{db: db}
}

// AddAssociation creates an association between a place and supercharger
func (ops *PlaceAssociationOperations) AddAssociation(placeID, superchargerID string) error {
	place := Place{PlaceID: placeID}
	supercharger := Supercharger{PlaceID: superchargerID}
	return ops.db.Model(&place).Association("Superchargers").Append(&supercharger)
}

// RemoveAssociation removes an association between a place and supercharger
func (ops *PlaceAssociationOperations) RemoveAssociation(placeID, superchargerID string) error {
	place := Place{PlaceID: placeID}
	supercharger := Supercharger{PlaceID: superchargerID}
	return ops.db.Model(&place).Association("Superchargers").Delete(&supercharger)
}

// AssociationExists checks if an association exists between a place and supercharger
func (ops *PlaceAssociationOperations) AssociationExists(placeID, superchargerID string) (bool, error) {
	var place Place
	if err := ops.db.Where("place_id = ?", placeID).First(&place).Error; err != nil {
		return false, err
	}

	var supercharger Supercharger
	if err := ops.db.Where("place_id = ?", superchargerID).First(&supercharger).Error; err != nil {
		return false, err
	}

	count := ops.db.Model(&place).Association("Superchargers").Count()
	if count == 0 {
		return false, nil
	}

	// Check if this specific supercharger is associated
	var result Supercharger
	err := ops.db.Model(&place).Association("Superchargers").Find(&result, "place_id = ?", superchargerID)
	return err == nil && result.PlaceID == superchargerID, nil
}

// GetSuperchargersForPlace retrieves all superchargers associated with a place
func (ops *PlaceAssociationOperations) GetSuperchargersForPlace(placeID string) ([]Supercharger, error) {
	var place Place
	if err := ops.db.Where("place_id = ?", placeID).First(&place).Error; err != nil {
		return nil, err
	}

	var superchargers []Supercharger
	err := ops.db.Model(&place).Association("Superchargers").Find(&superchargers)
	return superchargers, err
}

// GetPlacesForSupercharger retrieves all places associated with a supercharger
func (ops *PlaceAssociationOperations) GetPlacesForSupercharger(superchargerID string) ([]Place, error) {
	var supercharger Supercharger
	if err := ops.db.Where("place_id = ?", superchargerID).First(&supercharger).Error; err != nil {
		return nil, err
	}

	var places []Place
	err := ops.db.Model(&supercharger).Association("Places").Find(&places)
	return places, err
}

// GetPlaceAssociationOps returns operations for place-supercharger relationships
func (s *Service) GetPlaceAssociationOps() *PlaceAssociationOperations {
	return NewPlaceAssociationOperations(s.db)
}
