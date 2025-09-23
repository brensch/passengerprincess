import { useState, useRef, useEffect } from 'react'
import { AutocompleteResult } from '../types'

interface SearchFormProps {
    onSearch: (origin: string, destination: string) => void
    isLoading: boolean
    statusMessage: string
    isError: boolean
    userLocation: [number, number] | null
    initialOrigin?: string
    initialDestination?: string
}

const SearchForm = ({ onSearch, isLoading, statusMessage, isError, userLocation: _userLocation, initialOrigin = '', initialDestination = '' }: SearchFormProps) => {
    const [origin, setOrigin] = useState('')
    const [destination, setDestination] = useState('')
    const [originSuggestions, setOriginSuggestions] = useState<AutocompleteResult[]>([])
    const [destinationSuggestions, setDestinationSuggestions] = useState<AutocompleteResult[]>([])
    const [showOriginSuggestions, setShowOriginSuggestions] = useState(false)
    const [showDestinationSuggestions, setShowDestinationSuggestions] = useState(false)
    const [selectedOriginIndex, setSelectedOriginIndex] = useState(-1)
    const [selectedDestinationIndex, setSelectedDestinationIndex] = useState(-1)
    const [originSessionToken, setOriginSessionToken] = useState<string | null>(null)
    const [destinationSessionToken, setDestinationSessionToken] = useState<string | null>(null)
    const [errorMessage, setErrorMessage] = useState<string | null>(null)
    const [isApiError, setIsApiError] = useState(false)
    const [fullError, setFullError] = useState<string | null>(null)
    const [showErrorPopup, setShowErrorPopup] = useState(false)

    const originRef = useRef<HTMLInputElement>(null)
    const destinationRef = useRef<HTMLInputElement>(null)
    const debounceRef = useRef<NodeJS.Timeout | number>()

    useEffect(() => {
        const dropdowns = document.querySelectorAll('.mobile-dropdown')

        const handleTouchStart = (e: Event) => {
            // Prevent page scroll when touching dropdown
            e.stopPropagation()
        }

        const handleTouchMove = (e: Event) => {
            // Let the dropdown handle its own scrolling
            e.stopPropagation()
        }

        const handleTouchEnd = (e: Event) => {
            // Prevent any default behavior on touch end
            e.stopPropagation()
        }

        dropdowns.forEach(dropdown => {
            dropdown.addEventListener('touchstart', handleTouchStart)
            dropdown.addEventListener('touchmove', handleTouchMove)
            dropdown.addEventListener('touchend', handleTouchEnd)
        })

        // Add a listener to the window to see if page is scrolling
        const handleWindowScroll = () => {
            // Debug: window scrolled
        }
        window.addEventListener('scroll', handleWindowScroll)

        return () => {
            dropdowns.forEach(dropdown => {
                dropdown.removeEventListener('touchstart', handleTouchStart)
                dropdown.removeEventListener('touchmove', handleTouchMove)
                dropdown.removeEventListener('touchend', handleTouchEnd)
            })
            window.removeEventListener('scroll', handleWindowScroll)
        }
    }, [showOriginSuggestions, showDestinationSuggestions])

    useEffect(() => {
        const originInput = originRef.current
        const destinationInput = destinationRef.current

        const handleFocus = (event: FocusEvent) => {
            const target = event.target as HTMLInputElement
            // Small delay to allow keyboard to appear
            setTimeout(() => {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'center',
                    inline: 'nearest'
                })
            }, 300)
        }

        if (originInput) {
            originInput.addEventListener('focus', handleFocus)
        }
        if (destinationInput) {
            destinationInput.addEventListener('focus', handleFocus)
        }

        return () => {
            if (originInput) {
                originInput.removeEventListener('focus', handleFocus)
            }
            if (destinationInput) {
                destinationInput.removeEventListener('focus', handleFocus)
            }
        }
    }, [])

    useEffect(() => {
        if (!isError) {
            setFullError(null)
        }
    }, [isError])

    // Check URL params once on mount and populate inputs if they exist
    useEffect(() => {
        const params = new URLSearchParams(window.location.search)
        const urlOrigin = params.get('origin')
        const urlDestination = params.get('destination')

        if (urlOrigin) {
            setOrigin(decodeURIComponent(urlOrigin))
        } else if (initialOrigin) {
            setOrigin(initialOrigin)
        }
        if (urlDestination) {
            setDestination(decodeURIComponent(urlDestination))
        } else if (initialDestination) {
            setDestination(initialDestination)
        }
    }, []) // Empty dependency array - only run once on mount

    const handleSearch = (e: any) => {
        e.preventDefault()

        if (origin.trim() && destination.trim() && !isLoading) {
            setErrorMessage(null)
            setIsApiError(false)
            setFullError(null)
            onSearch(origin.trim(), destination.trim())
        }
    }

    const handleOriginChange = (value: string) => {
        setOrigin(value)
        setSelectedOriginIndex(-1)
        setErrorMessage(null)
        setIsApiError(false)
        setFullError(null)

        // Clear session token if input is cleared or significantly changed
        if (value.trim() === '') {
            setOriginSessionToken(null)
        }

        fetchSuggestions(value, 'origin')
    }

    const handleDestinationChange = (value: string) => {
        setDestination(value)
        setSelectedDestinationIndex(-1)
        setErrorMessage(null)
        setIsApiError(false)
        setFullError(null)

        // Clear session token if input is cleared or significantly changed
        if (value.trim() === '') {
            setDestinationSessionToken(null)
        }

        fetchSuggestions(value, 'destination')
    }

    const fetchSuggestions = (query: string, type: 'origin' | 'destination') => {
        if (debounceRef.current) {
            clearTimeout(debounceRef.current)
        }

        if (query.length < 3) {
            if (type === 'origin') {
                setOriginSuggestions([])
                setShowOriginSuggestions(false)
            } else {
                setDestinationSuggestions([])
                setShowDestinationSuggestions(false)
            }
            return
        }

        debounceRef.current = setTimeout(async () => {
            try {
                // Get the current session token for this input type
                const currentSessionToken = type === 'origin' ? originSessionToken : destinationSessionToken

                // Build URL with session token if available
                let url = `/autocomplete?partial=${encodeURIComponent(query)}`
                if (currentSessionToken) {
                    url += `&session_token=${currentSessionToken}`
                }

                const response = await fetch(url)
                const data = await response.json()

                if (response.ok && data.predictions) {
                    // Store the new session token for subsequent requests
                    if (data.session_token) {
                        if (type === 'origin') {
                            setOriginSessionToken(data.session_token)
                        } else {
                            setDestinationSessionToken(data.session_token)
                        }
                    }

                    if (type === 'origin') {
                        setOriginSuggestions(data.predictions)
                        setShowOriginSuggestions(true)
                    } else {
                        setDestinationSuggestions(data.predictions)
                        setShowDestinationSuggestions(true)
                    }
                }
            } catch (error) {
                console.error('Autocomplete error:', error)
                setErrorMessage('I couldn\'t fetch suggestions right now. Just type in the address and it will still work.')
                setIsApiError(true)
                setFullError(error instanceof Error ? error.message : 'Unknown error')
            }
        }, 300)
    }

    const handleSuggestionClick = (suggestion: AutocompleteResult, type: 'origin' | 'destination') => {
        if (suggestion.isMyLocation) {
            handleMyLocationSelection(type)
        } else {
            if (type === 'origin') {
                setOrigin(suggestion.description)
                setShowOriginSuggestions(false)
                // Clear session token after selection to complete the session
                setOriginSessionToken(null)
            } else {
                setDestination(suggestion.description)
                setShowDestinationSuggestions(false)
                // Clear session token after selection to complete the session
                setDestinationSessionToken(null)
            }
        }
    }

    const handleMyLocationSelection = async (type: 'origin' | 'destination') => {
        try {
            const setValue = type === 'origin' ? setOrigin : setDestination
            setValue('Getting your location...')

            // Get GPS location
            const position = await new Promise<GeolocationPosition>((resolve, reject) =>
                navigator.geolocation.getCurrentPosition(resolve, reject, {
                    enableHighAccuracy: true,
                    timeout: 20000
                })
            )

            const coords = position.coords
            const latitude = coords.latitude
            const longitude = coords.longitude
            const address = await reverseGeocode(latitude, longitude)
            setValue(address)

            if (type === 'origin') {
                setShowOriginSuggestions(false)
            } else {
                setShowDestinationSuggestions(false)
            }
        } catch (error) {
            console.error('Geolocation error:', error)
            const setValue = type === 'origin' ? setOrigin : setDestination
            setValue('')
            setErrorMessage('I can\'t find you. Please allow location access.')
            setIsApiError(false)
            setFullError(null)
        }
    }

    const reverseGeocode = async (lat: number, lon: number): Promise<string> => {
        try {
            // Use our backend reverse geocoding endpoint
            const response = await fetch(`/reverse-geocode?lat=${lat}&lon=${lon}`)
            if (response.ok) {
                const data = await response.json()
                if (data.address) {
                    return data.address
                }
            }
        } catch (error) {
            console.error('Reverse geocoding error:', error)
        }

        // Use coordinate format as fallback
        return `${lat.toFixed(6)}, ${lon.toFixed(6)}`
    }

    const handleKeyDown = (e: any, type: 'origin' | 'destination') => {
        const suggestions = type === 'origin' ? originSuggestions : destinationSuggestions
        const selectedIndex = type === 'origin' ? selectedOriginIndex : selectedDestinationIndex
        const setSelectedIndex = type === 'origin' ? setSelectedOriginIndex : setSelectedDestinationIndex
        const showSuggestions = type === 'origin' ? showOriginSuggestions : showDestinationSuggestions

        if (!showSuggestions || suggestions.length === 0) {
            if (e.key === 'Enter') {
                e.preventDefault()
                handleSearch(e as any)
            }
            return
        }

        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault()
                setSelectedIndex(Math.min(selectedIndex + 1, suggestions.length - 1))
                break
            case 'ArrowUp':
                e.preventDefault()
                setSelectedIndex(Math.max(selectedIndex - 1, -1))
                break
            case 'Enter':
                e.preventDefault()
                if (selectedIndex >= 0 && selectedIndex < suggestions.length) {
                    handleSuggestionClick(suggestions[selectedIndex], type)
                }
                break
            case 'Escape':
                if (type === 'origin') {
                    setShowOriginSuggestions(false)
                } else {
                    setShowDestinationSuggestions(false)
                }
                setSelectedIndex(-1)
                break
        }
    }

    const handleFocus = (type: 'origin' | 'destination') => {
        const value = type === 'origin' ? origin : destination

        if (value.trim() === '') {
            if (type === 'origin') {
                const myLocationOption: AutocompleteResult = {
                    description: "My Location",
                    place_id: "my_location",
                    isMyLocation: true
                }
                setOriginSuggestions([myLocationOption])
                setShowOriginSuggestions(true)
            } else {
                // Don't show "My Location" for destination
                setDestinationSuggestions([])
                setShowDestinationSuggestions(false)
            }
        }
    }

    const handleBlur = (type: 'origin' | 'destination') => {
        setTimeout(() => {
            if (type === 'origin') {
                setShowOriginSuggestions(false)
                setSelectedOriginIndex(-1)
            } else {
                setShowDestinationSuggestions(false)
                setSelectedDestinationIndex(-1)
            }
        }, 150)
    }

    return (
        <div className="w-full max-w-2xl p-8">
            <div className="text-center mb-8">
                <h1 className="text-6xl font-dancing text-princess-text-primary mb-4">
                    Passenger Princess Protector
                </h1>
            </div>

            <form onSubmit={handleSearch} className="space-y-6">
                <div className="relative">
                    <div className="flex items-center">
                        <span className="absolute left-4 text-2xl pointer-events-none">👸</span>
                        <input
                            ref={originRef}
                            type="text"
                            value={origin}
                            onChange={(e) => handleOriginChange(e.target.value)}
                            onKeyDown={(e) => handleKeyDown(e, 'origin')}
                            onFocus={() => handleFocus('origin')}
                            onBlur={() => handleBlur('origin')}
                            placeholder="Current Location"
                            className={`w-full pl-16 pr-6 py-4 text-lg rounded-2xl border-2 border-princess-border 
                         bg-princess-surface focus:outline-none focus:ring-2 focus:ring-princess-accent-lavender 
                         focus:border-transparent transition-all duration-300 
                         ${isLoading ? 'bg-gray-100 text-gray-500 cursor-not-allowed' : ''}`}
                            disabled={isLoading}
                        />
                    </div>

                    {showOriginSuggestions && originSuggestions.length > 0 && (
                        <div className="absolute top-full left-0 right-0 mt-2 bg-princess-surface border-2 border-princess-border 
                          rounded-xl shadow-lg max-h-48 overflow-y-auto z-50 mobile-dropdown
                          overscroll-contain touch-pan-y"
                            style={{
                                WebkitOverflowScrolling: 'touch',
                                touchAction: 'pan-y',
                                overscrollBehavior: 'contain'
                            }}>
                            {originSuggestions.map((suggestion, index) => (
                                <div
                                    key={suggestion.place_id}
                                    className={`px-4 py-3 cursor-pointer transition-colors border-b border-princess-border last:border-b-0
                            ${index === selectedOriginIndex
                                            ? 'bg-princess-accent-lavender text-princess-text-primary'
                                            : 'text-princess-text-primary hover:bg-princess-accent-lavender'
                                        }`}
                                    onMouseDown={() => handleSuggestionClick(suggestion, 'origin')}
                                >
                                    {suggestion.description}
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                <div className="relative">
                    <div className="flex items-center">
                        <span className="absolute left-4 text-2xl pointer-events-none">📌</span>
                        <input
                            ref={destinationRef}
                            type="text"
                            value={destination}
                            onChange={(e) => handleDestinationChange(e.target.value)}
                            onKeyDown={(e) => handleKeyDown(e, 'destination')}
                            onFocus={() => handleFocus('destination')}
                            onBlur={() => handleBlur('destination')}
                            placeholder="Preferred Place of Passage"
                            className={`w-full pl-16 pr-6 py-4 text-lg rounded-2xl border-2 border-princess-border 
                         bg-princess-surface focus:outline-none focus:ring-2 focus:ring-princess-accent-lavender 
                         focus:border-transparent transition-all duration-300 
                         ${isLoading ? 'bg-gray-100 text-gray-500 cursor-not-allowed' : ''}`}
                            disabled={isLoading}
                        />
                    </div>

                    {showDestinationSuggestions && destinationSuggestions.length > 0 && (
                        <div className="absolute top-full left-0 right-0 mt-2 bg-princess-surface border-2 border-princess-border 
                          rounded-xl shadow-lg max-h-48 overflow-y-auto z-50 mobile-dropdown
                          overscroll-contain touch-pan-y"
                            style={{
                                WebkitOverflowScrolling: 'touch',
                                touchAction: 'pan-y',
                                overscrollBehavior: 'contain'
                            }}>
                            {destinationSuggestions.map((suggestion, index) => (
                                <div
                                    key={suggestion.place_id}
                                    className={`px-4 py-3 cursor-pointer transition-colors border-b border-princess-border last:border-b-0
                            ${index === selectedDestinationIndex
                                            ? 'bg-princess-accent-lavender text-princess-text-primary'
                                            : 'text-princess-text-primary hover:bg-princess-accent-lavender'
                                        }`}
                                    onMouseDown={() => handleSuggestionClick(suggestion, 'destination')}
                                >
                                    {suggestion.description}
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                <button
                    type="submit"
                    disabled={!origin.trim() || !destination.trim() || isLoading}
                    className="w-full py-4 px-8 text-xl font-semibold rounded-2xl bg-gradient-to-r 
                   from-princess-accent-lavender to-princess-accent-rose text-princess-text-primary
                   hover:from-princess-accent-rose hover:to-princess-accent-lavender
                   disabled:opacity-50 disabled:cursor-not-allowed
                   transition-all duration-300 transform hover:scale-105 disabled:hover:scale-100
                   shadow-lg hover:shadow-xl flex items-center justify-center"
                >
                    {isLoading ? (
                        <>
                            <div className="w-5 h-5 border-2 border-princess-text-primary border-t-transparent rounded-full animate-spin mr-2"></div>
                            Pondering Possible Paths
                        </>
                    ) : 'Plan Princess Portage'}
                </button>
            </form>

            {(errorMessage || statusMessage) && (
                <div className={`mt-6 p-4 rounded-xl text-center font-medium ${(errorMessage && isApiError) || isError
                    ? 'bg-gradient-to-r from-princess-rose to-princess-blush text-princess-text-primary border border-princess-accent-rose'
                    : 'bg-gradient-to-r from-princess-surface-soft to-princess-lavender text-princess-text-secondary'
                    }`}>
                    <div className="flex items-center justify-between">
                        <span>Dear Princess, {errorMessage || (isError ? "I can't route you. Try again in a little bit." : statusMessage)}</span>
                        {((isApiError || isError) && (fullError || (isError && statusMessage))) && (
                            <button
                                onClick={() => {
                                    if (isError && !fullError) setFullError(statusMessage)
                                    setShowErrorPopup(true)
                                }}
                                className="ml-2 px-2 py-1 text-sm bg-princess-accent-lavender text-princess-text-primary rounded hover:bg-princess-accent-rose transition-colors"
                            >
                                Details
                            </button>
                        )}
                    </div>
                </div>
            )}

            {showErrorPopup && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onClick={() => setShowErrorPopup(false)}>
                    <div className="bg-princess-surface p-6 rounded-xl max-w-md w-full mx-4 max-h-[80dvh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
                        <h3 className="text-lg font-semibold mb-4 text-princess-text-primary">Error Details</h3>
                        <p className="text-princess-text-secondary mb-4 whitespace-pre-wrap">{fullError}</p>
                        <button
                            onClick={() => setShowErrorPopup(false)}
                            className="px-4 py-2 bg-princess-accent-lavender text-princess-text-primary rounded hover:bg-princess-accent-rose transition-colors"
                        >
                            Close
                        </button>
                    </div>
                </div>
            )}
        </div>
    )
}

export default SearchForm
