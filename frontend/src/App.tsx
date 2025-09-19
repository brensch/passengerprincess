import { useState, useEffect, useRef } from 'react'
import Map from './components/Map'
import SearchForm from './components/SearchForm'
import ResultsTable from './components/ResultsTable'
import TopToolbar from './components/TopToolbar'
import FilterModal from './components/FilterModal'
import HelpModal from './components/HelpModal'
import { RouteData, StationData, ViewMode } from './types'

const App = () => {
    const [viewMode, setViewMode] = useState<ViewMode>('search')
    const [routeData, setRouteData] = useState<RouteData | null>(null)
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

    const mapRef = useRef<any>(null)

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

            // Transform the route data to match StationData structure expected by components
            const transformedStations: StationData[] = (data.superchargers || []).map((item: any, index: number) => ({
                id: item.supercharger?.place_id || `station-${index}`,
                chargerInfo: {
                    supercharger: item.supercharger,
                    restaurants: item.restaurants || [],
                    eta_ms: item.eta_ms,
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

    const toggleView = () => {
        if (viewMode === 'results') {
            setViewMode('map')
        } else if (viewMode === 'map') {
            setViewMode('results')
        }
    }

    // Handle geolocation
    useEffect(() => {
        if (navigator.geolocation) {
            navigator.geolocation.getCurrentPosition(
                (position) => {
                    setUserLocation([position.coords.latitude, position.coords.longitude])
                },
                (error) => {
                    console.log('Geolocation error:', error)
                }
            )
        }
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
                <div className="h-full flex flex-col">
                    <Map
                        ref={mapRef}
                        routeData={routeData}
                        stationData={filteredStationData}
                        userLocation={userLocation}
                        searchFilters={{ searchTerm, cuisineFilters }}
                        className="flex-1"
                    />
                    <div className="absolute inset-0 z-10 bg-gradient-to-b from-princess-surface to-princess-lavender backdrop-blur-sm" style={{ backgroundColor: 'rgba(253, 251, 255, 0.95)' }}>
                        <div className="h-full flex items-center justify-center">
                            <SearchForm
                                onSearch={handleRouteSearch}
                                isLoading={isLoading}
                                statusMessage={statusMessage}
                                isError={isError}
                                userLocation={userLocation}
                            />
                        </div>
                    </div>
                </div>
            )}

            {(viewMode === 'results' || viewMode === 'map') && (
                <>
                    <TopToolbar
                        onNewSearch={handleNewSearch}
                        onToggleView={toggleView}
                        onOpenFilter={() => setIsFilterModalOpen(true)}
                        onOpenHelp={() => setIsHelpModalOpen(true)}
                        viewMode={viewMode}
                        filterCount={getFilterCount()}
                    />

                    <div className="flex" style={{ height: 'calc(100vh - 66px)', marginTop: '66px' }}>
                        {viewMode === 'results' ? (
                            <ResultsTable
                                stationData={filteredStationData}
                                searchTerm={searchTerm}
                                cuisineFilters={cuisineFilters}
                                isLoading={isLoading}
                                statusMessage={statusMessage}
                                className="w-full"
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
