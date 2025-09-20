import { ViewMode } from '../types'

interface TopToolbarProps {
    onNewSearch: () => void
    onToggleView: () => void
    onOpenFilter: () => void
    onRefresh: () => void
    viewMode: ViewMode
    filterCount: number
}

const TopToolbar = ({
    onNewSearch,
    onToggleView,
    onOpenFilter,
    onRefresh,
    viewMode,
    filterCount
}: TopToolbarProps) => {
    return (
        <div className="w-full h-12 flex justify-between items-center px-4 
                    bg-gradient-to-r from-princess-surface via-princess-lavender to-princess-accent-lavender 
                    border-0 flex-shrink-0">
            {/* Left side - Logo */}
            <button
                onClick={onNewSearch}
                className="flex items-center space-x-2"
            >
                <span className="text-2xl font-dancing font-bold text-princess-text-primary">
                    PPP
                </span>
            </button>

            {/* Right side - Action buttons */}
            <div className="flex items-center space-x-3">
                {/* Refresh button */}
                <button
                    onClick={onRefresh}
                    className="px-2 py-1 text-sm rounded-lg bg-princess-accent-lavender
                   text-princess-text-primary flex items-center space-x-1"
                >
                    <span>🔄</span>
                    <span>Refresh</span>
                </button>

                {/* Filter button */}
                <button
                    onClick={onOpenFilter}
                    className="relative px-2 py-1 text-sm rounded-lg bg-princess-accent-mint
                   text-princess-text-primary flex items-center space-x-1"
                >
                    <span>🔍</span>
                    <span>Filter</span>
                    {filterCount > 0 && (
                        <span className="absolute -bottom-2 -right-2 bg-princess-rose text-princess-text-primary 
                           text-xs rounded-full w-5 h-5 flex items-center justify-center font-bold">
                            {filterCount}
                        </span>
                    )}
                </button>

                {/* Toggle view button */}
                <button
                    onClick={onToggleView}
                    className="px-2 py-1 text-sm rounded-lg bg-princess-accent-peach
                   text-princess-text-primary flex items-center space-x-1"
                >
                    <span>{viewMode === 'results' ? '🗺️' : '📖'}</span>
                    <span>
                        {viewMode === 'results' ? 'Map' : 'Results'}
                    </span>
                </button>
            </div>
        </div>
    )
}

export default TopToolbar
