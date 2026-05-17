ALTER TABLE urls ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_is_deleted ON urls(is_deleted);