import { useState, useEffect, useRef } from 'react'
import Map, { MapRef } from './components/Map'
import SearchForm from './components/SearchForm'
import ResultsTable from './components/ResultsTable'
import TopToolbar from './components/TopToolbar'
import FilterModal from './components/FilterModal'
import HelpModal from './components/HelpModal'
import { ViewMode } from './types'
import { useRoute } from './hooks/useRoute'
import { useFilters } from './hooks/useFilters'

const App = () => {
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

    console.log('App render - viewMode:', viewMode, 'routeData:', !!routeData)

    // Add global function for popup buttons
    useEffect(() => {
        window.showChargerInResults = (_chargerId: string) => {
            setViewMode('results')
            // You could add scrolling to the specific charger here
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
        mapRef.current?.resetRouteInitialization()
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
            console.log('APP: Switching from results to map, restoring viewport')
            setViewMode('map')
            // Restore saved viewport when going to map (increased delay)
            setTimeout(() => {
                console.log('APP: Calling restoreViewport')
                mapRef.current?.restoreViewport()
            }, 200)
        } else if (viewMode === 'map') {
            console.log('APP: Switching from map to results, saving viewport')
            // Save current viewport before going to results
            mapRef.current?.saveViewport()
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
                                    console.log('APP: Current viewMode:', viewMode, 'switching to map')
                                    setViewMode('map')
                                    // Add delay to ensure map is fully rendered
                                    setTimeout(() => {
                                        console.log('APP: Calling showSuperchargerPopup')
                                        mapRef.current?.showSuperchargerPopup(placeId)
                                    }, 100)
                                }}
                                onShowRestaurantPopup={(placeId) => {
                                    console.log('APP: onShowRestaurantPopup called with placeId:', placeId)
                                    console.log('APP: Current viewMode:', viewMode, 'switching to map')
                                    setViewMode('map')
                                    // Add delay to ensure map is fully rendered
                                    setTimeout(() => {
                                        console.log('APP: Calling showRestaurantPopup')
                                        mapRef.current?.showRestaurantPopup(placeId)
                                    }, 100)
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

export default App
