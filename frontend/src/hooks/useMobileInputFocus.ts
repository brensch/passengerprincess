import { useEffect, useRef } from 'react'

export const useMobileInputFocus = () => {
    const inputRefs = useRef<Set<HTMLInputElement>>(new Set())

    useEffect(() => {
        const isAndroid = /Android/i.test(navigator.userAgent)
        
        const handleFocus = (event: FocusEvent) => {
            const target = event.target as HTMLInputElement
            if (target.tagName === 'INPUT' && target.type === 'text') {
                // Small delay to allow keyboard to appear
                setTimeout(() => {
                    target.scrollIntoView({
                        behavior: 'smooth',
                        block: 'center',
                        inline: 'nearest'
                    })
                }, 300)
            }
        }

        // Add focus listener to document for all inputs
        document.addEventListener('focusin', handleFocus)

        // On Android, also add touch event listeners to dropdowns to enable scrolling
        if (isAndroid) {
            const handleTouchStart = (event: TouchEvent) => {
                const target = event.target as HTMLElement
                const dropdown = target.closest('.mobile-dropdown') as HTMLElement
                if (dropdown) {
                    // Prevent parent scrolling when touching dropdown
                    event.stopPropagation()
                }
            }

            const handleTouchMove = (event: TouchEvent) => {
                const target = event.target as HTMLElement
                const dropdown = target.closest('.mobile-dropdown') as HTMLElement
                if (dropdown) {
                    // Allow scrolling within dropdown
                    const scrollTop = dropdown.scrollTop
                    const scrollHeight = dropdown.scrollHeight
                    const clientHeight = dropdown.clientHeight
                    
                    // Prevent default if trying to scroll beyond bounds
                    if ((scrollTop === 0 && event.touches[0].clientY > event.touches[0].clientY) ||
                        (scrollTop >= scrollHeight - clientHeight && event.touches[0].clientY < event.touches[0].clientY)) {
                        event.preventDefault()
                    }
                }
            }

            document.addEventListener('touchstart', handleTouchStart, { passive: false })
            document.addEventListener('touchmove', handleTouchMove, { passive: false })

            return () => {
                document.removeEventListener('focusin', handleFocus)
                document.removeEventListener('touchstart', handleTouchStart)
                document.removeEventListener('touchmove', handleTouchMove)
            }
        }

        return () => {
            document.removeEventListener('focusin', handleFocus)
        }
    }, [])

    const registerInput = (input: HTMLInputElement | null) => {
        if (input && input.type === 'text') {
            inputRefs.current.add(input)
        }
    }

    const unregisterInput = (input: HTMLInputElement | null) => {
        if (input) {
            inputRefs.current.delete(input)
        }
    }

    return { registerInput, unregisterInput }
}
