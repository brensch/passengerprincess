export interface Restaurant {
    place_id: string
    name: string
    rating?: number
    price_level?: number
    primary_type_display?: string
    latitude: number
    longitude: number
    distance_from_charger?: number
    google_maps_uri?: string
}

export interface Supercharger {
    place_id: string
    name: string
    latitude: number
    longitude: number
    address?: string
    rating?: number
    google_maps_uri?: string
    eta_ms?: number
    distance_along_route?: number
    distance_from_route?: number
}

export interface StationData {
    id: string
    chargerInfo: {
        supercharger: Supercharger
        restaurants: Restaurant[]
        eta_ms?: number
        distance_along_route?: number
        distance_from_route?: number
    }
    restaurants: Restaurant[]
}

export interface RouteData {
    route: {
        EncodedPolyline: string
        travelAdvisory?: {
            speedReadingIntervals?: Array<{
                startPolylinePointIndex: number
                endPolylinePointIndex: number
                speed: 'NORMAL' | 'SLOW' | 'TRAFFIC_JAM'
            }>
        }
    }
    superchargers: Array<{
        supercharger: Supercharger
        restaurants: Restaurant[]
        eta_ms?: number
        distance_along_route?: number
        distance_from_route?: number
    }>
}

export type ViewMode = 'search' | 'results' | 'map'

export interface AutocompleteResult {
    description: string
    place_id: string
    isMyLocation?: boolean
}

export interface SearchFilters {
    searchTerm: string
    cuisineFilters: string[]
}
