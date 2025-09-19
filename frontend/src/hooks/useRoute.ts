import { useState, useCallback } from 'react'
import { RouteResponse, SuperchargerWithETA, StationData } from '../types'

export interface RouteState {
    routeData: RouteResponse | null
    stationData: StationData[]
    isLoading: boolean
    error: string | null
}

export function useRoute() {
    const [state, setState] = useState<RouteState>({
        routeData: null,
        stationData: [],
        isLoading: false,
        error: null
    })

    const searchRoute = useCallback(async (origin: string, destination: string) => {
        setState(prev => ({ ...prev, isLoading: true, error: null }))

        try {
            const response = await fetch(`/route?origin=${encodeURIComponent(origin)}&destination=${encodeURIComponent(destination)}`)
            const data: RouteResponse = await response.json()

            if (!response.ok) {
                throw new Error((data as any).error || 'Failed to find route')
            }

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

            setState({
                routeData: data,
                stationData: transformedStations,
                isLoading: false,
                error: null
            })

            // Update URL
            const url = new URL(window.location.href)
            url.searchParams.set('origin', origin)
            url.searchParams.set('destination', destination)
            window.history.pushState({}, '', url)

            return data
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : 'Failed to find route'
            setState(prev => ({
                ...prev,
                isLoading: false,
                error: errorMessage
            }))
            throw error
        }
    }, [])

    const clearRoute = useCallback(() => {
        setState({
            routeData: null,
            stationData: [],
            isLoading: false,
            error: null
        })
        window.history.replaceState(null, '', window.location.pathname)
    }, [])

    return {
        ...state,
        searchRoute,
        clearRoute
    }
}
