package main

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
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
	http.HandleFunc("/", withGzip(serveFrontend)) // Serve the HTML file at the root
	http.HandleFunc("/autocomplete", withGzip(autocompleteHandler))
	http.HandleFunc("/route", withGzip(routeHandler))
	http.HandleFunc("/superchargers/viewport", withGzip(viewportHandler))
	http.HandleFunc("/reverse-geocode", withGzip(reverseGeocodeHandler))

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

// serveFrontend serves the frontend HTML file with API key templating
func serveFrontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the frontend HTML file
	htmlContent, err := os.ReadFile("frontend/index.html")
	if err != nil {
		log.Printf("Error reading frontend file: %v", err)
		writeJSONError(w, "Could not load frontend", http.StatusInternalServerError)
		return
	}

	// Parse the template and inject the API key
	tmpl, err := template.New("frontend").Parse(string(htmlContent))
	if err != nil {
		log.Printf("Error parsing frontend template: %v", err)
		writeJSONError(w, "Could not parse frontend", http.StatusInternalServerError)
		return
	}

	// Set content type to HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute template with API key
	data := struct {
		APIKey string
	}{
		APIKey: googleAPIKey,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing frontend template: %v", err)
		writeJSONError(w, "Could not render frontend", http.StatusInternalServerError)
		return
	}
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

	// Get autocomplete suggestions with session token
	suggestions, err := maps.GetAutocompleteSuggestions(ctx, googleAPIKey, partial, sessionToken)
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

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get database service
	service := db.GetDefaultService()

	// Get route with superchargers
	result, err := maps.GetSuperchargersOnRoute(ctx, service, googleAPIKey, origin, destination)
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
