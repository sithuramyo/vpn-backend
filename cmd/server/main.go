package main

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"vpn-backend/internal/auth"
	"vpn-backend/internal/config"
	"vpn-backend/internal/crypto"
	"vpn-backend/internal/database"
	"vpn-backend/internal/handlers"
	"vpn-backend/internal/middleware"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/services"
	"vpn-backend/internal/vpn"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}
	if cfg.SecretEncryptionKey == "" {
		log.Fatal("SECRET_ENCRYPTION_KEY must be set")
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}
	database.SetMigrationsFS(os.DirFS(migrationsPath))
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	repos := repositories.New(db)

	keyBytes, err := decodeSecretKey(cfg.SecretEncryptionKey)
	if err != nil {
		log.Fatalf("invalid SECRET_ENCRYPTION_KEY: %v", err)
	}
	secretBox, err := crypto.NewSecretBox(keyBytes)
	if err != nil {
		log.Fatalf("invalid SECRET_ENCRYPTION_KEY: %v", err)
	}

	googleVerifier := auth.NewGoogleVerifier(cfg.GoogleClientID)
	sessions := auth.NewSessionManager(cfg.JWTSecret, cfg.JWTExpiry)

	var provider vpn.VPNProvider = vpn.NewOutlineShadowsocksProvider(cfg.OutlineAPIURL, cfg.OutlineAPICertSHA256)

	auditService := services.NewAuditService(repos.AuditLogs)
	svc := &handlers.Services{
		Auth:       services.NewAuthService(googleVerifier, sessions, repos.Admins, auditService),
		Admins:     services.NewAdminService(repos.Admins),
		Users:      services.NewUserService(repos.Users, repos.Devices, repos.AccessKeys, provider),
		Devices:    services.NewDeviceService(repos.Devices),
		AccessKeys: services.NewAccessKeyService(repos.AccessKeys, repos.Servers, provider, secretBox, cfg.VPNDomain),
		Servers:    services.NewServerService(repos.Servers, repos.Metrics, repos.AccessKeys, provider),
		Usage:      services.NewUsageService(repos.Users, repos.AccessKeys, repos.Servers, repos.Metrics),
		Audit:      auditService,
	}

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	rateLimiter.StartCleanup(10 * time.Minute)

	router := handlers.NewRouter(cfg, svc, repos, sessions, rateLimiter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startExpirationSweep(ctx, svc.Users, svc.AccessKeys)

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s (env=%s)", cfg.AppPort, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func decodeSecretKey(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// startExpirationSweep periodically disables/revokes users and access keys
// whose expiry has passed, so expiration is enforced even without an
// admin-triggered request.
func startExpirationSweep(ctx context.Context, users *services.UserService, keys *services.AccessKeyService) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := users.SweepExpired(ctx); err != nil {
					log.Printf("user expiration sweep failed: %v", err)
				} else if n > 0 {
					log.Printf("expired %d user(s)", n)
				}
				if n, err := keys.SweepExpired(ctx); err != nil {
					log.Printf("access key expiration sweep failed: %v", err)
				} else if n > 0 {
					log.Printf("expired %d access key(s)", n)
				}
			}
		}
	}()
}
