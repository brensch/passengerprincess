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
    const viewportSuperchargers = useRef<Map<string, ViewportSuperchargerData>>(new globalThis.Map())
    const viewportRestaurants = useRef<Map<string, { data: Restaurant, marker: L.Marker }>>(new globalThis.Map())
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
        const routeStation = stationData.find(s => s.chargerInfo.supercharger.place_id === supercharger.place_id)
        const finalChargerId = chargerId || routeStation?.id
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
            content += `<button onclick="showChargerInResults('${finalChargerId}')" class="px-3 py-1.5 bg-gradient-to-r from-purple-500 to-purple-600 text-white rounded-md text-sm font-medium transition-all duration-200 hover:from-purple-600 hover:to-purple-700 hover:shadow-md">View in Results</button>`
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
        const mapping = viewportMappings.current.find(m => m.restaurant_id === restaurant.place_id)
        let distanceText = ''
        if (mapping) {
            const walkingDistance = (mapping.distance / 1609.34 * 5280).toFixed(0) // Convert miles to feet
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
            mapRef.current?.fitBounds(bounds)
        },
        getMap: () => mapRef.current,
        showSuperchargerPopup: (placeId: string) => {
            const map = mapRef.current
            if (!map) return

            const viewportMarkerData = viewportSuperchargers.current.get(placeId)
            if (viewportMarkerData?.marker) {
                viewportMarkerData.marker.openPopup()
                return
            }

            const station = stationData.find(s => s.chargerInfo.supercharger.place_id === placeId)
            if (station) {
                const supercharger = station.chargerInfo.supercharger
                L.popup({ className: 'custom-popup' })
                    .setLatLng([supercharger.latitude, supercharger.longitude])
                    .setContent(generateSuperchargerPopupContent(supercharger, station.id))
                    .openOn(map)
            }
        },
        showRestaurantPopup: (placeId: string) => {
            const map = mapRef.current
            if (!map) return

            const viewportMarkerData = viewportRestaurants.current.get(placeId)
            if (viewportMarkerData?.marker) {
                viewportMarkerData.marker.openPopup()
                return
            }

            for (const station of stationData) {
                const restaurant = station.restaurants?.find(r => r.place_id === placeId)
                if (restaurant) {
                    L.popup({ className: 'custom-popup' })
                        .setLatLng([restaurant.latitude, restaurant.longitude])
                        .setContent(generateRestaurantPopupContent(restaurant))
                        .openOn(map)
                    return
                }
            }
        }
    }))

    // Initialize map
    useEffect(() => {
        if (!mapContainerRef.current || mapRef.current) return

        const map = L.map(mapContainerRef.current, {
            zoomControl: false,
            touchZoom: 'center',
            maxBounds: [[-90, -180], [90, 180]],
            maxBoundsViscosity: 1.0
        }).setView([37.7749, -122.4194], 6)

        L.tileLayer('https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors',
            maxZoom: 19
        }).addTo(map)

        const viewportSuperchargersLayer = L.markerClusterGroup({
            maxClusterRadius: 50,
            disableClusteringAtZoom: 15,
            spiderfyOnMaxZoom: true,
            showCoverageOnHover: false,
            iconCreateFunction: function (cluster) {
                const childCount = cluster.getChildCount()
                let size = 40 + Math.floor(childCount / 50) * 10
                return L.divIcon({
                    html: `<div style="background-color: #FFB3C6; border-radius: 50%; display: flex; align-items: center; justify-content: center; color: #6B4D7C; font-weight: bold; font-size: 12px;">${childCount}</div>`,
                    className: 'custom-cluster-icon',
                    iconSize: [size, size],
                    iconAnchor: [size / 2, size / 2]
                })
            }
        })

        layersRef.current = {
            route: L.layerGroup().addTo(map),
            userLocation: L.layerGroup().addTo(map),
            viewportSuperchargers: viewportSuperchargersLayer.addTo(map),
            viewportRestaurants: L.layerGroup().addTo(map)
        }
        mapRef.current = map

        return () => {
            map.remove()
            mapRef.current = null
        }
    }, [])

    // --- Unified Marker Update Logic ---
    const updateMarkers = useCallback(() => {
        if (!mapRef.current || !layersRef.current || !viewportData) return

        const { viewportSuperchargers: scLayer, viewportRestaurants: rLayer } = layersRef.current
        const bounds = mapRef.current.getBounds()

        // --- 1. Determine Target Visibility from current data and filters ---
        const filteredRestaurants = viewportData.restaurants.filter(restaurant =>
            (!searchFilters.searchTerm || restaurant.name.toLowerCase().includes(searchFilters.searchTerm.toLowerCase())) &&
            (searchFilters.cuisineFilters.length === 0 || searchFilters.cuisineFilters.includes('') || searchFilters.cuisineFilters.some(cuisine =>
                cuisine !== '' && (restaurant.primary_type_display || '').toLowerCase().includes(cuisine.toLowerCase())
            ))
        )

        const hasActiveFilters = searchFilters.searchTerm !== '' || (searchFilters.cuisineFilters.length > 0 && !searchFilters.cuisineFilters.includes(''))

        const targetVisibleSuperchargerIds = new Set<string>()
        if (hasActiveFilters) {
            const filteredRestaurantIds = new Set(filteredRestaurants.map(r => r.place_id))
            const linkedSuperchargerIds = new Set<string>()
            viewportData.mappings.forEach(mapping => {
                if (filteredRestaurantIds.has(mapping.restaurant_id)) {
                    linkedSuperchargerIds.add(mapping.supercharger_id)
                }
            })
            const routeSuperchargerIds = new Set(stationData.map(s => s.chargerInfo.supercharger.place_id))

            viewportData.superchargers.forEach(sc => {
                if (bounds.contains(L.latLng(sc.latitude, sc.longitude)) && (linkedSuperchargerIds.has(sc.place_id) || routeSuperchargerIds.has(sc.place_id))) {
                    targetVisibleSuperchargerIds.add(sc.place_id)
                }
            })
        } else {
            viewportData.superchargers.forEach(sc => {
                if (bounds.contains(L.latLng(sc.latitude, sc.longitude))) {
                    targetVisibleSuperchargerIds.add(sc.place_id)
                }
            })
        }

        const targetVisibleRestaurantIds = new Set<string>()
        if (filteredRestaurants.length <= 100) { // Performance guard
            filteredRestaurants.forEach(r => {
                if (bounds.contains(L.latLng(r.latitude, r.longitude))) {
                    targetVisibleRestaurantIds.add(r.place_id)
                }
            })
        }

        // --- 2. Create and cache any markers that don't exist yet ---
        const routeSuperchargerMap = new globalThis.Map(routeData?.superchargers.map(rs => [rs.supercharger.place_id, rs]) || [])

        viewportData.superchargers.forEach(supercharger => {
            if (!viewportSuperchargers.current.has(supercharger.place_id)) {
                const isRouteCharger = routeSuperchargerMap.has(supercharger.place_id)
                const marker = L.marker([supercharger.latitude, supercharger.longitude], {
                    icon: L.divIcon({
                        className: 'emoji-icon charger-icon',
                        html: '⚡',
                        iconSize: isRouteCharger ? [36, 36] : [30, 30],
                        iconAnchor: isRouteCharger ? [18, 18] : [15, 15]
                    })
                })
                const routeStation = stationData.find(s => s.chargerInfo.supercharger.place_id === supercharger.place_id)
                marker.bindPopup(() => generateSuperchargerPopupContent(supercharger, routeStation?.id), { className: 'custom-popup' })

                viewportSuperchargers.current.set(supercharger.place_id, {
                    data: supercharger,
                    marker,
                    etaInfo: routeSuperchargerMap.get(supercharger.place_id)
                })
            }
        })

        viewportData.restaurants.forEach(restaurant => {
            if (!viewportRestaurants.current.has(restaurant.place_id)) {
                const routeRestaurant = stationData.some(s => s.restaurants.some(r => r.place_id === restaurant.place_id))
                const marker = L.marker([restaurant.latitude, restaurant.longitude], {
                    icon: L.divIcon({
                        className: 'emoji-icon restaurant-icon',
                        html: getRestaurantEmoji(restaurant.primary_type),
                        iconSize: routeRestaurant ? [24, 24] : [20, 20],
                        iconAnchor: routeRestaurant ? [12, 12] : [10, 10]
                    })
                })
                marker.bindPopup(() => generateRestaurantPopupContent(restaurant), { className: 'custom-popup' })
                viewportRestaurants.current.set(restaurant.place_id, { data: restaurant, marker })
            }
        })

        // --- 3. Calculate deltas and update layers efficiently ---
        const currentlyVisibleScIds = new Set<string>()
        for (const [id, { marker }] of viewportSuperchargers.current.entries()) {
            if (scLayer.hasLayer(marker)) {
                currentlyVisibleScIds.add(id)
            }
        }

        const scToAddIds = [...targetVisibleSuperchargerIds].filter(id => !currentlyVisibleScIds.has(id))
        const scToRemoveIds = [...currentlyVisibleScIds].filter(id => !targetVisibleSuperchargerIds.has(id))

        const scMarkersToAdd = scToAddIds.map(id => viewportSuperchargers.current.get(id)?.marker).filter((m): m is L.Marker => !!m)
        const scMarkersToRemove = scToRemoveIds.map(id => viewportSuperchargers.current.get(id)?.marker).filter((m): m is L.Marker => !!m)

        if (scMarkersToAdd.length > 0) scLayer.addLayers(scMarkersToAdd)
        if (scMarkersToRemove.length > 0) scLayer.removeLayers(scMarkersToRemove)

        const currentlyVisibleRestIds = new Set<string>()
        for (const [id, { marker }] of viewportRestaurants.current.entries()) {
            if (rLayer.hasLayer(marker)) {
                currentlyVisibleRestIds.add(id)
            }
        }

        const restToAddIds = [...targetVisibleRestaurantIds].filter(id => !currentlyVisibleRestIds.has(id))
        const restToRemoveIds = [...currentlyVisibleRestIds].filter(id => !targetVisibleRestaurantIds.has(id))

        restToAddIds.forEach(id => {
            const restaurantData = viewportRestaurants.current.get(id)
            if (restaurantData) restaurantData.marker.addTo(rLayer)
        })

        // *** THIS IS THE FIX ***
        restToRemoveIds.forEach(id => {
            const restaurantData = viewportRestaurants.current.get(id)
            if (restaurantData) rLayer.removeLayer(restaurantData.marker)
        })

    }, [viewportData, searchFilters, stationData, routeData, generateSuperchargerPopupContent, generateRestaurantPopupContent])

    // Effect to run the unified marker update logic
    useEffect(() => {
        updateMarkers()
    }, [updateMarkers])

    // Handle viewport data fetching
    const fetchViewportData = useCallback(async () => {
        if (!mapRef.current) return

        const bounds = mapRef.current.getBounds()
        const boundsStr = bounds.toBBoxString()
        if (lastBoundsRef.current === boundsStr) return
        lastBoundsRef.current = boundsStr

        const [minLng, minLat, maxLng, maxLat] = boundsStr.split(',').map(parseFloat)
        try {
            const response = await fetch(
                `/superchargers/viewport?min_lat=${minLat}&max_lat=${maxLat}&min_lng=${minLng}&max_lng=${maxLng}`
            )
            const data: ViewportResponse = await response.json()
            if (response.ok && mapRef.current?.getBounds().toBBoxString() === boundsStr) {
                setViewportData(data)
                viewportMappings.current = data.mappings || []
            }
        } catch (error) {
            console.error("Failed to load viewport data:", error)
        }
    }, [])

    // Set up viewport update handlers
    useEffect(() => {
        const map = mapRef.current
        if (!map) return

        let viewportTimeout: ReturnType<typeof setTimeout>
        const debouncedViewportUpdate = () => {
            clearTimeout(viewportTimeout)
            viewportTimeout = setTimeout(fetchViewportData, 300)
        }
        map.on('moveend zoomend', debouncedViewportUpdate)
        setTimeout(fetchViewportData, 100) // Initial load

        return () => {
            clearTimeout(viewportTimeout)
            map.off('moveend zoomend', debouncedViewportUpdate)
        }
    }, [fetchViewportData])

    // Update route polyline
    useEffect(() => {
        const routeLayer = layersRef.current?.route
        const map = mapRef.current
        if (!routeData || !routeLayer || !map) return

        routeLayer.clearLayers()
        if (routeData.route?.EncodedPolyline) {
            const coordinates = decodePolyline(routeData.route.EncodedPolyline)
            if (coordinates.length > 0) {
                const basePolyline = L.polyline(coordinates, { color: '#8B5CF6', weight: 5, opacity: 0.8 })
                routeLayer.addLayer(basePolyline)

                if (routeData.route.travelAdvisory?.speedReadingIntervals) {
                    const trafficSegments = buildTrafficSegments(routeData.route.EncodedPolyline, routeData.route.travelAdvisory.speedReadingIntervals)
                    drawTrafficSegments(trafficSegments, routeLayer)
                }
            }
        }
    }, [routeData])

    // Sync map view with external viewport context
    useEffect(() => {
        if (mapRef.current && viewport && viewport.shouldSync) {
            if (viewport.bounds) {
                mapRef.current.fitBounds(viewport.bounds, { animate: false, padding: [20, 20] })
            } else if (viewport.center && viewport.zoom !== undefined) {
                mapRef.current.setView(viewport.center, viewport.zoom, { animate: false })
            }
        }
    }, [viewport])

    // Update user location marker
    useEffect(() => {
        const userLocationLayer = layersRef.current?.userLocation
        if (!userLocationLayer) return

        userLocationLayer.clearLayers()
        if (userLocation) {
            const userMarker = L.marker(userLocation, {
                icon: L.divIcon({
                    className: 'princess-marker',
                    html: '👸',
                    iconSize: [36, 36],
                    iconAnchor: [18, 18]
                }),
                zIndexOffset: 10000
            }).bindPopup(`
                <div class="font-sans max-w-xs p-3 bg-gradient-to-br from-princess-lavender via-princess-lilac to-princess-rose rounded-lg shadow-lg border border-princess-border">
                    <h3 class="font-semibold text-lg mb-1 text-princess-text-primary">Princess Location</h3>
                    <p class="text-sm text-princess-text-secondary">You are extremely nice.</p>
                </div>
            `, { className: 'custom-popup' })
            userLocationLayer.addLayer(userMarker)
        }
    }, [userLocation])

    // Handle map resize
    useEffect(() => {
        if (mapRef.current) {
            setTimeout(() => mapRef.current?.invalidateSize(), 100)
        }
    }, [className])

    // Check location permission
    useEffect(() => {
        if ('permissions' in navigator) {
            navigator.permissions.query({ name: 'geolocation' }).then(result => {
                setLocationPermission(result.state as 'granted' | 'denied')
                result.onchange = () => setLocationPermission(result.state as 'granted' | 'denied')
            }).catch(() => setLocationPermission('denied'))
        } else {
            setLocationPermission('denied')
        }
    }, [])

    return (
        <div ref={mapContainerRef} className={`map-container ${className} relative`} style={{ height: '100%', width: '100%' }}>
            {locationPermission === 'granted' && (
                <div className="absolute top-4 right-4 z-[1000] flex flex-col space-y-2">
                    <button onClick={handleGoToUserLocation} className="px-3 py-2 text-sm rounded-lg bg-gradient-to-r from-princess-accent-lavender to-princess-accent-rose text-princess-text-primary shadow-lg hover:shadow-xl transition-all duration-200 hover:scale-105 flex items-center space-x-1" title="Go to your location">
                        <span>👸</span>
                    </button>
                    {onRefresh && (
                        <button onClick={onRefresh} disabled={isLoading} className={`px-3 py-2 text-sm rounded-lg bg-gradient-to-r from-princess-accent-mint to-princess-accent-peach text-princess-text-primary shadow-lg hover:shadow-xl transition-all duration-200 hover:scale-105 flex items-center justify-center ${isLoading ? 'opacity-50 cursor-not-allowed' : ''}`} title={isLoading ? "Refreshing..." : "Refresh route"}>
                            <span>{isLoading ? '⏳' : '🔄'}</span>
                        </button>
                    )}
                </div>
            )}
            {showLocationWarning && (
                <div className="absolute inset-0 z-[1001] flex items-center justify-center pointer-events-none">
                    <div className="bg-gradient-to-br from-princess-lavender via-princess-lilac to-princess-rose text-princess-text-primary px-6 py-4 rounded-lg shadow-xl border border-princess-border animate-fade-in font-dancing text-2xl">
                        You did not grant location permissions
                    </div>
                </div>
            )}
        </div>
    )
})

Map.displayName = 'Map'

// --- Utility Functions (unchanged) ---

interface TrafficSegment {
    coordinates: [number, number][]
    speed: 'NORMAL' | 'SLOW' | 'TRAFFIC_JAM'
}

function buildTrafficSegments(
    encodedPolyline: string,
    intervals: Array<{ startPolylinePointIndex: number; endPolylinePointIndex: number; speed: 'NORMAL' | 'SLOW' | 'TRAFFIC_JAM' }>
): TrafficSegment[] {
    const allCoords = decodePolyline(encodedPolyline)
    return intervals.map(interval => ({
        coordinates: allCoords.slice(interval.startPolylinePointIndex, interval.endPolylinePointIndex + 1),
        speed: interval.speed
    })).filter(segment => segment.coordinates.length > 1)
}

function drawTrafficSegments(segments: TrafficSegment[], routeLayer: L.LayerGroup): void {
    const colorMap = {
        NORMAL: '#34D399',      // green
        SLOW: '#FBBF24',        // yellow
        TRAFFIC_JAM: '#F87171'  // red
    }
    segments.forEach(segment => {
        const color = colorMap[segment.speed] || '#60A5FA' // fallback blue
        const trafficPolyline = L.polyline(segment.coordinates, { color, weight: 6, opacity: 0.85 })
        routeLayer.addLayer(trafficPolyline)
    })
}

function decodePolyline(encoded: string): [number, number][] {
    const coords: [number, number][] = []
    let index = 0, lat = 0, lng = 0

    while (index < encoded.length) {
        let b, shift = 0, result = 0
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