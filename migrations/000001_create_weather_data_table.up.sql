CREATE TABLE IF NOT EXISTS weather_data (
    id SERIAL PRIMARY KEY,
    pass_key VARCHAR(255) NOT NULL,
    station_type VARCHAR(50),
    model VARCHAR(100),
    freq VARCHAR(20),
    date_utc TIMESTAMPTZ NOT NULL,
    interval INTEGER,
    runtime INTEGER,
    heap INTEGER,

    -- Indoor measurements
    temp_in_c DOUBLE PRECISION,
    temp_in_f DOUBLE PRECISION,
    humidity_in INTEGER,

    -- Outdoor measurements
    temp_out_c DOUBLE PRECISION,
    temp_out_f DOUBLE PRECISION,
    humidity_out INTEGER,

    -- Barometric pressure
    barom_rel_hpa DOUBLE PRECISION,
    barom_abs_hpa DOUBLE PRECISION,
    barom_rel_in DOUBLE PRECISION,
    barom_abs_in DOUBLE PRECISION,

    -- Wind measurements
    wind_dir INTEGER,
    wind_speed_ms DOUBLE PRECISION,
    wind_gust_ms DOUBLE PRECISION,
    max_daily_gust_ms DOUBLE PRECISION,
    wind_speed_kmh DOUBLE PRECISION,
    wind_gust_kmh DOUBLE PRECISION,
    max_daily_gust_kmh DOUBLE PRECISION,
    wind_speed_mph DOUBLE PRECISION,
    wind_gust_mph DOUBLE PRECISION,
    max_daily_gust_mph DOUBLE PRECISION,

    -- Solar and UV
    solar_radiation DOUBLE PRECISION,
    uv INTEGER,

    -- Rain measurements (metric)
    rain_rate_mm_h DOUBLE PRECISION,
    event_rain_mm DOUBLE PRECISION,
    hourly_rain_mm DOUBLE PRECISION,
    daily_rain_mm DOUBLE PRECISION,
    weekly_rain_mm DOUBLE PRECISION,
    monthly_rain_mm DOUBLE PRECISION,
    yearly_rain_mm DOUBLE PRECISION,
    total_rain_mm DOUBLE PRECISION,

    -- Rain measurements (imperial)
    rain_rate_in DOUBLE PRECISION,
    event_rain_in DOUBLE PRECISION,
    hourly_rain_in DOUBLE PRECISION,
    daily_rain_in DOUBLE PRECISION,
    weekly_rain_in DOUBLE PRECISION,
    monthly_rain_in DOUBLE PRECISION,
    yearly_rain_in DOUBLE PRECISION,
    total_rain_in DOUBLE PRECISION,

    -- Additional measurements
    vpd DOUBLE PRECISION,
    wh65_batt INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_weather_data_date_utc ON weather_data(date_utc DESC);
CREATE INDEX IF NOT EXISTS idx_weather_data_pass_key ON weather_data(pass_key);
CREATE INDEX IF NOT EXISTS idx_weather_data_station_type ON weather_data(station_type);
