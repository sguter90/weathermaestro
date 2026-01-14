-- Add config column to stations table
ALTER TABLE stations
    ADD COLUMN config JSONB DEFAULT '{}';

-- Create index on config for faster queries
CREATE INDEX idx_stations_config ON stations USING GIN(config);