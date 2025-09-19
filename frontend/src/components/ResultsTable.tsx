import { StationData } from '../types'

interface ResultsTableProps {
    stationData: StationData[]
    searchTerm: string
    cuisineFilters: string[]
    isLoading: boolean
    statusMessage: string
    className?: string
    onShowSuperchargerPopup?: (placeId: string) => void
    onShowRestaurantPopup?: (placeId: string) => void
}

const ResultsTable = ({
    stationData,
    searchTerm,
    cuisineFilters,
    isLoading,
    statusMessage,
    className = '',
    onShowSuperchargerPopup,
    onShowRestaurantPopup
}: ResultsTableProps) => {

    const formatEpochMsToLocalTime = (epochMs?: number): string => {
        if (!epochMs || epochMs === 0) {
            return 'N/A'
        }

        try {
            const date = new Date(epochMs)
            return date.toLocaleTimeString('en-US', {
                hour: 'numeric',
                minute: '2-digit',
                hour12: true
            })
        } catch (error) {
            console.error('Error formatting epoch time:', error)
            return 'N/A'
        }
    }

    const highlightText = (text: string, searchTerm: string): string => {
        if (!searchTerm) return text
        const regex = new RegExp(`(${searchTerm})`, 'gi')
        return text.replace(regex, '<mark>$1</mark>')
    }

    const openInGoogleMaps = (uri?: string) => {
        if (uri) {
            window.open(uri, '_blank')
        }
    }

    const shareToTesla = (placeId: string, name: string, googleMapsUri?: string) => {
        // Tesla sharing functionality - this would need to be implemented
        console.log('Share to Tesla:', { placeId, name, googleMapsUri })
    }

    if (isLoading) {
        return (
            <div className={`${className} flex items-center justify-center bg-princess-surface`}>
                <div className="text-center">
                    <div className="text-6xl mb-4">✨</div>
                    <div className="text-xl font-dancing text-princess-text-primary">{statusMessage}</div>
                </div>
            </div>
        )
    }

    if (stationData.length === 0) {
        return (
            <div className={`${className} flex items-center justify-center bg-princess-surface`}>
                <div className="text-center text-princess-text-secondary">
                    <div className="text-6xl mb-4">🔍</div>
                    <div className="text-xl">No restaurants found matching your criteria</div>
                </div>
            </div>
        )
    }

    return (
        <div className={`${className} bg-princess-surface overflow-hidden`}>
            <div className="h-full overflow-y-auto custom-scrollbar">
                <table id="chargers-table" className="w-full text-sm border-collapse">
                    <thead className="bg-princess-surface-soft sticky top-0 z-10">
                        <tr className="border-b-2 border-princess-border">
                            <th className="text-left p-3 font-semibold text-princess-text-primary princess-table-header border-r border-princess-border">
                                Charger
                            </th>
                            <th className="text-left p-3 font-semibold text-princess-text-primary princess-table-header">
                                Restaurant
                            </th>
                            <th className="text-left p-3 font-semibold text-princess-text-primary princess-table-header">
                                Walk
                            </th>
                            <th className="text-left p-3 font-semibold text-princess-text-primary princess-table-header">
                                Cuisine
                            </th>
                        </tr>
                    </thead>
                    <tbody>
                        {stationData.map((station, stationIndex) => {
                            const { chargerInfo } = station
                            const restaurants = station.restaurants || []
                            const maxRows = Math.max(1, restaurants.length)

                            // Calculate distances
                            const distanceFromOrigin = chargerInfo.distance_from_route ?
                                (chargerInfo.distance_from_route / 1609.34).toFixed(1) + ' mi' : 'N/A'
                            const distanceAlongRoute = (chargerInfo.distance_along_route !== null && chargerInfo.distance_along_route !== undefined) ?
                                (chargerInfo.distance_along_route / 1609.34).toFixed(1) + ' mi' : 'N/A'
                            const totalDistance = ((chargerInfo.distance_along_route || 0) + (chargerInfo.distance_from_route || 0)) / 1609.34
                            const totalDistanceDisplay = totalDistance > 0 ? totalDistance.toFixed(1) + ' mi' : 'N/A'
                            const arrivalTime = formatEpochMsToLocalTime(chargerInfo.supercharger.arrival_time)
                            console.log(chargerInfo.distance_along_route)

                            if (restaurants.length === 0) {
                                // No restaurants case
                                return (
                                    <tr key={`${station.id}-no-restaurants`} className="border-b-2 border-purple-200">
                                        <td className="px-1 py-0 text-sm text-pink-500 border-r border-pink-200">
                                            <div className="flex flex-col">
                                                <div className="flex">
                                                    <div className="flex-1 flex flex-col">
                                                        <div className="text-xs text-pink-600">
                                                            <strong>{totalDistanceDisplay}</strong> (+ {distanceFromOrigin})
                                                        </div>
                                                        <div className="font-bold">{arrivalTime}</div>
                                                    </div>
                                                </div>
                                                <div className="w-full mb-1">
                                                    <div className="text-xs text-pink-600">{chargerInfo.supercharger.address}</div>
                                                </div>
                                                <div className="flex gap-1">
                                                    <button
                                                        onClick={() => onShowSuperchargerPopup?.(chargerInfo.supercharger.place_id)}
                                                        className="princess-button text-xs px-1 py-0.5 rounded"
                                                    >
                                                        Map
                                                    </button>
                                                    <button
                                                        onClick={() => shareToTesla(
                                                            chargerInfo.supercharger.place_id,
                                                            chargerInfo.supercharger.name,
                                                            chargerInfo.supercharger.google_maps_uri
                                                        )}
                                                        className="princess-button text-xs px-1 py-0.5 rounded bg-princess-accent-mint"
                                                    >
                                                        Send to Tesla
                                                    </button>
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-1 py-0 text-sm text-pink-700" colSpan={3}>
                                            No restaurants found
                                        </td>
                                    </tr>
                                )
                            }

                            // With restaurants case
                            return restaurants.map((restaurant, restaurantIndex) => (
                                <tr
                                    key={`${station.id}-${restaurantIndex}`}
                                    className={`border-b ${restaurantIndex === restaurants.length - 1 ? 'border-b-2 border-purple-200' : 'border-princess-border'} hover:bg-princess-lavender transition-colors`}
                                    data-charger-id={station.id}
                                    data-restaurant-id={restaurant.name}
                                >
                                    {restaurantIndex === 0 && (
                                        <td
                                            className="px-1 py-0 text-sm text-pink-500 border-r border-pink-200"
                                            rowSpan={maxRows}
                                        >
                                            <div className="flex flex-col">
                                                <div className="flex">
                                                    <div className="flex-1 flex flex-col">
                                                        <div className="text-xs text-pink-600">
                                                            <strong>{totalDistanceDisplay}</strong> (+ {distanceFromOrigin})
                                                        </div>
                                                        <div className="font-bold">{arrivalTime}</div>
                                                    </div>
                                                </div>
                                                <div className="w-full mb-1">
                                                    <div className="text-xs text-pink-600">{chargerInfo.supercharger.address}</div>
                                                </div>
                                                <div className="flex gap-1">
                                                    <button
                                                        onClick={() => onShowSuperchargerPopup?.(chargerInfo.supercharger.place_id)}
                                                        className="princess-button text-xs px-1 py-0.5 rounded"
                                                    >
                                                        Map
                                                    </button>
                                                    <button
                                                        onClick={() => shareToTesla(
                                                            chargerInfo.supercharger.place_id,
                                                            chargerInfo.supercharger.name,
                                                            chargerInfo.supercharger.google_maps_uri
                                                        )}
                                                        className="princess-button text-xs px-1 py-0.5 rounded bg-princess-accent-mint"
                                                    >
                                                        Send to Tesla
                                                    </button>
                                                </div>
                                            </div>
                                        </td>
                                    )}
                                    <td className="px-1 py-0 text-sm text-pink-700">
                                        <button
                                            onClick={() => onShowRestaurantPopup?.(restaurant.place_id)}
                                            className="text-pink-600 hover:text-pink-800 cursor-pointer restaurant-link underline decoration-princess-accent-rose hover:decoration-princess-text-primary transition-all duration-300"
                                            dangerouslySetInnerHTML={{
                                                __html: highlightText(restaurant.name, searchTerm)
                                            }}
                                        />
                                    </td>
                                    <td className="px-1 py-0 text-sm text-pink-700">
                                        {restaurant.distance ?
                                            `${Math.round(restaurant.distance)}m` : 'N/A'}
                                    </td>
                                    <td
                                        className="px-1 py-0 text-sm text-pink-700"
                                        data-original-cuisine={restaurant.primary_type_display || 'N/A'}
                                        dangerouslySetInnerHTML={{
                                            __html: highlightText(restaurant.primary_type_display || 'N/A', searchTerm)
                                        }}
                                    />
                                </tr>
                            ))
                        })}
                    </tbody>
                </table>
            </div>
        </div>
    )
}

export default ResultsTable
