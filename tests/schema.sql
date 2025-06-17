
-- SQLite schema for testing.
CREATE TABLE IF NOT EXISTS Test (
    id TEXT PRIMARY KEY NOT NULL,
    string TEXT,
    string_array TEXT,
    int INTEGER,
    int_array TEXT,
    int64 INTEGER,
    int64_array TEXT,
    float REAL,
    float_array TEXT,
    bool BOOLEAN,
    bool_array TEXT,
    bytes BLOB,
    bytes_array TEXT
);

CREATE TABLE IF NOT EXISTS Post (
    id TEXT PRIMARY KEY NOT NULL,
    title TEXT,
    content TEXT,
    tags TEXT,
    rating REAL,
    published INTEGER,
    "comments.count" INTEGER,
    "views.count" INTEGER,
    "likes.count" INTEGER,
    "shares.count" INTEGER,
    "favorites.count" INTEGER,
    "author.id" TEXT,
    "author.name" TEXT
);