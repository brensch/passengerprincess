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

    // Add global function for popup buttons
    useEffect(() => {
        window.showChargerInResults = (chargerId: string) => {
            setViewMode('results')

            // Wait for the view to update, then scroll to the charger
            setTimeout(() => {
                const targetRows = document.querySelectorAll(`tr[data-charger-id="${chargerId}"]`) as NodeListOf<HTMLElement>
                if (targetRows.length > 0) {
                    targetRows.forEach(row => {
                        row.style.backgroundColor = '#fbbf24' // amber-400
                        row.style.transition = 'background-color 0.3s ease'
                    })

                    targetRows[0].scrollIntoView({
                        behavior: 'auto',
                        block: 'center'
                    })

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
        try {
            const params = new URLSearchParams()
            params.set('origin', encodeURIComponent(origin))
            params.set('destination', encodeURIComponent(destination))
            window.history.replaceState({}, '', `${window.location.pathname}?${params.toString()}`)
            await searchRoute(origin, destination)
        } catch (error) {
            console.error('Route search error:', error)
        }
    }

    const handleNewSearch = () => {
        setViewMode('search')
        clearRoute()
        window.history.replaceState({}, '', window.location.pathname)
    }

    const handleRefresh = () => {
        if (!userLocation || !routeData) return
        const params = new URLSearchParams(window.location.search)
        const destination = params.get('destination')
        if (destination && userLocation) {
            const originCoords = `${userLocation[0]},${userLocation[1]}`
            handleRouteSearch(originCoords, decodeURIComponent(destination))
        }
    }

    const toggleView = () => {
        if (viewMode === 'results') {
            setViewMode('map')
            setTimeout(() => {
                restoreSavedViewport()
            }, 100)
        } else if (viewMode === 'map') {
            if (mapRef.current) {
                const map = mapRef.current.getMap()
                if (map) {
                    const center = map.getCenter()
                    const zoom = map.getZoom()
                    updateViewport([center.lat, center.lng], zoom)
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
                    (_error) => {
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
        if (origin && destination) {
            const decodedOrigin = decodeURIComponent(origin)
            const decodedDestination = decodeURIComponent(destination)
            handleRouteSearch(decodedOrigin, decodedDestination)
        }
    }, [])

    // Utility function to decode polyline
    const decodePolyline = (encoded: string): [number, number][] => {
        const coordinates: [number, number][] = []
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
            coordinates.push([lat / 1e5, lng / 1e5])
        }
        return coordinates
    }

    // Set viewport when route data is received
    useEffect(() => {
        if (routeData?.route?.EncodedPolyline) {
            const coordinates = decodePolyline(routeData.route.EncodedPolyline)
            if (coordinates.length > 0) {
                const polyline = L.polyline(coordinates)
                const bounds = polyline.getBounds().pad(0.1)
                setViewportToRoute(bounds)
            }
        }
    }, [routeData, setViewportToRoute])

    // Switch to map view when route data becomes available
    useEffect(() => {
        if (routeData && viewMode === 'search') {
            setViewMode('map')
        }
    }, [routeData, viewMode])

    return (
        <div className="h-screen overflow-hidden bg-princess-surface relative">
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
                    <div className="absolute top-0 left-0 right-0 z-10 h-16 bg-princess-surface">
                        <TopToolbar
                            onNewSearch={handleNewSearch}
                            onToggleView={toggleView}
                            onOpenFilter={() => setIsFilterModalOpen(true)}
                            viewMode={viewMode}
                            filterCount={filterCount}
                        />
                    </div>

                    <div className="h-full w-full pt-16">
                        <div className="h-full w-full overflow-hidden">
                            {viewMode === 'results' ? (
                                <ResultsTable
                                    stationData={filteredStationData}
                                    routeData={routeData}
                                    searchTerm={searchTerm}
                                    className="w-full"
                                    onShowSuperchargerPopup={(placeId) => {
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
                                    onRefresh={handleRefresh}
                                    isLoading={isLoading}
                                />
                            )}
                        </div>
                    </div>
                </>
            )}

            {viewMode === 'search' && (
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
            )}

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

