interface HelpModalProps {
    isOpen: boolean
    onClose: () => void
}

const HelpModal = ({ isOpen, onClose }: HelpModalProps) => {
    if (!isOpen) return null

    return (
        <div
            className="fixed inset-0 flex items-center justify-center z-[1200] bg-black bg-opacity-50 backdrop-blur-sm"
            onClick={(e) => e.target === e.currentTarget && onClose()}
        >
            <div className="rounded-xl p-8 max-w-md mx-4 shadow-2xl bg-princess-surface border-2 border-princess-border">
                <h3 className="text-2xl font-semibold mb-4 font-dancing text-princess-text-primary">
                    ✨ PPP's Purported Purpose ✨
                </h3>
                <div className="text-sm space-y-3 text-princess-text-secondary">
                    <p>⚡ Get Superchargers on your route</p>
                    <p>🗺️ ETAs and traffic according to google maps</p>
                    <p>🌭 See restaurants within walking distance</p>
                    <p>🔎 Filter restaurants by cuisine type or name</p>
                    <p>👸 Pick your pitstop perfectly</p>
                </div>
                <div className="flex justify-end mt-6">
                    <button
                        onClick={onClose}
                        className="px-6 py-2 font-medium rounded-lg transition-all duration-300
                     text-princess-text-secondary bg-princess-surface-soft border border-princess-border
                     hover:bg-princess-accent-lavender hover:text-princess-text-primary"
                    >
                        Got it!
                    </button>
                </div>
            </div>
        </div>
    )
}

export default HelpModal
