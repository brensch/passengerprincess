// Core database models matching the Go backend
export interface Restaurant {
    place_id: string
    name: string
    address: string
    latitude: number
    longitude: number
    rating: number
    user_ratings_total: number
    primary_type: string
    primary_type_display: string
    display_name: string
    google_maps_uri: string
    last_updated: string
}

export interface RestaurantWithDistance extends Restaurant {
    distance: number  // Walking distance to the supercharger in meters
}

export interface Supercharger {
    place_id: string
    name: string
    address: string
    latitude: number
    longitude: number
    google_maps_uri: string
    last_updated: string
    is_supercharger: boolean
}

export interface RestaurantSuperchargerMapping {
    restaurant_id: string
    supercharger_id: string
    distance: number
}

// Route API response types
export interface SuperchargerWithETA {
    supercharger: Supercharger
    restaurants: RestaurantWithDistance[]
    arrival_time: number  // Epoch milliseconds
    distance_from_route: number  // Distance from route in meters
    distance_along_route: number  // Distance along route in meters
    closest_point_on_route: { lat: number; lng: number }
}

export interface RouteInfo {
    DistanceMeters: number
    Duration: number  // Duration in nanoseconds (Go duration)
    EncodedPolyline: string
    travelAdvisory?: {
        speedReadingIntervals?: Array<{
            startPolylinePointIndex: number
            endPolylinePointIndex: number
            speed: 'NORMAL' | 'SLOW' | 'TRAFFIC_JAM'
        }>
    }
}

export interface RouteResponse {
    route: RouteInfo
    superchargers: SuperchargerWithETA[]
}

// Viewport API response types
export interface ViewportResponse {
    superchargers?: Supercharger[]
    restaurants?: Restaurant[]
    mappings?: RestaurantSuperchargerMapping[]
}

// UI types
export interface SearchFilters {
    selectedPlaces: string[]
    typedPlace: string
    selectedCuisines: string[]
}

export type ViewMode = 'search' | 'results' | 'map'

export interface AutocompleteResult {
    description: string
    place_id: string
    isMyLocation?: boolean
}

// Legacy compatibility type (for the table - to be refactored)
export interface StationData {
    id: string
    chargerInfo: {
        supercharger: Supercharger
        restaurants?: RestaurantWithDistance[]
        distance_along_route?: number
        distance_from_route?: number
    }
    restaurants?: RestaurantWithDistance[]
}
