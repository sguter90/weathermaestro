-- Add mode and service_name columns to stations table
ALTER TABLE stations
    ADD COLUMN mode VARCHAR(10) NOT NULL DEFAULT 'push',
    ADD COLUMN service_name VARCHAR(50) NOT NULL DEFAULT 'unknown';

-- Create index for faster lookups
CREATE INDEX idx_stations_mode_service ON stations(mode, service_name);

-- Add constraint to ensure valid modes
ALTER TABLE stations
    ADD CONSTRAINT check_valid_mode CHECK (mode IN ('push', 'pull'));