# Database Package

This package provides a comprehensive database layer using GORM with SQLite for the PassengerPrincess application. It includes models, repositories, and CRUD operations for all database entities.

## Features

- **GORM ORM**: Uses GORM for database operations with SQLite driver
- **Auto-migration**: Automatically creates and updates database schema
- **Repository pattern**: Clean separation of data access logic
- **Transaction support**: Built-in transaction handling
- **Relationships**: Many-to-many relationships between places and superchargers
- **Indexing**: Proper indexing for performance optimization
- **Logging**: API call logging and cache hit tracking

## Models

### Core Models
- **Place**: Google Places API locations with ratings and details
- **Supercharger**: Tesla supercharger locations
- **PlaceSupercharger**: Junction table for many-to-many relationships

### Logging Models
- **MapsCallLog**: Tracks Google Maps API calls
- **CacheHit**: Tracks cache hit/miss statistics
- **RouteCallLog**: Tracks route calculation requests

## Quick Start

### Initialize Database

```go
package main

import (
    "github.com/brensch/passengerprincess/pkg/db"
    "gorm.io/gorm/logger"
)

func main() {
    config := &db.Config{
        DatabasePath: "passengerprincess.db",
        LogLevel:     logger.Info,
    }

    if err := db.Initialize(config); err != nil {
        panic(err)
    }
    defer db.Close()

    // Get service instance
    service := db.GetDefaultService()
    
    // Use repositories...
}
```

### Basic CRUD Operations

#### Places

```go
// Create a place
place := &db.Place{
    PlaceID:            "ChIJN1t_tDeuEmsRUsoyG83frY4",
    Name:               "Sydney Opera House",
    Address:            "Bennelong Point, Sydney NSW 2000",
    Latitude:           -33.8568,
    Longitude:          151.2153,
    Rating:             4.5,
    UserRatingsTotal:   25000,
    PrimaryType:        "tourist_attraction",
    PrimaryTypeDisplay: "Tourist Attraction",
    DisplayName:        "Sydney Opera House",
}

err := service.Place.Create(place)

// Get by ID
place, err := service.Place.GetByID("ChIJN1t_tDeuEmsRUsoyG83frY4")

// Get with relationships
place, err := service.Place.GetByIDWithSuperchargers("ChIJN1t_tDeuEmsRUsoyG83frY4")

// Search by name/address
places, err := service.Place.Search("Opera House", 10)

// Get by location (bounding box)
places, err := service.Place.GetByLocation(-33.9, -33.8, 151.1, 151.3)

// Update
place.Rating = 4.6
err := service.Place.Update(place)

// Delete
err := service.Place.Delete("ChIJN1t_tDeuEmsRUsoyG83frY4")
```

#### Superchargers

```go
// Create a supercharger
supercharger := &db.Supercharger{
    PlaceID:   "ChIJexample_supercharger",
    Name:      "Sydney Supercharger",
    Address:   "123 Charging St, Sydney NSW",
    Latitude:  -33.8700,
    Longitude: 151.2100,
}

err := service.Supercharger.Create(supercharger)

// Similar CRUD operations as Places...
```

#### Associations

```go
// Associate a place with a supercharger
placeSuperchargerOps := service.GetPlaceSuperchargerOps()
err := placeSuperchargerOps.AddAssociation(placeID, superchargerID)

// Check if association exists
exists, err := placeSuperchargerOps.AssociationExists(placeID, superchargerID)

// Remove association
err := placeSuperchargerOps.RemoveAssociation(placeID, superchargerID)
```

### Logging Operations

#### Maps API Call Logging

```go
log := &db.MapsCallLog{
    SKU:            "placesearch",
    SuperchargerID: &superchargerID,
    PlaceID:        &placeID,
    Details:        "Searched for nearby restaurants",
}

err := service.MapsCallLog.Create(log)

// Get logs by time range
logs, err := service.MapsCallLog.GetByTimeRange(startTime, endTime, 100, 0)

// Get logs with errors
errorLogs, err := service.MapsCallLog.GetWithErrors(50, 0)
```

#### Cache Hit Tracking

```go
cacheHit := &db.CacheHit{
    ObjectID: placeID,
    Hit:      true,
    Type:     "place_details",
}

err := service.CacheHit.Upsert(cacheHit)

// Calculate hit rate
hitRate, err := service.CacheHit.GetHitRate("place_details")
fmt.Printf("Hit rate: %.2f%%\n", hitRate*100)
```

### Transactions

```go
err := service.Transaction(func(txService *db.Service) error {
    // Create place
    if err := txService.Place.Create(place); err != nil {
        return err
    }
    
    // Create supercharger
    if err := txService.Supercharger.Create(supercharger); err != nil {
        return err
    }
    
    // Create association
    ops := txService.GetPlaceSuperchargerOps()
    return ops.AddAssociation(place.PlaceID, supercharger.PlaceID)
})
```

## Configuration

The database can be configured with:

```go
config := &db.Config{
    DatabasePath: "custom.db",        // SQLite file path
    LogLevel:     logger.Silent,      // GORM log level
}
```

## Performance Features

- **WAL Mode**: Enabled for better concurrency
- **Proper Indexing**: Indexes on frequently queried columns
- **Connection Pooling**: Managed by GORM
- **Prepared Statements**: Automatic with GORM

## Error Handling

All repository methods return errors. Use `errors.Is(err, gorm.ErrRecordNotFound)` to check for "not found" conditions:

```go
place, err := service.Place.GetByID("nonexistent")
if errors.Is(err, gorm.ErrRecordNotFound) {
    // Handle not found
} else if err != nil {
    // Handle other errors
}
```

## Testing

Run the example to test the setup:

```bash
go run pkg/db/example/main.go
```

## Schema Migration

The package automatically handles schema migrations when `Initialize()` is called. It will create tables and update schemas as needed while preserving existing data.
