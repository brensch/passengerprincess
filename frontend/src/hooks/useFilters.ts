import { useState, useCallback, useMemo } from 'react'
import { StationData, SearchFilters } from '../types'

export function useFilters(stationData: StationData[]) {
    const [searchTerm, setSearchTerm] = useState('')
    const [cuisineFilters, setCuisineFilters] = useState<string[]>([''])

    const searchFilters: SearchFilters = useMemo(() => ({
        searchTerm,
        cuisineFilters
    }), [searchTerm, cuisineFilters])

    const filteredStationData = useMemo(() => {
        if (!stationData.length) return []

        return stationData.map(station => {
            const matchingRestaurants = station.restaurants.filter(restaurant => {
                const nameMatch = !searchTerm ||
                    restaurant.name.toLowerCase().includes(searchTerm.toLowerCase())
                const cuisineMatch = cuisineFilters.includes('') ||
                    cuisineFilters.length === 0 ||
                    cuisineFilters.some(cuisine =>
                        cuisine !== '' &&
                        (restaurant.primary_type_display || '').toLowerCase().includes(cuisine.toLowerCase())
                    )
                return nameMatch && cuisineMatch
            })

            if (matchingRestaurants.length > 0) {
                return { ...station, restaurants: matchingRestaurants }
            }
            return null
        }).filter((station): station is StationData => station !== null)
    }, [stationData, searchTerm, cuisineFilters])

    const updateFilters = useCallback((term: string, cuisines: string[]) => {
        setSearchTerm(term)
        setCuisineFilters(cuisines)
    }, [])

    const clearFilters = useCallback(() => {
        setSearchTerm('')
        setCuisineFilters([''])
    }, [])

    const filterCount = useMemo(() => {
        let count = 0
        if (cuisineFilters.length > 0 && !cuisineFilters.includes('')) {
            count += cuisineFilters.length
        }
        if (searchTerm.trim() !== '') {
            count += 1
        }
        return count
    }, [searchTerm, cuisineFilters])

    return {
        searchFilters,
        filteredStationData,
        searchTerm,
        cuisineFilters,
        updateFilters,
        clearFilters,
        filterCount
    }
}
