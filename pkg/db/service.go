package db

import (
	"gorm.io/gorm"
)

// Service provides a unified interface to all database operations
type Service struct {
	Restaurant   *RestaurantRepository
	Supercharger *SuperchargerRepository
	MapsCallLog  *MapsCallLogRepository
	CacheHit     *CacheHitRepository
	RouteCallLog *RouteCallLogRepository
	db           *gorm.DB
}

// NewService creates a new database service with all repositories
func NewService(db *gorm.DB) *Service {
	return &Service{
		Restaurant:   NewRestaurantRepository(db),
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

// RestaurantAssociationOperations provides operations for restaurant-supercharger associations using GORM's many-to-many
type RestaurantAssociationOperations struct {
	db *gorm.DB
}

// NewRestaurantAssociationOperations creates operations for restaurant-supercharger relationships
func NewRestaurantAssociationOperations(db *gorm.DB) *RestaurantAssociationOperations {
	return &RestaurantAssociationOperations{db: db}
}

// AddAssociation creates an association between a restaurant and supercharger
func (ops *RestaurantAssociationOperations) AddAssociation(restaurantID, superchargerID string) error {
	restaurant := Restaurant{PlaceID: restaurantID}
	supercharger := Supercharger{PlaceID: superchargerID}
	return ops.db.Model(&restaurant).Association("Superchargers").Append(&supercharger)
}

// RemoveAssociation removes an association between a restaurant and supercharger
func (ops *RestaurantAssociationOperations) RemoveAssociation(restaurantID, superchargerID string) error {
	restaurant := Restaurant{PlaceID: restaurantID}
	supercharger := Supercharger{PlaceID: superchargerID}
	return ops.db.Model(&restaurant).Association("Superchargers").Delete(&supercharger)
}

// AssociationExists checks if an association exists between a restaurant and supercharger
func (ops *RestaurantAssociationOperations) AssociationExists(restaurantID, superchargerID string) (bool, error) {
	var restaurant Restaurant
	if err := ops.db.Where("place_id = ?", restaurantID).First(&restaurant).Error; err != nil {
		return false, err
	}

	var supercharger Supercharger
	if err := ops.db.Where("place_id = ?", superchargerID).First(&supercharger).Error; err != nil {
		return false, err
	}

	count := ops.db.Model(&restaurant).Association("Superchargers").Count()
	if count == 0 {
		return false, nil
	}

	// Check if this specific supercharger is associated
	var result Supercharger
	err := ops.db.Model(&restaurant).Association("Superchargers").Find(&result, "place_id = ?", superchargerID)
	return err == nil && result.PlaceID == superchargerID, nil
}

// GetSuperchargersForRestaurant retrieves all superchargers associated with a restaurant
func (ops *RestaurantAssociationOperations) GetSuperchargersForRestaurant(restaurantID string) ([]Supercharger, error) {
	var restaurant Restaurant
	if err := ops.db.Where("place_id = ?", restaurantID).First(&restaurant).Error; err != nil {
		return nil, err
	}

	var superchargers []Supercharger
	err := ops.db.Model(&restaurant).Association("Superchargers").Find(&superchargers)
	return superchargers, err
}

// GetRestaurantsForSupercharger retrieves all restaurants associated with a supercharger
func (ops *RestaurantAssociationOperations) GetRestaurantsForSupercharger(superchargerID string) ([]Restaurant, error) {
	var supercharger Supercharger
	if err := ops.db.Where("place_id = ?", superchargerID).First(&supercharger).Error; err != nil {
		return nil, err
	}

	var restaurants []Restaurant
	err := ops.db.Model(&supercharger).Association("Restaurants").Find(&restaurants)
	return restaurants, err
}

// GetRestaurantAssociationOps returns operations for restaurant-supercharger relationships
func (s *Service) GetRestaurantAssociationOps() *RestaurantAssociationOperations {
	return NewRestaurantAssociationOperations(s.db)
}
