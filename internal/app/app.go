package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	pgmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"myS3/internal/auth"
	"myS3/internal/config"
	"myS3/internal/httpapi"
	"myS3/internal/repository/grant"
	"myS3/internal/repository/link"
	"myS3/internal/repository/object"
	repopg "myS3/internal/repository/postgres"
	"myS3/internal/repository/refresh"
	"myS3/internal/repository/user"
	"myS3/internal/service"
	"myS3/internal/storage"
	myMigrations "myS3/migrations"
)

type Application struct {
	cfg    *config.Config
	logger *zap.SugaredLogger
	server *http.Server
	pool   *pgxpool.Pool
	s3     *storage.S3

	userRepo    *user.UserRepo
	refreshRepo *refresh.RefreshRepo
	objectRepo  *object.ObjectRepo
	grantRepo   *grant.GrantRepo
	linkRepo    *link.LinkRepo
}

func NewApplication(cfg *config.Config, logger *zap.SugaredLogger) *Application {
	return &Application{cfg: cfg, logger: logger}
}

func (a *Application) Init(ctx context.Context) error {
	if err := a.initBackend(ctx); err != nil {
		return err
	}

	if a.cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	a.userRepo = user.NewUserRepo(a.pool)
	a.refreshRepo = refresh.NewRefreshRepo(a.pool)
	a.objectRepo = object.NewObjectRepo(a.pool)
	a.grantRepo = grant.NewGrantRepo(a.pool)
	a.linkRepo = link.NewLinkRepo(a.pool)

	manager := auth.NewManager(a.cfg.JWT.Secret, a.cfg.JWT.AccessTTL)
	authSvc := service.NewAuthService(a.userRepo, a.refreshRepo, manager, a.cfg.JWT.RefreshTTL)
	perms := service.NewPermissionService(a.objectRepo, a.grantRepo)
	objectSvc := service.NewObjectService(a.objectRepo, a.grantRepo, a.linkRepo, a.s3, perms, a.cfg.S3.Mode)

	router := httpapi.NewRouter(httpapi.Deps{
		Logger:        a.logger,
		Pool:          a.pool,
		S3:            a.s3,
		Users:         a.userRepo,
		AuthManager:   manager,
		AuthService:   authSvc,
		ObjectService: objectSvc,

		CORSAllowedOrigins: a.cfg.CORSAllowedOrigins,
		MaxUploadSize:      a.cfg.MaxUploadSize,
		StorageMode:        a.cfg.S3.Mode,
	})
	a.server = &http.Server{
		Addr:         ":" + a.cfg.Port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return nil
}

func (a *Application) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.logger.Infow("starting server", "addr", a.server.Addr)
		errCh <- a.server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		a.logger.Infow("received shutdown signal, starting graceful shutdown")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	a.pool.Close()
	a.logger.Infow("server stopped")
	return nil
}

func (a *Application) runMigrations() error {
	sqlDB, err := sql.Open("pgx", a.cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("open sql for migrations: %w", err)
	}
	defer sqlDB.Close()

	driver, err := pgmigrate.WithInstance(sqlDB, &pgmigrate.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}

	src, err := iofs.New(myMigrations.FS, ".")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func (a *Application) initBackend(ctx context.Context) error {
	pool, err := repopg.New(ctx, a.cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}
	a.pool = pool

	s, err := storage.New(a.cfg.S3)
	if err != nil {
		return fmt.Errorf("init s3: %w", err)
	}
	if err := s.EnsureBucket(ctx); err != nil {
		return fmt.Errorf("ensure bucket: %w", err)
	}
	a.s3 = s
	return nil
}

func (a *Application) Migrate(ctx context.Context) error {
	pool, err := repopg.New(ctx, a.cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}
	defer pool.Close()
	return a.runMigrations()
}
