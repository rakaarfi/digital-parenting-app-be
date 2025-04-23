package models

import (
	"time"
)

type Role struct {
	ID        int       `json:"id"`
	Name      string    `json:"name" validate:"required,min=3,max=50"`
	CreatedAt time.Time `json:"created_at,omitzero"`
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username" validate:"required,min=3,max=100"`
	Password  string    `json:"-"`
	Email     string    `json:"email" validate:"required,email"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	RoleID    int       `json:"role_id" validate:"required,gt=0"`
	Role      *Role     `json:"role,omitempty"`
	CreatedAt time.Time `json:"created_at,omitzero"`
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

type UserRelationship struct {
	ID        int       `json:"id"`
	ParentID  int       `json:"parent_id" validate:"required,gt=0"`
	ChildID   int       `json:"child_id" validate:"required,gt=0"`
	Child     *User     `json:"child,omitempty"`
	CreatedAt time.Time `json:"created_at,omitzero"`
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

type Task struct {
	ID              int       `json:"id"`
	TaskName        string    `json:"task_name" validate:"required,min=3,max=100"`
	TaskPoint       int       `json:"task_point" validate:"required,gt=0"`
	TaskDescription string    `json:"task_description,omitempty"`
	CreatedByUserID int       `json:"created_by_user_id" validate:"required,gt=0"`
	User            *User     `json:"user,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitzero"`
	UpdatedAt       time.Time `json:"updated_at,omitzero"`
}

type UserTaskStatus string

const (
	UserTaskStatusAssigned  UserTaskStatus = "assigned"
	UserTaskStatusApproved  UserTaskStatus = "approved"
	UserTaskStatusSubmitted UserTaskStatus = "submitted"
	UserTaskStatusRejected  UserTaskStatus = "rejected"
)

type UserTask struct {
	ID               int            `json:"id"`
	UserID           int            `json:"user_id" validate:"required,gt=0"` // Child
	TaskID           int            `json:"task_id" validate:"required,gt=0"`
	AssignedByUserID int            `json:"assigned_by_user_id" validate:"required,gt=0"` // Parent
	Status           UserTaskStatus `json:"status" validate:"required,oneof=assigned approved submitted rejected"`
	AssignedAt       time.Time      `json:"assigned_at,omitzero"`
	SubmittedAt      *time.Time     `json:"submitted_at,omitzero"`
	VerifiedByUserID int            `json:"verified_by_user_id,omitzero" validate:"omitempty,gt=0"` // Parent verification after submitted
	VerifiedAt       *time.Time     `json:"verified_at,omitzero"`
	CompletedAt      *time.Time     `json:"completed_at,omitzero"`
	Task             *Task          `json:"task,omitempty"`
	User             *User          `json:"user,omitempty"`
	CreatedAt        time.Time      `json:"created_at,omitzero"`
	UpdatedAt        time.Time      `json:"updated_at,omitzero"`
}

type Reward struct {
	ID                int       `json:"id"`
	RewardName        string    `json:"reward_name" validate:"required,min=3,max=100"`
	RewardPoint       int       `json:"reward_point" validate:"required,gt=0"`
	RewardDescription string    `json:"reward_description,omitempty"`
	CreatedByUserID   int       `json:"created_by_user_id" validate:"required,gt=0"`
	User              *User     `json:"user,omitempty"`
	CreatedAt         time.Time `json:"created_at,omitzero"`
	UpdatedAt         time.Time `json:"updated_at,omitzero"`
}

type UserRewardStatus string

const (
	UserRewardStatusPending  UserRewardStatus = "pending"
	UserRewardStatusApproved UserRewardStatus = "approved"
	UserRewardStatusRejected UserRewardStatus = "rejected"
)

type UserReward struct {
	ID               int              `json:"id"`
	UserID           int              `json:"user_id" validate:"required,gt=0"` // Child
	RewardID         int              `json:"reward_id" validate:"required,gt=0"`
	PointsDeducted   int              `json:"points_deducted" validate:"required,gte=0"`
	ClaimedAt        time.Time        `json:"claimed_at,omitzero"`
	Status           UserRewardStatus `json:"status" validate:"required,oneof=pending approved rejected"`
	ReviewedByUserID int              `json:"reviewed_by_user_id,omitzero" validate:"omitempty,gt=0"` // Parent
	ReviewedAt       *time.Time       `json:"reviewed_at,omitzero"`
	Reward           *Reward          `json:"reward,omitempty"`
	User             *User            `json:"user,omitempty"`
	CreatedAt        time.Time        `json:"created_at,omitzero"`
	UpdatedAt        time.Time        `json:"updated_at,omitzero"`
}

type TransactionType string

const (
	TransactionTypeCompletion       TransactionType = "task_completion"
	TransactionTypeRedemption       TransactionType = "reward_redemption"
	TransactionTypeManualAdjustment TransactionType = "manual_adjustment"
)

type PointTransaction struct {
	ID                  int             `json:"id"`
	UserID              int             `json:"user_id" validate:"required,gt=0"`
	ChangeAmount        int             `json:"change_amount" validate:"required"`
	TransactionType     TransactionType `json:"transaction_type" validate:"required,oneof=task_completion reward_redemption manual_adjustment"`
	RelatedUserTaskID   int             `json:"related_user_task_id,omitzero" validate:"omitempty,gt=0"`
	RelatedUserRewardID int             `json:"related_user_reward_id,omitzero" validate:"omitempty,gt=0"`
	CreatedByUserID     int             `json:"created_by_user_id" validate:"required,gt=0"`
	Notes               string          `json:"notes,omitempty"`
	User                *User           `json:"user,omitempty"`
	UserTask            *UserTask       `json:"user_task,omitempty"`
	UserReward          *UserReward     `json:"user_reward,omitempty"`
	CreatedAt           time.Time       `json:"created_at,omitzero"`
	UpdatedAt           time.Time       `json:"updated_at,omitzero"`
}

// Input struct terpisah untuk registrasi dan login
type RegisterUserInput struct {
	Username  string `json:"username" validate:"required,min=3,max=100"`
	Password  string `json:"password" validate:"required,min=6"`
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	RoleID    int    `json:"role_id" validate:"required,gt=0"`
}

type LoginUserInput struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type AdminUpdateUserInput struct {
	Username  string `json:"username" validate:"required,min=3,max=100"`
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	RoleID    int    `json:"role_id" validate:"required,gt=0"` // Pastikan role ID > 0
}

type UpdateProfileInput struct {
	Username  string `json:"username" validate:"required,min=3,max=100"`
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type UpdatePasswordInput struct {
	OldPassword string `json:"old_password" validate:"required,min=6"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// Input untuk menambahkan relasi anak
type AddChildInput struct {
	Identifier string `json:"identifier" validate:"required"` // Bisa username atau email anak
}

// --- Input Structs untuk Task Definition ---

// CreateTaskInput berisi data yang dibutuhkan untuk membuat Task baru oleh Parent.
type CreateTaskInput struct {
	TaskName        string `json:"task_name" validate:"required,min=3,max=255"` // Sesuaikan max length jika perlu
	TaskPoint       int    `json:"task_point" validate:"required,gt=0"`
	TaskDescription string `json:"task_description,omitempty"`
}

// UpdateTaskInput berisi data yang bisa diubah untuk Task yang sudah ada oleh Parent.
type UpdateTaskInput struct {
	TaskName        string `json:"task_name" validate:"required,min=3,max=255"`
	TaskPoint       int    `json:"task_point" validate:"required,gt=0"`
	TaskDescription string `json:"task_description,omitempty"`
}

// --- Input Structs untuk Reward Definition ---

// CreateRewardInput berisi data yang dibutuhkan untuk membuat Reward baru oleh Parent.
type CreateRewardInput struct {
	RewardName        string `json:"reward_name" validate:"required,min=3,max=255"`
	RewardPoint       int    `json:"reward_point" validate:"required,gt=0"`
	RewardDescription string `json:"reward_description,omitempty"`
}

// UpdateRewardInput berisi data yang bisa diubah untuk Reward yang sudah ada oleh Parent.
type UpdateRewardInput struct {
	RewardName        string `json:"reward_name" validate:"required,min=3,max=255"`
	RewardPoint       int    `json:"reward_point" validate:"required,gt=0"`
	RewardDescription string `json:"reward_description,omitempty"`
}

// Input untuk penyesuaian poin manual oleh Parent/Admin
type AdjustPointsInput struct {
	ChangeAmount int    `json:"change_amount" validate:"required,ne=0"`  // Harus ada, tidak boleh 0
	Notes        string `json:"notes" validate:"required,min=3,max=255"` // Wajib beri alasan
}

type CreateChildInput struct {
	Username  string `json:"username" validate:"required,min=3,max=100"`
	Password  string `json:"password" validate:"required,min=6"`
	Email     string `json:"email" validate:"omitempty,email"` // Email bisa opsional untuk anak?
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// Response standar untuk API
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
