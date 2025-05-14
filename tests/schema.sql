
-- SQLite schema for testing.
CREATE TABLE Test (
    id TEXT PRIMARY KEY NOT NULL,
    string TEXT,
    string_array TEXT,
    int INTEGER,
    int_array INTEGER,
    int64 INTEGER,
    int64_array INTEGER,
    float REAL,
    float_array REAL,
    bool BOOLEAN,
    bool_array BOOLEAN,
    bytes BLOB,
    bytes_array BLOB
);