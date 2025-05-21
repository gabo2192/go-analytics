-- +goose Up
CREATE TABLE author_views (
  id SERIAL PRIMARY KEY,
  author_id TEXT NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
  views BIGINT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE (author_id, created_at)
);

-- +goose Down
DROP TABLE IF EXISTS author_views;
