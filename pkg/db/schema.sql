
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
    supercharger_id TEXT
);

-- api call log
CREATE TABLE IF NOT EXISTS maps_call_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sku TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    supercharger_id TEXT,
    place_id TEXT,
    error TEXT,
    details TEXT
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