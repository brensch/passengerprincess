package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Global variable for the Google Maps API key.
var googleAPIKey = os.Getenv("GOOGLE_MAPS_API_KEY")

// APICallCounter tracks the number of API calls made during a request.
type APICallCounter struct {
	Directions int
	Places     int // Counts calls to the new Places API
	mu         sync.Mutex
}

// --- Structs for our final API response ---

// RouteResponse is the main structure for the /route endpoint response.
type RouteResponse struct {
	Route           RouteDetails       `json:"route"`
	Superchargers   []SuperchargerInfo `json:"superchargers"`
	DebugInfo       DebugInfo          `json:"debug_info"`
	TrafficSegments []TrafficSegment   `json:"traffic_segments,omitempty"`
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
	Name                    string           `json:"name"`
	Address                 string           `json:"address"`
	DistanceMeters          int              `json:"distance_meters"`            // Distance along route in meters
	DistanceFromRouteMeters int              `json:"distance_from_route_meters"` // Distance from route in meters
	ArrivalTime             string           `json:"arrival_time"`               // Estimated arrival time
	Lat                     float64          `json:"lat"`
	Lng                     float64          `json:"lng"`
	ClosestPointOnRoute     LatLng           `json:"closest_point_on_route"` // Closest point on the route
	Restaurants             []RestaurantInfo `json:"restaurants"`
	DistanceFromOriginKm    float64          `json:"-"` // Internal field for sorting
}

// RestaurantInfo contains details for a single restaurant.
type RestaurantInfo struct {
	Name                  string   `json:"name"`
	Address               string   `json:"address"`
	Rating                float64  `json:"rating"`
	IsOpenNow             bool     `json:"is_open_now"`
	Lat                   float64  `json:"lat"`
	Lng                   float64  `json:"lng"`
	CuisineTypes          []string `json:"cuisine_types"`
	WalkingDistanceMeters int      `json:"walking_distance_meters"`
}

// CumPoint represents a point on the route with cumulative distance and duration.
type CumPoint struct {
	Lat           float64
	Lng           float64
	CumDistKm     float64
	CumDurSeconds int
}

// TrafficSegment contains a polyline for a part of the route with a specific traffic speed.
type TrafficSegment struct {
	Polyline string `json:"polyline"`
	Speed    string `json:"speed"`
}

// --- Structs for Debugging ---
type DebugInfo struct {
	APICalls []APICallDetails `json:"api_calls"`
}

type APICallDetails struct {
	API         string      `json:"api"`
	URL         string      `json:"url"`
	RequestBody interface{} `json:"request_body,omitempty"`
	SearchArea  interface{} `json:"search_area,omitempty"` // For nearby searches
}

type SearchAreaDetails struct {
	CenterLat float64 `json:"center_lat"`
	CenterLng float64 `json:"center_lng"`
	RadiusM   float64 `json:"radius_m"`
}

// --- Structs for Routes API (Traffic-aware) ---

// RoutesRequest is the request body for the Routes API.
type RoutesRequest struct {
	Origin            Location `json:"origin"`
	Destination       Location `json:"destination"`
	TravelMode        string   `json:"travelMode"`
	RoutingPreference string   `json:"routingPreference,omitempty"`
	ExtraComputations []string `json:"extraComputations,omitempty"`
	PolylineQuality   string   `json:"polylineQuality,omitempty"`
	PolylineEncoding  string   `json:"polylineEncoding,omitempty"`
	DepartureTime     string   `json:"departureTime,omitempty"`
}

type Location struct {
	Address string `json:"address,omitempty"`
}

// RoutesResponse is the response from the Routes API.
type RoutesResponse struct {
	Routes []Route `json:"routes"`
}

type Route struct {
	Polyline       EncodedPolyline     `json:"polyline"`
	Legs           []RouteLeg          `json:"legs"`
	TravelAdvisory RouteTravelAdvisory `json:"travelAdvisory,omitempty"`
	Duration       string              `json:"duration"`
	DistanceMeters int                 `json:"distanceMeters"`
}

type RouteLeg struct {
	Polyline       EncodedPolyline        `json:"polyline"`
	Steps          []RouteStep            `json:"steps"`
	TravelAdvisory RouteLegTravelAdvisory `json:"travelAdvisory,omitempty"`
	Duration       string                 `json:"duration"`
	DistanceMeters int                    `json:"distanceMeters"`
}

type RouteStep struct {
	Polyline       EncodedPolyline `json:"polyline"`
	StaticDuration string          `json:"staticDuration"`
	DistanceMeters int             `json:"distanceMeters"`
}

type EncodedPolyline struct {
	EncodedPolyline string `json:"encodedPolyline"`
}

type RouteTravelAdvisory struct {
	SpeedReadingIntervals []SpeedReadingInterval `json:"speedReadingIntervals,omitempty"`
}

type RouteLegTravelAdvisory struct {
	SpeedReadingIntervals []SpeedReadingInterval `json:"speedReadingIntervals,omitempty"`
}

type SpeedReadingInterval struct {
	StartPolylinePointIndex int    `json:"startPolylinePointIndex,omitempty"`
	EndPolylinePointIndex   int    `json:"endPolylinePointIndex"`
	Speed                   string `json:"speed"`
}

// LatLng represents a geographical point.
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// --- Structs for Places API (New) ---

// SearchNearbyRequest is the request body for the new Places API.
type SearchNearbyRequest struct {
	IncludedTypes       []string            `json:"includedTypes,omitempty"`
	TextQuery           string              `json:"textQuery,omitempty"`
	MaxResultCount      int                 `json:"maxResultCount,omitempty"`
	LocationRestriction LocationRestriction `json:"locationRestriction"`
}

// SearchTextRequest is the request body for the Places Text Search API.
type SearchTextRequest struct {
	TextQuery      string       `json:"textQuery"`
	IncludedType   string       `json:"includedType,omitempty"`
	MaxResultCount int          `json:"maxResultCount,omitempty"`
	LocationBias   LocationBias `json:"locationBias"`
}

type LocationBias struct {
	Circle Circle `json:"circle"`
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
	ID                  string              `json:"id"`
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

func main() {
	// Check if the API key is set.
	if googleAPIKey == "" {
		googleAPIKey = "YOUR_GOOGLE_MAPS_API_KEY" // Fallback for local testing
		log.Println("WARNING: GOOGLE_MAPS_API_KEY environment variable not set. Using placeholder.")
	}
	if googleAPIKey == "YOUR_GOOGLE_MAPS_API_KEY" {
		log.Fatal("FATAL: Please replace 'YOUR_GOOGLE_MAPS_API_KEY' with your actual Google Maps API key.")
	}

	// Register handlers.
	http.HandleFunc("/", serveFrontend) // Serve the HTML file at the root
	http.HandleFunc("/autocomplete", autocompleteHandler)
	http.HandleFunc("/route", routeHandler)

	// Start the server.
	port := "8080"
	log.Printf("Server starting...")
	log.Printf("Access the web interface at http://localhost:%s/", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// serveFrontend serves the index.html file.
func serveFrontend(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("frontend/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		APIKey string
	}{
		APIKey: googleAPIKey,
	}
	tmpl.Execute(w, data)
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
	// Initialize API call counter and details slice for this request
	counter := &APICallCounter{}
	var apiCalls []APICallDetails

	// Defer the logging of the final counts
	defer func() {
		log.Printf("API Call Summary: Directions=%d, Places (New)=%d",
			counter.Directions, counter.Places)
	}()

	origin := r.URL.Query().Get("origin")
	destination := r.URL.Query().Get("destination")
	if origin == "" || destination == "" {
		http.Error(w, "Query parameters 'origin' and 'destination' are required", http.StatusBadRequest)
		return
	}

	// --- 1. Get Route Details (including traffic-aware polyline) ---
	routesData, err := getRoutesData(origin, destination, counter, &apiCalls)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get routes: %v", err), http.StatusInternalServerError)
		return
	}

	if routesData == nil || len(routesData.Routes) == 0 || len(routesData.Routes[0].Legs) == 0 {
		http.Error(w, "Could not find a route", http.StatusInternalServerError)
		return
	}

	route := routesData.Routes[0]
	leg := route.Legs[0]

	// Decode the main polyline for the overview
	decodedPolyline, err := decodePolyline(route.Polyline.EncodedPolyline)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode polyline: %v", err), http.StatusInternalServerError)
		return
	}
	if len(decodedPolyline) < 2 {
		http.Error(w, "Not enough points in polyline to process", http.StatusInternalServerError)
		return
	}

	// --- Create Traffic-Aware Segments from Speed Reading Intervals ---
	var trafficSegments []TrafficSegment
	if len(route.TravelAdvisory.SpeedReadingIntervals) > 0 {
		log.Printf("Processing %d speed reading intervals into traffic segments.", len(route.TravelAdvisory.SpeedReadingIntervals))
		intervals := route.TravelAdvisory.SpeedReadingIntervals
		for _, interval := range intervals {
			startIdx := interval.StartPolylinePointIndex
			endIdx := interval.EndPolylinePointIndex

			// Ensure indices are within the bounds of the decoded polyline
			if startIdx >= 0 && endIdx < len(decodedPolyline) && startIdx <= endIdx {
				// The end index is inclusive, so the slice is [startIdx:endIdx+1]
				subPath := decodedPolyline[startIdx : endIdx+1]
				if len(subPath) > 0 {
					subPolyline := encodePolyline(subPath)
					trafficSegments = append(trafficSegments, TrafficSegment{
						Polyline: subPolyline,
						Speed:    interval.Speed,
					})
				}
			} else {
				log.Printf("Warning: Speed reading interval indices [%d, %d] are out of bounds for polyline length %d.", startIdx, endIdx, len(decodedPolyline))
			}
		}
	} else {
		log.Println("No speed reading intervals found. Creating a single traffic segment for the whole route.")
		// Fallback: create one segment for the whole route with "NORMAL" speed
		trafficSegments = append(trafficSegments, TrafficSegment{
			Polyline: route.Polyline.EncodedPolyline,
			Speed:    "NORMAL",
		})
	}

	// Build cumulative profile for accurate ETAs
	var cumulativePoints []CumPoint
	cumDist := 0.0
	cumDur := 0

	// Calculate total duration with and without traffic to find a traffic multiplier.
	// This ensures that the time to each point along the route accounts for traffic.
	sumStaticDur := 0
	for _, step := range leg.Steps {
		sumStaticDur += parseDurationString(step.StaticDuration)
	}
	totalTrafficDur := parseDurationString(leg.Duration)

	trafficMultiplier := 1.0
	if sumStaticDur > 0 {
		trafficMultiplier = float64(totalTrafficDur) / float64(sumStaticDur)
	}

	// Build the point-by-point timeline for the route.
	for _, step := range leg.Steps {
		stepPoints, err := decodePolyline(step.Polyline.EncodedPolyline)
		if err != nil || len(stepPoints) == 0 {
			continue
		}
		totalStepDist := 0.0
		for i := 1; i < len(stepPoints); i++ {
			totalStepDist += haversineDistance(stepPoints[i-1].Lat, stepPoints[i-1].Lng, stepPoints[i].Lat, stepPoints[i].Lng)
		}

		// Apply the traffic multiplier to this step's static duration
		staticStepDur := parseDurationString(step.StaticDuration)
		trafficStepDur := int(float64(staticStepDur) * trafficMultiplier)

		stepCumDist := 0.0
		for i, p := range stepPoints {
			if i > 0 {
				dist := haversineDistance(stepPoints[i-1].Lat, stepPoints[i-1].Lng, p.Lat, p.Lng)
				stepCumDist += dist
				fraction := 0.0
				if totalStepDist > 0 {
					fraction = stepCumDist / totalStepDist
				}
				pointCumDur := cumDur + int(float64(trafficStepDur)*fraction)
				cumulativePoints = append(cumulativePoints, CumPoint{
					Lat:           p.Lat,
					Lng:           p.Lng,
					CumDistKm:     cumDist + stepCumDist,
					CumDurSeconds: pointCumDur,
				})
			} else { // First point of the step
				cumulativePoints = append(cumulativePoints, CumPoint{
					Lat:           p.Lat,
					Lng:           p.Lng,
					CumDistKm:     cumDist,
					CumDurSeconds: cumDur,
				})
			}
		}
		cumDist += totalStepDist
		cumDur += trafficStepDur
	}

	// --- 2. Comprehensive Search: Find ALL Superchargers by searching at intervals along the route ---
	log.Println("Starting comprehensive search for superchargers along the route...")

	const searchIntervalKm = 40.0    // How often to search along the route
	const searchRadiusMeters = 30000 // How far to search off the route (30km)

	// Collect all search points first
	var searchPoints []struct {
		center    LatLng
		pointDesc string
	}

	var distanceSinceLastSearch float64 = 0

	// Start loop from the beginning, including the starting point
	for i := 0; i < len(decodedPolyline); i++ {
		var currentPoint LatLng
		if i == 0 {
			// First point - search immediately
			currentPoint = decodedPolyline[0]
		} else {
			// Subsequent points - accumulate distance
			p1 := decodedPolyline[i-1]
			p2 := decodedPolyline[i]
			segmentDistance := haversineDistance(p1.Lat, p1.Lng, p2.Lat, p2.Lng)
			distanceSinceLastSearch += segmentDistance
			currentPoint = p2
		}

		// Collect search points if it's the first point, we've reached the interval, or it's the last point
		if i == 0 || distanceSinceLastSearch >= searchIntervalKm || i == len(decodedPolyline)-1 {
			pointDesc := fmt.Sprintf("point %d (%.6f, %.6f)", i, currentPoint.Lat, currentPoint.Lng)
			if i == 0 {
				pointDesc = fmt.Sprintf("starting point (%.6f, %.6f)", currentPoint.Lat, currentPoint.Lng)
			}
			searchPoints = append(searchPoints, struct {
				center    LatLng
				pointDesc string
			}{currentPoint, pointDesc})
			if i > 0 {
				distanceSinceLastSearch = 0 // Reset counter only for non-starting points
			}
		}
	}

	// Perform concurrent searches
	log.Printf("Performing concurrent searches at %d points along the route...", len(searchPoints))

	type searchResult struct {
		results []PlaceNew
		err     error
		desc    string
	}

	resultsChan := make(chan searchResult, len(searchPoints))
	var wg sync.WaitGroup

	for _, point := range searchPoints {
		wg.Add(1)
		go func(center LatLng, pointDesc string) {
			defer wg.Done()

			requestBody := SearchTextRequest{
				TextQuery:      "Tesla Supercharger",
				IncludedType:   "electric_vehicle_charging_station",
				MaxResultCount: 20, // Request maximum results
				LocationBias: LocationBias{
					Circle: Circle{
						Center: Center{Latitude: center.Lat, Longitude: center.Lng},
						Radius: float64(searchRadiusMeters),
					},
				},
			}
			fieldMask := "places.id,places.displayName,places.formattedAddress,places.location,places.types,places.primaryType,places.rating"
			results, err := performTextSearch(requestBody, fieldMask, counter, &apiCalls)

			resultsChan <- searchResult{
				results: results,
				err:     err,
				desc:    pointDesc,
			}
		}(point.center, point.pointDesc)
	}

	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	allSuperchargers := []PlaceNew{}
	for result := range resultsChan {
		if result.err != nil {
			log.Printf("Warning: search failed at %s: %v", result.desc, result.err)
		} else {
			log.Printf("Search at %s returned %d total results", result.desc, len(result.results))
			// Filter results to only include superchargers
			var filteredResults []PlaceNew
			for _, sc := range result.results {
				if strings.Contains(strings.ToLower(sc.DisplayName.Text), "supercharger") {
					filteredResults = append(filteredResults, sc)
				}
			}
			log.Printf("After filtering, %d superchargers remain", len(filteredResults))
			allSuperchargers = append(allSuperchargers, filteredResults...)
		}
	}

	// Remove duplicates by ID
	seen := make(map[string]bool)
	var uniqueSuperchargers []PlaceNew
	for _, sc := range allSuperchargers {
		if !seen[sc.ID] {
			seen[sc.ID] = true
			uniqueSuperchargers = append(uniqueSuperchargers, sc)
		}
	}

	log.Printf("Found %d unique superchargers from interval searches.", len(uniqueSuperchargers))

	// --- 3. Filter superchargers within 10km of route and collect with distances ---
	log.Println("Filtering superchargers within 10km of route...")
	type superchargerWithDist struct {
		sc             PlaceNew
		distFromRoute  float64
		distAlongRoute float64
		closestPoint   LatLng
		totalDistKm    float64
	}
	var finalSuperchargers []superchargerWithDist
	totalRouteDistanceKm := float64(leg.DistanceMeters) / 1000.0

	for _, sc := range uniqueSuperchargers {
		scLoc := LatLng{Lat: sc.Location.Latitude, Lng: sc.Location.Longitude}
		distFromRoute, distAlongRoute, closestPoint := distanceToPolyline(scLoc, decodedPolyline)
		if distFromRoute > 10.0 { // Filter out superchargers further than 10km from route
			continue
		}
		totalDistKm := distAlongRoute + distFromRoute
		if totalDistKm > totalRouteDistanceKm {
			continue // Beyond destination
		}
		finalSuperchargers = append(finalSuperchargers, superchargerWithDist{
			sc:             sc,
			distFromRoute:  distFromRoute,
			distAlongRoute: distAlongRoute,
			closestPoint:   closestPoint,
			totalDistKm:    totalDistKm,
		})
		log.Printf("Included Supercharger: %s, distAlong: %.1f km, distFrom: %.1f km, total: %.1f km", sc.DisplayName.Text, distAlongRoute, distFromRoute, totalDistKm)
	}

	log.Printf("Found %d superchargers within 10km of the route.", len(finalSuperchargers))

	// --- 3.5. Select 10 closest superchargers per segment ---
	log.Println("Selecting 10 closest superchargers per 40km segment...")
	const segmentSizeKm = 40.0
	segments := make(map[int][]superchargerWithDist)
	for _, sc := range finalSuperchargers {
		segmentIndex := int(sc.distAlongRoute / segmentSizeKm)
		segments[segmentIndex] = append(segments[segmentIndex], sc)
	}

	var selectedSuperchargers []superchargerWithDist
	for segmentIndex, segSCs := range segments {
		// Sort by distance from route (closest first)
		sort.Slice(segSCs, func(i, j int) bool {
			return segSCs[i].distFromRoute < segSCs[j].distFromRoute
		})
		// Take up to 10 closest
		numToTake := 10
		if len(segSCs) < 10 {
			numToTake = len(segSCs)
		}
		selectedSuperchargers = append(selectedSuperchargers, segSCs[:numToTake]...)
		log.Printf("Segment %d: selected %d superchargers", segmentIndex, numToTake)
	}

	log.Printf("Selected %d superchargers (10 closest per segment).", len(selectedSuperchargers))

	// --- 4. Process superchargers and calculate ETAs ---
	log.Println("Processing superchargers and calculating ETAs...")

	// Prepare supercharger processing data
	type superchargerData struct {
		sc             PlaceNew
		distFromRoute  float64
		distAlongRoute float64
		closestPoint   LatLng
		totalDistKm    float64
		selectedCumDur int
	}

	var processingData []superchargerData

	for _, sc := range selectedSuperchargers {
		// Find the closest cumulative point for accurate ETA
		var selectedCumDur int
		var foundDuration bool
		for _, cp := range cumulativePoints {
			if cp.CumDistKm >= sc.distAlongRoute {
				selectedCumDur = cp.CumDurSeconds
				foundDuration = true
				break
			}
		}
		if !foundDuration && len(cumulativePoints) > 0 {
			selectedCumDur = cumulativePoints[len(cumulativePoints)-1].CumDurSeconds
		}

		processingData = append(processingData, superchargerData{
			sc:             sc.sc,
			distFromRoute:  sc.distFromRoute,
			distAlongRoute: sc.distAlongRoute,
			closestPoint:   sc.closestPoint,
			totalDistKm:    sc.totalDistKm,
			selectedCumDur: selectedCumDur,
		})
	}

	// Perform concurrent restaurant searches
	log.Printf("Performing concurrent restaurant searches for %d superchargers...", len(processingData))

	type restaurantResult struct {
		restaurants []RestaurantInfo
		err         error
		index       int
	}

	restaurantChan := make(chan restaurantResult, len(processingData))
	var restaurantWg sync.WaitGroup

	for i, data := range processingData {
		restaurantWg.Add(1)
		go func(sc PlaceNew, idx int) {
			defer restaurantWg.Done()

			restaurants, err := findNearbyRestaurantsNew(sc, counter, &apiCalls)
			if err != nil {
				log.Printf("Warning: could not find restaurants for %s: %v", sc.DisplayName.Text, err)
				restaurants = []RestaurantInfo{} // Set to empty slice to avoid null in JSON
			}

			restaurantChan <- restaurantResult{
				restaurants: restaurants,
				err:         err,
				index:       idx,
			}
		}(data.sc, i)
	}

	// Close channel when all goroutines are done
	go func() {
		restaurantWg.Wait()
		close(restaurantChan)
	}()

	// Collect restaurant results
	restaurantResults := make([][]RestaurantInfo, len(processingData))
	for result := range restaurantChan {
		restaurantResults[result.index] = result.restaurants
	}

	// Build final supercharger list
	finalSuperchargerList := make([]SuperchargerInfo, 0)
	for i, data := range processingData {
		durationToSupercharger := time.Duration(data.selectedCumDur) * time.Second
		arrivalTime := time.Now().Add(durationToSupercharger)

		// Add time to travel from route to supercharger at 50 km/h
		extraTimeHours := data.distFromRoute / 50.0
		extraTimeSeconds := extraTimeHours * 3600
		arrivalTime = arrivalTime.Add(time.Duration(extraTimeSeconds) * time.Second)

		finalSuperchargerList = append(finalSuperchargerList, SuperchargerInfo{
			Name:                    data.sc.DisplayName.Text,
			Address:                 data.sc.FormattedAddress,
			DistanceMeters:          int(data.totalDistKm * 1000),
			DistanceFromRouteMeters: int(data.distFromRoute * 1000),
			ArrivalTime:             arrivalTime.Format(time.Kitchen),
			Lat:                     data.sc.Location.Latitude,
			Lng:                     data.sc.Location.Longitude,
			ClosestPointOnRoute:     data.closestPoint,
			Restaurants:             restaurantResults[i],
			DistanceFromOriginKm:    data.totalDistKm,
		})
	}

	log.Printf("Processed %d superchargers.", len(finalSuperchargerList))

	// Sort superchargers by their total distance
	sort.Slice(finalSuperchargerList, func(i, j int) bool {
		return finalSuperchargerList[i].DistanceFromOriginKm < finalSuperchargerList[j].DistanceFromOriginKm
	})

	// --- 5. Assemble Final Response ---
	response := RouteResponse{
		Route: RouteDetails{
			TotalDistance: fmt.Sprintf("%.1f km", float64(leg.DistanceMeters)/1000.0),
			TotalDuration: fmt.Sprintf("%d mins", parseDurationString(leg.Duration)/60),
			Polyline:      route.Polyline.EncodedPolyline,
		},
		Superchargers: finalSuperchargerList,
		DebugInfo: DebugInfo{
			APICalls: apiCalls,
		},
		TrafficSegments: trafficSegments,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// performNewNearbySearch executes a search using the Places API (New).
func performNewNearbySearch(requestBody SearchNearbyRequest, fieldMask string, counter *APICallCounter, apiCalls *[]APICallDetails) ([]PlaceNew, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	apiURL := "https://places.googleapis.com/v1/places:searchNearby"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", googleAPIKey)
	req.Header.Set("X-Goog-FieldMask", fieldMask)

	log.Printf("Calling Places API (New): %s with body %s", apiURL, string(jsonData))

	// Thread-safe counter increment and API call logging
	counter.mu.Lock()
	counter.Places++
	*apiCalls = append(*apiCalls, APICallDetails{
		API:         "Places API (New)",
		URL:         apiURL,
		RequestBody: requestBody,
		SearchArea: SearchAreaDetails{
			CenterLat: requestBody.LocationRestriction.Circle.Center.Latitude,
			CenterLng: requestBody.LocationRestriction.Circle.Center.Longitude,
			RadiusM:   requestBody.LocationRestriction.Circle.Radius,
		},
	})
	counter.mu.Unlock()

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
	if err := json.Unmarshal(body, &searchData); err != nil {
		return nil, fmt.Errorf("could not parse new places response. Body: %s", string(body))
	}

	return searchData.Places, nil
}

// performTextSearch executes a text search using the Places API.
func performTextSearch(requestBody SearchTextRequest, fieldMask string, counter *APICallCounter, apiCalls *[]APICallDetails) ([]PlaceNew, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	apiURL := "https://places.googleapis.com/v1/places:searchText"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", googleAPIKey)
	req.Header.Set("X-Goog-FieldMask", fieldMask)

	log.Printf("Calling Places Text Search API: %s with body %s", apiURL, string(jsonData))

	// Thread-safe counter increment and API call logging
	counter.mu.Lock()
	counter.Places++
	*apiCalls = append(*apiCalls, APICallDetails{
		API:         "Places Text Search API",
		URL:         apiURL,
		RequestBody: requestBody,
		SearchArea: SearchAreaDetails{
			CenterLat: requestBody.LocationBias.Circle.Center.Latitude,
			CenterLng: requestBody.LocationBias.Circle.Center.Longitude,
			RadiusM:   requestBody.LocationBias.Circle.Radius,
		},
	})
	counter.mu.Unlock()

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
	if err := json.Unmarshal(body, &searchData); err != nil {
		return nil, fmt.Errorf("could not parse text search response. Body: %s", string(body))
	}

	return searchData.Places, nil
}

// findNearbyRestaurantsNew finds restaurants using the Places API (New).
func findNearbyRestaurantsNew(supercharger PlaceNew, counter *APICallCounter, apiCalls *[]APICallDetails) ([]RestaurantInfo, error) {
	allRestaurants := []RestaurantInfo{}
	superchargerLoc := LatLng{Lat: supercharger.Location.Latitude, Lng: supercharger.Location.Longitude}

	requestBody := SearchNearbyRequest{
		IncludedTypes:  []string{"restaurant"},
		MaxResultCount: 20, // Explicitly request maximum results
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
	fieldMask := "places.displayName,places.formattedAddress,places.rating,places.currentOpeningHours,places.location,places.primaryType,places.types"
	nearbyPlaces, err := performNewNearbySearch(requestBody, fieldMask, counter, apiCalls)

	if err != nil {
		return nil, fmt.Errorf("could not parse restaurant data: %w", err)
	}

	for _, p := range nearbyPlaces {
		walkingDistKm := haversineDistance(superchargerLoc.Lat, superchargerLoc.Lng, p.Location.Latitude, p.Location.Longitude)

		allRestaurants = append(allRestaurants, RestaurantInfo{
			Name:                  p.DisplayName.Text,
			Address:               p.FormattedAddress,
			Rating:                p.Rating,
			IsOpenNow:             p.CurrentOpeningHours.OpenNow,
			Lat:                   p.Location.Latitude,
			Lng:                   p.Location.Longitude,
			CuisineTypes:          extractCuisineFromNewPlace(p),
			WalkingDistanceMeters: int(walkingDistKm * 1000),
		})
	}

	// Sort restaurants by walking distance (closest first)
	sort.Slice(allRestaurants, func(i, j int) bool {
		return allRestaurants[i].WalkingDistanceMeters < allRestaurants[j].WalkingDistanceMeters
	})

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
// It returns the shortest distance in km, the cumulative distance along the polyline to that closest point,
// and the closest point on the polyline.
func distanceToPolyline(point LatLng, polyline []LatLng) (float64, float64, LatLng) {
	minDist := math.MaxFloat64
	distAlongRoute := 0.0
	cumulativeDist := 0.0
	var closestPoint LatLng

	for i := 0; i < len(polyline)-1; i++ {
		p1 := polyline[i]
		p2 := polyline[i+1]
		dist := distanceToSegment(point, p1, p2)

		if dist < minDist {
			minDist = dist
			// Find where on the segment the closest point lies
			l2 := (p1.Lat-p2.Lat)*(p1.Lat-p2.Lat) + (p1.Lng-p2.Lng)*(p1.Lng-p2.Lng)
			if l2 == 0.0 {
				closestPoint = p1
				distAlongRoute = cumulativeDist
			} else {
				t := ((point.Lat-p1.Lat)*(p2.Lat-p1.Lat) + (point.Lng-p1.Lng)*(p2.Lng-p1.Lng)) / l2
				t = math.Max(0, math.Min(1, t)) // Clamp to segment
				segmentLength := haversineDistance(p1.Lat, p1.Lng, p2.Lat, p2.Lng)
				distAlongRoute = cumulativeDist + t*segmentLength
				closestPoint = LatLng{
					Lat: p1.Lat + t*(p2.Lat-p1.Lat),
					Lng: p1.Lng + t*(p2.Lng-p1.Lng),
				}
			}
		}
		cumulativeDist += haversineDistance(p1.Lat, p1.Lng, p2.Lat, p2.Lng)
	}
	return minDist, distAlongRoute, closestPoint
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

// --- POLYLINE ENCODING/DECODING ---

// decodePolyline decodes a Google Maps polyline string into a slice of LatLng points.
func decodePolyline(encoded string) ([]LatLng, error) {
	var points []LatLng
	index, lat, lng := 0, 0, 0

	for index < len(encoded) {
		var result int
		var shift uint
		for {
			if index >= len(encoded) {
				return points, fmt.Errorf("polyline decoding error: unexpected end of string")
			}
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
			if index >= len(encoded) {
				return points, fmt.Errorf("polyline decoding error: unexpected end of string")
			}
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

// encodeSingleValue encodes a single integer value into a polyline chunk.
func encodeSingleValue(value int) string {
	var sb strings.Builder
	inverted := value
	if inverted < 0 {
		inverted = ^(inverted << 1)
	} else {
		inverted = inverted << 1
	}

	for inverted >= 0x20 {
		sb.WriteByte(byte((0x20 | (inverted & 0x1f)) + 63))
		inverted >>= 5
	}
	sb.WriteByte(byte(inverted + 63))
	return sb.String()
}

// encodePolyline encodes a slice of LatLng points into a Google Maps polyline string.
func encodePolyline(path []LatLng) string {
	var lastLat, lastLng int
	var result strings.Builder

	for _, point := range path {
		lat := int(math.Round(point.Lat * 1e5))
		lng := int(math.Round(point.Lng * 1e5))

		dLat := lat - lastLat
		dLng := lng - lastLng

		result.WriteString(encodeSingleValue(dLat))
		result.WriteString(encodeSingleValue(dLng))

		lastLat = lat
		lastLng = lng
	}
	return result.String()
}

// --- API CALLER FUNCTIONS ---

func getRoutesData(origin, destination string, counter *APICallCounter, apiCalls *[]APICallDetails) (*RoutesResponse, error) {
	routesRequest := RoutesRequest{
		Origin: Location{
			Address: origin,
		},
		Destination: Location{
			Address: destination,
		},
		TravelMode:        "DRIVE",
		RoutingPreference: "TRAFFIC_AWARE_OPTIMAL",
		ExtraComputations: []string{"TRAFFIC_ON_POLYLINE"},
		PolylineQuality:   "HIGH_QUALITY",
		PolylineEncoding:  "ENCODED_POLYLINE",
		DepartureTime:     time.Now().Add(1 * time.Minute).Format(time.RFC3339),
	}

	requestBody, err := json.Marshal(routesRequest)
	if err != nil {
		return nil, err
	}

	apiURL := "https://routes.googleapis.com/directions/v2:computeRoutes"
	log.Printf("Calling Routes API: %s", apiURL)

	// Thread-safe counter increment and API call logging
	counter.mu.Lock()
	counter.Directions++ // Using Directions counter for Routes API calls
	*apiCalls = append(*apiCalls, APICallDetails{API: "Routes API", URL: apiURL, RequestBody: routesRequest})
	counter.mu.Unlock()

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", googleAPIKey)
	req.Header.Set("X-Goog-FieldMask", "routes.duration,routes.distanceMeters,routes.polyline.encodedPolyline,routes.legs.duration,routes.legs.distanceMeters,routes.legs.steps.polyline.encodedPolyline,routes.legs.steps.staticDuration,routes.legs.steps.distanceMeters,routes.travelAdvisory.speedReadingIntervals")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("routes API error: %s", string(body))
	}

	var routesData RoutesResponse
	if err := json.Unmarshal(body, &routesData); err != nil {
		return nil, err
	}

	return &routesData, nil
}

func parseDurationString(durationStr string) int {
	// Parse duration strings like "2420s" to seconds
	if strings.HasSuffix(durationStr, "s") {
		if val, err := strconv.Atoi(strings.TrimSuffix(durationStr, "s")); err == nil {
			return val
		}
	}
	// Fallback: try to parse as plain number
	if val, err := strconv.Atoi(durationStr); err == nil {
		return val
	}
	log.Printf("Warning: Could not parse duration string: %s", durationStr)
	return 0
}
