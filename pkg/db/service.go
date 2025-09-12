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
