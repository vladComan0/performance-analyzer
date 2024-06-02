package main

import (
	"crypto/tls"
	"database/sql"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"github.com/vladComan0/performance-analyzer/internal/data"
	"github.com/vladComan0/performance-analyzer/pkg/helpers"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type config struct {
	Addr           string    `mapstructure:"addr"`
	Environment    string    `mapstructure:"environment"`
	DSN            string    `mapstructure:"dsn"`
	DebugEnabled   bool      `mapstructure:"debug_enabled"`
	AllowedOrigins []string  `mapstructure:"allowed_origins"`
	Log            logConfig `mapstructure:"log"`
}

type logConfig struct {
	Level     string `mapstructure:"level"`
	Colorized bool   `mapstructure:"colorized"`
}

type application struct {
	environments data.EnvironmentStorageInterface
	workers      data.WorkerStorageInterface
	config       config
	helper       *helpers.Helper
	log          zerolog.Logger
}

func main() {
	var cfg config

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	getConfig(log, &cfg)
	log = configureLogger(cfg, log)

	db, err := openDB(cfg.DSN)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer func() {
		_ = db.Close()
	}()

	helper := helpers.NewHelper(log, cfg.DebugEnabled)

	environmentModel := &data.EnvironmentStorage{
		DB: db,
	}

	workerModel := &data.WorkerStorage{
		DB: db,
	}

	// dependency injection
	app := &application{
		environments: environmentModel,
		workers:      workerModel,
		config:       cfg,
		helper:       helper,
		log:          log,
	}

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

	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		TLSConfig:    tlsConfig,
	}

	log.Info().Msgf("Starting server on port: %s", strings.Split(server.Addr, ":")[1])
	//err := server.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	err = server.ListenAndServe()
	log.Fatal().Err(err)
}

func getConfig(log zerolog.Logger, config *config) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Msg("Error reading config file")
	}

	if err := viper.Unmarshal(config); err != nil {
		log.Fatal().Err(err).Msg("Unable to decode into struct")
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

func configureLogger(cfg config, log zerolog.Logger) zerolog.Logger {
	var logLevel zerolog.Level

	if cfg.Log.Level == "" {
		log.Info().Msg("Log level is not set, defaulting to info")
		logLevel = zerolog.InfoLevel
	} else {
		var err error
		logLevel, err = zerolog.ParseLevel(cfg.Log.Level)
		if err != nil {
			log.Warn().Msgf("Invalid log level %q, defaulting to info", cfg.Log.Level)
			logLevel = zerolog.InfoLevel
		}
	}
	log = log.Level(logLevel)

	if cfg.Log.Colorized {
		output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		log = log.Output(output)
	}

	return log
}
