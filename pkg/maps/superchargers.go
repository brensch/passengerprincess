package maps

import (
	"context"

	"github.com/brensch/passengerprincess/pkg/db"
)

const (
	// SuperchargerSearchRadiusMeters defines the search radius around each circle to look for superchargers
	SuperchargerSearchRadiusMeters = 5000
)

func GetSuperchargersOnRoute(ctx context.Context, broker *db.Service, apiKey, origin, destination string) ([]*db.Supercharger, error) {
	route, err := GetRoute(apiKey, origin, destination)
	if err != nil {
		return nil, err
	}

	circles, err := PolylineToCircles(route.EncodedPolyline, SuperchargerSearchRadiusMeters) // 10 km radius
	if err != nil {
		return nil, err
	}

	// get all the ids of superchargers along the route
	seenPlaceIDs := make(map[string]struct{})
	var superchargers []*db.Supercharger
	for _, circle := range circles {
		placeIDs, err := GetPlaceIDsViaTextSearch(apiKey, "tesla supercharger", circle)
		if err != nil {
			return nil, err
		}
		for _, id := range placeIDs {
			if _, seen := seenPlaceIDs[id]; seen {
				continue
			}

			seenPlaceIDs[id] = struct{}{}

			placeDetails, err := GetPlaceDetailsWithCache(broker, apiKey, id)
			if err != nil {
				return nil, err
			}

			superCharger := &db.Supercharger{
				PlaceID:   id,
				Name:      placeDetails.Name,
				Address:   placeDetails.FormattedAddress,
				Latitude:  placeDetails.Location.Latitude,
				Longitude: placeDetails.Location.Longitude,
			}
			superchargers = append(superchargers, superCharger)

		}
	}

	return superchargers, nil
}
