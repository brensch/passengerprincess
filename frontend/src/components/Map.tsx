import { useEffect, useRef, forwardRef, useImperativeHandle, useState, useCallback } from 'react'
import * as L from 'leaflet'
import 'leaflet.markercluster/dist/MarkerCluster.css'
import 'leaflet.markercluster/dist/MarkerCluster.Default.css'
import 'leaflet.markercluster'
import { RouteResponse, StationData, SearchFilters, ViewportResponse, Supercharger, Restaurant, RestaurantSuperchargerMapping } from '../types'
import { useViewport } from '../contexts/ViewportContext'
import { getRestaurantEmoji } from '../utils/restaurantEmojiMapping'

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
    onRefresh?: () => void
    isLoading?: boolean
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
    className = '',
    onRefresh,
    isLoading = false
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
    const { viewport } = useViewport()

    // Warning popup state
    const [showLocationWarning, setShowLocationWarning] = useState(false)
    const [locationPermission, setLocationPermission] = useState<'unknown' | 'granted' | 'denied'>('unknown')

    const formatEpochMsToLocalTime = useCallback((epochMs: number): string => {
        const date = new Date(epochMs)
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    }, [])

    const handleGoToUserLocation = useCallback(() => {
        if (userLocation && mapRef.current) {
            mapRef.current.setView(userLocation, 15, { animate: true })
        } else {
            setShowLocationWarning(true)
            setTimeout(() => setShowLocationWarning(false), 2000)
        }
    }, [userLocation])

    const generateSuperchargerPopupContent = useCallback((supercharger: Supercharger, chargerId?: string) => {
        // Check if this supercharger is part of the route
        const routeStation = stationData.find(s => s.chargerInfo.supercharger.place_id === supercharger.place_id)
        const finalChargerId = chargerId || routeStation?.id

        // Check for ETA info from viewport data
        const viewportSuperchargerData = viewportSuperchargers.current.get(supercharger.place_id)
        const etaInfo = viewportSuperchargerData?.etaInfo

        let content = `
            <div class="font-sans max-w-xs p-3 bg-gradient-to-br from-princess-lavender via-princess-lilac to-princess-rose rounded-lg shadow-lg border border-princess-border">
                <h3 class="font-semibold text-lg mb-1 text-princess-text-primary">${supercharger.name}</h3>
                <p class="text-sm text-princess-text-secondary mb-1">${supercharger.address}</p>
        `

        if (etaInfo) {
            const distFromRoute = (etaInfo.distance_from_route || 0) / 1609.34
            const totalDist = ((etaInfo.distance_along_route || 0) + (etaInfo.distance_from_route || 0)) / 1609.34
            content += `
                <div class="mt-1 text-sm text-princess-text-primary space-y-0.5">
                    <p><strong>Arrival:</strong> ${formatEpochMsToLocalTime(etaInfo.arrival_time)}</p>
                    <p><strong>Total Distance:</strong> ${totalDist.toFixed(1)} mi</p>
                    <p><strong>Deviation:</strong> ${distFromRoute.toFixed(1)} mi</p>
                </div>
            `
        }

        content += '<div class="flex flex-col gap-1.5 mt-2">'
        if (finalChargerId) {
            content += `<button onclick="showChargerInResults('${finalChargerId}')" class="px-3 py-1.5 bg-gradient-to-r from-purple-500 to-purple-600 text-white rounded-md text-sm font-medium transition-all duration-200 hover:from-purple-600 hover:to-purple-700 hover:shadow-md">View in List</button>`
        }
        content += `
            <button onclick="window.open('${supercharger.google_maps_uri}', '_blank')" 
                    class="px-3 py-1.5 bg-gradient-to-r from-orange-400 to-orange-500 text-white rounded-md text-sm font-medium transition-all duration-200 hover:from-orange-500 hover:to-orange-600 hover:shadow-md">
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
            distanceText = `Walking: ${walkingDistance} ft to supercharger`
        }

        const emoji = getRestaurantEmoji(restaurant.primary_type)

        return `
            <div class="font-sans max-w-xs p-3 bg-gradient-to-br from-princess-lavender via-princess-lilac to-princess-rose rounded-lg shadow-lg border border-princess-border">
                <h3 class="font-semibold text-lg mb-1 text-princess-text-primary">${restaurant.name}</h3>
                <p class="text-sm text-princess-text-secondary mb-1">${emoji} ${restaurant.primary_type_display || 'Restaurant'}</p>
                ${restaurant.rating ? `<p class="text-sm mb-1 text-princess-text-primary">Rating: ${'⭐'.repeat(Math.floor(restaurant.rating))} (${restaurant.rating})</p>` : ''}
                ${distanceText ? `<p class="text-sm text-princess-text-secondary mb-1">${distanceText}</p>` : ''}
                <button onclick="window.open('${restaurant.google_maps_uri}', '_blank')" 
                        class="mt-2 px-3 py-1.5 bg-gradient-to-r from-green-500 to-green-600 text-white rounded-md text-sm font-medium transition-all duration-200 hover:from-green-600 hover:to-green-700 hover:shadow-md">
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
            if (!mapRef.current) {
                return
            }

            // First try to find the marker in viewport data
            const viewportMarkerData = viewportSuperchargers.current.get(placeId)
            if (viewportMarkerData?.marker) {
                viewportMarkerData.marker.openPopup()
                return
            }

            // If not found in viewport, find the supercharger in station data and create a temporary popup
            const station = stationData.find(s => s.chargerInfo.supercharger.place_id === placeId)
            if (station) {
                const supercharger = station.chargerInfo.supercharger
                const latLng = L.latLng(supercharger.latitude, supercharger.longitude)

                // Create and show a temporary popup immediately
                L.popup({
                    className: 'custom-popup'
                })
                    .setLatLng(latLng)
                    .setContent(generateSuperchargerPopupContent(supercharger, station.id))
                    .openOn(mapRef.current)
            }
        },
        showRestaurantPopup: (placeId: string) => {
            if (!mapRef.current) {
                return
            }

            // First try to find the marker in viewport data
            const viewportMarkerData = viewportRestaurants.current.get(placeId)
            if (viewportMarkerData?.marker) {
                viewportMarkerData.marker.openPopup()
                return
            }

            // If not found in viewport, find the restaurant in station data and create a temporary popup
            for (const station of stationData) {
                const restaurant = station.restaurants?.find(r => r.place_id === placeId)
                if (restaurant) {
                    const latLng = L.latLng(restaurant.latitude, restaurant.longitude)

                    // Create and show a temporary popup immediately
                    L.popup({
                        className: 'custom-popup'
                    })
                        .setLatLng(latLng)
                        .setContent(generateRestaurantPopupContent(restaurant))
                        .openOn(mapRef.current)
                    return
                }
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
            // Failed to load viewport data
        }
    }, [])

    const applyViewportFilters = useCallback((data: ViewportResponse, bounds: L.LatLngBounds) => {
        if (!layersRef.current || !mapRef.current) return

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

        // Determine which superchargers should be visible based on filters
        const hasActiveFilters = searchFilters.searchTerm !== '' ||
            (searchFilters.cuisineFilters.length > 0 && !searchFilters.cuisineFilters.includes(''))

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

                marker.bindPopup(() => generateSuperchargerPopupContent(supercharger, stationId), {
                    className: 'custom-popup'
                })
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
                    const restaurantEmoji = getRestaurantEmoji(restaurant.primary_type)
                    const marker = L.marker([restaurant.latitude, restaurant.longitude], {
                        icon: L.divIcon({
                            className: 'emoji-icon restaurant-icon',
                            html: restaurantEmoji,
                            iconSize: isRouteRestaurant ? [24, 24] : [20, 20],
                            iconAnchor: isRouteRestaurant ? [12, 12] : [10, 10]
                        })
                    })

                    marker.bindPopup(() => generateRestaurantPopupContent(restaurant), {
                        className: 'custom-popup'
                    })
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

    // Update route polyline 
    useEffect(() => {
        if (!routeData || !layersRef.current || !mapRef.current) {
            return
        }

        const { route } = layersRef.current
        route.clearLayers()

        if (routeData.route?.EncodedPolyline) {
            const coordinates = decodePolyline(routeData.route.EncodedPolyline)
            if (coordinates.length > 0) {
                // Draw base route
                const basePolyline = L.polyline(coordinates, {
                    color: '#8B5CF6',
                    weight: 5,
                    opacity: 0.8
                })
                route.addLayer(basePolyline)

                // Draw traffic segments if available
                if (routeData.route.travelAdvisory?.speedReadingIntervals) {
                    const trafficSegments = buildTrafficSegments(
                        routeData.route.EncodedPolyline,
                        routeData.route.travelAdvisory.speedReadingIntervals
                    )
                    drawTrafficSegments(trafficSegments, route)
                }
            }
        }
    }, [routeData])

    // Sync map view with viewport context only when explicitly requested
    useEffect(() => {
        if (mapRef.current && viewport && viewport.shouldSync) {
            if (viewport.bounds) {
                // Use fitBounds for route bounds
                mapRef.current.fitBounds(viewport.bounds, { animate: false, padding: [20, 20] })
            } else if (viewport.center && viewport.zoom !== undefined) {
                // Use setView for center/zoom
                const currentCenter = mapRef.current.getCenter()
                const currentZoom = mapRef.current.getZoom()

                // Only sync if the map is not already at the target position
                const centerDiff = Math.abs(currentCenter.lat - viewport.center[0]) + Math.abs(currentCenter.lng - viewport.center[1])
                const zoomDiff = Math.abs(currentZoom - viewport.zoom)

                if (centerDiff > 0.001 || zoomDiff > 0.1) {
                    mapRef.current.setView(viewport.center, viewport.zoom, { animate: false })
                }
            }
        }
    }, [viewport])    // Update user location
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
                <div class="font-sans max-w-xs p-3 bg-gradient-to-br from-princess-lavender via-princess-lilac to-princess-rose rounded-lg shadow-lg border border-princess-border">
                    <h3 class="font-semibold text-lg mb-1 text-princess-text-primary">Princess Location</h3>
                    <p class="text-sm text-princess-text-secondary">You are extremely nice.</p>
                </div>
            `, {
                className: 'custom-popup'
            })

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

    // Check location permission
    useEffect(() => {
        if ('permissions' in navigator) {
            navigator.permissions.query({ name: 'geolocation' }).then(result => {
                setLocationPermission(result.state as 'unknown' | 'granted' | 'denied')
                result.addEventListener('change', () => {
                    setLocationPermission(result.state as 'unknown' | 'granted' | 'denied')
                })
            }).catch(() => {
                setLocationPermission('denied')
            })
        } else {
            setLocationPermission('denied')
        }
    }, [])

    return (
        <div
            ref={mapContainerRef}
            className={`map-container ${className} relative`}
            style={{ height: '100%', width: '100%' }}
        >
            {/* Control buttons */}
            {locationPermission === 'granted' && (
                <div className="absolute top-4 right-4 z-[1000] flex flex-col space-y-2">
                    {/* Location button */}
                    <button
                        onClick={handleGoToUserLocation}
                        className="px-3 py-2 text-sm rounded-lg 
                                 bg-gradient-to-r from-princess-accent-lavender to-princess-accent-rose
                                 text-princess-text-primary shadow-lg hover:shadow-xl 
                                 transition-all duration-200 hover:scale-105 flex items-center space-x-1"
                        title="Go to your location"
                    >
                        <span>👸</span>
                    </button>

                    {/* Refresh button */}
                    {onRefresh && (
                        <button
                            onClick={onRefresh}
                            disabled={isLoading}
                            className={`px-3 py-2 text-sm rounded-lg 
                                     bg-gradient-to-r from-princess-accent-mint to-princess-accent-peach
                                     text-princess-text-primary shadow-lg hover:shadow-xl 
                                     transition-all duration-200 hover:scale-105 flex items-center justify-center
                                     ${isLoading ? 'opacity-50 cursor-not-allowed' : ''}`}
                            title={isLoading ? "Refreshing..." : "Refresh route"}
                        >
                            <span>{isLoading ? '⏳' : '🔄'}</span>
                        </button>
                    )}
                </div>
            )}

            {/* Location permission warning */}
            {showLocationWarning && (
                <div className="absolute inset-0 z-[1001] flex items-center justify-center pointer-events-none">
                    <div className="bg-gradient-to-br from-princess-lavender via-princess-lilac to-princess-rose 
                                  text-princess-text-primary px-6 py-4 rounded-lg shadow-xl border border-princess-border
                                  animate-fade-in font-dancing text-2xl">
                        You did not grant location permissions
                    </div>
                </div>
            )}
        </div>
    )
})

Map.displayName = 'Map'

// Traffic segment types
interface TrafficSegment {
    coordinates: [number, number][]
    speed: 'NORMAL' | 'SLOW' | 'TRAFFIC_JAM'
}

// Traffic utility functions
function buildTrafficSegments(
    encodedPolyline: string,
    intervals: Array<{
        startPolylinePointIndex: number
        endPolylinePointIndex: number
        speed: 'NORMAL' | 'SLOW' | 'TRAFFIC_JAM'
    }>
): TrafficSegment[] {
    const allCoords = decodePolyline(encodedPolyline)
    const segments: TrafficSegment[] = []

    intervals.forEach(interval => {
        const start = interval.startPolylinePointIndex
        const end = interval.endPolylinePointIndex
        const coords = allCoords.slice(start, end + 1)
        if (coords.length > 1) {
            segments.push({
                coordinates: coords,
                speed: interval.speed
            })
        }
    })

    return segments
}

function drawTrafficSegments(segments: TrafficSegment[], routeLayer: L.LayerGroup): void {
    segments.forEach(segment => {
        if (segment.coordinates.length < 2) return

        let color: string
        switch (segment.speed) {
            case 'NORMAL':
                color = '#34D399' // Emerald 400 - green for normal speed
                break
            case 'SLOW':
                color = '#FBBF24' // Amber 400 - yellow for slow traffic
                break
            case 'TRAFFIC_JAM':
                color = '#F87171' // Red 400 - red for traffic jams
                break
            default:
                color = '#60A5FA' // Blue 400 - fallback
                break
        }

        const trafficPolyline = L.polyline(segment.coordinates, {
            color: color,
            weight: 6,
            opacity: 0.85
        })

        routeLayer.addLayer(trafficPolyline)
    })
}

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
