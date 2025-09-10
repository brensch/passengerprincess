
-- Enable foreign key constraints in SQLite
PRAGMA foreign_keys = ON;

-- Enable WAL mode for better concurrency (multiple readers, one writer)
PRAGMA journal_mode = WAL;

-- Set synchronous mode to NORMAL for balanced performance and safety
PRAGMA synchronous = NORMAL;

-- Increase cache size for better performance (1GB)
PRAGMA cache_size = 1000000;

-- Store temporary tables in memory for speed
PRAGMA temp_store = memory;

-- place_id is the unique identifier for a place from Google Places API
CREATE TABLE IF NOT EXISTS places (
    place_id TEXT PRIMARY KEY,
    name TEXT,
    address TEXT,
    latitude REAL,
    longitude REAL,
    rating REAL,
    user_ratings_total INTEGER,
    primary_type TEXT,
    primary_type_display TEXT,
    display_name TEXT,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- table to capture supercharger locations
CREATE TABLE IF NOT EXISTS superchargers (
    place_id TEXT PRIMARY KEY,
    name TEXT,
    address TEXT,
    latitude REAL,
    longitude REAL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- association table between places and superchargers
CREATE TABLE IF NOT EXISTS place_supercharger (
    place_id TEXT,
    supercharger_id TEXT,
    UNIQUE(place_id, supercharger_id),
    FOREIGN KEY (place_id) REFERENCES places(place_id) ON DELETE CASCADE,
    FOREIGN KEY (supercharger_id) REFERENCES superchargers(place_id) ON DELETE CASCADE
);

-- api call log
CREATE TABLE IF NOT EXISTS maps_call_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sku TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    supercharger_id TEXT,
    place_id TEXT,
    error TEXT,
    details TEXT,
    FOREIGN KEY (supercharger_id) REFERENCES superchargers(place_id) ON DELETE SET NULL,
    FOREIGN KEY (place_id) REFERENCES places(place_id) ON DELETE SET NULL
);

-- cache hits for place details
CREATE TABLE IF NOT EXISTS cache_hits (
    object_id TEXT PRIMARY KEY,
    hit BOOLEAN,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    type TEXT
);

-- route call log
CREATE TABLE IF NOT EXISTS route_call_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    origin TEXT,
    destination TEXT,
    error TEXT,
    ip_address TEXT
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_places_lat_lng ON places(latitude, longitude);
CREATE INDEX IF NOT EXISTS idx_superchargers_lat_lng ON superchargers(latitude, longitude);
CREATE INDEX IF NOT EXISTS idx_place_supercharger_place_id ON place_supercharger(place_id);
CREATE INDEX IF NOT EXISTS idx_place_supercharger_supercharger_id ON place_supercharger(supercharger_id);
CREATE INDEX IF NOT EXISTS idx_maps_call_log_supercharger_id ON maps_call_log(supercharger_id);
CREATE INDEX IF NOT EXISTS idx_maps_call_log_place_id ON maps_call_log(place_id);
CREATE INDEX IF NOT EXISTS idx_maps_call_log_timestamp ON maps_call_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_route_call_log_timestamp ON route_call_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_cache_hits_type ON cache_hits(type);