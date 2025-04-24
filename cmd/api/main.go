package main

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/rakaarfi/digital-parenting-app-be/configs"
	v1 "github.com/rakaarfi/digital-parenting-app-be/internal/api/v1"
	"github.com/rakaarfi/digital-parenting-app-be/internal/api/v1/handlers"
	"github.com/rakaarfi/digital-parenting-app-be/internal/database"
	applogger "github.com/rakaarfi/digital-parenting-app-be/internal/logger"
	appmiddleware "github.com/rakaarfi/digital-parenting-app-be/internal/middleware"
	"github.com/rakaarfi/digital-parenting-app-be/internal/repository"
	"github.com/rakaarfi/digital-parenting-app-be/internal/service"
	zlog "github.com/rs/zerolog/log"

	// Import untuk Swagger/OpenAPI documentation
	_ "github.com/rakaarfi/digital-parenting-app-be/docs" // Import side effect untuk registrasi docs Swagger yang digenerate
	fiberSwagger "github.com/swaggo/fiber-swagger"        // Middleware Fiber untuk menyajikan Swagger UI
)

// --- Anotasi Global Swagger/OpenAPI ---
// Anotasi ini dibaca oleh 'swag init' untuk menghasilkan dokumentasi API.
// @title Digital Parenting App BE API
// @version 1.0
// @description API backend for digital parenting application.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3001
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description "Type 'Bearer YOUR_JWT_TOKEN' into the value field."
// --- Akhir Anotasi Swagger ---

// main adalah fungsi entry point aplikasi Go.
func main() {
	// --- Langkah 0: Load Konfigurasi dari .env ---
	// Membaca file .env dan memuat variabelnya ke environment process.
	// Harus dijalankan *sebelum* komponen lain yang bergantung pada env vars (seperti logger, db).
	configs.LoadConfig()
	// Hindari logging sebelum logger siap. fmt.Println bisa digunakan jika benar-benar perlu.
	// fmt.Println("Configuration loaded (pre-logger)")

	// --- Langkah 1: Setup Logger (Zerolog) ---
	// Menginisialisasi logger global (Zerolog) berdasarkan konfigurasi env vars (LOG_LEVEL, dll.).
	// Mengembalikan io.Closer jika file logging diaktifkan.
	logCloser := applogger.SetupLogger()
	// Menjadwalkan penutupan file log (jika ada) saat fungsi main selesai.
	if logCloser != nil {
		defer func() {
			zlog.Info().Msg("Closing log file...") // Log bahwa kita mencoba menutup file
			if err := logCloser.Close(); err != nil {
				// Jika gagal menutup, log error ke Stderr karena logger file mungkin sudah tidak bisa diakses.
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to close log file: %v\n", err)
			}
		}()
	}
	// Log pertama menggunakan Zerolog setelah setup selesai.
	zlog.Info().Msg("Configuration loaded")

	// --- Langkah 2: Koneksi ke Database (PostgreSQL) ---
	dbPool, err := database.NewPgxPool()
	if err != nil {
		zlog.Fatal().Err(err).Msg("Could not connect to the database")
	}
	// Menjadwalkan penutupan connection pool saat fungsi main selesai.
	// Defer ini diletakkan setelah defer logger agar error penutupan DB masih bisa di-log.
	defer dbPool.Close()
	zlog.Info().Msg("Database connection pool established")

	// --- Langkah 3: Inisialisasi Lapisan Repository ---
	// Membuat instance konkret dari setiap repository, menyuntikkan (injecting)
	// connection pool (dbPool) sebagai dependensi.
	userRepo := repository.NewUserRepository(dbPool)
	roleRepo := repository.NewRoleRepository(dbPool)
	userRelRepo := repository.NewUserRelationshipRepository(dbPool)
	taskRepo := repository.NewTaskRepository(dbPool)
	userTaskRepo := repository.NewUserTaskRepository(dbPool, userRelRepo)
	rewardRepo := repository.NewRewardRepository(dbPool)
	userRewardRepo := repository.NewUserRewardRepository(dbPool, userRelRepo)
	pointRepo := repository.NewPointTransactionRepository(dbPool)
	invitationCodeRepo := repository.NewInvitationCodeRepository(dbPool)
	zlog.Info().Msg("Repositories initialized")

	// --- Langkah 4: Inisialisasi Lapisan Service ---
	// Membuat instance konkret dari setiap service, menyuntikkan repository
	// yang relevan sebagai dependensi.
	authService := service.NewAuthService(userRepo, roleRepo) // <-- Buat instance AuthService
	taskService := service.NewTaskService(dbPool, userTaskRepo, pointRepo, userRelRepo)
	rewardService := service.NewRewardService(dbPool, rewardRepo, userRewardRepo, pointRepo, userRelRepo)
	userService := service.NewUserService(dbPool, userRepo, roleRepo, userRelRepo)
	invitationService := service.NewInvitationService(dbPool, invitationCodeRepo, userRelRepo, userRepo)
	zlog.Info().Msg("Services initialized")

	// --- Langkah 4: Inisialisasi Lapisan Handler ---
	// Membuat instance konkret dari setiap handler, menyuntikkan repository
	// yang relevan sebagai dependensi.
	authHandler := handlers.NewAuthHandler(authService)
	adminHandler := handlers.NewAdminHandler(userRepo, roleRepo)
	userHandler := handlers.NewUserHandler(userService)
	parentHandler := handlers.NewParentHandler(
		userRelRepo, taskRepo, userTaskRepo, rewardRepo, userRewardRepo,
		pointRepo, userRepo, taskService, rewardService,
		userService,
		invitationService,
	)
	childHandler := handlers.NewChildHandler(
		userTaskRepo, rewardRepo, userRewardRepo, pointRepo, rewardService,
	)
	zlog.Info().Msg("Handlers initialized")

	// --- Langkah 5: Setup Aplikasi Fiber ---
	// Membuat instance baru dari aplikasi web Fiber.
	// Mengkonfigurasi ErrorHandler global kustom dari paket handlers.
	app := fiber.New(fiber.Config{
		ErrorHandler: handlers.ErrorHandler,
	})
	zlog.Info().Msg("Fiber app initialized")

	// --- Langkah 6: Setup Middleware Global dan Rute ---
	// Mendaftarkan middleware global (seperti logger request, CORS, recover) ke aplikasi Fiber.
	appmiddleware.SetupGlobalMiddleware(app)

	// Mendaftarkan endpoint untuk Swagger UI.
	// Harus didaftarkan *sebelum* rute API utama jika prefix-nya sama atau tumpang tindih.
	// URL: http://<host>/swagger/index.html
	app.Get("/swagger/*", fiberSwagger.WrapHandler)
	zlog.Info().Msg("Swagger UI endpoint registered at /swagger/*")

	// Mendaftarkan semua rute API versi 1 (/api/v1/...) dengan menyuntikkan handler yang sesuai.
	v1.SetupRoutes(
		app,
		authHandler,
		adminHandler,
		userHandler,
		parentHandler,
		childHandler,
	)
	zlog.Info().Msg("API v1 routes registered")

	// --- Langkah 7: Start Server HTTP ---
	// Mendapatkan port dari environment variable atau menggunakan default "3000".
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "3000"
	}

	// Mencatat bahwa server akan dimulai pada port yang ditentukan.
	zlog.Info().Msgf("Server is starting on port %s...", appPort)
	// Mulai mendengarkan request HTTP pada port yang ditentukan.
	// app.Listen bersifat blocking, akan berjalan terus sampai dihentikan atau error.
	startErr := app.Listen(fmt.Sprintf(":%s", appPort))
	if startErr != nil {
		// Jika terjadi error saat memulai server (misal: port sudah digunakan),
		// log error fatal dan hentikan aplikasi.
		zlog.Fatal().Err(startErr).Msg("Failed to start server")
	}
}
