package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	deliveryHTTP "mns/backend/internal/delivery/http"
	"mns/backend/internal/delivery/http/handler"
	"mns/backend/internal/delivery/http/middleware"
	"mns/backend/internal/repository/postgres"
	"mns/backend/internal/usecase"
	"mns/backend/pkg/access"
	"mns/backend/pkg/config"
	"mns/backend/pkg/database"
	"mns/backend/pkg/email"
	"mns/backend/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.App.Env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	// appCtx drives background goroutines (rate-limiter cleanup) and is
	// cancelled on shutdown.
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	pool, err := database.NewPostgresPool(dbCtx, cfg.Database)
	if err != nil {
		log.Fatal("connect database", zap.Error(err))
	}
	defer pool.Close()

	log.Info("database connected")

	// Repositories
	userRepo := postgres.NewUserRepository(pool)
	accountRepo := postgres.NewAccountRepository(pool)
	sessionRepo := postgres.NewSessionRepository(pool)
	verificationRepo := postgres.NewVerificationRepository(pool)
	twoFactorRepo := postgres.NewTwoFactorRepository(pool)
	authRepo := postgres.NewAuthRepository(pool)

	// Email sender: real SMTP when configured, otherwise log links (dev).
	var emailSender usecase.EmailSender
	if cfg.Email.SMTPHost != "" {
		emailSender = email.NewSMTPSender(
			cfg.Email.SMTPHost, cfg.Email.SMTPPort, cfg.Email.SMTPUser, cfg.Email.SMTPPass,
			cfg.Email.FromAddress, cfg.Email.FromName, cfg.Email.AppBaseURL,
		)
		log.Info("email sender: SMTP", zap.String("host", cfg.Email.SMTPHost))
	} else {
		emailSender = email.NewLogSender(cfg.Email.AppBaseURL, log)
		log.Warn("email sender: log only (no EMAIL_SMTP_HOST set)")
	}

	// Usecases
	userUC := usecase.NewUserUsecase(userRepo)
	twoFactorUC := usecase.NewTwoFactorUsecase(userRepo, accountRepo, twoFactorRepo, cfg.App.Name)
	verificationUC := usecase.NewVerificationUsecase(userRepo, accountRepo, sessionRepo, verificationRepo, emailSender)
	authUC := usecase.NewAuthUsecase(
		userRepo, accountRepo, sessionRepo, authRepo,
		twoFactorUC, verificationUC, cfg.RequireEmailVerification, cfg.Session.Expiry,
	)
	accessControl := access.DefaultController()
	adminUC := usecase.NewAdminUsecase(userRepo, accountRepo, sessionRepo, authRepo, accessControl, cfg.Session.Expiry)

	// Handlers
	userHandler := handler.NewUserHandler(userUC)
	authHandler := handler.NewAuthHandler(authUC, cfg.Session)
	twoFactorHandler := handler.NewTwoFactorHandler(twoFactorUC)
	verificationHandler := handler.NewVerificationHandler(verificationUC)
	adminHandler := handler.NewAdminHandler(adminUC, cfg.Session)

	// Rate limiters (in-memory). Auth endpoints get a stricter limit.
	globalLimiter := middleware.NewRateLimiter(cfg.RateLimit.Max, cfg.RateLimit.Window)
	authLimiter := middleware.NewRateLimiter(10, cfg.RateLimit.Window)
	go globalLimiter.Cleanup(appCtx)
	go authLimiter.Cleanup(appCtx)

	router := deliveryHTTP.NewRouter(deliveryHTTP.Deps{
		Log:                 log,
		GlobalLimiter:       globalLimiter,
		AuthLimiter:         authLimiter,
		Auth:                authUC,
		CookieName:          cfg.Session.CookieName,
		TrustedOrigins:      cfg.TrustedOrigins,
		AccessControl:       accessControl,
		UserHandler:         userHandler,
		AuthHandler:         authHandler,
		TwoFactorHandler:    twoFactorHandler,
		VerificationHandler: verificationHandler,
		AdminHandler:        adminHandler,
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router.Engine(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info("server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")
	appCancel() // stop background goroutines

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}

	log.Info("server stopped")
}
