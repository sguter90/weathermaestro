
CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE stations (
    id SERIAL PRIMARY KEY,
    pass_key VARCHAR(255) UNIQUE NOT NULL,
    station_type VARCHAR(50) NOT NULL,
    model VARCHAR(100),
    freq VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE weather_data (
    time TIMESTAMPTZ NOT NULL,
    station_id INTEGER REFERENCES stations(id),

    -- Indoor
    temp_in_c DOUBLE PRECISION,
    temp_in_f DOUBLE PRECISION,
    humidity_in INTEGER,

    -- Outdoor
    temp_out_c DOUBLE PRECISION,
    temp_out_f DOUBLE PRECISION,
    humidity_out INTEGER,

    -- Pressure
    barom_rel_hpa DOUBLE PRECISION,
    barom_abs_hpa DOUBLE PRECISION,
    barom_rel_in DOUBLE PRECISION,
    barom_abs_in DOUBLE PRECISION,

    -- Wind
    wind_dir INTEGER,
    wind_speed_ms DOUBLE PRECISION,
    wind_gust_ms DOUBLE PRECISION,
    wind_speed_kmh DOUBLE PRECISION,
    wind_gust_kmh DOUBLE PRECISION,
    wind_speed_mph DOUBLE PRECISION,
    wind_gust_mph DOUBLE PRECISION,

    -- Rain
    rain_rate_mm_h DOUBLE PRECISION,
    daily_rain_mm DOUBLE PRECISION,

    -- Solar
    solar_radiation DOUBLE PRECISION,
    uv INTEGER,

    PRIMARY KEY (time, station_id)
);

-- Convert to hypertable for time-series optimization
SELECT create_hypertable('weather_data', 'time');

-- Create indexes
CREATE INDEX idx_weather_data_station_time ON weather_data (station_id, time DESC);
CREATE INDEX idx_stations_pass_key ON stations (pass_key);
