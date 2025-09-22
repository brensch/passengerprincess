import React, { useState, useEffect } from 'react'
import { Supercharger, Restaurant } from '../types'
import { getRestaurantEmoji } from '../utils/restaurantEmojiMapping'
import { globalViewportData } from './Map'

interface SuperchargerPopupProps {
    supercharger: Supercharger
    etaInfo?: {
        arrival_time: number
        distance_from_route: number
        distance_along_route: number
    }
    chargerId?: string
    onViewInResults?: (chargerId: string) => void
    onZoom?: (lat: number, lng: number) => void
}

interface RestaurantPopupProps {
    restaurant: Restaurant
    distanceText?: string
}

type PopupContentProps =
    | ({ type: 'supercharger' } & SuperchargerPopupProps)
    | ({ type: 'restaurant' } & RestaurantPopupProps)

const formatEpochMsToLocalTime = (epochMs: number): string => {
    const date = new Date(epochMs)
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const PopupContent: React.FC<PopupContentProps> = (props) => {
    if (props.type === 'supercharger') {
        const { supercharger, etaInfo, chargerId, onViewInResults, onZoom } = props

        const [restaurantCount, setRestaurantCount] = useState(() =>
            globalViewportData.mappings.filter(m => m.supercharger_id === supercharger.place_id).length
        )

        useEffect(() => {
            const updateCount = () => {
                setRestaurantCount(globalViewportData.mappings.filter(m => m.supercharger_id === supercharger.place_id).length)
            }
            globalViewportData.listeners.push(updateCount)
            return () => {
                globalViewportData.listeners = globalViewportData.listeners.filter(l => l !== updateCount)
            }
        }, [supercharger.place_id])

        return (
            <div className="font-sans p-3 bg-gradient-to-br from-princess-lavender via-princess-lilac to-princess-rose rounded-lg shadow-lg border border-princess-border">
                <h3 className="font-semibold text-lg mb-0 text-princess-text-primary">{supercharger.name}</h3>
                <p className="text-sm text-princess-text-secondary mb-1">{supercharger.address}</p>
                {etaInfo ? (
                    <div className="mt-1 text-sm text-princess-text-primary grid grid-cols-[auto_1fr] gap-x-3 items-center leading-tight">
                        <span className="font-semibold">Arrival</span>
                        <span>{formatEpochMsToLocalTime(etaInfo.arrival_time)}</span>
                        <span className="font-semibold">Distance</span>
                        <span>{((etaInfo.distance_along_route || 0) + (etaInfo.distance_from_route || 0)) / 1609.34 > 0 ? (((etaInfo.distance_along_route || 0) + (etaInfo.distance_from_route || 0)) / 1609.34).toFixed(1) + ' mi' : 'N/A'}</span>
                        <span className="font-semibold">Deviation</span>
                        <span>{(etaInfo.distance_from_route / 1609.34).toFixed(1)} mi</span>
                        <span className="font-semibold">Food spots</span>
                        <span>{restaurantCount}</span>
                    </div>
                ) : (
                    <p className="text-sm text-princess-text-primary mt-1"><strong>Food spots:</strong> {restaurantCount}</p>
                )}
                <div className="flex flex-row flex-wrap gap-1.5 mt-2">
                    {chargerId && onViewInResults && (
                        <button
                            onClick={() => onViewInResults(chargerId)}
                            className="px-3 py-1.5 bg-princess-lavender text-princess-text-primary rounded-md text-sm font-medium transition-all duration-200"
                        >
                            View in Results
                        </button>
                    )}
                    {onZoom && (
                        <button
                            onClick={() => onZoom(supercharger.latitude, supercharger.longitude)}
                            className="px-3 py-1.5 bg-princess-mint text-princess-text-primary rounded-md text-sm font-medium transition-all duration-200"
                        >
                            Zoom
                        </button>
                    )}
                    <button
                        onClick={() => window.open(supercharger.google_maps_uri, '_blank')}
                        className="px-3 py-1.5 bg-princess-peach text-princess-text-primary rounded-md text-sm font-medium transition-all duration-200"
                    >
                        Open in Maps
                    </button>
                </div>
            </div>
        )
    } else {
        const { restaurant, distanceText } = props

        const emoji = getRestaurantEmoji(restaurant.primary_type)

        return (
            <div className="font-sans p-3 bg-gradient-to-br from-princess-lavender via-princess-lilac to-princess-rose rounded-lg shadow-lg border border-princess-border">
                <h3 className="font-semibold text-lg mb-1 text-princess-text-primary">{restaurant.name}</h3>
                <p className="text-sm text-princess-text-secondary mb-1">{emoji} {restaurant.primary_type_display || 'Restaurant'}</p>
                {distanceText && (
                    <p className="text-sm text-princess-text-secondary mb-1">{distanceText}</p>
                )}
                <button
                    onClick={() => window.open(restaurant.google_maps_uri, '_blank')}
                    className="mt-2 px-3 py-1.5 bg-princess-mint text-princess-text-primary rounded-md text-sm font-medium transition-all duration-200"
                >
                    Open in Maps
                </button>
            </div>
        )
    }
}

export default PopupContent
