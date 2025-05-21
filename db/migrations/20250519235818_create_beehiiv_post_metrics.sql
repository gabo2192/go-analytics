-- +goose Up
CREATE TABLE beehiiv_post_metrics (
  post_id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  slug TEXT,
  publish_date TIMESTAMP,

  email_recipients INTEGER,
  email_delivered INTEGER,
  email_opens INTEGER,
  email_unique_opens INTEGER,
  email_clicks INTEGER,
  email_unique_clicks INTEGER,
  email_open_rate FLOAT,
  email_click_rate FLOAT,

  web_views INTEGER,
  web_clicks INTEGER,
  web_click_rate FLOAT,

  total_engagements INTEGER,

  created_at TIMESTAMP DEFAULT now(),
  inserted_at TIMESTAMP DEFAULT now(),
  updated_at TIMESTAMP DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS beehiiv_post_metrics;
