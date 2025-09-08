package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// Global variable for the Google Maps API key.
var googleAPIKey = os.Getenv("GOOGLE_MAPS_API_KEY")

// APICallCounter tracks the number of API calls made during a request.
type APICallCounter struct {
	Directions    int
	Places        int
	PlacesDetails int // Counts calls to the new Places API or legacy Place Details
}

// --- Structs for our final API response ---

// RouteResponse is the main structure for the /route endpoint response.
type RouteResponse struct {
	Route         RouteDetails       `json:"route"`
	Superchargers []SuperchargerInfo `json:"superchargers"`
}

// RouteDetails contains information about the overall route.
type RouteDetails struct {
	TotalDistance string `json:"total_distance"`
	TotalDuration string `json:"total_duration"`
	// The encoded polyline can be used by mapping libraries (like Google Maps JS API) to draw the route.
	Polyline string `json:"polyline"`
}

// SuperchargerInfo contains details about a supercharger and nearby restaurants.
type SuperchargerInfo struct {
	Name                 string           `json:"name"`
	Address              string           `json:"address"`
	Distance             string           `json:"distance"`
	ArrivalTime          string           `json:"arrival_time"` // Estimated arrival time
	Lat                  float64          `json:"lat"`
	Lng                  float64          `json:"lng"`
	Restaurants          []RestaurantInfo `json:"restaurants"`
	DistanceFromOriginKm float64          `json:"-"` // Internal field for sorting
}

// RestaurantInfo contains details for a single restaurant.
type RestaurantInfo struct {
	Name            string   `json:"name"`
	Address         string   `json:"address"`
	Rating          float64  `json:"rating"`
	IsOpenNow       bool     `json:"is_open_now"`
	Lat             float64  `json:"lat"`
	Lng             float64  `json:"lng"`
	CuisineTypes    []string `json:"cuisine_types"`
	WalkingDistance string   `json:"walking_distance"`
}

// --- Structs for parsing Google Maps API responses ---

// GeoBounds defines a named struct for a geographic bounding box.
type GeoBounds struct {
	Southwest LatLng `json:"southwest"`
	Northeast LatLng `json:"northeast"`
}

// GoogleDirectionsResponse is for parsing the Directions API response.
type GoogleDirectionsResponse struct {
	Routes []struct {
		OverviewPolyline struct {
			Points string `json:"points"`
		} `json:"overview_polyline"`
		Bounds GeoBounds `json:"bounds"`
		Legs   []struct {
			Distance struct {
				Value int    `json:"value"` // in meters
				Text  string `json:"text"`
			} `json:"distance"`
			DurationInTraffic struct {
				Value int    `json:"value"` // in seconds, with traffic
				Text  string `json:"text"`
			} `json:"duration_in_traffic"`
			Duration struct {
				Value int    `json:"value"` // in seconds, without traffic
				Text  string `json:"text"`
			} `json:"duration"`
		} `json:"legs"`
	} `json:"routes"`
}

// (Legacy) GooglePlacesResponse is for parsing the Places API (Nearby Search) response for superchargers.
type GooglePlacesResponse struct {
	Results       []PlaceResult `json:"results"`
	NextPageToken string        `json:"next_page_token"`
	Status        string        `json:"status"`
}

// (Legacy) PlaceResult represents a single place from the Places API.
type PlaceResult struct {
	PlaceID  string   `json:"place_id"`
	Name     string   `json:"name"`
	Vicinity string   `json:"vicinity"`
	Rating   float64  `json:"rating"`
	Types    []string `json:"types"`
	Geometry struct {
		Location LatLng `json:"location"`
	} `json:"geometry"`
	OpeningHours struct {
		OpenNow bool `json:"open_now"`
	} `json:"opening_hours"`
}

// --- Structs for Places API (New) ---

// SearchNearbyRequest is the request body for the new Places API.
type SearchNearbyRequest struct {
	IncludedTypes       []string            `json:"includedTypes"`
	MaxResultCount      int                 `json:"maxResultCount"`
	LocationRestriction LocationRestriction `json:"locationRestriction"`
}
type LocationRestriction struct {
	Circle Circle `json:"circle"`
}
type Circle struct {
	Center Center  `json:"center"`
	Radius float64 `json:"radius"`
}
type Center struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// SearchNearbyResponse is the response from the new Places API.
type SearchNearbyResponse struct {
	Places []PlaceNew `json:"places"`
}

// PlaceNew represents a place object from the new Places API.
type PlaceNew struct {
	DisplayName         DisplayName         `json:"displayName"`
	FormattedAddress    string              `json:"formattedAddress"`
	Rating              float64             `json:"rating"`
	CurrentOpeningHours CurrentOpeningHours `json:"currentOpeningHours"`
	Location            LocationNew         `json:"location"`
	PrimaryType         string              `json:"primaryType"`
	Types               []string            `json:"types"`
}
type DisplayName struct {
	Text string `json:"text"`
}
type CurrentOpeningHours struct {
	OpenNow bool `json:"openNow"`
}
type LocationNew struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// LatLng represents a geographical point.
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func main() {
	// Check if the API key is set.
	if googleAPIKey == "" {
		googleAPIKey = "YOUR_GOOGLE_MAPS_API_KEY" // Fallback for local testing
		log.Println("WARNING: GOOGLE_MAPS_API_KEY environment variable not set. Using placeholder.")
	}
	if googleAPIKey == "YOUR_GOOGLE_MAPS_API_KEY" {
		log.Fatal("FATAL: Please replace 'YOUR_GOOGLE_MAPS_API_KEY' with your actual Google Maps API key.")
	}

	// Register handlers for the two endpoints.
	http.HandleFunc("/autocomplete", autocompleteHandler)
	http.HandleFunc("/route", routeHandler)

	// Start the server.
	port := "8080"
	log.Printf("Server starting on port %s...\n", port)
	log.Printf("Available endpoints: http://localhost:%s/autocomplete and http://localhost:%s/route", port, port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// autocompleteHandler handles place autocomplete requests.
func autocompleteHandler(w http.ResponseWriter, r *http.Request) {
	partial := r.URL.Query().Get("partial")
	if partial == "" {
		http.Error(w, "Query parameter 'partial' is required", http.StatusBadRequest)
		return
	}

	apiURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/place/autocomplete/json?input=%s&key=%s",
		url.QueryEscape(partial),
		googleAPIKey,
	)

	log.Printf("Calling Autocomplete API: %s", apiURL)
	resp, err := http.Get(apiURL)
	if err != nil {
		http.Error(w, "Failed to contact Google Maps API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response from Google Maps API", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// routeHandler implements the new, efficient strategy for finding superchargers and restaurants.
func routeHandler(w http.ResponseWriter, r *http.Request) {
	// Initialize API call counter for this request
	counter := &APICallCounter{}
	// Defer the logging of the final counts
	defer func() {
		log.Printf("API Call Summary: Directions=%d, Places (Nearby Legacy)=%d, Places (Nearby New)=%d",
			counter.Directions, counter.Places, counter.PlacesDetails) // PlacesDetails now counts Places (New) calls
	}()

	origin := r.URL.Query().Get("origin")
	destination := r.URL.Query().Get("destination")
	if origin == "" || destination == "" {
		http.Error(w, "Query parameters 'origin' and 'destination' are required", http.StatusBadRequest)
		return
	}

	// --- 1. Get Route Details (including polyline) ---
	directionsURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/directions/json?origin=%s&destination=%s&departure_time=now&traffic_model=best_guess&key=%s",
		url.QueryEscape(origin),
		url.QueryEscape(destination),
		googleAPIKey,
	)
	log.Printf("Calling Directions API for main route: %s", directionsURL)
	counter.Directions++ // Increment directions counter
	resp, err := http.Get(directionsURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Failed to get directions: %v", err), http.StatusInternalServerError)
		return
	}
	directionsBody, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	var directionsData GoogleDirectionsResponse
	if json.Unmarshal(directionsBody, &directionsData) != nil || len(directionsData.Routes) == 0 {
		http.Error(w, "Could not parse directions or find a route", http.StatusInternalServerError)
		return
	}

	route := directionsData.Routes[0]
	leg := route.Legs[0]
	decodedPolyline, _ := decodePolyline(route.OverviewPolyline.Points)
	if len(decodedPolyline) < 2 {
		http.Error(w, "Not enough points in polyline to process", http.StatusInternalServerError)
		return
	}

	// --- 2. Comprehensive Search: Find ALL Superchargers by searching at intervals along the route ---
	log.Println("Starting comprehensive search for superchargers along the route...")
	allSuperchargersMap := make(map[string]PlaceResult) // Use map for de-duplication by PlaceID

	const searchIntervalKm = 40.0    // How often to search along the route
	const searchRadiusMeters = 20000 // How far to search off the route (20km)

	var distanceSinceLastSearch float64 = 0

	for i := 1; i < len(decodedPolyline); i++ {
		p1 := decodedPolyline[i-1]
		p2 := decodedPolyline[i]
		segmentDistance := haversineDistance(p1.Lat, p1.Lng, p2.Lat, p2.Lng)
		distanceSinceLastSearch += segmentDistance

		if distanceSinceLastSearch >= searchIntervalKm || i == len(decodedPolyline)-1 {
			results, err := performLegacyNearbySearch(p2, searchRadiusMeters, "Tesla Supercharger", "electric_vehicle_charging_station", counter)
			if err != nil {
				log.Printf("Warning: search failed at point %d: %v", i, err)
			}
			for _, res := range results {
				allSuperchargersMap[res.PlaceID] = res
			}
			distanceSinceLastSearch = 0 // Reset counter
		}
	}

	var allSuperchargersInArea []PlaceResult
	for _, sc := range allSuperchargersMap {
		allSuperchargersInArea = append(allSuperchargersInArea, sc)
	}
	log.Printf("Found %d unique potential superchargers along the route.", len(allSuperchargersInArea))

	// --- 3. Filter Precisely: Prune superchargers to only those near the polyline ---
	log.Println("Filtering superchargers by proximity to route polyline...")
	var relevantSuperchargers []PlaceResult
	for _, sc := range allSuperchargersInArea {
		dist, _ := distanceToPolyline(sc.Geometry.Location, decodedPolyline)
		if dist <= 1.0 { // 1 km deviation allowance
			relevantSuperchargers = append(relevantSuperchargers, sc)
		}
	}
	log.Printf("Found %d superchargers within 1km of the route.", len(relevantSuperchargers))

	// --- 4. Narrow Search & Refined Timing: For each relevant supercharger, find restaurants and get accurate arrival time ---
	log.Println("Getting accurate arrival times and finding restaurants...")
	var finalSuperchargerList []SuperchargerInfo
	for _, sc := range relevantSuperchargers {
		restaurants, err := findNearbyRestaurantsNew(sc, counter)
		if err != nil {
			log.Printf("Warning: could not find restaurants for %s: %v", sc.Name, err)
		}

		// New: Get traffic-aware travel time to this specific supercharger
		var arrivalTime time.Time
		durationToSupercharger, err := getDurationToDestination(origin, sc.Geometry.Location, counter)
		if err != nil {
			log.Printf("Warning: could not get specific travel time for %s, using approximation. Error: %v", sc.Name, err)
			// Fallback to old method if specific route fails
			_, distAlongRouteFallback := distanceToPolyline(sc.Geometry.Location, decodedPolyline)
			arrivalRatio := distAlongRouteFallback / (float64(leg.Distance.Value) / 1000.0)
			durationToSupercharger = time.Duration(float64(leg.Duration.Value)*arrivalRatio) * time.Second
		}

		// Add a "smudge factor" for the final deviation from the route
		distFromRoute, distAlongRoute := distanceToPolyline(sc.Geometry.Location, decodedPolyline)
		smudgeFactorSeconds := (distFromRoute / 50.0) * 3600 // Assume 50 km/h average for final leg
		arrivalTime = time.Now().Add(durationToSupercharger + time.Duration(smudgeFactorSeconds)*time.Second)

		finalSuperchargerList = append(finalSuperchargerList, SuperchargerInfo{
			Name:                 sc.Name,
			Address:              sc.Vicinity,
			Distance:             fmt.Sprintf("%.1f km", distAlongRoute),
			ArrivalTime:          arrivalTime.Format(time.Kitchen),
			Lat:                  sc.Geometry.Location.Lat,
			Lng:                  sc.Geometry.Location.Lng,
			Restaurants:          restaurants,
			DistanceFromOriginKm: distAlongRoute,
		})
	}

	// Sort superchargers by their order along the route
	sort.Slice(finalSuperchargerList, func(i, j int) bool {
		return finalSuperchargerList[i].DistanceFromOriginKm < finalSuperchargerList[j].DistanceFromOriginKm
	})

	// --- 5. Assemble Final Response ---
	response := RouteResponse{
		Route: RouteDetails{
			TotalDistance: leg.Distance.Text,
			TotalDuration: leg.DurationInTraffic.Text, // Use duration_in_traffic for the main route
			Polyline:      route.OverviewPolyline.Points,
		},
		Superchargers: finalSuperchargerList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getDurationToDestination gets a traffic-aware travel time from an origin to a specific point.
func getDurationToDestination(origin string, destination LatLng, counter *APICallCounter) (time.Duration, error) {
	destStr := fmt.Sprintf("%f,%f", destination.Lat, destination.Lng)
	apiURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/directions/json?origin=%s&destination=%s&departure_time=now&traffic_model=best_guess&key=%s",
		url.QueryEscape(origin),
		url.QueryEscape(destStr),
		googleAPIKey,
	)
	log.Printf("Calling Directions API for arrival time: %s", apiURL)
	counter.Directions++ // Increment directions counter
	resp, err := http.Get(apiURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var directionsData GoogleDirectionsResponse
	if json.Unmarshal(body, &directionsData) != nil || len(directionsData.Routes) == 0 {
		return 0, fmt.Errorf("could not parse directions to supercharger")
	}

	// Use duration_in_traffic if available, otherwise fall back to normal duration
	durationSeconds := directionsData.Routes[0].Legs[0].Duration.Value
	if directionsData.Routes[0].Legs[0].DurationInTraffic.Value > 0 {
		durationSeconds = directionsData.Routes[0].Legs[0].DurationInTraffic.Value
	}

	return time.Duration(durationSeconds) * time.Second, nil
}

// performLegacyNearbySearch executes a paginated nearby search using the older API.
func performLegacyNearbySearch(location LatLng, radiusMeters int, keyword, placeType string, counter *APICallCounter) ([]PlaceResult, error) {
	var allResults []PlaceResult
	nextPageToken := ""

	for {
		var apiURL string
		if nextPageToken == "" {
			apiURL = fmt.Sprintf(
				"https://maps.googleapis.com/maps/api/place/nearbysearch/json?location=%f,%f&radius=%d&keyword=%s&type=%s&key=%s",
				location.Lat, location.Lng, radiusMeters, url.QueryEscape(keyword), url.QueryEscape(placeType), googleAPIKey,
			)
			log.Printf("Calling Legacy Places Nearby Search API: %s", apiURL)
		} else {
			time.Sleep(2 * time.Second) // Required delay for next page token
			apiURL = fmt.Sprintf("https://maps.googleapis.com/maps/api/place/nearbysearch/json?pagetoken=%s&key=%s", nextPageToken, googleAPIKey)
			log.Printf("Calling Legacy Places Nearby Search API (next page): %s", apiURL)
		}
		counter.Places++ // Increment places counter for each page
		resp, err := http.Get(apiURL)
		if err != nil {
			return nil, err
		}

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var placesData GooglePlacesResponse
		if json.Unmarshal(body, &placesData) != nil || placesData.Status == "INVALID_REQUEST" {
			log.Printf("Warning: Invalid Places API response. Body: %s", string(body))
			break
		}

		allResults = append(allResults, placesData.Results...)

		if placesData.NextPageToken == "" {
			break
		}
		nextPageToken = placesData.NextPageToken
	}
	return allResults, nil
}

// findNearbyRestaurantsNew finds restaurants using the Places API (New).
func findNearbyRestaurantsNew(supercharger PlaceResult, counter *APICallCounter) ([]RestaurantInfo, error) {
	var allRestaurants []RestaurantInfo
	superchargerLoc := supercharger.Geometry.Location

	requestBody := SearchNearbyRequest{
		IncludedTypes:  []string{"restaurant"},
		MaxResultCount: 20,
		LocationRestriction: LocationRestriction{
			Circle: Circle{
				Center: Center{
					Latitude:  superchargerLoc.Lat,
					Longitude: superchargerLoc.Lng,
				},
				Radius: 500.0,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	apiURL := "https://places.googleapis.com/v1/places:searchNearby"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", googleAPIKey)
	req.Header.Set("X-Goog-FieldMask", "places.displayName,places.formattedAddress,places.rating,places.currentOpeningHours,places.location,places.primaryType,places.types")

	log.Printf("Calling Places API (New) for restaurants near %s: %s", supercharger.Name, apiURL)
	counter.PlacesDetails++ // Using this counter for the new API calls

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var searchData SearchNearbyResponse
	if json.Unmarshal(body, &searchData) != nil {
		return nil, fmt.Errorf("could not parse new places response. Body: %s", string(body))
	}

	for _, p := range searchData.Places {
		walkingDistKm := haversineDistance(superchargerLoc.Lat, superchargerLoc.Lng, p.Location.Latitude, p.Location.Longitude)
		walkingDistStr := fmt.Sprintf("%.0f m", walkingDistKm*1000)

		allRestaurants = append(allRestaurants, RestaurantInfo{
			Name:            p.DisplayName.Text,
			Address:         p.FormattedAddress,
			Rating:          p.Rating,
			IsOpenNow:       p.CurrentOpeningHours.OpenNow,
			Lat:             p.Location.Latitude,
			Lng:             p.Location.Longitude,
			CuisineTypes:    extractCuisineFromNewPlace(p),
			WalkingDistance: walkingDistStr,
		})
	}

	return allRestaurants, nil
}

// extractCuisineFromNewPlace extracts cuisine types from a PlaceNew object, prioritizing primary_type.
func extractCuisineFromNewPlace(place PlaceNew) []string {
	genericTypes := map[string]bool{
		"restaurant": true, "food": true, "point_of_interest": true, "establishment": true,
	}

	// 1. Prioritize primary_type if it exists and is not generic
	if place.PrimaryType != "" && !genericTypes[place.PrimaryType] {
		formattedType := strings.ReplaceAll(place.PrimaryType, "_", " ")
		return []string{strings.Title(formattedType)}
	}

	// 2. Fallback to filtering the generic 'types' array if primary_type is not useful.
	var filteredTypes []string
	for _, placeType := range place.Types {
		if !genericTypes[placeType] {
			filteredTypes = append(filteredTypes, strings.Title(strings.ReplaceAll(placeType, "_", " ")))
		}
	}

	if len(filteredTypes) > 0 {
		return filteredTypes
	}

	// 3. If all else fails, return the primary type even if it's generic.
	if place.PrimaryType != "" {
		return []string{strings.Title(strings.ReplaceAll(place.PrimaryType, "_", " "))}
	}

	return []string{} // Return empty if no suitable type found
}

// --- GEOMETRIC HELPER FUNCTIONS ---

// distanceToPolyline calculates the shortest distance from a point to a polyline.
// It returns the shortest distance in km and the cumulative distance along the polyline to that closest point.
func distanceToPolyline(point LatLng, polyline []LatLng) (float64, float64) {
	minDist := math.MaxFloat64
	distAlongRoute := 0.0
	cumulativeDist := 0.0

	for i := 0; i < len(polyline)-1; i++ {
		p1 := polyline[i]
		p2 := polyline[i+1]
		dist := distanceToSegment(point, p1, p2)

		if dist < minDist {
			minDist = dist
			// Find where on the segment the closest point lies
			l2 := (p1.Lat-p2.Lat)*(p1.Lat-p2.Lat) + (p1.Lng-p2.Lng)*(p1.Lng-p2.Lng)
			if l2 == 0.0 {
				distAlongRoute = cumulativeDist
			} else {
				t := ((point.Lat-p1.Lat)*(p2.Lat-p1.Lat) + (point.Lng-p1.Lng)*(p2.Lng-p1.Lng)) / l2
				t = math.Max(0, math.Min(1, t)) // Clamp to segment
				segmentLength := haversineDistance(p1.Lat, p1.Lng, p2.Lat, p2.Lng)
				distAlongRoute = cumulativeDist + t*segmentLength
			}
		}
		cumulativeDist += haversineDistance(p1.Lat, p1.Lng, p2.Lat, p2.Lng)
	}
	return minDist, distAlongRoute
}

// distanceToSegment calculates the shortest distance from a point to a line segment.
func distanceToSegment(p, v, w LatLng) float64 {
	l2 := (v.Lat-w.Lat)*(v.Lat-w.Lat) + (v.Lng-w.Lng)*(v.Lng-w.Lng)
	if l2 == 0.0 {
		return haversineDistance(p.Lat, p.Lng, v.Lat, v.Lng)
	}
	t := ((p.Lat-v.Lat)*(w.Lat-v.Lat) + (p.Lng-v.Lng)*(w.Lng-v.Lng)) / l2
	t = math.Max(0, math.Min(1, t))

	closestLat := v.Lat + t*(w.Lat-v.Lat)
	closestLng := v.Lng + t*(w.Lng-v.Lng)
	return haversineDistance(p.Lat, p.Lng, closestLat, closestLng)
}

// haversineDistance calculates the distance between two lat/lng points in kilometers.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in kilometers
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// decodePolyline decodes a Google Maps polyline string into a slice of LatLng points.
func decodePolyline(encoded string) ([]LatLng, error) {
	var points []LatLng
	index, lat, lng := 0, 0, 0

	for index < len(encoded) {
		var result int
		var shift uint
		for {
			b := int(encoded[index]) - 63
			index++
			result |= (b & 0x1f) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}
		dlat := (result >> 1) ^ -(result & 1)
		lat += dlat

		result = 0
		shift = 0
		for {
			b := int(encoded[index]) - 63
			index++
			result |= (b & 0x1f) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}
		dlng := (result >> 1) ^ -(result & 1)
		lng += dlng

		points = append(points, LatLng{
			Lat: float64(lat) / 1e5,
			Lng: float64(lng) / 1e5,
		})
	}
	return points, nil
}
