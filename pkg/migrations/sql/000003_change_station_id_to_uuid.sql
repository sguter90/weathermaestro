-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create a temporary column for the new UUID
ALTER TABLE stations ADD COLUMN uuid_id UUID DEFAULT uuid_generate_v4();

-- Update weather_data to use UUID
ALTER TABLE weather_data ADD COLUMN station_uuid UUID;

-- Copy the UUID references
UPDATE weather_data wd
SET station_uuid = s.uuid_id
FROM stations s
WHERE wd.station_id = s.id;

-- Drop old foreign key and column
ALTER TABLE weather_data DROP CONSTRAINT weather_data_station_id_fkey;
ALTER TABLE weather_data DROP COLUMN station_id;

-- Rename new column
ALTER TABLE weather_data RENAME COLUMN station_uuid TO station_id;

-- Drop old primary key and create new one
ALTER TABLE stations DROP CONSTRAINT stations_pkey;
ALTER TABLE stations DROP COLUMN id;
ALTER TABLE stations RENAME COLUMN uuid_id TO id;
ALTER TABLE stations ADD PRIMARY KEY (id);

-- Add foreign key constraint
ALTER TABLE weather_data
ADD CONSTRAINT weather_data_station_id_fkey
FOREIGN KEY (station_id) REFERENCES stations(id) ON DELETE CASCADE;

-- Add NOT NULL constraint
ALTER TABLE weather_data ALTER COLUMN station_id SET NOT NULL;

-- Create index on station_id in weather_data
CREATE INDEX IF NOT EXISTS idx_weather_data_station_id ON weather_data(station_id);
