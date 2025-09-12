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

// GetByID retrieves a restaurant by its ID
func (r *RestaurantRepository) GetByID(restaurantID string) (*Restaurant, error) {
	var restaurant Restaurant
	err := r.db.Where("place_id = ?", restaurantID).First(&restaurant).Error
	if err != nil {
		return nil, err
	}
	return &restaurant, nil
}

// GetByLocation retrieves restaurants within a bounding box
func (r *RestaurantRepository) GetByLocation(minLat, maxLat, minLng, maxLng float64) ([]Restaurant, error) {
	var restaurants []Restaurant
	err := r.db.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
		minLat, maxLat, minLng, maxLng).Find(&restaurants).Error
	return restaurants, err
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

// GetByID retrieves a supercharger by its ID
func (r *SuperchargerRepository) GetByID(placeID string) (*Supercharger, error) {
	var supercharger Supercharger
	err := r.db.Where("place_id = ?", placeID).First(&supercharger).Error
	if err != nil {
		return nil, err
	}
	return &supercharger, nil
}

// GetByLocation retrieves superchargers within a bounding box
func (r *SuperchargerRepository) GetByLocation(minLat, maxLat, minLng, maxLng float64) ([]Supercharger, error) {
	var superchargers []Supercharger
	err := r.db.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
		minLat, maxLat, minLng, maxLng).Find(&superchargers).Error
	return superchargers, err
}

// GetRestaurantsForSupercharger retrieves all restaurants associated with a supercharger with distances
func (r *SuperchargerRepository) GetRestaurantsForSupercharger(superchargerID string) ([]RestaurantWithDistance, error) {
	var results []struct {
		Restaurant
		Distance float64 `json:"distance"`
	}

	err := r.db.Table("restaurants").
		Select("restaurants.*, restaurant_supercharger_mappings.distance").
		Joins("JOIN restaurant_supercharger_mappings ON restaurants.place_id = restaurant_supercharger_mappings.restaurant_id").
		Where("restaurant_supercharger_mappings.supercharger_id = ?", superchargerID).
		Scan(&results).Error

	restaurantsWithDistance := make([]RestaurantWithDistance, len(results))
	for i, result := range results {
		restaurantsWithDistance[i] = RestaurantWithDistance{
			Restaurant: result.Restaurant,
			Distance:   result.Distance,
		}
	}

	return restaurantsWithDistance, err
}

// AddSuperchargerWithRestaurants creates a supercharger and associates it with multiple restaurants with distances
func (r *SuperchargerRepository) AddSuperchargerWithRestaurants(supercharger *Supercharger, restaurants []RestaurantWithDistance) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create the supercharger
		if err := tx.Create(supercharger).Error; err != nil {
			return err
		}

		// Create restaurants if they don't exist
		for _, restaurant := range restaurants {
			var existing Restaurant
			if err := tx.Where("place_id = ?", restaurant.PlaceID).First(&existing).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					// Restaurant doesn't exist, create it
					newRestaurant := Restaurant{
						PlaceID:            restaurant.PlaceID,
						Name:               restaurant.Name,
						Address:            restaurant.Address,
						Latitude:           restaurant.Latitude,
						Longitude:          restaurant.Longitude,
						Rating:             restaurant.Rating,
						UserRatingsTotal:   restaurant.UserRatingsTotal,
						PrimaryType:        restaurant.PrimaryType,
						PrimaryTypeDisplay: restaurant.PrimaryTypeDisplay,
						DisplayName:        restaurant.DisplayName,
						LastUpdated:        restaurant.LastUpdated,
					}
					if err := tx.Create(&newRestaurant).Error; err != nil {
						return err
					}
				} else {
					return err
				}
			}

			// Create the mapping with distance
			mapping := RestaurantSuperchargerMapping{
				RestaurantID:   restaurant.PlaceID,
				SuperchargerID: supercharger.PlaceID,
				Distance:       restaurant.Distance,
			}
			err := tx.Create(&mapping).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}
