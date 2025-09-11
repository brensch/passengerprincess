package db

import (
	"gorm.io/gorm"
)

// RestaurantRepository provides CRUD operations for Restaurant entities
type RestaurantRepository struct {
	db *gorm.DB
}

// NewRestaurantRepository creates a new RestaurantRepository
func NewRestaurantRepository(db *gorm.DB) *RestaurantRepository {
	return &RestaurantRepository{db: db}
}

// Create creates a new restaurant
func (r *RestaurantRepository) Create(restaurant *Restaurant) error {
	return r.db.Create(restaurant).Error
}

// CreateBatch creates multiple restaurants in a single transaction
func (r *RestaurantRepository) CreateBatch(restaurants []Restaurant) error {
	if len(restaurants) == 0 {
		return nil
	}
	return r.db.Create(&restaurants).Error
}

// CreateWithSupercharger creates a restaurant and associates it with a supercharger
func (r *RestaurantRepository) CreateWithSupercharger(restaurant *Restaurant, superchargerID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create the restaurant
		if err := tx.Create(restaurant).Error; err != nil {
			return err
		}

		// Associate with supercharger
		var supercharger Supercharger
		if err := tx.Where("place_id = ?", superchargerID).First(&supercharger).Error; err != nil {
			return err
		}

		return tx.Model(restaurant).Association("Superchargers").Append(&supercharger)
	})
}

// AssociateWithSupercharger associates an existing restaurant with a supercharger
func (r *RestaurantRepository) AssociateWithSupercharger(restaurantID, superchargerID string) error {
	var restaurant Restaurant
	var supercharger Supercharger

	if err := r.db.Where("place_id = ?", restaurantID).First(&restaurant).Error; err != nil {
		return err
	}

	if err := r.db.Where("place_id = ?", superchargerID).First(&supercharger).Error; err != nil {
		return err
	}

	return r.db.Model(&restaurant).Association("Superchargers").Append(&supercharger)
}

// BatchAssociateWithSuperchargers efficiently associates multiple restaurants with their superchargers
func (r *RestaurantRepository) BatchAssociateWithSuperchargers(associations []struct {
	RestaurantID   string
	SuperchargerID string
}) error {
	// Use raw SQL for efficient batch association
	for _, assoc := range associations {
		err := r.db.Exec(
			"INSERT OR IGNORE INTO restaurant_superchargers (restaurant_place_id, supercharger_place_id) VALUES (?, ?)",
			assoc.RestaurantID, assoc.SuperchargerID,
		).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// GetByID retrieves a restaurant by its ID
func (r *RestaurantRepository) GetByID(restaurantID string) (*Restaurant, error) {
	var restaurant Restaurant
	err := r.db.Where("place_id = ?", restaurantID).First(&restaurant).Error
	if err != nil {
		return nil, err
	}
	return &restaurant, nil
}

// GetByIDWithSuperchargers retrieves a restaurant by its ID with associated superchargers
func (r *RestaurantRepository) GetByIDWithSuperchargers(restaurantID string) (*Restaurant, error) {
	var restaurant Restaurant
	err := r.db.Preload("Superchargers").Where("place_id = ?", restaurantID).First(&restaurant).Error
	if err != nil {
		return nil, err
	}
	return &restaurant, nil
}

// GetAll retrieves all restaurants with optional pagination
func (r *RestaurantRepository) GetAll(limit, offset int) ([]Restaurant, error) {
	var restaurants []Restaurant
	query := r.db

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&restaurants).Error
	return restaurants, err
}

// GetByLocation retrieves restaurants within a bounding box
func (r *RestaurantRepository) GetByLocation(minLat, maxLat, minLng, maxLng float64) ([]Restaurant, error) {
	var restaurants []Restaurant
	err := r.db.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
		minLat, maxLat, minLng, maxLng).Find(&restaurants).Error
	return restaurants, err
}

// Update updates an existing restaurant
func (r *RestaurantRepository) Update(restaurant *Restaurant) error {
	return r.db.Save(restaurant).Error
}

// Delete deletes a restaurant by ID
func (r *RestaurantRepository) Delete(restaurantID string) error {
	return r.db.Where("place_id = ?", restaurantID).Delete(&Restaurant{}).Error
}

// Search searches restaurants by name or address
func (r *RestaurantRepository) Search(query string, limit int) ([]Restaurant, error) {
	var restaurants []Restaurant
	searchQuery := "%" + query + "%"
	err := r.db.Where("name LIKE ? OR address LIKE ?", searchQuery, searchQuery).
		Limit(limit).Find(&restaurants).Error
	return restaurants, err
}

// Count returns total number of restaurants
func (r *RestaurantRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&Restaurant{}).Count(&count).Error
	return count, err
}

// SuperchargerRepository provides CRUD operations for Supercharger entities
type SuperchargerRepository struct {
	db *gorm.DB
}

// NewSuperchargerRepository creates a new SuperchargerRepository
func NewSuperchargerRepository(db *gorm.DB) *SuperchargerRepository {
	return &SuperchargerRepository{db: db}
}

// Create creates a new supercharger
func (r *SuperchargerRepository) Create(supercharger *Supercharger) error {
	return r.db.Create(supercharger).Error
}

// CreateBatch creates multiple superchargers in a single transaction
func (r *SuperchargerRepository) CreateBatch(superchargers []Supercharger) error {
	if len(superchargers) == 0 {
		return nil
	}
	return r.db.Create(&superchargers).Error
}

// GetByID retrieves a supercharger by its ID
func (r *SuperchargerRepository) GetByID(placeID string) (*Supercharger, error) {
	var supercharger Supercharger
	err := r.db.Where("place_id = ?", placeID).First(&supercharger).Error
	if err != nil {
		return nil, err
	}
	return &supercharger, nil
}

// GetByIDWithRestaurants retrieves a supercharger by its ID with associated restaurants
func (r *SuperchargerRepository) GetByIDWithRestaurants(placeID string) (*Supercharger, error) {
	var supercharger Supercharger
	err := r.db.Preload("Restaurants").Where("place_id = ?", placeID).First(&supercharger).Error
	if err != nil {
		return nil, err
	}
	return &supercharger, nil
}

// GetAll retrieves all superchargers with optional pagination
func (r *SuperchargerRepository) GetAll(limit, offset int) ([]Supercharger, error) {
	var superchargers []Supercharger
	query := r.db

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&superchargers).Error
	return superchargers, err
}

// GetByLocation retrieves superchargers within a bounding box
func (r *SuperchargerRepository) GetByLocation(minLat, maxLat, minLng, maxLng float64) ([]Supercharger, error) {
	var superchargers []Supercharger
	err := r.db.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
		minLat, maxLat, minLng, maxLng).Find(&superchargers).Error
	return superchargers, err
}

// Update updates an existing supercharger
func (r *SuperchargerRepository) Update(supercharger *Supercharger) error {
	return r.db.Save(supercharger).Error
}

// Delete deletes a supercharger by ID
func (r *SuperchargerRepository) Delete(placeID string) error {
	return r.db.Where("place_id = ?", placeID).Delete(&Supercharger{}).Error
}

// Search searches superchargers by name or address
func (r *SuperchargerRepository) Search(query string, limit int) ([]Supercharger, error) {
	var superchargers []Supercharger
	searchQuery := "%" + query + "%"
	err := r.db.Where("name LIKE ? OR address LIKE ?", searchQuery, searchQuery).
		Limit(limit).Find(&superchargers).Error
	return superchargers, err
}

// Count returns total number of superchargers
func (r *SuperchargerRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&Supercharger{}).Count(&count).Error
	return count, err
}
