package models

import (
	"time"
)

// File ini mendefinisikan struktur data (models) yang digunakan di seluruh aplikasi,
// termasuk entitas database, Data Transfer Objects (DTO) untuk input API,
// dan tipe data enumerasi (enum).

// ====================================================================================
// Core Domain Entities (Representasi Tabel Database)
// ====================================================================================

// Role merepresentasikan peran pengguna dalam sistem (misal: 'parent', 'child', 'admin').
type Role struct {
	ID        int       `json:"id"`                                    // ID unik peran
	Name      string    `json:"name" validate:"required,min=3,max=50"` // Nama peran (harus unik)
	CreatedAt time.Time `json:"created_at,omitzero"`                   // Waktu pembuatan record
	UpdatedAt time.Time `json:"updated_at,omitzero"`                   // Waktu terakhir pembaruan record
}

// User merepresentasikan data pengguna dalam sistem.
type User struct {
	ID        int       `json:"id"`                                         // ID unik pengguna
	Username  string    `json:"username" validate:"required,min=3,max=100"` // Username unik untuk login
	Password  string    `json:"-"`                                          // Hash kata sandi (tidak dikirim dalam JSON response)
	Email     string    `json:"email" validate:"required,email"`            // Alamat email unik pengguna
	FirstName string    `json:"first_name,omitempty"`                       // Nama depan (opsional)
	LastName  string    `json:"last_name,omitempty"`                        // Nama belakang (opsional)
	RoleID    int       `json:"role_id" validate:"required,gt=0"`           // Foreign key ke tabel Role
	Role      *Role     `json:"role,omitempty"`                             // Relasi ke Role (bisa di-preload)
	CreatedAt time.Time `json:"created_at,omitzero"`                        // Waktu pembuatan record
	UpdatedAt time.Time `json:"updated_at,omitzero"`                        // Waktu terakhir pembaruan record
}

// UserRelationship merepresentasikan hubungan antara orang tua (Parent) dan anak (Child).
type UserRelationship struct {
	ID        int       `json:"id"`                                 // ID unik relasi
	ParentID  int       `json:"parent_id" validate:"required,gt=0"` // Foreign key ke User (Parent)
	ChildID   int       `json:"child_id" validate:"required,gt=0"`  // Foreign key ke User (Child)
	Child     *User     `json:"child,omitempty"`                    // Relasi ke User (Child) (bisa di-preload)
	CreatedAt time.Time `json:"created_at,omitzero"`                // Waktu pembuatan record
	UpdatedAt time.Time `json:"updated_at,omitzero"`                // Waktu terakhir pembaruan record
}

// Task merepresentasikan definisi sebuah tugas yang dibuat oleh orang tua.
type Task struct {
	ID              int       `json:"id"`                                          // ID unik tugas
	TaskName        string    `json:"task_name" validate:"required,min=3,max=100"` // Nama tugas
	TaskPoint       int       `json:"task_point" validate:"required,gt=0"`         // Jumlah poin yang didapat jika tugas selesai
	TaskDescription string    `json:"task_description,omitempty"`                  // Deskripsi detail tugas (opsional)
	CreatedByUserID int       `json:"created_by_user_id" validate:"required,gt=0"` // Foreign key ke User (Parent yang membuat)
	User            *User     `json:"user,omitempty"`                              // Relasi ke User (Pembuat) (bisa di-preload)
	CreatedAt       time.Time `json:"created_at,omitzero"`                         // Waktu pembuatan record
	UpdatedAt       time.Time `json:"updated_at,omitzero"`                         // Waktu terakhir pembaruan record
}

// Reward merepresentasikan definisi sebuah hadiah yang dapat diklaim oleh anak.
type Reward struct {
	ID                int       `json:"id"`                                            // ID unik hadiah
	RewardName        string    `json:"reward_name" validate:"required,min=3,max=100"` // Nama hadiah
	RewardPoint       int       `json:"reward_point" validate:"required,gt=0"`         // Jumlah poin yang dibutuhkan untuk klaim
	RewardDescription string    `json:"reward_description,omitempty"`                  // Deskripsi detail hadiah (opsional)
	CreatedByUserID   int       `json:"created_by_user_id" validate:"required,gt=0"`   // Foreign key ke User (Parent yang membuat)
	User              *User     `json:"user,omitempty"`                                // Relasi ke User (Pembuat) (bisa di-preload)
	CreatedAt         time.Time `json:"created_at,omitzero"`                           // Waktu pembuatan record
	UpdatedAt         time.Time `json:"updated_at,omitzero"`                           // Waktu terakhir pembaruan record
}

// UserTask merepresentasikan tugas yang telah ditugaskan (assigned) kepada seorang anak.
type UserTask struct {
	ID               int            `json:"id"`                                                                    // ID unik penugasan
	UserID           int            `json:"user_id" validate:"required,gt=0"`                                      // Foreign key ke User (Anak yang ditugaskan)
	TaskID           int            `json:"task_id" validate:"required,gt=0"`                                      // Foreign key ke Task (Definisi tugas)
	AssignedByUserID int            `json:"assigned_by_user_id" validate:"required,gt=0"`                          // Foreign key ke User (Parent yang menugaskan)
	Status           UserTaskStatus `json:"status" validate:"required,oneof=assigned approved submitted rejected"` // Status penugasan saat ini
	AssignedAt       time.Time      `json:"assigned_at,omitzero"`                                                  // Waktu penugasan
	SubmittedAt      *time.Time     `json:"submitted_at,omitzero"`                                                 // Waktu anak submit tugas (nullable)
	VerifiedByUserID int            `json:"verified_by_user_id,omitzero" validate:"omitempty,gt=0"`                // Foreign key ke User (Parent yang verifikasi) (nullable)
	VerifiedAt       *time.Time     `json:"verified_at,omitzero"`                                                  // Waktu verifikasi oleh parent (nullable)
	CompletedAt      *time.Time     `json:"completed_at,omitzero"`                                                 // Waktu tugas dianggap selesai (setelah approved) (nullable)
	Task             *Task          `json:"task,omitempty"`                                                        // Relasi ke Task (bisa di-preload)
	User             *User          `json:"user,omitempty"`                                                        // Relasi ke User (Anak) (bisa di-preload)
	CreatedAt        time.Time      `json:"created_at,omitzero"`                                                   // Waktu pembuatan record
	UpdatedAt        time.Time      `json:"updated_at,omitzero"`                                                   // Waktu terakhir pembaruan record
}

// UserReward merepresentasikan hadiah yang telah diklaim oleh seorang anak.
type UserReward struct {
	ID               int              `json:"id"`                                                         // ID unik klaim hadiah
	UserID           int              `json:"user_id" validate:"required,gt=0"`                           // Foreign key ke User (Anak yang klaim)
	RewardID         int              `json:"reward_id" validate:"required,gt=0"`                         // Foreign key ke Reward (Definisi hadiah)
	PointsDeducted   int              `json:"points_deducted" validate:"required,gte=0"`                  // Jumlah poin yang dikurangi saat klaim
	ClaimedAt        time.Time        `json:"claimed_at,omitzero"`                                        // Waktu klaim dibuat
	Status           UserRewardStatus `json:"status" validate:"required,oneof=pending approved rejected"` // Status klaim saat ini
	ReviewedByUserID int              `json:"reviewed_by_user_id,omitzero" validate:"omitempty,gt=0"`     // Foreign key ke User (Parent yang review) (nullable)
	ReviewedAt       *time.Time       `json:"reviewed_at,omitzero"`                                       // Waktu review oleh parent (nullable)
	Reward           *Reward          `json:"reward,omitempty"`                                           // Relasi ke Reward (bisa di-preload)
	User             *User            `json:"user,omitempty"`                                             // Relasi ke User (Anak) (bisa di-preload)
	CreatedAt        time.Time        `json:"created_at,omitzero"`                                        // Waktu pembuatan record
	UpdatedAt        time.Time        `json:"updated_at,omitzero"`                                        // Waktu terakhir pembaruan record
}

// PointTransaction merepresentasikan catatan perubahan poin seorang anak.
type PointTransaction struct {
	ID                  int             `json:"id"`                                                                                             // ID unik transaksi poin
	UserID              int             `json:"user_id" validate:"required,gt=0"`                                                               // Foreign key ke User (Anak yang poinnya berubah)
	ChangeAmount        int             `json:"change_amount" validate:"required"`                                                              // Jumlah perubahan poin (+/-)
	TransactionType     TransactionType `json:"transaction_type" validate:"required,oneof=task_completion reward_redemption manual_adjustment"` // Jenis transaksi penyebab perubahan poin
	RelatedUserTaskID   int             `json:"related_user_task_id,omitzero" validate:"omitempty,gt=0"`                                        // Foreign key ke UserTask (jika terkait penyelesaian tugas) (nullable)
	RelatedUserRewardID int             `json:"related_user_reward_id,omitzero" validate:"omitempty,gt=0"`                                      // Foreign key ke UserReward (jika terkait klaim hadiah) (nullable)
	CreatedByUserID     int             `json:"created_by_user_id" validate:"required,gt=0"`                                                    // Foreign key ke User (yang menyebabkan transaksi, misal Parent verifikasi, Anak klaim, Admin adjust)
	Notes               string          `json:"notes,omitempty"`                                                                                // Catatan tambahan (misal: alasan manual adjustment)
	User                *User           `json:"user,omitempty"`                                                                                 // Relasi ke User (Anak) (bisa di-preload)
	UserTask            *UserTask       `json:"user_task,omitempty"`                                                                            // Relasi ke UserTask (bisa di-preload)
	UserReward          *UserReward     `json:"user_reward,omitempty"`                                                                          // Relasi ke UserReward (bisa di-preload)
	CreatedAt           time.Time       `json:"created_at,omitzero"`                                                                            // Waktu pembuatan record
	UpdatedAt           time.Time       `json:"updated_at,omitzero"`                                                                            // Waktu terakhir pembaruan record
}

// InvitationCode merepresentasikan data kode undangan di database.
type InvitationCode struct {
	ID                int              `json:"id"`                                                   // ID unik kode undangan
	Code              string           `json:"code" validate:"required,len=10"`                      // Kode undangan unik (panjang sesuai implementasi service)
	ChildID           int              `json:"child_id" validate:"required,gt=0"`                    // Foreign key ke User (Anak yang diundang)
	CreatedByParentID int              `json:"created_by_parent_id" validate:"required,gt=0"`        // Foreign key ke User (Parent yang membuat kode)
	Status            InvitationStatus `json:"status" validate:"required,oneof=active used expired"` // Status kode undangan saat ini
	ExpiresAt         time.Time        `json:"expires_at" validate:"required"`                       // Waktu kedaluwarsa kode
	CreatedAt         time.Time        `json:"created_at,omitzero"`                                  // Waktu pembuatan record
	UpdatedAt         time.Time        `json:"updated_at,omitzero"`                                  // Waktu terakhir pembaruan record

	// Relasi (opsional, bisa di-preload jika perlu ditampilkan bersamaan)
	// Child *User `json:"child,omitempty"`
	// Creator *User `json:"creator,omitempty"`
}

// ====================================================================================
// Enumerations (Tipe Data Konstanta)
// ====================================================================================

// UserTaskStatus mendefinisikan status yang mungkin untuk sebuah UserTask.
type UserTaskStatus string

const (
	UserTaskStatusAssigned  UserTaskStatus = "assigned"  // Tugas baru ditugaskan ke anak
	UserTaskStatusSubmitted UserTaskStatus = "submitted" // Anak telah menandai tugas sebagai selesai
	UserTaskStatusApproved  UserTaskStatus = "approved"  // Parent telah menyetujui tugas yang disubmit
	UserTaskStatusRejected  UserTaskStatus = "rejected"  // Parent telah menolak tugas yang disubmit
)

// UserRewardStatus mendefinisikan status yang mungkin untuk sebuah UserReward (klaim hadiah).
type UserRewardStatus string

const (
	UserRewardStatusPending  UserRewardStatus = "pending"  // Klaim hadiah baru dibuat oleh anak, menunggu review parent
	UserRewardStatusApproved UserRewardStatus = "approved" // Parent telah menyetujui klaim hadiah
	UserRewardStatusRejected UserRewardStatus = "rejected" // Parent telah menolak klaim hadiah
)

// TransactionType mendefinisikan jenis transaksi yang menyebabkan perubahan poin.
type TransactionType string

const (
	TransactionTypeCompletion       TransactionType = "task_completion"   // Poin didapat dari menyelesaikan tugas
	TransactionTypeRedemption       TransactionType = "reward_redemption" // Poin dikurangi karena klaim hadiah
	TransactionTypeManualAdjustment TransactionType = "manual_adjustment" // Poin diubah manual oleh Parent/Admin
)

// InvitationStatus mendefinisikan status yang mungkin untuk kode undangan.
type InvitationStatus string

const (
	InvitationStatusActive  InvitationStatus = "active"  // Kode aktif dan bisa digunakan
	InvitationStatusUsed    InvitationStatus = "used"    // Kode sudah pernah digunakan
	InvitationStatusExpired InvitationStatus = "expired" // Kode sudah melewati batas waktu penggunaan
)

// ====================================================================================
// Input Data Transfer Objects (DTOs) - Digunakan untuk menerima data dari request API
// ====================================================================================

// RegisterUserInput adalah DTO untuk request registrasi pengguna baru.
type RegisterUserInput struct {
	Username  string `json:"username" validate:"required,min=3,max=100"` // Username yang diinginkan
	Password  string `json:"password" validate:"required,min=6"`         // Kata sandi (minimal 6 karakter)
	Email     string `json:"email" validate:"required,email"`            // Alamat email valid
	FirstName string `json:"first_name,omitempty"`                       // Nama depan (opsional)
	LastName  string `json:"last_name,omitempty"`                        // Nama belakang (opsional)
	RoleID    int    `json:"role_id" validate:"required,gt=0"`           // ID peran yang didaftarkan (misal: Parent)
}

// LoginUserInput adalah DTO untuk request login pengguna.
type LoginUserInput struct {
	Username string `json:"username" validate:"required"` // Username yang terdaftar
	Password string `json:"password" validate:"required"` // Kata sandi
}

// AdminUpdateUserInput adalah DTO untuk request pembaruan data pengguna oleh Admin.
// Field bersifat opsional (omitempty).
type AdminUpdateUserInput struct {
	Username  string `json:"username" validate:"omitempty,min=3,max=100"` // Username baru (jika ingin diubah)
	Email     string `json:"email" validate:"omitempty,email"`            // Email baru (jika ingin diubah)
	FirstName string `json:"first_name,omitempty"`                        // Nama depan baru (jika ingin diubah)
	LastName  string `json:"last_name,omitempty"`                         // Nama belakang baru (jika ingin diubah)
	RoleID    int    `json:"role_id" validate:"omitempty,gt=0"`           // ID peran baru (jika ingin diubah)
}

// UpdateProfileInput adalah DTO untuk request pembaruan profil oleh pengguna sendiri.
type UpdateProfileInput struct {
	Username  string `json:"username" validate:"required,min=3,max=100"` // Username (wajib)
	Email     string `json:"email" validate:"required,email"`            // Email (wajib)
	FirstName string `json:"first_name,omitempty"`                       // Nama depan (opsional)
	LastName  string `json:"last_name,omitempty"`                        // Nama belakang (opsional)
}

// UpdatePasswordInput adalah DTO untuk request perubahan kata sandi oleh pengguna sendiri.
type UpdatePasswordInput struct {
	OldPassword string `json:"old_password" validate:"required,min=6"` // Kata sandi lama untuk verifikasi
	NewPassword string `json:"new_password" validate:"required,min=6"` // Kata sandi baru (minimal 6 karakter)
}

// AddChildInput adalah DTO untuk request penambahan relasi anak oleh Parent.
type AddChildInput struct {
	Identifier string `json:"identifier" validate:"required"` // Bisa username atau email anak yang sudah terdaftar
}

// CreateTaskInput adalah DTO untuk request pembuatan definisi Task baru oleh Parent.
type CreateTaskInput struct {
	TaskName        string `json:"task_name" validate:"required,min=3,max=255"` // Nama tugas (maks 255 char)
	TaskPoint       int    `json:"task_point" validate:"required,gt=0"`         // Poin tugas (harus > 0)
	TaskDescription string `json:"task_description,omitempty"`                  // Deskripsi (opsional)
}

// UpdateTaskInput adalah DTO untuk request pembaruan definisi Task oleh Parent.
type UpdateTaskInput struct {
	TaskName        string `json:"task_name" validate:"required,min=3,max=255"` // Nama tugas baru
	TaskPoint       int    `json:"task_point" validate:"required,gt=0"`         // Poin tugas baru
	TaskDescription string `json:"task_description,omitempty"`                  // Deskripsi baru
}

// CreateRewardInput adalah DTO untuk request pembuatan definisi Reward baru oleh Parent.
type CreateRewardInput struct {
	RewardName        string `json:"reward_name" validate:"required,min=3,max=255"` // Nama hadiah
	RewardPoint       int    `json:"reward_point" validate:"required,gt=0"`         // Poin hadiah (harus > 0)
	RewardDescription string `json:"reward_description,omitempty"`                  // Deskripsi (opsional)
}

// UpdateRewardInput adalah DTO untuk request pembaruan definisi Reward oleh Parent.
type UpdateRewardInput struct {
	RewardName        string `json:"reward_name" validate:"required,min=3,max=255"` // Nama hadiah baru
	RewardPoint       int    `json:"reward_point" validate:"required,gt=0"`         // Poin hadiah baru
	RewardDescription string `json:"reward_description,omitempty"`                  // Deskripsi baru
}

// AssignTaskInput adalah DTO untuk request penugasan Task ke Child oleh Parent.
type AssignTaskInput struct {
	TaskID int `json:"task_id" validate:"required,gt=0"` // ID Task yang akan ditugaskan
}

// VerifyTaskInput adalah DTO untuk request verifikasi Task yang disubmit Child oleh Parent.
type VerifyTaskInput struct {
	Status string `json:"status" validate:"required,oneof=approved rejected"` // Status verifikasi ('approved' atau 'rejected')
}

// ReviewClaimInput adalah DTO untuk request review klaim Reward oleh Parent.
type ReviewClaimInput struct {
	Status string `json:"status" validate:"required,oneof=approved rejected"` // Status review ('approved' atau 'rejected')
}

// AdjustPointsInput adalah DTO untuk request penyesuaian poin manual oleh Parent/Admin.
type AdjustPointsInput struct {
	ChangeAmount int    `json:"change_amount" validate:"required,ne=0"`  // Jumlah perubahan poin (tidak boleh 0)
	Notes        string `json:"notes" validate:"required,min=3,max=255"` // Alasan penyesuaian (wajib)
}

// CreateChildInput adalah DTO untuk request pembuatan akun Child baru oleh Parent.
type CreateChildInput struct {
	Username  string `json:"username" validate:"required,min=3,max=100"` // Username anak
	Password  string `json:"password" validate:"required,min=6"`         // Kata sandi anak
	Email     string `json:"email" validate:"omitempty,email"`           // Email anak (opsional?)
	FirstName string `json:"first_name,omitempty"`                       // Nama depan anak (opsional)
	LastName  string `json:"last_name,omitempty"`                        // Nama belakang anak (opsional)
}

// AcceptInvitationInput adalah DTO untuk request menerima undangan menggunakan kode.
type AcceptInvitationInput struct {
	Code string `json:"invitation_code" validate:"required,len=10"` // Kode undangan yang diterima (panjang sesuai implementasi)
}

// ====================================================================================
// Response Data Transfer Objects (DTOs) - Digunakan untuk mengirim data ke client
// ====================================================================================

// Response adalah struktur standar untuk semua response API.
type Response struct {
	Success bool        `json:"success"`        // Menandakan apakah operasi berhasil (true) atau gagal (false)
	Message string      `json:"message"`        // Pesan deskriptif tentang hasil operasi
	Data    interface{} `json:"data,omitempty"` // Data payload response (opsional, hanya ada jika sukses dan ada data)
}
