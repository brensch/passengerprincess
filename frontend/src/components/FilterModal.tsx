import { useState, useEffect, useMemo } from 'react'
import { StationData } from '../types'

interface FilterModalProps {
    isOpen: boolean
    onClose: () => void
    selectedPlaces: string[]
    selectedContainingPlaces: string[]
    typedPlace: string
    selectedCuisines: string[]
    onFilter: (selectedPlaces: string[], selectedContainingPlaces: string[], typedPlace: string, selectedCuisines: string[]) => void
    stationData: StationData[]
}

const FilterModal = ({
    isOpen,
    onClose,
    selectedPlaces,
    selectedContainingPlaces,
    typedPlace,
    selectedCuisines,
    onFilter,
    stationData
}: FilterModalProps) => {
    const [localSelectedPlaces, setLocalSelectedPlaces] = useState<string[]>(selectedPlaces)
    const [localSelectedContainingPlaces, setLocalSelectedContainingPlaces] = useState<string[]>(selectedContainingPlaces)
    const [localTypedPlace, setLocalTypedPlace] = useState(typedPlace)
    const [localSelectedCuisines, setLocalSelectedCuisines] = useState<string[]>(selectedCuisines)
    const [availableCuisines, setAvailableCuisines] = useState<string[]>([])
    const [cuisineCounts, setCuisineCounts] = useState<Record<string, number>>({})
    const [restaurantCounts, setRestaurantCounts] = useState<Record<string, number>>({})
    const [restaurantNames, setRestaurantNames] = useState<string[]>([])
    const [placeSuggestions, setPlaceSuggestions] = useState<string[]>([])
    const [showPlaceSuggestions, setShowPlaceSuggestions] = useState(false)
    const [cuisineSuggestions, setCuisineSuggestions] = useState<string[]>([])
    const [showCuisineSuggestions, setShowCuisineSuggestions] = useState(false)
    const [restaurantToCuisines, setRestaurantToCuisines] = useState<Record<string, string[]>>({})
    const [cuisineToRestaurants, setCuisineToRestaurants] = useState<Record<string, string[]>>({})

    useEffect(() => {
        setLocalSelectedPlaces(selectedPlaces)
        setLocalSelectedContainingPlaces(selectedContainingPlaces)
        setLocalTypedPlace(typedPlace)
        setLocalSelectedCuisines(selectedCuisines)
    }, [selectedPlaces, selectedContainingPlaces, typedPlace, selectedCuisines])

    useEffect(() => {
        // Extract cuisine and restaurant counts from station data
        const cuisineCountsMap: Record<string, number> = {}
        const restaurantCountsMap: Record<string, number> = {}
        const restToCuis: Record<string, string[]> = {}
        const cuisineToRests: Record<string, string[]> = {}
        const seenPlaceIds = new Set<string>()
        stationData.forEach(station => {
            station.restaurants?.forEach(restaurant => {
                if (restaurant.primary_type_display) {
                    const type = restaurant.primary_type_display.trim()
                    cuisineCountsMap[type] = (cuisineCountsMap[type] || 0) + 1
                }
                if (restaurant.name && restaurant.place_id) {
                    const name = restaurant.name.trim()
                    const placeId = restaurant.place_id
                    if (!seenPlaceIds.has(placeId)) {
                        seenPlaceIds.add(placeId)
                        restaurantCountsMap[name] = (restaurantCountsMap[name] || 0) + 1
                    }
                    // Add to maps
                    if (restaurant.primary_type_display) {
                        const type = restaurant.primary_type_display.trim()
                        if (!cuisineToRests[type]) cuisineToRests[type] = []
                        if (!cuisineToRests[type].includes(name)) {
                            cuisineToRests[type].push(name)
                        }
                        if (!restToCuis[name]) restToCuis[name] = []
                        if (!restToCuis[name].includes(type)) {
                            restToCuis[name].push(type)
                        }
                    }
                }
            })
        })
        setCuisineCounts(cuisineCountsMap)
        setAvailableCuisines(Object.keys(cuisineCountsMap).sort())
        setRestaurantCounts(restaurantCountsMap)
        setRestaurantNames(Object.keys(restaurantCountsMap).sort((a, b) => restaurantCountsMap[b] - restaurantCountsMap[a]))
        setRestaurantToCuisines(restToCuis)
        setCuisineToRestaurants(cuisineToRests)
    }, [stationData])

    const filteredRestaurantNames = useMemo(() => {
        if (localSelectedCuisines.length === 0) return restaurantNames
        return restaurantNames.filter(name => {
            const cuisines = restaurantToCuisines[name] || []
            return localSelectedCuisines.some(c => cuisines.includes(c))
        })
    }, [restaurantNames, restaurantToCuisines, localSelectedCuisines])

    const filteredCuisines = useMemo(() => {
        if (localSelectedPlaces.length === 0) return availableCuisines
        return availableCuisines.filter(cuisine => {
            const rests = cuisineToRestaurants[cuisine] || []
            return localSelectedPlaces.some(p => rests.includes(p))
        })
    }, [availableCuisines, cuisineToRestaurants, localSelectedPlaces])

    const handlePlaceSelect = (place: string) => {
        const newPlaces = localSelectedPlaces.includes(place) ? localSelectedPlaces : [...localSelectedPlaces, place]
        setLocalSelectedPlaces(newPlaces)
        setLocalTypedPlace('')
        setShowPlaceSuggestions(false)
        onFilter(newPlaces, localSelectedContainingPlaces, '', localSelectedCuisines)
    }

    const handlePlaceRemove = (place: string) => {
        const newPlaces = localSelectedPlaces.filter(p => p !== place)
        setLocalSelectedPlaces(newPlaces)
        onFilter(newPlaces, localSelectedContainingPlaces, localTypedPlace, localSelectedCuisines)
    }

    const handleContainingPlaceSelect = (place: string) => {
        if (!localSelectedContainingPlaces.includes(place)) {
            const newContainingPlaces = [...localSelectedContainingPlaces, place]
            setLocalSelectedContainingPlaces(newContainingPlaces)
            setLocalTypedPlace('')
            setShowPlaceSuggestions(false)
            onFilter(localSelectedPlaces, newContainingPlaces, '', localSelectedCuisines)
        }
    }

    const handleContainingPlaceRemove = (place: string) => {
        const newContainingPlaces = localSelectedContainingPlaces.filter(p => p !== place)
        setLocalSelectedContainingPlaces(newContainingPlaces)
        onFilter(localSelectedPlaces, newContainingPlaces, localTypedPlace, localSelectedCuisines)
    }

    const handleCuisineSelect = (cuisine: string) => {
        const newCuisines = localSelectedCuisines.includes(cuisine) ? localSelectedCuisines : [...localSelectedCuisines, cuisine]
        setLocalSelectedCuisines(newCuisines)
        setShowCuisineSuggestions(false)
        onFilter(localSelectedPlaces, localSelectedContainingPlaces, localTypedPlace, newCuisines)
    }

    const handleCuisineRemove = (cuisine: string) => {
        const newCuisines = localSelectedCuisines.filter(c => c !== cuisine)
        setLocalSelectedCuisines(newCuisines)
        onFilter(localSelectedPlaces, localSelectedContainingPlaces, localTypedPlace, newCuisines)
    }

    const handleApply = () => {
        onClose()
    }

    const handleClear = () => {
        setLocalSelectedPlaces([])
        setLocalSelectedContainingPlaces([])
        setLocalTypedPlace('')
        setLocalSelectedCuisines([])
        onFilter([], [], '', [])
        onClose()
    }

    if (!isOpen) return null

    return (
        <div
            className="fixed inset-0 flex items-center justify-center z-[1200] bg-black bg-opacity-50 backdrop-blur-sm"
            onClick={(e) => e.target === e.currentTarget && onClose()}
        >
            <div className="rounded-xl max-w-lg w-full mx-4 shadow-2xl bg-princess-surface border-2 border-princess-border flex flex-col max-h-[80vh]">
                <div className="flex flex-col h-full">
                    <div className="p-6 pb-4">
                        <h2 className="text-3xl font-semibold my-0 font-dancing">Pick Places Please</h2>
                    </div>
                    <div className="border-t border-princess-border"></div>
                    {/* Places Section */}
                    <div className="p-6 pb-4">
                        <div className="relative">
                            <input
                                type="text"
                                value={localTypedPlace}
                                onChange={(e) => {
                                    const value = e.target.value
                                    setLocalTypedPlace(value)
                                    onFilter(localSelectedPlaces, localSelectedContainingPlaces, value, localSelectedCuisines)
                                    if (value.trim() === '') {
                                        setPlaceSuggestions(filteredRestaurantNames.slice(0, 40))
                                        setShowPlaceSuggestions(true)
                                    } else {
                                        setPlaceSuggestions(filteredRestaurantNames.filter(name =>
                                            name.toLowerCase().includes(value.toLowerCase())
                                        ).slice(0, 40))
                                        setShowPlaceSuggestions(true)
                                    }
                                }}
                                onFocus={() => {
                                    setShowPlaceSuggestions(true)
                                    if (localTypedPlace.trim() === '') {
                                        setPlaceSuggestions(filteredRestaurantNames.slice(0, 40))
                                    }
                                }}
                                onBlur={() => setTimeout(() => setShowPlaceSuggestions(false), 200)}
                                className="w-full px-4 py-3 rounded-md shadow-sm focus:outline-none focus:ring-2 
                         focus:ring-princess-accent-lavender text-base bg-princess-surface 
                         border border-princess-border"
                                placeholder="Names"
                            />

                            {showPlaceSuggestions && (
                                <div className="absolute top-full left-0 right-0 bg-princess-surface border border-princess-border rounded-xl mt-1 max-h-60 overflow-y-auto z-10 shadow-lg">
                                    <div className="p-2">
                                        {placeSuggestions.map(name => (
                                            <div
                                                key={name}
                                                onClick={() => handlePlaceSelect(name)}
                                                className="p-2 hover:bg-princess-surface-soft rounded cursor-pointer"
                                            >
                                                {name} ({restaurantCounts[name]})
                                            </div>
                                        ))}
                                        {localTypedPlace.trim() !== '' && (
                                            <div
                                                onClick={() => handleContainingPlaceSelect(localTypedPlace)}
                                                className="p-2 hover:bg-princess-surface-soft rounded cursor-pointer text-princess-text-secondary border-t border-princess-border mt-2"
                                            >
                                                Add "{localTypedPlace}" (matches containing)
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )}
                        </div>
                        {/* Chips for selected places */}
                        <div className="flex flex-wrap gap-2 mt-4">
                            {localSelectedPlaces.map(place => (
                                <div key={place} className="flex items-center bg-princess-accent-lavender text-princess-text-primary px-2 py-1 rounded-xl text-xs">
                                    {place}
                                    <button onClick={() => handlePlaceRemove(place)} className="ml-2 text-princess-text-primary">×</button>
                                </div>
                            ))}
                            {localSelectedContainingPlaces.map(place => (
                                <div key={place} className="flex items-center bg-princess-peach text-princess-text-primary px-2 py-1 rounded-xl text-xs">
                                    {place} (contains)
                                    <button onClick={() => handleContainingPlaceRemove(place)} className="ml-2 text-princess-text-primary">×</button>
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="border-t border-princess-border"></div>

                    {/* Cuisines Section */}
                    <div className="p-6 pb-4">
                        <div className="relative">
                            <input
                                type="text"
                                onChange={(e) => {
                                    const value = e.target.value
                                    if (value.trim() === '') {
                                        setCuisineSuggestions(filteredCuisines)
                                        setShowCuisineSuggestions(true)
                                    } else {
                                        setCuisineSuggestions(filteredCuisines.filter(cuisine =>
                                            cuisine.toLowerCase().includes(value.toLowerCase())
                                        ))
                                        setShowCuisineSuggestions(true)
                                    }
                                }}
                                onFocus={() => {
                                    setShowCuisineSuggestions(true)
                                    setCuisineSuggestions(filteredCuisines)
                                }}
                                onBlur={() => setTimeout(() => setShowCuisineSuggestions(false), 200)}
                                className="w-full px-4 py-3 rounded-md shadow-sm focus:outline-none focus:ring-2 
                         focus:ring-princess-accent-lavender text-base bg-princess-surface 
                         border border-princess-border"
                                placeholder="Cuisines"
                            />

                            {showCuisineSuggestions && (
                                <div className="absolute top-full left-0 right-0 bg-princess-surface border border-princess-border rounded-xl mt-1 max-h-60 overflow-y-auto z-10 shadow-lg">
                                    <div className="p-2">
                                        {cuisineSuggestions.map(cuisine => (
                                            <div
                                                key={cuisine}
                                                onClick={() => handleCuisineSelect(cuisine)}
                                                className="p-2 hover:bg-princess-surface-soft rounded cursor-pointer"
                                            >
                                                {cuisine} ({cuisineCounts[cuisine]})
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>
                        {/* Chips for selected cuisines */}
                        <div className="flex flex-wrap gap-2 mt-4">
                            {localSelectedCuisines.map(cuisine => (
                                <div key={cuisine} className="flex items-center bg-princess-accent-lavender text-princess-text-primary px-2 py-1 rounded-xl text-xs">
                                    {cuisine}
                                    <button onClick={() => handleCuisineRemove(cuisine)} className="ml-2 text-princess-text-primary">×</button>
                                </div>
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
