import { useState, useEffect, useRef } from 'react'
import Map, { MapRef } from './components/Map'
import SearchForm from './components/SearchForm'
import ResultsTable from './components/ResultsTable'
import TopToolbar from './components/TopToolbar'
import FilterModal from './components/FilterModal'
import HelpModal from './components/HelpModal'
import { RouteResponse, StationData, ViewMode, SuperchargerWithETA, ViewportResponse } from './types'

const App = () => {
    const [viewMode, setViewMode] = useState<ViewMode>('search')
    const [routeData, setRouteData] = useState<RouteResponse | null>(null)
    const [stationData, setStationData] = useState<StationData[]>([])
    const [filteredStationData, setFilteredStationData] = useState<StationData[]>([])
    const [searchTerm, setSearchTerm] = useState('')
    const [cuisineFilters, setCuisineFilters] = useState<string[]>([''])
    const [isLoading, setIsLoading] = useState(false)
    const [statusMessage, setStatusMessage] = useState('')
    const [isError, setIsError] = useState(false)
    const [isFilterModalOpen, setIsFilterModalOpen] = useState(false)
    const [isHelpModalOpen, setIsHelpModalOpen] = useState(false)
    const [userLocation, setUserLocation] = useState<[number, number] | null>(null)

    const mapRef = useRef<MapRef>(null)

    console.log('App render - viewMode:', viewMode, 'routeData:', !!routeData)

    const handleRouteSearch = async (origin: string, destination: string) => {
        setIsLoading(true)
        setViewMode('results')
        setStatusMessage('Finding your route...')

        try {
            const response = await fetch(`/route?origin=${encodeURIComponent(origin)}&destination=${encodeURIComponent(destination)}`)
            const data = await response.json()

            if (!response.ok) {
                throw new Error(data.error || 'Failed to find route')
            }

            setRouteData(data)

            // Transform the route data to match StationData structure expected by legacy components
            const transformedStations: StationData[] = (data.superchargers || []).map((item: SuperchargerWithETA, index: number) => ({
                id: item.supercharger?.place_id || `station-${index}`,
                chargerInfo: {
                    supercharger: item.supercharger,
                    restaurants: item.restaurants || [],
                    distance_along_route: item.distance_along_route,
                    distance_from_route: item.distance_from_route
                },
                restaurants: item.restaurants || []
            }))

            setStationData(transformedStations)
            setFilteredStationData(transformedStations)
            setStatusMessage('')
            setIsError(false)

            // Update URL
            const url = new URL(window.location.href)
            url.searchParams.set('origin', origin)
            url.searchParams.set('destination', destination)
            window.history.pushState({}, '', url)

        } catch (error) {
            console.error('Route search error:', error)
            setStatusMessage(error instanceof Error ? error.message : 'Failed to find route')
            setIsError(true)
            setViewMode('search')
        } finally {
            setIsLoading(false)
        }
    }

    const handleFilter = (term: string, cuisines: string[]) => {
        setSearchTerm(term)
        setCuisineFilters(cuisines)

        const filtered = stationData.map(station => {
            const matchingRestaurants = station.restaurants.filter(restaurant => {
                const nameMatch = !term || restaurant.name.toLowerCase().includes(term.toLowerCase())
                const cuisineMatch = cuisines.includes('') || cuisines.length === 0 ||
                    cuisines.some(cuisine => cuisine !== '' && (restaurant.primary_type_display || '').toLowerCase().includes(cuisine.toLowerCase()))
                return nameMatch && cuisineMatch
            })

            if (matchingRestaurants.length > 0) {
                return { ...station, restaurants: matchingRestaurants }
            }
            return null
        }).filter(station => station !== null) as StationData[]

        setFilteredStationData(filtered)
    }

    const handleNewSearch = () => {
        setViewMode('search')
        setRouteData(null)
        setStationData([])
        setFilteredStationData([])
        setSearchTerm('')
        setCuisineFilters([''])
        setStatusMessage('')
        setIsError(false)

        // Clear URL params
        window.history.replaceState(null, '', window.location.pathname)
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
        if (viewMode === 'results') {
            setViewMode('map')
        } else if (viewMode === 'map') {
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
                        statusMessage={statusMessage}
                        isError={isError}
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
                        filterCount={getFilterCount()}
                    />

                    <div className="flex" style={{ height: 'calc(100vh - 50px)', marginTop: '50px' }}>
                        {viewMode === 'results' ? (
                            <ResultsTable
                                stationData={filteredStationData}
                                routeData={routeData}
                                searchTerm={searchTerm}
                                cuisineFilters={cuisineFilters}
                                isLoading={isLoading}
                                statusMessage={statusMessage}
                                className="w-full"
                                onShowSuperchargerPopup={(placeId) => {
                                    setViewMode('map')
                                    setTimeout(() => {
                                        mapRef.current?.showSuperchargerPopup(placeId)
                                    }, 100)
                                }}
                                onShowRestaurantPopup={(placeId) => {
                                    setViewMode('map')
                                    setTimeout(() => {
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
                                searchFilters={{ searchTerm, cuisineFilters }}
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
                onFilter={handleFilter}
                stationData={stationData}
            />

            <HelpModal
                isOpen={isHelpModalOpen}
                onClose={() => setIsHelpModalOpen(false)}
            />
        </div>
    )

    function getFilterCount(): number {
        let count = 0
        if (cuisineFilters.length > 0 && !cuisineFilters.includes('')) {
            count += cuisineFilters.length
        }
        if (searchTerm.trim() !== '') {
            count += 1
        }
        return count
    }
}

export default App
