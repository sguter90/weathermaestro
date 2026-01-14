-- Create sensors table (physical sensors on a station)
CREATE TABLE IF NOT EXISTS sensors (
   id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    station_id UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    sensor_type VARCHAR(20) NOT NULL,
    location VARCHAR(100) NOT NULL, -- 'indoor', 'outdoor', 'roof', etc.
    name VARCHAR(100), -- custom name, e.g., "Balcony Temp Sensor"
    model VARCHAR(100),
    battery_level INTEGER, -- 0-100 or NULL if wired
    signal_strength INTEGER, -- RSSI or similar
    enabled BOOLEAN DEFAULT TRUE,
    remote_id VARCHAR(255), -- pusher or puller unique identifier
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create sensor_readings table (actual measurements)
CREATE TABLE IF NOT EXISTS sensor_readings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sensor_id UUID NOT NULL REFERENCES sensors(id) ON DELETE CASCADE,
    value NUMERIC(10, 2) NOT NULL,
    date_utc TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_sensors_station_id ON sensors(station_id);
CREATE INDEX idx_sensors_location ON sensors(location);
CREATE INDEX idx_sensor_readings_sensor_id ON sensor_readings(sensor_id);
CREATE INDEX idx_sensor_readings_date_utc ON sensor_readings(date_utc DESC);
CREATE INDEX idx_sensor_readings_sensor_date ON sensor_readings(sensor_id, date_utc DESC);

-- Trigger for updated_at
CREATE TRIGGER update_sensors_updated_at
    BEFORE UPDATE ON sensors
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();