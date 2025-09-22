import { useEffect, useRef } from 'react'

export const useMobileInputFocus = () => {
    const inputRefs = useRef<Set<HTMLInputElement>>(new Set())

    useEffect(() => {
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
