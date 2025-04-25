package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rakaarfi/digital-parenting-app-be/internal/api/v1/handlers"
	"github.com/rakaarfi/digital-parenting-app-be/internal/middleware"
)

// File ini bertanggung jawab untuk mendefinisikan dan mendaftarkan semua rute (endpoints)
// untuk API versi 1 (/api/v1).

// SetupRoutes mengkonfigurasi dan mendaftarkan semua rute API v1 ke instance aplikasi Fiber.
// Fungsi ini menerima instance aplikasi Fiber dan semua handler yang dibutuhkan sebagai dependensi.
func SetupRoutes(
	app *fiber.App, // Instance aplikasi Fiber utama
	authHandler *handlers.AuthHandler, // Handler untuk endpoint autentikasi
	adminHandler *handlers.AdminHandler, // Handler untuk endpoint administrasi
	userHandler *handlers.UserHandler, // Handler untuk endpoint pengguna umum
	parentHandler *handlers.ParentHandler, // Handler untuk endpoint khusus Parent
	childHandler *handlers.ChildHandler, // Handler untuk endpoint khusus Child
) {
	// Membuat grup rute utama dengan prefix /api/v1
	// Semua rute yang didefinisikan di bawah ini akan memiliki prefix ini.
	api := app.Group("/api/v1")

	// =========================================================================
	// Rute Autentikasi (Publik)
	// =========================================================================
	// Rute-rute ini tidak memerlukan autentikasi (tidak ada middleware Protected).
	auth := api.Group("/auth")
	{
		// POST /api/v1/auth/register - Mendaftarkan pengguna baru (Parent/Child/Admin)
		auth.Post("/register", authHandler.Register)
		// POST /api/v1/auth/login - Login pengguna dan mendapatkan token JWT
		auth.Post("/login", authHandler.Login)
	}

	// =========================================================================
	// Rute Admin (Memerlukan Autentikasi & Otorisasi Admin)
	// =========================================================================
	// Rute-rute ini memerlukan token JWT yang valid (Protected) dan pengguna harus memiliki peran "Admin" (Authorize).
	admin := api.Group("/admin", middleware.Protected(), middleware.Authorize("Admin"))
	{
		// --- Manajemen Pengguna oleh Admin ---
		// GET    /api/v1/admin/users - Mendapatkan daftar semua pengguna (dengan paginasi)
		admin.Get("/users", adminHandler.GetAllUsers)
		// GET    /api/v1/admin/users/:userId - Mendapatkan detail pengguna berdasarkan ID
		admin.Get("/users/:userId", adminHandler.GetUserByID)
		// PATCH  /api/v1/admin/users/:userId - Memperbarui data pengguna berdasarkan ID
		admin.Patch("/users/:userId", adminHandler.UpdateUser)
		// DELETE /api/v1/admin/users/:userId - Menghapus pengguna berdasarkan ID
		admin.Delete("/users/:userId", adminHandler.DeleteUser)

		// --- Manajemen Peran (Role) oleh Admin ---
		// POST   /api/v1/admin/roles - Membuat peran baru
		admin.Post("/roles", adminHandler.CreateRole)
		// GET    /api/v1/admin/roles - Mendapatkan daftar semua peran
		admin.Get("/roles", adminHandler.GetAllRoles)
		// GET    /api/v1/admin/roles/:roleId - Mendapatkan detail peran berdasarkan ID
		admin.Get("/roles/:roleId", adminHandler.GetRoleByID)
		// PATCH  /api/v1/admin/roles/:roleId - Memperbarui data peran berdasarkan ID
		admin.Patch("/roles/:roleId", adminHandler.UpdateRole)
		// DELETE /api/v1/admin/roles/:roleId - Menghapus peran berdasarkan ID
		admin.Delete("/roles/:roleId", adminHandler.DeleteRole)
	}

	// =========================================================================
	// Rute Pengguna Umum (Memerlukan Autentikasi)
	// =========================================================================
	// Rute-rute ini memerlukan token JWT yang valid (Protected).
	// Semua peran yang terautentikasi (Parent, Child, Admin) dapat mengakses endpoint ini.
	user := api.Group("/user", middleware.Protected())
	{
		// GET   /api/v1/user/profile - Mendapatkan profil pengguna yang sedang login
		user.Get("/profile", userHandler.GetMyProfile)
		// PATCH /api/v1/user/profile - Memperbarui profil pengguna yang sedang login
		user.Patch("/profile", userHandler.UpdateMyProfile)
		// PATCH /api/v1/user/password - Mengubah kata sandi pengguna yang sedang login
		user.Patch("/password", userHandler.UpdateMyPassword)
	}

	// =========================================================================
	// Rute Parent (Memerlukan Autentikasi & Otorisasi Parent)
	// =========================================================================
	// Rute-rute ini memerlukan token JWT yang valid (Protected) dan pengguna harus memiliki peran "Parent" (Authorize).
	parent := api.Group("/parent", middleware.Protected(), middleware.Authorize("Parent"))
	{
		// --- Manajemen Anak & Undangan ---
		// GET    /api/v1/parent/children - Mendapatkan daftar anak yang terhubung dengan Parent
		parent.Get("/children", parentHandler.GetMyChildren)
		// POST   /api/v1/parent/children - Menambahkan relasi dengan anak yang sudah ada (menggunakan username/email)
		parent.Post("/children", parentHandler.AddChild) // Handler ini mungkin perlu diganti/dihapus jika pakai invitation code
		// POST   /api/v1/parent/children/create - Membuat akun anak baru dan langsung menautkannya
		parent.Post("/children/create", parentHandler.CreateChildAccount)
		// DELETE /api/v1/parent/children/:childId - Menghapus relasi dengan anak tertentu
		parent.Delete("/children/:childId", parentHandler.RemoveChild)
		// POST   /api/v1/parent/children/:childId/invitations - Membuat kode undangan untuk anak tertentu
		parent.Post("/children/:childId/invitations", parentHandler.GenerateInvitationCode)
		// POST   /api/v1/parent/join-child - Bergabung/menambahkan relasi dengan anak menggunakan kode undangan
		parent.Post("/join-child", parentHandler.JoinWithInvitationCode)

		// --- Manajemen Definisi Tugas (Task Definition) ---
		// POST   /api/v1/parent/tasks - Membuat definisi tugas baru
		parent.Post("/tasks", parentHandler.CreateTaskDefinition)
		// GET    /api/v1/parent/tasks - Mendapatkan daftar definisi tugas yang dibuat oleh Parent ini
		parent.Get("/tasks", parentHandler.GetMyTaskDefinitions)
		// PATCH  /api/v1/parent/tasks/:taskId - Memperbarui definisi tugas tertentu
		parent.Patch("/tasks/:taskId", parentHandler.UpdateMyTaskDefinition)
		// DELETE /api/v1/parent/tasks/:taskId - Menghapus definisi tugas tertentu
		parent.Delete("/tasks/:taskId", parentHandler.DeleteMyTaskDefinition)

		// --- Penugasan & Verifikasi Tugas (Task Assignment & Verification) ---
		// POST   /api/v1/parent/children/:childId/tasks - Menugaskan task ke anak tertentu
		parent.Post("/children/:childId/tasks", parentHandler.AssignTaskToChild)
		// GET    /api/v1/parent/children/:childId/tasks - Melihat daftar tugas yang ditugaskan ke anak tertentu (bisa filter status)
		parent.Get("/children/:childId/tasks", parentHandler.GetTasksForChild)
		// PATCH  /api/v1/parent/tasks/:userTaskId/verify - Memverifikasi (approve/reject) tugas yang sudah disubmit anak (berdasarkan ID UserTask)
		parent.Patch("/tasks/:userTaskId/verify", parentHandler.VerifySubmittedTask)

		// --- Manajemen Definisi Hadiah (Reward Definition) ---
		// POST   /api/v1/parent/rewards - Membuat definisi hadiah baru
		parent.Post("/rewards", parentHandler.CreateRewardDefinition)
		// GET    /api/v1/parent/rewards - Mendapatkan daftar definisi hadiah yang dibuat oleh Parent ini
		parent.Get("/rewards", parentHandler.GetMyRewardDefinitions)
		// PATCH  /api/v1/parent/rewards/:rewardId - Memperbarui definisi hadiah tertentu
		parent.Patch("/rewards/:rewardId", parentHandler.UpdateMyRewardDefinition)
		// DELETE /api/v1/parent/rewards/:rewardId - Menghapus definisi hadiah tertentu
		parent.Delete("/rewards/:rewardId", parentHandler.DeleteMyRewardDefinition)

		// --- Peninjauan Klaim Hadiah (Reward Claim Review) ---
		// GET    /api/v1/parent/claims/pending - Melihat daftar klaim hadiah dari anak-anaknya yang statusnya 'pending'
		parent.Get("/claims/pending", parentHandler.GetPendingClaims)
		// PATCH  /api/v1/parent/claims/:claimId/review - Meninjau (approve/reject) klaim hadiah tertentu (berdasarkan ID UserReward)
		parent.Patch("/claims/:claimId/review", parentHandler.ReviewRewardClaim)

		// --- Penyesuaian Poin Anak (Point Adjustment) ---
		// POST   /api/v1/parent/children/:childId/points - Menyesuaikan poin anak tertentu secara manual (tambah/kurang)
		parent.Post("/children/:childId/points", parentHandler.AdjustChildPoints)
	}

	// =========================================================================
	// Rute Child (Memerlukan Autentikasi & Otorisasi Child)
	// =========================================================================
	// Rute-rute ini memerlukan token JWT yang valid (Protected) dan pengguna harus memiliki peran "Child" (Authorize).
	child := api.Group("/child", middleware.Protected(), middleware.Authorize("Child"))
	{
		// --- Melihat & Menyelesaikan Tugas ---
		// GET   /api/v1/child/tasks - Melihat daftar tugas yang ditugaskan untuk dirinya (bisa filter status)
		child.Get("/tasks", childHandler.GetMyTasks)
		// PATCH /api/v1/child/tasks/:userTaskId/submit - Menandai tugas tertentu sebagai selesai (submit)
		child.Patch("/tasks/:userTaskId/submit", childHandler.SubmitMyTask)

		// --- Poin & Hadiah ---
		// GET  /api/v1/child/points - Melihat total poin saat ini
		child.Get("/points", childHandler.GetMyPoints)
		// GET  /api/v1/child/points/history - Melihat riwayat transaksi poin
		child.Get("/points/history", childHandler.GetMyPointHistory)
		// GET  /api/v1/child/rewards - Melihat daftar hadiah yang tersedia (dari semua parent yang terhubung)
		child.Get("/rewards", childHandler.GetAvailableRewards)
		// POST /api/v1/child/rewards/:rewardId/claim - Mengklaim hadiah tertentu
		child.Post("/rewards/:rewardId/claim", childHandler.ClaimReward)
		// GET  /api/v1/child/claims - Melihat riwayat klaim hadiah yang pernah dilakukan
		child.Get("/claims", childHandler.GetMyClaims)
	}

	// =========================================================================
	// Rute Lain-lain / Utilitas (Publik)
	// =========================================================================
	// GET /api/v1/health - Endpoint publik untuk memeriksa status kesehatan API
	api.Get("/health", HealthCheck)

}

// HealthCheck godoc
// @Summary Check API Health Status
// @Description Public endpoint to verify that the API is running and responsive.
// @Tags Public, Health
// @ID health-check-v1
// @Produce json
// @Success 200 {object} map[string]string "{"status":"UP"}"
// @Router /health [get]
func HealthCheck(c *fiber.Ctx) error {
	// Mengembalikan response JSON sederhana yang menandakan API aktif.
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "UP"})
}
