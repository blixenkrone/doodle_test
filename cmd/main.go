package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blixenkrone/sdk/logger"
	"github.com/blixenkrone/sdk/storage/postgres"

	sdkhttp "github.com/blixenkrone/sdk/http"

	_ "github.com/blixenkrone/doodle/docs" // Required for swaggo to find the generated docs.
	"github.com/blixenkrone/doodle/internal/onboarding"
	"github.com/blixenkrone/doodle/internal/storage"

	"github.com/caarlos0/env/v11"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// @title API
// @version 1.0
// @description Doodle API
// @contact.name test
// @BasePath /
// @schemes http
func main() {
	log := logger.New()
	if os.Getenv("DEBUG") != "" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	envConfig, err := env.ParseAs[envConfig]()
	if err != nil {
		log.Fatalf("error loading env config: %v", err)
		return
	}

	shutdownGracePeriod := 30 * time.Second
	if os.Getenv("SHUTDOWN_GRACE_PERIOD") != "" {
		if period, perr := time.ParseDuration(os.Getenv("SHUTDOWN_GRACE_PERIOD")); perr != nil {
			log.Error("Failed to parse SHUTDOWN_GRACE_PERIOD, using default (30s)")
		} else {
			shutdownGracePeriod = period
		}
	}

	serviceName := envConfig.SERVICE_NAME

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	pgConnConfig := postgres.ConnConfig{
		User:         envConfig.DATABASE_USERNAME,
		Password:     envConfig.DATABASE_PASSWORD,
		Host:         envConfig.DATABASE_HOST,
		Port:         "5432",
		DBName:       envConfig.DATABASE_NAME,
		SSLMode:      "disable",
		PoolMaxConns: 10,
	}
	postgresDB, err := postgres.NewFromConnectionString(appCtx, pgConnConfig.BuildDSNConnStr(), serviceName)
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer postgresDB.Close(appCtx)
	if err := postgresDB.Ping(appCtx); err != nil {
		log.Fatalf("failed to ping DB: %v", err)
		return
	}

	store := storage.NewStore(postgresDB.Pool())

	addr := ":8080"
	router := mux.NewRouter()
	srv := sdkhttp.NewServer(log, router, addr, serviceName)
	mw := sdkhttp.NewHTTPMiddleware(serviceName, log)
	srv.Use(mw.LogRoutes)

	// Handlers
	onboardingH := onboarding.NewHTTPHandler(log, store)

	// Onboarding
	srv.AddRoute("/users", onboardingH.CreateUser(), http.MethodPost)

	// Timeslot management
	srv.AddRoute("/timeslots", meetings.RegisterTimeslot(), http.MethodPost)
	// srv.AddRoute("/meetings", meetings.meetings(), http.MethodPost)
	// srv.AddRoute("/meetings/{meetings}", meetings.Overview(), http.MethodGet)

	if err := srv.RegisterRoutes(); err != nil {
		log.Fatalln(err)
		return
	}

	go func() {
		log.Infof("starting HTTP server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server listen error: %s", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the main goroutine
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	s := <-quit
	log.Infof("shutting down app from signal: %s", s)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("server forced to shutdown: ", err)
	}

	// Stop background workers and let in-flight work drain if any
	appCancel()
	log.Info("App exiting")
}

// envConfig holds the runtime configuration, populated from environment variables.
type envConfig struct {
	SERVICE_NAME string `env:"SERVICE_NAME,required"`
	ENVIRONMENT  string `env:"ENVIRONMENT,required"`

	DATABASE_NAME     string `env:"DATABASE_NAME,required"`
	DATABASE_HOST     string `env:"DATABASE_HOST,required"`
	DATABASE_USERNAME string `env:"DATABASE_USERNAME,required"`
	DATABASE_PASSWORD string `env:"DATABASE_PASSWORD,required"`
}
