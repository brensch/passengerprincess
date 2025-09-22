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
            let keyboardVisible = false
            
            // Detect keyboard visibility
            const checkKeyboardVisibility = () => {
                const viewportHeight = window.visualViewport?.height || window.innerHeight
                const windowHeight = window.innerHeight
                keyboardVisible = viewportHeight < windowHeight * 0.9 // Keyboard is visible if viewport is significantly smaller
            }
            
            window.visualViewport?.addEventListener('resize', checkKeyboardVisibility)
            checkKeyboardVisibility()

            const handleTouchStart = (event: TouchEvent) => {
                const target = event.target as HTMLElement
                const dropdown = target.closest('.mobile-dropdown') as HTMLElement
                if (dropdown) {
                    // Prevent parent scrolling when touching dropdown
                    event.stopPropagation()
                    
                    // If keyboard is visible, be more aggressive with event handling
                    if (keyboardVisible) {
                        event.preventDefault()
                        ;(dropdown.style as any).webkitOverflowScrolling = 'auto'
                        ;(dropdown.style as any).touchAction = 'auto'
                    }
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
                    
                    // If keyboard is visible, allow all touch moves within dropdown
                    if (keyboardVisible) {
                        event.stopPropagation()
                        // Don't prevent default - let the browser handle scrolling
                        return
                    }
                    
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
                window.visualViewport?.removeEventListener('resize', checkKeyboardVisibility)
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
