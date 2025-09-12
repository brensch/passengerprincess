package db

import (
	"time"
)

// Restaurant represents a restaurant from Google Places API
type Restaurant struct {
	PlaceID            string    `gorm:"primaryKey;column:place_id" json:"place_id"`
	Name               string    `gorm:"column:name" json:"name"`
	Address            string    `gorm:"column:address" json:"address"`
	Latitude           float64   `gorm:"column:latitude" json:"latitude"`
	Longitude          float64   `gorm:"column:longitude" json:"longitude"`
	Rating             float64   `gorm:"column:rating" json:"rating"`
	UserRatingsTotal   int       `gorm:"column:user_ratings_total" json:"user_ratings_total"`
	PrimaryType        string    `gorm:"column:primary_type" json:"primary_type"`
	PrimaryTypeDisplay string    `gorm:"column:primary_type_display" json:"primary_type_display"`
	DisplayName        string    `gorm:"column:display_name" json:"display_name"`
	LastUpdated        time.Time `gorm:"column:last_updated;default:CURRENT_TIMESTAMP" json:"last_updated"`
}

// TableName returns the table name for Restaurant
func (Restaurant) TableName() string {
	return "restaurants"
}

// Supercharger represents a Tesla supercharger location
type Supercharger struct {
	PlaceID     string    `gorm:"primaryKey;column:place_id" json:"place_id"`
	Name        string    `gorm:"column:name" json:"name"`
	Address     string    `gorm:"column:address" json:"address"`
	Latitude    float64   `gorm:"column:latitude" json:"latitude"`
	Longitude   float64   `gorm:"column:longitude" json:"longitude"`
	LastUpdated time.Time `gorm:"column:last_updated;default:CURRENT_TIMESTAMP" json:"last_updated"`
	// this is in order to keep track of IDs that get returned that aren't actually superchargers
	IsSupercharger bool `gorm:"column:is_supercharger" json:"is_supercharger"`
}

// TableName returns the table name for Supercharger
func (Supercharger) TableName() string {
	return "superchargers"
}

// MapsCallLog represents API call logging for maps operations
type MapsCallLog struct {
	ID             uint      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	SKU            string    `gorm:"column:sku" json:"sku"`
	Timestamp      time.Time `gorm:"column:timestamp;default:CURRENT_TIMESTAMP" json:"timestamp"`
	SuperchargerID *string   `gorm:"column:supercharger_id" json:"supercharger_id"`
	PlaceID        *string   `gorm:"column:place_id" json:"place_id"`
	Error          string    `gorm:"column:error" json:"error"`
	Details        string    `gorm:"column:details" json:"details"`
}

// CacheHit represents cache hit tracking
type CacheHit struct {
	ObjectID    string    `gorm:"primaryKey;column:object_id" json:"object_id"`
	Hit         bool      `gorm:"column:hit" json:"hit"`
	LastUpdated time.Time `gorm:"column:last_updated;default:CURRENT_TIMESTAMP" json:"last_updated"`
	Type        string    `gorm:"column:type" json:"type"`
}

// RestaurantWithDistance represents a restaurant with its distance to a supercharger
type RestaurantWithDistance struct {
	Restaurant
	Distance float64 `json:"distance"`
}

// RestaurantSuperchargerMapping represents the mapping between restaurants and superchargers with distance
type RestaurantSuperchargerMapping struct {
	RestaurantID   string       `gorm:"primaryKey;column:restaurant_id;constraint:OnDelete:CASCADE" json:"restaurant_id"`
	SuperchargerID string       `gorm:"primaryKey;column:supercharger_id;constraint:OnDelete:CASCADE" json:"supercharger_id"`
	Distance       float64      `gorm:"column:distance" json:"distance"`
	Restaurant     Restaurant   `gorm:"foreignKey:RestaurantID;references:PlaceID"`
	Supercharger   Supercharger `gorm:"foreignKey:SuperchargerID;references:PlaceID"`
}

// TableName returns the table name for RestaurantSuperchargerMapping
func (RestaurantSuperchargerMapping) TableName() string {
	return "restaurant_supercharger_mappings"
}

// RouteCallLog represents route API call logging
type RouteCallLog struct {
	ID          uint      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Timestamp   time.Time `gorm:"column:timestamp;default:CURRENT_TIMESTAMP" json:"timestamp"`
	Origin      string    `gorm:"column:origin" json:"origin"`
	Destination string    `gorm:"column:destination" json:"destination"`
	Error       string    `gorm:"column:error" json:"error"`
	IPAddress   string    `gorm:"column:ip_address" json:"ip_address"`
}
