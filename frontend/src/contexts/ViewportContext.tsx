import { createContext, useContext, useState, useCallback, ReactNode } from 'react'

interface Viewport {
    center: [number, number]
    zoom: number
    shouldSync?: boolean // Flag to indicate if map should sync to this viewport
}

interface ViewportContextType {
    viewport: Viewport | null
    setViewport: (viewport: Viewport) => void
    setViewportToRoute: (bounds: L.LatLngBounds, center?: L.LatLng, zoom?: number, shouldSync?: boolean) => void
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
        setViewportState(newViewport)
    }, [])

    const setViewportToRoute = useCallback((bounds: L.LatLngBounds, center?: L.LatLng, zoom?: number, shouldSync: boolean = true) => {
        // Use provided center/zoom if available, otherwise calculate from bounds
        const viewportCenter = center ? [center.lat, center.lng] as [number, number] : [bounds.getCenter().lat, bounds.getCenter().lng] as [number, number]

        // Calculate appropriate zoom if not provided
        let viewportZoom = zoom || 10 // Default fallback zoom
        if (zoom === undefined) {
            // Calculate zoom level based on bounds size
            const latDiff = bounds.getNorth() - bounds.getSouth()
            const lngDiff = bounds.getEast() - bounds.getWest()
            const maxDiff = Math.max(latDiff, lngDiff)

            if (maxDiff < 0.01) viewportZoom = 15
            else if (maxDiff < 0.05) viewportZoom = 12
            else if (maxDiff < 0.1) viewportZoom = 10
            else if (maxDiff < 0.5) viewportZoom = 8
            else if (maxDiff < 1) viewportZoom = 7
            else if (maxDiff < 2) viewportZoom = 6
            else if (maxDiff < 5) viewportZoom = 5
            else viewportZoom = 4
        }

        const newViewport = {
            center: viewportCenter,
            zoom: viewportZoom,
            shouldSync
        }
        setViewportState(newViewport)
    }, [])

    const setViewportToLocation = useCallback((center: [number, number], zoom: number = 18) => {
        const newViewport = { center, zoom, shouldSync: true }
        setViewportState(newViewport)
    }, [])

    const updateViewport = useCallback((center: [number, number], zoom: number) => {
        const newViewport = { center, zoom, shouldSync: false } // Don't sync back to map
        setViewportState(newViewport)
    }, [])

    const restoreSavedViewport = useCallback(() => {
        if (viewport) {
            const restoredViewport = { ...viewport, shouldSync: true }
            setViewportState(restoredViewport)
        } else {
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
