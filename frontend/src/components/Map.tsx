import { useEffect, useRef, forwardRef, useImperativeHandle } from 'react'
import * as L from 'leaflet'
import { RouteData, StationData, SearchFilters } from '../types'

// Fix for default markers in Leaflet with Vite
delete (L.Icon.Default.prototype as any)._getIconUrl
L.Icon.Default.mergeOptions({
    iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon-2x.png',
    iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon.png',
    shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-shadow.png',
})

interface MapProps {
    routeData: RouteData | null
    stationData: StationData[]
    userLocation: [number, number] | null
    searchFilters: SearchFilters
    className?: string
}

export interface MapRef {
    fitBounds: (bounds: L.LatLngBounds) => void
    getMap: () => L.Map | null
    showSuperchargerPopup: (placeId: string) => void
    showRestaurantPopup: (placeId: string) => void
}

const Map = forwardRef<MapRef, MapProps>(({
    routeData,
    stationData,
    userLocation,
    searchFilters,
    className = ''
}, ref) => {
    const mapRef = useRef<L.Map | null>(null)
    const mapContainerRef = useRef<HTMLDivElement>(null)
    const layersRef = useRef<{
        route: L.LayerGroup
        superchargers: L.LayerGroup
        restaurants: L.LayerGroup
        userLocation: L.LayerGroup
    } | null>(null)
    const markersRef = useRef<{
        superchargers: { [key: string]: L.Marker }
        restaurants: { [key: string]: L.Marker }
    }>({ superchargers: {}, restaurants: {} })

    useImperativeHandle(ref, () => ({
        fitBounds: (bounds: L.LatLngBounds) => {
            if (mapRef.current) {
                mapRef.current.fitBounds(bounds)
            }
        },
        getMap: () => mapRef.current,
        showSuperchargerPopup: (placeId: string) => {
            const marker = markersRef.current.superchargers[placeId]
            if (marker && mapRef.current) {
                mapRef.current.setView(marker.getLatLng(), 15)
                marker.openPopup()
            }
        },
        showRestaurantPopup: (placeId: string) => {
            const marker = markersRef.current.restaurants[placeId]
            if (marker && mapRef.current) {
                mapRef.current.setView(marker.getLatLng(), 15)
                marker.openPopup()
            }
        }
    }))

    // Initialize map
    useEffect(() => {
        if (!mapContainerRef.current || mapRef.current) return

        const map = L.map(mapContainerRef.current).setView([37.7749, -122.4194], 6)

        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors'
        }).addTo(map)

        const layers = {
            route: L.layerGroup().addTo(map),
            superchargers: L.layerGroup().addTo(map),
            restaurants: L.layerGroup().addTo(map),
            userLocation: L.layerGroup().addTo(map)
        }

        mapRef.current = map
        layersRef.current = layers

        return () => {
            map.remove()
            mapRef.current = null
            layersRef.current = null
        }
    }, [])

    // Update route
    useEffect(() => {
        if (!routeData || !layersRef.current || !mapRef.current) return

        const { route } = layersRef.current
        route.clearLayers()

        if (routeData.route?.EncodedPolyline) {
            const coordinates = decodePolyline(routeData.route.EncodedPolyline)
            if (coordinates.length > 0) {
                const polyline = L.polyline(coordinates, {
                    color: 'blue',
                    weight: 5,
                    opacity: 0.8
                })
                route.addLayer(polyline)

                // Fit map to route bounds
                mapRef.current.fitBounds(polyline.getBounds().pad(0.1))
            }
        }
    }, [routeData])

    // Update superchargers and restaurants
    useEffect(() => {
        if (!layersRef.current) return

        const { superchargers, restaurants } = layersRef.current
        superchargers.clearLayers()
        restaurants.clearLayers()

        // Clear marker refs
        markersRef.current.superchargers = {}
        markersRef.current.restaurants = {}

        stationData.forEach((station) => {
            const { chargerInfo } = station

            // Add supercharger marker
            const superchargerMarker = L.marker([
                chargerInfo.supercharger.latitude,
                chargerInfo.supercharger.longitude
            ], {
                icon: L.divIcon({
                    className: 'emoji-icon charger-icon',
                    html: '⚡',
                    iconSize: [30, 30],
                    iconAnchor: [15, 15]
                })
            })

            superchargerMarker.bindPopup(`
        <div class="p-4">
          <h3 class="font-semibold text-lg mb-2">${chargerInfo.supercharger.name}</h3>
          <p class="text-sm text-gray-600 mb-2">${chargerInfo.supercharger.address || ''}</p>
          ${chargerInfo.supercharger.arrival_time ? `<p class="text-sm">ETA: ${formatEpochMsToLocalTime(chargerInfo.supercharger.arrival_time)}</p>` : ''}
          <button onclick="window.open('${chargerInfo.supercharger.google_maps_uri}', '_blank')" 
                  class="mt-2 px-3 py-1 bg-blue-500 text-white rounded text-sm hover:bg-blue-600">
            Open in Maps
          </button>
        </div>
      `)

            superchargers.addLayer(superchargerMarker)
            markersRef.current.superchargers[chargerInfo.supercharger.place_id] = superchargerMarker

            // Add restaurant markers (filtered)
            station.restaurants.forEach((restaurant) => {
                const nameMatch = !searchFilters.searchTerm ||
                    restaurant.name.toLowerCase().includes(searchFilters.searchTerm.toLowerCase())
                const cuisineMatch = searchFilters.cuisineFilters.includes('') ||
                    searchFilters.cuisineFilters.length === 0 ||
                    searchFilters.cuisineFilters.some(cuisine =>
                        cuisine !== '' && (restaurant.primary_type_display || '').toLowerCase().includes(cuisine.toLowerCase())
                    )

                if (nameMatch && cuisineMatch) {
                    const restaurantMarker = L.marker([restaurant.latitude, restaurant.longitude], {
                        icon: L.divIcon({
                            className: 'emoji-icon restaurant-icon',
                            html: '🍽️',
                            iconSize: [24, 24],
                            iconAnchor: [12, 12]
                        })
                    })

                    restaurantMarker.bindPopup(`
            <div class="p-4">
              <h3 class="font-semibold text-lg mb-2">${restaurant.name}</h3>
              <p class="text-sm text-gray-600 mb-2">${restaurant.primary_type_display || 'Restaurant'}</p>
              ${restaurant.rating ? `<p class="text-sm">Rating: ${'⭐'.repeat(Math.floor(restaurant.rating))} (${restaurant.rating})</p>` : ''}
              <button onclick="window.open('${restaurant.google_maps_uri}', '_blank')" 
                      class="mt-2 px-3 py-1 bg-green-500 text-white rounded text-sm hover:bg-green-600">
                Open in Maps
              </button>
            </div>
          `)

                    restaurants.addLayer(restaurantMarker)
                    markersRef.current.restaurants[restaurant.place_id] = restaurantMarker
                }
            })
        })
    }, [stationData, searchFilters])

    // Update user location
    useEffect(() => {
        if (!layersRef.current) return

        const { userLocation: userLocationLayer } = layersRef.current
        userLocationLayer.clearLayers()

        if (userLocation) {
            const userMarker = L.marker(userLocation, {
                icon: L.divIcon({
                    className: 'princess-marker',
                    html: '👸',
                    iconSize: [36, 36],
                    iconAnchor: [18, 18]
                })
            })

            userMarker.bindPopup(`
        <div class="p-4">
          <h3 class="font-semibold text-lg mb-2">Princess Location</h3>
          <p class="text-sm">You are extremely nice.</p>
        </div>
      `)

            userLocationLayer.addLayer(userMarker)
        }
    }, [userLocation])

    // Handle map resize
    useEffect(() => {
        if (mapRef.current) {
            setTimeout(() => {
                mapRef.current?.invalidateSize()
            }, 100)
        }
    }, [className])

    return (
        <div
            ref={mapContainerRef}
            className={`map-container ${className}`}
            style={{ height: '100%', width: '100%' }}
        />
    )
})

Map.displayName = 'Map'

// Utility functions
function decodePolyline(encoded: string): [number, number][] {
    const len = encoded.length
    const coords: [number, number][] = []
    let index = 0
    let lat = 0
    let lng = 0

    while (index < len) {
        let b: number
        let shift = 0
        let result = 0
        do {
            b = encoded.charCodeAt(index++) - 63
            result |= (b & 0x1f) << shift
            shift += 5
        } while (b >= 0x20)
        const dlat = ((result & 1) ? ~(result >> 1) : (result >> 1))
        lat += dlat

        shift = 0
        result = 0
        do {
            b = encoded.charCodeAt(index++) - 63
            result |= (b & 0x1f) << shift
            shift += 5
        } while (b >= 0x20)
        const dlng = ((result & 1) ? ~(result >> 1) : (result >> 1))
        lng += dlng

        coords.push([lat / 1e5, lng / 1e5])
    }

    return coords
}

function formatEpochMsToLocalTime(epochMs: number): string {
    if (!epochMs || epochMs === 0) {
        return 'N/A'
    }

    try {
        const date = new Date(epochMs)
        return date.toLocaleTimeString('en-US', {
            hour: 'numeric',
            minute: '2-digit',
            hour12: true
        })
    } catch (error) {
        console.error('Error formatting epoch time:', error)
        return 'N/A'
    }
}

export default Map
