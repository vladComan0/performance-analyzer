package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"github.com/vladComan0/performance-analyzer/internal/model/repository"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/vladComan0/performance-analyzer/internal/config"
	"github.com/vladComan0/performance-analyzer/internal/service"
	"github.com/vladComan0/performance-analyzer/pkg/helpers"

	_ "github.com/go-sql-driver/mysql"
)

type application struct {
	environmentService service.EnvironmentService
	workerService      service.WorkerService
	config             config.Config
	helper             *helpers.Helper
	log                zerolog.Logger
}

func main() {
	cfg := config.GetConfig()
	logger := configureLogger(cfg)

	db, err := openDB(cfg.DSN)
	if err != nil {
		logger.Fatal().Err(err)
	}
	defer func() {
		if db != nil { // done only to remain consistent, it is taken care of by the cleanup method
			_ = db.Close()
		}
	}()

	helper := helpers.NewHelper(logger, cfg.DebugEnabled)

	environmentRepository := repository.NewEnvironmentRepositoryDB(db)
	environmentService := service.NewEnvironmentService(environmentRepository)
	workerRepository := repository.NewWorkerRepositoryDB(db)
	workerService := service.NewWorkerService(workerRepository, environmentRepository, logger)

	app := newApplication(environmentService, workerService, cfg, helper, logger)
	server := newServer(cfg, app)

	go app.cleanup(db, server)

	logger.Info().Msgf("Starting server on port: %s", strings.Split(server.Addr, ":")[1])
	//err := server.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	err = server.ListenAndServe()
	logger.Fatal().Err(err)
}

func newApplication(environmentService service.EnvironmentService, workerService service.WorkerService, cfg config.Config, helper *helpers.Helper, log zerolog.Logger) *application {
	return &application{
		environmentService: environmentService,
		workerService:      workerService,
		config:             cfg,
		helper:             helper,
		log:                log,
	}
}

func newServer(cfg config.Config, app *application) *http.Server {
	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	return &http.Server{
		Addr:         cfg.Addr,
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		TLSConfig:    tlsConfig,
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func configureLogger(cfg config.Config) zerolog.Logger {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	var logLevel zerolog.Level

	if cfg.Log.Level == "" {
		logger.Info().Msg("Log level is not set, defaulting to info")
		logLevel = zerolog.InfoLevel
	} else {
		var err error
		logLevel, err = zerolog.ParseLevel(cfg.Log.Level)
		if err != nil {
			logger.Warn().Msgf("Invalid log level %q, defaulting to info", cfg.Log.Level)
			logLevel = zerolog.InfoLevel
		}
	}
	logger = logger.Level(logLevel)

	if cfg.Log.HumanReadable {
		output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		logger = logger.Output(output)
	}

	return logger
}

func (app *application) cleanup(db *sql.DB, server *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)

	sig := <-interruptChan

	app.log.Info().Msgf("Received shutdown signal %s, cleaning up...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		app.log.Error().Err(err).Msg("Error shutting server down")
	}

	if err := db.Close(); err != nil {
		app.log.Error().Msgf("Error closing the database: %s", err)
	}

	os.Exit(0)
}
