package maps

import (
	"context"
	"fmt"
	"sync"

	"github.com/brensch/passengerprincess/pkg/db"
	"gorm.io/gorm"
)

const (
	// SuperchargerSearchRadiusMeters defines the search radius around each circle to look for superchargers
	SuperchargerSearchRadiusMeters = 5000
)

type superchargerResult struct {
	supercharger *db.Supercharger
	err          error
}

// SuperchargersOnRouteResult holds both the route information and the superchargers found along it
type SuperchargersOnRouteResult struct {
	Route         *RouteInfo
	Superchargers []*db.Supercharger
	SearchCircles []Circle
}

func GetSuperchargersOnRoute(ctx context.Context, broker *db.Service, apiKey, origin, destination string) (*SuperchargersOnRouteResult, error) {
	route, err := GetRoute(apiKey, origin, destination)
	if err != nil {
		return nil, err
	}

	circles, err := PolylineToCircles(route.EncodedPolyline, SuperchargerSearchRadiusMeters)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// get all the ids of superchargers along the route
	seenPlaceIDs := make(map[string]struct{})
	for _, circle := range circles {
		places, err := GetPlacesViaTextSearch(ctx, apiKey, "tesla supercharger", "places.id", circle)
		if err != nil {
			return nil, err
		}
		for _, place := range places {
			seenPlaceIDs[place.ID] = struct{}{}
		}
	}

	// fetch details concurrently
	resultsChan := make(chan superchargerResult, len(seenPlaceIDs))
	var wg sync.WaitGroup
	for id := range seenPlaceIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			superCharger, err := GetSuperchargerWithCache(ctx, broker, apiKey, id)
			resultsChan <- superchargerResult{supercharger: superCharger, err: err}
		}(id)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var superchargers []*db.Supercharger
	for res := range resultsChan {
		if res.err != nil {
			return nil, res.err
		}
		superchargers = append(superchargers, res.supercharger)
	}

	return &SuperchargersOnRouteResult{
		Route:         route,
		Superchargers: superchargers,
		SearchCircles: circles,
	}, nil
}

const (
	FieldMaskRestaurantTextSearch = "places.id,places.displayName,places.formattedAddress,places.location,places.primaryType,places.primaryTypeDisplayName"
	FieldMaskSuperchargerDetails  = "id,name,formattedAddress,location"
)

// GetSuperchargerWithCache retrieves place details with database caching
// First checks the database, then falls back to API if not found
func GetSuperchargerWithCache(ctx context.Context, broker *db.Service, apiKey, placeID string) (*db.Supercharger, error) {
	// First try to get from database
	supercharger, err := broker.Supercharger.GetByIDWithRestaurants(placeID)
	if err == nil {
		// Found in database, convert to PlaceDetails format
		return supercharger, nil
	}

	// Check if error is "not found" (expected when place doesn't exist in DB)
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query supercharger from database: %w", err)
	}

	// Not found in database, fetch from API
	// this field map ensure the essentials tier
	superchargerDetails, err := GetPlaceDetails(ctx, apiKey, placeID, FieldMaskSuperchargerDetails)
	if err != nil {
		return nil, err
	}

	restaurants, err := GetPlacesViaTextSearch(ctx, apiKey, "restaurant", FieldMaskRestaurantTextSearch, Circle{
		Center: Center{
			Latitude:  superchargerDetails.Location.Latitude,
			Longitude: superchargerDetails.Location.Longitude,
		},
		Radius: 500, // 500 meter radius
	})
	if err != nil {
		return nil, err
	}

	var dbRestaurants []db.Restaurant
	for _, restaurant := range restaurants {
		dbRestaurant := db.Restaurant{
			PlaceID:            restaurant.ID,
			Name:               derefDisplayName(restaurant.DisplayName),
			Address:            derefString(restaurant.FormattedAddress),
			Latitude:           restaurant.Location.Latitude,
			Longitude:          restaurant.Location.Longitude,
			PrimaryType:        derefString(restaurant.PrimaryType),
			PrimaryTypeDisplay: derefDisplayName(restaurant.PrimaryTypeDisplayName),
		}
		dbRestaurants = append(dbRestaurants, dbRestaurant)
	}

	// Store in database for future use
	supercharger = &db.Supercharger{
		PlaceID:     superchargerDetails.ID,
		Name:        derefDisplayName(superchargerDetails.DisplayName),
		Address:     derefString(superchargerDetails.FormattedAddress),
		Latitude:    superchargerDetails.Location.Latitude,
		Longitude:   superchargerDetails.Location.Longitude,
		Restaurants: dbRestaurants,
	}

	err = broker.Supercharger.Create(supercharger)
	if err != nil {
		// Log the error but don't fail the request since we already have the data
		fmt.Printf("Warning: failed to cache supercharger %s in database: %v\n", placeID, err)
	}

	return supercharger, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefDisplayName(dn *DisplayNameObj) string {
	if dn == nil {
		return ""
	}
	return dn.Text
}
