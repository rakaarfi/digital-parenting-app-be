package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rakaarfi/digital-parenting-app-be/internal/models"
	zlog "github.com/rs/zerolog/log"
)

type userRelationshipRepo struct {
	db *pgxpool.Pool
}

// NewUserRelationshipRepository membuat instance baru dari UserRelationshipRepository.
func NewUserRelationshipRepository(db *pgxpool.Pool) UserRelationshipRepository {
	return &userRelationshipRepo{db: db}
}

// AddRelationship menambahkan relasi baru antara parent dan child.
func (r *userRelationshipRepo) AddRelationship(ctx context.Context, parentID int, childID int) error {
	query := `INSERT INTO user_relationship (parent_id, child_id) VALUES ($1, $2)`
	_, err := r.db.Exec(ctx, query, parentID, childID)

	if err != nil {
		// Handle potential unique constraint violation (pasangan parent_id, child_id sudah ada)
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			// Periksa apakah constraint yang dilanggar adalah unique_parent_child
			if pgErr.ConstraintName == "unique_parent_child" {
				zlog.Warn().Int("parent_id", parentID).Int("child_id", childID).Msg("Relationship already exists")
				// Mengembalikan error spesifik atau nil tergantung kebutuhan (apakah duplikat dianggap error?)
				// Untuk konsistensi, kembalikan error yang menandakan konflik.
				return fmt.Errorf("relationship between parent %d and child %d already exists", parentID, childID)
			}
		}
		// Handle potential foreign key violation (parent_id atau child_id tidak ada di tabel users)
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23503" {
			zlog.Warn().Int("parent_id", parentID).Int("child_id", childID).Msg("Foreign key violation when adding relationship (user not found?)")
			// Tentukan user mana yang tidak ada (opsional, perlu cek constraint name jika berbeda)
			return fmt.Errorf("parent or child ID does not exist in users table")
		}

		// Error umum lainnya
		zlog.Error().Err(err).Int("parent_id", parentID).Int("child_id", childID).Msg("Error adding user relationship")
		return fmt.Errorf("error adding relationship: %w", err)
	}

	zlog.Info().Int("parent_id", parentID).Int("child_id", childID).Msg("User relationship added successfully")
	return nil
}

// GetChildrenByParentID mengambil daftar semua anak (data user lengkap) yang terhubung ke parentID tertentu.
func (r *userRelationshipRepo) GetChildrenByParentID(ctx context.Context, parentID int) ([]models.User, error) {
	// Query untuk mendapatkan data anak (dari tabel users) yang ID-nya ada di user_relationship dengan parentID yang sesuai.
	// Kita juga JOIN dengan roles untuk mendapatkan nama role anak.
	query := `SELECT
				u.id, u.username, u.email, u.first_name, u.last_name, u.role_id, u.created_at, u.updated_at,
				r.id as roleid, r.name as rolename
			 FROM users u
			 JOIN roles r ON u.role_id = r.id
			 JOIN user_relationship ur ON u.id = ur.child_id
			 WHERE ur.parent_id = $1 AND r.name = 'Child' -- Pastikan hanya mengambil user dengan role 'Child'
			 ORDER BY u.username ASC` // Urutkan berdasarkan username anak

	rows, err := r.db.Query(ctx, query, parentID)
	if err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Msg("Error querying children by parent ID")
		return nil, fmt.Errorf("error getting children for parent %d: %w", parentID, err)
	}
	defer rows.Close()

	children := []models.User{}
	for rows.Next() {
		var child models.User
		child.Role = &models.Role{} // Inisialisasi pointer Role
		scanErr := rows.Scan(
			&child.ID, &child.Username, &child.Email, &child.FirstName, &child.LastName,
			&child.RoleID, &child.CreatedAt, &child.UpdatedAt,
			&child.Role.ID, &child.Role.Name,
		)
		if scanErr != nil {
			zlog.Warn().Err(scanErr).Int("parent_id", parentID).Msg("Error scanning child row")
			// Pertimbangkan untuk menghentikan proses jika ada error scan
			return children, fmt.Errorf("error scanning child data: %w", scanErr)
		}
		children = append(children, child)
	}

	if err = rows.Err(); err != nil {
		zlog.Error().Err(err).Int("parent_id", parentID).Msg("Error iterating children rows")
		return children, fmt.Errorf("error iterating children data: %w", err)
	}

	zlog.Debug().Int("parent_id", parentID).Int("children_count", len(children)).Msg("Retrieved children by parent ID")
	return children, nil
}

// GetParentsByChildID mengambil daftar semua parent (data user lengkap) yang terhubung ke childID tertentu.
func (r *userRelationshipRepo) GetParentsByChildID(ctx context.Context, childID int) ([]models.User, error) {
	// Query untuk mendapatkan data parent (dari tabel users) yang ID-nya ada di user_relationship dengan childID yang sesuai.
	// Kita juga JOIN dengan roles untuk mendapatkan nama role parent.
	query := `SELECT
				u.id, u.username, u.email, u.first_name, u.last_name, u.role_id, u.created_at, u.updated_at,
				r.id as roleid, r.name as rolename
			FROM users u
			JOIN roles r ON u.role_id = r.id
			JOIN user_relationship ur ON u.id = ur.parent_id
			WHERE ur.child_id = $1 AND r.name = 'Parent' -- Pastikan hanya mengambil user dengan role 'Parent'
			ORDER BY u.username ASC` // Urutkan berdasarkan username parent

	rows, err := r.db.Query(ctx, query, childID)
	if err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Msg("Error querying parents by child ID")
		return nil, fmt.Errorf("error getting parents for child %d: %w", childID, err)
	}
	defer rows.Close()

	parents := []models.User{}
	for rows.Next() {
		var parent models.User
		parent.Role = &models.Role{} // Inisialisasi pointer Role
		scanErr := rows.Scan(
			&parent.ID, &parent.Username, &parent.Email, &parent.FirstName, &parent.LastName,
			&parent.RoleID, &parent.CreatedAt, &parent.UpdatedAt,
			&parent.Role.ID, &parent.Role.Name,
		)
		if scanErr != nil {
			zlog.Warn().Err(scanErr).Int("child_id", childID).Msg("Error scanning parent row")
			return parents, fmt.Errorf("error scanning parent data: %w", scanErr)
		}
		parents = append(parents, parent)
	}

	if err = rows.Err(); err != nil {
		zlog.Error().Err(err).Int("child_id", childID).Msg("Error iterating parents rows")
		return parents, fmt.Errorf("error iterating parents data: %w", err)
	}

	zlog.Debug().Int("child_id", childID).Int("parents_count", len(parents)).Msg("Retrieved parents by child ID")
	return parents, nil
}

// IsParentOf memeriksa apakah relasi spesifik antara parentID dan childID ada di database.
func (r *userRelationshipRepo) IsParentOf(ctx context.Context, parentID int, childID int) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM user_relationship WHERE parent_id = $1 AND child_id = $2)`
	var exists bool
	err := r.db.QueryRow(ctx, query, parentID, childID).Scan(&exists)
	if err != nil {
		// Error saat query, bukan karena tidak ada
		zlog.Error().Err(err).Int("parent_id", parentID).Int("child_id", childID).Msg("Error checking parent-child relationship existence")
		return false, fmt.Errorf("error checking relationship: %w", err)
	}
	return exists, nil
}

// RemoveRelationship menghapus relasi spesifik antara parentID dan childID.
func (r *userRelationshipRepo) RemoveRelationship(ctx context.Context, parentID int, childID int) error {
	query := `DELETE FROM user_relationship WHERE parent_id = $1 AND child_id = $2`
	tag, err := r.db.Exec(ctx, query, parentID, childID)

	if err != nil {
		// Error umum saat eksekusi query
		zlog.Error().Err(err).Int("parent_id", parentID).Int("child_id", childID).Msg("Error removing user relationship")
		return fmt.Errorf("error removing relationship: %w", err)
	}

	// Cek apakah ada baris yang benar-benar terhapus
	if tag.RowsAffected() == 0 {
		zlog.Warn().Int("parent_id", parentID).Int("child_id", childID).Msg("Attempted to remove non-existent relationship")
		// Mengembalikan ErrNoRows agar handler bisa mendeteksi bahwa relasi tidak ditemukan
		return pgx.ErrNoRows
	}

	zlog.Info().Int("parent_id", parentID).Int("child_id", childID).Msg("User relationship removed successfully")
	return nil
}

// --- Metode Tx (Tambahan) ---

// IsParentOfTx memeriksa relasi parent-child dalam konteks transaksi database.
func (r *userRelationshipRepo) IsParentOfTx(ctx context.Context, tx pgx.Tx, parentID int, childID int) (bool, error) {
	// Query sama dengan IsParentOf, tapi menggunakan tx.QueryRow
	query := `SELECT EXISTS (SELECT 1 FROM user_relationship WHERE parent_id = $1 AND child_id = $2)`
	var exists bool
	// Gunakan tx.QueryRow bukan r.db.QueryRow
	err := tx.QueryRow(ctx, query, parentID, childID).Scan(&exists)
	if err != nil {
		// Log error dengan konteks Tx
		zlog.Error().Err(err).Int("parent_id", parentID).Int("child_id", childID).Msg("RepoTx: Error checking parent-child relationship existence")
		// Kembalikan error untuk di-handle oleh service (yang akan rollback)
		return false, fmt.Errorf("repoTx error checking relationship: %w", err)
	}
	return exists, nil
}

func (r *userRelationshipRepo) AddRelationshipTx(ctx context.Context, tx pgx.Tx, parentID int, childID int) error {
	query := `INSERT INTO user_relationship (parent_id, child_id) VALUES ($1, $2)`
	_, err := tx.Exec(ctx, query, parentID, childID) // Gunakan tx.Exec

	if err != nil {
		// Handle error spesifik (23505 - unique, 23503 - FK) seperti di AddRelationship non-Tx
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "unique_parent_child" {
				zlog.Warn().Int("parent_id", parentID).Int("child_id", childID).Msg("RepoTx: Relationship already exists")
				return fmt.Errorf("relationship between parent %d and child %d already exists", parentID, childID)
			}
			if pgErr.Code == "23503" {
				zlog.Warn().Int("parent_id", parentID).Int("child_id", childID).Msg("RepoTx: Foreign key violation when adding relationship")
				return fmt.Errorf("parent or child ID does not exist in users table")
			}
		}
		// Error umum
		zlog.Error().Err(err).Int("parent_id", parentID).Int("child_id", childID).Msg("RepoTx: Error adding user relationship")
		return fmt.Errorf("repoTx error adding relationship: %w", err)
	}
	return nil
}

// Anda bisa menambahkan metode Tx lain jika diperlukan, misalnya AddRelationshipTx atau RemoveRelationshipTx
// jika operasi tersebut perlu menjadi bagian dari transaksi yang lebih besar di service layer.
// func (r *userRelationshipRepo) AddRelationshipTx(ctx context.Context, tx pgx.Tx, parentID int, childID int) error { ... }
// func (r *userRelationshipRepo) RemoveRelationshipTx(ctx context.Context, tx pgx.Tx, parentID int, childID int) error { ... }
