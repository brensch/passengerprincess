package db

import (
	"time"

	"gorm.io/gorm"
)

// MapsCallLogRepository provides CRUD operations for MapsCallLog entities
type MapsCallLogRepository struct {
	db *gorm.DB
}

// NewMapsCallLogRepository creates a new MapsCallLogRepository
func NewMapsCallLogRepository(db *gorm.DB) *MapsCallLogRepository {
	return &MapsCallLogRepository{db: db}
}

// Create creates a new maps call log entry
func (r *MapsCallLogRepository) Create(log *MapsCallLog) error {
	return r.db.Create(log).Error
}

// GetByID retrieves a maps call log by its ID
func (r *MapsCallLogRepository) GetByID(id uint) (*MapsCallLog, error) {
	var log MapsCallLog
	err := r.db.Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetByTimeRange retrieves logs within a time range
func (r *MapsCallLogRepository) GetByTimeRange(start, end time.Time, limit, offset int) ([]MapsCallLog, error) {
	var logs []MapsCallLog
	query := r.db.Where("timestamp BETWEEN ? AND ?", start, end).Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// GetBySKU retrieves logs by SKU
func (r *MapsCallLogRepository) GetBySKU(sku string, limit, offset int) ([]MapsCallLog, error) {
	var logs []MapsCallLog
	query := r.db.Where("sku = ?", sku).Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// GetWithErrors retrieves logs that have errors
func (r *MapsCallLogRepository) GetWithErrors(limit, offset int) ([]MapsCallLog, error) {
	var logs []MapsCallLog
	query := r.db.Where("error != ''").Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// Delete deletes a maps call log by ID
func (r *MapsCallLogRepository) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&MapsCallLog{}).Error
}

// DeleteOlderThan deletes logs older than the specified time
func (r *MapsCallLogRepository) DeleteOlderThan(cutoff time.Time) error {
	return r.db.Where("timestamp < ?", cutoff).Delete(&MapsCallLog{}).Error
}

// Count returns total number of logs
func (r *MapsCallLogRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&MapsCallLog{}).Count(&count).Error
	return count, err
}

// CacheHitRepository provides CRUD operations for CacheHit entities
type CacheHitRepository struct {
	db *gorm.DB
}

// NewCacheHitRepository creates a new CacheHitRepository
func NewCacheHitRepository(db *gorm.DB) *CacheHitRepository {
	return &CacheHitRepository{db: db}
}

// Create creates a new cache hit entry
func (r *CacheHitRepository) Create(cacheHit *CacheHit) error {
	return r.db.Create(cacheHit).Error
}

// GetByID retrieves a cache hit by its object ID
func (r *CacheHitRepository) GetByID(objectID string) (*CacheHit, error) {
	var cacheHit CacheHit
	err := r.db.Where("object_id = ?", objectID).First(&cacheHit).Error
	if err != nil {
		return nil, err
	}
	return &cacheHit, nil
}

// GetByType retrieves cache hits by type
func (r *CacheHitRepository) GetByType(cacheType string, limit, offset int) ([]CacheHit, error) {
	var cacheHits []CacheHit
	query := r.db.Where("type = ?", cacheType).Order("last_updated DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&cacheHits).Error
	return cacheHits, err
}

// Update updates an existing cache hit
func (r *CacheHitRepository) Update(cacheHit *CacheHit) error {
	return r.db.Save(cacheHit).Error
}

// Upsert creates or updates a cache hit entry
func (r *CacheHitRepository) Upsert(cacheHit *CacheHit) error {
	return r.db.Save(cacheHit).Error
}

// Delete deletes a cache hit by object ID
func (r *CacheHitRepository) Delete(objectID string) error {
	return r.db.Where("object_id = ?", objectID).Delete(&CacheHit{}).Error
}

// GetHitRate calculates cache hit rate for a specific type
func (r *CacheHitRepository) GetHitRate(cacheType string) (float64, error) {
	var total, hits int64

	// Count total entries of this type
	err := r.db.Model(&CacheHit{}).Where("type = ?", cacheType).Count(&total).Error
	if err != nil {
		return 0, err
	}

	if total == 0 {
		return 0, nil
	}

	// Count hits
	err = r.db.Model(&CacheHit{}).Where("type = ? AND hit = true", cacheType).Count(&hits).Error
	if err != nil {
		return 0, err
	}

	return float64(hits) / float64(total), nil
}

// RouteCallLogRepository provides CRUD operations for RouteCallLog entities
type RouteCallLogRepository struct {
	db *gorm.DB
}

// NewRouteCallLogRepository creates a new RouteCallLogRepository
func NewRouteCallLogRepository(db *gorm.DB) *RouteCallLogRepository {
	return &RouteCallLogRepository{db: db}
}

// Create creates a new route call log entry
func (r *RouteCallLogRepository) Create(log *RouteCallLog) error {
	return r.db.Create(log).Error
}

// GetByID retrieves a route call log by its ID
func (r *RouteCallLogRepository) GetByID(id uint) (*RouteCallLog, error) {
	var log RouteCallLog
	err := r.db.Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetByTimeRange retrieves logs within a time range
func (r *RouteCallLogRepository) GetByTimeRange(start, end time.Time, limit, offset int) ([]RouteCallLog, error) {
	var logs []RouteCallLog
	query := r.db.Where("timestamp BETWEEN ? AND ?", start, end).Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// GetByIPAddress retrieves logs by IP address
func (r *RouteCallLogRepository) GetByIPAddress(ipAddress string, limit, offset int) ([]RouteCallLog, error) {
	var logs []RouteCallLog
	query := r.db.Where("ip_address = ?", ipAddress).Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// GetWithErrors retrieves logs that have errors
func (r *RouteCallLogRepository) GetWithErrors(limit, offset int) ([]RouteCallLog, error) {
	var logs []RouteCallLog
	query := r.db.Where("error != ''").Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// Delete deletes a route call log by ID
func (r *RouteCallLogRepository) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&RouteCallLog{}).Error
}

// DeleteOlderThan deletes logs older than the specified time
func (r *RouteCallLogRepository) DeleteOlderThan(cutoff time.Time) error {
	return r.db.Where("timestamp < ?", cutoff).Delete(&RouteCallLog{}).Error
}

// Count returns total number of route logs
func (r *RouteCallLogRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&RouteCallLog{}).Count(&count).Error
	return count, err
}
