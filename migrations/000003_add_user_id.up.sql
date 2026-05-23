ALTER TABLE urls ADD COLUMN user_id VARCHAR(64) NOT NULL DEFAULT '';

CREATE INDEX idx_user_id ON urls (user_id);