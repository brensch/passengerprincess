package maps

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

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

// SuperchargerWithETA contains supercharger information along with arrival time
type SuperchargerWithETA struct {
	Supercharger        *db.Supercharger `json:"supercharger"`
	ArrivalTime         string           `json:"arrival_time"`           // Formatted arrival time
	DistanceFromRoute   float64          `json:"distance_from_route"`    // Distance from route in meters
	DistanceAlongRoute  float64          `json:"distance_along_route"`   // Distance along route in meters
	ClosestPointOnRoute Center           `json:"closest_point_on_route"` // Closest point on the route
}

// CumPoint represents a point on the route with cumulative distance and duration
type CumPoint struct {
	Lat           float64
	Lng           float64
	CumDistKm     float64
	CumDurSeconds int
}

// LatLng represents a geographical point (for compatibility with main API)
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// distanceToPolyline calculates the shortest distance from a point to a polyline.
// It returns the shortest distance in meters, the cumulative distance along the polyline to that closest point,
// and the closest point on the polyline.
func distanceToPolyline(point Center, polyline []Center) (float64, float64, Center) {
	minDist := math.MaxFloat64
	distAlongRoute := 0.0
	cumulativeDist := 0.0
	var closestPoint Center

	for i := 0; i < len(polyline)-1; i++ {
		p1 := polyline[i]
		p2 := polyline[i+1]
		dist := distanceToSegment(point, p1, p2)

		if dist < minDist {
			minDist = dist
			// Find where on the segment the closest point lies
			l2 := (p1.Latitude-p2.Latitude)*(p1.Latitude-p2.Latitude) + (p1.Longitude-p2.Longitude)*(p1.Longitude-p2.Longitude)
			if l2 == 0.0 {
				closestPoint = p1
				distAlongRoute = cumulativeDist
			} else {
				t := ((point.Latitude-p1.Latitude)*(p2.Latitude-p1.Latitude) + (point.Longitude-p1.Longitude)*(p2.Longitude-p1.Longitude)) / l2
				t = math.Max(0, math.Min(1, t)) // Clamp to segment
				segmentLength := haversineDistance(p1, p2)
				distAlongRoute = cumulativeDist + t*segmentLength
				closestPoint = Center{
					Latitude:  p1.Latitude + t*(p2.Latitude-p1.Latitude),
					Longitude: p1.Longitude + t*(p2.Longitude-p1.Longitude),
				}
			}
		}
		cumulativeDist += haversineDistance(p1, p2)
	}
	return minDist, distAlongRoute, closestPoint
}

// distanceToSegment calculates the shortest distance from a point to a line segment.
func distanceToSegment(p, v, w Center) float64 {
	l2 := (v.Latitude-w.Latitude)*(v.Latitude-w.Latitude) + (v.Longitude-w.Longitude)*(v.Longitude-w.Longitude)
	if l2 == 0.0 {
		return haversineDistance(p, v)
	}
	t := ((p.Latitude-v.Latitude)*(w.Latitude-v.Latitude) + (p.Longitude-v.Longitude)*(w.Longitude-v.Longitude)) / l2
	t = math.Max(0, math.Min(1, t))

	closestLat := v.Latitude + t*(w.Latitude-v.Latitude)
	closestLng := v.Longitude + t*(w.Longitude-v.Longitude)
	return haversineDistance(p, Center{Latitude: closestLat, Longitude: closestLng})
}

// calculateETA computes the estimated arrival time for a supercharger
// based on route duration and distance from route
func calculateETA(cumulativePoints []CumPoint, distAlongRoute, distFromRoute float64) time.Time {
	// Find the closest cumulative point for accurate ETA
	var selectedCumDur int
	var foundDuration bool
	for _, cp := range cumulativePoints {
		if cp.CumDistKm >= distAlongRoute/1000.0 { // Convert meters to km
			selectedCumDur = cp.CumDurSeconds
			foundDuration = true
			break
		}
	}
	if !foundDuration && len(cumulativePoints) > 0 {
		selectedCumDur = cumulativePoints[len(cumulativePoints)-1].CumDurSeconds
	}

	// Calculate arrival time
	durationToSupercharger := time.Duration(selectedCumDur) * time.Second
	arrivalTime := time.Now().Add(durationToSupercharger)

	// Add time to travel from route to supercharger at 50 km/h
	extraTimeHours := (distFromRoute / 1000.0) / 50.0 // Convert meters to km, then to hours
	extraTimeSeconds := extraTimeHours * 3600
	arrivalTime = arrivalTime.Add(time.Duration(extraTimeSeconds) * time.Second)

	return arrivalTime
}

// SuperchargersOnRouteResult holds both the route information and the superchargers found along it
type SuperchargersOnRouteResult struct {
	Route         *RouteInfo            `json:"route"`
	Superchargers []SuperchargerWithETA `json:"superchargers"` // Superchargers with ETA information
	SearchCircles []Circle              `json:"search_circles"`
}

func GetSuperchargersOnRoute(ctx context.Context, broker *db.Service, apiKey, origin, destination string) (*SuperchargersOnRouteResult, error) {
	totalStart := time.Now()
	defer func() {
		log.Printf("GetSuperchargersOnRoute total time: %v", time.Since(totalStart))
	}()

	// Get route data (now enhanced with traffic information when available)
	routeStart := time.Now()
	route, err := GetRoute(apiKey, origin, destination)
	if err != nil {
		return nil, fmt.Errorf("failed to get route: %w", err)
	}
	log.Printf("Get route time: %v", time.Since(routeStart))

	// Decode the polyline to get route points
	decodeStart := time.Now()
	routePoints, err := DecodePolyline(route.EncodedPolyline)
	if err != nil {
		return nil, fmt.Errorf("failed to decode polyline: %w", err)
	}
	log.Printf("Decode polyline time: %v", time.Since(decodeStart))

	// Build cumulative profile for accurate ETAs if we have enhanced route data
	cumulativeStart := time.Now()
	var cumulativePoints []CumPoint
	if len(route.Legs) > 0 && len(route.Legs[0].Steps) > 0 {
		// We have enhanced route data with steps
		leg := route.Legs[0]
		cumDist := 0.0
		cumDur := 0

		// Calculate total duration with and without traffic to find a traffic multiplier
		sumStaticDur := 0
		for _, step := range leg.Steps {
			sumStaticDur += parseDurationString(step.StaticDuration)
		}
		totalTrafficDur := int(route.Duration.Seconds())

		trafficMultiplier := 1.0
		if sumStaticDur > 0 {
			trafficMultiplier = float64(totalTrafficDur) / float64(sumStaticDur)
		}

		// Build the point-by-point timeline for the route
		for _, step := range leg.Steps {
			stepPoints, err := DecodePolyline(step.Polyline.EncodedPolyline)
			if err != nil || len(stepPoints) == 0 {
				continue
			}
			totalStepDist := 0.0
			for i := 1; i < len(stepPoints); i++ {
				totalStepDist += haversineDistance(stepPoints[i-1], stepPoints[i])
			}

			// Apply the traffic multiplier to this step's static duration
			staticStepDur := parseDurationString(step.StaticDuration)
			trafficStepDur := int(float64(staticStepDur) * trafficMultiplier)

			stepCumDist := 0.0
			for i, p := range stepPoints {
				if i > 0 {
					dist := haversineDistance(stepPoints[i-1], p)
					stepCumDist += dist
					fraction := 0.0
					if totalStepDist > 0 {
						fraction = stepCumDist / totalStepDist
					}
					pointCumDur := cumDur + int(float64(trafficStepDur)*fraction)
					cumulativePoints = append(cumulativePoints, CumPoint{
						Lat:           p.Latitude,
						Lng:           p.Longitude,
						CumDistKm:     (cumDist + stepCumDist) / 1000.0, // Convert to km
						CumDurSeconds: pointCumDur,
					})
				} else {
					// First point of the step
					cumulativePoints = append(cumulativePoints, CumPoint{
						Lat:           p.Latitude,
						Lng:           p.Longitude,
						CumDistKm:     cumDist / 1000.0, // Convert to km
						CumDurSeconds: cumDur,
					})
				}
			}
			cumDist += totalStepDist
			cumDur += trafficStepDur
		}
	}
	log.Printf("Build cumulative profile time: %v", time.Since(cumulativeStart))

	// Get search circles
	circlesStart := time.Now()
	circles, err := PolylineToCircles(route.EncodedPolyline, SuperchargerSearchRadiusMeters)
	if err != nil {
		return nil, err
	}
	log.Printf("Get search circles time: %v", time.Since(circlesStart))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Get all the ids of superchargers along the route
	searchStart := time.Now()
	seenPlaceIDs := make(map[string]struct{})

	// Parallel search for superchargers
	type searchResult struct {
		places []*PlaceDetails
		err    error
	}
	searchResultsChan := make(chan searchResult, len(circles))
	var searchWg sync.WaitGroup

	for _, circle := range circles {
		searchWg.Add(1)
		go func(c Circle) {
			defer searchWg.Done()
			places, err := GetPlacesViaTextSearch(ctx, apiKey, "tesla supercharger", "places.id", c)
			searchResultsChan <- searchResult{places: places, err: err}
		}(circle)
	}

	go func() {
		searchWg.Wait()
		close(searchResultsChan)
	}()

	// Collect results
	for res := range searchResultsChan {
		if res.err != nil {
			cancel()
			return nil, res.err
		}
		for _, place := range res.places {
			seenPlaceIDs[place.ID] = struct{}{}
		}
	}
	log.Printf("Get supercharger IDs time: %v", time.Since(searchStart))

	// Fetch details concurrently
	fetchStart := time.Now()
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

	log.Printf("Fetch supercharger details time: %v", time.Since(fetchStart))

	// Process results and calculate ETAs
	processStart := time.Now()
	// Prepare the final list of superchargers with ETA info

	var superchargersWithETA []SuperchargerWithETA
	for res := range resultsChan {
		if res.err != nil {
			return nil, res.err
		}

		// skip non-superchargers
		if !res.supercharger.IsSupercharger {
			continue
		}

		sc := res.supercharger
		scLocation := Center{
			Latitude:  sc.Latitude,
			Longitude: sc.Longitude,
		}

		// Find closest point on route and calculate distances
		distFromRoute, distAlongRoute, closestPoint := distanceToPolyline(scLocation, routePoints)

		var arrivalTime time.Time
		if len(cumulativePoints) > 0 {
			// Calculate ETA using enhanced route data
			arrivalTime = calculateETA(cumulativePoints, distAlongRoute, distFromRoute)
		} else {
			// Basic ETA calculation without traffic data
			// Assume average speed of 80 km/h for highway driving
			avgSpeedKmh := 80.0
			timeToReachRoutePoint := (distAlongRoute / 1000.0) / avgSpeedKmh * 3600.0 // Convert to seconds
			timeToSupercharger := (distFromRoute / 1000.0) / 50.0 * 3600.0            // 50 km/h off-route, convert to seconds

			totalTravelTime := time.Duration(timeToReachRoutePoint+timeToSupercharger) * time.Second
			arrivalTime = time.Now().Add(totalTravelTime)
		}

		superchargersWithETA = append(superchargersWithETA, SuperchargerWithETA{
			Supercharger:        sc,
			ArrivalTime:         arrivalTime.Format(time.Kitchen), // e.g., "3:45PM"
			DistanceFromRoute:   distFromRoute,
			DistanceAlongRoute:  distAlongRoute,
			ClosestPointOnRoute: closestPoint,
		})
	}
	log.Printf("Fetch and process superchargers time: %v", time.Since(processStart))

	return &SuperchargersOnRouteResult{
		Route:         route,
		Superchargers: superchargersWithETA, // Superchargers with ETA information
		SearchCircles: circles,
	}, nil
}

const (
	FieldMaskRestaurantTextSearch = "places.id,places.displayName,places.formattedAddress,places.location,places.primaryType,places.primaryTypeDisplayName"
	// this is pro because of the usage of displayName. Without it we get non superchargers returned.
	// There is no way to force it to contain the exact text.
	FieldMaskSuperchargerDetails = "id,name,displayName,formattedAddress,location"
)

// GetSuperchargerWithCache retrieves place details with database caching
// First checks the database, then falls back to API if not found
func GetSuperchargerWithCache(ctx context.Context, broker *db.Service, apiKey, placeID string) (*db.Supercharger, error) {
	// First try to get from database
	start := time.Now()
	defer func() {
		// log the start, end, and duration
		log.Printf("GetSuperchargerWithCache start: %v end: %v total time for %s: %v", start, time.Now(), placeID, time.Since(start))
	}()
	supercharger, err := broker.Supercharger.GetByIDWithRestaurants(placeID)
	if err == nil {
		// Found in database, convert to PlaceDetails format
		return supercharger, nil
	}

	// Check if error is "not found" (expected when place doesn't exist in DB)
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query supercharger from database: %w", err)
	}

	log.Println("Supercharger not found in DB, fetching from API:", placeID)

	// Not found in database, fetch from API
	// this field map ensure the essentials tier
	superchargerDetails, err := GetPlaceDetails(ctx, apiKey, placeID, FieldMaskSuperchargerDetails)
	if err != nil {
		return nil, err
	}

	// exit early if site not a supercharger
	if !strings.Contains(strings.ToLower(superchargerDetails.DisplayName.Text), "supercharger") {
		log.Printf("Warning: Place ID %s does not appear to be a supercharger (name: %s). Recording without restaurants", placeID, superchargerDetails.DisplayName.Text)
		// Store in database for future use
		supercharger = &db.Supercharger{
			PlaceID:        superchargerDetails.ID,
			Name:           derefDisplayName(superchargerDetails.DisplayName),
			Address:        derefString(superchargerDetails.FormattedAddress),
			Latitude:       superchargerDetails.Location.Latitude,
			Longitude:      superchargerDetails.Location.Longitude,
			IsSupercharger: false,
		}

		err = broker.Supercharger.Create(supercharger)
		if err != nil {
			// Log the error but don't fail the request since we already have the data
			fmt.Printf("Warning: failed to cache supercharger %s in database: %v\n", placeID, err)
		}
		return supercharger, nil
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
		// check if restaurant is within 500m of supercharger
		if restaurant.Location == nil {
			continue
		}
		dist := haversineDistance(Center{
			Latitude:  superchargerDetails.Location.Latitude,
			Longitude: superchargerDetails.Location.Longitude,
		}, Center{
			Latitude:  restaurant.Location.Latitude,
			Longitude: restaurant.Location.Longitude,
		})
		if dist > 500 {
			continue
		}
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
		PlaceID:        superchargerDetails.ID,
		Name:           derefDisplayName(superchargerDetails.DisplayName),
		Address:        derefString(superchargerDetails.FormattedAddress),
		Latitude:       superchargerDetails.Location.Latitude,
		Longitude:      superchargerDetails.Location.Longitude,
		Restaurants:    dbRestaurants,
		IsSupercharger: true,
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
