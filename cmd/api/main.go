package main

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/brensch/passengerprincess/pkg/db"
	"github.com/brensch/passengerprincess/pkg/maps"
	"gorm.io/gorm/logger"
)

// Global variable for the Google Maps API key.
var googleAPIKey = os.Getenv("MAPS_API_KEY")

// gzipResponseWriter wraps http.ResponseWriter to enable gzip compression
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	return g.Writer.Write(data)
}

// withGzip is a middleware that enables gzip compression for responses
func withGzip(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzw := &gzipResponseWriter{ResponseWriter: w, Writer: gz}
		fn(gzw, r)
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if strings.Contains(ip, ":") {
		ip, _, _ = strings.Cut(ip, ":")
	}
	return ip
}

// generateSessionToken creates a random session token for Google Places Autocomplete
func generateSessionToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func main() {
	// Check if the API key is set.
	if googleAPIKey == "" {
		googleAPIKey = "YOUR_GOOGLE_MAPS_API_KEY" // Fallback for local testing
		log.Println("WARNING: MAPS_API_KEY environment variable not set. Using placeholder.")
	}
	if googleAPIKey == "YOUR_GOOGLE_MAPS_API_KEY" {
		log.Fatal("FATAL: Please replace 'YOUR_GOOGLE_MAPS_API_KEY' with your actual Google Maps API key.")
	}

	// Initialize database
	config := &db.Config{
		DatabasePath: "db/passengerprincess.db",
		LogLevel:     logger.Warn,
	}
	if err := db.Initialize(config); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Register handlers.
	http.Handle("/", http.FileServer(http.Dir("frontend/dist/"))) // Serve static files from frontend dist
	http.HandleFunc("/autocomplete", withGzip(autocompleteHandler))
	http.HandleFunc("/route", withGzip(routeHandler))
	http.HandleFunc("/superchargers/viewport", withGzip(viewportHandler))
	http.HandleFunc("/reverse-geocode", withGzip(reverseGeocodeHandler))
	http.HandleFunc("/stats", statsHandler)

	// Start the server.
	port := "8040"
	log.Printf("Server starting...")
	log.Printf("Access the web interface at http://localhost:%s/", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// writeJSONError sends a JSON-formatted error message.
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]string{"error": message})
}

// autocompleteHandler handles place autocomplete requests
func autocompleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	partial := strings.TrimSpace(r.URL.Query().Get("partial"))
	if partial == "" {
		writeJSONError(w, "partial parameter is required", http.StatusBadRequest)
		return
	}

	// Get session token from query parameter, or generate a new one
	sessionToken := strings.TrimSpace(r.URL.Query().Get("session_token"))
	if sessionToken == "" {
		// Generate new session token
		newToken, err := generateSessionToken()
		if err != nil {
			log.Printf("Error generating session token: %v", err)
			writeJSONError(w, "Failed to generate session token", http.StatusInternalServerError)
			return
		}
		sessionToken = newToken
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get database service
	service := db.GetDefaultService()

	// Get autocomplete suggestions with session token
	suggestions, err := maps.GetAutocompleteSuggestions(ctx, service, googleAPIKey, partial, sessionToken, getClientIP(r))
	if err != nil {
		log.Printf("Error getting autocomplete suggestions: %v", err)
		writeJSONError(w, "Failed to get autocomplete suggestions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]interface{}{
		"predictions":   suggestions,
		"session_token": sessionToken,
	})
}

// routeHandler handles route planning requests with superchargers
func routeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	origin := strings.TrimSpace(r.URL.Query().Get("origin"))
	destination := strings.TrimSpace(r.URL.Query().Get("destination"))

	if origin == "" || destination == "" {
		writeJSONError(w, "Both origin and destination parameters are required", http.StatusBadRequest)
		return
	}

	// Extract client information
	ipAddress := getClientIP(r)
	var latitude, longitude *float64
	if latStr := r.URL.Query().Get("lat"); latStr != "" {
		if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
			latitude = &lat
		}
	}
	if lngStr := r.URL.Query().Get("lng"); lngStr != "" {
		if lng, err := strconv.ParseFloat(lngStr, 64); err == nil {
			longitude = &lng
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get database service
	service := db.GetDefaultService()

	// Get route with superchargers
	result, err := maps.GetSuperchargersOnRoute(ctx, service, googleAPIKey, origin, destination, ipAddress, latitude, longitude)
	if err != nil {
		log.Printf("Error getting superchargers on route: %v", err)
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(result)
}

// viewportHandler handles requests for superchargers within a viewport
func viewportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse viewport bounds from query parameters
	minLatStr := r.URL.Query().Get("min_lat")
	maxLatStr := r.URL.Query().Get("max_lat")
	minLngStr := r.URL.Query().Get("min_lng")
	maxLngStr := r.URL.Query().Get("max_lng")

	if minLatStr == "" || maxLatStr == "" || minLngStr == "" || maxLngStr == "" {
		writeJSONError(w, "All viewport bounds (min_lat, max_lat, min_lng, max_lng) are required", http.StatusBadRequest)
		return
	}

	minLat, err := strconv.ParseFloat(minLatStr, 64)
	if err != nil {
		writeJSONError(w, "Invalid min_lat parameter", http.StatusBadRequest)
		return
	}

	maxLat, err := strconv.ParseFloat(maxLatStr, 64)
	if err != nil {
		writeJSONError(w, "Invalid max_lat parameter", http.StatusBadRequest)
		return
	}

	minLng, err := strconv.ParseFloat(minLngStr, 64)
	if err != nil {
		writeJSONError(w, "Invalid min_lng parameter", http.StatusBadRequest)
		return
	}

	maxLng, err := strconv.ParseFloat(maxLngStr, 64)
	if err != nil {
		writeJSONError(w, "Invalid max_lng parameter", http.StatusBadRequest)
		return
	}

	// Get database service
	service := db.GetDefaultService()

	// Get superchargers within the viewport bounds
	superchargers, err := service.Supercharger.GetByLocation(minLat, maxLat, minLng, maxLng)
	if err != nil {
		log.Printf("Error getting superchargers by location: %v", err)
		writeJSONError(w, "Failed to get superchargers", http.StatusInternalServerError)
		return
	}

	// Get restaurants within the viewport bounds
	restaurants, err := service.Restaurant.GetByLocation(minLat, maxLat, minLng, maxLng)
	if err != nil {
		log.Printf("Error getting restaurants by location: %v", err)
		writeJSONError(w, "Failed to get restaurants", http.StatusInternalServerError)
		return
	}

	// Get mappings for the superchargers
	var superchargerIDs []string
	for _, sc := range superchargers {
		superchargerIDs = append(superchargerIDs, sc.PlaceID)
	}
	var mappings []db.RestaurantSuperchargerMapping
	if len(superchargerIDs) > 0 {
		mappings, err = service.Supercharger.GetMappingsForSuperchargers(superchargerIDs)
		if err != nil {
			log.Printf("Error getting mappings: %v", err)
			writeJSONError(w, "Failed to get mappings", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]interface{}{
		"superchargers": superchargers,
		"restaurants":   restaurants,
		"mappings":      mappings,
	})
}

// reverseGeocodeHandler handles reverse geocoding requests (coordinates to address)
func reverseGeocodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get latitude and longitude from query parameters
	latStr := strings.TrimSpace(r.URL.Query().Get("lat"))
	lonStr := strings.TrimSpace(r.URL.Query().Get("lon"))

	if latStr == "" || lonStr == "" {
		writeJSONError(w, "Both lat and lon parameters are required", http.StatusBadRequest)
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		writeJSONError(w, "Invalid lat parameter", http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		writeJSONError(w, "Invalid lon parameter", http.StatusBadRequest)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get database service
	service := db.GetDefaultService()

	// Call Google Maps Geocoding API
	url := "https://maps.googleapis.com/maps/api/geocode/json?latlng=" + latStr + "," + lonStr + "&key=" + googleAPIKey

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("Error creating reverse geocoding request: %v", err)
		writeJSONError(w, "Failed to create geocoding request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making reverse geocoding request: %v", err)
		writeJSONError(w, "Failed to geocode location", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var geocodeData struct {
		Status  string `json:"status"`
		Results []struct {
			FormattedAddress string `json:"formatted_address"`
		} `json:"results"`
		ErrorMessage string `json:"error_message,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geocodeData); err != nil {
		log.Printf("Error decoding reverse geocoding response: %v", err)
		writeJSONError(w, "Failed to decode geocoding response", http.StatusInternalServerError)
		return
	}

	// Log the API call
	if service != nil {
		logEntry := &db.MapsCallLog{
			SKU:       "geocoding",
			IPAddress: getClientIP(r),
			Details:   fmt.Sprintf("Reverse geocoding for lat: %s, lon: %s", latStr, lonStr),
		}
		if err := service.MapsCallLog.Create(logEntry); err != nil {
			// Log error but don't fail the API call
			log.Printf("Failed to log maps call: %v", err)
		}
	}

	if geocodeData.Status != "OK" {
		log.Printf("Reverse geocoding failed with status: %s, message: %s", geocodeData.Status, geocodeData.ErrorMessage)
		// Fallback to coordinates if reverse geocoding fails
		formattedCoords := strconv.FormatFloat(lat, 'f', 6, 64) + ", " + strconv.FormatFloat(lon, 'f', 6, 64)
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.Encode(map[string]interface{}{
			"address": formattedCoords,
			"status":  "fallback",
		})
		return
	}

	if len(geocodeData.Results) == 0 {
		// Fallback to coordinates if no results
		formattedCoords := strconv.FormatFloat(lat, 'f', 6, 64) + ", " + strconv.FormatFloat(lon, 'f', 6, 64)
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.Encode(map[string]interface{}{
			"address": formattedCoords,
			"status":  "fallback",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]interface{}{
		"address": geocodeData.Results[0].FormattedAddress,
		"status":  "success",
	})
}

// statsHandler handles requests for API usage statistics
func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get database service
	service := db.GetDefaultService()

	// Get data for last 30 days
	// thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	// Get total API calls in last 30 days
	totalCalls, err := service.MapsCallLog.Count()
	if err != nil {
		log.Printf("Error getting total calls: %v", err)
		writeJSONError(w, "Failed to get statistics", http.StatusInternalServerError)
		return
	}

	// Get all logs (simplified approach)
	allLogs, err := service.MapsCallLog.GetBySKU("", 10000, 0)
	if err != nil {
		log.Printf("Error getting all logs: %v", err)
		allLogs = []db.MapsCallLog{}
	}

	log.Printf("Retrieved %d total logs", len(allLogs))

	// Aggregate by IP
	ipStats := make(map[string]int)
	skuStats := make(map[string]int)
	for _, log := range allLogs {
		ipStats[log.IPAddress]++
		skuStats[log.SKU]++
	}

	// Create sorted list of SKUs
	var sortedSKUs []string
	for sku := range skuStats {
		sortedSKUs = append(sortedSKUs, sku)
	}
	sort.Strings(sortedSKUs)

	// Create sorted list of IPs
	var sortedIPs []string
	for ip := range ipStats {
		sortedIPs = append(sortedIPs, ip)
	}
	sort.Strings(sortedIPs)

	// Get last 50 route calls
	routeLogs, err := service.RouteCallLog.GetByIPAddress("", 50, 0)
	if err != nil {
		log.Printf("Error getting route logs: %v", err)
		routeLogs = []db.RouteCallLog{}
	}

	// Generate HTML response
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>API Usage Statistics</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1, h2 { color: #333; }
        table { border-collapse: collapse; width: 100%%; margin-bottom: 20px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .summary { background-color: #e8f4f8; padding: 10px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <h1>API Usage Statistics</h1>
    
    <div class="summary">
        <h2>Summary (Last 30 Days)</h2>
        <p><strong>Total API Calls:</strong> %d</p>
        <p><strong>Unique IPs:</strong> %d</p>
    </div>

    <h2>API Calls by SKU</h2>
    <table>
        <tr><th>SKU</th><th>Count</th></tr>`,
		totalCalls, len(ipStats))

	for _, sku := range sortedSKUs {
		fmt.Fprintf(w, "<tr><td>%s</td><td>%d</td></tr>", sku, skuStats[sku])
	}

	fmt.Fprintf(w, `
    </table>

    <h2>API Calls by IP Address</h2>
    <table>
        <tr><th>IP Address</th><th>Count</th></tr>`)

	for _, ip := range sortedIPs {
		count := ipStats[ip]
		if ip == "" {
			ip = "(empty)"
		}
		fmt.Fprintf(w, "<tr><td>%s</td><td>%d</td></tr>", ip, count)
	}

	fmt.Fprintf(w, `
    </table>

    <h2>Last 50 Route Requests</h2>
    <table>
        <tr><th>Timestamp</th><th>Origin</th><th>Destination</th><th>IP Address</th><th>Latitude</th><th>Longitude</th></tr>`)

	for _, route := range routeLogs {
		lat := ""
		lon := ""
		if route.Latitude != nil {
			lat = fmt.Sprintf("%.6f", *route.Latitude)
		}
		if route.Longitude != nil {
			lon = fmt.Sprintf("%.6f", *route.Longitude)
		}
		fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			route.Timestamp.Format("2006-01-02 15:04:05"), route.Origin, route.Destination, route.IPAddress, lat, lon)
	}

	fmt.Fprintf(w, `
    </table>
</body>
</html>`)
}
