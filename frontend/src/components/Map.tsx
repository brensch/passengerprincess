import { useEffect, useRef, forwardRef, useImperativeHandle, useState, useCallback } from 'react'
import * as L from 'leaflet'
import 'leaflet.markercluster/dist/MarkerCluster.css'
import 'leaflet.markercluster/dist/MarkerCluster.Default.css'
import 'leaflet.markercluster'
import { RouteResponse, StationData, SearchFilters, ViewportResponse, Supercharger, Restaurant, RestaurantSuperchargerMapping } from '../types'
import { useViewport } from '../contexts/ViewportContext'
import { getRestaurantEmoji } from '../utils/restaurantEmojiMapping'
import PopupContent from './PopupContent'
import { createRoot } from 'react-dom/client'

const globalViewportData = {
    mappings: [] as RestaurantSuperchargerMapping[],
    listeners: [] as (() => void)[]
}

export { globalViewportData }

const popupWidth = '200px'
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
        zoomToSupercharger?: (lat: number, lng: number) => void
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
    const [isViewportLoading, setIsViewportLoading] = useState(false)
    const lastBoundsRef = useRef<string | null>(null)
    const viewportSuperchargers = useRef<Map<string, ViewportSuperchargerData>>(new globalThis.Map())
    const viewportRestaurants = useRef<Map<string, { data: Restaurant, marker: L.Marker }>>(new globalThis.Map())
    const [viewportMappings, setViewportMappings] = useState<RestaurantSuperchargerMapping[]>([])
    const { viewport } = useViewport()

    // Caching state
    const [allSuperchargers, setAllSuperchargers] = useState<Supercharger[]>([])
    const [allRestaurants, setAllRestaurants] = useState<Restaurant[]>([])
    const [allMappings, setAllMappings] = useState<RestaurantSuperchargerMapping[]>([])
    const [coveredBounds, setCoveredBounds] = useState<{ minLat: number, maxLat: number, minLng: number, maxLng: number } | null>(null)

    // Warning popup state
    const [showLocationWarning, setShowLocationWarning] = useState(false)
    const [locationPermission, setLocationPermission] = useState<'unknown' | 'granted' | 'denied'>('unknown')
    const [showPermissionModal, setShowPermissionModal] = useState(false)

    const handleRefresh = useCallback(() => {
        if (locationPermission !== 'granted') {
            setShowPermissionModal(true)
            setTimeout(() => setShowPermissionModal(false), 3000)
            return
        }
        if (onRefresh) onRefresh()
    }, [onRefresh, locationPermission])

    const handleGoToUserLocation = useCallback(() => {
        if (locationPermission !== 'granted') {
            setShowPermissionModal(true)
            setTimeout(() => setShowPermissionModal(false), 3000)
            return
        }
        if (userLocation && mapRef.current) {
            mapRef.current.setView(userLocation, 15, { animate: true })
        } else {
            setShowLocationWarning(true)
            setTimeout(() => setShowLocationWarning(false), 2000)
        }
    }, [userLocation, locationPermission])

    const generateSuperchargerPopupContent = useCallback((supercharger: Supercharger, chargerId?: string) => {
        const routeStation = stationData.find(s => s.chargerInfo.supercharger.place_id === supercharger.place_id)
        const finalChargerId = chargerId || routeStation?.id
        const viewportSuperchargerData = viewportSuperchargers.current.get(supercharger.place_id)
        let etaInfo = viewportSuperchargerData?.etaInfo

        // If no etaInfo from viewport, try to get from routeData
        if (!etaInfo && routeData) {
            const routeSupercharger = routeData.superchargers.find(s => s.supercharger.place_id === supercharger.place_id)
            if (routeSupercharger) {
                etaInfo = {
                    arrival_time: routeSupercharger.arrival_time,
                    distance_from_route: routeStation?.chargerInfo.distance_from_route || 0,
                    distance_along_route: routeStation?.chargerInfo.distance_along_route || 0
                }
            }
        }

        const container = document.createElement('div')
        container.style.width = popupWidth
        container.style.maxWidth = popupWidth
        const root = createRoot(container)
        root.render(
            <PopupContent
                type="supercharger"
                supercharger={supercharger}
                etaInfo={etaInfo}
                chargerId={finalChargerId}
                onViewInResults={finalChargerId ? (id) => window.showChargerInResults?.(id) : undefined}
                onZoom={(lat, lng) => window.zoomToSupercharger?.(lat, lng)}
            />
        )
        return container
    }, [stationData, routeData])

    const generateRestaurantPopupContent = useCallback((restaurant: Restaurant) => {
        const mapping = viewportMappings.find(m => m.restaurant_id === restaurant.place_id)
        let distanceText = ''
        if (mapping) {
            const walkingDistance = (mapping.distance / 1609.34 * 5280).toFixed(0) // Convert miles to feet
            distanceText = `Walking: ${walkingDistance} ft to supercharger`
        }

        const container = document.createElement('div')
        container.style.width = popupWidth
        container.style.maxWidth = popupWidth
        const root = createRoot(container)
        root.render(
            <PopupContent
                type="restaurant"
                restaurant={restaurant}
                distanceText={distanceText || undefined}
            />
        )
        return container
    }, [viewportMappings, stationData])

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
                L.popup({ className: 'custom-popup', maxWidth: 400 })
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
                    L.popup({ className: 'custom-popup', maxWidth: 400 })
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
            tap: false,
            maxBounds: [[-90, -180], [90, 180]],
            maxBoundsViscosity: 1.0,
            // Additional mobile-specific options to prevent fullscreen behavior
            dragging: true,
            doubleClickZoom: false,
            boxZoom: false,
            keyboard: false,
            scrollWheelZoom: true,
            // Prevent tap from triggering browser UI changes
            tapTolerance: 15
        } as any).setView([37.7749, -122.4194], 6)

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

        const viewportRestaurantsLayer = L.layerGroup().addTo(map)

        layersRef.current = {
            route: L.layerGroup().addTo(map),
            userLocation: L.layerGroup().addTo(map),
            viewportSuperchargers: viewportSuperchargersLayer.addTo(map),
            viewportRestaurants: viewportRestaurantsLayer
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
        const allRestaurantNames = new Set(viewportData.restaurants?.map(r => r.name) || [])
        const filteredRestaurants = viewportData.restaurants?.filter(restaurant => {
            const nameMatch = searchFilters.selectedPlaces.length > 0 ? searchFilters.selectedPlaces.includes(restaurant.name) : (searchFilters.typedPlace === '' || (searchFilters.typedPlace && !allRestaurantNames.has(searchFilters.typedPlace) && restaurant.name.toLowerCase().includes(searchFilters.typedPlace.toLowerCase())))
            const cuisineMatch = searchFilters.selectedCuisines.length === 0 || searchFilters.selectedCuisines.some(cuisine =>
                (restaurant.primary_type_display || '').toLowerCase().includes(cuisine.toLowerCase())
            )
            return nameMatch && cuisineMatch
        }) || []

        const hasActiveFilters = searchFilters.selectedPlaces.length > 0 || searchFilters.typedPlace !== '' || searchFilters.selectedCuisines.length > 0

        const targetVisibleSuperchargerIds = new Set<string>()
        if (hasActiveFilters) {
            const filteredRestaurantIds = new Set(filteredRestaurants.map(r => r.place_id))
            const linkedSuperchargerIds = new Set<string>()
            viewportData.mappings?.forEach(mapping => {
                if (filteredRestaurantIds.has(mapping.restaurant_id)) {
                    linkedSuperchargerIds.add(mapping.supercharger_id)
                }
            })

            viewportData.superchargers?.forEach(sc => {
                if (bounds.contains(L.latLng(sc.latitude, sc.longitude)) && linkedSuperchargerIds.has(sc.place_id)) {
                    targetVisibleSuperchargerIds.add(sc.place_id)
                }
            })
        } else {
            viewportData.superchargers?.forEach(sc => {
                if (bounds.contains(L.latLng(sc.latitude, sc.longitude))) {
                    targetVisibleSuperchargerIds.add(sc.place_id)
                }
            })
        }

        const targetVisibleRestaurantIds = new Set<string>()
        if ((viewportData.restaurants?.length || 0) <= 200) {
            filteredRestaurants.forEach(r => {
                if (bounds.contains(L.latLng(r.latitude, r.longitude))) {
                    targetVisibleRestaurantIds.add(r.place_id)
                }
            })
        }

        // --- 2. Create and cache any markers that don't exist yet ---
        const routeSuperchargerMap = new globalThis.Map(routeData?.superchargers.map(rs => [rs.supercharger.place_id, rs]) || [])

        viewportData.superchargers?.forEach(supercharger => {
            if (!viewportSuperchargers.current.has(supercharger.place_id)) {
                const isRouteCharger = routeSuperchargerMap.has(supercharger.place_id)
                const marker = L.marker([supercharger.latitude, supercharger.longitude], {
                    icon: L.divIcon({
                        className: 'emoji-icon charger-icon',
                        html: '🔌',
                        iconSize: isRouteCharger ? [36, 36] : [30, 30],
                        iconAnchor: isRouteCharger ? [18, 18] : [15, 15]
                    })
                })
                const routeStation = stationData.find(s => s.chargerInfo.supercharger.place_id === supercharger.place_id)
                marker.bindPopup(() => generateSuperchargerPopupContent(supercharger, routeStation?.id), { className: 'custom-popup', maxWidth: 400 })

                viewportSuperchargers.current.set(supercharger.place_id, {
                    data: supercharger,
                    marker,
                    etaInfo: routeSuperchargerMap.get(supercharger.place_id)
                })
            }
        })

        viewportData.restaurants?.forEach(restaurant => {
            if (!viewportRestaurants.current.has(restaurant.place_id)) {
                const routeRestaurant = stationData.some(s => s.restaurants?.some(r => r.place_id === restaurant.place_id))
                const marker = L.marker([restaurant.latitude, restaurant.longitude], {
                    icon: L.divIcon({
                        className: 'emoji-icon restaurant-icon',
                        html: getRestaurantEmoji(restaurant.primary_type),
                        iconSize: routeRestaurant ? [24, 24] : [20, 20],
                        iconAnchor: routeRestaurant ? [12, 12] : [10, 10]
                    })
                })
                marker.bindPopup(() => generateRestaurantPopupContent(restaurant), { className: 'custom-popup', maxWidth: 400 })
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

        restToRemoveIds.forEach(id => {
            const restaurantData = viewportRestaurants.current.get(id)
            if (restaurantData) rLayer.removeLayer(restaurantData.marker)
        })

    }, [viewportData, searchFilters, stationData, routeData, generateSuperchargerPopupContent, generateRestaurantPopupContent])

    // Effect to run the unified marker update logic
    useEffect(() => {
        updateMarkers()
    }, [updateMarkers])

    // Helper functions for caching
    const isBoundsWithinCovered = (bounds: L.LatLngBounds, covered: { minLat: number, maxLat: number, minLng: number, maxLng: number }) => {
        const { minLat, maxLat, minLng, maxLng } = covered
        return bounds.getSouth() >= minLat && bounds.getNorth() <= maxLat && bounds.getWest() >= minLng && bounds.getEast() <= maxLng
    }

    const filterDataByBounds = (bounds: L.LatLngBounds) => {
        const minLat = bounds.getSouth()
        const maxLat = bounds.getNorth()
        const minLng = bounds.getWest()
        const maxLng = bounds.getEast()

        const filteredSuperchargers = allSuperchargers.filter(sc =>
            sc.latitude >= minLat && sc.latitude <= maxLat && sc.longitude >= minLng && sc.longitude <= maxLng
        )
        const filteredRestaurants = allRestaurants.filter(r =>
            r.latitude >= minLat && r.latitude <= maxLat && r.longitude >= minLng && r.longitude <= maxLng
        )
        const superchargerIds = new Set(filteredSuperchargers.map(sc => sc.place_id))
        const filteredMappings = allMappings.filter(m => superchargerIds.has(m.supercharger_id))

        return {
            superchargers: filteredSuperchargers,
            restaurants: filteredRestaurants,
            mappings: filteredMappings
        }
    }

    const mergeData = (newData: ViewportResponse) => {
        setAllSuperchargers(prev => {
            const existingIds = new Set(prev.map(sc => sc.place_id))
            const newOnes = newData.superchargers?.filter(sc => !existingIds.has(sc.place_id)) || []
            return [...prev, ...newOnes]
        })
        setAllRestaurants(prev => {
            const existingIds = new Set(prev.map(r => r.place_id))
            const newOnes = newData.restaurants?.filter(r => !existingIds.has(r.place_id)) || []
            return [...prev, ...newOnes]
        })
        setAllMappings(prev => {
            const existingKeys = new Set(prev.map(m => `${m.restaurant_id}-${m.supercharger_id}`))
            const newOnes = newData.mappings?.filter(m => !existingKeys.has(`${m.restaurant_id}-${m.supercharger_id}`)) || []
            return [...prev, ...newOnes]
        })
    }

    const updateCoveredBounds = (newBounds: L.LatLngBounds) => {
        setCoveredBounds(prev => {
            if (!prev) {
                return {
                    minLat: newBounds.getSouth(),
                    maxLat: newBounds.getNorth(),
                    minLng: newBounds.getWest(),
                    maxLng: newBounds.getEast()
                }
            }
            return {
                minLat: Math.min(prev.minLat, newBounds.getSouth()),
                maxLat: Math.max(prev.maxLat, newBounds.getNorth()),
                minLng: Math.min(prev.minLng, newBounds.getWest()),
                maxLng: Math.max(prev.maxLng, newBounds.getEast())
            }
        })
    }

    // Handle viewport data fetching
    const fetchViewportData = useCallback(async () => {
        if (!mapRef.current) return

        const bounds = mapRef.current.getBounds()
        const boundsStr = bounds.toBBoxString()
        if (lastBoundsRef.current === boundsStr) return
        lastBoundsRef.current = boundsStr

        // Check if current bounds are within covered bounds
        if (coveredBounds && isBoundsWithinCovered(bounds, coveredBounds)) {
            const filteredData = filterDataByBounds(bounds)
            setViewportData(filteredData)
            setViewportMappings(filteredData.mappings)
            globalViewportData.mappings = filteredData.mappings
            globalViewportData.listeners.forEach(l => l())
            return
        }

        setIsViewportLoading(true)
        const [minLng, minLat, maxLng, maxLat] = boundsStr.split(',').map(parseFloat)
        try {
            const response = await fetch(
                `/superchargers/viewport?min_lat=${minLat}&max_lat=${maxLat}&min_lng=${minLng}&max_lng=${maxLng}`
            )
            const data: ViewportResponse = await response.json()
            if (response.ok) {
                mergeData(data)
                updateCoveredBounds(bounds)
                setViewportData(data)
                setViewportMappings(data.mappings || [])
                globalViewportData.mappings = data.mappings || []
                globalViewportData.listeners.forEach(l => l())
            }
        } catch (error) {
            console.error("Failed to load viewport data:", error)
        } finally {
            setIsViewportLoading(false)
        }
    }, [coveredBounds, allSuperchargers, allRestaurants, allMappings])

    // Set up viewport update handlers
    useEffect(() => {
        const map = mapRef.current
        if (!map) return

        const debouncedViewportUpdate = () => {
            fetchViewportData()
        }

        map.on('moveend zoomend', debouncedViewportUpdate)
        setTimeout(fetchViewportData, 100) // Initial load

        return () => {
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
            {isViewportLoading && (
                <div className="absolute top-4 left-4 z-[1000] flex items-center space-x-2">
                    <span className="text-2xl animate-pulse">🧠</span>
                </div>
            )}
            <div className="absolute top-4 right-4 z-[1000] flex flex-col space-y-2">
                <button onClick={handleGoToUserLocation} className="px-3 py-2 text-sm rounded-lg bg-gradient-to-r from-princess-accent-lavender to-princess-accent-rose text-princess-text-primary shadow-lg hover:shadow-xl transition-all duration-200 hover:scale-105 flex items-center space-x-1" title="Go to your location">
                    <span>👸</span>
                </button>
                {onRefresh && (
                    <button onClick={handleRefresh} disabled={isLoading} className={`px-3 py-2 text-sm rounded-lg bg-gradient-to-r from-princess-accent-mint to-princess-accent-peach text-princess-text-primary shadow-lg hover:shadow-xl transition-all duration-200 hover:scale-105 flex items-center justify-center ${isLoading ? 'opacity-50 cursor-not-allowed' : ''}`} title={isLoading ? "Refreshing..." : "Refresh route"}>
                        <span>{isLoading ? '⏳' : '🔄'}</span>
                    </button>
                )}
            </div>
            {showLocationWarning && (
                <div className="absolute inset-0 z-[1001] flex items-center justify-center pointer-events-none">
                    <div className="bg-gradient-to-br from-princess-lavender via-princess-lilac to-princess-rose text-princess-text-primary px-6 py-4 rounded-lg shadow-xl border border-princess-border animate-fade-in font-dancing text-2xl">
                        You did not grant location permissions
                    </div>
                </div>
            )}
            {showPermissionModal && (
                <div className="absolute inset-0 z-[1001] flex items-center justify-center pointer-events-none">
                    <div className="bg-gradient-to-br from-princess-lavender via-princess-lilac to-princess-rose text-princess-text-primary px-6 py-4 rounded-lg shadow-xl border border-princess-border animate-fade-in font-dancing text-3xl max-w-[80vw]">
                        I can't find your location. Enable location permissions please Princess
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