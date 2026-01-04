-- Create stations table
CREATE TABLE IF NOT EXISTS stations (
    id SERIAL PRIMARY KEY,
    pass_key VARCHAR(255) NOT NULL UNIQUE,
    station_type VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on pass_key for faster lookups
CREATE INDEX idx_stations_pass_key ON stations(pass_key);

-- Add foreign key to weather_data
ALTER TABLE weather_data
    ADD COLUMN station_id INTEGER,
    ADD CONSTRAINT fk_weather_data_station
        FOREIGN KEY (station_id)
        REFERENCES stations(id)
        ON DELETE CASCADE;

-- Create index on station_id for faster joins
CREATE INDEX idx_weather_data_station_id ON weather_data(station_id);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to update updated_at on stations table
CREATE TRIGGER update_stations_updated_at
    BEFORE UPDATE ON stations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
