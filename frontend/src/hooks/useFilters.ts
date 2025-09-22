import { useState, useCallback, useMemo } from 'react'
import { StationData, SearchFilters } from '../types'

export function useFilters(stationData: StationData[]) {
    const [selectedPlaces, setSelectedPlaces] = useState<string[]>([])
    const [typedPlace, setTypedPlace] = useState('')
    const [selectedCuisines, setSelectedCuisines] = useState<string[]>([])

    const searchFilters: SearchFilters = useMemo(() => ({
        selectedPlaces,
        typedPlace,
        selectedCuisines
    }), [selectedPlaces, typedPlace, selectedCuisines])

    const filteredStationData = useMemo((): StationData[] => {
        if (!stationData.length) return []

        // Compute all restaurant names and cuisines for checking if typed is on route
        const allRestaurantNames = new Set<string>()
        stationData.forEach(station => {
            station.restaurants?.forEach(r => {
                if (r.name) allRestaurantNames.add(r.name)
            })
        })

        const filteredStations: StationData[] = []

        for (const station of stationData) {
            const matchingRestaurants = station.restaurants?.filter(restaurant => {
                const nameMatch = (typedPlace && !allRestaurantNames.has(typedPlace) && restaurant.name.toLowerCase().includes(typedPlace.toLowerCase())) ||
                    selectedPlaces.includes(restaurant.name)
                const cuisineMatch = selectedCuisines.length === 0 ||
                    selectedCuisines.some(cuisine =>
                        (restaurant.primary_type_display || '').toLowerCase().includes(cuisine.toLowerCase())
                    )
                return nameMatch && cuisineMatch
            }) || []

            if (matchingRestaurants.length > 0) {
                filteredStations.push({ ...station, restaurants: matchingRestaurants })
            }
        }

        return filteredStations
    }, [stationData, selectedPlaces, typedPlace, selectedCuisines])

    const updateFilters = useCallback((selPlaces: string[], typPlace: string, selCuisines: string[]) => {
        setSelectedPlaces(selPlaces)
        setTypedPlace(typPlace)
        setSelectedCuisines(selCuisines)
    }, [])

    const clearFilters = useCallback(() => {
        setSelectedPlaces([])
        setTypedPlace('')
        setSelectedCuisines([])
    }, [])

    const filterCount = useMemo(() => {
        let count = 0
        count += selectedPlaces.length
        if (typedPlace.trim() !== '') {
            count += 1
        }
        count += selectedCuisines.length
        return count
    }, [selectedPlaces, typedPlace, selectedCuisines])

    return {
        searchFilters,
        filteredStationData,
        selectedPlaces,
        typedPlace,
        selectedCuisines,
        updateFilters,
        clearFilters,
        filterCount
    }
}
