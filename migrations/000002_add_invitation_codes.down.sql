-- migrations/000002_add_invitation_codes.down.sql

-- Hapus Trigger DULU
DROP TRIGGER IF EXISTS set_timestamp_invitation_codes ON invitation_codes;

-- Hapus Index
DROP INDEX IF EXISTS idx_invitation_codes_code;
DROP INDEX IF EXISTS idx_invitation_codes_child_id;
DROP INDEX IF EXISTS idx_invitation_codes_status;
DROP INDEX IF EXISTS idx_invitation_codes_expires_at;

-- Hapus Tabel
DROP TABLE IF EXISTS invitation_codes;

-- Hapus Custom Type (ENUM)
DROP TYPE IF EXISTS invitation_status;