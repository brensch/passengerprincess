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
	restaurants  []db.RestaurantWithDistance
	err          error
}

// SuperchargerWithETA contains supercharger information along with arrival time
type SuperchargerWithETA struct {
	Supercharger        *db.Supercharger            `json:"supercharger"`
	Restaurants         []db.RestaurantWithDistance `json:"restaurants"`
	ArrivalTime         string                      `json:"arrival_time"`           // Formatted arrival time
	DistanceFromRoute   float64                     `json:"distance_from_route"`    // Distance from route in meters
	DistanceAlongRoute  float64                     `json:"distance_along_route"`   // Distance along route in meters
	ClosestPointOnRoute Center                      `json:"closest_point_on_route"` // Closest point on the route
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

// PolylineIndex provides spatial indexing for fast distance calculations to polylines
type PolylineIndex struct {
	gridSize   float64 // degrees
	minLat     float64
	maxLat     float64
	minLng     float64
	maxLng     float64
	gridWidth  int
	gridHeight int
	grid       [][][]PolylineSegment // [y][x][]segments
	polyline   []Center
}

// PolylineSegment represents a segment in the polyline with its index and cumulative distance
type PolylineSegment struct {
	StartIdx       int
	EndIdx         int
	CumulativeDist float64
}

// buildPolylineIndex creates a spatial index for the given polyline
func buildPolylineIndex(polyline []Center, gridSize float64) *PolylineIndex {
	if len(polyline) < 2 {
		return nil
	}

	// Find bounds
	minLat, maxLat := polyline[0].Latitude, polyline[0].Latitude
	minLng, maxLng := polyline[0].Longitude, polyline[0].Longitude

	for _, p := range polyline {
		if p.Latitude < minLat {
			minLat = p.Latitude
		}
		if p.Latitude > maxLat {
			maxLat = p.Latitude
		}
		if p.Longitude < minLng {
			minLng = p.Longitude
		}
		if p.Longitude > maxLng {
			maxLng = p.Longitude
		}
	}

	// Add padding to bounds
	padding := gridSize
	minLat -= padding
	maxLat += padding
	minLng -= padding
	maxLng += padding

	// Calculate grid dimensions
	latRange := maxLat - minLat
	lngRange := maxLng - minLng
	gridWidth := int(math.Ceil(lngRange / gridSize))
	gridHeight := int(math.Ceil(latRange / gridSize))

	if gridWidth <= 0 {
		gridWidth = 1
	}
	if gridHeight <= 0 {
		gridHeight = 1
	}

	// Initialize grid
	grid := make([][][]PolylineSegment, gridHeight)
	for i := range grid {
		grid[i] = make([][]PolylineSegment, gridWidth)
	}

	// Build cumulative distances and assign segments to grid cells
	cumulativeDist := 0.0
	for i := 0; i < len(polyline)-1; i++ {
		p1 := polyline[i]
		p2 := polyline[i+1]

		// Create segment
		segment := PolylineSegment{
			StartIdx:       i,
			EndIdx:         i + 1,
			CumulativeDist: cumulativeDist,
		}

		// Find grid cells this segment intersects
		minSegLat := math.Min(p1.Latitude, p2.Latitude)
		maxSegLat := math.Max(p1.Latitude, p2.Latitude)
		minSegLng := math.Min(p1.Longitude, p2.Longitude)
		maxSegLng := math.Max(p1.Longitude, p2.Longitude)

		// Convert to grid coordinates
		startY := int((minSegLat - minLat) / gridSize)
		endY := int((maxSegLat - minLat) / gridSize)
		startX := int((minSegLng - minLng) / gridSize)
		endX := int((maxSegLng - minLng) / gridSize)

		// Clamp to grid bounds
		if startY < 0 {
			startY = 0
		}
		if endY >= gridHeight {
			endY = gridHeight - 1
		}
		if startX < 0 {
			startX = 0
		}
		if endX >= gridWidth {
			endX = gridWidth - 1
		}

		// Add segment to all intersecting grid cells
		for y := startY; y <= endY; y++ {
			for x := startX; x <= endX; x++ {
				grid[y][x] = append(grid[y][x], segment)
			}
		}

		// Update cumulative distance
		cumulativeDist += haversineDistance(p1, p2)
	}

	return &PolylineIndex{
		gridSize:   gridSize,
		minLat:     minLat,
		maxLat:     maxLat,
		minLng:     minLng,
		maxLng:     maxLng,
		gridWidth:  gridWidth,
		gridHeight: gridHeight,
		grid:       grid,
		polyline:   polyline,
	}
}

// distanceToPolylineWithIndex calculates distance using spatial index for better performance
func distanceToPolylineWithIndex(point Center, index *PolylineIndex) (float64, float64, Center) {
	if index == nil || len(index.polyline) < 2 {
		return distanceToPolyline(point, index.polyline)
	}

	// Find candidate segments in nearby grid cells
	var candidateSegments []PolylineSegment

	// Check point's grid cell and adjacent cells
	pointY := int((point.Latitude - index.minLat) / index.gridSize)
	pointX := int((point.Longitude - index.minLng) / index.gridSize)

	// Check 3x3 area around the point
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			y := pointY + dy
			x := pointX + dx

			if y >= 0 && y < index.gridHeight && x >= 0 && x < index.gridWidth {
				candidateSegments = append(candidateSegments, index.grid[y][x]...)
			}
		}
	}

	// If no candidates found (point outside bounds), check all segments
	if len(candidateSegments) == 0 {
		return distanceToPolyline(point, index.polyline)
	}

	// Remove duplicates (segments might be in multiple cells)
	seen := make(map[int]bool)
	var uniqueSegments []PolylineSegment
	for _, seg := range candidateSegments {
		if !seen[seg.StartIdx] {
			seen[seg.StartIdx] = true
			uniqueSegments = append(uniqueSegments, seg)
		}
	}

	// Calculate distances only for candidate segments
	minDist := math.MaxFloat64
	distAlongRoute := 0.0
	var closestPoint Center

	for _, segment := range uniqueSegments {
		p1 := index.polyline[segment.StartIdx]
		p2 := index.polyline[segment.EndIdx]
		dist := distanceToSegment(point, p1, p2)

		if dist < minDist {
			minDist = dist
			// Find where on the segment the closest point lies
			l2 := (p1.Latitude-p2.Latitude)*(p1.Latitude-p2.Latitude) + (p1.Longitude-p2.Longitude)*(p1.Longitude-p2.Longitude)
			if l2 == 0.0 {
				closestPoint = p1
				distAlongRoute = segment.CumulativeDist
			} else {
				t := ((point.Latitude-p1.Latitude)*(p2.Latitude-p1.Latitude) + (point.Longitude-p1.Longitude)*(p2.Longitude-p1.Longitude)) / l2
				t = math.Max(0, math.Min(1, t)) // Clamp to segment
				segmentLength := haversineDistance(p1, p2)
				distAlongRoute = segment.CumulativeDist + t*segmentLength
				closestPoint = Center{
					Latitude:  p1.Latitude + t*(p2.Latitude-p1.Latitude),
					Longitude: p1.Longitude + t*(p2.Longitude-p1.Longitude),
				}
			}
		}
	}

	return minDist, distAlongRoute, closestPoint
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

// calculateETA calculates the estimated arrival time at a supercharger
// based on route duration and distance from route
func calculateETA(cumulativePoints []CumPoint, distAlongRoute, distFromRoute float64, totalRouteDist float64, totalRouteDur time.Duration) time.Time {
	// Find the closest cumulative point for accurate ETA
	var selectedCumDur int
	var foundDuration bool
	if len(cumulativePoints) > 0 {
		for _, cp := range cumulativePoints {
			if cp.CumDistKm >= distAlongRoute/1000.0 { // Convert meters to km
				selectedCumDur = cp.CumDurSeconds
				foundDuration = true
				break
			}
		}
		if !foundDuration {
			selectedCumDur = cumulativePoints[len(cumulativePoints)-1].CumDurSeconds
		}
	} else {
		// No detailed cumulative points, estimate based on total route
		if totalRouteDist > 0 {
			fraction := distAlongRoute / totalRouteDist
			selectedCumDur = int(float64(totalRouteDur.Seconds()) * fraction)
		}
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

// processSuperchargers processes supercharger results concurrently to calculate ETAs and distances
func processSuperchargers(resultsChan <-chan superchargerResult, routePoints []Center, cumulativePoints []CumPoint, polylineIndex *PolylineIndex, route *RouteInfo) ([]SuperchargerWithETA, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var superchargersWithETA []SuperchargerWithETA
	errChan := make(chan error, 1)

	for res := range resultsChan {
		wg.Add(1)
		go func(res superchargerResult) {
			defer wg.Done()
			if res.err != nil {
				select {
				case errChan <- res.err:
				default:
				}
				return
			}

			// skip non-superchargers
			if !res.supercharger.IsSupercharger {
				return
			}

			sc := res.supercharger
			scLocation := Center{
				Latitude:  sc.Latitude,
				Longitude: sc.Longitude,
			}

			// Find closest point on route and calculate distances
			distFromRoute, distAlongRoute, closestPoint := distanceToPolylineWithIndex(scLocation, polylineIndex)

			arrivalTime := calculateETA(cumulativePoints, distAlongRoute, distFromRoute, float64(route.DistanceMeters), route.Duration)

			eta := SuperchargerWithETA{
				Supercharger:        sc,
				ArrivalTime:         arrivalTime.Format(time.Kitchen), // e.g., "3:45PM"
				DistanceFromRoute:   distFromRoute,
				DistanceAlongRoute:  distAlongRoute,
				ClosestPointOnRoute: closestPoint,
				Restaurants:         res.restaurants,
			}

			mu.Lock()
			superchargersWithETA = append(superchargersWithETA, eta)
			mu.Unlock()
		}(res)
	}

	wg.Wait()

	select {
	case err := <-errChan:
		return nil, err
	default:
		return superchargersWithETA, nil
	}
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

	// Build spatial index for fast distance calculations
	indexStart := time.Now()
	polylineIndex := buildPolylineIndex(routePoints, 0.01) // 0.01 degrees ≈ 1.11km grid size
	log.Printf("Build spatial index time: %v", time.Since(indexStart))

	// Build cumulative profile for accurate ETAs if we have enhanced route data
	cumulativeStart := time.Now()
	var cumulativePoints []CumPoint
	// Simplified: no detailed steps available, so cumulativePoints remains empty
	// ETA will be calculated based on total duration and distance from route
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
			superCharger, restaurants, err := GetSuperchargerWithCache(ctx, broker, apiKey, id)
			resultsChan <- superchargerResult{supercharger: superCharger, restaurants: restaurants, err: err}
		}(id)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	log.Printf("Fetch supercharger details time: %v", time.Since(fetchStart))

	// Process results and calculate ETAs
	processStart := time.Now()
	superchargersWithETA, err := processSuperchargers(resultsChan, routePoints, cumulativePoints, polylineIndex, route)
	if err != nil {
		return nil, err
	}
	log.Printf("process superchargers time: %v", time.Since(processStart))

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
func GetSuperchargerWithCache(ctx context.Context, broker *db.Service, apiKey, placeID string) (*db.Supercharger, []db.RestaurantWithDistance, error) {
	// First try to get from database
	supercharger, err := broker.Supercharger.GetByID(placeID)
	if err == nil {
		restaurants, err := broker.Supercharger.GetRestaurantsForSupercharger(placeID)
		return supercharger, restaurants, err
	}

	// Check if error is "not found" (expected when place doesn't exist in DB)
	if err != gorm.ErrRecordNotFound {
		return nil, nil, fmt.Errorf("failed to query supercharger from database: %w", err)
	}

	log.Println("Supercharger not found in DB, fetching from API:", placeID)

	// Not found in database, fetch from API
	// this field map ensure the essentials tier
	superchargerDetails, err := GetPlaceDetails(ctx, apiKey, placeID, FieldMaskSuperchargerDetails)
	if err != nil {
		return nil, nil, err
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
		return supercharger, []db.RestaurantWithDistance{}, nil
	}

	restaurants, err := GetPlacesViaTextSearch(ctx, apiKey, "restaurant", FieldMaskRestaurantTextSearch, Circle{
		Center: Center{
			Latitude:  superchargerDetails.Location.Latitude,
			Longitude: superchargerDetails.Location.Longitude,
		},
		Radius: 500, // 500 meter radius
	})
	if err != nil {
		return nil, nil, err
	}

	var dbRestaurants []db.RestaurantWithDistance
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
		dbRestaurants = append(dbRestaurants, db.RestaurantWithDistance{
			Restaurant: dbRestaurant,
			Distance:   dist,
		})
	}

	// Store in database for future use
	supercharger = &db.Supercharger{
		PlaceID:        superchargerDetails.ID,
		Name:           derefDisplayName(superchargerDetails.DisplayName),
		Address:        derefString(superchargerDetails.FormattedAddress),
		Latitude:       superchargerDetails.Location.Latitude,
		Longitude:      superchargerDetails.Location.Longitude,
		IsSupercharger: true,
	}

	err = broker.Supercharger.AddSuperchargerWithRestaurants(supercharger, dbRestaurants)
	if err != nil {
		// Log the error but don't fail the request since we already have the data
		fmt.Printf("Warning: failed to cache supercharger %s in database: %v\n", placeID, err)
	}

	return supercharger, dbRestaurants, nil
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
