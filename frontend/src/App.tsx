import { useState, useEffect, useRef } from 'react'
import Map, { MapRef } from './components/Map'
import SearchForm from './components/SearchForm'
import ResultsTable from './components/ResultsTable'
import TopToolbar from './components/TopToolbar'
import FilterModal from './components/FilterModal'
import HelpModal from './components/HelpModal'
import { ViewportProvider, useViewport } from './contexts/ViewportContext'
import { ViewMode } from './types'
import { useRoute } from './hooks/useRoute'
import { useFilters } from './hooks/useFilters'
import * as L from 'leaflet'

const AppContent = () => {
    const [viewMode, setViewMode] = useState<ViewMode>('search')
    const [isFilterModalOpen, setIsFilterModalOpen] = useState(false)
    const [isHelpModalOpen, setIsHelpModalOpen] = useState(false)
    const [userLocation, setUserLocation] = useState<[number, number] | null>(null)

    const mapRef = useRef<MapRef>(null)
    const { routeData, stationData, isLoading, error, searchRoute, clearRoute } = useRoute()
    const {
        searchFilters,
        filteredStationData,
        searchTerm,
        cuisineFilters,
        updateFilters,
        filterCount
    } = useFilters(stationData)
    const { setViewportToLocation, setViewportToRoute, updateViewport, restoreSavedViewport } = useViewport()

    console.log('App render - viewMode:', viewMode, 'routeData:', !!routeData)

    // Add global function for popup buttons
    useEffect(() => {
        window.showChargerInResults = (chargerId: string) => {
            setViewMode('results')

            // Wait for the view to update, then scroll to the charger
            setTimeout(() => {
                // Find all rows for this charger (including all restaurant rows)
                const targetRows = document.querySelectorAll(`tr[data-charger-id="${chargerId}"]`) as NodeListOf<HTMLElement>
                if (targetRows.length > 0) {
                    // Highlight all rows for this charger
                    targetRows.forEach(row => {
                        row.style.backgroundColor = '#fbbf24' // amber-400
                        row.style.transition = 'background-color 0.3s ease'
                    })

                    // Jump directly to the first row (no smooth scrolling)
                    targetRows[0].scrollIntoView({
                        behavior: 'auto',
                        block: 'center'
                    })

                    // Remove highlight after 2 seconds
                    setTimeout(() => {
                        targetRows.forEach(row => {
                            row.style.backgroundColor = ''
                        })
                    }, 2000)
                }
            }, 100)
        }

        return () => {
            // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
            delete (window as any).showChargerInResults
        }
    }, [])

    const handleRouteSearch = async (origin: string, destination: string) => {
        setViewMode('results')

        try {
            await searchRoute(origin, destination)
        } catch (error) {
            console.error('Route search error:', error)
            setViewMode('search')
        }
    }

    const handleNewSearch = () => {
        setViewMode('search')
        clearRoute()
    }

    const handleRefresh = () => {
        if (!userLocation || !routeData) return

        // Get the destination from the current URL params or use a default
        const params = new URLSearchParams(window.location.search)
        const destination = params.get('destination')

        if (destination && userLocation) {
            // Use current location as origin and refresh the route
            handleRouteSearch('My Location', decodeURIComponent(destination))
        }
    }

    const toggleView = () => {
        console.log('APP: toggleView called, current viewMode:', viewMode)
        if (viewMode === 'results') {
            console.log('APP: Switching from results to map, restoring saved viewport')
            setViewMode('map')
            // Restore the saved viewport with sync enabled
            setTimeout(() => {
                restoreSavedViewport()
            }, 100)
        } else if (viewMode === 'map') {
            console.log('APP: Switching from map to results, saving current viewport')
            // Save current viewport before going to results
            if (mapRef.current) {
                const map = mapRef.current.getMap()
                if (map) {
                    const center = map.getCenter()
                    const zoom = map.getZoom()
                    updateViewport([center.lat, center.lng], zoom)
                    console.log('APP: Saved viewport:', { center: [center.lat, center.lng], zoom })
                }
            }
            setViewMode('results')
        }
    }

    // Handle geolocation - get GPS location
    useEffect(() => {
        const getUserLocation = () => {
            if (navigator.geolocation) {
                navigator.geolocation.getCurrentPosition(
                    (position) => {
                        setUserLocation([position.coords.latitude, position.coords.longitude])
                    },
                    (error) => {
                        console.log('Geolocation error:', error)
                        // If GPS fails, just leave userLocation as null
                    }
                )
            }
        }

        getUserLocation()
    }, [])

    // Load search from URL on mount
    useEffect(() => {
        const params = new URLSearchParams(window.location.search)
        const origin = params.get('origin')
        const destination = params.get('destination')

        console.log('URL params check - origin:', origin, 'destination:', destination)

        if (origin && destination) {
            console.log('Found URL params, starting route search')
            handleRouteSearch(decodeURIComponent(origin), decodeURIComponent(destination))
        } else {
            console.log('No URL params, staying in search mode')
        }
    }, [])

    // Utility function to decode polyline
    const decodePolyline = (encoded: string): [number, number][] => {
        const coordinates: [number, number][] = []
        let index = 0
        let lat = 0
        let lng = 0

        while (index < encoded.length) {
            // Decode latitude
            let result = 0
            let shift = 0
            let byte: number
            do {
                byte = encoded.charCodeAt(index++) - 63
                result |= (byte & 0x1F) << shift
                shift += 5
            } while (byte >= 0x20)
            lat += (result & 1) ? ~(result >> 1) : (result >> 1)

            // Decode longitude
            result = 0
            shift = 0
            do {
                byte = encoded.charCodeAt(index++) - 63
                result |= (byte & 0x1F) << shift
                shift += 5
            } while (byte >= 0x20)
            lng += (result & 1) ? ~(result >> 1) : (result >> 1)

            coordinates.push([lat / 1e5, lng / 1e5])
        }
        return coordinates
    }

    // Set viewport when route data is received
    useEffect(() => {
        if (routeData?.route?.EncodedPolyline) {
            console.log('APP: Setting viewport to route bounds')
            const coordinates = decodePolyline(routeData.route.EncodedPolyline)
            if (coordinates.length > 0) {
                const polyline = L.polyline(coordinates)
                const bounds = polyline.getBounds().pad(0.1)
                const center = bounds.getCenter()

                // Calculate optimal zoom using actual viewport size
                const EARTH_RADIUS = 6378137 // Earth's radius in meters
                const MAX_ZOOM = 18

                const latDiff = (bounds.getNorth() - bounds.getSouth()) * Math.PI / 180
                const lngDiff = (bounds.getEast() - bounds.getWest()) * Math.PI / 180

                // Convert to meters at the center latitude
                const centerLat = center.lat * Math.PI / 180
                const latDistance = latDiff * EARTH_RADIUS
                const lngDistance = lngDiff * EARTH_RADIUS * Math.cos(centerLat)

                // Use actual viewport size instead of hardcoded values
                const containerWidth = window.innerWidth
                const containerHeight = window.innerHeight

                console.log('APP: Container size:', { width: containerWidth, height: containerHeight })

                // Calculate zoom for both dimensions and take the smaller (more zoomed out)
                // Add some padding by reducing effective container size
                const effectiveWidth = containerWidth * 0.9  // 10% padding
                const effectiveHeight = containerHeight * 0.9 // 10% padding

                const zoomForWidth = Math.log2(effectiveWidth * 156543.03392 * Math.cos(centerLat) / lngDistance)
                const zoomForHeight = Math.log2(effectiveHeight * 156543.03392 / latDistance)

                let optimalZoom = Math.min(zoomForWidth, zoomForHeight)
                optimalZoom = Math.max(1, Math.min(MAX_ZOOM, Math.floor(optimalZoom)))

                setViewportToRoute(bounds, center, optimalZoom)
                console.log('APP: Set viewport to route bounds with calculated zoom:', {
                    center: [center.lat, center.lng],
                    zoom: optimalZoom,
                    latDistance: latDistance / 1000, // km
                    lngDistance: lngDistance / 1000, // km
                    containerSize: { width: containerWidth, height: containerHeight },
                    zoomForWidth,
                    zoomForHeight
                })
            }
        }
    }, [routeData, setViewportToRoute])

    console.log('Rendering - viewMode:', viewMode)

    return (
        <div className="h-screen overflow-hidden bg-princess-surface">
            {viewMode === 'search' && (
                <div className="h-full flex items-center justify-center bg-gradient-to-b from-princess-surface to-princess-lavender">
                    <SearchForm
                        onSearch={handleRouteSearch}
                        isLoading={isLoading}
                        statusMessage={error || ''}
                        isError={!!error}
                        userLocation={userLocation}
                    />
                </div>
            )}

            {(viewMode === 'results' || viewMode === 'map') && (
                <>
                    <TopToolbar
                        onNewSearch={handleNewSearch}
                        onToggleView={toggleView}
                        onOpenFilter={() => setIsFilterModalOpen(true)}
                        onRefresh={handleRefresh}
                        viewMode={viewMode}
                        filterCount={filterCount}
                    />

                    <div className="flex" style={{ height: 'calc(100vh - 50px)', marginTop: '50px' }}>
                        {viewMode === 'results' ? (
                            <ResultsTable
                                stationData={filteredStationData}
                                routeData={routeData}
                                searchTerm={searchTerm}
                                isLoading={isLoading}
                                statusMessage={error || ''}
                                className="w-full"
                                onShowSuperchargerPopup={(placeId) => {
                                    console.log('APP: onShowSuperchargerPopup called with placeId:', placeId)
                                    // Find the supercharger in station data
                                    const station = filteredStationData.find(s => s.chargerInfo.supercharger.place_id === placeId)
                                    if (station) {
                                        const supercharger = station.chargerInfo.supercharger
                                        setViewportToLocation([supercharger.latitude, supercharger.longitude], 18)
                                        setViewMode('map')
                                        setTimeout(() => {
                                            mapRef.current?.showSuperchargerPopup(placeId)
                                        }, 100)
                                    }
                                }}
                                onShowRestaurantPopup={(placeId) => {
                                    console.log('APP: onShowRestaurantPopup called with placeId:', placeId)
                                    // Find the restaurant in station data
                                    for (const station of filteredStationData) {
                                        const restaurant = station.restaurants?.find(r => r.place_id === placeId)
                                        if (restaurant) {
                                            setViewportToLocation([restaurant.latitude, restaurant.longitude], 18)
                                            setViewMode('map')
                                            setTimeout(() => {
                                                mapRef.current?.showRestaurantPopup(placeId)
                                            }, 100)
                                            return
                                        }
                                    }
                                }}
                            />
                        ) : (
                            <Map
                                ref={mapRef}
                                routeData={routeData}
                                stationData={filteredStationData}
                                userLocation={userLocation}
                                searchFilters={searchFilters}
                                className="w-full"
                            />
                        )}
                    </div>
                </>
            )}

            {/* Floating Help Button */}
            <button
                onClick={() => setIsHelpModalOpen(true)}
                className="fixed bottom-6 right-6 w-16 h-16 bg-gradient-to-r from-princess-accent-lavender to-princess-accent-rose 
                         text-princess-text-primary text-3xl rounded-full shadow-lg hover:shadow-xl
                         hover:from-princess-accent-rose hover:to-princess-accent-lavender
                         transition-all duration-300 transform hover:scale-110 z-50
                         border-2 border-princess-border backdrop-blur-sm"
                title="Help"
            >
                ❓
            </button>

            <FilterModal
                isOpen={isFilterModalOpen}
                onClose={() => setIsFilterModalOpen(false)}
                searchTerm={searchTerm}
                cuisineFilters={cuisineFilters}
                onFilter={updateFilters}
                stationData={stationData}
            />

            <HelpModal
                isOpen={isHelpModalOpen}
                onClose={() => setIsHelpModalOpen(false)}
            />
        </div>
    )
}

const App = () => {
    return (
        <ViewportProvider>
            <AppContent />
        </ViewportProvider>
    )
}

export default App
