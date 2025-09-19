import { createContext, useContext, useState, useCallback, ReactNode } from 'react'

interface Viewport {
    center: [number, number]
    zoom: number
    shouldSync?: boolean // Flag to indicate if map should sync to this viewport
}

interface ViewportContextType {
    viewport: Viewport | null
    setViewport: (viewport: Viewport) => void
    setViewportToRoute: (bounds: L.LatLngBounds) => void
    setViewportToLocation: (center: [number, number], zoom?: number) => void
    updateViewport: (center: [number, number], zoom: number) => void
    restoreSavedViewport: () => void
}

const ViewportContext = createContext<ViewportContextType | undefined>(undefined)

interface ViewportProviderProps {
    children: ReactNode
}

export const ViewportProvider = ({ children }: ViewportProviderProps) => {
    const [viewport, setViewportState] = useState<Viewport | null>(null)

    const setViewport = useCallback((newViewport: Viewport) => {
        console.log('VIEWPORT: Setting viewport to:', newViewport)
        setViewportState(newViewport)
    }, [])

    const setViewportToRoute = useCallback((bounds: L.LatLngBounds) => {
        // Calculate center and zoom from bounds
        const center = bounds.getCenter()
        const newViewport = {
            center: [center.lat, center.lng] as [number, number],
            zoom: 6, // Default zoom for route view
            shouldSync: true
        }
        console.log('VIEWPORT: Setting viewport to route bounds:', newViewport)
        setViewportState(newViewport)
    }, [])

    const setViewportToLocation = useCallback((center: [number, number], zoom: number = 18) => {
        const newViewport = { center, zoom, shouldSync: true }
        console.log('VIEWPORT: Setting viewport to location:', newViewport)
        setViewportState(newViewport)
    }, [])

    const updateViewport = useCallback((center: [number, number], zoom: number) => {
        const newViewport = { center, zoom, shouldSync: false } // Don't sync back to map
        console.log('VIEWPORT: Updating viewport:', newViewport)
        setViewportState(newViewport)
    }, [])

    const restoreSavedViewport = useCallback(() => {
        if (viewport) {
            const restoredViewport = { ...viewport, shouldSync: true }
            console.log('VIEWPORT: Restoring saved viewport with sync:', restoredViewport)
            setViewportState(restoredViewport)
        } else {
            console.log('VIEWPORT: No saved viewport to restore')
        }
    }, [viewport])

    return (
        <ViewportContext.Provider value={{
            viewport,
            setViewport,
            setViewportToRoute,
            setViewportToLocation,
            updateViewport,
            restoreSavedViewport
        }}>
            {children}
        </ViewportContext.Provider>
    )
}

export const useViewport = () => {
    const context = useContext(ViewportContext)
    if (context === undefined) {
        throw new Error('useViewport must be used within a ViewportProvider')
    }
    return context
}
