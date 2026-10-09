package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"myS3/internal/auth"
	"myS3/internal/dto"
	"myS3/internal/httpapi/handlers"
	"myS3/internal/httpapi/middleware"
	"myS3/internal/repository/user"
	"myS3/internal/service"
	"myS3/internal/storage"
)

type Deps struct {
	Logger        *zap.SugaredLogger
	Pool          *pgxpool.Pool
	S3            *storage.S3
	Users         *user.UserRepo
	AuthManager   *auth.Manager
	AuthService   *service.AuthService
	ObjectService *service.ObjectService

	CORSAllowedOrigins []string
	MaxUploadSize      int64
	StorageMode        string
}

func NewRouter(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger(deps.Logger))
	r.Use(middleware.CORS(deps.CORSAllowedOrigins))
	r.Use(middleware.MaxBodyBytes(deps.MaxUploadSize))

	r.GET("/healthz", healthz(deps))
	r.GET("/readyz", readyz(deps))

	authHandlers := handlers.NewAuth(deps.AuthService)
	userHandlers := handlers.NewUser(deps.Users)
	objectHandlers := handlers.NewObjects(deps.ObjectService, deps.Logger)
	adminHandlers := handlers.NewAdmin(deps.ObjectService)

	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", authHandlers.Register)
	v1.POST("/auth/login", authHandlers.Login)
	v1.POST("/auth/refresh", authHandlers.Refresh)
	v1.POST("/auth/logout", authHandlers.Logout)
	v1.GET("/public/objects/:id/content", objectHandlers.PublicContent)

	private := v1.Group("", middleware.Auth(deps.AuthManager))
	private.GET("/me", userHandlers.Me)
	private.GET("/config", configHandler(deps))
	private.POST("/objects", objectHandlers.CreateFolder)
	private.GET("/objects", objectHandlers.ListRoot)
	private.GET("/objects/graph", objectHandlers.Graph)
	private.GET("/objects/:id", objectHandlers.Get)
	private.GET("/objects/:id/children", objectHandlers.ListChildren)
	private.POST("/objects/:id/files", objectHandlers.Upload)
	private.GET("/objects/:id/content", objectHandlers.Content)
	private.PATCH("/objects/:id", objectHandlers.Update)
	private.DELETE("/objects/:id", objectHandlers.Delete)
	private.PATCH("/objects/:id/visibility", objectHandlers.SetVisibility)
	private.GET("/objects/:id/grants", objectHandlers.ListGrants)
	private.PUT("/objects/:id/grants", objectHandlers.SetGrant)
	private.DELETE("/objects/:id/grants/:userId", objectHandlers.DeleteGrant)

	admin := private.Group("/admin", middleware.Admin())
	admin.POST("/repair-links", adminHandlers.RepairLinks)
	admin.POST("/cleanup-orphans", adminHandlers.CleanupOrphans)

	return r
}

func healthz(_ Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func configHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, dto.ConfigResponse{StorageMode: deps.StorageMode})
	}
}

func readyz(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if err := deps.Pool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "db": err.Error()})
			return
		}
		if err := deps.S3.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "s3": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
