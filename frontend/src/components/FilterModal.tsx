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

    useEffect(() => {
        setLocalSearchTerm(searchTerm)
        setLocalCuisineFilters(cuisineFilters)
    }, [searchTerm, cuisineFilters])

    useEffect(() => {
        // Extract unique cuisines from station data
        const cuisines = new Set<string>()
        stationData.forEach(station => {
            station.restaurants.forEach(restaurant => {
                if (restaurant.primary_type_display) {
                    cuisines.add(restaurant.primary_type_display.trim())
                }
            })
        })
        setAvailableCuisines(Array.from(cuisines).sort())
    }, [stationData])

    const handleCuisineToggle = (cuisine: string) => {
        if (cuisine === '') {
            // "All Cuisines" selected
            setLocalCuisineFilters([''])
        } else {
            // Individual cuisine toggle
            if (localCuisineFilters.includes('') || localCuisineFilters.length === 0) {
                // If "All" is selected or nothing is selected, replace with this cuisine
                setLocalCuisineFilters([cuisine])
            } else if (localCuisineFilters.includes(cuisine)) {
                // Remove cuisine
                const newFilters = localCuisineFilters.filter(c => c !== cuisine)
                setLocalCuisineFilters(newFilters.length === 0 ? [''] : newFilters)
            } else {
                // Add cuisine
                setLocalCuisineFilters([...localCuisineFilters.filter(c => c !== ''), cuisine])
            }
        }
    }

    const handleApply = () => {
        onFilter(localSearchTerm, localCuisineFilters)
        onClose()
    }

    const handleClear = () => {
        setLocalSearchTerm('')
        setLocalCuisineFilters([''])
    }

    if (!isOpen) return null

    return (
        <div
            className="fixed inset-0 flex items-center justify-center z-[1200] bg-black bg-opacity-50 backdrop-blur-sm"
            onClick={(e) => e.target === e.currentTarget && onClose()}
        >
            <div className="rounded-xl max-w-lg w-full mx-4 shadow-2xl bg-princess-surface border-2 border-princess-border flex flex-col h-[80vh]">
                <div className="p-6 flex-1 overflow-y-auto min-h-0">
                    <input
                        type="text"
                        value={localSearchTerm}
                        onChange={(e) => setLocalSearchTerm(e.target.value)}
                        className="w-full px-4 py-3 rounded-md shadow-sm focus:outline-none focus:ring-2 
                     focus:ring-princess-accent-lavender text-base bg-princess-surface 
                     border border-princess-border mb-4"
                        placeholder="Pick places please"
                    />

                    <div className="flex-1 overflow-y-auto overscroll-contain">
                        <div className="flex flex-wrap gap-2 p-1">
                            <button
                                onClick={() => handleCuisineToggle('')}
                                className={`px-2 py-1 rounded-lg text-xs font-medium transition-all duration-200 
                        border touch-manipulation select-none ${localCuisineFilters.includes('') || localCuisineFilters.length === 0
                                        ? 'bg-princess-accent-lavender text-princess-text-primary border-princess-accent-lavender'
                                        : 'bg-princess-surface text-princess-text-primary border-princess-border md:hover:bg-princess-accent-lavender active:scale-95'
                                    }`}
                            >
                                All Cuisines
                            </button>

                            {availableCuisines.map(cuisine => (
                                <button
                                    key={cuisine}
                                    onClick={() => handleCuisineToggle(cuisine)}
                                    className={`px-2 py-1 rounded-lg text-xs font-medium transition-all duration-200 
                          border touch-manipulation select-none ${localCuisineFilters.includes(cuisine)
                                            ? 'bg-princess-accent-lavender text-princess-text-primary border-princess-accent-lavender'
                                            : 'bg-princess-surface text-princess-text-primary border-princess-border md:hover:bg-princess-accent-lavender active:scale-95'
                                        }`}
                                >
                                    {cuisine}
                                </button>
                            ))}
                        </div>
                    </div>
                </div>

                <div className="p-4 border-t border-princess-border flex gap-3">
                    <button
                        onClick={handleClear}
                        className="flex-1 px-4 py-3 text-sm font-medium rounded-lg transition-all duration-300
                     bg-transparent text-princess-text-primary border-2 border-princess-border
                     hover:bg-princess-surface-soft"
                    >
                        Clear All
                    </button>
                    <button
                        onClick={handleApply}
                        className="flex-1 px-4 py-3 text-sm font-medium rounded-lg transition-all duration-300
                     bg-princess-accent-lavender text-princess-text-primary border-none
                     hover:bg-princess-accent-rose"
                    >
                        Perfect!
                    </button>
                </div>
            </div>
        </div>
    )
}

export default FilterModal
