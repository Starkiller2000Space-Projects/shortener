DROP INDEX IF EXISTS idx_is_deleted;

ALTER TABLE urls DROP COLUMN is_deleted;