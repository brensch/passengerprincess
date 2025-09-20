import { createContext, useContext, useState, useCallback, ReactNode } from 'react'

interface Viewport {
    center?: [number, number]
    zoom?: number
    bounds?: L.LatLngBounds
    shouldSync?: boolean // Flag to indicate if map should sync to this viewport
}

interface ViewportContextType {
    viewport: Viewport | null
    setViewport: (viewport: Viewport) => void
    setViewportToRoute: (bounds: L.LatLngBounds, shouldSync?: boolean) => void
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

    const setViewportToRoute = useCallback((bounds: L.LatLngBounds, shouldSync: boolean = true) => {
        const newViewport = {
            bounds,
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
