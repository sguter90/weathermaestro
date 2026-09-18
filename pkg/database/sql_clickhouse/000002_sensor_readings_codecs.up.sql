ALTER TABLE sensor_readings MODIFY COLUMN date_utc   DateTime64(3, 'UTC') CODEC(DoubleDelta, ZSTD(3));
ALTER TABLE sensor_readings MODIFY COLUMN created_at DateTime             CODEC(DoubleDelta, ZSTD(3));
ALTER TABLE sensor_readings MODIFY COLUMN value      Float64              CODEC(Gorilla, ZSTD(3));
