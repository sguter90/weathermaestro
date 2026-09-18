CREATE TABLE IF NOT EXISTS sensor_readings (
    id         UUID DEFAULT generateUUIDv4(),
    sensor_id  UUID,
    value      Float64,
    date_utc   DateTime64(3, 'UTC'),
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date_utc)
ORDER BY (sensor_id, date_utc)
