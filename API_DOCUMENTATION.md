# Passenger Princess API Documentation

## Overview
The Passenger Princess API is a Go-based web service that provides route planning with Tesla Supercharger and restaurant information. It uses Google Maps APIs to calculate routes, find nearby superchargers, and discover restaurants at those locations.

## Base URL
```
http://localhost:8080
```

## Endpoints

### 1. GET `/` - Frontend
Serves the main HTML interface for the application.

**Response**: HTML page with embedded Google Maps integration.

### 2. GET `/autocomplete` - Place Autocomplete
Provides Google Places autocomplete suggestions for origin/destination input.

#### Request Parameters
- `partial` (string, required): Partial place name to autocomplete

#### Example Request
```bash
GET /autocomplete?partial=New%20York
```

#### Example Response
```json
{
  "predictions": [
    {
      "description": "New York, NY, USA",
      "place_id": "ChIJOwg_06VPwokRYv534QaPC8g",
      "structured_formatting": {
        "main_text": "New York",
        "secondary_text": "NY, USA"
      }
    }
  ],
  "status": "OK"
}
```

### 3. GET `/route` - Route Planning with Superchargers & Restaurants
The main endpoint that calculates a route and finds nearby superchargers with restaurants.

#### Request Parameters
- `origin` (string, required): Starting location (address, city, or coordinates)
- `destination` (string, required): Ending location (address, city, or coordinates)

#### Example Request
```bash
GET /route?origin=New%20York%2C%20NY&destination=Boston%2C%20MA
```

#### Example Response
```json
{
  "route": {
    "total_distance": "215 mi",
    "total_duration": "3 hours 45 mins",
    "polyline": "encoded_polyline_string_here"
  },
  "superchargers": [
    {
      "name": "Tesla Supercharger - New Haven",
      "address": "100 State St, New Haven, CT 06511, USA",
      "distance_meters": 45000,
      "distance_from_route_meters": 1200,
      "arrival_time": "2:15PM",
      "lat": 41.3083,
      "lng": -72.9279,
      "closest_point_on_route": {
        "lat": 41.3100,
        "lng": -72.9250
      },
      "restaurants": [
        {
          "name": "Local Burger Joint",
          "address": "123 Main St, New Haven, CT",
          "rating": 4.2,
          "is_open_now": true,
          "lat": 41.3085,
          "lng": -72.9280,
          "cuisine_types": ["American", "Burgers"],
          "walking_distance_meters": 150
        }
      ]
    }
  ],
  "steps": [
    {
      "polyline": "encoded_step_polyline",
      "duration": 1800,
      "duration_in_traffic": 2100
    }
  ],
  "debug_info": {
    "api_calls": [
      {
        "api": "Directions (Main Route)",
        "url": "https://maps.googleapis.com/maps/api/directions/json?...",
        "request_body": null,
        "search_area": null
      },
      {
        "api": "Places Text Search API",
        "url": "https://places.googleapis.com/v1/places:searchText",
        "request_body": {
          "textQuery": "Tesla Supercharger",
          "includedType": "electric_vehicle_charging_station",
          "maxResultCount": 20,
          "locationBias": {
            "circle": {
              "center": {
                "latitude": 41.5,
                "longitude": -72.5
              },
              "radius": 30000
            }
          }
        },
        "search_area": {
          "center_lat": 41.5,
          "center_lng": -72.5,
          "radius_m": 30000
        }
      }
    ]
  }
}
```

## Data Structures

### RouteDetails
```json
{
  "total_distance": "215 mi",
  "total_duration": "3 hours 45 mins",
  "polyline": "encoded_polyline_string"
}
```

### SuperchargerInfo
```json
{
  "name": "Tesla Supercharger - Location",
  "address": "Full address string",
  "distance_meters": 45000,
  "distance_from_route_meters": 1200,
  "arrival_time": "2:15PM",
  "lat": 41.3083,
  "lng": -72.9279,
  "closest_point_on_route": {
    "lat": 41.3100,
    "lng": -72.9250
  },
  "restaurants": [...]
}
```

### RestaurantInfo
```json
{
  "name": "Restaurant Name",
  "address": "Restaurant address",
  "rating": 4.2,
  "is_open_now": true,
  "lat": 41.3085,
  "lng": -72.9280,
  "cuisine_types": ["American", "Burgers"],
  "walking_distance_meters": 150
}
```

### StepInfo
```json
{
  "polyline": "encoded_step_polyline",
  "duration": 1800,
  "duration_in_traffic": 2100
}
```

## Key Features

1. **Route Calculation**: Uses Google Directions API with real-time traffic data
2. **Supercharger Discovery**: Finds Tesla Superchargers within 10km of the route
3. **Segment-based Selection**: Selects 10 closest superchargers per 40km route segment
4. **Restaurant Search**: Finds restaurants within 500m walking distance of each selected supercharger
5. **ETA Calculation**: Estimates arrival times at superchargers considering traffic
6. **Debug Information**: Provides detailed API call logs for troubleshooting

## Error Responses

### Missing Parameters
```json
{
  "error": "Query parameters 'origin' and 'destination' are required"
}
```

### Route Not Found
```json
{
  "error": "Could not find a route"
}
```

### API Errors
```json
{
  "error": "Failed to get directions: [error details]"
}
```

## Usage Notes

- The API requires a valid Google Maps API key set as the `GOOGLE_MAPS_API_KEY` environment variable
- All coordinates use the WGS84 coordinate system
- Distances are provided in both meters and human-readable formats
- The service runs on port 8080 by default
- Restaurant searches are limited to 20 results per supercharger
- Superchargers are filtered to those within 10km of the route and selected based on proximity (10 closest per 40km segment)</content>
<parameter name="filePath">/home/brensch/passengerprincess/API_DOCUMENTATION.md
