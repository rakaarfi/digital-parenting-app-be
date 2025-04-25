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

// ====================================================================================
// Swagger / OpenAPI Documentation Annotations
// ====================================================================================
// Anotasi ini dibaca oleh 'swag init' untuk menghasilkan dokumentasi API.
// @title Digital Parenting App BE API
// @version 1.0
// @description API backend for digital parenting application.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT License
// @license.url https://opensource.org/licenses/MIT

// @host localhost:3001
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description "Type 'Bearer YOUR_JWT_TOKEN' into the value field."
// ====================================================================================

// main adalah fungsi entry point aplikasi Go.
func main() {
	// ====================================================================================
	// Langkah 0: Load Konfigurasi Aplikasi
	// ====================================================================================
	// Membaca variabel lingkungan dari file .env (jika ada) dan memuatnya.
	// Ini harus dilakukan paling awal karena komponen lain (logger, db) mungkin bergantung padanya.
	configs.LoadConfig()
	// Hindari logging sebelum logger diinisialisasi. Gunakan fmt jika perlu untuk debug awal.
	// fmt.Println("[DEBUG] Configuration loaded (pre-logger)")

	// ====================================================================================
	// Langkah 1: Setup Logger Aplikasi (Zerolog)
	// ====================================================================================
	// Menginisialisasi logger global (Zerolog) berdasarkan konfigurasi dari env vars.
	// Mengembalikan io.Closer jika output log diarahkan ke file.
	logCloser := applogger.SetupLogger()
	// Menjadwalkan penutupan file log (jika ada) saat aplikasi berhenti.
	if logCloser != nil {
		defer func() {
			zlog.Info().Msg("Attempting to close log file...")
			if err := logCloser.Close(); err != nil {
				// Jika gagal menutup, log error ke Stderr karena logger file mungkin sudah tidak bisa diakses.
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to close log file: %v\n", err)
			} else {
				fmt.Println("[INFO] Log file closed successfully.") // Konfirmasi penutupan ke Stderr
			}
		}()
	}
	// Log pertama menggunakan logger yang sudah dikonfigurasi.
	zlog.Info().Msg("Application configuration loaded successfully.")

	// ====================================================================================
	// Langkah 2: Koneksi ke Database (PostgreSQL via PgxPool)
	// ====================================================================================
	// Membuat connection pool ke database PostgreSQL menggunakan konfigurasi dari env vars.
	dbPool, err := database.NewPgxPool()
	if err != nil {
		// Jika koneksi gagal, log error fatal dan hentikan aplikasi.
		zlog.Fatal().Err(err).Msg("FATAL: Could not establish database connection pool")
	}
	// Menjadwalkan penutupan connection pool saat aplikasi berhenti.
	// Defer ini ditempatkan setelah defer logger agar pesan penutupan pool bisa di-log.
	defer func() {
		zlog.Info().Msg("Closing database connection pool...")
		dbPool.Close() // pgxpool.Pool.Close() tidak mengembalikan error.
		zlog.Info().Msg("Database connection pool closed.")
	}()
	zlog.Info().Msg("Database connection pool established successfully.")

	// ====================================================================================
	// Langkah 3: Inisialisasi Lapisan Repository (Data Access Layer)
	// ====================================================================================
	// Membuat instance konkret dari setiap interface repository.
	// Setiap repository di-inject dengan dependensi `dbPool` untuk akses database.
	userRepo := repository.NewUserRepository(dbPool)
	roleRepo := repository.NewRoleRepository(dbPool)
	userRelRepo := repository.NewUserRelationshipRepository(dbPool)
	taskRepo := repository.NewTaskRepository(dbPool)
	userTaskRepo := repository.NewUserTaskRepository(dbPool, userRelRepo) // UserTaskRepo butuh UserRelRepo
	rewardRepo := repository.NewRewardRepository(dbPool)
	userRewardRepo := repository.NewUserRewardRepository(dbPool, userRelRepo) // UserRewardRepo butuh UserRelRepo
	pointRepo := repository.NewPointTransactionRepository(dbPool)
	invitationCodeRepo := repository.NewInvitationCodeRepository(dbPool)
	zlog.Info().Msg("Repositories initialized successfully.")

	// ====================================================================================
	// Langkah 4: Inisialisasi Lapisan Service (Business Logic Layer)
	// ====================================================================================
	// Membuat instance konkret dari setiap interface service.
	// Setiap service di-inject dengan dependensi repository yang relevan.
	authService := service.NewAuthService(userRepo, roleRepo)
	taskService := service.NewTaskService(dbPool, userTaskRepo, pointRepo, userRelRepo)
	rewardService := service.NewRewardService(dbPool, rewardRepo, userRewardRepo, pointRepo, userRelRepo)
	userService := service.NewUserService(dbPool, userRepo, roleRepo, userRelRepo)
	invitationService := service.NewInvitationService(dbPool, invitationCodeRepo, userRelRepo, userRepo)
	zlog.Info().Msg("Services initialized successfully.")

	// ====================================================================================
	// Langkah 5: Inisialisasi Lapisan Handler (API Layer)
	// ====================================================================================
	// Membuat instance konkret dari setiap handler.
	// Setiap handler di-inject dengan dependensi service (atau repository jika logikanya sederhana).
	authHandler := handlers.NewAuthHandler(authService)
	adminHandler := handlers.NewAdminHandler(userRepo, roleRepo) // Contoh: Admin handler mungkin langsung pakai repo
	userHandler := handlers.NewUserHandler(userService)
	parentHandler := handlers.NewParentHandler(
		userRelRepo, taskRepo, userTaskRepo, rewardRepo, userRewardRepo,
		pointRepo, userRepo, taskService, rewardService,
		userService, invitationService, // Inject services
	)
	childHandler := handlers.NewChildHandler(
		userTaskRepo, rewardRepo, userRewardRepo, pointRepo, rewardService, // Inject services/repos
	)
	zlog.Info().Msg("Handlers initialized successfully.")

	// ====================================================================================
	// Langkah 6: Setup Aplikasi Web (Fiber)
	// ====================================================================================
	// Membuat instance baru dari aplikasi web Fiber.
	// Mengkonfigurasi ErrorHandler global kustom untuk menangani error secara konsisten.
	app := fiber.New(fiber.Config{
		ErrorHandler: handlers.ErrorHandler, // Menggunakan error handler kustom
	})
	zlog.Info().Msg("Fiber application instance created.")

	// ====================================================================================
	// Langkah 7: Setup Middleware dan Routing
	// ====================================================================================
	// Mendaftarkan middleware global yang akan dijalankan untuk setiap request.
	// Contoh: Logger request, CORS, Recover (panic handling).
	appmiddleware.SetupGlobalMiddleware(app)
	zlog.Info().Msg("Global middleware registered.")

	// Mendaftarkan endpoint untuk menyajikan dokumentasi Swagger UI.
	// URL: http://<host>/swagger/index.html
	app.Get("/swagger/*", fiberSwagger.WrapHandler)
	zlog.Info().Msg("Swagger UI endpoint registered at /swagger/*")

	// Mendaftarkan semua rute API versi 1 (prefix /api/v1).
	// Fungsi SetupRoutes menerima instance app dan semua handler yang dibutuhkan.
	v1.SetupRoutes(
		app,
		authHandler,
		adminHandler,
		userHandler,
		parentHandler,
		childHandler,
	)
	zlog.Info().Msg("API v1 routes registered successfully.")

	// ====================================================================================
	// Langkah 8: Start Server HTTP
	// ====================================================================================
	// Mendapatkan port dari environment variable (APP_PORT) atau menggunakan default "3000".
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "3000" // Default port jika APP_PORT tidak diset
		zlog.Warn().Msgf("APP_PORT environment variable not set, using default port %s", appPort)
	}

	// Mencatat informasi bahwa server akan segera dimulai.
	zlog.Info().Msgf("Starting HTTP server on port %s...", appPort)

	// Mulai mendengarkan koneksi masuk pada alamat dan port yang ditentukan.
	// app.Listen() adalah operasi blocking, eksekusi akan berhenti di sini sampai server dihentikan.
	startErr := app.Listen(fmt.Sprintf(":%s", appPort))
	if startErr != nil {
		// Jika terjadi error saat memulai server (misal: port sudah digunakan),
		// log error fatal dan hentikan aplikasi.
		zlog.Fatal().Err(startErr).Msgf("FATAL: Failed to start server on port %s", appPort)
	}

	// Pesan ini biasanya tidak akan tercapai kecuali server dihentikan secara normal,
	// yang jarang terjadi dalam praktik untuk server yang berjalan terus menerus.
	zlog.Info().Msg("Server stopped gracefully.")
}
