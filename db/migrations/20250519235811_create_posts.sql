-- +goose Up
CREATE TABLE posts (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  author_id TEXT REFERENCES authors(id) ON DELETE SET NULL,
  main_category TEXT,
  sub_category TEXT,
  published_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT now(),
  updated_at TIMESTAMP DEFAULT now(),
  yesterday_views INTEGER DEFAULT 0,
  last_seven_days_views INTEGER DEFAULT 0,
  last_14_days_views INTEGER DEFAULT 0,
  last_30_days_views INTEGER DEFAULT 0,
  last_90_days_views INTEGER DEFAULT 0,
  last_180_days_views INTEGER DEFAULT 0,
  last_365_days_views INTEGER DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS posts;
