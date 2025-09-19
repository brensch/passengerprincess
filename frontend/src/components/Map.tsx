import { useEffect, useRef, forwardRef, useImperativeHandle, useState, useCallback } from 'react'
import * as L from 'leaflet'
import 'leaflet.markercluster/dist/MarkerCluster.css'
import 'leaflet.markercluster/dist/MarkerCluster.Default.css'
import 'leaflet.markercluster'
import { RouteResponse, StationData, SearchFilters, ViewportResponse, Supercharger, Restaurant, RestaurantSuperchargerMapping } from '../types'

// Fix for default markers in Leaflet with Vite
delete (L.Icon.Default.prototype as any)._getIconUrl
L.Icon.Default.mergeOptions({
    iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon-2x.png',
    iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon.png',
    shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-shadow.png',
})

// Add global functions for popup buttons
declare global {
    interface Window {
        showChargerInResults?: (chargerId: string) => void
    }
}

interface MapProps {
    routeData: RouteResponse | null
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

interface ViewportSuperchargerData {
    data: Supercharger
    marker: L.Marker
    etaInfo?: {
        arrival_time: number
        distance_from_route: number
        distance_along_route: number
    }
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
        userLocation: L.LayerGroup
        viewportSuperchargers: L.MarkerClusterGroup
        viewportRestaurants: L.LayerGroup
    } | null>(null)

    // Viewport state
    const [viewportData, setViewportData] = useState<ViewportResponse | null>(null)
    const lastBoundsRef = useRef<string | null>(null)
    const viewportSuperchargers = useRef<Map<string, ViewportSuperchargerData>>(new (globalThis.Map)())
    const viewportRestaurants = useRef<Map<string, { data: Restaurant, marker: L.Marker }>>(new (globalThis.Map)())
    const viewportMappings = useRef<RestaurantSuperchargerMapping[]>([])

    const formatEpochMsToLocalTime = useCallback((epochMs: number): string => {
        const date = new Date(epochMs)
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    }, [])

    const generateSuperchargerPopupContent = useCallback((supercharger: Supercharger, chargerId?: string) => {
        // Check if this supercharger is part of the route
        const routeStation = stationData.find(s => s.chargerInfo.supercharger.place_id === supercharger.place_id)
        const finalChargerId = chargerId || routeStation?.id

        // Check for ETA info from viewport data
        const viewportSuperchargerData = viewportSuperchargers.current.get(supercharger.place_id)
        const etaInfo = viewportSuperchargerData?.etaInfo

        let content = `
            <div class="font-sans max-w-xs p-4">
                <h3 class="font-semibold text-lg mb-2">${supercharger.name}</h3>
                <p class="text-sm text-gray-600 mb-2">${supercharger.address}</p>
        `

        if (etaInfo) {
            const distFromRoute = (etaInfo.distance_from_route || 0) / 1609.34
            const totalDist = ((etaInfo.distance_along_route || 0) + (etaInfo.distance_from_route || 0)) / 1609.34
            content += `
                <div class="mt-2 text-sm text-gray-700">
                    <p><strong>Arrival:</strong> ${formatEpochMsToLocalTime(etaInfo.arrival_time)}</p>
                    <p><strong>Total Distance:</strong> ${totalDist.toFixed(1)} mi</p>
                    <p><strong>Deviation:</strong> ${distFromRoute.toFixed(1)} mi</p>
                </div>
            `
        }

        content += '<div class="flex flex-col gap-2 mt-3">'
        if (finalChargerId) {
            content += `<button onclick="showChargerInResults('${finalChargerId}')" class="px-3 py-1 bg-pink-500 text-white rounded text-sm hover:bg-pink-600">View in List</button>`
        }
        content += `
            <button onclick="window.open('${supercharger.google_maps_uri}', '_blank')" 
                    class="px-3 py-1 bg-blue-500 text-white rounded text-sm hover:bg-blue-600">
                Open in Maps
            </button>
        </div>
        </div>`

        return content
    }, [stationData, formatEpochMsToLocalTime])

    const generateRestaurantPopupContent = useCallback((restaurant: Restaurant) => {
        // Find the closest supercharger using mappings
        const mapping = viewportMappings.current.find(m => m.restaurant_id === restaurant.place_id)
        let distanceText = ''
        if (mapping) {
            const walkingDistance = (mapping.distance / 1609.34 * 5280).toFixed(0) // Convert to feet
            distanceText = `<p class="text-sm text-gray-600">Walking: ${walkingDistance} ft to supercharger</p>`
        }

        return `
            <div class="font-sans max-w-xs p-4">
                <h3 class="font-semibold text-lg mb-2">${restaurant.name}</h3>
                <p class="text-sm text-gray-600 mb-2">${restaurant.primary_type_display || 'Restaurant'}</p>
                ${restaurant.rating ? `<p class="text-sm">Rating: ${'⭐'.repeat(Math.floor(restaurant.rating))} (${restaurant.rating})</p>` : ''}
                ${distanceText}
                <button onclick="window.open('${restaurant.google_maps_uri}', '_blank')" 
                        class="mt-2 px-3 py-1 bg-green-500 text-white rounded text-sm hover:bg-green-600">
                    Open in Maps
                </button>
            </div>
        `
    }, [])

    useImperativeHandle(ref, () => ({
        fitBounds: (bounds: L.LatLngBounds) => {
            if (mapRef.current) {
                mapRef.current.fitBounds(bounds)
            }
        },
        getMap: () => mapRef.current,
        showSuperchargerPopup: (placeId: string) => {
            const viewportMarkerData = viewportSuperchargers.current.get(placeId)
            const marker = viewportMarkerData?.marker

            if (marker && mapRef.current) {
                mapRef.current.setView(marker.getLatLng(), 15)
                marker.openPopup()
            }
        },
        showRestaurantPopup: (placeId: string) => {
            const viewportMarkerData = viewportRestaurants.current.get(placeId)
            const marker = viewportMarkerData?.marker

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

        L.tileLayer('https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors',
            maxZoom: 19
        }).addTo(map)

        // Create marker cluster group for viewport superchargers
        const viewportSuperchargersLayer = L.markerClusterGroup({
            maxClusterRadius: 50,
            disableClusteringAtZoom: 15,
            spiderfyOnMaxZoom: true,
            showCoverageOnHover: false
        })

        const layers = {
            route: L.layerGroup().addTo(map),
            userLocation: L.layerGroup().addTo(map),
            viewportSuperchargers: viewportSuperchargersLayer.addTo(map),
            viewportRestaurants: L.layerGroup().addTo(map)
        }

        mapRef.current = map
        layersRef.current = layers

        return () => {
            map.remove()
            mapRef.current = null
            layersRef.current = null
        }
    }, [])

    // Handle viewport updates
    const updateViewport = useCallback(async () => {
        if (!mapRef.current || !layersRef.current) return

        const bounds = mapRef.current.getBounds()
        const boundsStr = bounds.toBBoxString()

        // Only fetch if bounds changed significantly
        if (lastBoundsRef.current === boundsStr) return
        lastBoundsRef.current = boundsStr

        const [minLng, minLat, maxLng, maxLat] = boundsStr.split(',').map(parseFloat)

        try {
            const response = await fetch(
                `/superchargers/viewport?min_lat=${minLat}&max_lat=${maxLat}&min_lng=${minLng}&max_lng=${maxLng}`
            )
            const data: ViewportResponse = await response.json()

            if (response.ok) {
                setViewportData(data)
                viewportMappings.current = data.mappings || []
                applyViewportFilters(data, bounds)
            }
        } catch (error) {
            console.error('Failed to load viewport data:', error)
        }
    }, [])

    const applyViewportFilters = useCallback((data: ViewportResponse, bounds: L.LatLngBounds) => {
        if (!layersRef.current || !mapRef.current) return

        console.log('Applying viewport filters:', {
            searchTerm: searchFilters.searchTerm,
            cuisineFilters: searchFilters.cuisineFilters,
            restaurantCount: data.restaurants.length,
            superchargerCount: data.superchargers.length
        })

        const { viewportSuperchargers: viewportSuperchargersLayer, viewportRestaurants: viewportRestaurantsLayer } = layersRef.current

        // Filter restaurants based on search criteria
        const filteredRestaurants = data.restaurants.filter(restaurant => {
            const nameMatch = !searchFilters.searchTerm ||
                restaurant.name.toLowerCase().includes(searchFilters.searchTerm.toLowerCase())
            const cuisineMatch = searchFilters.cuisineFilters.includes('') ||
                searchFilters.cuisineFilters.length === 0 ||
                searchFilters.cuisineFilters.some(cuisine =>
                    cuisine !== '' && (restaurant.primary_type_display || '').toLowerCase().includes(cuisine.toLowerCase())
                )
            return nameMatch && cuisineMatch
        })

        console.log('Filtered restaurants:', filteredRestaurants.length, 'out of', data.restaurants.length)

        // Determine which superchargers should be visible based on filters
        const hasActiveFilters = searchFilters.searchTerm !== '' ||
            (searchFilters.cuisineFilters.length > 0 && !searchFilters.cuisineFilters.includes(''))

        console.log('Has active filters:', hasActiveFilters)

        let visibleSuperchargerIds = new Set<string>()

        if (hasActiveFilters && filteredRestaurants.length > 0) {
            // Show only superchargers that have mappings to filtered restaurants OR are on the route
            const filteredRestaurantIds = new Set(filteredRestaurants.map(r => r.place_id))
            data.mappings.forEach(mapping => {
                if (filteredRestaurantIds.has(mapping.restaurant_id)) {
                    visibleSuperchargerIds.add(mapping.supercharger_id)
                }
            })
            // Also include route superchargers when filters are active
            stationData.forEach(station => {
                visibleSuperchargerIds.add(station.chargerInfo.supercharger.place_id)
            })
        } else if (!hasActiveFilters) {
            // No filters, show all superchargers
            data.superchargers.forEach(sc => visibleSuperchargerIds.add(sc.place_id))
        }

        console.log('Visible supercharger IDs:', visibleSuperchargerIds.size)

        if (hasActiveFilters && filteredRestaurants.length > 0) {
            // Show only superchargers that have mappings to filtered restaurants OR are on the route
            const filteredRestaurantIds = new Set(filteredRestaurants.map(r => r.place_id))
            data.mappings.forEach(mapping => {
                if (filteredRestaurantIds.has(mapping.restaurant_id)) {
                    visibleSuperchargerIds.add(mapping.supercharger_id)
                }
            })
            // Also include route superchargers when filters are active
            stationData.forEach(station => {
                visibleSuperchargerIds.add(station.chargerInfo.supercharger.place_id)
            })
        } else if (!hasActiveFilters) {
            // No filters, show all superchargers
            data.superchargers.forEach(sc => visibleSuperchargerIds.add(sc.place_id))
        }

        // Limit superchargers for performance (max 500)
        const maxViewportSuperchargers = 500
        const limitedSuperchargers = data.superchargers.slice(0, maxViewportSuperchargers)

        // Create a map of route superchargers for ETA enrichment
        const routeSuperchargerMap = new (globalThis.Map)<string, {
            arrival_time: number
            distance_from_route: number
            distance_along_route: number
        }>()

        if (routeData?.superchargers) {
            routeData.superchargers.forEach(routeStation => {
                routeSuperchargerMap.set(routeStation.supercharger.place_id, {
                    arrival_time: routeStation.arrival_time,
                    distance_from_route: routeStation.distance_from_route,
                    distance_along_route: routeStation.distance_along_route
                })
            })
        }

        // DELTA-BASED SUPERCHARGER UPDATES

        // 1. Remove superchargers that are no longer in viewport bounds
        for (const [placeId, superchargerData] of viewportSuperchargers.current) {
            if (!bounds.contains(superchargerData.marker.getLatLng())) {
                if (viewportSuperchargersLayer.hasLayer(superchargerData.marker)) {
                    viewportSuperchargersLayer.removeLayer(superchargerData.marker)
                }
                viewportSuperchargers.current.delete(placeId)
            }
        }

        // 2. Determine which superchargers should be visible
        const shouldBeVisibleSuperchargerIds = new Set<string>()
        limitedSuperchargers.forEach(supercharger => {
            if (bounds.contains(L.latLng(supercharger.latitude, supercharger.longitude)) &&
                visibleSuperchargerIds.has(supercharger.place_id)) {
                shouldBeVisibleSuperchargerIds.add(supercharger.place_id)
            }
        })

        // 3. Get currently visible superchargers in the layer
        const currentlyVisibleSuperchargerIds = new Set<string>()
        for (const [placeId, superchargerData] of viewportSuperchargers.current) {
            if (viewportSuperchargersLayer.hasLayer(superchargerData.marker)) {
                currentlyVisibleSuperchargerIds.add(placeId)
            }
        }

        // 4. Calculate deltas for superchargers
        const superchargersToRemove = new Set<string>()
        const superchargersToAdd = new Set<string>()

        // Find superchargers to remove (currently visible but should not be)
        for (const placeId of currentlyVisibleSuperchargerIds) {
            if (!shouldBeVisibleSuperchargerIds.has(placeId)) {
                superchargersToRemove.add(placeId)
            }
        }

        // Find superchargers to add (should be visible but currently not)
        for (const placeId of shouldBeVisibleSuperchargerIds) {
            if (!currentlyVisibleSuperchargerIds.has(placeId)) {
                superchargersToAdd.add(placeId)
            }
        }

        // 5. Add new superchargers to the collection (if not already there)
        limitedSuperchargers.forEach(supercharger => {
            if (!viewportSuperchargers.current.has(supercharger.place_id) &&
                bounds.contains(L.latLng(supercharger.latitude, supercharger.longitude))) {

                const isRouteCharger = routeSuperchargerMap.has(supercharger.place_id)
                const marker = L.marker([supercharger.latitude, supercharger.longitude], {
                    icon: L.divIcon({
                        className: 'emoji-icon charger-icon',
                        html: '⚡',
                        iconSize: isRouteCharger ? [30, 30] : [24, 24],
                        iconAnchor: isRouteCharger ? [15, 15] : [12, 12]
                    })
                })

                const routeStation = stationData.find(s => s.chargerInfo.supercharger.place_id === supercharger.place_id)
                const stationId = routeStation?.id

                const superchargerData: ViewportSuperchargerData = {
                    data: supercharger,
                    marker,
                    etaInfo: routeSuperchargerMap.get(supercharger.place_id)
                }

                marker.bindPopup(() => generateSuperchargerPopupContent(supercharger, stationId))
                viewportSuperchargers.current.set(supercharger.place_id, superchargerData)
            }
        })

        // 6. Apply delta changes - remove superchargers
        for (const placeId of superchargersToRemove) {
            const superchargerData = viewportSuperchargers.current.get(placeId)
            if (superchargerData && viewportSuperchargersLayer.hasLayer(superchargerData.marker)) {
                viewportSuperchargersLayer.removeLayer(superchargerData.marker)
            }
        }

        // 7. Apply delta changes - add superchargers
        for (const placeId of superchargersToAdd) {
            const superchargerData = viewportSuperchargers.current.get(placeId)
            if (superchargerData && !viewportSuperchargersLayer.hasLayer(superchargerData.marker)) {
                viewportSuperchargersLayer.addLayer(superchargerData.marker)
            }
        }

        // DELTA-BASED RESTAURANT UPDATES

        // Clear restaurants that are out of bounds
        for (const [placeId, restaurantData] of viewportRestaurants.current) {
            if (!bounds.contains(restaurantData.marker.getLatLng())) {
                if (viewportRestaurantsLayer.hasLayer(restaurantData.marker)) {
                    viewportRestaurantsLayer.removeLayer(restaurantData.marker)
                }
                viewportRestaurants.current.delete(placeId)
            }
        }

        if (filteredRestaurants.length <= 100) {
            // Create a map of route restaurants for consistent handling
            const routeRestaurantMap = new (globalThis.Map)<string, boolean>()
            stationData.forEach(station => {
                station.restaurants.forEach(restaurant => {
                    routeRestaurantMap.set(restaurant.place_id, true)
                })
            })

            // Determine which restaurants should be visible
            const shouldBeVisibleRestaurantIds = new Set<string>()
            filteredRestaurants.forEach(restaurant => {
                if (bounds.contains(L.latLng(restaurant.latitude, restaurant.longitude))) {
                    shouldBeVisibleRestaurantIds.add(restaurant.place_id)
                }
            })

            // Get currently visible restaurants
            const currentlyVisibleRestaurantIds = new Set<string>()
            for (const [placeId, restaurantData] of viewportRestaurants.current) {
                if (viewportRestaurantsLayer.hasLayer(restaurantData.marker)) {
                    currentlyVisibleRestaurantIds.add(placeId)
                }
            }

            // Calculate deltas for restaurants
            const restaurantsToRemove = new Set<string>()
            const restaurantsToAdd = new Set<string>()

            for (const placeId of currentlyVisibleRestaurantIds) {
                if (!shouldBeVisibleRestaurantIds.has(placeId)) {
                    restaurantsToRemove.add(placeId)
                }
            }

            for (const placeId of shouldBeVisibleRestaurantIds) {
                if (!currentlyVisibleRestaurantIds.has(placeId)) {
                    restaurantsToAdd.add(placeId)
                }
            }

            // Add new restaurants to collection
            filteredRestaurants.forEach(restaurant => {
                if (!viewportRestaurants.current.has(restaurant.place_id) &&
                    bounds.contains(L.latLng(restaurant.latitude, restaurant.longitude))) {

                    const isRouteRestaurant = routeRestaurantMap.has(restaurant.place_id)
                    const marker = L.marker([restaurant.latitude, restaurant.longitude], {
                        icon: L.divIcon({
                            className: 'emoji-icon restaurant-icon',
                            html: '🍽️',
                            iconSize: isRouteRestaurant ? [24, 24] : [20, 20],
                            iconAnchor: isRouteRestaurant ? [12, 12] : [10, 10]
                        })
                    })

                    marker.bindPopup(() => generateRestaurantPopupContent(restaurant))
                    viewportRestaurants.current.set(restaurant.place_id, { data: restaurant, marker })
                }
            })

            // Apply restaurant deltas
            for (const placeId of restaurantsToRemove) {
                const restaurantData = viewportRestaurants.current.get(placeId)
                if (restaurantData && viewportRestaurantsLayer.hasLayer(restaurantData.marker)) {
                    viewportRestaurantsLayer.removeLayer(restaurantData.marker)
                }
            }

            for (const placeId of restaurantsToAdd) {
                const restaurantData = viewportRestaurants.current.get(placeId)
                if (restaurantData && !viewportRestaurantsLayer.hasLayer(restaurantData.marker)) {
                    viewportRestaurantsLayer.addLayer(restaurantData.marker)
                }
            }
        } else {
            // Too many restaurants, hide all
            for (const [, restaurantData] of viewportRestaurants.current) {
                if (viewportRestaurantsLayer.hasLayer(restaurantData.marker)) {
                    viewportRestaurantsLayer.removeLayer(restaurantData.marker)
                }
            }
        }

        // Adjust clustering based on density
        const zoom = mapRef.current.getZoom()
        const totalSuperchargers = viewportSuperchargers.current.size

        if (totalSuperchargers > 100) {
            (viewportSuperchargersLayer as any).options.disableClusteringAtZoom = Math.max(10, zoom - 2)
        } else if (totalSuperchargers > 50) {
            (viewportSuperchargersLayer as any).options.disableClusteringAtZoom = Math.max(12, zoom - 1)
        } else {
            (viewportSuperchargersLayer as any).options.disableClusteringAtZoom = Math.max(15, zoom + 1)
        }

        setTimeout(() => viewportSuperchargersLayer.refreshClusters(), 50)

    }, [searchFilters, stationData, routeData, generateSuperchargerPopupContent, generateRestaurantPopupContent])

    // Set up viewport update handlers
    useEffect(() => {
        if (!mapRef.current) return

        const map = mapRef.current

        // Debounce viewport updates
        let viewportTimeout: ReturnType<typeof setTimeout>
        const debouncedViewportUpdate = () => {
            clearTimeout(viewportTimeout)
            viewportTimeout = setTimeout(updateViewport, 300)
        }

        map.on('moveend zoomend', debouncedViewportUpdate)

        // Initial viewport load
        setTimeout(updateViewport, 100)

        return () => {
            clearTimeout(viewportTimeout)
            map.off('moveend zoomend', debouncedViewportUpdate)
        }
    }, [updateViewport])

    // Apply filters when search filters change
    useEffect(() => {
        if (viewportData && mapRef.current) {
            const bounds = mapRef.current.getBounds()
            applyViewportFilters(viewportData, bounds)
        }
    }, [searchFilters, applyViewportFilters, viewportData, stationData])

    // Update route polyline only
    useEffect(() => {
        if (!routeData || !layersRef.current || !mapRef.current) return

        const { route } = layersRef.current
        route.clearLayers()

        if (routeData.route?.EncodedPolyline) {
            const coordinates = decodePolyline(routeData.route.EncodedPolyline)
            if (coordinates.length > 0) {
                const polyline = L.polyline(coordinates, {
                    color: '#8B5CF6',
                    weight: 5,
                    opacity: 0.8
                })
                route.addLayer(polyline)

                // Fit map to route bounds
                mapRef.current.fitBounds(polyline.getBounds().pad(0.1))
            }
        }
    }, [routeData])

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

export default Map
