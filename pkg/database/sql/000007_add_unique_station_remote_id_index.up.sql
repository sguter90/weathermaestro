-- Add unique constraint on station_id + remote_id combination
-- This ensures that each remote_id is unique per station
CREATE UNIQUE INDEX idx_sensors_station_remote_id ON sensors(station_id, remote_id);