DROP INDEX IF EXISTS idx_outbox_unpublished;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS idempotency_records;
DROP TRIGGER IF EXISTS borrowers_updated_at ON borrowers;
DROP TABLE IF EXISTS borrowers;
DROP FUNCTION IF EXISTS set_updated_at();
