import { StationData } from '../types'

interface ResultsTableProps {
    stationData: StationData[]
    searchTerm: string
    cuisineFilters: string[]
    isLoading: boolean
    statusMessage: string
    className?: string
}

const ResultsTable = ({
    stationData,
    searchTerm,
    cuisineFilters,
    isLoading,
    statusMessage,
    className = ''
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
                <table className="w-full text-sm">
                    <thead className="bg-princess-surface-soft sticky top-0 z-10">
                        <tr className="border-b-2 border-princess-border">
                            <th className="text-left p-3 font-semibold text-princess-text-primary">City / ETA</th>
                            <th className="text-left p-3 font-semibold text-princess-text-primary">Restaurant</th>
                            <th className="text-left p-3 font-semibold text-princess-text-primary">Cuisine</th>
                            <th className="text-left p-3 font-semibold text-princess-text-primary">Rating</th>
                            <th className="text-left p-3 font-semibold text-princess-text-primary">Distance</th>
                            <th className="text-left p-3 font-semibold text-princess-text-primary">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {stationData.map((station, stationIndex) => {
                            const { chargerInfo } = station
                            const restaurants = station.restaurants || []

                            return restaurants.map((restaurant, restaurantIndex) => (
                                <tr
                                    key={`${station.id}-${restaurantIndex}`}
                                    className="border-b border-princess-border hover:bg-princess-lavender transition-colors"
                                >
                                    <td className="p-3">
                                        {restaurantIndex === 0 && (
                                            <div>
                                                <button
                                                    onClick={() => openInGoogleMaps(chargerInfo.supercharger.google_maps_uri)}
                                                    className="text-princess-text-secondary hover:text-princess-text-primary 
                                   underline decoration-princess-accent-rose hover:decoration-princess-text-primary
                                   transition-all duration-300"
                                                >
                                                    {chargerInfo.supercharger.name}
                                                </button>
                                                {chargerInfo.eta_ms && (
                                                    <div className="text-xs text-princess-text-accent mt-1">
                                                        ETA: {formatEpochMsToLocalTime(chargerInfo.eta_ms)}
                                                    </div>
                                                )}
                                            </div>
                                        )}
                                    </td>
                                    <td className="p-3">
                                        <button
                                            onClick={() => openInGoogleMaps(restaurant.google_maps_uri)}
                                            className="text-princess-text-secondary hover:text-princess-text-primary 
                               underline decoration-princess-accent-rose hover:decoration-princess-text-primary
                               transition-all duration-300"
                                            dangerouslySetInnerHTML={{
                                                __html: highlightText(restaurant.name, searchTerm)
                                            }}
                                        />
                                    </td>
                                    <td
                                        className="p-3 text-princess-text-accent"
                                        dangerouslySetInnerHTML={{
                                            __html: highlightText(restaurant.primary_type_display || '', searchTerm)
                                        }}
                                    />
                                    <td className="p-3 text-princess-text-accent">
                                        {restaurant.rating ? (
                                            <span>
                                                {'⭐'.repeat(Math.floor(restaurant.rating))} ({restaurant.rating})
                                            </span>
                                        ) : (
                                            'N/A'
                                        )}
                                    </td>
                                    <td className="p-3 text-princess-text-accent">
                                        {restaurant.distance_from_charger ?
                                            `${Math.round(restaurant.distance_from_charger)}m` :
                                            'N/A'
                                        }
                                    </td>
                                    <td className="p-3">
                                        <div className="flex gap-2">
                                            <button
                                                onClick={() => openInGoogleMaps(restaurant.google_maps_uri)}
                                                className="px-2 py-1 text-xs bg-princess-accent-mint text-princess-text-primary 
                                 rounded hover:bg-princess-accent-peach transition-colors"
                                            >
                                                🗺️
                                            </button>
                                            {navigator.share && (
                                                <button
                                                    onClick={() => {
                                                        const shareData = {
                                                            title: restaurant.name,
                                                            text: `Check out this restaurant: ${restaurant.name}`,
                                                            url: restaurant.google_maps_uri || ''
                                                        }
                                                        navigator.share(shareData).catch(console.error)
                                                    }}
                                                    className="px-2 py-1 text-xs bg-princess-accent-lavender text-princess-text-primary 
                                   rounded hover:bg-princess-accent-rose transition-colors"
                                                >
                                                    📤
                                                </button>
                                            )}
                                        </div>
                                    </td>
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
