-- migrations/000002_add_invitation_codes.up.sql

-- Buat tipe ENUM untuk status undangan
CREATE TYPE invitation_status AS ENUM ('active', 'used', 'expired');

-- Buat tabel untuk menyimpan kode undangan
CREATE TABLE invitation_codes (
    id SERIAL PRIMARY KEY,                                  -- ID unik untuk setiap kode
    code VARCHAR(16) NOT NULL UNIQUE,                       -- Kode undangan unik (panjang bisa disesuaikan)
    child_id INT NOT NULL,                                  -- ID anak yang undangannya terkait
    created_by_parent_id INT NOT NULL,                      -- ID parent yang membuat undangan
    status invitation_status NOT NULL DEFAULT 'active',      -- Status kode (default aktif)
    expires_at TIMESTAMPTZ NOT NULL,                        -- Waktu kedaluwarsa kode
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Foreign Key Constraints
    CONSTRAINT fk_invitation_child
        FOREIGN KEY(child_id)
        REFERENCES users(id)
        ON DELETE CASCADE, -- Jika user anak dihapus, kode undangannya tidak valid lagi

    CONSTRAINT fk_invitation_creator_parent
        FOREIGN KEY(created_by_parent_id)
        REFERENCES users(id)
        ON DELETE CASCADE -- Jika parent pembuat dihapus, kode undangannya juga hilang
);

-- Tambahkan index pada kolom yang sering dicari
CREATE INDEX idx_invitation_codes_code ON invitation_codes (code);
CREATE INDEX idx_invitation_codes_child_id ON invitation_codes (child_id);
CREATE INDEX idx_invitation_codes_status ON invitation_codes (status);
CREATE INDEX idx_invitation_codes_expires_at ON invitation_codes (expires_at);

-- Tambahkan trigger untuk updated_at (menggunakan fungsi yang sudah ada)
CREATE TRIGGER set_timestamp_invitation_codes
BEFORE UPDATE ON invitation_codes
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();