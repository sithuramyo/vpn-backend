package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"vpn-backend/internal/auth"
	"vpn-backend/internal/config"
	"vpn-backend/internal/metrics"
	"vpn-backend/internal/middleware"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/services"
)

type Services struct {
	Auth       *services.AuthService
	Admins     *services.AdminService
	Users      *services.UserService
	Devices    *services.DeviceService
	AccessKeys *services.AccessKeyService
	Servers    *services.ServerService
	Usage      *services.UsageService
	Audit      *services.AuditService
}

var (
	allRoles   = []models.AdminRole{models.AdminRoleAdmin, models.AdminRoleOperator, models.AdminRoleViewer}
	writeRoles = []models.AdminRole{models.AdminRoleAdmin, models.AdminRoleOperator}
	adminOnly  = []models.AdminRole{models.AdminRoleAdmin}
)

func NewRouter(cfg *config.Config, svc *Services, repos *repositories.Repositories, sessions *auth.SessionManager, rateLimiter *middleware.RateLimiter) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Recovery(), middleware.RequestLogger(), metrics.Middleware(), middleware.SecureHeaders())
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(rateLimiter.Middleware())

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	authHandler := NewAuthHandler(svc.Auth)
	adminHandler := NewAdminHandler(svc.Admins, svc.Audit)
	userHandler := NewUserHandler(svc.Users, svc.Audit)
	deviceHandler := NewDeviceHandler(svc.Devices, svc.Audit)
	keyHandler := NewAccessKeyHandler(svc.AccessKeys, svc.Audit)
	serverHandler := NewServerHandler(svc.Servers, svc.Audit)
	usageHandler := NewUsageHandler(svc.Usage)
	auditHandler := NewAuditLogHandler(svc.Audit)
	summaryHandler := NewMetricsSummaryHandler(svc.Users, svc.Devices, svc.AccessKeys, svc.Servers)

	v1 := r.Group("/api/v1")

	v1.POST("/auth/google", authHandler.Login)

	authed := v1.Group("")
	authed.Use(middleware.RequireAuth(sessions, repos.Admins))
	{
		authed.GET("/auth/me", authHandler.Me)
		authed.POST("/auth/logout", authHandler.Logout)

		admins := authed.Group("/admins")
		admins.Use(middleware.RequireRole(adminOnly...))
		{
			admins.GET("", adminHandler.List)
			admins.GET("/:id", adminHandler.Get)
			admins.PATCH("/:id", adminHandler.Update)
		}

		users := authed.Group("/users")
		{
			users.GET("", middleware.RequireRole(allRoles...), userHandler.List)
			users.GET("/:id", middleware.RequireRole(allRoles...), userHandler.Get)
			users.POST("", middleware.RequireRole(writeRoles...), userHandler.Create)
			users.PATCH("/:id", middleware.RequireRole(writeRoles...), userHandler.Update)
			users.POST("/:id/disable", middleware.RequireRole(writeRoles...), userHandler.Disable)
			users.DELETE("/:id", middleware.RequireRole(writeRoles...), userHandler.Delete)
		}

		devices := authed.Group("/devices")
		{
			devices.GET("", middleware.RequireRole(allRoles...), deviceHandler.List)
			devices.GET("/:id", middleware.RequireRole(allRoles...), deviceHandler.Get)
			devices.PATCH("/:id", middleware.RequireRole(writeRoles...), deviceHandler.Update)
			devices.DELETE("/:id", middleware.RequireRole(writeRoles...), deviceHandler.Delete)
		}

		accessKeys := authed.Group("/access-keys")
		{
			accessKeys.GET("", middleware.RequireRole(allRoles...), keyHandler.List)
			accessKeys.GET("/:id", middleware.RequireRole(allRoles...), keyHandler.Get)
			accessKeys.POST("", middleware.RequireRole(writeRoles...), keyHandler.Create)
			accessKeys.DELETE("/:id", middleware.RequireRole(writeRoles...), keyHandler.Delete)
			accessKeys.POST("/:id/revoke", middleware.RequireRole(writeRoles...), keyHandler.Revoke)
			accessKeys.POST("/:id/rotate", middleware.RequireRole(writeRoles...), keyHandler.Rotate)
			accessKeys.GET("/:id/config", middleware.RequireRole(writeRoles...), keyHandler.GetConfig)
			accessKeys.GET("/:id/qr", middleware.RequireRole(writeRoles...), keyHandler.GetQR)
		}

		serversGroup := authed.Group("/servers")
		{
			serversGroup.GET("", middleware.RequireRole(allRoles...), serverHandler.List)
			serversGroup.GET("/:id", middleware.RequireRole(allRoles...), serverHandler.Get)
			serversGroup.GET("/:id/health", middleware.RequireRole(allRoles...), serverHandler.Health)
			serversGroup.GET("/:id/metrics", middleware.RequireRole(allRoles...), serverHandler.MetricsHistory)
			serversGroup.POST("", middleware.RequireRole(adminOnly...), serverHandler.Create)
			serversGroup.PATCH("/:id", middleware.RequireRole(adminOnly...), serverHandler.Update)
			serversGroup.DELETE("/:id", middleware.RequireRole(adminOnly...), serverHandler.Delete)
		}

		usage := authed.Group("/usage")
		usage.Use(middleware.RequireRole(allRoles...))
		{
			usage.GET("/summary", usageHandler.Summary)
			usage.GET("/users", usageHandler.ByUser)
			usage.GET("/servers", usageHandler.ByServer)
		}

		authed.GET("/audit-logs", middleware.RequireRole(allRoles...), auditHandler.List)
		authed.GET("/metrics/summary", middleware.RequireRole(allRoles...), summaryHandler.Summary)
	}

	return r
}
