import { useState, useEffect } from 'react'
import { StationData } from '../types'

interface FilterModalProps {
    isOpen: boolean
    onClose: () => void
    searchTerm: string
    cuisineFilters: string[]
    onFilter: (searchTerm: string, cuisineFilters: string[]) => void
    stationData: StationData[]
}

const FilterModal = ({
    isOpen,
    onClose,
    searchTerm,
    cuisineFilters,
    onFilter,
    stationData
}: FilterModalProps) => {
    const [localSearchTerm, setLocalSearchTerm] = useState(searchTerm)
    const [localCuisineFilters, setLocalCuisineFilters] = useState<string[]>(cuisineFilters)
    const [availableCuisines, setAvailableCuisines] = useState<string[]>([])
    const [cuisineCounts, setCuisineCounts] = useState<Record<string, number>>({})
    const [restaurantCounts, setRestaurantCounts] = useState<Record<string, number>>({})
    const [restaurantNames, setRestaurantNames] = useState<string[]>([])
    const [suggestions, setSuggestions] = useState<string[]>([])
    const [showSuggestions, setShowSuggestions] = useState(false)

    useEffect(() => {
        setLocalSearchTerm(searchTerm)
        setLocalCuisineFilters(cuisineFilters)
    }, [searchTerm, cuisineFilters])

    useEffect(() => {
        // Extract cuisine and restaurant counts from station data
        const cuisineCountsMap: Record<string, number> = {}
        const restaurantCountsMap: Record<string, number> = {}
        stationData.forEach(station => {
            station.restaurants?.forEach(restaurant => {
                if (restaurant.primary_type_display) {
                    const type = restaurant.primary_type_display.trim()
                    cuisineCountsMap[type] = (cuisineCountsMap[type] || 0) + 1
                }
                if (restaurant.name) {
                    const name = restaurant.name.trim()
                    restaurantCountsMap[name] = (restaurantCountsMap[name] || 0) + 1
                }
            })
        })
        setCuisineCounts(cuisineCountsMap)
        setAvailableCuisines(Object.keys(cuisineCountsMap).sort())
        setRestaurantCounts(restaurantCountsMap)
        setRestaurantNames(Object.keys(restaurantCountsMap).sort((a, b) => restaurantCountsMap[b] - restaurantCountsMap[a]))
    }, [stationData])

    const handleCuisineToggle = (cuisine: string) => {
        let newFilters: string[]
        if (cuisine === '') {
            // "All Cuisines" selected
            newFilters = ['']
        } else {
            // Individual cuisine toggle
            if (localCuisineFilters.includes('') || localCuisineFilters.length === 0) {
                // If "All" is selected or nothing is selected, replace with this cuisine
                newFilters = [cuisine]
            } else if (localCuisineFilters.includes(cuisine)) {
                // Remove cuisine
                newFilters = localCuisineFilters.filter(c => c !== cuisine)
                newFilters = newFilters.length === 0 ? [''] : newFilters
            } else {
                // Add cuisine
                newFilters = [...localCuisineFilters.filter(c => c !== ''), cuisine]
            }
        }
        setLocalCuisineFilters(newFilters)
        onFilter(localSearchTerm, newFilters)
    }

    const handleApply = () => {
        onClose()
    }

    const handleClear = () => {
        setLocalSearchTerm('')
        setLocalCuisineFilters([''])
        onFilter('', [''])
        onClose()
    }

    if (!isOpen) return null

    return (
        <div
            className="fixed inset-0 flex items-center justify-center z-[1200] bg-black bg-opacity-50 backdrop-blur-sm"
            onClick={(e) => e.target === e.currentTarget && onClose()}
        >
            <div className="rounded-xl max-w-lg w-full mx-4 shadow-2xl bg-princess-surface border-2 border-princess-border flex flex-col h-[80vh]">
                <div className="flex flex-col h-full">
                    <div className="p-6 pb-4">
                        <div className="relative">
                            <input
                                type="text"
                                value={localSearchTerm}
                                onChange={(e) => {
                                    const value = e.target.value
                                    setLocalSearchTerm(value)
                                    onFilter(value, localCuisineFilters)
                                    if (value.trim() === '') {
                                        setSuggestions(restaurantNames.slice(0, 10))
                                        setShowSuggestions(true)
                                    } else {
                                        setSuggestions(restaurantNames.filter(name =>
                                            name.toLowerCase().includes(value.toLowerCase())
                                        ).slice(0, 10))
                                        setShowSuggestions(true)
                                    }
                                }}
                                onFocus={() => {
                                    setShowSuggestions(true)
                                    if (localSearchTerm.trim() === '') {
                                        setSuggestions(restaurantNames.slice(0, 10))
                                    }
                                }}
                                onBlur={() => setTimeout(() => setShowSuggestions(false), 200)}
                                className="w-full px-4 py-3 rounded-md shadow-sm focus:outline-none focus:ring-2 
                         focus:ring-princess-accent-lavender text-base bg-princess-surface 
                         border border-princess-border"
                                placeholder="Pick places please"
                            />

                            {showSuggestions && (
                                <div className="absolute top-full left-0 right-0 bg-princess-surface border border-princess-border rounded-xl mt-1 max-h-60 overflow-y-auto z-10 shadow-lg">
                                    <div className="p-2">
                                        {suggestions.map(name => (
                                            <div
                                                key={name}
                                                onClick={() => {
                                                    setLocalSearchTerm(name)
                                                    onFilter(name, localCuisineFilters)
                                                    setShowSuggestions(false)
                                                }}
                                                className="p-2 hover:bg-princess-surface-soft rounded cursor-pointer"
                                            >
                                                {name} ({restaurantCounts[name]})
                                            </div>
                                        ))}
                                        <div className="text-xs text-princess-text-secondary p-2 border-t border-princess-border mt-2">
                                            Only showing suggestions from along route. You can type a value not here and it will appear on the map in places not along your current route if you're really keen for that restaurant.
                                        </div>
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>

                    <div className="border-t border-princess-border"></div>

                    <div className="flex-1 overflow-y-auto px-6 pt-4">
                        <div className="flex flex-wrap gap-2 p-1">
                            <button
                                onClick={() => handleCuisineToggle('')}
                                className={`px-2 py-1 rounded-xl text-xs font-medium transition-all duration-200 
                        border touch-manipulation select-none ${localCuisineFilters.includes('') || localCuisineFilters.length === 0
                                        ? 'bg-princess-accent-lavender text-princess-text-primary border-princess-accent-lavender'
                                        : 'bg-princess-surface text-princess-text-primary border-princess-border md:hover:bg-princess-surface-soft'
                                    }`}
                            >
                                All Cuisines
                            </button>

                            {availableCuisines.map(cuisine => (
                                <button
                                    key={cuisine}
                                    onClick={() => handleCuisineToggle(cuisine)}
                                    className={`px-2 py-1 rounded-xl text-xs font-medium transition-all duration-200 
                          border touch-manipulation select-none ${localCuisineFilters.includes(cuisine)
                                            ? 'bg-princess-accent-lavender text-princess-text-primary border-princess-accent-lavender'
                                            : 'bg-princess-surface text-princess-text-primary border-princess-border md:hover:bg-princess-surface-soft'
                                        }`}
                                >
                                    {cuisine} ({cuisineCounts[cuisine]})
                                </button>
                            ))}
                        </div>
                    </div>

                    <div className="p-4 border-t border-princess-border flex gap-2 justify-end">
                        <button
                            onClick={handleClear}
                            className="px-3 py-2 text-xs font-medium rounded-xl transition-all duration-300
                     bg-transparent text-princess-text-primary border-2 border-princess-border
                     hover:bg-princess-surface-soft"
                        >
                            Clear All
                        </button>
                        <button
                            onClick={handleApply}
                            className="px-3 py-2 text-xs font-medium rounded-xl transition-all duration-300
                     bg-princess-accent-lavender text-princess-text-primary border border-princess-accent-lavender
                     hover:bg-princess-accent-rose"
                        >
                            Perfect
                        </button>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default FilterModal
