# WeatherMaestro

A comprehensive weather data aggregation and management platform built in Go, designed to collect, and process weather data from multiple sources.

## Overview

WeatherMaestro is a self-hosted alternative to [Weathercloud](https://weathercloud.net/en) or [Weather Underground](https://www.wunderground.com/).

It is a modular weather data platform that:
- Pulls weather data from various sources (Netatmo, etc.)
- Stores and manages weather station data
- Supports weather push from weather stations (Ecowitt, etc.)
- Provides a RESTful API for data access
- Supports multiple weather stations and sensors

**Currently supported:**
* Ecowitt (via push endpoint)
* Netatmo (via Netatmo-API pull)

This project is the backend part.  
For the frontend see [sguter90/weathermaestro-ui](https://github.com/sguter90/weathermaestro-ui).

## Architecture

The project follows a clean, layered architecture:
```
weathermaestro/
├── cmd/cli/              # Main application entry point
│   ├── main.go          # Application bootstrap
│   ├── cmd_*.go         # CLI commands
│   ├── handler_*.go     # HTTP handlers
│   └── registry.go      # Service registry
├── pkg/                 # Reusable packages
│   ├── database/        # Database layer
│   ├── models/          # Domain models
│   ├── puller/          # Data pulling services
│   └── pusher/          # Data pushing services
└── deployments/         # Deployment configurations
    └── docker/          # Docker setup
```

## Features

### Data Sources (Pullers)
- **Netatmo Integration**: Pull weather data from Netatmo weather stations
- Extensible architecture for adding new data sources

### Data Destinations (Pushers)
- **Ecowitt Integration**: Push weather data to Ecowitt services
- Support for multiple sensor types and measurements

### API Endpoints
- Health monitoring
- Station management
- Sensor data access
- Weather readings retrieval
- Pusher endpoint management

### User Management
- User authentication and authorization
- CLI-based user creation

## Prerequisites

- Go 1.25 or higher
- PostgreSQL database
- Docker (optional, for containerized deployment)

## Installation

### Using docker-compose (recommended)
1. Copy ``deployments/docker/compose.prod.yml`` (as ``compose.yml``) and ``deployments/docker/.env.example`` (as ``.env``) to the installation directory.
2. Adjust configuration in ``.env``
3. Start via ``docker compose up -d``

This will start the server (API), database (postgres) and the ui (frontend).
For details about the UI see the related [weathermaestro-ui](https://github.com/sguter90/weathermaestro-ui) project.

### From Source

```bash
git clone https://github.com/sguter90/weathermaestro.git
cd weathermaestro
go mod download
cd cmd/cli
CGO_ENABLED=0 go build -ldflags="-w -s" -o ./weathermaestro
```

## Configuration
```bash
# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=weather_user
DB_PASSWORD=change_me_in_production
DB_NAME=weather_db
DB_SSLMODE=disable

# Server Configuration
SERVER_PORT=8059 # port of the API
SERVER_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000 # allowed origin = UI/Frontend URL
SERVER_PUBLIC_URL=http://localhost:8059 # public URL of the API server
JWT_SECRET=change_me_in_production # random string - e.g. via: openssl rand -base64 45

# UI Configuration
UI_APP_NAME=WeatherMaestro # application name shown in UI
UI_APP_DESCRIPTION="Weather Service" # application description shown in UI header
UI_PORT=8058 # port of the UI

# Timezone
TZ=Europe/Berlin
```

## Usage
When using docker-compose then the command needs to be executed within the container:
``docker compose exec server weathermaestro <command>``

### Adding a station
```bash
./weathermaestro station add
```
You then will be guided through the setup.  
When using pusher like ecowitt you will need a passkey which can be found in the Configuration-Interface of the weather station.

### Creating a user
When authenticated with a user you can do some extra stuff like adding dashboards.  
To create a user:
```bash
./weathermaestro user create
```

## API Usage
The API does not need an authenticated user.
Data like weather station readings or dashboards are public and can be fetched by default. (GET requests)

### Auth
For accessing protected routes you will need a JWT token.  
```
# Login
POST /api/v1/auth/login

# Logout
POST /api/v1/auth/logout

# Profile
GET /api/v1/auth/me

# Refresh JWT token
POST /api/v1/auth/refresh 
```

**UserInfo-Model**:
```json
{
    "id": "ada81a02-3716-4656-93d4-92e366dbb905",
    "username": "weather"
}
```

**Login-Request**:
```json
{
    "username": "<your_username>",
    "password": "<your_password>"
}
```
**Login-Response**:
```json
{
    "success": true, 
    "token": "eyJ1c2VyX2lkIjoiYWRhODFhMDItMzcxNi00NjU2LTk", 
    "expires_at": "2026-02-10T15:54:04.872094932Z", 
    "user": "<UserInfo-Model>"
}
```

**Profile-Response**: UserInfo-Model  
**Refresh-Response**: Login-Response

### Health check
```
GET /api/v1/health
```

### Stations
```
# List all stations
GET /api/v1/stations

# Get station details
GET /api/v1/stations/{id}

# Create new station
POST /api/v1/stations

# Update station
PUT /api/v1/stations/{id}

# Delete station
DELETE /api/v1/stations/{id}
```

Station-Model:
```json
[
	{
		"id": "68f5e855-b9fe-49c4-a6bf-7c05beac4ba6",
		"pass_key": "abcdefg",
		"station_type": "EasyWeatherPro_V5.2.2",
		"model": "WS2900_V2.02.06",
		"total_readings": 580209,
		"first_reading": "2026-02-04T17:16:12Z",
		"last_reading": "2026-02-09T15:54:00Z"
	}
]
```

### Sensors
```
# List sensors for a station
GET /api/v1/stations/{stationId}/sensors

# Get sensor details
GET /api/v1/sensors/{id}
```

Sensor-Model:
```json
[
	{
		"sensor": {
			"id": "edb615c2-2d45-40b6-901c-ec453b7dfd4a",
			"station_id": "68f5e855-b9fe-49c4-a6bf-7c05beac4ba6",
			"sensor_type": "Humidity",
			"location": "Indoor",
			"name": "Humidity",
			"enabled": true,
			"created_at": "2026-02-04T17:16:13.529393Z",
			"updated_at": "2026-02-09T16:02:43.17983Z"
		}
	}
]
```

### Readings
```
GET /api/v1/readings
```

Query params:
- **station_id**: filter by station UUID
- **sensor_id**: filter by sensor UUID (can be comma-separated list)
- **sensor_type**: filter by sensor type
- **location**: filter by sensor location
- **start**: start time (RFC3339 or Unix timestamp)
- **end**: end time (RFC3339 or Unix timestamp)
- **limit**: max number of results (default: 100, max: 10000)
- **offset**: pagination offset
- **order**: sort order (asc/desc, default: desc)
- **aggregate**: aggregation interval (1m, 5m, 15m, 1h, 6h, 1d, 1w, 1M)
- **aggregate_func**: aggregation function (avg, min, max, sum, count, first, last)
- **group_by**: group results by (sensor, sensor_type, location)

Response-Model (without aggregate):
```json
{
    "total": 580902,
    "page": 1,
    "total_pages": 5810,
    "limit": 100,
    "has_more": true,
    "is_aggregated": false,
    "data": [
        {
            "id": "6baf3031-8402-4d37-b323-87f6a5ebca9c",
            "sensor_id": "e507f902-27a5-4c83-9d9c-08a17e5855d9",
            "value": 2,
            "date_utc": "2026-02-09T16:02:42Z"
        }
    ]
}
```

Response-Model (with aggregate):
```json
{
  "total": 580902,
  "page": 1,
  "total_pages": 5810,
  "limit": 100,
  "has_more": true,
  "is_aggregated": true,
  "data": [
    {
      "dateutc": "2026-02-09T16:02:42Z",
      "sensor_id": "e507f902-27a5-4c83-9d9c-08a17e5855d9",
      "sensor_type": "Temperature",
      "location": "Indoor",
      "value": 2,
      "count": 10,
      "min_value": 1,
      "max_value": 2
    }
  ]
}
```

### Dashboards
```
# List all dashboards
GET /api/v1/dashboards

# Get dashboard details
GET /api/v1/dashboards/{id}

# Create dashboard (protected)
POST /api/v1/dashboards

# Update dashboard (protected)
PUT /api/v1/dashboards/{id}

# Delete dashboard (protected)
DELETE /api/v1/dashboards/{id}
```

Dashboard-Model:
```json
[
  {
    "id": "22c6d33f-d0ee-440c-a2b0-faae2bfe0bac",
    "name": "Temperature",
    "description": "",
    "config": {
      // no specific format - can be customized as needed
    },
    "is_default": false,
    "created_at": "2026-02-08T17:42:58.736567Z",
    "updated_at": "2026-02-08T17:46:48.578605Z"
  }
]
```

### Pusher endpoints
```
# Ecowitt
POST /api/v1/data/report
```

## Development
### Project Structure
* **cmd/cli**: Command-line interface and HTTP handlers
* **pkg/database**: Database management and migrations
* **pkg/models**: Data models and domain entities
* **pkg/puller**: Data pulling services and clients
* **pkg/pusher**: Data pushing services and publishers

## Contributing
Contributions are welcome! Please follow these steps:
1. Fork the repository
2. Create a feature branch (git checkout -b feature/amazing-feature)
3. Commit your changes (git commit -m 'Add amazing feature')
4. Push to the branch (git push origin feature/amazing-feature)
5. Open a Pull Request

## License
This project is licensed under the MIT License - see the LICENSE file for details.

## Support
For issues, questions, or contributions, please open an issue on GitHub.

## Acknowledgments
* Built with [Cobra](https://github.com/spf13/cobra) for CLI
* Uses [Gorilla Mux](https://github.com/gorilla/mux) for HTTP routing