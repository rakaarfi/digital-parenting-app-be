package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rakaarfi/digital-parenting-app-be/internal/api/v1/handlers" // Handler spesifik v1
	"github.com/rakaarfi/digital-parenting-app-be/internal/middleware"      // Middleware aplikasi (Auth, dll)
)

func SetupRoutes(
	app *fiber.App,
	authHandler *handlers.AuthHandler,
	adminHandler *handlers.AdminHandler,
	userHandler *handlers.UserHandler,
	parentHandler *handlers.ParentHandler,
	childHandler *handlers.ChildHandler,
) {
	// -------------------------------------------------------------------------
	// Grouping Rute API v1
	// -------------------------------------------------------------------------
	api := app.Group("/api/v1")

	// =========================================================================
	// Rute Autentikasi (Publik)
	// =========================================================================
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// =========================================================================
	// Rute Admin
	// =========================================================================
	admin := api.Group("/admin", middleware.Protected(), middleware.Authorize("Admin"))
	// User Management by Admin
	admin.Get("/users", adminHandler.GetAllUsers)
	admin.Get("/users/:userId", adminHandler.GetUserByID)
	admin.Patch("/users/:userId", adminHandler.UpdateUser) // Gunakan PATCH untuk update parsial? Atau PUT
	admin.Delete("/users/:userId", adminHandler.DeleteUser)
	// Role Management by Admin
	admin.Post("/roles", adminHandler.CreateRole)
	admin.Get("/roles", adminHandler.GetAllRoles)
	admin.Get("/roles/:roleId", adminHandler.GetRoleByID)
	admin.Patch("/roles/:roleId", adminHandler.UpdateRole) // Atau PUT
	admin.Delete("/roles/:roleId", adminHandler.DeleteRole)

	// =========================================================================
	// Rute Pengguna (Profil Pribadi)
	// =========================================================================
	user := api.Group("/user", middleware.Protected()) // Semua role yang login bisa akses profil
	user.Get("/profile", userHandler.GetMyProfile)
	user.Patch("/profile", userHandler.UpdateMyProfile) // Atau PUT
	user.Patch("/password", userHandler.UpdateMyPassword)

	// =========================================================================
	// Rute Parent
	// =========================================================================
	parent := api.Group("/parent", middleware.Protected(), middleware.Authorize("Parent"))
	// --- Child Management & Invitations ---
	parent.Get("/children", parentHandler.GetMyChildren)
	parent.Post("/children", parentHandler.AddChild)
	parent.Post("/children/create", parentHandler.CreateChildAccount)                   // Membuat akun anak baru
	parent.Delete("/children/:childId", parentHandler.RemoveChild)                      // Hapus relasi dengan anak
	parent.Post("/children/:childId/invitations", parentHandler.GenerateInvitationCode) // Generate kode untuk anak spesifik
	parent.Post("/join-child", parentHandler.JoinWithInvitationCode)                    // Join menggunakan kode undangan

	// --- Task Definition Management ---
	parent.Post("/tasks", parentHandler.CreateTaskDefinition)
	parent.Get("/tasks", parentHandler.GetMyTaskDefinitions) // List task buatan parent
	parent.Patch("/tasks/:taskId", parentHandler.UpdateMyTaskDefinition)
	parent.Delete("/tasks/:taskId", parentHandler.DeleteMyTaskDefinition)

	// --- Task Assignment & Verification ---
	parent.Post("/children/:childId/tasks", parentHandler.AssignTaskToChild)     // Assign task ke anak spesifik
	parent.Get("/children/:childId/tasks", parentHandler.GetTasksForChild)       // Lihat tugas anak spesifik (filter by status?)
	parent.Patch("/tasks/:userTaskId/verify", parentHandler.VerifySubmittedTask) // Verify tugas spesifik (berdasarkan ID UserTask)

	// --- Reward Definition Management ---
	parent.Post("/rewards", parentHandler.CreateRewardDefinition)
	parent.Get("/rewards", parentHandler.GetMyRewardDefinitions) // List reward buatan parent
	parent.Patch("/rewards/:rewardId", parentHandler.UpdateMyRewardDefinition)
	parent.Delete("/rewards/:rewardId", parentHandler.DeleteMyRewardDefinition)

	// --- Reward Claim Review ---
	parent.Get("/claims/pending", parentHandler.GetPendingClaims)            // Lihat semua klaim pending dari anak-anaknya
	parent.Patch("/claims/:claimId/review", parentHandler.ReviewRewardClaim) // Approve/reject klaim spesifik

	// --- Point Adjustment ---
	parent.Post("/children/:childId/points", parentHandler.AdjustChildPoints) // Adjust poin anak

	// =========================================================================
	// Rute Child
	// =========================================================================
	child := api.Group("/child", middleware.Protected(), middleware.Authorize("Child"))
	// Task Viewing & Submission
	child.Get("/tasks", childHandler.GetMyTasks)                        // Lihat tugas diri sendiri (filter by status?)
	child.Patch("/tasks/:userTaskId/submit", childHandler.SubmitMyTask) // Submit tugas spesifik
	// Points & Rewards
	child.Get("/points", childHandler.GetMyPoints)                   // Lihat total poin
	child.Get("/points/history", childHandler.GetMyPointHistory)     // Lihat riwayat transaksi poin
	child.Get("/rewards", childHandler.GetAvailableRewards)          // Lihat reward yang bisa diklaim
	child.Post("/rewards/:rewardId/claim", childHandler.ClaimReward) // Klaim reward spesifik
	child.Get("/claims", childHandler.GetMyClaims)                   // Lihat riwayat klaim reward diri sendiri

	// =========================================================================
	// Rute Lain-lain (Publik)
	// =========================================================================
	api.Get("/health", HealthCheck) // Pindahkan ke /api/v1/health?

}

// HealthCheck godoc
// @Summary Check Health
// @Description Public endpoint to verify that the API is running and responsive.
// @Tags Public
// @ID health-check
// @Produce json
// @Success 200 {object} map[string]string `json:"status"`
// @Router /health [get]
func HealthCheck(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "UP"})
}
