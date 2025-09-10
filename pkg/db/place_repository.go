package db

import (
	"gorm.io/gorm"
)

// PlaceRepository provides CRUD operations for Place entities
type PlaceRepository struct {
	db *gorm.DB
}

// NewPlaceRepository creates a new PlaceRepository
func NewPlaceRepository(db *gorm.DB) *PlaceRepository {
	return &PlaceRepository{db: db}
}

// Create creates a new place
func (r *PlaceRepository) Create(place *Place) error {
	return r.db.Create(place).Error
}

// CreateBatch creates multiple places in a single transaction
func (r *PlaceRepository) CreateBatch(places []Place) error {
	if len(places) == 0 {
		return nil
	}
	return r.db.Create(&places).Error
}

// CreateWithSupercharger creates a place and associates it with a supercharger
func (r *PlaceRepository) CreateWithSupercharger(place *Place, superchargerID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create the place
		if err := tx.Create(place).Error; err != nil {
			return err
		}

		// Associate with supercharger
		var supercharger Supercharger
		if err := tx.Where("place_id = ?", superchargerID).First(&supercharger).Error; err != nil {
			return err
		}

		return tx.Model(place).Association("Superchargers").Append(&supercharger)
	})
}

// AssociateWithSupercharger associates an existing place with a supercharger
func (r *PlaceRepository) AssociateWithSupercharger(placeID, superchargerID string) error {
	var place Place
	var supercharger Supercharger

	if err := r.db.Where("place_id = ?", placeID).First(&place).Error; err != nil {
		return err
	}

	if err := r.db.Where("place_id = ?", superchargerID).First(&supercharger).Error; err != nil {
		return err
	}

	return r.db.Model(&place).Association("Superchargers").Append(&supercharger)
}

// BatchAssociateWithSuperchargers efficiently associates multiple places with their superchargers
func (r *PlaceRepository) BatchAssociateWithSuperchargers(associations []struct {
	PlaceID        string
	SuperchargerID string
}) error {
	// Use raw SQL for efficient batch association
	for _, assoc := range associations {
		err := r.db.Exec(
			"INSERT OR IGNORE INTO place_superchargers (place_place_id, supercharger_place_id) VALUES (?, ?)",
			assoc.PlaceID, assoc.SuperchargerID,
		).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// GetByID retrieves a place by its ID
func (r *PlaceRepository) GetByID(placeID string) (*Place, error) {
	var place Place
	err := r.db.Where("place_id = ?", placeID).First(&place).Error
	if err != nil {
		return nil, err
	}
	return &place, nil
}

// GetByIDWithSuperchargers retrieves a place by its ID with associated superchargers
func (r *PlaceRepository) GetByIDWithSuperchargers(placeID string) (*Place, error) {
	var place Place
	err := r.db.Preload("Superchargers").Where("place_id = ?", placeID).First(&place).Error
	if err != nil {
		return nil, err
	}
	return &place, nil
}

// GetAll retrieves all places with optional pagination
func (r *PlaceRepository) GetAll(limit, offset int) ([]Place, error) {
	var places []Place
	query := r.db

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&places).Error
	return places, err
}

// GetByLocation retrieves places within a bounding box
func (r *PlaceRepository) GetByLocation(minLat, maxLat, minLng, maxLng float64) ([]Place, error) {
	var places []Place
	err := r.db.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
		minLat, maxLat, minLng, maxLng).Find(&places).Error
	return places, err
}

// Update updates an existing place
func (r *PlaceRepository) Update(place *Place) error {
	return r.db.Save(place).Error
}

// Delete deletes a place by ID
func (r *PlaceRepository) Delete(placeID string) error {
	return r.db.Where("place_id = ?", placeID).Delete(&Place{}).Error
}

// Search searches places by name or address
func (r *PlaceRepository) Search(query string, limit int) ([]Place, error) {
	var places []Place
	searchQuery := "%" + query + "%"
	err := r.db.Where("name LIKE ? OR address LIKE ?", searchQuery, searchQuery).
		Limit(limit).Find(&places).Error
	return places, err
}

// Count returns total number of places
func (r *PlaceRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&Place{}).Count(&count).Error
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

// GetByIDWithPlaces retrieves a supercharger by its ID with associated places
func (r *SuperchargerRepository) GetByIDWithPlaces(placeID string) (*Supercharger, error) {
	var supercharger Supercharger
	err := r.db.Preload("Places").Where("place_id = ?", placeID).First(&supercharger).Error
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
